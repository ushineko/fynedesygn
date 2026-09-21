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

**R3. A rule under the header, and between the rows.**

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
