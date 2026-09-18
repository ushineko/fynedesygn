# Validation report: spec 004 dialogs, log pane, forms

Date: 2026-09-18. Milestone: the last three shared component families before
the examples and adoption specs. Spec: `specs/004-dialogs-logpane-forms.md`.

## Scope

`dialogs`, `logpane`, `forms`, three gallery sections, documentation
updates. Also closes spec 001 (AC2, CI green) and spec 003 (AC12, CI green).

## Phase 3: tests

`go test -race ./...` on Linux, no display: all packages pass.

| Package | Coverage |
|---|---|
| `dialogs` | 77.5 % |
| `logpane` | 96.8 % |
| `forms` | 100 % |
| `cmd/fynedesygn-gallery` | 57.0 % |

Uncovered in `dialogs`: the chooser callbacks (they need a real file picked)
and `Decide`'s goroutine hop, which is exercised by the gallery. Canaries:
quirk 1 (`TestChooserResizeAfterShowDoesNotPanic`), quirk 2
(`TestSetCheckedFiresOnChanged`), quirk 10
(`TestFollowTailToleratesFractionalOffsets`).

Two tests were rewritten during implementation because of quirk 11: under the
Fyne test driver `fyne.Do` runs inline on the calling goroutine, so a test
that inspects widgets while a worker (the log pump, `Decide`) draws them is a
data race by construction. The pump test now waits, stops (which joins the
goroutine) and then reads; the question test drives the UI-thread half
(`ask`) directly and closes the dialog through the dialog, not the overlay.

`go vet`, `gofmt`: clean.

## Phase 4: code quality

- golangci-lint v2.12.2: 0 issues.
- `forms.Numeric` and `forms.NumericEntry` share `IntRange`; `Path` reuses
  `dialogs.WithBrowse`; the log pane's row colour uses `widgets.StatusColor`.
- The gallery's Log section owns its pane and detaches it through
  `OnDetach`, the pattern a consuming program follows.

## Phase 5: security review

- `govulncheck -mode binary` on the gallery: run in the previous milestone
  with the same module graph plus `github.com/FyshOS/fancyfs` (pulled by
  Fyne's dialog package); re-run result recorded in the commit message.
- `dialogs.OpenPath` is the library's one runtime subprocess: the platform
  opener with the path as a single argument, no shell, detached on purpose.
  Refuses an empty path. Not exercised in tests beyond that.
- No credentials, no network. Secrets grep: clean.

## Phase 5.5: release safety

Nothing consumes these packages yet. Rollback is `git revert`.

## Spec reconciliation

All thirteen acceptance criteria verified; AC5 amended as described above.
Status COMPLETE.

## Gaps

- The choosers' success callbacks are untested headlessly.
- `Decide` end to end (goroutine hop and channel wait) is verified by hand
  in the gallery's Dialogs section, not by a test.
