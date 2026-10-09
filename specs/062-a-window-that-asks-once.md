# Spec 062: a window that asks once

**Issue**: [#183](https://github.com/ushineko/fynedesygn/issues/183)

## Status: COMPLETE

## Executive Summary

`wizard.RunConfirm` adds the smaller shape beside the wizard: a window that
asks one question, runs a job, and reports, in three stages of one layout
that does not change. It shares the wizard's app set-up (now `newApp`) and
its job rules: the job's own error decides the outcome, and closing during
the job asks and waits. `SkipQuestion` serves a caller's `--yes`. fynstall's
uninstaller is the first user. Reviewers should look first at
`wizard/confirm.go`: `run`, `jobDone` and `Cancel`.

## Context

Spec 061 added the wizard: pages in order, with a page list, for a task that
asks several things. fynstall's uninstaller asks one thing. "Uninstall Hello
0.1.0?" is the whole decision, and after it there is a short job and a
result. Shown as a wizard, it had three pages and a page list for one
question, and the user asked for one question in a small window.

That shape (ask once, run, report) is not the wizard and not a dialog. A
dialog needs a window to sit over, and an uninstaller started from the
launcher has none. It is a small window of its own, and it shares the
wizard's rules: the job's own error decides the outcome, closing during the
job asks and then waits, and nothing moves while it runs. So it goes in the
`wizard` package and uses the same window set-up.

## Requirements

- R1 `wizard.RunConfirm(o ConfirmOptions) Result` shows a fixed-size window
  and blocks until it closes. `NewConfirm`, `NewConfirmIn` (over an existing
  app, leaving its theme alone) and `HeadlessConfirm` (for tests) match the
  wizard's constructors.
- R2 `ConfirmOptions`:
  - `AppID`, `Name`, `Icon`, `Size` and `Appearance`, as for the wizard;
  - `Question`, the bold first line;
  - `Detail`, Markdown under it: what will happen and what is left alone;
  - `Action`, the label of the button that runs the job; `Destructive`
    styles it as a destructive action;
  - `Job`, a `func(context.Context) error`;
  - `Done`, the message line's text on success;
  - `SkipQuestion`, which runs the job as soon as the window shows, for a
    caller's `--yes`.
- R3 The window has three stages, in one layout that does not change:
  - **Asking**: Cancel and the action button are enabled. Cancel, or the
    title bar's close, ends with the outcome Cancelled.
  - **Running**: the action button is disabled and an indeterminate bar
    runs in a region of fixed height. Cancel and the title bar's close ask
    for confirmation, cancel the job's context, and close when the job
    returns.
  - **Result**: the message line shows `Done` (good) or the error (bad).
    The action button becomes Close and Cancel is disabled. Closing ends
    with Finished or Failed.
- R4 The job's own error decides the outcome, as in R4 of spec 061. An error
  that wraps `context.Canceled` is Cancelled, any other error is Failed,
  and nil is Finished, even if Cancel arrived as the job ended.
- R5 Nothing moves. The action button is as wide as the wider of `Action`
  and Close, the bar's region keeps its height when the bar is hidden, and
  the message line is always there.
- R6 Documentation, gallery and example. `docs/wizard.md` gets a section
  for this window, with a screenshot. The gallery's Wizard section gets a
  button that opens one. `examples/installer-wizard -confirm` shows one,
  for the screenshot harness and the desk check. The README package row
  and the changelog are updated in the same commit.

## Acceptance Criteria

- [x] R3 Asking, then the action, then Done: the message line shows Done,
      the action button reads Close, Cancel is disabled, and Close gives
      Finished (headless).
- [x] R3 Cancel while asking gives Cancelled, and the job never runs
      (headless).
- [x] R3/R4 A failing job shows its error and gives Failed on Close. A job
      cancelled while running asks once, receives `ctx.Done()`, and the
      window closes only when it returns, with Cancelled. A job that
      finishes as Cancel arrives gives Finished (headless).
- [x] R2 `SkipQuestion` runs the job at once and shows only the result
      (headless).
- [x] R5 The button row has the same width in all three stages (headless).
- [x] R1 `NewConfirmIn` lays out a real window under the test driver.
- [x] R6 At the desk: `examples/installer-wizard -confirm` in Breeze Dark
      and Breeze Light, through ask, run and result, and with `-fail`.
- [x] R6 `docs/wizard.md` section and screenshot, gallery button, README
      row and changelog.

## Risks & Assumptions

- **Two shapes in one package.** The confirm window shares the wizard's
  window and job code. If a third shape appears, the shared part should
  move to an internal package rather than the `wizard` package growing.
- **Rollback.** Additive: a new function and type. Reverting the commit
  removes them and changes nothing else.

## E2E Test Plan

- Environment: KDE Plasma 6 (Wayland), this machine.
- Steps: the R6 desk check, then fynstall's uninstaller in its spec 001
  phase 4 desk check, from the launcher's Uninstall action and from a
  double-click.
- Expected result: one question, then the result, with no movement of the
  buttons; `--yes` shows only the result.
- Coverage: R1–R6.

### Results (2026-10-08)

On CachyOS with KDE Plasma 6 on Wayland. `make test`, `make lint` and
`govulncheck` pass.

- Headless tests: `TestAskThenRunThenReport`,
  `TestCancelWhileAskingRunsNothing`,
  `TestAFailingJobShowsItsErrorAndClosesAsFailed`,
  `TestCancelWhileRunningAsksAndWaitsForTheJob`,
  `TestAJobThatFinishesAsCancelArrivesStillFinishes`,
  `TestSkipQuestionShowsOnlyTheResult` and
  `TestNewConfirmInLaysOutARealWindow`. A mutation that let the context
  decide the outcome failed the finished-as-cancel test.
- `examples/installer-wizard -confirm` started and stayed up in each mode
  before the desk check. The user then ran it in Breeze Dark and Breeze
  Light, with `-fail` and with `-yes`, and reports that it works: one
  question, the result, no movement.
- `docs/img/confirm-window.png` was captured with `tools/screenshot.sh`
  through a wrapper that adds `-confirm`, and `docs/img/gallery-wizard.png`
  was refreshed for the new button.
