# Spec 083: a row of cells by how full

**Issue**: [#246](https://github.com/ushineko/fynedesygn/issues/246)

## Status: COMPLETE

## Executive Summary

`glance.HeatStrip` draws a label and one cell per value, each coloured from
the empty track to the primary by how full it is; a NaN keeps an outlined
cell in its place, and the strip's minimum size never changes with its
values or their number. A reviewer should look first at `heatRenderer` in
`glance/heatstrip.go`.

## Context

A meter is one proportion with a limit. A program with many of one kind -- a
processor's cores, each one's load (ushineko/hayami#333, spec 095) -- has no
piece that shows them together: a meter each is a line each, and a row of
percentages is read figure by figure rather than at a glance.

## Requirements

- R1 `glance.HeatStrip`: a label, pinned to a width as a meter's is, and one
  cell per value sharing the rest of the width in equal parts, in order,
  with `HeatCellGap` between them.
- R2 A cell's colour runs from the empty track (`ColorNameDisabledButton`)
  to the primary, by its fraction, clamped to 0..1: an idle cell is the
  track and a full one the colour a meter fills with.
- R3 A NaN value is an empty cell in its place, outlined in the track's
  colour, so the cells after it do not move.
- R4 The strip's minimum size is the same for any number of values and any
  values: it asks for no width beyond its label, and its height is
  `HeatCellHeight`. Nothing it shows reflows a panel.
- R5 It is a `Piece`: it takes the panel's theme and restyles in it.
- R6 docs/glance.md says when to use it rather than meters; a changelog line.

## Acceptance Criteria

- [x] A cell per value, equal widths, in order, with a gap between each.
- [x] Colours: an empty cell the track, a full one the primary, a half one
  between; a value over one clamped.
- [x] A NaN cell is empty and keeps its place.
- [x] The minimum size is unchanged by 0, 1, 3 and 128 values and by any
  values.
- [x] Each test falsified by breaking what it covers.

## Risks & Assumptions

- A colour alone carries the value; the program shows the figures elsewhere
  (hayami's summary has the whole load as a meter). A colour-blind reader
  reads the strip as light and dark, which the track-to-primary run keeps.
- **Rollback**: additive; revert.

## E2E Test Plan

Through hayami's CPU drawer on Plasma and the Windows VM (hayami spec 095):
the strip drawn in a real window, a busy core's cell at the primary.

### Prod visibility

Not applicable: a library.

### E2E results (2026-11-15)

Through hayami's CPU drawer (hayami spec 095), built against this branch:

- Plasma 6 Wayland at 1.5: 32 cells for an i9-14900K's 32 possible
  processors, the two the machine has offline (8 and 9) outlined gaps in
  place; the drawer kept its width.
- Windows 11 VM at 150 %: 14 cells for its 14 logical processors.

Falsified: each of four breakages (no clamp, a height that grows with the
cells, cells with no gap between, a NaN cell filled) fails a test.
