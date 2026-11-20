# Spec 085: a heat strip's cell is a choice

**Issue**: [#252](https://github.com/ushineko/fynedesygn/issues/252)

## Status: INCOMPLETE

## Executive Summary

A heat strip's cells can now be chosen: `SetCellLabels` puts a word under
each cell at one fixed extra line, `SetSelected` marks a cell with an outline
and a bold label, `CellAt` maps a canvas position to a cell, and
`Drawer.SetCellTapped` routes a click on a cell to the program, as
`SetRowTapped` does rows. All additive. Reviewers should look first at the
dispatch order in `Drawer.tapped` (cells before rows) and at `heatCells.MinSize`.

Falsified: seven breakages (labels off-centre, no line for labels, a height
that follows the label count, no outline, the gap belonging to no cell, the
labels' line counted as a cell, the drawer ignoring cells) each fail a test.

## Context

`glance.HeatStrip` (spec 083) was built to be read: a processor's cores,
each one's load. hayami's Development card (hayami#360) wants a strip that is
also chosen from: two weeks of a person's repository activity, one cell per
day, where a click on a day opens what happened that day and the day stays
marked while its page is open.

Three things are missing, and a spike against v0.1.121 (patched in a scratch
copy, run on a real Plasma window) found the shape of each:

- **A click cannot reach a cell.** A click in a drawer is the drawer's
  (`Drawer.tapped`): its buttons, its switch, the rows given to
  `SetRowTapped`, or nothing. A widget's own `Tapped` is never called, since
  the panel takes every click to find the card or drawer under it.
- **A cell cannot be marked as the one chosen.**
- **Nothing says which cell is which.** A row of fourteen cells is read as
  days only when each carries its day's initial, aligned under it; a label
  beside the strip cannot be.

## Requirements

- **R1** `HeatStrip.SetCellLabels(labels []string)` draws a word under each
  cell, centred on it, in the placeholder colour at a smaller size. From the
  first call with labels the strip is a fixed height, the cells' and one
  line's, whatever the labels say and however many there are; `nil` puts the
  strip back to the cells' height.
- **R2** `HeatStrip.SetSelected(i int)` marks cell `i`: an outline in the
  foreground colour, and its label in the foreground colour and bold. `-1`,
  or an index with no cell, marks none. `Selected()` reads it back.
  Marking a cell changes no size.
- **R3** `HeatStrip.CellAt(at fyne.Position) int` is the cell under a canvas
  position, or `-1` when the position is off the cells (the label, the
  labels' line, outside). The gap after a cell is that cell's, so a click
  between two cells lands somewhere.
- **R4** `Drawer.SetCellTapped(on func(*HeatStrip, int), strips ...*HeatStrip)`
  makes the cells of the strips given clickable, as `SetRowTapped` does rows:
  a click on a cell calls `on` with the strip and the cell, and nothing else
  in the drawer is told. A click elsewhere in the drawer behaves as before.
  No strips, or a nil function, makes the cells a thing to read again.
- **R5** docs/glance.md says when a strip's cells are choices, and a
  changelog line under `### Unreleased`.

## Acceptance Criteria

- [x] AC1 Labels: one per cell, each centred under its cell (R1).
- [x] AC2 The strip's minimum height with labels is the same for one-letter
  and long labels and for 1, 14 and 128 cells, and returns to
  `HeatCellHeight` with `nil` (R1).
- [x] AC3 Selecting outlines exactly that cell; `-1` and an index past the
  end outline none; the minimum size is unchanged (R2).
- [x] AC4 `CellAt` gives each cell's index at its middle and in the gap after
  it, and `-1` on the label and on the labels' line (R3).
- [x] AC5 In an open drawer, a tap on a cell calls `SetCellTapped`'s function
  with that strip and index, and not `SetRowTapped`'s; a tap on a row still
  reaches the row's (R4).
- [x] AC6 Each test falsified by breaking what it covers.

## Risks & Assumptions

- Additive: no existing call changes, and a strip given no labels draws as it
  did. **Rollback**: revert.
- The labels' size is the theme's text size scaled by 0.85, measured with
  `fyne.MeasureText`, so the height follows the theme rather than a constant.
- Hit-testing uses the cells' laid-out geometry, as `rowAt` does for rows; a
  strip that is hidden or has no cells is never hit.

## E2E Test Plan

Through hayami's Development drawer (hayami spec 102) on Plasma 6 Wayland at
1.5: a real click (ydotool) on a day's cell opens that day's page and marks
the cell; a click on the same cell again goes back; the strip keeps its
height throughout. Covers AC1–AC5 on a real window.

### Prod visibility

Not applicable: a library.
