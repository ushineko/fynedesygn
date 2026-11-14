# Spec 079: several drawers beside the cards

**Issue**: [#233](https://github.com/ushineko/fynedesygn/issues/233)

## Status: COMPLETE

## Executive Summary

A panel can have several drawers open in a row beside its cards.
`Panel.AddDrawer` makes them, `Fits`/`FitsHere`/`DrawersWithoutRoom` cap the
row by the room on the screen, and `Tap.Shift` and `Switch.OnShifted` report
the Shift-click a program opens one with. The side moved from the drawer to
the panel. A reviewer should look first at `drawerLayout.Layout` and
`Drawer.SetOpen` in `glance/drawer.go`.

## Context

A panel has one drawer (spec 068). hayami wants several open at once: the
processor's, the graphics card's and the memory's lists side by side, or the
bandwidth by process beside the cooler's processes, each read while the
others are still in view (hayami spec 088). The person asks for another with
a Shift-click; a plain click goes on meaning what it means now.

Four things in the package assume one drawer:

- `Panel` holds `drawer *Drawer`, and `drawerLayout`, `Size`, `resize` and
  `reshape` add one drawer's width.
- The side, the held edge and the auto choice are fields of the drawer, so
  two drawers could stand on different sides of the cards.
- `Drawer.wireKeys` sets the canvas's typed-key and typed-rune handlers. A
  second drawer's call replaces the first's.
- `Tap` says what was clicked and not how. `menuCatcher.Tapped` gets a
  `fyne.PointEvent`, which carries no modifier keys.

The fork of Fyne the package pins has `desktop.Driver.CurrentKeyModifiers`,
which the GLFW driver answers from the keyboard state it already keeps. Read
at the tap, it says whether Shift is held.

## Requirements

### The drawers

- R1 `Panel.AddDrawer()` returns a new drawer, closed and empty, with the
  panel's theme. `Panel.Drawer()` is still the first, there from the start;
  a program that never calls `AddDrawer` has spec 068's panel, unchanged.
  `Panel.Drawers()` lists every drawer made, the first first.
- R2 The open drawers stand in a row beside the cards in the order they were
  opened: the first opened next to the cards, each later one further out.
  A drawer that closes leaves the row and those beyond it move in towards
  the cards. The cards do not move on the screen, on either side (spec 068).
- R3 Each drawer is a card's surface `DrawerWidthInEms` wide, `CardGap` from
  its neighbour, as tall as the cards and never setting the window's height.
  The window is the cards' width plus each open drawer's `Reach`.
- R4 Opening or closing any drawer is one change of shape: the window is
  resized once, in the same call, growing leftwards for a left side as spec
  068 does for one drawer.
- R5 **The side is the panel's, not a drawer's.** `SetSide`, `SetAuto`,
  `Choose` and `SetEdgeHeld` on any drawer set it for all; `Side()` and
  `Auto()` on any drawer read it. The side is settled when a drawer opens
  with none open (spec 068's choice by room) and kept while any is open.

### The room, and the cap

- R6 **A drawer never makes the window run off its screen's work area.**
  `Panel.Fits(left, right float32) bool` says whether one more drawer fits on
  the side the open drawers are on, given the room between the window and
  each edge of the work area. With none open it is true: the first drawer
  opens as spec 068 says, whatever the room.
- R7 `Panel.FitsHere() (fits, known bool)` reads the room itself where the
  window can (Windows, `roomBeside`) and answers `Fits` from it. Where it
  cannot, `known` is false and the program asks the compositor (on Plasma,
  `kwin.ReportRoomScript`) and calls `Fits`.
- R8 Where nothing can tell the room, a program caps the row at
  `DrawersWithoutRoom` (2). The constant is the package's so every program
  caps alike.
- R9 The library does not decide what happens at the cap. `Panel.Outermost()`
  is the open drawer furthest from the cards, or nil, for a program that
  shows the new detail there instead.

### The click

- R10 `Tap.Shift` is true when Shift was held at the primary click. The
  library reports it and decides nothing.
- R11 `Switch.OnShifted(id)` is called for a primary click on a choice with
  Shift held, on any choice including the selected one, and `OnChanged` is
  not: the selection does not change. With `OnShifted` nil a Shift-click on
  a choice is a plain click.
- R12 A click in a drawer is that drawer's (its buttons, its switch, its
  rows), as spec 068 says for one. A click that lands in no drawer is a
  card's and goes to `Panel.OnTapped`.

### The keyboard and the pointer

- R13 **The keys go to one drawer**: the one opened or clicked last, while it
  is open. When it closes they go to the one opened last of those still
  open. The filter, switch and scroll keys of spec 068 and 069 work on that
  drawer alone. The panel holds the canvas's key handlers once; a drawer's
  `SetFilter` and `SetSwitch` no longer set them.
- R14 The hover of spec 063 is per drawer: the pointer over a drawer's row
  lights that row, and leaving a drawer clears its own.
- R15 Closing the last open drawer gives the keyboard back (spec 068
  `YieldFocus`); closing one of several does not.

### Documentation and the example

- R16 docs/glance.md, "A drawer beside the cards": several drawers, the row,
  the panel's side, the room and the cap, Shift, where the keys go.
- R17 `examples/glance-monitor`: a Shift-click on a card opens its drawer
  beside the ones already open; a Shift-click on a card whose drawer is open
  closes that one; a plain click with several open closes them all.
- R18 README: the changelog entry.

## Acceptance Criteria

Headless, each falsified:

- [x] A panel with no `AddDrawer` call lays out, sizes and reports taps
  exactly as before: the existing drawer tests pass unchanged.
- [x] Two drawers open on the right: the window is the cards' width plus two
  `Reach`; the first opened is next to the cards, the second beyond it; the
  cards' place in the window and their width are unchanged
  (`TestTwoDrawersStandInARowOnTheRight`).
- [x] The same on the left with the window moving itself: the cards stand
  one `Reach` further from the window's edge at each opening, and the drawer
  opened last is at the edge (`TestTwoDrawersStandInARowOnTheLeft`). The
  window's move is `moveWindowBy`, which the test driver cannot observe; the
  criterion as drafted named it.
- [x] Closing the inner of two drawers moves the outer next to the cards and
  takes one `Reach` off the window
  (`TestClosingTheInnerDrawerMovesTheOuterIn`).
- [x] `SetSide(Left)` on the second drawer puts both on the left
  (`TestTheSideIsThePanels`); a row that is out keeps its side when the next
  drawer is placed by the room (`TestARowThatIsOutKeepsItsSide`).
- [x] `Fits`: true with none open whatever the room; with one open, true when
  the room on its side is at least one `Reach` and false below
  (`TestFitsIsTheRoomOnTheRowsSide`).
- [x] `Outermost` is the last opened, and after it closes the one before
  (`TestOutermostIsTheLastOpened`).
- [x] A tap with Shift held reports `Tap.Shift`; without, not
  (`TestAClickSaysWhetherShiftWasHeld`).
- [x] A Shift-click on a switch choice calls `OnShifted` with its ID and not
  `OnChanged`, and leaves the selection as it was
  (`TestAShiftClickOnAChoiceIsReportedAndSelectsNothing`).
- [x] A click in the second drawer reaches its switch and not the first's,
  and not `OnTapped` (`TestAClickInTheSecondDrawerIsItsOwn`).
- [x] Keys: with two drawers open, a typed filter key starts the filter of
  the one opened last; after a click in the first, the first's. Closing the
  keyed drawer hands the keys to the other (`TestTheKeysGoToOneDrawer`).
- [x] Closing one of two drawers does not call the yield; closing the last
  does (`TestOnlyTheLastDrawerClosingGivesTheKeyboardBack`).
- [x] The example: Shift-click opens a second drawer, at the cap the
  outermost takes the card, Shift-click on an open card closes its drawer, a
  plain click with two open closes both (`TestAShiftClickOpensADrawerBeside`).

Each was falsified: nine mutations of the library (row order reversed, keys
not handed on, a click not taking the keys, `Fits` always true, every close
yielding, a row's side not kept, `Tap.Shift` dropped, `OnShifted` never
called, clicks reaching only the first drawer) and two of the example (no
cap, no close-all) each failed a test.

On a real window (docs rule: a claim about what the window looks like is made
against a screenshot), Plasma 6 on Wayland, two monitors at 1.5:

- [x] `examples/glance-monitor`, clicked through ydotool with Shift held by
  key events, its window rule installed: a click opens one drawer (+395.3
  units), a Shift-click a second (+2 × 395.3), a third Shift-click at the cap
  leaves the width, and a plain click closes both. **Shift reaches the panel
  on Wayland.**
- [x] The cards' position on the screen, recovered from the window's
  geometry and the cards' place in each grab: unchanged in the window
  (offset 0 in every picture). On the screen KWin moved the window left by
  half of each widening, with or without the rule, and does the same to the
  example built from `main` with one drawer (200 → 2): it is how KWin
  places this example's window, not this change. hayami, which places its
  window through KWin scripts, is where the screen position is checked
  (hayami spec 088).

## Risks & Assumptions

- **Shift on Wayland.** The modifier state reaches a client with the
  keyboard focus. The click that opens a drawer activates the panel; a
  panel whose rule refuses focus (`kwin.Rule.NoFocus`) gets no modifiers and
  every Shift-click is a plain click. Verified on Plasma before release.
- **Room reported in steps.** `roomBeside` and the KWin script measure the
  room at the moment of asking; a window dragged after the answer may run
  off by a drawer until the next opening. The same holds for spec 068's
  choice of side and is accepted there.
- **Rollback**: revert the release; hayami pins the version and does not
  move to it until its own spec is ready.

## Threat model delta

No boundary touched: a layout change and a modifier key read.

## Found after release

On hayami's panel (Plasma, drawers on the left), a plain click with three
drawers out closed them one `SetOpen(false)` at a time: three resizes in one
moment. KWin answered with the widths in between, `Panel.resize` read the
answer as a width the person had dragged, and the window stayed three drawers
wide with the cards stretched across it until the program restarted. R4 says
a change of shape is one resize; closing a row was three.

- [x] `Panel.CloseDrawers` closes every drawer with one resize and one yield
  (`TestCloseDrawersIsOneChangeOfShape`, falsified by closing each with
  `SetOpen(false)`: three resizes). The example and the docs use it.

With that fixed, the window came back to the cards' width but, in two of
three closes of several drawers, two drawers' width to the left. KWin's log
(a script printing every geometry change of the window) showed the close
held correctly, then 31 ms later KWin restored the old x with the new width:
its answer to the change of focus the close hands back, crossing the resize.
The hold script acted only on a change of width and passed it by.

- [x] `kwin.HoldRightEdgeScript` remembers the edge it held and, for 250 ms,
  puts a change that keeps the width but not that edge back on it; a drag
  (`w.move`) forgets the edge (`TestTheHoldScriptUndoesAPutBackOfThePlace`).
  The logged sequence replayed through the script with a fake window: the
  edge held through the put-back, a full put-back and a drag left alone.
- [x] On hayami's panel, Plasma 6.8 on Wayland, drawers on the left: two
  drawers then a plain click, five times, and three at once, once: the right
  edge at 4000.3 each time (it had moved in two of three before). The cards'
  place on the screen, from the grabs, unchanged through one, two and three
  drawers, the cap, a close of all and a close of one.

### The right side (#236)

Checking the row on the right on hayami's panel showed the cards moving left
by half a drawer at each opening. The glance-monitor example built from
`main` did the same with one drawer: spec 071 had held the right edge for a
left drawer on Plasma 6.8 and nothing held the left edge for a right one.

- [x] `kwin.HoldLeftEdgeScript` keeps the left edge, from the same template
  as the right edge's (`holdEdgeScript`), put-back guard included
  (`TestTheLeftHoldScriptKeepsTheLeftEdge`,
  `TestTheHoldScriptUndoesAPutBackOfThePlace` for both). Replayed against
  Plasma 6.8's centring with a fake window: a right drawer's window keeps its
  x through two openings, a close, a put-back; a drag and a later move are
  left alone; the right edge's script behaves as before.
