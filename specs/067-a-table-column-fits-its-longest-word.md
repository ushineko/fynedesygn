# Spec 067: a table column fits its longest word

**Issue**: [#194](https://github.com/ushineko/fynedesygn/issues/194)

## Status: COMPLETE

## Executive Summary

A markdown table's column is never narrower than the longest word in it,
measured in the face, size and padding it is drawn with. The width left over is
shared by the content weights as before. hayami's README table read "Sectio n",
"Periph erals" and "Bandwi dth" on its About page; it now reads whole.
Reviewers should look first at `grid.widths` and `grid.measure` in
`markdown/tablegrid.go`.

## Context

`columnWeights` sizes columns from the longest cell in each, counted in
characters and clamped to 8–90. Beside two columns of paragraphs, a column of
eleven-character labels came to 11/191 of the table: about 56 points at a
1000-point table, less than "Section" in bold plus the RichText's inner
padding on both sides. A character count cannot know the face, the size or the
padding. Fyne's word wrap then broke the word where the column ended.

## Requirements

- R1. A column is at least as wide as its widest word, measured with
  `fyne.MeasureText` in the segment's own text style and size (bold in the
  header, monospace in code, which also gets its chip's padding), plus the
  cell's inner padding on both sides.
- R2. Where the floors fit, a column whose weighted share is under its floor
  is held at the floor and the rest is shared by the weights among the others,
  repeated until none is under. Where nothing is under, the widths are what
  the weights alone gave.
- R3. Where the floors do not fit, they are shared in proportion to
  themselves.
- R4. The rows and the column rules use one value, so the rules stay between
  the cells.
- R5. The floors are measured again when the text size changes.

## Acceptance Criteria

- [x] `TestANarrowColumnDoesNotBreakAWord`: hayami's table shape at 1000
  points wide keeps each label on one line. Falsified: with the floors forced
  to zero, the label column is 56 points and the test fails.
- [x] `TestTheFloorsLeaveTheRestToTheWeights` covers R2 and R3.
- [x] The existing table tests pass unchanged apart from the constructors.
- [x] A picture of hayami's About page built against this branch shows the
  sections table with every label whole, against 0.9.7's broken one.

## Risks & Assumptions

- A word wider than the table cannot fit; R3 breaks every column evenly
  rather than one completely.
- The floor uses the theme's sizes without a widget, as `fyne.MeasureText`
  does. A table under an overridden theme with a different text size is
  measured at the application's size.

## E2E Test Plan

hayami's About page, built with a `go.work` that uses this checkout, scrolled
to "The four sections" by its Contents link and photographed: before and
after.
