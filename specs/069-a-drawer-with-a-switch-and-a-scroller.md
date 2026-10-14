# Spec 069: a drawer with a switch, and a region that scrolls

**Issue**: [#207](https://github.com/ushineko/fynedesygn/issues/207)

## Status: INCOMPLETE

## Context

A drawer shows one detail, chosen by the click that opened it (spec 068 R11,
R15). hayami's cooler card has three (processor, graphics card, memory), and
choosing by the row clicked has two costs: a click anywhere else on the card
means only one of them, and changing the list means going back to the card.
hayami wants the choice in the drawer (hayami #197).

Its usage card wants every account's quotas at once. That can be taller than
the cards, and the drawer has no answer: it never sets the window's height
(R2), and R2 also says "No scroller".

One fact of the package stands in the way of the switch: a click inside the
drawer reaches nothing. Nothing under the catcher is clicked directly (quirk
24), and `Panel.tapped`, which finds what is under the pointer for it, walks
the cards only. The wheel is not caught: Fyne looks for the last `Scrollable`
under the pointer (`processMouseScrolled`), the catcher is not one, and a
scroller in the drawer is.

## Requirements

### The switch

- R1 `glance.Switch` is a piece for a drawer: one line of named choices, one
  of them selected. `SetChoices([]Choice)` names them (`Choice{ID, Label,
  Key}`), `Select(id)` and `Selected()` set and read the selection, and
  `OnChanged(id)` tells the program when a person changes it. `Select` from
  the program does not call `OnChanged`.
- R2 The selected choice is drawn in the foreground colour and bold, the
  others in the placeholder colour. Each choice is as wide as its label in
  bold whether selected or not, so changing the selection moves nothing. The
  line is one row's height.
- R3 `Drawer.SetSwitch` wires it, as `SetFilter` wires a filter; the switch is
  drawn where the program put it in `SetContent`. A primary click on a choice
  selects it. A choice's `Key`, typed while the drawer is open, the panel
  holds the keyboard and no filter is being typed, selects it. A filter being
  typed takes every key, as now.
- R4 Selecting the choice already selected does nothing and reports nothing.
  A switch with fewer than two choices is not drawn and takes no height.
- R5 A click in the drawer that is not on a choice is not reported: the
  drawer has no other control, and `Panel.OnTapped` stays a report about
  cards.
- R6 The library does not decide what a choice means. What the drawer shows
  after `OnChanged` is the program's.

### The region that scrolls

- R7 `Drawer.SetScrolling(true)` makes the content under the heading scroll
  vertically when it is taller than `Available()`. Content that fits is drawn
  exactly as without it. Off is the default and is spec 068's drawer.
- R8 The drawer's height is still the cards' and it still never sets the
  window's (spec 068 R2). R2's "No scroller" becomes "a list is cut to the
  height; a breakdown may scroll": `Fit` stays, and a filtered list is still
  the way to reach a name far down.
- R9 The scroll bar is drawn over the content's right margin and only while
  the content is taller than the room. It takes no width, so content does
  not change width when it starts to scroll.
- R10 The wheel scrolls it while the pointer is over the drawer. With the
  drawer open and no filter being typed, Up, Down, Page Up, Page Down, Home
  and End scroll it too. The bar is not dragged: a press on the panel is the
  catcher's (quirk 24) and on Windows the window's (spec 052).
- R11 `SetContent` puts the offset back to the top. Setting pieces in place
  keeps it, clamped to the new height, so a reading that arrives does not
  move what a person is looking at.
- R12 `Drawer.SetOpen(false)` puts the offset back to the top.

### A rule, and where the cards are

Added while hayami was built on this, for what it found it needed.

- R13 `glance.Rule` is a piece: one line across its room in the colour of a
  card's border, in half a line of height, for between groups of readings.
- R14 `Panel.CardsSize()` is the size of the cards without the drawer, a
  dragged width included, and `Drawer.Reach()` what the drawer adds to the
  window, open or not. A program that remembers its panel's place works out
  the cards' corner from them and the window's width.

### Documentation and the gallery

- R15 docs/glance.md, "A drawer beside the cards": one target for each card,
  the switch, when content may scroll and when a list should be cut, the
  rule, and the position of the cards.
- R16 `examples/glance-monitor` opens a drawer from anywhere on a card. Its
  thermal card's two lists are a switch (CPU, Coolant) and its usage card's
  drawer is four accounts of quotas with rules between them, scrolled.
- R17 README: the changelog entry and the example's row. The gallery's
  picture of the drawer is of the bandwidth list, which has no switch and
  does not scroll, and is not taken again.

## Acceptance Criteria

Headless, each falsified:

- [x] A click on a choice selects it and calls `OnChanged` once with its ID;
  a click on the selected one, on the heading or on a line calls nothing,
  and none of them reaches `Panel.OnTapped`
  (`TestAClickOnAChoiceSelectsIt`, `TestSelectingFromTheProgramReportsNothing`).
- [x] A choice's key selects it with the drawer open; not with it closed, and
  not while a filter is being typed, when the key goes into the filter
  (`TestAChoicesKeySelectsIt`, `TestASwitchNeedsNoFilter`).
- [x] Every choice's position and the line's size are the same before and
  after each selection (`TestChangingTheSelectionMovesNothing`).
- [x] A switch of one choice or none takes no height
  (`TestASwitchWithNothingToChooseTakesNoRoom`).
- [x] With scrolling on and content taller than the room: the window is the
  size it was; a scroll event over the drawer moves the content and one over
  a card does not; the keys in R10 move it; the offset never passes either
  end (`scroll_test.go`).
- [x] With content that fits, the content is in the same place at the same
  size with scrolling on and off, and has nowhere to scroll to
  (`TestContentThatFitsIsDrawnAsWithoutScrolling`). The criterion as drafted
  compared rendered images; places and sizes are what a reader would see
  move.
- [x] `SetContent` and closing reset the offset; setting a piece in place
  keeps it; content set while the drawer is open scrolls
  (`TestTheOffsetIsKeptThroughAReadingAndNotThroughNewContent`,
  `TestContentSetWhileOpenScrolls`).
- [x] A rule is one line across its room and lower than a row
  (`TestARuleIsALineAcrossItsRoom`); the cards' size and the drawer's reach
  (`TestTheCardsSizeLeavesTheDrawerOut`).
- [x] The example's thermal drawer switches between its two lists and its
  usage drawer scrolls, through twenty polls at one window size
  (`examples/glance-monitor/drawer_test.go`).
- [x] `make lint`, `make test`, `make vuln`.

On a real window:

- [x] Plasma 6 on Wayland, through hayami's panel built on this branch, by
  the person at the desk (Verification).
- [ ] Windows 11 (`FYNEDESYGN_WINDOW_TEST=1`): the same, with posted clicks
  and wheel messages. **Not run**: there is no Windows desk in reach of this
  work. The spec stays INCOMPLETE on this criterion alone.

## Risks & Assumptions

- The wheel's route was read in Fyne 2.8.1's source
  (`processMouseScrolled` matches `fyne.Scrollable` alone) and has not been
  run. A canary test holds it, so a Fyne that routes the wheel as it routes a
  click fails one test. If the catcher ever becomes `Scrollable` the drawer's
  scroller stops hearing the wheel; the catcher's comment says so.
- The scroller is Fyne's `container.Scroll`. Its minimum height is not zero;
  the drawer's frame is sized by `drawerLayout` and its minimum is not
  consulted, which a test holds (`TestADrawersContentDoesNotSetTheHeight`
  with scrolling on).
- The switch's keys share the canvas's typed-key handler with the filter,
  and the filter takes its keys first. A choice whose key is the filter's key
  (`f` by default) is never reached; `SetSwitch` says so. Two choices with
  one ID panic in `SetChoices`.
- `Drawer.Fit` no longer counts a piece that is not drawn among those above
  the list. Before, a hidden piece cost the list a line.
- A scrolled list of live readings moves under the eye when its order
  changes. That is why lists are still cut and filtered and R13 says so.
- The project rule "one scroll per section" holds: the drawer has one, and a
  piece in it has none.
- The Windows real-window criterion cannot be run from the Linux desk.
- Release safety: additive. A drawer with no switch and scrolling off is
  0.1.100's. Rollback is a revert.

## Verification

Plasma 6 on Wayland, scale 1.5, grid arrangement, 2026-10-13, on hayami's
panel built against this branch and used by the person at the desk, with a
real pointer and keyboard:

- The switch on the cooler card's drawer, by click and by key, and the
  usage card's drawer of every account with rules between them, were looked
  at and accepted. The drawer was on the left.
- The wheel over the drawer was not separately reported on. The headless
  test holds its route, and no measurement of it was taken on the window.
- No rectangle was read from KWin during those checks. The window's
  rectangle was read for the position checks of hayami spec 062: 304 by 523
  closed and 553 by 523 open, the same height throughout.

Headless, on Linux: twenty-three breaks of one behaviour each across the
switch, the scroller, the rule and the cards' size, each failing a test,
apart from two. Taking the clamp out of `scrollTo` does not compile, and
Fyne clamps the offset itself; and a click in the drawer reaching the cards
cannot happen where the drawer is not over a card, which
`TestAClickOffTheCardsIsNotReported` already held.

## Alternatives Considered

- Report every click in the drawer through `Panel.OnTapped` and let the
  program hit-test its own pieces: rejected, because each consumer would
  write the hit test and the key handling again.
- A taller window for a tall breakdown: rejected by spec 068 R2 and by the
  glance rule that only a card appearing changes the window's height.
- Paging keys only, with no wheel: rejected; a person at a panel has the
  pointer on it, having just clicked.

## E2E Test Plan

A library has no deployment. The end-to-end check is the consumer's: hayami
spec 062 builds on this and drives its real window on both desks.

### Prod visibility

Not applicable: a library, with nothing in production to observe.
