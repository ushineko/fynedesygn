# Spec 048: a cell with a bar

**Issue**: [#148](https://github.com/ushineko/fynedesygn/issues/148)

## Status: COMPLETE

## Context

A `glance.Cell` (spec 016) is a name, a reading and a state, stacked, for
a card of devices read side by side. A percentage in a cell is a number the
eye has to read; a meter's bar is a length the eye compares. hayami's
peripherals card wants the bar under a level reading and nothing under a
band reading, whose segments already draw the level.

The bar must not change the cell's height when it comes or goes: a card
that grew a row because a device gained a level would reflow the window,
which is the rule the whole archetype is built on.

## Requirements

- R1 `(*Cell).SetBar(fraction float64, st fynedesygn.Status)` draws a bar
  under the reading, the cell's width, the meter's bar height and colour
  rules (`MeterFraction` clamping, the status colour, the empty track in
  the disabled colour). `(*Cell).ClearBar()` hides it. A new cell has no
  bar.
- R2 The bar's row is reserved in the cell's layout whether or not a bar is
  shown, so `MinSize` is the same with and without one. A cell with no bar
  draws nothing in the row.
- R3 A dimmed cell (`Reading.Stale`) dims its bar as it dims its reading.
- R4 `Restyle` and `SetTheme` recolour the bar.
- R5 The gallery's Glance section shows a live card with a bar under one
  cell and none under another; the README screenshot and alt text follow.
  `docs/glance.md` gains the bar under "Cells".
- R6 Changelog under `### Unreleased` naming spec 048 and #148.

## Acceptance Criteria

- [x] `make test` and `make lint` pass.
- [x] A test: a cell's `MinSize` is equal before `SetBar`, after it, and
  after `ClearBar`.
- [x] A test: the bar's drawn width is the fraction of the cell's width
  (through the existing line or rectangle finders) and its colour is the
  status colour; a stale cell's bar is the disabled colour.
- [x] The gallery builds; the Glance section is photographed with the
  harness (`--section Glance`) and the README image refreshed.
  *The gallery builds (`make gallery`); the photograph is
  (needs a display): shoot `--section Glance` and replace
  `docs/img/gallery-glance.png`. The alt text already describes the bars.*
  Shot 2026-09-30 with the harness: bars under the mouse, the buds and the
  phones, none under the keyboard.
- [x] `docs/glance.md` and the changelog in the same commit.

## Risks & Assumptions

- **Every cell grows by the bar row**, bar or not, once. That is one
  reflow at upgrade for a consumer, and none afterwards.
- **Rollback**: revert.

## Verification

- `make test` (race): all packages ok. `make lint`: 0 issues (per-checkout
  cache). `make gallery`: builds.
- Tests, in `glance/cellbar_test.go`:
  - `TestACellIsTheSameSizeWithABarAndWithout` -- `MinSize` equal before
    `SetBar`, after it and after `ClearBar` (R2).
  - `TestACellWithNoBarDrawsNothingInItsRow` -- a new cell and a cleared cell
    draw no rectangles; a barred one draws a track and a fill (R1, R2).
  - `TestACellsBarIsItsFractionOfTheCellsWidthInItsStatusColour` -- track is
    the cell's width, fill is 40 % of it at `BarHeight`, fill is the warning
    colour, track the meter's empty colour (R1).
  - `TestACellsBarIsClampedToItsTrack` -- 1.8 fills the track, -0.5 draws
    nothing (R1).
  - `TestAStaleCellsBarIsDimmed` -- stale gives the disabled colour; fresh
    again gives the status colour back (R3).
  - `TestACellsBarFollowsItsTheme` -- `SetTheme` and `Restyle` each recolour
    the fill (R4).
  Each was checked against a mutation (row not reserved, stale not dimming,
  theme not passed to the bar) and failed.
- The spec's `MeterFraction` is the meter's `clamp01`; no function of that
  name exists, and the bar reuses the meter's unexported `bar` widget, so the
  track is `ColorNameDisabledButton` exactly as a meter's is.
