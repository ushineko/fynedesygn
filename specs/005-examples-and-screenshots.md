# Spec 005: Examples and screenshots

> **Note**: This work has no associated issue tracker ticket. The repository
> is a personal public project without an issue tracker.

## Status: COMPLETE

## Executive Summary

Adds four example programs (master-detail, settings, job-runner,
document-viewer), each a complete program on `shell.Run` with a headless
test; the two small pieces they needed, `forms.Saver` (coalesced save) and
the `steps` package (a job's step list); the generic screenshot harness
ported from angou; and the first README screenshots, captured on this
machine. Reviewers should look first at `examples/job-runner/main.go`, the
pattern that exercises the most of the library at once.

## Context

The user asked for reference UIs as examples of the patterns the library
supports. The gallery shows components one at a time; a program built on
several of them shows how they fit. Four patterns cover what the three
consuming programs do: a listing with a detail view and a reload, a settings
page that saves itself, a job with steps and a log that can be cancelled, and
a document viewer. Two small pieces those patterns need are not yet in the
library: the coalesced save (clockwork-orange's `scheduleSave`) and the step
list (nmsbonker's `buildRun` steps).

angou's `tools/screenshot.sh` is the capture harness the apps share; it
drives a window through `--section` and `--scheme`, redirects `HOME` and the
XDG directories so a capture cannot photograph the developer's own data, and
checks focus and geometry before trusting a shot. It ports as a generic tool
parameterised by window class and binary. This machine has its dependencies
(kdotool, spectacle, Pillow).

## Requirements

### R1. `forms.Saver`

- R1.1 `NewSaver(delay time.Duration, save func()) *Saver` with
  `Schedule()` (coalescing: one save `delay` after the last call, sequence
  guarded), `Flush()` (save now if one is pending, for quit), `Pending()
  bool`. The save runs on the UI thread through `fyne.Do`. Default delay
  when zero: `DefaultSaveDelay` = 1 s.

### R2. `steps` package

- R2.1 `State` (`Pending`, `Running`, `Done`, `Failed`, `Cancelled`) with
  `Status()` and `Icon()`; `Step{Name, State, Note}`.
- R2.2 `List`: `New(names ...string)`, `Set(i int, st State, note string)`,
  `Advance(i, note)` (i running, earlier ones done), `Finish(i, note)`,
  `Stop(st, note)` (running steps become st), `Reset()`, `Steps() []Step`,
  `Widget() fyne.CanvasObject` updated in place (rows are marker, name and
  dim note), `Detach()`. Safe to call `Set` from any goroutine through
  `fyne.Do` only; documented.

### R3. Examples

Each is a `main` package under `examples/`, builds with `go build ./...`,
runs on `shell.Run`, and has a headless test building every section.

- R3.1 `examples/master-detail`: a table of items with one status per row,
  a detail card for the selected row with actions gated while working, a
  Refresh that reloads through `Perform` with a simulated delay, a status-bar
  segment with the count, and the loaded-flag pattern in the program's state.
- R3.2 `examples/settings`: a `forms.Form` over a JSON file in the user's
  config directory (sandboxed in tests), auto-saved through `forms.Saver`
  with the "Saved" banner, Revert, the standard Appearance section, and
  `OnStop` flushing a pending save.
- R3.3 `examples/job-runner`: a `steps.List` beside a `logpane.Pane`, a
  fake job run through `PerformCancellable` that advances steps and logs,
  a Cancel path that marks the running step cancelled, and a result banner.
- R3.4 `examples/document-viewer`: `markdown.Section` over an embedded
  guide with a mermaid diagram rendered through the pipeline (its own
  `diagrams/` and `go:generate` directive), plus an About page.

### R4. Screenshot harness

- R4.1 `tools/screenshot.sh`: generic port of angou's harness. Flags
  `--class`, `--bin`, `--section`, `--scheme`, `--with-dialog`, `--home DIR`
  (a throwaway HOME the script creates when absent), positional output.
  `--all` captures every gallery section in Breeze Dark into
  `docs/img/gallery-<section>.png`. Polls for the window, activates and
  confirms focus, grabs with spectacle, checks the aspect ratio.
- R4.2 `tools/crop.py` as in angou (desktop capture cropped to the window
  with the output's scale from kscreen-doctor).
- R4.3 Makefile `screenshots` target; README "Screenshots" section with alt
  text describing each image; `docs/design-system.md` "Testing" notes the
  harness.

### R5. Build and CI

- R5.1 `make build-examples` builds every example; CI's Ubuntu test job runs
  it; the diagram check also covers `examples/document-viewer`.
- R5.2 README gains an "Examples" section.

## Acceptance Criteria

- [x] AC1 `Saver`: three `Schedule` calls within the delay produce one save;
  `Flush` saves a pending one immediately and nothing when none is pending;
  `Pending` reflects it (R1.1).
- [x] AC2 `steps.List`: `Advance` marks earlier steps done and the target
  running; `Stop(Cancelled)` marks only running steps; the widget's text
  shows names and notes; `Detach` then `Set` does not panic (R2).
- [x] AC3 Each example's sections render headlessly in the default scheme;
  `go build ./...` builds all four (R3).
- [x] AC4 master-detail: selecting a row shows its detail; Refresh reloads
  and the count segment updates (R3.1).
- [x] AC5 settings: a change in a field schedules a save that writes the
  JSON file under the sandboxed config dir; Revert restores the file's
  values into the fields (R3.2).
- [x] AC6 job-runner: running the job headlessly completes every step and
  logs lines; a cancelled context marks the running step cancelled (R3.3).
- [x] AC7 document-viewer: the Guide section shows the guide text and its
  diagram image, nothing scrollable inside (R3.4).
- [x] AC8 `tools/screenshot.sh --help` prints usage and exits 0;
  `shellcheck` passes on it when installed; `make screenshots` on this
  machine produces the gallery set and the README references each image
  with alt text (R4). _Ten captures at 1782 x 1194 in Breeze Dark; the
  README shows six. macOS Dark and Windows Light were captured as well and
  checked by eye: rounder corners and system blue for one, flat 4 px
  controls and the Windows accent for the other._
- [x] AC9 `make test`, `make lint`, `go vet`, `make build-examples` clean;
  CI green (R5). _Commit 57d2106 failed on Windows (two test assumptions:
  forward-slash URI paths, CRLF checkout) and its fix e637f8f failed on
  Ubuntu (a flaky select in the job-runner test). Commit a7673fa: all four
  jobs green._

## Risks & Assumptions

- **Assumption**: screenshots are taken on this KDE Plasma 6 Wayland machine
  with the harness; they are committed PNGs and refreshed by hand.
- **Risk**: the harness raises windows on the developer's desktop for a
  second each; it must never be run unattended on a machine in use for
  something else.
- **Risk**: the settings example writes a real file under the user's config
  directory when run as a program; tests sandbox it.
- **Rollback**: `git revert`.

## Alternatives Considered

- Considered putting the examples in the gallery as sections; rejected, a
  reader wants a whole small program to copy, and the gallery's purpose is
  one component at a time.
- Considered a Go screenshot harness using the Fyne software painter;
  rejected because it would not show the real compositor, cursor theme or
  fonts, which are what the design system is about.
