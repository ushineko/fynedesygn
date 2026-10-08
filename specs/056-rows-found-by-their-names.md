# Spec 056: rows found by their names

**Issue**: [#168](https://github.com/ushineko/fynedesygn/issues/168)

## Status: COMPLETE

## Context

A glance `Card` takes rows with `AddRow`, which appends, and lists them with
`Rows`. Its pieces go into one column in the order they are added, so a row
appended after a plot is drawn under the plot.

hayami works round both. It builds each card once with four spare hidden rows
(`RowSlack`) placed before the plot, because a reason for a gap can arrive at
any poll. It pins the label column of a row holding a hardware name by the
row's position ("the cooler's named rows come first and stay first"). When a
section outgrows the spares it appends a row, which lands under the plot: the
"Coolant  no cooler" row drawn through the trend that RowSlack was introduced
to stop.

hayami's architecture review (phase 2, its spec 044) makes the cooler a list of
probes whose length depends on the machine: one graphics card or two, a cooler
or none, a motherboard or VRM temperature where LibreHardwareMonitor gives one.
Rows then come and go with what is read, which spare rows and positions cannot
follow.

## Requirements

- R1 A row has an optional ID: `(*Row).SetID(id string)`, `(*Row).ID() string`.
  IDs are unique within a card. `InsertRow` refuses a second row with an ID the
  card holds (`ErrRowID`); `AddRow` panics on one, since it returns nothing and
  a duplicate is a fault in the program building the card. An empty ID is never
  looked up and never collides, so consumers that set none are unaffected.
- R2 `(*Card).InsertRow(at int, r *Row) error` puts r at position at among the
  rows (0 above every row, `len(Rows())` under the last). A position outside
  that range is `ErrRowIndex`. A refused insert leaves the card unchanged.
- R3 Among the rows, never under what follows them: at the end, the row goes
  straight after the last row in the card's column, so an object added after
  the rows (a plot, a meter, a grid) stays under it; with no rows yet, it goes
  to the top of the card.
- R4 `(*Card).RemoveRow(id string) bool` takes the row out of `Rows` and off
  the screen, and reports whether there was one. The empty ID removes nothing.
- R5 `(*Card).RowByID(id string) *Row` finds a row, or nil. The empty ID finds
  nothing.
- R6 A row's per-row settings travel with it. The label column a row was
  pinned to (`NewRowWidth`) is the row's, so it holds wherever the row ends up.
- R7 A row inserted into a card showing last-known values (`SetStale`,
  `SetLastKnown`) is dimmed with the rest.
- R8 Inserting or removing a row changes the card's height by the row and
  nothing else; it is a change of shape, like `SetShown`, and the consumer
  calls the panel's `Resize` after it. Nothing here resizes on a value.
- R9 No change for a consumer that only appends: `AddRow` and `Rows` behave as
  before for rows without IDs.

## Acceptance Criteria

- [x] R2 A row inserted between two takes its place in `Rows` and in the drawn
      column; the rows after it keep their IDs (headless test).
- [x] R3 A row inserted at the end of a card with a plot is drawn above the
      plot; the first row of a card holding only a plot goes above it.
- [x] R4 A row removed by ID leaves `Rows` and the column; a second removal and
      the empty ID remove nothing.
- [x] R5 `RowByID` finds by ID; an unknown or empty ID finds nothing, even when
      the card holds a row without an ID.
- [x] R1 A duplicate ID is refused by `InsertRow` (card unchanged) and panics
      in `AddRow`; rows without IDs never collide.
- [x] R2 A position outside the rows is refused and changes nothing.
- [x] R6 A pinned label column keeps its width after a row is inserted above
      its row.
- [x] R7 A row inserted into a stale card reads as stale.
- [x] R8 Inserting a row grows the card's height and not its width (same
      label); removing it gives the height back.
- [x] Each test falsified: the end-insert placed at the column's end, the first
      row placed at the column's end, uniqueness off, stale dimming off,
      removal leaving the object drawn, the empty ID found, the index check
      off; each fails its test and passes restored. The pinned-column test
      fails with the row built unpinned.
- [x] Gallery: "Card, InsertRow over a plot" in the glance section; rendered
      and looked at (both inserted rows above the plot).
- [x] `docs/glance.md` ("A card whose rows follow what is read") and the
      changelog updated.

## Risks & Assumptions

- **Additive.** A consumer that sets no IDs sees no change. `AddRow` panics
  only for a duplicate non-empty ID, which no existing consumer can produce
  since IDs did not exist.
- **Lookup is a scan.** A card holds a handful of rows; a map would be a
  second structure to keep in step with `rows` for no measurable gain.
- **`AddRow` into a stale card does not dim the row**, as before; `InsertRow`
  does (R7). Changing `AddRow` would change behaviour for existing consumers,
  and R9 rules that out.
- **Thread**: like every `Card` method, the new ones run on the UI thread; a
  consumer reaching them from a poll goes through `fyne.Do`, as it already
  does for `Set`.
- Rollback: revert. Nothing persists.

## Gaps found

- `CellGrid` has the same spare-slot pattern in hayami (`CellSlack`) and could
  take the same treatment. Not needed for the cooler; left until a consumer
  asks.

## E2E Test Plan

hayami spec 044 builds its cooler card from probes with these methods and drops
`RowSlack` for that card. A machine whose second graphics card or cooler
appears after start draws the new row above the trend, without a restart.
Photograph it there.

## Verification

2026-10-07, Windows 11, Go 1.26, Fyne 2.8.1: `go test -race ./...` and
`make lint`'s configuration (golangci-lint v2.12.2) as recorded in the commit;
the gallery card rendered through the test driver.
