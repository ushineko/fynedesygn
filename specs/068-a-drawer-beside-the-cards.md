# Spec 068: a drawer beside the cards

**Issue**: [#198](https://github.com/ushineko/fynedesygn/issues/198)

## Status: INCOMPLETE

Built, with its headless tests passing and falsified, and checked on Plasma
6.7.5 on Wayland. The real-window tests on Windows are written and not yet
run against this build; see Acceptance Criteria.

## Executive Summary

A glance panel gains a drawer: a surface in the same window, beside the card
stack and as tall as it, that the program fills with the detail behind a
reading. A primary click on a card, a row or a meter is reported to the
program; the window grows sideways to show the drawer and comes back when it
closes. A list in it can be filtered from the keyboard, and the keyboard goes
back to the window the person was in afterwards.

## Context

A card draws one reading per line and has no room for what is behind it.
hayami wants the processes behind a bandwidth rate, behind the processor's,
the graphics card's and the memory's load, and a provider's usage in full
(hayami #191). `Card.SetTip` is the only place a card says more, and a hover
note cannot hold a list that refreshes or take a key.

Two facts of the package stood in the way and are changed here. A primary
click reached nothing in a glance window: the catcher over the panel is the
last match in the click walk (quirk 24), and on Windows its press started the
move loop (spec 052). And docs/glance.md said the context menu is the whole
interface; a drawer is a second thing a person does to a panel, and the page
now says when it is the right one.

A prototype was driven on real windows on Windows 11 and on Plasma 6.7.5 on
Wayland before this was written. Its measurements are under Verification and
decide the requirements below.

## Requirements

### The drawer

- R1 A panel has one drawer, `Panel.Drawer()`, closed until asked. It is drawn
  beside the card stack in the same window, on a card's surface (fill, border,
  radius, margins), `CardGap` from the cards, with a heading (`SetTitle`) and
  content (`SetContent(pieces ...Piece)`: rows, meters, a filter).
- R2 The drawer is the height of the card stack and never sets it.
  `Drawer.Fit(line, above...)` is how many lines fit under the pieces always
  shown, and `Available` the room itself. No scroller.
- R3 Its width is a factor of the text size (`DrawerWidthInEms`), in the
  panel's own face, and changes with a restyle.
- R4 Opening and closing (`SetOpen`) is a change of shape: the window is
  resized once, in the same call. The cards do not move on the screen, on
  either side.
- R5 A width the user dragged (`Options.Resizable`) is a width of the cards:
  it survives the drawer opening and closing and is not read from the wider
  window.
- R6 The drawer and its content take the panel's theme (`Panel.SetTheme`) and
  are restyled with it. `SetTranslucent` leaves the gap beside it clear, as
  between cards.
- R7 In `Grid` and `Lines` the drawer is beside the whole stack, as in
  `Stack`.

### Which side

- R8 `Drawer.SetSide(Right | Left)` names a side and keeps it. Until then, and
  after `SetAuto`, the side is chosen at each opening: the right if the drawer
  fits between the window and the edge of its screen's work area, the left if
  it fits only there, the side with more room if it fits on neither.
- R9 On the left the window grows leftwards: its right edge stays where it is.
  On Windows the window moves itself (`SetWindowPos`). On Plasma the
  compositor holds the edge, by `kwin.HoldRightEdgeScript`, loaded only while
  a left drawer is in use; the program says so with `Drawer.SetEdgeHeld`.
  **A drawer that cannot grow leftwards opens on the right**, whatever was
  asked: the alternative is cards that jump under the pointer.
- R10 On Windows the drawer reads the room itself. On Plasma
  `kwin.ReportRoomScript` reports the room on each side over D-Bus, once per
  opening, and the program hands it to `Drawer.Choose(left, right)`. A drawer
  nobody told opens on the right.

### The click

- R11 A primary click on a drawn card is reported to the program
  (`Panel.OnTapped`) as a `Tap`: the card, and the row or the meter under the
  pointer, or neither. Every glance window has the catcher that reports it,
  with or without a menu.
- R12 On Windows the move loop starts on the first movement of a press, not on
  the press, so a press that does not move is a click. A drag moves and snaps
  the window as before (specs 052, 058).
- R13 A press that moves further than the toolkit's drag threshold before its
  release is not a click, on any desktop: the catcher is `Draggable`
  everywhere (quirk 50).
- R14 After the compositor moves the window under a pointer that has not
  moved, the next click is read at the pointer's true place (quirk 48). The
  panel knows how far its content moved and tells Fyne, by calling Fyne's own
  GLFW cursor callback with the position GLFW last heard moved by that much
  (`repoint`), and again whenever the window's focus changes, because Fyne
  forgets its pointer position when the focus leaves. Where there is no GLFW
  window to reach, a click at exactly the position of the last one is read
  that much further along.
- R15 The library does not decide what a click means. Opening on a click,
  changing what is shown on a click elsewhere and closing on a second click of
  the same thing are the program's, and the example does them.

### The filter

- R16 `glance.Filter` is a piece for a drawer: one line, always present, so
  starting or clearing a filter moves no row. It shows a hint when empty, the
  text and a cursor while it is typed, and the filter in force afterwards. A
  long filter is drawn by its end.
- R17 `Drawer.SetFilter` points the window's keys at it. With the drawer open
  and the panel holding the keyboard, the filter's key (`f`, `Filter.Key`)
  starts it; typing narrows as it goes; Backspace takes a character; Return
  keeps the filter; Escape clears it. Closing the drawer ends the typing and
  keeps the text.
- R18 The text is a regular expression matched without regard to case. Text
  that does not compile is matched as written and drawn in the warning
  colour, so a half-typed expression never empties the list.
- R19 `Filter.Matches(string) bool` is what the program asks, and
  `Filter.OnChanged` tells it to draw again; the library does not hold the
  list.

### The keyboard

- R20 When the drawer closes, and when a filter ends by Return or Escape, the
  keyboard goes back to the window the person was in (`Drawer.YieldFocus`). It
  acts only when the panel holds the keyboard, and never brings a window
  forward otherwise.
- R21 On Windows the window handed to is the one that was in the foreground
  when the click arrived, recorded on `WM_MOUSEACTIVATE`; the first eligible
  window under the panel in the Z-order where none was recorded.
- R22 Elsewhere the program gives its own way (`Drawer.SetYield`). On Plasma
  that is `kwin.YieldFocusScript`: if `workspace.activeWindow` is the panel,
  the first window down `workspace.stackingOrder` that is a normal window, not
  minimised, takes input and is on the current desktop
  (`w.onAllDesktops || w.desktops.indexOf(workspace.currentDesktop) >= 0`;
  Plasma 6 has no `onCurrentDesktop`). With no way given the panel keeps the
  keyboard, as it does today.
- R23 A program whose rule sets `NoFocus` cannot have a filter: the panel is
  never given the keyboard. `Drawer.SetFilter` and docs/glance.md say so.

### Documentation and the gallery

- R24 docs/glance.md gains "A drawer beside the cards", and its Windows note
  and its inversion table are corrected. docs/fyne-quirks.md gains 48 (a
  click after a compositor move), 49 (a window hidden and shown) and 50 (a
  press that moves is a tap). That KWin's `loadScript` does nothing for a
  relative path is KWin's and not Fyne's, and is recorded in docs/glance.md
  and the `kwin` package comment.
- R25 `examples/glance-monitor` has a drawer of generated processes with a
  filter; the gallery shows a panel with its drawer out on each side and a
  filter in each state. README: the package and example rows and the
  changelog.

## Acceptance Criteria

Headless, `go test ./...` on Windows 11 with Go 1.27, each falsified
(thirty-one breaks, each failing the test named for it):

- [x] Opening grows the window by the drawer's width and the gap, and closing
  restores it; the height does not change, whatever the drawer holds
  (`TestADrawerWidensTheWindowAndClosingPutsItBack`,
  `TestADrawersContentDoesNotSetTheHeight`).
- [x] The drawer's height is the card stack's with one card and with two
  (`TestADrawerIsTheHeightOfTheCards`), and `Fit` is the lines that go in it
  (`TestFitIsTheLinesThatGoInTheRoom`).
- [x] The cards stand at the edge the drawer is not on
  (`TestTheCardsStandWhereTheDrawerIsNot`).
- [x] A dragged width survives an open and a close
  (`TestADraggedWidthSurvivesTheDrawer`).
- [x] The side: by room, a named side kept, and the right when the window
  cannot grow leftwards (`TestADrawerChoosesTheSideWithRoom`,
  `TestANamedSideIsKept`,
  `TestADrawerThatCannotGrowLeftwardsOpensOnTheRight`).
- [x] The drawer's width and its content follow the panel's face
  (`TestADrawerIsWiderInALargerFace`,
  `TestADrawersContentTakesThePanelsTheme`).
- [x] A click is reported with its card, row or meter; not for a hidden card
  or the drawer; with no menu (`TestAClickIsReportedWithWhatItLandedOn`,
  `TestAClickOffTheCardsIsNotReported`,
  `TestAClickNeedsNoMenuAndNoListener`).
- [x] A second click without the pointer moving is on the same row after the
  window moved, and a click elsewhere is read where it is
  (`TestASecondClickWithoutMovingThePointerIsOnTheSameRow`,
  `TestAClickElsewhereIsReadWhereItIs`).
- [x] The filter: its key, typing, an expression without case, a half-typed
  one, Backspace, Return, Escape, another key, one line of one height, a long
  one by its end, keys only while open (`filter_test.go`).
- [x] Closing the drawer and ending a filter give the keyboard back, by the
  program's way where the window has none
  (`TestClosingTheDrawerGivesTheKeyboardBack`,
  `TestEndingAFilterGivesTheKeyboardBack`).
- [x] The three scripts say what they must and cannot be escaped by their
  strings (`glance/kwin/drawer_test.go`).
- [x] The example: a click opens, switches and closes; the list is what fits;
  the filter narrows it; the window is one size through twenty polls with the
  drawer open (`examples/glance-monitor/drawer_test.go`).

On a real window, written and to be run (`FYNEDESYGN_WINDOW_TEST=1`; they
take the pointer):

- [ ] Windows: a real click opens the drawer and a second closes it, on a
  named right, a named left, by room mid-screen and by room near the right
  edge; the edge the drawer is not on is the same pixel
  (`TestADrawerOpensBesideTheCardsOnWindows`).
- [ ] Windows: a drag still moves the window and is not a click, and a click
  after it is (`TestADragMovesTheWindowAndIsNotAClick`); spec 058's two
  real-window tests pass unchanged.
- [ ] Windows: the foreground window is the one before the click after the
  drawer closes, after Return and after Escape
  (`TestTheKeyboardGoesBackToTheWindowThatHadIt`).
- [ ] Windows: a filter typed does not resize the window
  (`TestAFilterTypedDoesNotResizeTheWindow`).
On Plasma 6.7.5 on Wayland, with the three scripts as the library returns
them, run by a scratch program:

- [x] A left drawer keeps its right edge, open and closed; without the edge
  held it opens on the right and the cards do not jump.
- [x] A press that moves 3 px or more is not a click; one that moves 0, 1 or
  2 px is (R13).
- [x] The side follows the room: right mid-screen, left 40 px from the right
  edge of the placement area.
- [x] The keyboard goes back after the closing click, after Return and after
  Escape.
- [x] A filter typed with real keys; the window's size does not change.
- [x] Translucent with the rule: the drawer is the cards' height and the gap
  beside it is clear, on both sides.
- [x] `make test` (race on) and `make lint`.
- [x] R14: a click in place is read on the row under the pointer after the
  window moved, after the pointer moved while the drawer was open, and after
  the keyboard was handed back, mid-screen and at the right edge. Two faults
  found on the way are under Verification.
- [x] Closing and reopening at the right edge with the side chosen by room.

## Risks & Assumptions

- The drawer wraps the card stack in one more container.
  `TestANewOrderIsTheGridsOrder` indexed into the panel's tree and now reaches
  the stack through the body.
- Changing the Windows press from "move now" to "move on the first movement"
  touches the path spec 052 and spec 058 depend on. The grab point is a few
  pixels from the press; the snap code takes its own from the move loop, so
  snapping is unaffected. Confirmed by hand on the prototype; the real-window
  tests are the guard.
- **Fyne's drag threshold is 2 units.** A click during which the pointer
  moves two pixels is a drag: on Windows the window moves by that much and
  the drawer does not open. Not seen by hand; a hand that is unsteady on the
  button would see it.
- R14 makes up the motion event the compositor did not send. It reaches
  Fyne's GLFW window by reflection, as the tips' leave watcher does (quirk
  42), and assumes a GLFW coordinate is a Fyne one times the canvas's scale,
  which is how Fyne reads one. A pointer that GLFW reports at the same place
  twice is taken not to have moved; one moved away and brought back to the
  same coordinate between two openings would be put wrong by the drawer's
  width until it next moves. It applies only where the compositor moves the
  window, and only for a drawer on the left.
- Every glance window now has the catcher, so on Linux a window with no menu
  takes primary clicks it used to let through. Nothing under the catcher was
  ever clickable where there was a menu, and the package draws no controls.
- `Drawer.SetFilter` takes the canvas's typed-key and typed-rune handlers. A
  program that set its own on a glance window loses them.
- The stacking order is not the focus history. The window under the panel is
  wrong when the window the person was in was not the topmost normal one.
  Windows records the true one (R21); Plasma would need a script kept loaded,
  which this spec does not add.
- Setting `workspace.activeWindow` also raises that window. It was already the
  top normal one when measured.
- The compositor's three parts work on KWin only. On another Wayland desktop
  the drawer opens on the right and the panel keeps the keyboard.
- Flicker on opening was not recorded on either desktop. One geometry event
  per toggle on Plasma suggests none; a left drawer on Windows moves and then
  resizes, which are two steps.
- The Windows files add findings of the kind the package's other Windows
  files already have under the linter run on Windows (`unsafe`, integer
  conversions at the system-call boundary). CI lints as Linux, where they are
  not built.
- Rollback: revert. A panel whose program never opens the drawer and sets no
  `OnTapped` draws as before. Two behaviours change for every glance window:
  on Windows the move starts on the first movement of a press, and a press
  that moves is no longer a tap.

## Alternatives Considered

- **A second window for the drawer.** It would have to be placed beside the
  first and kept there through a drag, which on Wayland only the compositor
  can do, and two windows share one app ID. One window moves as one and lines
  up by construction.
- **Opening on hover.** A panel is crossed by the pointer all day; it would
  open unasked.
- **Opening from the context menu only.** It cannot say which row was meant.
- **An entry widget for the filter.** The catcher takes every click, so an
  entry could not be clicked, and a focused widget changes where keys go for
  the whole canvas. A drawn line and the canvas's own key handlers do it.
- **Not taking the keyboard on a click.** Nothing a Wayland client can reach
  does that; layer-shell does, and is another kind of window that cannot be
  Meta-dragged.
- **Hide and show to give the keyboard back.** Measured on Plasma: the panel
  retakes it 3 ms later and is placed somewhere else (quirk 49).
- **A left drawer that lets the cards jump** where nothing holds the edge. It
  opens on the right instead (R9).
- **Reading the pointer afresh at each click** for R14. GLFW's position on
  Wayland is the last motion event's too, so there is nothing fresher to read.
- **Correcting the click when it arrives**, which was the first build. It
  helps only while the place Fyne believes the pointer to be is inside the
  window. After a left drawer closes it can be past the narrower window's
  edge, where Fyne finds nothing and no click arrives to correct.

## Gaps found

- The package has no call that runs a KWin script; a program makes the D-Bus
  calls itself (spec 037). Three more scripts make a helper that loads, runs
  and unloads one worth having; it would make `godbus` a direct dependency of
  a library package, which is a decision for its own spec. Until then the
  example cannot run the scripts, and takes `-drawer-edge-held` for a tester
  who has loaded the hold script by hand.
- `-kwin-install` in the example writes the rule and does not ask KWin to
  re-read it; the window stayed decorated until `reconfigure` was called.
- `Row` has no getter for its label; the example keeps the names it drew for
  its own test.
- **A window mapped under a pointer that has not moved reads its first click
  at 0,0.** Fyne has no pointer position until a motion event, asks GLFW at
  the click, and GLFW has none either. Confirmed on Plasma with no drawer and
  no script: a click over the CPU row of a fresh panel was read as the
  top-left card, twice, and correctly after a 3 px movement. It is Fyne's and
  older than this spec; a panel that reports clicks now shows it. Not fixed
  here.
- A terminal panel has no drawer. What a consumer's terminal shows instead is
  the consumer's.
- No gallery screenshot was retaken: the new demonstrations are below the
  first screen of the Glance section, which is what the README shows.
  `docs/img/glance-drawer.png` is the example on Windows 11.

## Verification

### Before the spec: the prototype, 2026-10-12

Windows 11, real window, clicks and keys posted as window messages:

- 300x376 to 599x376 and back to the same rectangle on the right, origin
  fixed; on the left the left edge moves by the drawer's width and the right
  edge stays at the same pixel.
- 40 px from the right edge of a 2560 px work area the drawer opens on the
  left; in the middle, on the right.
- 17 rows fit 358 units of height; 16 with the filter's line.
- `ch` leaves two names; `^(k|z)|fox$` leaves six; the window stays 599x376.
- After Return, Escape and the closing click the foreground window is the one
  before; a panel without the keyboard changes nothing.
- By hand: drag, edge snap and click work.

Plasma 6.7.5, Wayland, scale 1.5, clicks and keys injected:

- A click opens and closes; one geometry event each way; Meta-drag moves the
  window. Top-left fixed on the right.
- On the left, unaided, the cards jump right by the drawer's width; with the
  hold script the resize and the move are logged in the same millisecond and
  closing returns to the origin.
- After that move three clicks on the same card did nothing until the pointer
  moved a pixel or two.
- A press, 36 px of movement and a release opened the drawer.
- Geometry and screen area by script: 1.1 to 2.0 ms per round trip, 30 runs.
- Keys arrive once a click has activated the panel; with `acceptfocus=false`
  the panel is never active.
- Hidden and shown, the panel retakes the focus 3 ms later and is placed
  afresh; `RequestFocus` does nothing.
- Giving the keyboard back by script: 1.3 to 1.5 ms; focus-stealing
  prevention at its default did not interfere.
- Translucent with the rule: the drawer is the window's height and the gap is
  clear.

### The build, 2026-10-12

Windows 11, Go 1.27:

- `go test ./...`: every package passes.
- Thirty-one breaks, one behaviour each, and the test named for each fails;
  two tests were tightened because a break passed them (what the content says
  it needs, which is what Fyne fits a window to).
- The pinned linter (v2.12.2, `GOTOOLCHAIN=go1.26.0`) was run here and is not
  evidence: on Windows it reads the Windows files, and asked to read as Linux
  it could not load the packages that need cgo. It is to be run on Linux.
- The example on a real window, clicks posted as messages: by room mid-screen
  1130..1430 to 1130..1729 and back; by room 40 px from the right edge
  2220..2520 to 1921..2520 and back; a named left 1130..1430 to 831..1430 and
  back. A filter typed and cleared leaves the window 599x376.

Plasma 6.7.5 on Wayland, the first commit, by the Linux desk's session, with
clicks and keys injected and the library's scripts run by a scratch program:

- Left, edge held: 1130, 300 wide, to 830.67, 599.33 wide, and back; two
  geometry events each time, the resize and then the move.
- A second click without moving the pointer closes, a third reopens; after a
  move, the CPU row and the Coolant row are each read as themselves.
- **A lost click.** Open, move the pointer, click, click in place (closes),
  click in place: nothing reached `OnTapped`. The place Fyne believed the
  pointer to be was about x=449 on a window now 300 wide. This is what the
  second commit (`repoint`) is for.
- Left without the edge held: opens on the right, x unchanged.
- A press, 36 px and a release: no click. 0 px: a click; 1 px, three of
  three; 2 px, two of two; 3, 4 and 6 px: none.
- Room: mid-screen 1130 and 1130, right, in 1.5 and 1.9 ms; 40 px from the
  right edge 2220 and 40, left, in 3.0 ms with the hold script loaded, and the
  window opened with its right edge at 2520.
- The keyboard goes back after Return, Escape and the closing click; the
  script took 1.5 to 2.7 ms.
- Filter by real keys: `z`, `zs`, Backspace, Return, then Escape; the window
  stays 599.33x376 with no geometry event.
- Translucent with the rule: the drawer is opaque over the full 376, with 13
  to 14 px clear beside it, on both sides.
- The run stopped part-way through the room check because the desk was in
  use.

Linux (CachyOS), both commits, in a scratch clone: `make lint`, 0 issues;
`make test`, with the race detector, every package passes.

Plasma 6.7.5 on Wayland, scale 1.5, the later commits, by the same session:

- With `repoint`: open, move, click, close in place, click in place now
  reopens on the row. Sixteen clicks 8 px inside the cards' left and right
  edges, each after a move of the window, landed on the card clicked, so a
  GLFW coordinate is a Fyne one times the scale at 1.5.
- **A second lost click**, with the keyboard handed back: after a left drawer
  closed and `YieldFocus` ran, a click in place activated the panel and
  reached nothing until the pointer moved a pixel. Fyne sets its pointer
  position to nothing when the window loses the focus
  (`processFocused`) and asks GLFW at the next click, which has the position
  from before the move. The last commit says the position again on a focus
  change.
- With that: after a hand-back by the closing click, by Return and by Escape,
  the next click in place is read on the first try, mid-screen and at the
  right edge; a right drawer with the hand-back, and a left one without it,
  are as before.
- At the right edge with the side by room: 1100 and 40, opens left with the
  right edge at 3960, closes and reopens in place; back mid-screen 570 and
  570, opens right.
- Not checked: what Fyne takes to be hovered after a move. The example has
  no tips to show it.

Linux (CachyOS), all four commits, in a scratch clone: `make lint`, 0 issues;
`make test`, with the race detector, every package passes.
