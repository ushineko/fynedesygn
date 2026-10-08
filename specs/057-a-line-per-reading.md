# Spec 057: a line per reading

**Issue**: [#170](https://github.com/ushineko/fynedesygn/issues/170)

## Status: COMPLETE

## Context

A glance `Panel` lays its cards out as `Stack` or `Grid` (`glance.Arrangement`,
issue #124). hayami has a third arrangement, `row`: one line per reading across
the whole width, its label at the left, its value at the right and a bar taking
the slack, with no section headings. It is the shape of a session manager's
pane (hayami's usage pane is `row` with one section selected), and hayami's
rule is that every section renders in every arrangement in both of its shells.

Its terminal draws `row` (`internal/view/arrange.go`, `renderRow`): a section's
rows as lines, each cell as a line (name and state, bar, reading), each meter
as a line (label, bar, figures, reset), with the label, figures and reset in
columns shared across the pane so a figure is in the same place on every line.
Its window maps `row` to `Stack`, because this library had nothing to draw it
with, and building a layout of its own from glance internals in the
application is what the design system's rules forbid. hayami spec 047 (its
parity test, hayami#144) recorded the gap and opened issue #170.

## Requirements

- R1 A third `glance.Arrangement`, `Lines`. `Panel.SetArrangement(Lines)`
  hands it to every card, and `Panel.Add` hands the panel's arrangement to a
  card added later. It is one column, as `Stack`.
- R2 In `Lines` a card leaves out its heading and everything that is not a
  reading: a plot, and any object added with `AddObject` (or a piece that is
  not a row, meter or grid). Rows are drawn as they are.
- R3 A meter in `Lines` is one line: label at the left, caption and trailing
  value at the right, the bar between them taking the slack and centred on the
  line. The stats row is left out.
- R4 A grid of cells in `Lines` is one cell per line, whatever the width. A
  cell is one line: name, then state, at the left, each elided to the name
  budget; the reading at the right, bold and in its verdict's colour, at the
  name's size; the bar, when the cell has one, between them.
- R5 Leaving `Lines` puts back exactly what it hid. An object the consumer had
  hidden itself stays hidden.
- R6 Rows are not touched: their IDs (spec 056) and the consumer's handles
  survive a switch, and a value set in one arrangement is drawn in the next.
  An object added while in `Lines` is hidden until `Lines` is left.
- R7 Switching while the panel is in a window resizes it once, at the switch.
- R8 The lines of one card share columns: its meters take the widest label,
  caption and trailing value among them, and the cells of a grid the widest
  name, state and reading, so every bar starts and ends at the same x.
- R9 `Card.SetArrangement` is exported, for a card shown outside a panel (the
  gallery's demo is one).
- R10 docs/glance.md ("A line per reading"), the gallery, the changelog.

## Design

**The name.** `Row` is the card's row type, so the arrangement could not be
`Row`. `Strip` is what hayami calls a meter drawn as segments. `Lines` says
what it draws: one line per reading.

**The panel hands the arrangement to the cards; nothing is rebuilt.** Stack
and Grid already share one layout so that switching while the window is up
does not swap containers under it, which loses sizes and makes the panel jump.
`Lines` keeps that: the cards' objects stay in place and only their visibility
and layout change. A meter keeps the same label, bar, caption and trailing
objects and moves them into a one-line container; a cell keeps its texts and
its layout switches branch. That is what makes R6 hold for free.

**What is hidden is remembered.** The card records what it hid (`tucked`) and
shows exactly that on leaving, so an object its consumer hid is not shown by a
change of arrangement (R5).

**Shared columns, from the peers, at layout time.** A card hands each of its
meters a pointer to the card's list of meters; a cell knows its grid. The
one-line layouts read their peers' widths when they lay out, so a caption
that arrives after the switch moves the shared column with it. The widths
are the formatters' fixed widths, so the columns do not jitter (the glance
no-reflow rule).

**Hidden, not dropped.** A plot in `Lines` is hidden rather than drawn as a
one-line trail as hayami's terminal draws it. A sparkline one text line high
says little, and a card whose plot is its main reading can stay in `Stack`;
hayami's parity allow-list can say so for its trails.

## Acceptance Criteria

- [x] R1: `TestAPanelHandsLinesToItsCards`: a card present and a card added
  later both lose their headings in `Lines` and get them back in `Stack`.
- [x] R2: `TestLinesLeavesOutTheHeadingAndThePlot`.
- [x] R3: `TestAMeterInLinesIsOneLine`: label, caption and trailing on one
  line, the trailing value against the right edge, no stats row.
- [x] R4: `TestCellsInLinesAreOnePerLine`, at a width that would fit two cells
  as lines side by side, so one per line is the arrangement's doing.
- [x] R5: `TestLeavingLinesPutsBackWhatItHid`: heading, plot, stats row and
  side-by-side cells come back; a consumer-hidden object stays hidden.
- [x] R6: `TestRowsAndLateObjectsSurviveASwitch`.
- [x] R7: `TestSwitchingToLinesResizesTheWindowOnce`: shorter in `Lines`, back
  to its height on leaving.
- [x] R8: `TestTheBarsOfACardLineUpInLines`: meters' bars and cells' bars start
  and end at the same x, with names, states and figures of different widths.
- [x] Falsified: the heading kept; the grid's one-per-line rule removed; every
  added object shown on leaving; `Panel.Add` not handing the arrangement on;
  the trailing value placed at the left; the panel not telling its cards; a
  meter or a cell laid out by its own widths alone. Each failed its test and
  passed once restored.
- [x] R9, R10: the gallery's "Card, Lines: rows, cells, meters" demo; docs and
  changelog.

## Risks & Assumptions

- **Additive.** A consumer that never sets `Lines` draws exactly as before:
  the new code runs only in `Lines`, and `Card.SetArrangement(Stack|Grid)` on
  a card never set to `Lines` returns at once.
- **The heading's `GoneMarker` is hidden in `Lines`**; a stale card says it by
  its dimmed rows alone. Recorded in docs/glance.md.
- **Rollback**: revert. hayami keeps drawing `row` as `Stack` until it takes
  this release.

## Gaps found

- The real-window screenshot tool needs KDE; on the Windows desk this was
  built on, the demo was rendered through Fyne's test driver
  (`Canvas().Capture()`) and looked at, and the README's Glance screenshot is
  not regenerated.
- A cell's bar in `Lines` sits on the line, so a cell with a bar and one
  without are the same height there too; a meter's stats row has no place in a
  line and is not drawn.

## Verification

2026-10-07, Windows 11, Go 1.26.0, gcc 16.2.0 (MSYS2 UCRT64), Fyne v2.8.1:

- `go test -race ./...`: every package ok.
- golangci-lint v2.12.2 with the repository's configuration: no new findings
  in the changed packages.
- The gallery's `Lines` demo rendered through the test driver: the
  Processors rows without heading or plot, the Peripherals cells one per line
  with their bars in one column, the Usage meters one per line with bars from
  x=82 to x=191 and captions from x=200 on every line.
