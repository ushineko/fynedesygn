# Spec 066: a second question after the job

**Issue**: [#193](https://github.com/ushineko/fynedesygn/issues/193)

## Status: COMPLETE

## Executive Summary

`ConfirmOptions.Then` takes a `ConfirmStep`: a second question asked in the
same window after the job succeeds. Its question and detail come from `Ask`,
which runs after the job, so they can show what the job found. Its button
labels are fields, so nothing moves. Declining it, or cancelling its job,
ends as Finished, because the first job's work is done. fynstall's
uninstaller uses it to offer removing the files a program left. Reviewers
should look first at `jobDone` and `askThen` in `wizard/confirm.go`.

## Context

Spec 062 added `wizard.RunConfirm`: a window that asks once, runs a job,
and reports. fynstall's uninstaller is its user. fynstall follows Windows
Installer: an uninstall removes only what the install put there. When the
program has made files of its own in the install directory, such as a
database or a cache, the uninstaller lists them and asks whether to remove
them too (fynstall spec 002, D5). Its command line does this with
`--remove-leftovers`.

The confirm window cannot ask that. Its result stage shows `Done`, a string
fixed when the window is built, so the job cannot say what it found. Its
only action is then Close. The question can only be asked after the
uninstall, because only the uninstall knows what is left.

So the window needs an optional second question, asked in the same window
after the job succeeds and built from what the job found. It keeps spec
062's rules: one layout that does not move, the job's own error decides,
and closing during a job asks and waits.

## Requirements

- R1 `ConfirmOptions.Then *ConfirmStep` is an optional second question. A
  nil `Then` leaves the window exactly as spec 062 describes.
- R2 `ConfirmStep`:
  - `Action`, the label of the button that runs the step's job;
    `Destructive` styles it as a destructive action;
  - `Decline`, the label of the button that leaves without it, such as
    "Keep them"; the default is Close;
  - `Ask func() (question, detail string, ok bool)`, called on the UI
    thread after the first job succeeds. It returns the bold question and
    its Markdown detail. `ok` false skips the step, and the window shows the
    plain result;
  - `Job func(context.Context) error`;
  - `Done`, the message line's text when the step's job succeeds.

  The labels are fields and not part of `Ask`, so the button widths are
  known when the window is built (R5).
- R3 After the first job succeeds and `Ask` returns `ok`, the window asks
  again:
  - the question and detail are replaced by the step's; the detail
    scrolls back to its top;
  - the message line keeps the first job's `Done`, in the good style, so
    the person sees what has already happened;
  - the action button reads `Action`, and the cancel button reads
    `Decline` and is enabled.
- R4 Outcomes. The first job has done its work by the time the step is
  asked, so:
  - `Decline`, or the title bar's close, while the step is asked ends with
    Finished;
  - the step's job succeeding shows `Done`, and Close ends with Finished;
  - the step's job failing shows its error, and Close ends with Failed and
    that error;
  - the step's job returning an error that wraps `context.Canceled` (after
    Cancel while it runs) ends with Finished, since the first job's work
    stands.
  While the step's job runs, Cancel and the title bar's close behave as in
  spec 062 R3.
- R5 Nothing moves. The cancel button is as wide as the wider of Cancel and
  `Decline`. The action button is as wide as the widest of the first
  action, Close and the step's `Action`. The detail region keeps its size;
  a long list scrolls inside it.
- R6 `SkipQuestion` skips only the first question. The step is still asked,
  because its answer is a separate decision that `--yes` did not make.
- R7 Documentation, example and changelog: the confirm section of
  `docs/wizard.md` describes the step. `examples/installer-wizard -confirm
  -leftovers` shows it, for the desk check. The README changelog entry is
  updated in the same commit.

## Acceptance Criteria

- [x] R1 A window without `Then` passes spec 062's tests unchanged.
- [x] R3 After the first job, the question, detail, buttons and message
      line show the step as R3 says (headless).
- [x] R4 Decline gives Finished and never runs the step's job; the step's
      job succeeding shows its Done and gives Finished; failing gives
      Failed with its error; cancelled while running asks, waits for it,
      and gives Finished (headless).
- [x] R2 `Ask` returning false shows the plain result, and the step's job
      never runs (headless).
- [x] R5 The button row has the same width in every stage, through the
      step (headless).
- [x] R6 `SkipQuestion` runs the first job at once and then asks the step
      (headless).
- [x] R7 At the desk: `examples/installer-wizard -confirm -leftovers` in
      Breeze Dark and Breeze Light, through the step with Decline and with
      the action, and with `-yes`.
- [x] R7 `docs/wizard.md`, the example and the changelog.

## Risks & Assumptions

- **One step, not a sequence.** A second question is what the uninstaller
  needs. A chain of questions would be the wizard; if a third one is ever
  wanted, that is the signal to use the wizard instead.
- **`Ask` runs on the UI thread.** It must not block. The first job
  computes what the question needs and `Ask` only formats it.
- **Rollback.** Additive: a new field and type. Reverting the commit
  removes them and changes nothing else.

## Alternatives Considered

- Considered letting the job set `Done` and the detail through a reporter,
  as the wizard's progress page does. Rejected because the second action
  would still need a place, and the result stage would change shape.
- Considered a dialog over the confirm window for the second question.
  Rejected because a dialog over a small fixed window covers the list it
  asks about.

## E2E Test Plan

- Environment: KDE Plasma 6 (Wayland), this machine.
- Steps: the R7 desk check, then fynstall's uninstaller with a file left by
  the program, from the launcher's Uninstall action (fynstall spec 002
  phase 3).
- Expected result: the uninstall result, then the list and "Remove them
  too", with no movement; Decline leaves the files, the action removes
  them.
- Coverage: R1–R7.

### Results (2026-10-09)

On CachyOS with KDE Plasma 6 on Wayland. `make test`, `make lint` (0
issues) and `govulncheck` (no vulnerabilities) pass.

- Headless tests: `TestTheStepIsAskedAfterTheJobWithWhatItFound`,
  `TestDecliningTheStepFinishesWithoutRunningIt`,
  `TestAFailingStepClosesAsFailed`, `TestACancelledStepAsksWaitsAndFinishes`,
  `TestAStepThatAsksNothingShowsThePlainResult`,
  `TestAFailingFirstJobNeverAsksTheStep` and
  `TestSkipQuestionStillAsksTheStep`; spec 062's tests pass unchanged. A
  mutation that made a cancelled step end as Cancelled failed
  `TestACancelledStepAsksWaitsAndFinishes`.
- The user ran `examples/installer-wizard -confirm -leftovers` through the
  step and reports that it works and the buttons do not move.
- fynstall's uninstaller, built against this branch, asked about a file
  the program had left (`TestTheUninstallWindowOffersToRemoveTheLeftovers`
  in fynstall, and its spec 002 phase 3 desk check).
- `docs/img/gallery-wizard.png` was refreshed for the new row.
