# Spec 051: a row in parts, with a pinned label

**Issue**: [#158](https://github.com/ushineko/fynedesygn/issues/158)

## Status: COMPLETE

## Context

hayami spec 031 (ushineko/hayami#112) colours each bandwidth rate by its
size and labels the cooler rows with hardware names. `glance.Row` lacks two
shapes for it.

1. **A value in parts.** A row draws its value as one `canvas.Text` in one
   colour. A bandwidth row's value is two rates, down and up, each with its
   own band. Colouring the whole value by the stronger rate paints a
   2 KiB/s download orange because the upload is 40 MiB/s.
2. **A pinned label width.** `NewMeter` takes a label width; `NewRow` does
   not. A row labelled by a hardware name ("Kraken Elite V2", "i7-13700K")
   grew its card from 194 to 230 px on one desk when the name arrived.

Both are held to the archetype's rule: nothing that arrives later may change
a row's size, because a row that grows grows the window.

**`widgets.FixedWidth` is a floor, not a pin.** It stacks a transparent
rectangle of the width under the object, and a stack is as wide as its
widest child, so a label longer than the width still widens the row (and
the meter that uses it). A pinned label needs a layout whose minimum width
is the width and nothing else, and that gives the label exactly that width
to draw in, so the overhang is clipped (quirk 39).

## Requirements

- R1 `glance.Part{Text, Status, Colour, Bold}` and `Reading.Parts []Part`.
  `glance.Parted(parts ...Part) Reading` builds a reading whose `Text` is
  the parts' concatenation and whose `Status` is Info.
- R2 A row given a reading with parts draws them side by side in the
  monospace face, each in its own colour: `Colour` when set, the
  foreground for Info, the status colour otherwise. `Stale` dims every
  part. `Bold` draws that part bold where the theme has the face
  (quirk 35). A reading without parts draws exactly as before.
- R3 The row's value slot is measured from the parts' concatenation in the
  plain face, the same measurement a plain reading of that text gets, so
  the row's `MinSize` is the same for a parted reading and the same text
  unparted, and switching between the two does not change it. A parted
  reading with an empty `Text` has it filled from the parts.
- R4 `Set` reuses the part texts when the part count is unchanged and lays
  out the value slot only when the count, the mode or a part's width
  changed.
- R5 `Restyle` and `SetTheme` repaint and refit the parts.
- R6 `glance.NewRowWidth(label, blank string, labelWidth float32) *Row`
  pins the label column to `labelWidth` px; 0 is `NewRow`. A pinned label
  longer than the width is clipped to it (callers truncate with an
  ellipsis themselves) and `SetLabel` never changes the row's size.
- R7 Card helpers: `Card.AddRow` takes built rows, so there is no
  card-level constructor to thread the options through. Nothing changes
  there.
- R8 `docs/glance.md` describes both, citing hayami spec 031; the README
  changelog gets an entry under `### Unreleased`; the gallery's Glance
  section shows a parted value and a pinned label.

## Acceptance Criteria

- [x] A test: a parted reading paints each part in its own colour -- status
  colour, `Colour` override, foreground for Info -- and a stale one paints
  every part disabled.
- [x] A test: `Bold` is set on the asked-for part where the theme has a bold
  mono face, and on none where it does not.
- [x] A test: the row's `MinSize` is equal for a parted reading and the same
  text unparted; `Set` with parts of the same widths keeps the size and the
  same `canvas.Text` objects; switching parted to plain and back keeps the
  size.
- [x] A test: a pinned row's `MinSize` is identical for a short label, a long
  label and after `SetLabel` to a different length; the long label is drawn
  at the pinned width.
- [x] A test: a row with parts and a pinned label renders to an image with
  the software painter without a panic.
- [x] `make test`, `make lint` and `make check-diagrams` pass; the gallery
  builds.
- [x] `docs/glance.md`, the changelog and the gallery in the same commit.

## Risks & Assumptions

- **Bold in a family whose bold advance differs from the regular one.** The
  slot is measured from the regular face, so a bold part in such a family
  overhangs the slot by the difference. The design system's mono families
  keep one advance across weights; the note is in the doc.
- **`NewMeter`'s label width stays a floor.** Changing it is a behaviour
  change for every meter consumer and is not asked for here.
- **The README screenshot does not yet show the new gallery card.** The
  Glance section gained "Row, Parted + NewRowWidth"; `docs/img/gallery-glance.png`
  and its alt text are refreshed at the next `--section Glance` shoot, which
  needs the desktop and is not part of this spec's criteria.
- **Rollback**: revert. Everything is additive; no existing signature
  changes.

## Verification

- `make test` (race): all packages ok. `make lint`: 0 issues (per-checkout
  cache). `make check-diagrams`: up to date. `make gallery`: builds.
- Tests, in `glance/rowparts_test.go`:
  - `TestAPartedValuePaintsEachPartInItsOwnColour` -- foreground for Info,
    the status colour, a `Colour` override, all monospace (R2).
  - `TestAStalePartedValueIsDimmedThroughout` -- every part disabled; fresh
    again restores the colour (R2).
  - `TestABoldPartIsBoldOnlyWhereTheThemeHasTheFace` -- under the test theme
    (no bold mono) and Fyne's default theme (R2, quirk 35).
  - `TestAPartedValueIsTheWidthOfTheSameTextUnparted`,
    `TestSwitchingBetweenPartsAndPlainKeepsTheRowsSize` (R3).
  - `TestSettingSameWidthPartsKeepsTheRowAndItsTexts` -- same `MinSize`,
    same `canvas.Text` pointers, same positions (R4).
  - `TestAPartedRowFollowsItsTheme` -- `SetTheme` refits the parts into the
    new face and recolours them (R5).
  - `TestAPinnedLabelKeepsTheRowsWidthWhateverItSays`,
    `TestAPinnedLabelIsAPinWhereAMetersLabelWidthIsAFloor`,
    `TestAPinnedLabelStaysPinnedThroughARestyle` (R6).
  - `TestAPartedPinnedRowRendersToAnImage` -- software painter capture
    (quirk 25 style).
  Mutations checked: recreating the part texts on every `Set` fails the
  reuse test; measuring the slot from the parts with padding fails both
  width tests.
