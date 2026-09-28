# 042 — A panel with its own theme

**Issue**: #110

## Context

A program with a glance panel and a settings window wants them to look
different: a panel is read from across a desk, a settings window at arm's
length. Until now this package's answer was that the **panel** takes the
application's theme and the **window** takes a `container.ThemeOverride`, on
the grounds that a card is a canvas object and an override does not reach one.
`Panel.SetTheme` said so in as many words, and `shell.Options.OwnAppearance`
was built for the other half of it.

That arrangement has a hole in it, and it is not small. A dialog, a
`widget.Select`'s dropdown and a context menu are added to the canvas's
**overlay** stack rather than to a window's content. An override wraps content.
So nothing overrides an overlay, and it draws in the application's theme —
which under this arrangement is deliberately the panel's.

Reported against hayami: opening the font chooser from the preferences window
drew the whole dialog in the panel's face and the panel's card opacity, so the
list was see-through and the sample was at the panel's size. The window's own
navigation showed it too.

There is nothing to patch around it. Fyne's overlay stack is not reachable from
a content-level override, and `Select`'s dropdown is Fyne's own widget. The
arrangement is wrong, so it is turned round.

## Requirements

### The panel carries its own theme

`Panel.SetTheme` hands the theme to every card; a card hands it to its rows and
to every `Piece` it holds; each of them resolves sizes and colours from it
rather than from the package-level `theme` helpers.

A nil theme is the application's, which is what every panel that has not asked
for anything else does and has always done.

The subtree override stays, because it is still the only thing that reaches a
standard Fyne widget a consumer has put inside a panel. The two are
complementary: the override for widgets, the handed-down theme for the canvas
objects this package draws.

### A card can be given something that has a face

`Card.AddObject` takes a `fyne.CanvasObject`, and an object cannot be asked for
its theme: a meter handed over as `meter.Object()` is a `*fyne.Container` by the
time the card sees it. That is why a meter and a grid of cells were the two
pieces left wearing the application's face.

`Card.Add` takes a `Piece` — anything with `Object()` and `SetTheme` — and
keeps it in step. `AddObject` stays for a consumer's own object, which is its
own to style.

### Being told is not enough

A `canvas.Text` holds a literal size and colour rather than asking the theme
when it draws, so every `SetTheme` here restyles.

### The variant stays the application's

A panel and a window may differ in face and size. They may not differ about
whether it is night: a program whose two windows disagreed on that would be a
bug rather than a setting. The design system's own themes ignore the argument
and let the palette decide, so this costs nothing and prevents a nonsense.

## Acceptance criteria

- [x] `Panel.SetTheme` gives every card a theme, and a card added afterwards
      takes it too.
- [x] A card hands it to its rows, its meters and its cells.
- [x] `Card.Add` exists, takes a `Piece`, and keeps it in step.
- [x] A panel with a theme of its own does not change the application's.
- [x] A nil theme returns the panel to the application's.
- [x] Nothing in `glance` reads the package-level `theme` helpers for drawing,
      except the exported margin functions, which are documented as the
      application's face.
- [x] `widgets.StatusColorName` exists, and `StatusColor` is written in terms
      of it, so the mapping lives in one place.
- [x] The gallery uses `Card.Add` for its meter and its cells.
- [x] `docs/fyne-quirks.md` has the overlay quirk, numbered and in order,
      naming its canary.
- [x] `docs/glance.md` says which window takes which theme, and why round that
      way.
- [x] `README.md` carries the changelog entry under `### Unreleased`.
- [x] The canary was falsified.
- [x] `make test` passes.

## Risks & Assumptions

- **This reverses documented advice.** `Panel.SetTheme` told consumers to do
  the opposite. Its comment is replaced rather than softened, and says what
  changed and why, so the next reader does not reverse it back.
- **Additive for a consumer that does nothing.** A panel that never calls
  `SetTheme` follows the application exactly as before, and `AddObject` is
  unchanged. `Card.Add` is new.
- **Sixty call sites moved from a package helper to a method.** Mechanical, and
  the tests assert on what reached the canvas objects rather than on what the
  theme was asked for, which is the difference that matters.
- **Rollback**: revert the commit.

## Status: COMPLETE
