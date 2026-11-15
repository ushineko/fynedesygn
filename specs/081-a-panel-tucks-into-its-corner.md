# Spec 081: a panel tucks into a square at its corner

**Issue**: [#241](https://github.com/ushineko/fynedesygn/issues/241)

## Status: INCOMPLETE

## Executive Summary

(Filled in before the PR.)

## Context

A glance panel stays on top, and while it covers something a person is
working on there is no way to put it aside short of quitting or dragging it
off and back. hayami (ushineko/hayami#328) asks for a control that shrinks
the panel to a small square carrying the program's icon, at the panel's lower
corner, which brings the panel back where it was when clicked.

## Requirements

### Tucking

- R1 **`Panel.Tuck(left, right float32)`** replaces the window's content with
  a square bearing the application's icon (`fyne.CurrentApp().Icon()`, or
  one set with `Panel.SetTuckIcon`), `TuckSizeInEms` text sizes on a side
  (3), on the card surface's colour and radius, and resizes the window to it.
  `left` and `right` are the room beside the window, as for `Fits`.
- R2 **`Panel.TuckHere() (known bool)`** is `Tuck` with the room read by the
  window itself where it can (Windows); where it cannot, the program asks the
  compositor and calls `Tuck`.
- R3 **The square stands at the cards' bottom corner on the screen-edge
  side**: with drawers open, the corner away from them; with none, the corner
  on the side with less room. The cards' bottom edge and that side edge are
  where the square's are.
- R4 **The drawers are not closed.** They are hidden with the cards, keep
  their row, content and state, and come back with the cards.
- R5 **`Panel.Untuck()`** puts the cards and drawers back and the window to
  the size it had, holding the square's corner, so a square that was not
  moved brings the cards back where they were to the pixel. A tap on the
  square calls it.

### Moving the square

- R5a **The square moves the way the panel does**: on Plasma by Meta-drag,
  which is the compositor's and cannot be refused; on Windows by a press-drag
  on the square (spec 068's `beginMove`), a press that does not move being the
  tap that untucks. It snaps to the work area's edges as the panel does.
- R5b **A moved square brings the panel back at its new place**: the corner
  the square was tucked into stays the anchor, and the panel unfolds from
  wherever the square now is.
- R5c **The panel never unfolds off its screen.** Where the panel at the
  anchor would cross the work area's edge, it is moved in until it does not
  (on Plasma inside the hold-corner script, which knows the work area; on
  Windows by the window itself), and the corner is not held for that
  unfolding.
- R6 **`Panel.Tucked() bool`**, and **`Panel.SetOnTuck(func(tucked bool))`**
  is called after each change, so a program stops and starts its readings.
- R7 While tucked, `Resize` keeps the square's size (a reading arriving does
  not grow it), `Restyle` resizes the square for the text size, and drawer
  and card calls change the hidden content without resizing the window.

### Holding the corner

- R8 On Windows the window moves itself, as a left drawer does.
- R9 **On Plasma, `kwin.HoldCornerScript(t, corner)`** holds a bottom corner
  of the window while it is resized (Plasma 6.8 otherwise keeps the centre,
  #236), as the hold-edge scripts hold an edge; **`Panel.SetCornerHeld`**
  says it is loaded. A panel whose corner is not held tucks at the window's
  top-left and says so in `Tuck`'s result (`false`), so a program can choose
  not to offer it.

### The control

- R10 **`Panel.SetTuckable(on bool)`** shows a small glyph (a minimise
  mark) in the cards' top corner on the screen-edge side when the pointer is
  over the cards, and hides it when the pointer leaves; a tap on it calls the
  program's tuck handler (`SetOnTuckAsked(func())`), which asks the room and
  calls `Tuck`. The glyph does not take room from the cards: it is drawn over
  the first card's heading corner, and the heading text is never under it
  (a heading that would be is truncated short of it).

### Documentation and the example

- R11 `docs/glance.md`: a section on tucking.
- R12 The glance-monitor example tucks and untucks.

## Acceptance Criteria

Headless, each falsified:

- [ ] `Tuck` makes the window `TuckSizeInEms × textSize` square and shows the
  icon; `Untuck` restores the size it had, drawers open
  (`TestTuckAndBack`).
- [ ] The corner is the one away from the drawers with drawers open, and the
  side with less room without (`TestTheCornerIsTheScreenEdges`).
- [ ] Drawers open before tucking are open, in the same order and with the
  same content, after untucking (`TestTuckKeepsTheDrawers`).
- [ ] `Resize` while tucked keeps the square (`TestAReadingDoesNotUntuck`).
- [ ] `SetOnTuck` is called with true then false (`TestOnTuck`).
- [ ] A tap on the square untucks; a tap on the hover glyph asks to tuck;
  the glyph shows only while hovered (`TestTheGlyphAndTheSquare`).
- [ ] `HoldCornerScript` is quoted and names the corner
  (`TestHoldCornerScript`).

On a real window (Plasma 6.8 Wayland at 1.5, Windows 11 at 150 %), the
glance-monitor example, at both screen edges and with drawers on each side:

- [ ] Tucked: the square's bottom and side edges are the cards' to the pixel.
- [ ] Untucked: the cards' rectangle recovered from the grab is the one
  before tucking, and the drawers are back.
- [ ] The square moved (Meta-drag on Plasma, press-drag on Windows), then
  untucked: the cards' screen-edge bottom corner is the square's; moved into
  the opposite corner of the screen, the panel unfolds inside the work area.
- [ ] On Windows a press on the square that does not move untucks; one that
  moves does not.

## Risks & Assumptions

- A KWin rule with a minimum size would refuse the square. The panel's rule
  sets none; checked on the desk.
- **Rollback**: revert; additive API.

## Threat model delta

B5 (outside itself): one more KWin script, `HoldCornerScript`, written and
loaded the way the hold-edge scripts are (a new `0600` file, strings quoted,
acting only on the panel's window). No change for an attacker.
