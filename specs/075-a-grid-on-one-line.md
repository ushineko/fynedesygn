# 075 — a grid on one line

**Issue**: #222

## Status: COMPLETE

## Executive Summary

`CellGrid.SetOneLine` puts every shown cell on one line and makes the grid
ask for that line's width, so a card of cells grows wider instead of taller.
Off by default; `Lines` still wins. Reviewers should look first at
`cellGridLayout.across` and `MinSize` in `glance/cell.go`, the one place the
choice is made.

## Context

A `CellGrid` fits as many equal-width cells on a line as its width allows
and wraps the rest. In a 260 px glance panel that is two across, so hayami's
peripherals card with four devices is a two-by-two block, and each further
device makes the card taller. A wider window, dragged by the user, fits more
across.

hayami wants its user to choose the other direction (hayami #236): every
cell on one line, so the card, and with it the panel, grows wider instead of
taller. The grid has no such mode, and a consumer cannot get one without
replacing the layout. The rule both repositories keep is that a shape the
library lacks is added here and not copied into the app.

`Lines` (spec 057) already overrides how a grid wraps: one cell per line,
whatever the width. This is the opposite case, every cell on one line,
whatever the width.

## Requirements

- R1 `CellGrid.SetOneLine(on bool)` and `CellGrid.OneLine() bool`. Off by
  default, which keeps today's behaviour: as many across as fit.
- R2 With it on, every shown cell is on one line. The grid's `MinSize` is
  that line: `n` cells of the widest cell's minimum width, with `n-1` gaps
  between them, and one cell high. A fixed-size glance window grows to it on
  its own (quirk 34).
- R3 A width beyond the minimum is shared equally among the cells, as it is
  today when a line is full. A hidden cell takes no space (`visible`,
  unchanged), so hiding one lowers the minimum, and the consumer's
  `Panel.Resize` shrinks the window back.
- R4 `Lines` wins. A grid in the `Lines` arrangement is one cell per line
  whether or not one-line is on, because `Lines` is the full-width,
  one-reading-per-line shape and a line of cells side by side is not that.
  Turning `Lines` off brings one-line back.
- R5 `SetOneLine` re-lays the grid. It does not resize the window: the grid
  cannot see the panel. The consumer calls `Panel.Resize` after it, in the
  same call, as it does after any change to a card's contents
  (docs/glance.md, "Sizing").
- R6 docs/glance.md documents the mode beside `Lines`, and the README
  changelog has the line under `### Unreleased`.

## Acceptance Criteria

- [x] Off: four cells in a grid two cells wide wrap to two lines, unchanged
      (the existing layout tests pass unchanged).
- [x] On: four cells are laid out at the same y, left to right, and the
      grid's `MinSize` width is four cells and three gaps, its height one
      cell.
- [x] On, at a width wider than the minimum: the cells fill it equally, the
      last one ending at the grid's right edge.
- [x] On, with one cell hidden: three cells on the line, and `MinSize`
      narrows to three cells and two gaps.
- [x] On, in `Lines`: one cell per line. With `Lines` turned off again, one
      line.
- [x] Each new test fails with the one-line branch removed (all five in
      `glance/oneline_test.go` fail with `oneLine()` answering false).
- [x] A glance window holding a one-line grid grows to the line's width and
      shrinks back after `Panel.Resize` when a cell is hidden, asserted from
      the window's size and not only from the layout.
- [x] `make test`, `make lint` pass.

## Out of scope

- A cap on how many go on a line (`SetAcross(n)`). Nothing asks for one;
  "as many as fit" and "all of them" are the two directions a card can grow.
- Scrolling a line wider than the screen. A glance panel with more cells
  than a screen's width is not a case anyone has.

## Risks & Assumptions

- A long line widens every card, because each card takes the widest card's
  width (docs/glance.md). That is the point of the mode and is the
  consumer's choice to make.
- The minimum width follows the cell count, so a cell appearing or going
  changes the window's width. That is the cell-count exception hayami
  already accepts for height (its spec 070); the mode moves it to width.
- Additive API; off by default. Rollback: revert, and a consumer that called
  `SetOneLine` stops compiling, which is the honest failure.

## Verification

Headless layout tests over the Fyne test driver for the geometry, and a test
window for the size the window takes. The picture of a real panel is
hayami's, through its spec 071.
