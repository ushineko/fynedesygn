# 072 — a drawer that is worked and not only read

**Issue**: #215

## Status: INCOMPLETE

Partial. Everything below is built and used at the Linux desk (Plasma 6,
Wayland, 1.5× scale) through hayami's bandwidth spike and then its cooler, which
is what found each of these. Nothing here has tests yet, and nothing has been run on
Windows or macOS. The spec is written down now so the design decisions and
the reasons behind them are not carried only in a working tree.

## Executive Summary

A drawer was a page to read: its only control was the switch, nothing else
answered to a pointer, and the content was cut to the height rather than
scrolled. A drawer that a person navigates — stopping a reading, walking
into a row, coming back out — needs more than that. This adds a row of
buttons to the heading, a lead button for the way back, clicks and hover on
rows, and a scroll offset that survives a rebuild. Reviewers should look
first at `Drawer.SetOffset` and `SetRowTapped`, which are what the two
existing paths lacked, and at `buttonTop`/`buttonBottom`, which are uneven
on purpose.

## Context

hayami spec 060 gave the drawer a list of processes; spec 069 gave it a
switch and a scroller; spec 062 opened it from anywhere on a card. All three
treat the drawer as a page: the program writes it, the person reads it.

A breakdown that a person walks into is a different thing. Driving one
through the bandwidth spike found, in order, that the drawer could not
report a click on a row, could not keep its place when its content changed,
had no hover at all, had no control that looked like a control, and was too
narrow for a column of hostnames. Each of those is the library's, not the
program's, which is why they are here and not in hayami's spec.

## Requirements

### The scroller keeps its place

- R1 `Drawer.SetOffset(y float32)` puts the content back to where it was
  scrolled to. `Offset()` could already be read; nothing could write it.
- R2 The offset is clamped to what there is to scroll, so a rebuild that
  made the content shorter settles at its end and not past it. `scrollTo`
  already does this; `SetOffset` is its public form.
- R3 A caller rebuilding content reads `Offset()` before and gives it back
  after. The library does not do this for them: it cannot know whether the
  new content is the same list or another one.

### A click on a row

- R4 `Drawer.SetRowTapped(on func(*Row), rows ...*Row)` names the rows that
  answer to a primary click and says which was hit. Passing no rows, or a
  nil function, makes the drawer a page to read again.
- R5 A click in the drawer that is not on the switch, a button or a named
  row does nothing and is not reported, as before.
- R6 The row carries what it means: `Row.SetID`/`ID` already exist and the
  caller puts its own meaning there. The library does not interpret it.

### Hover

- R7 `Drawer.SetHighlight(on bool)` turns hover on. It is off unless asked
  for: a panel is glanced at, and a list that lit up as a pointer crossed
  it on the way elsewhere would twitch.
- R8 `Row.SetHovered(on bool)` lights the row's ground with
  `theme.ColorNameHover`. Subtle on purpose — it says which row the pointer
  is on and must not read as a selection, because nothing has been selected
  by pointing at it.
- R9 Every `Row` gets its ground at construction, not only the rows that can
  be hovered, so a row that lights and one that does not are the same height
  and the list does not shift when highlighting is turned on.
- R10 The pointer is heard on the catcher that already lies over the panel
  for clicks (`menuCatcher`, now `desktop.Hoverable`). Only rows with a
  non-empty ID light: a row with nothing to ask does not pretend otherwise.

### Buttons in the heading

- R11 `Drawer.SetButtons(specs ...DrawerSpec)` draws a row of controls at
  the right of the heading, in the order given, the last right-most.
  Buttons are built as the number grows and hidden as it shrinks.
- R12 `DrawerSpec` carries `Label`, `On`, `Words` and `Active`.
- R13 `Words` is every label that button will ever show. The button is held
  at the width of the widest, so a control that changes its word does not
  change its shape: one that grows and shrinks moves out from under the
  pointer about to press it again.
- R14 The row is right-aligned, so a contextual button appearing on the
  left pushes nothing that was already there.
- R15 `Active` marks an engaged toggle: filled with
  `ColorNamePrimary` and bold, against `ColorNameHover` and regular for
  idle. It is the fill that changes and not a shade of it, because which
  state a toggle is in must be seen and not only read.
- R16 `Drawer.SetLeadButton(*DrawerSpec)` puts one button at the left of the
  heading, before its words, and nil takes it away. Where a back belongs:
  leaving a page is not one more thing the page offers, it is the way out,
  and the way out is in the same corner every time.
- R17 A button is a pill: its corner radius is half its height.
- R23 The heading is measured again when its buttons change. A button is
  built hidden and shown when its spec arrives, and the row that holds it
  is laid out in between; a container measures what is visible, so without
  asking again the row keeps the width it had while its buttons were
  hidden, and they are there, correct and invisible.
- R18 The room above and below the label is uneven — `pad*0.8` above,
  `pad*2.2` below. A text's box runs ascender to descender and a word like
  "Pause" or "Back" has nothing below the baseline, so centring the box
  leaves the ink high and the button reads as bottom-heavy. The ink is
  centred instead.

### Width

### The face

- R20 Every text the drawer draws is set in the theme's face, the buttons'
  labels included. A text takes its face only when it is told to
  (`themed.refit`), so a control gets it at build and again in
  `Drawer.Restyle`.
- R21 One place paints a button (`drawerButton.restyle`), and both a label
  change and a theme change go through it, so the two cannot leave a button
  looking like two different buttons.
- R22 A button's pill radius is recomputed when the face changes: a larger
  face is a taller button, and half of the old height is no longer half of
  it.

### Width

- R19 `DrawerWidthInEms` is 32, was 24. A drawer holding a name column and
  two rates has no room for a hostname at 24: `← 192.168.86.212` collided
  with its own figures.

## Acceptance Criteria

- [ ] `SetOffset` puts a rebuilt drawer back where it was, clamped.
- [ ] `SetRowTapped` reports the row under a click, and only for rows given.
- [ ] A click on the switch or a button is not also reported as a row.
- [ ] `SetHighlight(false)` lights nothing; `(true)` lights the row under
      the pointer and only one at a time.
- [ ] A row with an empty ID never lights.
- [ ] Row heights are the same with highlighting on and off.
- [ ] `SetButtons` with two specs draws two, and with one hides the second.
- [ ] A button given `Words` does not change width when its `Label` changes
      to another of them.
- [ ] `Active` draws the filled form and false the outlined one.
- [ ] `SetLeadButton(nil)` removes the lead button.
- [ ] A button shown after its row was laid out is visible: the heading is
      measured again (R23).
- [ ] A button's label is centred on its ink, not on its text box.
- [x] Every text in the drawer is set in the theme's face after a theme
      change, the buttons' labels included
      (`TestEveryTextInTheDrawerIsSetInTheThemesFace`).
- [x] A button added after a theme change takes it
      (`TestAButtonAddedAfterTheThemeTakesIt`).
- [x] A button grows with the face (`TestAButtonGrowsWithTheFace`).
- [ ] Every test above runs on Linux; the Windows real-window tests compile
      and are run before release.

## Out of scope

- Keyboard navigation of rows. The switch and filter already take keys; a
  drill that is walked by keyboard is a later spec.
- A disabled button state. Nothing has needed one: a control that does not
  apply is not drawn.
- Hover on cards. This is the drawer's.

## Risks & Assumptions

- `DrawerWidthInEms` 24 → 32 widens every drawer in every program using
  glance, not only the one that asked. Sections with short names gain
  whitespace rather than losing anything, but it breaks the relationship
  the old comment described — "a little wider than a card's least"
  (`CardWidthInEms` = 22). 28 would keep that relationship and may be
  enough; 32 is what was measured as comfortable.
- `menuCatcher` becoming `desktop.Hoverable` means the panel now receives
  mouse-move events it did not before. On a desktop widget that is cheap,
  but it has not been measured on a slow machine.
- The uneven button padding is tuned to one face (Adwaita Sans) at one size
  (10pt) and one scale (1.5×). It is expressed in multiples of the theme's
  padding, so it should follow the theme, and that has not been checked at
  another size.
- Glyphs in labels are the caller's, and a face may not have one: U+2190
  rendered as a missing-glyph box in Adwaita Sans. The library cannot know;
  callers should prefer glyphs they have seen render.

## Gaps found

Each of these was found by use, not by review:

1. **Drawer clicks were unreportable.** `Panel.OnTapped` is about cards;
   `drawer.tapped` swallowed everything that was not the switch. A list
   that is navigated was not expressible at all.
2. **A rebuild lost the scroll position.** `renderParts` rebuilds when the
   shape changes, and a live list changes shape whenever a row comes or
   goes, so a scrolled breakdown was dragged back to the top on nearly
   every reading. Spec 069 R12's promise — a reading leaves a scrolled
   breakdown where it is — holds only while the shape is constant, which is
   true of accounts and false of anything live.
3. **No hover existed anywhere.** Not a gap in the drawer: the library had
   no notion of a pointer that was not clicking.
4. **A button kept the face it was born with.** Found by changing the
   panel's font: the buttons' size followed (it is read on every `apply`)
   but their face did not, so they looked half-themed. `refit` is what sets
   a text's face and nothing called it for them, and `Drawer.Restyle` did
   not touch them either. The deeper fault was two places painting a
   button, only one of which the theme path could reach; there is now one.
   Any control added to the drawer needs `refit` at build and a line in
   `Restyle`, which is a sharp edge in the library rather than a one-off
   bug -- hence R20 and the test that asserts over every text there is,
   not only over the buttons.
5. **A button could be drawn at nought.** Found taking the drawer to a
   second section (hayami spec 066) and not on the first, though the fault
   was in neither: the heading row was measured while the button boxes were
   still hidden and nothing measured it again. It had happened to work in
   the order the first section used. This is the second time a gap here was
   a thing that was never asked to lay itself out twice, the first being
   the scroll offset.
6. **The two render paths are not equivalent.** `Lines` cuts to the height
   and never scrolls; `Parts` draws everything and scrolls. A caller who
   picks the wrong one is silently truncated, and because `Detail.Top` is a
   prefix cut, it is the last rows that vanish — which in a list ending with
   its own remainder is the row that says the list is incomplete.

## Compatibility

`Row` now holds a ground behind its contents for the hover (R9), so
`Row.Object()` is a stack whose first child is that ground and not the
label column. Nothing in the library indexes into it -- it is used for
`Visible`, `MinSize` and `Hide` only -- but one of the library's own tests
did, and was changed to ask `Row.Head()` for the label column instead. A
caller reaching into a row's object tree by position would be affected.

## Verification

Built and driven at the Linux desk through hayami's bandwidth spike:
buttons pressed, rows hovered and clicked, drills entered and left, and a
list of ~40 rows scrolled while rows came and went. The scroll holds its
place; the scrollbar resizes as the content does, which is correct and was
judged acceptable.

Not verified: any of it under test, on Windows, on macOS, at another text
size, or at 1× scale.

## E2E Test Plan

Not applicable yet: this is a library, and the end-to-end exercise of it is
hayami's spec. The library's own criteria above are unit-testable and the
tests are the outstanding work.
