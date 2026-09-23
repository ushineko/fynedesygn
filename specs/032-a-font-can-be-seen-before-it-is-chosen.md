# Spec 032: a font can be seen before it is chosen

**Issue**: TBD — to be filed before the PR.

## Status: COMPLETE

## Executive Summary

Fonts are chosen in a chooser that shows a sample in the highlighted family
and commits nothing until Choose, rather than in a dropdown of names drawn in
a face that tells you nothing about any of them. The list itself is *not*
drawn in the fonts it lists: this machine carries 311 families at 2.5 MB each,
and rendering the menu that way would cost 778 MB and 731 ms. `PreviewFont`
reads a family without keeping it, so looking through every font costs what
looking at one costs. Reviewers should read `dialogs/choosefont.go` and
`theme/fonts.go` (`LoadFont`, `PreviewFont`, `readFamily`).

## Context

Reported from jira-viewer:

> make font selection dropdowns that render the font entry in its style, for
> judging the font visually before committing to selection. if this is a
> performance or memory issue, make a font preview mechanism that will show
> what the font will look like as you select it.

The Appearance section offered two `widget.Select`s of family names. Choosing
applied immediately to the whole window, so the only way to see a font was to
take it, and the only way back was to take another. With 311 names that is a
search by trial.

## Why the list is not drawn in its own fonts

Fyne draws every widget in the app's theme, so the only way to render a name in
its own face is to put it under a theme of its own — a `container.ThemeOverride`
per row, each holding a `*Font` read from disk.

Measured on this machine (`theme.TestLoadingEveryFamilyIsNotFree`):

| | |
|---|---|
| Families found | 311 |
| Regular faces, all families | 230 MB |
| All four faces, all families | 778 MB |
| Time to read them all | 731 ms |
| Per family | 2.5 MB, 2.4 ms |

`LoadFont` caches for the life of the process, which is right for the two or
three families a program draws in. A row-per-font list would read every file it
drew and never give any of it back — and a `widget.List` recycles rows, so
scrolling the list once would walk the whole 778 MB.

So the design is the second of the two the reporter offered.

## Design

- **`theme.PreviewFont(name)`** reads a family's faces and does not cache
  them, returning a cached copy when the program already draws in that family.
  `readFamily` is the shared reader; `LoadFont` is `readFamily` plus the cache.
- **`Appearance.PreviewTheme(family, mono)`** is the appearance with one family
  swapped, so a preview changes one thing and not two.
- **`dialogs.ChooseFont`** is a filter, a plain list of names, and a sample
  drawn under `PreviewTheme` for the highlighted family. One font is alive at a
  time. Choose answers; Cancel and browsing answer nothing.
- The Appearance section's two dropdowns become buttons carrying the current
  family, which open the chooser.

The sample shows the family's own name, a line of prose, and the shapes that
tell families apart (`0O1lI!|`, a ticket key, a hex colour, brackets) — a font
here is chosen for a window full of ticket numbers and log lines as often as
for reading.

### A sample most families cannot draw

Reported after the first build: the sentence looked the same whatever was
highlighted. Two causes, both real.

The preview followed `OnSelected`, which a click and Space fire and the arrow
keys do not — `widget.List` moves its own `currentHighlight` and reports that
through `OnHighlighted`. Arrowing down the list moved a bright highlight while
the sample sat on the last thing clicked, so the list named one family and the
sample drew another. Two highlights on screen at once, which is what the
screenshots showed. The preview listens to both now.

And most families cannot draw the sample at all. Noto ships a font per writing
system, and most of those carry punctuation and digits and not one Latin
letter: `NotoSerifOriya-Regular.ttf` is missing 35 of the pangram's 44
characters. What appears is another font's glyphs standing in — identical for
every such family, and indistinguishable from a preview that has stopped
working.

So `Font.Missing` counts what a family cannot draw and `Font.Covered` returns
what it can. A family that draws the sentence names its face; one that cannot
says so and shows its own digits and punctuation instead, in its own shapes.
This needs a font parser: `golang.org/x/image/font/sfnt`, already in the
module graph as Fyne's own dependency and now a direct one.

### The sample is drawn from the file, not through a theme

The first three attempts previewed with `container.ThemeOverride`, and the
sample kept its old face. Every probe said it should not: the override reached
the painted `canvas.Text` objects, and `theme.CurrentForWidget` on them
resolved to the highlighted family — under the real GL driver as well as the
test one. It took a reader pointing at a sans-serif sample captioned
`drawn in LiberationSerif-Regular.ttf` to make the point that the theme is not
what decides the glyphs.

Three ways the family is lost between being decided and being drawn:

- text is measured through a path handed no object at all
  (`canvas.Text.MinSize` → `Driver.RenderedTextSize`), so it cannot see a
  scope;
- the face is cached against a scope id (`cache.WidgetScopeID`) that the text
  objects do not reliably carry, and a miss falls back to the app's own theme;
- monospace text is resolved through a *list* — the theme's face, Fyne's
  bundled monospace, a system font for the generic "monospace" family — and
  what comes out is not reliably the face the theme named, which is why every
  family in the monospace chooser drew identically.

`canvas.Text.FontSource` survives all three: it is Fyne's own "draw this
string from this file", checked before any theme or scope
(`painter.CachedFontFace`). The sample is built from `canvas.Text` with the
family's file on it, and there is no override in the chooser at all.

## Requirements

- R1. A family can be seen before it is chosen, at the size and scheme the
  window is using.
- R2. Browsing commits nothing.
- R3. Looking through every family costs about what looking at one costs.
- R4. A family the program already draws in is not re-read to preview it.
- R5. The monospace family is previewed as monospace, and chosen separately.
- R6. The preview follows the highlight however it moved — click, Space or the
  arrow keys.
- R7. A family that cannot draw the sample says so rather than showing another
  font's glyphs as if they were its own.
- R8. The sample is drawn from the family's file, not through a theme.
- R9. A proportional family offered as a monospace face says so.

## Acceptance Criteria

- [x] AC1. The chooser draws its sample under a theme of its own and names the
  family (`TestChooseFontPreviewsBeforeCommitting`).
- [x] AC2. Opening the chooser and browsing answers nothing; Choose answers the
  highlighted family (same test).
- [x] AC3. Cancel answers nothing and leaves no overlay
  (`TestChooseFontCancelsWithoutChoosing`).
- [x] AC4. `PreviewFont` leaves nothing in the font cache, and `LoadFont` still
  does (`TestPreviewingDoesNotFillTheCache`).
- [x] AC5. The cost of a family is measured, and the measurement fails if a
  face ever becomes small enough to change the design
  (`TestLoadingEveryFamilyIsNotFree`).
- [x] AC6. Moving the keyboard highlight moves the preview
  (`TestTheKeyboardMovesThePreview`).
- [x] AC7. A family with no Latin letters says so and shows what it does carry
  (`TestAFamilyThatCannotDrawTheSampleSaysSo`); one that draws the whole
  sample names its face instead (`TestAFamilyThatDrawsTheSampleNamesItsFace`).
- [x] AC8. Every sample line carries the family's file in `FontSource`
  (`TestChooseFontPreviewsBeforeCommitting`), and two families drawn as
  monospace differ from each other
  (`TestTheMonospaceChooserPreviewsTheFamily`).
- [x] AC9. A proportional family offered as a monospace face says "columns
  will not line up" (`TestAProportionalFamilySaysSoInTheMonospaceChooser`).
- [x] AC10. The whole suite passes and `make lint` is clean.

## Risks & Assumptions

- A preview re-reads the file each time the highlight moves. At 2.4 ms per
  family that is imperceptible for one, and it is the cost that buys the
  bounded memory.
- The filter narrows on a substring of the family name, which is what somebody
  typing "mono" or "noto" means. It does not know styles or foundries.
- Rollback is a revert: `LoadFont` keeps its behaviour, and the section falls
  back to two dropdowns.

## Alternatives Considered

- **A row per font in the list.** Rejected on the measurement above.
- **A bounded cache of preview fonts.** Rejected as premature: one font at a
  time already bounds it, and a cache would only help somebody moving back and
  forth between two families, at the cost of holding both.
