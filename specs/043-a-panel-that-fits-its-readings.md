# 043 — A panel that fits its readings

**Issue**: #112

## Context

A glance window is supposed to be the size of what it is drawing. A consumer
measured its widest card at 218 pixels and got a 298-pixel window at every
start, and dragging it narrower by hand did not survive a restart. Three faults
were behind it, and a fourth turned up on the way.

## Requirements

### The first resize takes the content

`Panel.Resize` refuses to pull the window narrower than its **current** width
when the panel is resizable. That is right for a width the user chose and wrong
for the one Fyne made before the first layout, and the comparison cannot tell
them apart — so the first guess is kept for the life of the program.

The panel remembers the width it last **asked** for. The first resize always
takes the content; afterwards a current width that is not the one it asked for
is a width something else set — the user, or the compositor on their behalf —
and is honoured.

The height follows the content either way. That is what quirk 34 is about and
it does not change.

### A floor that can be declined

`MinWidth` is a good guard against a panel of one short reading looking like a
chip, and costs nothing while the content is wider than it. A consumer whose
cards are narrower pays for it in dead space.

`NoMinWidth` asks for no floor. Zero still means the default, so nothing that
does not ask for it changes.

### A grid uses the columns it needs

`cellGridLayout` lays out `perLine(width)` columns whether or not there are
that many cells. Two devices in a grid with room for three sit in the first two
thirds and leave the last third empty. Never more columns than cells.

### Text is re-measured when its size changes

A `canvas.Text` draws inside its Size and clips what does not fit. A container
gives it that size from `MinSize` when the container is laid out, and a
`Refresh` does not re-run a parent's layout — so a title that grew from 8 pt to
9 kept the width it was measured at and lost its last letter.

Every `Restyle` here ends by resizing its texts to what they now measure and
refreshing the row that holds them. It is Fyne's behaviour rather than this
package's, so it gets a quirk.

## Acceptance criteria

- [x] A resizable panel's first resize takes the content's width, not the
      window's.
- [x] A width set after that — by the user or the compositor — is kept.
- [x] `NoMinWidth` gives a panel as wide as its widest card; zero still means
      the default floor; a floor that is asked for is kept.
- [x] A grid never lays out more columns than it has cells, and the cells reach
      the right-hand edge of the width they are given.
- [x] Card titles, rows, cells and meters are re-measured on a size change.
- [x] `docs/fyne-quirks.md` has the text-clipping quirk, numbered and in order.
- [x] `README.md` carries the changelog entry under `### Unreleased`.
- [x] `make test` passes.

## Risks & Assumptions

- **The re-measure has no test that can fail.** The Fyne test driver lays out
  on demand, which is the very thing whose absence the fault is about: a canary
  written for it passed with the fix removed, so it was deleted rather than
  kept as decoration. It is verified the way this repository's own rule says to
  verify anything visual — on a photograph of a real window, at 9 pt and at 13,
  with the size changed at runtime. The quirk row says so rather than naming a
  canary it does not have.
- **`asked` is a width, not a promise.** A compositor that grants something
  other than what was asked for looks, on the next poll, like the user
  resizing. That is the right reading: it *is* the window being a size this
  panel did not choose.
- **Not a breaking change.** A consumer that passes no `MinWidth` and never
  resizes sees the same floor and the same behaviour, minus the first-guess
  lock-in — which is the fix.
- **Rollback**: revert the commit.

## Status: COMPLETE
