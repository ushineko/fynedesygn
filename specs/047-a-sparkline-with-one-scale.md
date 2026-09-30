# Spec 047: a sparkline with one scale

**Issue**: [#143](https://github.com/ushineko/fynedesygn/issues/143)

## Status: COMPLETE

## Context

`glance.Sparkline` (spec 016) holds several series and scales each to its
own range, widened to a minimum span about its middle. That is the right
plot for the cooler card: coolant at 40 °C and a processor at 90 °C on one
line, each readable. It is the wrong plot for bandwidth. Two interfaces at
1 KiB/s and 20 MiB/s would draw with the same amplitude, and an interface's
down and up could not be compared. A bandwidth plot wants one scale for
every series: zero at the bottom, the greatest sample in the window at the
top.

A consumer drawing several series also needs colours that tell them apart
and carry no status meaning. The design system owns colour; a consumer
choosing its own is the copy the library exists to end.

## Requirements

- R1 `Sparkline.SetScale(Scale)` with `ScaleEach` (today's behaviour, the
  default) and `ScaleShared`. Under `ScaleShared` every series is drawn
  against one range: low is zero, span is the greatest sample across all
  series, widened to the largest of the series' minimum spans when the
  peak is below it. A window with no samples draws nothing, as now.
- R2 `SeriesColour(th fyne.Theme, i int) color.Color` returns the i-th
  series colour. Series 0 is the theme's link colour (`ColorNameHyperlink`),
  so the first series follows the scheme's accent. Series 1, 2 and 3 are a
  fixed categorical set that avoids every status hue: violet, teal, magenta,
  in two variants chosen by the luminance of the theme's background
  (`ColorNameBackground`): lighter and more saturated on a dark background,
  darker on a light one, picked to read on Breeze Dark and Breeze Light and
  named in the code as the design system's categorical set, with why the
  palette cannot supply them. Index 4 and beyond wraps. (Revised during
  implementation; the first draft drew all four from the palette, which
  cannot supply them. See Verification.)
- R3 `Faded(c color.Color) color.Color` returns c at reduced alpha (the
  factor a constant with its reason), for a secondary series drawn as a
  pair with a primary one.
- R4 The gallery's glance section shows a second sparkline under
  `ScaleShared` with two pairs of series, so the shape is visible. The
  README's glance screenshot is refreshed if that section's screenshot is
  the one the README shows; `docs/glance.md` describes the two scales and
  when each is right.
- R5 Changelog under `### Unreleased` naming this spec.

## Acceptance Criteria

- [x] `make test` and `make lint` pass.
- [x] A test: two series, one peaking at 100 and one at 1, under
  `ScaleShared` the second's drawn height is a hundredth of the first's
  (asserted through whatever the widget exposes for a test; add an
  exported measurement hook if none exists, named in the spec); under
  `ScaleEach` both fill the height.
- [x] `SeriesColour(i)` for i in 0..3 differs from each other and from the
  theme's warn and bad colours, in every scheme.
- [x] `Faded` keeps the hue and lowers alpha.
- [x] The gallery builds and the new sparkline is photographed
  (`docs/img/`), or the screenshot harness is run by the reviewer; noted.
  The gallery builds (`make gallery`); the photograph, taken at review:
  `tools/screenshot.sh --section Glance --scheme "Breeze Dark" docs/img/gallery-glance.png`. The
  README's alt text for that image already describes the new plot.
  Shot 2026-09-30 with the harness (`--section Glance --scheme "Breeze Dark"`):
  eno1 in link blue with its upload faded, wlan0 in violet as the low line
  along the floor, one scale.
- [x] `docs/glance.md` and the changelog in the same commit.

## Risks & Assumptions

- **Shared scale is opt-in**, so the cooler card does not change.
- **Rollback**: revert.

## Verification

- Shared scale (`glance/sparkline_test.go`):
  `TestASharedScaleDrawsTracesAgainstOneRange` (peaks 100 and 1 at 100 px
  draw rises of 100 and 1, both from the bottom),
  `TestEachScaleIsTheDefaultAndFillsTheHeight` (the same pair under
  `ScaleEach`, set, switched back, and never set, each fills the height),
  `TestASharedScaleIsWidenedToTheLargestMinimumSpan`,
  `TestASharedScaleOfZerosDrawsAlongTheBottom`.
- No measurement hook was added: the existing tests already read the drawn
  segments through `fynetest.All[*canvas.Line]`, and a segment's rise is the
  trace's drawn height.
- Colours (`glance/seriescolour_test.go`), each over `theme.Schemes()`
  where it applies: `TestSeriesColourCyclesAndWraps` (series 0 is the link
  colour, 4..7 and -1 wrap), `TestSeriesColoursAreDistinctAndNotWarnOrBad`,
  `TestSeriesColourFollowsTheBackground`,
  `TestFadedKeepsTheHueAndLowersTheAlpha`.
- Why R2 was revised. The first draft cycled link, the selection accent,
  positive and hover from the palette, which cannot meet a "not a status
  colour" criterion in any of the nine schemes: the selection accent is
  `ColorNamePrimary`, which is `StatusInfo`'s colour; positive is
  `ColorNameSuccess`, which is `StatusGood`'s; hover equals the selection
  accent in Breeze Dark, Breeze Light, Adwaita Dark and Adwaita Light; and
  link equals it in Windows Dark and Windows Light. Hover is also plain
  white or black in the Windows and macOS schemes.
- Categorical values: dark background violet `#b197fc`, teal `#4fd1c5`,
  magenta `#e07bd0` (contrast on Breeze Dark's window 6.5, 8.5, 6.0); light
  background violet `#7c4dbd`, teal `#0f8a80`, magenta `#b0369a` (on Breeze
  Light's window 5.0, 3.7, 4.8; all above the 3:1 non-text minimum).
- `make test` (race), `make lint` (0 issues), `make gallery` pass.
- Photograph: taken at review with `--section Glance`, docs/img/gallery-glance.png.
