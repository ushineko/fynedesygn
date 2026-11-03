# 076 — a theme override that does not leak

**Issue**: #225

## Status: COMPLETE

## Executive Summary

`widgets.ThemeOverride` applies its theme when it is built and when it is
told to, never on `Refresh`. Fyne's own override applies its theme at every
refresh, and each application leaks that scope's fonts for the life of the
program. `shell` and `glance` use the new type, and a Markdown link no
longer uses an override at all. Fyne is pinned to `develop`
at `b655332`, where a scope that is still made costs about 138 KB per style
rather than 6.3 MB. Reviewers should look first at `widgets/override.go`, then at
`bareLink` in `markdown/links.go`, then at the quirk 51 row.

## Context

jira-viewer's desktop shell held 3.3 GB of Go heap after six days, for 66
tickets, and spent 13.5 cores collecting it (jira-viewer spec 025). The heap
profile put the growth under `painter.CachedFontFace` → `loadMeasureFont`.

Fyne's `container.ThemeOverride` applies its theme in `NewThemeOverride`, in
`CreateRenderer` and at every `Refresh`. Each application takes a new scope id,
and `painter.CachedFontFace` keys its font cache by style and scope. A new
scope therefore builds new faces for the theme font and every fallback, and
keeps them until `ClearFontCache` runs on an application theme change.
`fyne.Container.Refresh` refreshes every child, so refreshing anything around
an override applies its theme again. Upstream:
[fyne-io/fyne#6640](https://github.com/fyne-io/fyne/issues/6640), with a
fix proposed in [fyne-io/fyne#6642](https://github.com/fyne-io/fyne/pull/6642). Both are
open.

Measured on 2026-11-03 by refreshing a `ThemeOverride` around text in four
styles and drawing it through the software canvas:

| Fyne | Growth per refresh |
|---|---|
| v2.8.1, this module's pin | about 6.3 MB per style |
| `develop` @ `b655332` | about 550 KB (about 138 KB per style) |
| `develop` @ `b655332` + fyne-io/fyne#6642 | 0 |

`develop` carries [fyne-io/fyne#6555](https://github.com/fyne-io/fyne/pull/6555), which
shares a parsed font between faces. A new scope then costs new faces, but no
longer a new parse.

This module makes overrides in three places:

- `shell` for a window that owns its appearance, applied again after each
  section is installed (quirk 37).
- `glance.Panel.SetTheme`, applied again for each card added.
- `markdown/links.go`, once per link on each re-render.

## Decisions

- **D1 — Pin Fyne `develop` at `b655332`, with no fork.** It is a twelvefold
  reduction and it needs no `replace`, so every consumer gets it through
  ordinary version selection by requiring this module's release. The
  operator chose this over carrying fyne-io/fyne#6642 in a fork. The pin moves to the
  first tagged release that contains fyne-io/fyne#6642.
- **D2 — `widgets.ThemeOverride` applies its theme only when told to.** It
  applies it when it is built, on `SetContent`, on `SetTheme` and on
  `Reapply`. `Refresh` refreshes the content only. This closes the path
  where a refresh makes a scope, which is the one that grows without bound.
- **D3 — The caller says when the tree has changed.** Quirk 37 is why Fyne
  applies its theme at every refresh: a subtree added later is not covered.
  `shell` and `glance` already know the moment they add a subtree, and they
  call `Reapply` there. Lists, trees and text grids give the rows they
  create their parent's scope themselves.
- **D4 — A Markdown link uses no override at all.** `markdown/links.go`
  made one per link widget, and a RichText makes new link widgets at each
  refresh inside a list (quirk 47), as does every block the pane renders and
  every document built. `widgets.ThemeOverride` cannot help there, because a
  new widget needs a new override. The first version of this spec left it as
  a known remainder. Measured in jira-viewer, it was the largest source: about
  6 MB per ticket opened, 42.7 of 58 MB in one session. A link is now drawn
  by a `bareLink`, a widget whose box is its text's box, with Fyne's Hyperlink
  hidden beside it so the RichText's updates still land. It gives the click,
  the hover underline and the pointing hand that the Hyperlink gave. Unlike
  the Hyperlink, it takes no keyboard focus.
- **D5 — Quirk 6 is fixed on `develop`, and its row and canary are retired.**
  `markdown.CodePanel` stays, because it wraps code rather than letting it
  run past the pane, and that is a design rule in its own right.

## Requirements

- R1 — `go.mod` requires `fyne.io/fyne/v2 v2.8.3-0.20261103091334-b655332b73c2`
  in a commit of its own, with every canary run (library rule).
- R2 — `widgets.ThemeOverride` with `NewThemeOverride`, `Content`,
  `SetContent`, `SetTheme`, `Reapply` and a `Refresh` that does not apply
  the theme.
- R3 — `shell` and `glance` use it. No `container.NewThemeOverride` remains
  outside `widgets/override.go`.
- R6 — `markdown` draws a link with no theme override, and every part of a
  listed link still takes its click (quirk 47).
- R4 — Quirk 51 in `docs/fyne-quirks.md`, with a canary that fails once Fyne
  stops keeping fonts per scope.
- R5 — Changelog, the Fyne pin in the README and in `.claude/CLAUDE.md`, and
  `docs/performance.md`.

## Acceptance Criteria

- [x] AC1 — `make test` (race) and `make lint` pass on the `develop` pin.
      The only canary that failed after the bump is quirk 6's, which is
      retired (R1, D5).
- [x] AC2 — `TestARefreshedThemeOverrideKeepsNoFontsBehind` and
      `TestARefreshReachingTheOverrideThroughItsContainerKeepsNoFontsBehind`
      pass, and were falsified. With `Refresh` applying the theme the way
      Fyne's override does, both fail at about 55 MB per 100 refreshes (R2).
- [x] AC3 — `TestAnObjectAddedLaterIsCoveredOnceReapplied`,
      `TestSetContentDrawsTheNewContentInTheTheme` and
      `TestSetThemeRedrawsTheContentInTheNewTheme` pass (R2).
- [x] AC4 — `shell.TestASectionReachedLaterStillDrawsInTheWindowsOwnTheme`
      and `glance.TestAPanelWithItsOwnThemeDrawsItsCardsInIt` still pass
      with `shell` and `glance` on the new override (R3).
- [x] AC5 — `widgets.TestFyneStillKeepsFontsForEveryThemeScope` passes on
      the pin (R4).
- [x] AC6 — Seen on a real window: a throwaway own-appearance shell on
      Xvfb, with the application at 8 px text and the window at 22 px. The
      section reached by a real click on its row draws at 22 px, before and
      after switching back and forth. No example themes a glance panel, so
      that half rests on AC4's `glance` test rather than on a window.
- [x] AC7 — jira-viewer spec 025 builds and passes against this branch,
      through a `replace` in a scratch worktree only.

- [x] AC8 — `markdown.TestDrawingLinksAgainKeepsNoFontsBehind` passes, and
      was falsified: with a per-link override put back, fifty rounds of five
      links keep 33.7 MB. `TestEveryPartOfAListedLinkTakesTheClick` passes, and
      fails with Fyne's Hyperlink left visible (R6).
- [x] AC9 — Seen on a real window: this README in a markdown section on
      Xvfb. Its links draw in the link colour, inline and in the contents
      list. With the pointer beside a listed link, nothing is underlined; on
      it, that link underlines. A real click at 80% of the height of
      "Screenshots" scrolls the pane to that heading (R6).

## Risks & Assumptions

- **`develop` is unreleased**, 478 commits past v2.8.2. Every consumer
  inherits the pin through version selection once it takes this release.
  Consumers must not be released on it without their own real-window check.
  jira-viewer's is AC6 of its spec 025.
- **A theme's font resource must be comparable on `develop`.** Parsed fonts
  are keyed by `fyne.Resource`, and per fyne-io/fyne#6642 a non-comparable concrete type
  panics. `theme.Theme` returns cached `*fyne.StaticResource` values or
  Fyne's own bundled fonts, all of which are pointers.
- **A caller who adds content and does not call `Reapply` sees it drawn in
  the application theme.** That is quirk 37, and it is the same with Fyne's
  override until the next refresh. The doc comment says so.
- **A link no longer takes keyboard focus.** Fyne's Hyperlink did. Nothing
  in a consumer relied on tabbing to a link in a document.
- **Rollback**: revert. `go.mod` returns to v2.8.1, and `shell` and `glance`
  return to Fyne's override.

## E2E Test Plan

1. `make test`, `make lint`. Covers AC1 to AC5.
2. Run the gallery with an own-appearance shell, switch sections, and
   screenshot. Covers AC6.
3. Build jira-viewer against this branch and run its suites. Covers AC7.
