# Spec 082: a drawer as wide as what it shows

**Issue**: [#244](https://github.com/ushineko/fynedesygn/issues/244)

## Status: COMPLETE

## Executive Summary

A program sets a drawer's width with `Drawer.SetWidthInEms`, when it opens
or is given a new subject, and asks `Panel.Room` how wide a drawer the
screen still takes; drawers of different widths stand side by side, and
Fill widens each by the same number of pixels from its own width. A drawer
never set is the 32 ems it was, and spec 079's and 080's tests pass
unchanged. A reviewer should look first at `planFill` and `placeFilled` in
`glance/fill.go`, then the row's layout and `Room` in `glance/drawer.go`.

## Context

Every drawer is `DrawerWidthInEms` (32) text sizes wide (spec 068), whatever
it shows. Spec 079 lays a row of drawers out at `i × need()`, so every drawer
in it is taken to be that width, and `Fits` asks whether one more of the
first drawer's `need()` fits. Spec 080 fills a full row by giving every
drawer the same number of pixels.

A program with more to say than a name and one figure gets the room only by
filling. hayami's process lists (ushineko/hayami#332, spec 092) carry columns
that do not fit 32 ems, and their owner wants them in an ordinary drawer:
the drawer as wide as its list, decided when it is drawn, and never resized
by what arrives afterwards (ushineko/hayami#335).

## Requirements

### The width

- R1 **`Drawer.SetWidthInEms(ems float32)`** sets the drawer's own width in
  text sizes of the panel's face; **`Drawer.WidthInEms() float32`** reports
  it. A new drawer's is `DrawerWidthInEms`, so a program that never calls it
  sees no change. A width below `CardWidthInEms` is taken as
  `CardWidthInEms`.
- R2 **The width is the program's to set, never the library's to work out.**
  Nothing the drawer is given (`SetContent`, a row's `Set`, a title) changes
  it. The library does not measure content.
- R3 **When it takes effect.** Set on a closed drawer, it is the width the
  drawer opens at. Set on an open drawer to a different value, it is a change
  of shape: the window is resized once, in the same call, as for a drawer
  opening. Set to the width it has, nothing happens.
- R4 **Being in ems, the width follows the text size**, as the default does
  now; `Restyle` keeps each drawer's ems.

### The row

- R5 **`Drawer.Reach()` is the drawer's own**: its width and the gap. Each
  drawer in the row stands beyond the reaches of the drawers opened before
  it, on their side; drawers of different widths sit side by side with one
  gap between each, the cards unmoved.
- R6 **`Panel.Room(left, right float32) float32`** is the widest drawer, in
  text sizes, that fits beside the window now, given the room between the
  window and each edge of its work area as `Fits` takes it: the room on the
  drawers' side less a gap, in ems; with no drawer open, the larger side's.
  Zero or less is no room. **`Panel.RoomHere() (ems float32, known bool)`**
  is the same read by the window itself where it can (Windows), as
  `FitsHere` is.
- R7 **`Fits` is unchanged in meaning**: whether a drawer of
  `DrawerWidthInEms` fits (`Room >= DrawerWidthInEms`). A program that sets
  widths asks `Room` and chooses the width to open at, at most what it
  returns.

### Filling

- R8 **A filled row widens every drawer by the same number of device
  pixels**, each from its own width. Each drawer's own width is taken in
  whole pixels at the window's scale (rounded as spec 080 R4 rounds the
  cards), the room's whole pixels are divided by the number of drawers, and
  what the division leaves over is padding beyond the outermost, as now.
  With every drawer at the same own width this is spec 080's row, pixel for
  pixel. *Amends spec 080 R4*, which made every filled drawer the same width.
- R9 `SetWidthInEms` on an open drawer of a filled row ends filling in the
  same change of shape, as closing a drawer does (spec 080 R7): the widths
  were planned from the old one.
- R10 `Drawer.Available()` reports the width the drawer is shown at: its own,
  or its filled width.

### Documentation

- R11 `docs/glance.md`, "A drawer beside the cards": the width is the
  program's, set when the drawer opens or the person gives it a new subject,
  never by a reading; how to choose it with `Room`.

## Acceptance Criteria

Headless, in `glance/width_test.go`. Five mutations each failed a test: the
row laid out at one drawer's width (`TestDrawersOfDifferentWidths`), Fill
dividing the room equally (`TestFillWidensEachByTheSame`; spec 080's tests
still passed, their drawers being one width), a new width that did not end
filling (`TestANewWidthEndsFilling`), `Room` without the gap
(`TestRoomIsTheWidestThatFits`), and no lower bound
(`TestTheWidthsBounds`):

- [x] A drawer opened without `SetWidthInEms` is `DrawerWidthInEms` wide,
  and a row of such drawers lays out, fits and fills as before: spec 079's
  and 080's tests pass unchanged (`TestTheWidthsBounds`, the package).
- [x] Three drawers of 32, 44 and 38 ems: each is its own width, one gap
  apart, beyond the cards on either side; the window is the cards and the
  three reaches wide, and the cards did not move
  (`TestDrawersOfDifferentWidths`).
- [x] `SetWidthInEms` on a closed drawer does not resize the window; on an
  open one to a new value it resizes once; to the same value, not at all
  (`TestSettingAWidthIsOneChangeOfShape`).
- [x] `SetContent` and a row's `Set` on a drawer of 44 ems leave it 44
  (`TestContentDoesNotSetTheWidth`).
- [x] Below `CardWidthInEms` is taken as `CardWidthInEms`
  (`TestTheWidthsBounds`).
- [x] `Room`: with room for 40 ems and a gap on the drawers' side it returns
  40; with none open, the larger side's; `Fits` agrees with
  `Room >= DrawerWidthInEms` across a sweep of rooms
  (`TestRoomIsTheWidestThatFits`).
- [x] Filled, drawers of 32, 44 and 38 ems at a scale of 1.5: each grows by
  the same whole number of pixels and the pixels add up to the window, the
  padding fewer pixels than drawers; on a window, on either side, each grew
  by the same amount and unfilled is its own width again
  (`TestFillWidensEachByTheSame`).
- [x] `SetWidthInEms` on a filled row's drawer ends filling in one resize
  (`TestANewWidthEndsFilling`).
- [x] `Restyle` keeps each drawer's ems (`TestRestyleKeepsTheEms`).
- [x] `Available()` reports each drawer's own width less its margins, and
  its filled width less the same while filled
  (`TestAvailableIsTheShownWidth`).

On a real window, through hayami (its spec 093), the library having no
program of its own with lists of different widths:

- [x] Plasma 6.8 at 1.5: drawers of different widths open side by side,
  measured from a grab, each its own width; filled and unfilled as above;
  the cards did not move.
- [x] Windows 11 at 150 %: the same.

## Risks & Assumptions

- A wider drawer leaves room for fewer beside the panel, so a row reaches its
  cap sooner. That is the program's trade to make: it chooses the width.
- A program could resize an open drawer on every reading by calling
  `SetWidthInEms` with a measured value. The library cannot tell a reading
  from a new subject; R11 says which it is for.
- Spec 080's equal-width tests hold because every drawer in them is the
  default width; R8 changes the rule only for rows whose drawers differ.
- **Rollback**: revert. The API is additive and a program that does not call
  it sees no change.

## Threat model delta

No boundary touched: layout only, no new input, script or file.

## Alternatives Considered

- A width the library measures from the drawer's content: rejected, because
  content changes with readings and the width would follow them (the glance
  rule), and only the program knows the widest a list can be.
- Filled drawers all the same width, as spec 080 had them: rejected, because
  a wide drawer and a narrow one filled to one width throws away the reason
  one was wider.
- A user setting for every drawer's width: rejected in ushineko/hayami#335;
  the program knows what each list needs.

## E2E Test Plan

Through hayami's spec 093 on the desk (Plasma, Wayland, at 1.5) and the
Windows VM (150 %), its branch built against this one through a local
`replace`:

1. Open the CPU drawer, then Shift-click a Network row: two drawers of
   different widths, side by side, the cards unmoved (R5).
2. Open to the cap and Fill: each drawer grew by the same pixels, the window
   at the screen's edge (R8). Unfill: the own widths again.
3. Change the text size in preferences: the drawers keep their proportion
   (R4).

Results, 2026-11-15, through hayami's spec 093 (its results have the
detail):

- Plasma 6.8 at 1.5: Bandwidth by process at the default width and CPU by
  process at its own stood side by side, 481 and 533 pixels, one gap apart,
  the cards unmoved (step 1). With a third drawer at the cap, filled: each
  grew by 47 pixels (533, 483, 534 to 580, 530, 581), the window at the
  screen's edge, the cards at their place; unfilled, the own widths again
  (step 2).
- Windows 11 at 150 %: the same row of different widths (557 and 493 pixels
  added to the window), and four drawers filled to the screen's edge and
  back, the cards at x 1100 throughout.
- Step 3, the text size, was not made on a window: `TestRestyleKeepsTheEms`
  holds it.

### Prod visibility

Not applicable: a library with no telemetry.
