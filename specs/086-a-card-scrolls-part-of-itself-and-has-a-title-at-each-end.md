# Spec 086: a card scrolls part of itself, and has a title at each end

**Issue**: [#255](https://github.com/ushineko/fynedesygn/issues/255)

## Status: COMPLETE

## Executive Summary

Two additive shapes for a glance card. `glance.NewScroller(n)` is a run of
rows held to `n` rows' height inside a card: the wheel moves it a row a notch,
the bar is in the card's right margin, a plot added after it stays put, and
a click lands on the row under the pointer. `Card.SetTitle` and
`Card.SetTitleEnd` make the heading a reading -- a clock's time at the left
and its date at the right -- held at the widest it has been.
`docs/glance.md` replaces "no scroller anywhere" and "do not scroll a list of
live readings" with the rule consumers already followed. Reviewers should
look first at `Scroller.scrolled` (rows from wheel travel) and the scroller
loop in `Panel.tapped`.

Falsified: the height following the rows (R2), the wheel ignored (R3), a
click reaching a row scrolled out of sight (R6), the notch rounded to rows
(found on the desk, below), the right title drawn at the left (R9), a
narrower title narrowing the card (R10) and a heading-only card losing its
heading in Lines (R10a) each fail a test.

## Context

Two shapes hayami asks for, neither of which a card has.

**A run of rows that scrolls in a fixed height.** hayami's Network card lists
the VPNs under the interfaces (hayami spec 085). With several up, the card
grows a line or two per VPN and pushes every card under it down. hayami wants
the VPNs held to the height three of them take now, the rest reached with the
wheel, and the card's sparkline left where it is, at the bottom, outside the
scrolled part.

`docs/glance.md` forbids it twice: "No scroller anywhere" in *What inverts*,
and "Do not scroll a list of live readings" under *Content that scrolls*.
The second is already not what consumers do: hayami's detail drawers call
`Drawer.SetScrolling(true)` on live rows (Network → Peers, among others), and
nobody is worse off for it. The reason given, that a list's order changes
under the eye, is about a list sorted by a reading; a list in an order the
person chose does not move when its readings do. The rule is replaced, not
broken.

**A heading that is the reading.** hayami wants a clock as a section of its
own (hayami spec 106). A title saying "Clock" over the time is a line spent
telling the reader what they can see. The first try was a card with no
title and one left-aligned row; on the desk the person reading it put it
differently: with no title, the clock *is* the title. Of three styles mocked
on the desk they chose the title's own: bold, the title's grey, the time with
its icon at the left and the date with its icon at the right. A card's
heading has one title at the left and the gone marker at the right; it has no
second title, and its title is set once.

## Requirements

### A scrolling run of rows

- R1 A new piece, `glance.Scroller`, holds rows and shows at most
  N of them at a time. N is the consumer's (`SetVisible(n)`). It goes into a
  card with `Card.Add` like any piece, so what is added after it -- a
  sparkline -- stays under it and does not scroll.
- R2 **Its height is N rows' height whenever it holds more than N rows**, and
  its rows' height when it holds N or fewer. A row added or removed past N
  changes nothing outside it: the card, and the window, keep their height.
- R3 The wheel scrolls it while the pointer is over it, by whole rows, within
  what there is to scroll. Over the rest of the card the wheel does what it
  did.
- R4 The bar is the drawer's (spec 069): in the right margin, no width of its
  own, drawn only while there is something to scroll, not dragged.
- R5 A row set in place (its value changed) does not move the offset. Rows
  added or removed keep the first visible row where it was when it is still
  there, and clamp otherwise.
- R6 A click on a row inside it reaches the program as a click on a card's
  row does today (`Panel.OnTapped`, with the row in `Tap.Row`), at the row
  under the pointer after scrolling.
- R7 In `Lines` (one line per reading) its rows are lines like any other, and
  the cap is N lines.
- R8 It takes the card's theme (it is a `Piece`), and its rows share the
  card's columns, so a value in a scrolled row lines up with one above it.

### A heading that is the reading

- R9 `Card.SetTitleEnd(text, icon)` puts a second title at the heading's
  right end, before the gone marker, in the title's face, with an icon
  before it drawn as `SetIcon`'s is. An empty text and a nil icon take it
  away. `Card.SetTitle(text)` changes the title.
- R10 Both titles are held at the widest they have been, so a narrower one
  -- `11:11:11` after `18:01:18` in a face that is not monospace -- leaves
  the card's width alone. A change of face measures them again.
- R10a In `Lines` a card that has nothing but its heading keeps it; a card
  with rows leaves it out, as before.

### The rules

- R11 (rules) `docs/glance.md` *What inverts*: "No scroller anywhere" becomes "No
  window scroller. The window is the height of its cards; a part of a card or
  a drawer may scroll inside a height that does not change."
- R12 *Content that scrolls* drops "Do not scroll a list of live readings"
  and says instead: scroll a list in an order the person chose or that does
  not follow its readings; a list sorted by a reading is cut with `Fit` and
  filtered, since its order moves under the eye.
- R13 A section documents `Scroller` and a heading that is the reading, with
  a picture of each.

## Acceptance Criteria

- [x] `make test`, `make lint`.
- [x] Tests: a scroller with N+2 rows measures N rows' height, and the card
      measures the same before and after a row is added past N; with N or
      fewer it measures its rows; a wheel event scrolls by a row and clamps at
      both ends; a value change keeps the offset; a removed row clamps; a
      click after scrolling reaches the row under the pointer; the clock's
      icon starts where another card's title does and its date ends where
      that card's figures do, its icon before it; a narrower title and end
      leave the card's width; a heading-only card keeps its heading in
      Lines and one with rows does not.
- [x] Real window: the glance-monitor example with a card holding a scroller
      of five rows, N = 3, over a sparkline. Screenshot before and after a
      wheel turn; geometry recovered from the picture shows the card's height
      and the sparkline's position unchanged and the rows moved by one. The
      clock card in the same window: its time at the titles' left edge and
      its date at the figures' right edge, and its width unchanged across a
      minute of ticks.
- [x] Falsified: each of R2 (height follows rows), R3 (wheel ignored), R6
      (click on the unscrolled row), R9 (end at the left), R10 (width
      follows the text) and R10a (heading left out) broken in turn fails a
      test.
- [x] `docs/glance.md` says R11-R13.

## Risks & Assumptions

- The panel's catcher takes every press (the drawer's bar is not dragged for
  this reason). Wheel events reach the scroller under the pointer as they
  reach the drawer: Fyne finds the Scrollable under the pointer, and the
  catcher is not one. Confirmed on the desk.
- **A notch carries more than a row on a scaled desktop.** Fyne makes a
  notch 25 (Linux, Windows) times the wheel's count, which arrived as 37.5
  at 1.5; rounding that to rows moved two rows a notch. A wheel event now
  moves one row, and only travel past four rows (Fyne's accelerated spin)
  moves more. `TestANotchIsOneRowWhateverItCarries` holds it.
- **A window that appears under a still pointer gets no wheel** until the
  pointer moves: Fyne has not been told where it is (quirk 48's cause). A
  person always moves the mouse to the card; a test steering the pointer
  must move it in, not leave it where the last run's window was.
- Additive: no consumer's card changes unless it adds a scroller or a second
  title. The title now stands in a holder that keeps its widest width; a
  title set once, as every consumer's is, is measured as before. Rollback is reverting the release.

## E2E Test Plan

Done on the desk (Plasma 6.8 Wayland, scale 1.5), with
`examples/glance-monitor -clock -tunnels` built from this branch: the pointer
steered onto the tunnel rows with ydotool, three wheel notches, a capture
after each. The top row went wg0, wg1, wg2, and stayed at wg2 on the third
notch; the window stayed 450x804 px and the plot's trace stayed on rows
218-226 of the card's crop in all four captures (R1-R4). The clock card,
captured every ten seconds for a minute: its ink starts at x = 25 px, where
the Peripherals title's does, and ends at 423, where that card's figures end
at 421 (a bold "1" against a "%"), in every capture; the window stayed
450x809 px (R9, R10).
`docs/img/glance-scroller.png` is two of those captures.

hayami spec 105 is the consumer: its Network card with five VPNs up, run on
the desk, screenshot before and after a wheel turn (covers R1-R6 in a real
program). hayami spec 106's clock card covers R9.
