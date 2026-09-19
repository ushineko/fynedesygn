# Spec 015: A table stops re-shaping its text

## Status: COMPLETE

- **Issue**: #3
- **Priority**: High
- **Estimated Complexity**: Low
- **Branch**: `fix/table-resize-cost`
- **Found by**: terrariabonker, on a 7-column catalog table of 6,954 entries

## Context

Dragging a window that shows a `table.Detail` stalls. The window moves a
little, hangs for a second or more, moves again. Reported against
terrariabonker's Compendium section on KDE Wayland at 1.5 display scale, and
the maintainer's first three guesses were all wrong: the row count, the
thumbnails, and the number of matches shown. Filtering the table down to a
handful of rows changed nothing, which is what ruled all three out.

A CPU profile taken during the drag ended the guessing:

```
glfw.(*window).resized                        49%
 └ widget.(*tableRenderer).Layout
    └ widget.(*tableCellsRenderer).Refresh     44%
       └ widget.(*Label).Refresh               43%
          └ widget.(*RichText).updateRowBounds 42%   <- harfbuzz
runtime.gcDrain                                36%   <- clearing up after it
```

**Fyne resizes every visible cell on every layout.** A `widget.Label` whose
wrapping or truncation is on re-shapes its text when it is resized, through
harfbuzz, because that is how it decides where to wrap or where to put the
ellipsis. The text has not changed; the width has. `Detail` set
`Truncation = TextTruncateEllipsis` on every body cell and every header, so a
drag re-shaped the whole visible table on every frame, and the allocation that
came with it kept the collector busy for another third of the CPU.

The size of a table does not come into it, which is why filtering did not help:
the cost is per *visible* cell, and a screenful is a screenful.

### The path Fyne leaves open

`widget.lineBounds` returns before measuring anything when wrapping and
truncation are both off:

```go
if max.Width <= 0 || wrap == fyne.TextWrapOff && trunc == fyne.TextTruncateOff {
	return lines, 0 // don't bother returning a calculated height
}
```

`Detail` can take that path because it already approximates text width by rune
count -- `Measure` has counted runes at 7px each since it was written, to size
the columns. The same metric cuts the cell's own text, so the ellipsis survives
and nothing is shaped.

That approximation was already load-bearing for the column widths; this spec
does not introduce it, it reuses it. A proportional font puts the cut a rune
early or late on occasion. The ellipsis is there either way, so no word is
silently halved.

## Requirements

- R1 A body cell draws its text with wrapping and truncation off, and cuts its
  own text to the column width with an ellipsis.
- R2 A header cell does the same: Fyne refreshes the header row with the body.
- R3 Neither cell nor header calls `SetText` when the label already says what it
  is being asked to say.
- R4 The column width and the cell's cut come from one metric, declared once.
- R5 A benchmark pins the resize cost, and says in its comment what it is for.

## Acceptance Criteria

- [x] AC1 `BenchmarkResize` over a 300-row, 8-column table stays under 100 µs
      per resize. Measured: **738,199 ns before, 27,179 ns after** -- 27x.
- [x] AC2 The existing table tests pass unchanged: the cut is invisible to
      `Cell`, which still returns the full text.
- [x] AC3 `make lint` clean.
- [x] AC4 A consumer sees it: terrariabonker's Compendium was the report, and
      the same build it was reported against carries the fix.

## Risks & Assumptions

- **The cut is approximate.** A column whose text is mostly wide glyphs will cut
  a rune or two early. The alternative is measuring, which is the cost being
  removed. `Measure` has made the same assumption for column widths since it was
  written and nobody has reported a badly sized column.
- **A cell's full text is no longer on screen in full when it is long.** That was
  already true -- it was ellipsised before -- but the cut point may now differ by
  a character from what Fyne would have chosen.
- **Rollback** is a revert of one commit; the table's API does not change.

## Alternatives Considered

- *Leave the truncation to Fyne and cache the shaped result.* Fyne offers no
  hook for it: the shaping happens inside `RichText.Refresh`, which the table
  calls through `Resize`.
- *Use `canvas.Text` cells instead of `widget.Label`.* Cheaper still, and it
  loses the theme's text colour handling and importance, which is what colours a
  row by its `Status`.
- *Guard `UpdateCell` alone.* Tried first, and it is in the fix -- but it moved
  nothing on its own, because the expensive refresh comes from Fyne resizing the
  cell, not from the table setting its text. Recorded here because it is the
  obvious fix and it is not sufficient.
