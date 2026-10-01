# Spec 049: a position script that waits for the window

**Issue**: [#152](https://github.com/ushineko/fynedesygn/issues/152)

## Status: COMPLETE

## Context

`kwin.PositionScript` (spec 037) sets `frameGeometry` on a window that is
already on screen. A program that restores its position with it has to run
it after its window exists, and a toolkit does not say when that is: hayami
slept 600 ms. Traced on the desk on 2026-09-30, a cold Fyne window took
2.15 s to appear; the script ran against an empty list, unloaded, and the
panel opened centred on the other monitor (ushineko/hayami#106).

The compositor knows the moment. `workspace.windowAdded` is the event, and
`WatchGeometryScript` already stays loaded to connect to it.

## Requirements

- R1 `kwin.PlaceScript(appID, x, y)` is a script that places a window of
  the class if one is on screen, and otherwise connects to
  `workspace.windowAdded` and places the first window of the class that
  appears. The size is the window's own, as `PositionScript` keeps it.
- R2 It places one window and no more: a guard stops a second window of the
  class, which on Wayland is a program's preferences window, from being
  moved onto the first; the handler disconnects once it has placed one.
- R3 The script stays loaded until the program unloads it, like
  `WatchGeometryScript`. Having placed a window it does nothing more.
- R4 Its strings cannot escape their literals, as the other scripts' cannot.
- R5 `docs/glance.md` names it as the way to put a window back where it was.

## Acceptance Criteria

- [x] R1 (`TestThePlaceScriptPlacesNowOrWhenTheWindowAppears`)
- [x] R2 (`TestThePlaceScriptPlacesOnlyTheFirstWindow`)
- [x] R4 (`TestAPositionScriptCannotBeEscapedByItsStrings`, extended)
- [x] R5 (`docs/glance.md`, Desktop integration)
- [ ] On the desk: hayami 0.8.2 built on this script, restarted cold, comes
      up at its saved position (recorded in hayami spec 029)

## Risks & Assumptions

- A script's text is asserted, not its behaviour; KWin is not in the test.
  The behaviour is checked on the desk in the consumer, as spec 037's was.
- `frameGeometry` at `windowAdded` is assumed settled enough to keep its
  width and height. KWin emits the signal once a Wayland window is managed,
  after its first buffer, so the size is the client's. Checked on the desk.
- `PositionScript` is unchanged; nothing that calls it moves.
