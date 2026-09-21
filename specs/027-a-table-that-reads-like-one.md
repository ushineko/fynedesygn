# Spec 027: a table that reads like one

**Issue**: [#35](https://github.com/ushineko/fynedesygn/issues/35)

## Status: COMPLETE

## Context

`markdown.Pane` drew a pipe table with `container.NewGridWithColumns`, which
gives every cell the same size. Three consequences, all visible at once in
hotaru's About section:

- **Every row was as tall as the tallest row in the table.** A two-column
  table of a short label beside a paragraph came out with an inch of nothing
  between the rows.
- **Every column was the same width** whatever was in it, so a column of
  one-word labels took half the table.
- **Nothing separated the rows**, so the eye had nothing to follow across.

It read as two columns of floating text.

### What makes it a table

Columns proportioned to what is in them, rows each as tall as their own
content, and a rule under the header with a fainter one between rows.

Column widths come from the longest cell in each column, which is what a
reader's eye uses and what a browser's table layout approximates. Clamped at
both ends: a column of one-word labels beside a column of paragraphs would
otherwise be drawn at a twentieth of the width and wrap every label into a
column of single words, and a table should not tip over on one long cell.

### Not Fyne's table widget

It lives in a scroller, and a scroller inside a document takes the wheel from
the page -- the same trap `CodePanel` exists to avoid, and quirk 7 in
`docs/fyne-quirks.md`.

### Ruled on every side a reader follows

Under the header, between the rows, under the last one, and down the column
boundaries. The first version had only the first two, and the report was
exact: "no vertical lines, bottom line, and there is an extra line at the
top". A grid of lines is what tells somebody they are looking at a table
before they have read a word of it.

The verticals are drawn *over* the rows rather than between them, so a row
does not have to know how many lines the table has or where they go.

### An empty header cell is not a horizontal rule

The extra line at the top was two, one per column, and they were the header.

A two-column table of label and description is often written with an empty
header -- `| | |` is how hotaru's README writes it -- and the first version
made a header bold by wrapping each cell in asterisks. An empty cell became
`****`, which Markdown renders as a thematic break.

Editing somebody's Markdown to change how it looks is the fault, and the empty
cell is only the case that showed it: a cell already carrying emphasis, or a
pipe, or a backtick was going to produce something nobody wrote. The header is
emphasised by style now, on the segments the cell parsed to, leaving what they
are alone.

### Both forms of table

A pipe table always has a first row with a `|---|---|` under it -- that is
what makes it a table rather than text with pipes in it -- so a table written
*without* headers is written with empty ones:

	| | |
	|---|---|
	| Lighting | every device OpenRGB can see |

Both are ordinary Markdown and common, and hotaru's README uses the second.
Drawn as a header, an empty row is a blank strip above the table; drawn as
data, it is a blank first line. It is neither: the first row is the header
whatever it holds, and an empty one is simply not drawn.

That was two mistakes in a row -- first drawing the empty header as a header,
then, having noticed, demoting it to a data row.

### And sides

A table ruled on the inside and open at the edges is a table somebody has to
infer the shape of. The verticals are the two edges and every boundary between
them; a single-column table has no boundary and still has two sides.

### A roof

A table is closed. Without a rule above the first row the table hangs off the
one under its header, and the top reads as the page having run out rather than
as the table having started.

### The measurement, again

The first attempt asked each cell its height and then gave it its width, and
every row came out 47 pixels tall whether it held a word or a paragraph -- a
wrapping `RichText` answers "how tall are you" with one line until it has been
resized.

This is spec 021's finding for the third time, and a layout is where it bites
hardest: `Layout` knows the width and `MinSize` does not. So the measurement
is taken during `Layout`, at the width it was given, and kept for `MinSize` to
answer with.

Worth saying plainly because it was in the comment before it was in the code:
the comment claimed width-first and the code did height-first, and the test is
what said so.

## Requirements

**R1. Each row is as tall as its own content.**

**R2. Columns are proportioned to what they hold**, clamped so a narrow
column still fits a word and a wide one does not take the whole table.

**R3. Ruled on every side a reader follows**: above the first row, under the
header, between the rows, under the last one, down each column boundary, and
down both edges.

**R3b. Both forms of table draw as what they are.** A table written without
headers is written with empty ones, and an empty header is neither a header
nor a row.

**R3a. A cell is never rewritten to change how it looks.** The header is
emphasised by style, because an empty cell wrapped in asterisks is a
horizontal rule and a cell with a backtick in it is something else again.

**R4. No scroller**, because a document's wheel belongs to the document.

**R5. A table does not set the document's minimum width.** Its cells are as
wide as the table, and a layout reporting that as a minimum would make the
page scroll sideways.

## Acceptance Criteria

- [x] AC1. A one-line row is shorter than a four-line row in the same table.
- [x] AC2. A column of labels is narrower than a column of paragraphs, and
      two columns of the same length come out even.
- [x] AC3. A narrow column keeps enough width to fit a word.
- [x] AC4. A table contains no scrollable.
- [x] AC5. A row with more cells than the header does not panic.
- [x] AC6. The text of every cell is still readable from the tree.
- [x] AC7. An empty header cell renders as text rather than as a thematic
      break, and a header cell's text is unchanged.
- [x] AC8. There is a rule above the first row, under the header, between
      each pair of rows, under the last row, and one vertical per column
      boundary.
- [x] AC9. A table whose header cells are all empty draws no header row and
      no blank one in its place; a header naming only one of its columns is
      still a header.
- [x] AC10. The outermost verticals are at the table's left and right edges,
      and a single-column table has two of them.

## Risks & Assumptions

- **Width by character count is an approximation**, not a measurement. A
  column of wide glyphs against one of narrow ones will be proportioned by
  count rather than by pixels; the clamps bound how wrong that can be, and a
  real measurement would mean shaping every cell twice.
- **The measurement is cached against the last width laid out at.** A width
  change re-measures; a cell whose *content* changes without a resize would
  keep the old height, which no document does because a block's content is
  fixed once parsed.
- **Rollback** is one function.

## Alternatives Considered

Considered `widget.Table`; rejected because it brings a scroller and a
document's wheel belongs to the document.

Considered `layout.NewFormLayout` for the common two-column case; rejected
because it sizes the first column to its widest cell with no clamp, which is
the same failure in the other direction for a table whose first column holds
a sentence.
