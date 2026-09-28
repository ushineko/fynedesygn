# 040 — A reading that is a block

**Issue**: #106

## Context

`glance` has one shape for a reading: `Row`, which is a label at the left, a
value at the right and the slack between them. It is right for most of what a
glance panel carries. A rate, a temperature and a fan speed are two facts of
equal weight, and a column of them reads down.

It is wrong for a battery. The monitor these components come from draws a
peripheral as a block — the device's name centred, the percentage large and in
the middle beneath it, what the battery is doing quiet beneath that — and the
emphasis is the point. In a row the percentage is a small number at the right
margin in the same weight as the name, so the eye reads the name and then hunts
for the figure. In a block the figure is what the eye lands on and the name is
only which device it belongs to. For a battery that is the right way round: the
number is the reading and the name is the index.

hayami is the reporter. It drew its peripherals as rows, argued in a doc comment
that the monitor's blocks were an artefact of a narrow panel, and reversed that
after putting the two programs side by side. It now has the shape in its own
`internal/view` and draws it in the terminal panel; it cannot draw it in the
window, because this library has `Row`, `Meter` and `Sparkline` and nothing
block-shaped.

## Requirements

### A cell is a name, a reading and a state

`glance.Cell` is three stacked, centred pieces:

- **Name** at the top, in the interface face, at the ordinary text size.
- **Reading** in the middle, in the monospace face, **larger** than the other
  two, carrying the `Reading` type this package already has — its text, its
  status colour and its stale dimming, exactly as `Row` does.
- **State** underneath, in the interface face, at the ordinary size, in the
  disabled colour.

It is a **live handle**, like `Row` and not like a build function. A card is
redrawn from a poll several times a minute, and rebuilding the widget tree at
that rate is what "build the tree once; update it in place" exists to stop.

The reading is larger because that is the whole of what distinguishes a cell
from a row. The factor is a function of the text size rather than a pixel
count, for the reason `CardPadH` gives: a size set for one face is
proportionally wrong at another.

### A cell grid lays them out in columns

`glance.CellGrid` holds cells and lays them out across the width it is given,
as many to a line as fit, wrapping.

- Every cell is the same width, which is the widest cell's minimum. A column
  that fitted its own contents would put the readings at a different place on
  every line, and a column of figures that do not line up is what the
  fixed-width formatters exist to prevent.
- The count follows the width. The monitor draws two to a line because its
  panel is 260 px wide and it has four fixed slots; this fits what it can,
  which is the same answer at that width and a better one at any other.
- A grid goes into a card through `AddObject`, like a sparkline.

### The width rule holds

Nothing transient may reflow the panel, and a device name is as transient as
the device. A cell's name is **truncated** to the cell's width rather than
widening it, so a mouse waking up with a long name does not resize the window.

The grid's own minimum width is a whole number of cells, so a card of cells is
as wide as its widest reading and not as wide as the longest name in it.

### No icon

The monitor draws a battery glyph beside the percentage. It is deliberately not
carried: an icon per device type means the library knowing what a mouse is,
which is an application concept and the one rule this module does not bend. A
consumer that wants one puts it in the name.

## Acceptance criteria

- [x] `glance.Cell` exists, with `NewCell`, `Set`, `SetName`, `SetNote`,
      `Reading`, `Restyle`, `SetShown`, `Shown` and `Object`, each documented.
- [x] A cell's reading is drawn larger than its name and its state.
- [x] A cell's reading takes its colour from `Reading` exactly as a row's does,
      including the stale dimming winning over the verdict.
- [x] `Set` repaints without re-laying-out, as `Row.Set` does.
- [x] `Restyle` brings a cell up to date after a scheme or text-size change,
      because a `canvas.Text` holds a literal colour and size.
- [x] `glance.CellGrid` exists, with `NewCellGrid`, `Add`, `Cells`, `Restyle`
      and `Object`, each documented.
- [x] A grid puts as many cells on a line as its width fits, and wraps.
- [x] Every cell in a grid is the same width.
- [x] A name longer than its cell is truncated rather than widening the cell,
      verified by a cell whose name is changed to a much longer one leaving the
      grid's minimum size unchanged.
- [x] A grid narrower than two cells puts one to a line rather than clipping.
- [x] The gallery's glance section shows a card of cells.
- [x] `docs/glance.md` has a "Cells" section saying when a reading is a cell
      and not a row.
- [x] `README.md` carries the changelog entry under `### Unreleased`.
- [x] `make test` passes. `make lint` reports eight issues, all of them in a
      stale `../../fd-wrap` path that predates this branch and reproduces on an
      untouched `main`; nothing in this change is flagged. It needs its own
      issue and is not this spec's to fix.

## Risks & Assumptions

- **A new exported shape is a contract.** `Cell` and `CellGrid` are additive:
  nothing existing changes, so a consumer on the current version is unaffected
  and the module stays `v0.x` with no breaking change to name.
- **The larger reading is a judgement.** The factor is chosen to match the
  archetype at its own face and is a named constant, so a consumer that
  disagrees can see what it is rather than measuring pixels.
- **The grid's layout is custom rather than `layout.NewGridWrapLayout`.** That
  layout takes one fixed size for every cell and is told it rather than
  measuring, so a text-size change would leave it stale. This one measures.
- **Rollback**: revert the commit. Nothing existing is touched.

## Status: COMPLETE
