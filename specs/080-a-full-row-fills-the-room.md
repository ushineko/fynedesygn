# Spec 080: a full row of drawers fills the room left beside it

**Issue**: [#240](https://github.com/ushineko/fynedesygn/issues/240)

## Status: COMPLETE

## Executive Summary

A row of drawers at its cap can fill the room left beside it: `Panel.Fill`
widens every open drawer to the same whole number of device pixels until the
window reaches the work area's edge, the remainder padding beyond the
outermost, and `SetFillable` puts a Fill button on the outermost drawer. Any
change of the row ends it. Tried on Plasma and on Windows 11
through hayami. A reviewer should look first at `planFill` and
`placeFilled` in `glance/fill.go`.

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

Headless, each falsified: eleven mutations each failed a test. Of two more
for what the desk found, the rounding failed `TestAWholePixelIsNotRoundedUp`;
the cards stood on the planned pixels is not seen at a test canvas's scale of
1, and was checked on the desk:

- [x] Two drawers, 101 of room on their side, scale 1: `Fill` makes both
  drawers the same width and the window 101 wider; the padding is under 2
  (`TestFillSharesTheRoomEqually`).
- [x] At a scale of 1.5, three drawers, every room from 1 to 300 in steps of
  0.37: the pixels add up, the padding is fewer than three, and the window
  is the one it was and the room's whole pixels
  (`TestFillAtAFractionalScale`).
- [x] Cards a whole number of pixels wide are planned at that number, not
  one more (`TestAWholePixelIsNotRoundedUp`).
- [x] The cards' size and place are the same before `Fill`, after it and
  after `Unfill`, drawers on either side; on the left the drawers stand a
  gap apart from the cards outwards and the padding is at the far end
  (`TestFillLeavesTheCards`).
- [x] Closing one drawer of three while filled ends filling and resizes once,
  and `CloseDrawers` ends it (`TestClosingADrawerEndsFilling`).
- [x] `SetContent` on a filled drawer keeps it filled
  (`TestNewContentKeepsTheRowFilled`).
- [x] `Fill` with no drawer open, or less than a pixel of room on the
  drawers' side, returns false and changes nothing
  (`TestFillWithNothingToFill`).
- [x] The Fill button is on the outermost drawer only, follows it when the
  row changes, is drawn engaged while filled, and a tap calls the toggle
  (`TestTheFillButtonIsOnTheOutermost`).
- [x] `DrawersWidth` equals the window's width less the cards', filled and
  not (`TestDrawersWidth`); `Restyle` ends filling (`TestRestyleEndsFilling`).

On a real window. The example reads the room only where the window can
(Windows); on Plasma the room is the compositor's, so the check there was
made through hayami (spec 090), which asks it:

- [x] Plasma 6.8 at 1.5, filled on the right and on the left: the window's
  outer edge is the work area's, each drawer's width recovered from the grab
  is equal (563 x 3, 580 x 5), the cards did not move.
- [x] Plasma: unfilled, and with every drawer closed while filled: the widths
  are the drawers' own again and the cards did not move.
- [x] Windows 11 at 150 %: the same, through hayami 0.15.0 (this
  release), where the window reads the room itself (`FillHere`): four
  drawers at the cap, filled, the window's right edge on the screen's at
  3840, the drawers' headings 592, 592 and 591 pixels apart (579-pixel
  drawers and 13-pixel gaps), the cards at x 1100 throughout; unfilled and
  closed, the widths the drawers' own again (hayami spec 090).

## Found on the desk

The first build planned cards 289.333 wide at a scale of 1.5 at 435 pixels:
289.333 in float32 times 1.5 is 434.00001, and rounded up it was 435. The
cards' left edge moved a pixel when the row filled. The rounding up now
forgives what float32 adds, and a filled row stands the cards on the pixels
it planned for them, as it stands the drawers (`drawerLayout`).

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
