# Spec 088: a heading drawn larger, and a title end clear of the tuck glyph

**Issue**: [#263](https://github.com/ushineko/fynedesygn/issues/263)

## Status: COMPLETE

## Executive Summary

`Card.SetTitleSize(name)` draws a card's heading -- both titles and their
icons -- at a theme size other than the ordinary text's, and a card with a
title end now keeps it clear of the panel's tuck glyph. Both are for hayami's
clock, whose heading is its reading. Reviewers should look first at
`Card.sizeClear`, whose arithmetic ties the card to the glyph's place in
`glyphLayout`.

## Context

hayami's clock (hayami spec 106) is a card whose heading is the time at the
left and the date at the right (spec 086). On the desk, two things were
wrong with it:

- **It is small for what it is.** It is drawn at the title's size, which is
  the ordinary text size, and the clock is a line read from further away
  than a card's rows.
- **The date runs under the tuck glyph.** The glyph (spec 081) stands half
  a side margin in from the first card's right edge and is 1.25 text sizes
  wide; the heading ends a whole side margin in. The glyph reaches 0.625 text
  sizes past the heading's end, and a title end -- the date, on the top
  card -- is drawn under it.

## Requirements

- R1 `Card.SetTitleSize(name fyne.ThemeSizeName)` draws the title, the title
  end and both icons at that theme size. The default is
  `theme.SizeNameText`, which is what a card drew before. A change of theme
  re-measures them at the new face's size.
- R2 The card's margins, rows and gone marker keep the ordinary text size; a
  larger heading makes the card taller by the heading's growth and nothing
  else. Another card is unchanged.
- R3 A card with a title end keeps room after it, so the end stops a quarter
  of a text size short of where the tuck glyph stands. **Every** card with a
  title end keeps it, not only the top one: a date that moved when its card
  was moved down the stack would be a reflow of its own. A card without a
  title end is unchanged.
- R4 The glyph's side is one factor (`glyphEms`) that the glyph and the
  card's room both read, so the two cannot drift apart.

## Acceptance Criteria

- [x] `TestAHeadingDrawnLarger`: at `SubHeadingText` the time and date are
      that size, the icons `IconScale` times it, the card taller, another
      card's title unchanged; back at `Text` the card is its old height.
- [x] `TestATitleEndStopsShortOfTheTuckGlyph`: with the glyph shown over the
      clock card, the date ends at least a quarter text size before the
      glyph's left edge and less than a text size before it.
- [x] `TestATitleAtTheHeadingsRightEnd` (spec 086 R9) says the date ends
      short of the figures' end, within three text sizes of it.
- [x] Falsified: the room set to zero fails the glyph test (the date's end
      at 381 against the glyph's left edge at 371.25); the heading held at
      the text size fails the size test.
- [x] The gallery's clock card and `examples/glance-monitor -clock` draw
      their heading at `SubHeadingText`.
- [x] `docs/glance.md` and the changelog say so.
- [x] `make test`, `make lint`.

## Risks & Assumptions

- Additive: `SetTitleSize` is new and its default is the old behaviour. The
  room after a title end moves an existing consumer's date in by about a
  text size; hayami is the only consumer and asked for it.
- The room is computed from the card's own text size and the glyph's from
  the panel's. They are the same face in every program built on the panel;
  a card given a face of its own different from its panel's would misplace
  the room by the difference.

## Threat model delta

No boundary touched: layout only.

## E2E Test Plan

hayami built against this branch, its panel captured with
`tools/screenshot.sh --what panel` with the glyph shown: the clock's time
and date at the sub-heading size, the date ending clear of the glyph.
Done on the desk, 2026-11-22.
