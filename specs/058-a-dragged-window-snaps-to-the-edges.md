# Spec 058: a dragged window snaps to the edges

**Issue**: [#175](https://github.com/ushineko/fynedesygn/issues/175)

## Status: COMPLETE

## Context

On Linux a glance window snaps to the screen's edges as it is dragged. The
window does nothing for that: KWin moves it (Meta-drag, or the window rule's
placement), and KWin's border snap zone, 10 px by default, pulls a window it
is moving onto an edge.

On Windows the window moves itself. Spec 052 hands a primary press anywhere on
a glance window to the system's move loop as a press on the caption
(`WM_NCLBUTTONDOWN`, `HTCAPTION`, `glance/move_windows.go`). That loop has no
edge pull. Aero Snap's docking (half screen, maximise) applies to a window
with a sizing frame and is not what a glance panel wants. So on Windows a
panel dropped beside an edge stayed a few pixels off it, which hayami's 0.9.x
sit test noticed: the panel did not snap the way it does on the Linux desk.

## Requirements

- R1 During the move loop the window answers `WM_MOVING`, whose `lParam` is
  the rectangle the window is about to take and may be changed: an edge of
  that rectangle within the snap distance of the same edge of the work area
  of the monitor it is over (`MonitorFromRect`, nearest; `GetMonitorInfo`'s
  `rcWork`, so the taskbar is left out) is moved onto that edge, the window
  keeping its size. Each axis is decided alone, so near a corner both snap.
  Anywhere else the rectangle is unchanged and the window follows the pointer
  exactly.
- R2 The snap distance is KWin's 10 px at 96 DPI, scaled for the window's
  DPI (`GetDpiForWindow`; 96 when the system gives none), so the pull is the
  same physical distance on a scaled screen.
- R3 The arithmetic is platform-free and tested everywhere: edges, the exact
  distance and one beyond it, corners, the taskbar's edge rather than the
  screen's, a monitor with negative coordinates, a window near both sides,
  DPI scaling.
- R4 GLFW owns the window procedure. The window's procedure is replaced for
  that one window (`SetWindowLongPtrW`, `GWLP_WNDPROC`) by a wrapper that
  adjusts `WM_MOVING` and passes every message to the procedure it replaced
  (`CallWindowProcW`); on `WM_NCDESTROY`, the last message a window gets, it
  puts the original back. It is installed once, from `startMove`, the first
  time the window is dragged.
- R5 Linux and macOS are unchanged: the window manager snaps there.
- R6 docs/glance.md says how a drag snaps on each desktop; the Platform rules
  in docs/design-system.md say a dragged window should snap as the user's
  desktop does. README changelog under `### Unreleased`.

## Acceptance Criteria

- [x] The arithmetic's table tests pass on every platform (R1–R3).
- [x] On a real window on Windows 11, a glance window dragged with the real
  pointer to 6 px from the left and 4 px below the top of the work area ends
  exactly on both edges.
- [x] The same window dropped away from every edge ends exactly where it was
  dropped.
- [x] Falsified: without the wrapper the window does not reach the edge; with
  a snap distance larger than the screen the middle drop is pulled to the
  corner. Restored, both pass.
- [x] `go test -race ./...`, `go vet` and golangci-lint pass on Windows.

## Risks & Assumptions

- **comctl32's `SetWindowSubclass`** would do the bookkeeping, but it is
  exported by name only from comctl32 version 6, which a Go program loads
  only with a manifest it does not carry. The `SetWindowLongPtrW` wrapper
  needs none and chains to whatever procedure the window had, GLFW's or a
  later wrapper's.
- **The wrapper is a Go callback** (`syscall.NewCallback`), made once for the
  process because the runtime has a fixed number of them; a map from window
  to the replaced procedure lets it serve any number of windows.
- **A program that wraps the same window's procedure later** chains to this
  wrapper, which is fine; one that wrapped it earlier and expects to be the
  outermost would not be, and none in the module does.
- **Rollback**: revert. The drag works as before without the wrapper.

## Gaps found

- Snapping to other windows' edges, which KWin also does, is not done: a
  glance panel lives against a screen edge, not against another window.

## Verification

2026-10-08, Windows 11 Pro 26200, a 3840x2160 monitor at 150 % (work area
3840x2160, the taskbar hidden), Go 1.26.0, the glance-monitor example built by
the test and dragged with the real pointer (`FYNEDESYGN_WINDOW_TEST=1`):

- The window at (1695, 1079); dragged so its top-left would be at (6, 4); it
  ended at (0, 0).
- Dragged so its top-left would be at (1695, 1025); it ended at (1695, 1025).
- Without the wrapper (`snapEdges` not called): the first drag ended at
  (-48, 143), off the edge, and the test failed.
- With the snap distance raised to 2000 px: the middle drop ended at (0, 0)
  and the test failed.

### Addendum: a snapped window stuck to the edge

Found in hayami's 0.9.2 sit test: once the panel snapped onto the top or the
bottom edge it stayed there whatever the drag did, and came off only after
being released and dragged again, several times.

- **Cause.** The first version snapped the rectangle `WM_MOVING` proposed.
  Windows builds each proposal from the rectangle as the last message left it,
  plus the pointer's movement since, so after a snap every small movement
  started from the edge, stayed within the snap distance and was snapped back.
  The test missed it because its synthetic drag moved about 33 px a step, past
  the snap distance in one message; a hand moves a few.
- **Fix.** The window's place is worked out from the pointer: the point it is
  grabbed by is taken from the move's first `WM_MOVING` (whose proposal is
  still Windows' own), and every later message puts the window there relative
  to the pointer's position for that message (`GetMessagePos`), then snaps it.
  The pull is now from inside only: an edge pushed past the work area's goes
  with the pointer, part way off the screen or on to the next monitor, as
  asked for in the sit test.
- **Tests.** `TestASnappedGlanceWindowComesOffTheEdgeInTheSameDrag` drags onto
  the top edge and on to the middle of the screen in one drag, in steps of at
  most 3 px. Before the fix it ended at (1695, 0), stuck on the edge; after,
  at (1695, 1025), under the pointer. The table tests gain pushing past an
  edge and a corner and crossing to the next monitor; restoring the symmetric
  pull fails three of them. The corner and middle drops still land at (0, 0)
  and (1695, 1025). Not tested on two monitors (one on the desk); the
  crossing is covered by the arithmetic.
