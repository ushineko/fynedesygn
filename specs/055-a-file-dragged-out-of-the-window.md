# Spec 055: a file dragged out of the window

**Issue**: [#166](https://github.com/ushineko/fynedesygn/issues/166)

## Status: INCOMPLETE

Phase 1 (Linux) complete; phase 2 (Windows, macOS) open.

## Executive Summary

Phase 1 adds `dragout`, which starts the platform's own drag of files out of
a Fyne window: the core Wayland protocol on GLFW's connection, and XDND on a
second X11 connection. After the drag, Fyne is sent the button release it
never saw. Windows and macOS report it unsupported until phase 2. Reviewers
should look first at `dragout/wayland_linux.c` (`dragout_wl_prepare` and the
queue handling) and at `glfwBackend.release` in `dragout/backend_linux.go`.

## Context

Fyne 2.8.1 receives drops (`Window.SetOnDropped`) but cannot start one.
`fyne.Draggable` moves things inside the canvas. GLFW has no drag-source API
on any platform. An app that shows files cannot let the user drag one into a
file manager, a browser upload field or a chat window. clockwork-orange's
image review is the first consumer (clockwork-orange #40).

External helpers (`ripdrag`, `dragon`) can do this on Linux, but they add a
runtime dependency and do nothing on Windows or macOS. So each platform's
native drag source is called directly, behind build tags, the way
`glance/opacity_x11.go` sets window opacity.

Per platform:

- **Wayland**, which is Fyne's default on KDE Plasma 6 (`loop_desktop.go`):
  `wl_data_device.start_drag(source, origin, icon, serial)` needs the serial
  of the button press that began the drag. GLFW keeps that serial in its
  private library state. `glfw.GetWaylandDisplay()` gives the display, and
  `RunNative` gives the surface. The package binds its own `wl_seat` and
  `wl_pointer` on GLFW's display. Those proxies sit on the default queue,
  which GLFW already dispatches, so the package sees the same button serials
  GLFW does. Everything used is in the core protocol (`wayland-client.h`), so
  no scanner and no generated code are needed.
- **X11**, also used for XWayland and `FYNE_PLATFORM=x11`: an XDND source on
  a second `Display` connection, driven on its own goroutine with its own
  pointer grab. The XDND and `SelectionRequest` traffic never reaches GLFW's
  event loop. GLFW handles `SelectionRequest` for the clipboard and would
  answer `XdndSelection` wrongly.
- **Windows**: OLE `DoDragDrop` with a `CF_HDROP` data object and
  `DROPEFFECT_COPY`. `DoDragDrop` runs its own modal message loop on the
  calling thread, which is the GLFW main thread.
- **macOS**: `-[NSView beginDraggingSessionWithItems:event:source:]` on the
  window's content view. The event is `[NSApp currentEvent]`, which is the
  mouse-dragged event Fyne is handling, and each item is an `NSURL`.

## Requirements

- R1 Package `dragout`. `dragout.New(content fyne.CanvasObject, paths
  func() []string) *Source` wraps `content`. When the pointer is pressed on it
  and moves past a threshold, it calls `paths` and starts an OS drag of those
  files. Taps, hover and secondary taps still reach `content`.
- R2 The drag offers files, never bytes or text. The data is `text/uri-list`
  on Wayland and X11 (`file://` URIs per RFC 8089, percent-encoded, CRLF
  separated), `CF_HDROP` on Windows and file `NSURL`s on macOS. The only
  operation offered is copy. On Wayland the compositor holds the target to
  the offered actions, so the source file is never moved. On X11 the XDND
  action is advisory: a target that lets the user choose (Dolphin's drop
  menu) can move the file itself, and the source cannot prevent it. A
  consumer must survive the file disappearing.
- R3 Only the paths are checked: each must be absolute and name an existing
  regular file when the drag starts. Any path that fails is left out, and if
  none remain no drag starts. `paths` returning nil or empty is how a
  consumer says "nothing to drag" (for example, an error shown in place of
  the image).
- R4 `dragout.Supported() bool` reports whether this build and session can
  start a drag. Under the test driver, without cgo, or on an unsupported
  platform it is false, and `Source` never starts a drag. It behaves as a
  plain container around `content`.
- R5 `Source.OnFailed func(error)` is optional and is called on the UI
  thread when a drag could not start (no seat, no serial, OLE failure). The
  library does not log.
- R6 The window stays usable after a drag, whether it was dropped, cancelled
  with Escape, or dropped on nothing. The next click, hover and keypress
  reach the widgets without an extra click to "release" Fyne's drag state,
  because the platform takes the pointer and Fyne never sees the release.
  The fix is recorded in `docs/fyne-quirks.md` with a canary test.
- R7 The drag icon is the platform default (the cursor, or the file icon on
  macOS and Windows). Drawing a thumbnail under the cursor is out of scope.
- R8 The only new runtime libraries are ones Fyne's GLFW already links
  (`libwayland-client`, `libX11`, `ole32`, `AppKit`). The direct
  `github.com/go-gl/glfw/v3.4/glfw` import already exists (`widgets`,
  `glance`).
- R9 The gallery has a section with a sample file to drag out. The README
  package table, changelog and `docs/design-system.md` are updated in the
  same commit.

Delivery is in two phases on one branch, each its own PR. Phase 1 is Linux
(Wayland and X11). Phase 2 is Windows and macOS. Until phase 2 lands,
`Supported()` is false there.

## Acceptance Criteria

Phase 1:

- [x] R2 `text/uri-list` encoding: spaces, `#`, `%`, non-ASCII and a
      path with a newline are encoded so they round-trip through
      `url.Parse` (unit test). `TestAURIListRoundTripsAwkwardPaths`.
- [x] R3 Relative, missing, directory and symlink-to-directory paths are
      dropped; an all-invalid list starts no drag (unit test against a fake
      backend).
- [x] R1/R4 Under the test driver `Supported()` is false; a `Source`
      dragged in a test window never calls the backend and still passes taps
      to `content` (headless test).
- [x] R1 The threshold: a press and a move under the threshold is not a
      drag (headless test, fake backend).
- [x] R2 KDE Plasma 6 Wayland: dragging a file from the gallery into
      Dolphin copies it; the source file is unchanged (manual, recorded in
      the PR).
- [x] R2 Same in an X11 session or with `FYNE_PLATFORM=x11` (manual).
      Verified with `FYNE_PLATFORM=x11` under XWayland on Plasma 6, not in
      a native X11 session.
- [x] R2 A drop into Firefox's file upload field attaches the file
      (manual, Wayland).
- [x] R6 After a drop, after Escape, and after a drop on the desktop, the
      next click on a gallery button works the first time (manual, both
      backends; canary test for the Fyne state).
- [x] R9 Gallery demo (`dragout.Source` in the Widgets section), README
      rows, changelog, design-system entry.

Phase 2:

- [ ] R2 Windows 11: drag into Explorer copies the file (manual).
- [ ] R2 macOS 13+: drag into Finder copies the file (manual).
- [ ] R6 Both platforms: the window is usable after drop and cancel
      (manual).

## Risks & Assumptions

- **GLFW internals on Wayland.** The serial approach assumes GLFW
  dispatches the default queue, as GLFW 3.4's `wl_init.c` does. A GLFW change
  to a private queue would stop the package's `wl_pointer` seeing events. The
  symptom is `OnFailed` reporting "no serial", not a crash. The Fyne-pin
  canaries cover this: a Fyne or GLFW bump re-runs the manual Wayland AC.
- **X11 targets may move.** Found in manual testing: Dolphin under
  XWayland offered Move and moved the gallery's sample away; the next drag
  found nothing (R3, silently). The gallery now rewrites its sample at each
  drag. Windows (`DROPEFFECT_COPY` only) and macOS (copy-only operation mask)
  are expected to enforce copy as Wayland does; phase 2 checks.
- **Compositor differences.** Only KDE Plasma 6 is tested. GNOME/Mutter is
  assumed to accept the same core-protocol calls.
- **R6 is the likeliest bug.** When the platform takes the pointer
  mid-press, GLFW never sees the release. If Fyne cannot be reset through
  public API, R6 needs a narrow workaround documented as a quirk.
- **cgo.** `make test` stays cgo-free: backends are behind `cgo` build tags,
  and the tests use a fake backend.
- **Rollback**: additive package; revert the commit. No existing API
  changes.

## E2E Test Plan

- Environment: the gallery built with `make gallery` on KDE Plasma 6 Wayland,
  then the same binary with `FYNE_PLATFORM=x11`. Phase 2: the gallery on the
  Windows 11 VM and on macOS.
- Steps: drag the gallery's sample file into Dolphin (Explorer, Finder), then
  into a Firefox upload field. Cancel a drag with Escape. Drop one on the
  desktop. After each, click a gallery button once.
- Expected: the file is copied, the upload field shows it, and the source is
  unchanged (compare `sha256sum`). The button reacts to the first click
  every time.
- Covers the R2 and R6 manual criteria. clockwork-orange #40 repeats the
  Dolphin step from its review pane.
