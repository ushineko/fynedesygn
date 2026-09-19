# Spec 017: Glance in the gallery

## Status: COMPLETE

- **Issue**: #9
- **Priority**: Medium
- **Estimated Complexity**: Low
- **Branch**: `feat/glance-gallery-screenshots`
- **Follows**: spec 016, which listed a gallery section as deliberately not
  built

## Context

Spec 016 built `glance` and left two things undone that the module's own rules
require.

**The library rule is not met.** "Every exported identifier has a doc comment
and appears in the gallery." `glance` appeared nowhere in
`cmd/fynedesygn-gallery`, so a change to a card's styling could only be judged
from a test assertion. The gallery exists precisely so a change is judged by
looking at it before it is judged in a program.

**The vocabulary had a hole in it.** Drawing the pieces side by side showed
that the archetype's other recurring shape was missing entirely: a proportion
of something with a limit, drawn as a labelled bar with a caption — the quota
block the monitor uses for Claude Code usage. `glance.Meter` adds it.

**`docs/glance.md` had no pictures.** It describes a shape — a stack of cards
on a desktop, values that hold their column, a plot with no legend, a card
dimmed because its source went away — entirely in prose.

The window itself cannot be drawn inside the gallery: frameless and
always-on-top rendered inside another window is a picture of neither. So the
gallery shows the vocabulary, and a separate capture of
`examples/glance-monitor` shows the archetype as it actually sits on a desktop.

### What the screenshots found

The captures are the point of this spec rather than a by-product of it. Five
defects were invisible until something was rendered and looked at — two of them
only against a real desktop and a real thirteen-rule `kwinrulesrc` — and all
five are fixed here, plus a sixth found by the new meter's own test:

1. **Values were clipped at the right edge.** A glance card drawn across a
   1400 px content pane put its value column under the scrollbar. The cards are
   now drawn at `panelWidth`, a little over `glance.MinWidth`, which is the
   width a real glance window settles at. A card stretched to a gallery pane is
   a shape no program produces.
2. **The context menu's tap catcher took a card's worth of height and caught
   almost nothing.** It was added beside the cards in the panel's stack rather
   than over them, so it contributed its own minimum height — a visible empty
   band at the top of the window, 52 px of it — and a secondary tap only opened
   the menu inside that strip. It is now stacked over the panel through
   `Panel.Overlay`, with a zero `MinSize`, which is also what makes it win the
   pointer (quirk 24).
3. **Cards had no surface of their own.** `docs/glance.md` says separation
   comes from contrast because Fyne cannot draw a translucent window (quirk
   32) — and `Card` drew no background at all, so three cards ran together as
   one column of rows. `Card` now draws a fill, a hairline border and the
   scheme's corner radius from palette tokens.
4. **Cards were square.** `Card` took the scheme's radius token, which is a
   *control* radius: 2 in Breeze, Oxygen and Adwaita. At card size that reads
   as a square, and no desktop draws a floating panel that way. `CardRadius`
   is 8, between the monitor's 8 for a section and 12 for the window around
   them.
5. **`glance/kwin` did not restore a file it had changed.** Read against a real
   `kwinrulesrc` of thirteen rules, an install added one blank line per
   section, and an install followed by a remove left a stray newline at the
   end: the separator written before the appended section was re-read as a
   trailing blank of the section above it and survived the removal. `render`
   now counts a blank line the file already had and ends the file with exactly
   one newline. Three tests cover it; none of the original fifteen did,
   because they all asserted on substrings rather than on the whole file.

6. **A meter's caption changed the window's width as it filled.** Caught by
   `TestFillingAMeterDoesNotResizeTheWindow`, and it took two goes. Formatting
   the percentage with `fmt.Sprintf("%.0f%%", pct)` grew the caption from "5%"
   to "100%"; switching to `Percent`, which pads, was not enough either,
   because the caption was drawn in a proportional face where a space is
   narrower than a digit, so `"  9 %"` is visibly narrower than `"100 %"`. The
   caption is now monospace, like a row's value. The rulebook already said
   both halves of this and the widget obeyed neither.

The doc was wrong about which token, too. It named `ViewAltBG` for the card
surface; that is the alternating-row token and is *darker* than the window in
every dark scheme, so a card drawn in it disappears into the panel. `ButtonBG`
is the one that sits a step from the window in both light and dark.

### One harness change

`tools/screenshot.sh` always passed `--section`, which every program built on
the shell accepts. A glance window has no sections, and Go's `flag` package
exits on a flag it does not declare rather than ignoring it, so the program
never started. `--section` is now passed only when there is one, the way
`--scheme` already was, and `SETTLE` allows a longer pause for a window whose
content arrives over several polls.

## Requirements

- R1 A Glance section in the gallery showing cards live and stale, rows, the
  formatters at several magnitudes and the sparkline, each captioned with its
  Go name, in the shape the Widgets section uses.
- R2 Cards in the gallery drawn at the width a glance window actually uses.
- R3 `docs/img/gallery-glance.png`, captured by the existing harness.
- R4 A capture of `examples/glance-monitor` as it sits on a desktop, in
  `docs/glance.md`.
- R5 Alt text for both images describing what is in them.
- R6 The defects the captures exposed are fixed, with a test for each that
  could have caught it.
- R7 A meter: the labelled, captioned, graded bar the archetype uses for a
  quota, in the package, in the gallery, in the example and in the rulebook.

## Acceptance Criteria

- [x] AC1 `cmd/fynedesygn-gallery` has a Glance section, listed between Forms
      and Documents, and the section-name test names it. (R1)
- [x] AC2 Cards in the section are drawn at `panelWidth` and no value is
      clipped. (R2)
- [x] AC3 `docs/img/gallery-glance.png` is captured through
      `tools/screenshot.sh --section Glance`, so `make screenshots` keeps it
      current without a special case. (R3)
- [x] AC4 `docs/img/glance-monitor.png` shows the frameless panel as a glance
      window actually appears, and `docs/glance.md` embeds it. Capturing it
      needed the KWin rule installed on the developer's own desktop, which was
      done with the file backed up first and removed again afterwards. (R4)
- [x] AC5 Both images have alt text in the file that embeds them, checked
      against what the image shows. (R5)
- [x] AC6 `TestTheMenuCatcherAddsNoHeightToTheWindow` fails on the old
      arrangement. `TestTheMenuIsBuiltFreshEveryTimeItIsOpened` and
      `TestAWindowWithNoMenuOpensNothing` cover the rest of the menu path. (R6)
- [x] AC7 `TestACardDrawsItsOwnSurface` asserts the card's fill and border come
      from palette tokens and that its radius is a surface radius rather than
      the scheme's control radius. (R6)
- [x] AC8 `glance/kwin` restores a file byte for byte across an install and a
      remove, and repeated installs do not grow it.
      `TestRewritingAFileWithoutChangingItIsByteIdentical`,
      `TestInstallingRepeatedlyDoesNotGrowTheFile`,
      `TestTheFileEndsWithExactlyOneNewline`. (R6)
- [x] AC9 `glance.Meter` draws a label, a monospace caption and a bar graded by
      status; the fraction is clamped, the bar reserves height and no width,
      and four statuses give four fills. Five tests in `glance/meter_test.go`,
      and `TestFillingAMeterDoesNotResizeTheWindow` in the example holds the
      width rule over 200 polls. (R7)
- [x] AC10 The meter appears in the gallery as `glance.Card + Meter`, in
      `examples/glance-monitor` as a Usage card, and in `docs/glance.md` under
      Meters — including that a reading with no limit is a row, because
      drawing it as a bar would invent a maximum. (R7)
- [x] AC11 `go test -race ./...` passes headless and `make lint` reports 0
      issues.

## Risks & Assumptions

- **A glance window still gets a titlebar on KDE Wayland.** `CreateSplashWindow`
  sets GLFW's `Decorated` hint to false, and KWin decorates it anyway. This is
  not a Fyne bug to work around in code: it is the reason `kwin.Rule.NoBorder`
  exists, and `docs/glance.md` claiming the rule "duplicates what a splash
  window already is" was wrong. The rule is required on this desktop, not
  belt-and-braces.
- **Capturing the frameless window needs that rule installed**, which changes
  the developer's own desktop configuration rather than the throwaway HOME the
  harness redirects: KWin reads the real user's config. It is reversible with
  `glance-monitor -kwin-remove`.
- **The gallery cannot show the archetype**, only its vocabulary. Anyone
  judging the window itself has to run `examples/glance-monitor`.
- **Rollback** is a revert. The gallery section and both images are additive;
  the five fixes change `glance` and `glance/kwin` behaviour and are each
  covered by a test.
- **The capture is not reproducible in CI** and is not meant to be. It needs a
  KDE session, the window rule, and a human to look at the result — which is
  the whole point of a screenshot, and why `make screenshots` has always been a
  command someone runs rather than a check.

## Alternatives Considered

- *Render the screenshots headlessly with `test.Canvas().Capture()`.*
  Deterministic and reproducible in CI, and rejected: the software painter
  draws the sparkline's near-horizontal segments as a dashed line, so the image
  would show an artifact the GL painter does not produce, and it would not be
  what the window looks like on a desktop.
- *Add a `--frameless` mode to the harness.* Deferred. Making `--section`
  conditional was enough to capture this window; a mode that knows how to find
  and crop undecorated always-on-top windows is worth writing when there is a
  second one.
- *Crop the titlebar off the capture.* Rejected: presenting a decorated window
  as frameless is a picture of something that did not happen.

## Executive Summary

Adds a Glance section to the gallery and the two images `docs/glance.md` and
the README were missing. Drawing the widgets for the first time exposed three
defects that prose review had not: values clipped at full pane width, a context
menu catcher that added 52 px of dead space and caught taps almost nowhere, and
cards with no surface of their own despite the rulebook specifying one. Each is
fixed with a test.

Five defects were found, not three: the two above plus square cards and a
`glance/kwin` that did not restore a file it had changed, both of which only
showed up against a real desktop and a real thirteen-rule `kwinrulesrc`.

Reviewers should look at `docs/img/glance-monitor.png` first — it is the
archetype, and it is also the evidence that `kwin.Rule.NoBorder` is required
rather than redundant on KDE Wayland.
