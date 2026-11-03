# 077 — a theme override applies its theme at every refresh

**Issue**: #228

## Status: COMPLETE

## Executive Summary

`widgets.ThemeOverride` applies its theme again at every `Refresh`, as Fyne's
own override does. Spec 076 made it apply the theme only when told to, which
stopped a font leak and lost the theme instead: Fyne's cache takes a widget
out of its override when it expires the widget's renderer. The leak is now
avoided where it was measured, by keeping overrides out of anything refreshed
often. Reviewers should look at `Refresh` in `widgets/override.go` and at the
quirk 51 row.

## Context

Spec 076 (0.1.111) gave `widgets.ThemeOverride` a `Refresh` that redraws the
content without applying the theme again. Each application is a theme scope
whose fonts Fyne keeps for the life of the program (quirk 51), so the type
stopped leaking at every refresh.

Fyne's cache destroys, every 30 seconds, the renderers of widgets that have not
been drawn for a minute. Since fyne-io/fyne#5002 it also deletes those widgets'
entries from the override map (`destroyExpiredRenderers`). Fyne's own override
hides this: the next refresh applies the theme again and puts them back. An
override that does not apply it again never does.

jira-viewer hit it within one session: with its ticket table in Carlito
through a kept override, a few rows showed in the interface font, larger and
cut short. `shell` (own appearance) and `glance.Panel.SetTheme` use the same
type and are open to the same thing.

## Decisions

- **D1 — `Refresh` applies the theme again.** Correctness comes first. The
  type behaves as Fyne's override does, with the theme and content still
  settable and the theme readable (`OwnTheme`).
- **D2 — The leak is avoided where overrides are built, not inside the
  type.** An override belongs once per window, section or dialog, never per
  row, cell or link, and never around a table or list a timer redraws. Text
  that needs a face of its own carries it in `canvas.Text.FontSource`, which
  needs no override. Spec 076's link change already follows this, and
  jira-viewer's table is changed to follow it (jira-viewer spec 025).
- **D3 — The API stays.** `Reapply` remains, and now does what `Refresh`
  does. Removing it would break consumers for no gain.

## Requirements

- R1 — `widgets.ThemeOverride.Refresh` applies the theme to the content again.
- R2 — Quirk 51 and `docs/performance.md` say where an override belongs and
  why not applying the theme again is no cure.
- R3 — The changelog names the 0.1.111 regression.

## Acceptance Criteria

- [x] AC1 — `TestAnObjectAddedLaterIsCoveredByARefresh` passes, and fails
      with 0.1.111's `Refresh` (R1).
- [x] AC2 — `shell` and `glance` tests pass, and the quirk 51 canary still
      holds on the pin (R1).
- [x] AC3 — Quirk 51, `docs/performance.md` and the changelog are updated
      (R2, R3).
- [x] AC4 — `make test` (race) and `make lint` pass.

## Risks & Assumptions

- **An `OwnAppearance` shell and a themed glance panel leak a scope at each
  refresh again**, about 138 KB per text style on the pin, as they did before
  0.1.111. They are refreshed on section changes and card additions, not on
  timers. fyne-io/fyne#6642 would remove it.
- **Rollback**: revert.

## E2E Test Plan

1. `make test`, `make lint`. Covers AC1 to AC4.
2. jira-viewer's table, which no longer uses an override, is checked on a real
   window by jira-viewer spec 025.
