# Spec 080: a full row of drawers fills the room left beside it

**Issue**: [#240](https://github.com/ushineko/fynedesygn/issues/240)

## Status: INCOMPLETE

## Executive Summary

(Filled in before the PR.)

## Context

Spec 079 caps a row of drawers by the room on the screen: a drawer opens only
where its `Reach` fits (`Panel.Fits`), and a program puts the next detail in
the outermost drawer once none does. The row then stops short of the screen's
edge by anything up to one `Reach`, and that strip is unused while the
drawers it could widen are scrolled or truncated.

hayami (ushineko/hayami#327) asks for a way to fill it: every open drawer
widened by the same amount, the cards left as they are.

## Requirements

### Filling

- R1 **`Panel.Fill(left, right float32) bool`** widens the open drawers so
  the window reaches the edge of the work area on the drawers' side, given
  the room between the window and each edge (as `Fits` takes it). It returns
  false and changes nothing when no drawer is open or the room on that side
  is not positive.
- R2 **`Panel.FillHere() (ok, known bool)`** is `Fill` with the room read by
  the window itself, where it can (Windows); where it cannot, `known` is false
  and the program asks the compositor (on Plasma `kwin.ReportRoomScript`, as
  for `Fits`).
- R3 **`Panel.Unfill()`** puts every open drawer back to its own width;
  **`Panel.Filled() bool`** reports the state.
- R4 **Every open drawer is exactly the same width, in device pixels.** The
  drawers' share of the window is worked out at the window's scale: the
  window's width in pixels as Fyne makes it (`ceil(width × scale)`, the rule
  `pixelsMoved` follows, #238), less the cards' pixels and each gap's, divided
  by the number of drawers. **What the division leaves over, fewer pixels than
  there are drawers, is padding beyond the outermost drawer**, on the side
  away from the cards; it is the panel's background, not drawer.
- R5 The cards' width and place are unchanged by filling and unfilling: on
  the left the window grows leftwards as for a drawer opening (the edge held,
  `growsLeft`), and a panel that cannot grow leftwards fills only on the
  right.
- R6 The height is unchanged.

### When it ends

- R7 **Closing a drawer, or `CloseDrawers`, ends filling**, in the same
  change of shape: the drawers left are at their own width and the window is
  resized once.
- R8 Changing what an open drawer shows does not end it (the program's
  replace-at-the-cap of spec 079 R9 keeps the row filled).
- R9 Opening a drawer while filled is refused by `Fits` (there is no room)
  and, opened anyway, ends filling first.
- R10 A change of text size (`Restyle`) ends filling: the widths were worked
  out from the old one. So do cards that change width: the window would run
  off the screen or short of its edge.

### The control

- R11 **`Panel.SetFillable(on bool, toggle func())`** shows a button in the
  outermost open drawer's heading, beside its other buttons (spec 063),
  reading "Fill" and, engaged, "Fill" drawn active; a press calls `toggle`.
  The button is on the outermost drawer only, moves to the new outermost when
  the row changes, and is hidden with `on` false. The program decides when the
  row is at its cap, because only it can ask the compositor; the panel does
  not ask on its own.

### Place

- R12 **`Panel.DrawersWidth() float32`** is what the open drawers add to the
  window's width now, filled or not, so a program that works the cards'
  corner out of a window's width (hayami's `drawerShift`) can recognise a
  filled row.

### Documentation and the example

- R13 `docs/glance.md`: the drawers' section says how to fill a full row.
- R14 The glance-monitor example gets a Fill button at the cap.

## Acceptance Criteria

Headless, each falsified:

- [ ] Two drawers, room 101 on their side, scale 1: `Fill` makes both
  drawers the same width and the window 101 wider; the padding is 1 px
  (`TestFillSharesTheRoomEqually`).
- [ ] At scale 1.5, three drawers, room chosen so the division leaves 2 px:
  each drawer's width in pixels is equal and the padding is 2 px
  (`TestFillAtAFractionalScale`).
- [ ] The cards' size and place are the same before `Fill`, after it and
  after `Unfill`, drawers on either side (`TestFillLeavesTheCards`).
- [ ] Closing one drawer of three while filled ends filling and resizes once
  (`TestClosingADrawerEndsFilling`); `CloseDrawers` likewise.
- [ ] `SetContent` on a filled drawer keeps it filled
  (`TestNewContentKeepsTheRowFilled`).
- [ ] `Fill` with no drawer open, or no room, returns false and changes
  nothing (`TestFillWithNothingToFill`).
- [ ] The Fill button is on the outermost drawer only, follows it when the
  row changes, and a tap calls the toggle (`TestTheFillButtonIsOnTheOutermost`).
- [ ] `DrawersWidth` equals the window's width less the cards', filled and
  not (`TestDrawersWidth`).

On a real window (Plasma 6.8 Wayland at 1.5, Windows 11 at 150 %), the
glance-monitor example:

- [ ] Filled on the right and on the left: the window's outer edge is the
  work area's to the pixel, each drawer's width recovered from the grab is
  equal, the cards did not move.
- [ ] Unfilled, and with a drawer closed while filled: the widths are the
  drawers' own again and the cards did not move.

## Risks & Assumptions

- The room reported by the compositor is accurate to the pixel; spec 079
  relied on it to the drawer and this relies on it to the pixel. If it is off
  by the window's border the window would run a pixel off its screen; the
  real-window check measures it.
- A KWin hold-edge script is loaded for the side as for a drawer (#236); a
  fill is one more resize through it.
- **Rollback**: revert; the API is additive and nothing calls it until a
  program does.

## Threat model delta

No boundary touched: the room is the one already reported, and no new script
or file is written.
