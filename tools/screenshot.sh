#!/usr/bin/env bash
# Capture a fynedesygn window for a README, on KDE/Wayland.
#
# Ported from angou's tools/screenshot.sh (same author, MIT) and made generic:
# any program built on the shell takes --section and --scheme, so the script
# starts a fresh window on the section it wants, grabs it, and kills it.
# Refreshing a whole set is one command with nothing to click, which is the
# difference between screenshots that track the interface and screenshots that
# quietly go stale.
#
# Two things make this less trivial than "take a screenshot":
#
#   1. The active window is almost never the one we want. Refreshing these
#      usually means a terminal driving the capture, so whatever has focus is
#      the terminal. The window is raised first, and found by window *class*
#      (the app ID): searching by name also matches a browser sitting on the
#      project's GitHub page.
#   2. When a dialog is open the dialog *is* the active window, so an
#      active-window grab returns the dialog alone on a transparent
#      background. For those shots pass --with-dialog: it captures the whole
#      desktop and crops to the window's geometry.
#
# HOME and the XDG config/data directories are redirected to a throwaway
# directory so the window cannot reach the developer's preferences or data:
# these images go into a public README. XDG_RUNTIME_DIR is deliberately NOT
# redirected; that is where the Wayland socket lives.
#
# Requires kdotool (Wayland's xdotool), spectacle, and python3 with Pillow.
set -euo pipefail

REPO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CLASS="${FYNEDESYGN_CLASS:-io.ushineko.fynedesygn.gallery}"
BIN="${FYNEDESYGN_BIN:-$REPO_DIR/fynedesygn-gallery}"
HOME_DIR=""

usage() {
    cat <<'USAGE'
usage: tools/screenshot.sh [--class APPID] [--bin PATH] [--scheme NAME] [--with-dialog]
                           [--home DIR] --section NAME <output.png>
       tools/screenshot.sh --all

  --class APPID   the window class to find (the program's app ID);
                  default io.ushineko.fynedesygn.gallery ($FYNEDESYGN_CLASS)
  --bin PATH      the program to start; default ./fynedesygn-gallery ($FYNEDESYGN_BIN)
  --section NAME  which section to open on
  --scheme NAME   colour scheme for this run; not saved over the user's choice
  --with-dialog   a dialog is open: capture the desktop and crop, rather than
                  grabbing the active window (which would be the dialog alone)
  --home DIR      the throwaway HOME to run under; created when absent
  --all           refresh the gallery set into docs/img/gallery-<section>.png,
                  in Breeze Dark unless --scheme says otherwise

The alt text in the README describes what is in each image. It is the only
description a screen-reader user gets, and a stale one is worse than none:
check it still matches before committing a new capture.
USAGE
}

with_dialog=0
section=""
scheme=""
out=""
all=0
while [ $# -gt 0 ]; do
    case "$1" in
        --class) shift; CLASS="${1:-}" ;;
        --bin) shift; BIN="${1:-}" ;;
        --with-dialog) with_dialog=1 ;;
        --section) shift; section="${1:-}" ;;
        --scheme) shift; scheme="${1:-}" ;;
        --home) shift; HOME_DIR="${1:-}" ;;
        --all) all=1 ;;
        -h|--help) usage; exit 0 ;;
        -*) echo "unknown option: $1" >&2; usage >&2; exit 2 ;;
        *) out="$1" ;;
    esac
    shift
done

for tool in kdotool spectacle python3; do
    command -v "$tool" >/dev/null || { echo "$tool is not installed" >&2; exit 1; }
done
python3 -c "import PIL" 2>/dev/null || { echo "python3 Pillow is not installed" >&2; exit 1; }
[ -x "$BIN" ] || { echo "$BIN not found or not executable (set --bin, or run make gallery)" >&2; exit 1; }

made_home=0
if [ -z "$HOME_DIR" ]; then
    HOME_DIR=$(mktemp -d)
    made_home=1
fi
mkdir -p "$HOME_DIR/.config" "$HOME_DIR/.local/share"
cleanup() { [ "$made_home" -eq 1 ] && rm -rf "$HOME_DIR"; return 0; }
trap cleanup EXIT

# capture starts a window on the requested section, grabs it, and stops it
# again. Starting fresh per shot rather than reusing one window keeps each
# image independent of whatever the previous one left selected.
capture() {
    local sect="$1" dest="$2"

    HOME="$HOME_DIR" XDG_CONFIG_HOME="$HOME_DIR/.config" XDG_DATA_HOME="$HOME_DIR/.local/share" \
        "$BIN" --section "$sect" ${scheme:+--scheme "$scheme"} >/dev/null 2>&1 &
    local pid=$!
    # shellcheck disable=SC2064  # pid is captured deliberately, at trap-set time
    trap "kill $pid 2>/dev/null || true; wait $pid 2>/dev/null || true" RETURN

    # Find the window by the process we started, not by class alone: the
    # developer's own copy of the program may be running (hidden in a tray),
    # and a class search returns it first, so every capture would be of that
    # window. Poll rather than sleeping a fixed time: a cold start after a
    # rebuild is much slower than a warm one. Some compositors cannot report
    # a window's pid; after the pid search has had its chance, fall back to
    # the class and say so.
    local wid="" waited=0 w
    while [ "$waited" -lt 40 ]; do
        for w in $(timeout 10 kdotool search --class "$CLASS" 2>/dev/null || true); do
            if [ "$(timeout 10 kdotool getwindowpid "$w" 2>/dev/null || true)" = "$pid" ]; then
                wid=$w
                break
            fi
        done
        [ -n "$wid" ] && break
        if [ "$waited" -ge 24 ]; then
            wid=$(timeout 10 kdotool search --class "$CLASS" 2>/dev/null | head -1 || true)
            if [ -n "$wid" ]; then
                echo "  note: no window of class $CLASS reports pid $pid; using the first one" >&2
                break
            fi
        fi
        sleep 0.25
        waited=$((waited + 1))
    done
    [ -n "$wid" ] || { echo "the window never appeared (no window of class $CLASS)" >&2; return 1; }

    # Activating is asynchronous, and `spectacle -a` grabs whatever is active
    # at the moment it fires. Activate, confirm focus, and only then grab.
    local active="" tries=0
    while [ "$tries" -lt 12 ]; do
        timeout 10 kdotool windowactivate "$wid" >/dev/null 2>&1 || true
        sleep 0.5
        active=$(timeout 10 kdotool getactivewindow 2>/dev/null || true)
        [ "$active" = "$wid" ] && break
        tries=$((tries + 1))
    done
    [ "$active" = "$wid" ] || { echo "could not focus the window (active=$active want=$wid)" >&2; return 1; }
    sleep 1.0                   # let it repaint after the raise

    rm -f "$dest"
    if [ "$with_dialog" -eq 0 ]; then
        # -S drops the compositor's drop shadow, which otherwise pads the image unevenly
        timeout 30 spectacle -a -b -n -S -o "$dest" >/dev/null 2>&1 || true
        sleep 1.5
    else
        local tmp; tmp=$(mktemp --suffix=.png)
        timeout 30 spectacle -f -b -n -o "$tmp" >/dev/null 2>&1 || true
        sleep 1.5
        local geo; geo=$(timeout 10 kdotool getwindowgeometry "$wid")
        python3 "${REPO_DIR}/tools/crop.py" "$tmp" "$dest" \
            "$(printf '%s' "$geo" | awk '/Position/{print $2}')" \
            "$(printf '%s' "$geo" | awk '/Geometry/{print $2}')"
        rm -f "$tmp"
    fi

    [ -s "$dest" ] || { echo "capture produced nothing" >&2; return 1; }

    # A last sanity check on the geometry: an image wildly wider or taller
    # than the window we asked for is not a screenshot of it.
    local geo_check; geo_check=$(timeout 10 kdotool getwindowgeometry "$wid" 2>/dev/null || true)
    python3 - "$dest" "$(printf '%s' "$geo_check" | awk '/Geometry/{print $2}')" <<'PY'
import os, sys
from PIL import Image

path, dim = sys.argv[1], sys.argv[2] if len(sys.argv) > 2 else ""
im = Image.open(path)
if dim and "x" in dim:
    w, h = (float(v) for v in dim.split("x"))
    want, got = w / h, im.width / im.height
    if abs(want - got) / want > 0.05:
        sys.exit("captured %dx%d, but the window is %gx%g -- wrong window grabbed"
                 % (im.width, im.height, w, h))
print("  %s  %dx%d  %.0fK" % (os.path.basename(path), im.width, im.height,
                              os.path.getsize(path) / 1024))
PY
}

if [ "$all" -eq 1 ]; then
    # Force a scheme unless one was asked for, so two refreshes on two
    # machines produce the same colours.
    : "${scheme:=Breeze Dark}"
    mkdir -p "${REPO_DIR}/docs/img"
    for s in $("$BIN" --help 2>&1 | sed -n 's/.*open on this section: //p' | tr -d ',' ); do
        low=$(printf '%s' "$s" | tr '[:upper:]' '[:lower:]')
        capture "$s" "${REPO_DIR}/docs/img/gallery-${low}.png"
    done
    echo
    echo "Now check the alt text in README.md still describes what is in each image."
    exit 0
fi

[ -n "$out" ] || { usage >&2; exit 2; }
[ -n "$section" ] || { echo "--section is required (or use --all)" >&2; exit 2; }
capture "$section" "$out"
