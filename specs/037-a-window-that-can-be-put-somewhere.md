# Spec 037: a window that can be put somewhere

**Issue**: [#77](https://github.com/ushineko/fynedesygn/issues/77)

## Status: COMPLETE

## Executive Summary

`kwin.PositionScript` moves a glance window that is already on screen and
`kwin.ReportGeometryScript` reads its true geometry back over the session bus,
which are the only two routes that work: `desktop.Window.RequestPosition` was
measured as a no-op on Plasma 6 on Wayland. `docs/glance.md` gains the
measurement, the distinction from a rule's `position`, and the sentence it was
missing — a person moves a frameless window with Meta and drag. Reviewers
should start with the measurement in "Context", then
`TestAPositionScriptCannotBeEscapedByItsStrings`, which is the only part with
a way to go wrong.

## Context

A glance window opens where the compositor puts it and stays there. Three
things a panel like this wants, measured on Plasma 6 on Wayland:

| Route | Result |
|---|---|
| `desktop.Window.RequestPosition(900, 400)` | the window did not move a pixel |
| A rule's `position`, forced | places the window, at its creation, on one screen (spec 035) |
| KWin scripting, `w.frameGeometry` | moved the same window to 900,400 while it was up |

Fyne's doc comment on `RequestPosition` says the request "may be ignored (for
example Linux Wayland)". On this compositor it is not ignored sometimes. Qt
hits the same wall from the other side, which is why
`ag-scripts/peripheral-battery-monitor/kwin_window_position.py` drives the
scripting API rather than calling `move()`, and has done in production for a
year.

Reading the position back has no alternative at all. Fyne exposes none
(quirk 33), and on Wayland every move belongs to the compositor — the user's
own Meta-drag included — so nothing fires that a toolkit can observe. The
monitor found the same from Qt, where `moveEvent` does not fire for a
compositor move.

Spec 035 deferred scripting helpers on the grounds that the program makes the
calls with the D-Bus client it already has. That reasoning holds for the
calls and not for the script text, which is what spec 036 settled when it
added `OpacityScript`. This follows that shape rather than inventing another.

Separately, `docs/glance.md` told a reader that a frameless window cannot be
moved and stopped there, which reads as "you cannot move it". Meta and drag is
the compositor's own fallback, goes around the client, and works on KDE and
GNOME. Confirmed by hand on the running panel.

## Requirements

- R1 `kwin.PositionScript(appID, x, y)` moves every window of an app id to a
  point in the compositor's coordinates and keeps the window's size.
- R2 `kwin.ReportGeometryScript(appID, service, path, iface, method)` calls
  the window's x, y, width and height back to a method the program exports.
- R3 Neither script can be escaped by the strings it interpolates.
- R4 `docs/glance.md` records the three measurements, says which tool is for
  which question, and tells a reader how a person moves the window.

## Acceptance Criteria

- [x] R1: `TestThePositionScriptMovesAndKeepsTheSize` and `TestThePositionScriptTakesAScreenToTheLeftOfTheOrigin`. Measured on the running panel: 3147,1025 to 900,400.
- [x] R2: `TestTheReportScriptCallsBackWithTheFourNumbers`.
- [x] R3: `TestAPositionScriptCannotBeEscapedByItsStrings`, which removes every quoted literal and asserts the payload is not in what is left. An assertion on the raw script cannot say this: the payload is present either way, and the first version of this test failed a correct implementation for that reason.
- [x] R4: the document read; the style canaries pass.

## Risks & Assumptions

- The size is carried through from `frameGeometry` rather than set. A glance
  window is sized by its content, and a script that set a size would fight
  `Panel.Resize` on the next card that appears.
- `callDBus` requires the program to own a bus name and export the method
  before the script runs. That is the program's half and the library cannot
  check it; a report that goes nowhere is silent.
- One compositor, one version, as with every measurement on this page.
- Rollback: revert the commit. Both identifiers are new and nothing in the
  module calls them.

## Alternatives Considered

- A `Position` field on `kwin.Rule`. It works, and spec 035 already declined
  it: a position in a rule is fixed to one screen and applied once, so it
  cannot follow a window to the pointer's screen or restore a drag.
- A D-Bus client in `glance/kwin` to run the scripts. Declined for the third
  time, for the reason `ReconfigureCall` documents: the library keeps to Fyne
  as its only dependency, and the program owns the bus.
- Waiting for Fyne to fix `RequestPosition`. There is no issue open for it and
  the hedge is in its documented contract.
