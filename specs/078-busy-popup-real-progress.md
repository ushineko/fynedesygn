# 078 — the busy popup can show a real value

**Issue**: #230

## Status: COMPLETE

## Executive Summary

`Shell.BusyProgress(fraction, caption)` gives the busy popup's bar a real
value, for a job that measures its progress. The bar still has no value unless
a program asks for one, and nothing changes for a caller that does not. The
need came from clockwork-orange (#60), whose model download is minutes long
and has no log pane to host a step list. Reviewers should look at
`BusyProgress` and `showBusyProgress` in `shell/busy.go`, and at the one
sentence in `docs/design-system.md` that used to say "by design".

## Context

`docs/design-system.md` says the busy bar has no value by design, because a
bar that filled steadily would invent a number, and sends a program with real
steps to a step list beside a log pane. That holds for a scan or a round trip.
It leaves out a third case: a job that measures. clockwork-orange pulls
multi-gigabyte model files with an exact byte count, has no log pane in that
section, and showed a spinner for the whole download. A bar there is not an
invention; it is the number.

## Decisions

- **D1 — Opt-in.** The bar has no end until a program calls `BusyProgress`;
  `Busy`, `BusyCancellable`, `Perform` and `PerformCancellable` do not change.
- **D2 — A method on `Shell`, not a new popup kind.** The job already holds the
  popup through one of those four calls. A second way to start an operation
  would repeat the count, the Cancel button and the gate.
- **D3 — The operation's name stays.** `BusyProgress` replaces the popup's
  caption, not `BusyWhat`: a refusal that quotes "Copying 41 of 64 MiB" names a
  moment, not the job.
- **D4 — A new operation starts clean, and so does the next one after this
  ends.** Progress describes one operation and is not carried to another.
- **D5 — No estimates.** The doc says so. The type cannot check it.

## Requirements

- R1 — `Shell.BusyProgress(fraction, caption)`: fraction in [0, 1] fills the
  bar; above one counts as one; below zero or NaN returns the bar with no end;
  an empty caption keeps the text.
- R2 — Safe from any goroutine and at any rate; a call with nothing busy does
  nothing and is not remembered.
- R3 — A value reported before the popup appears is what it opens with.
- R4 — `BusyWhat` and `BusyReason` are unchanged by it.
- R5 — A new operation, and the end of the last one, discard reported progress.
- R6 — `docs/design-system.md` and the changelog say when to use it.
- R7 — An example shows it.

## Acceptance Criteria

- [x] AC1 — The popup swaps between the bar with a value and the bar with no
      end, shows the caption, clamps, and keeps `BusyWhat` (R1, R4):
      `TestBusyProgressFillsTheBarAndKeepsTheOperationsName`,
      `TestBusyProgressClampsAndCanGiveTheBarItsEndBack`.
- [x] AC2 — Progress reported before the popup appears is shown when it does
      (R3): `TestBusyProgressBeforeThePopupAppearsIsWhatItOpensWith`.
- [x] AC3 — Progress with nothing busy is dropped; a new operation and the end
      of the last one reset it (R2, R5):
      `TestBusyProgressWithNothingBusyDoesNothingAndIsNotRemembered`,
      `TestANewOperationDoesNotInheritTheLastOnesProgress`,
      `TestBusyProgressIsForgottenWhenTheOperationEnds`.
- [x] AC4 — `examples/transfer` runs to the end headlessly and a cancelled
      context stops it with the error wrapped (R7).
- [x] AC5 — `make test` (race), `make lint` and `make check-diagrams` pass.
- [x] AC6 — A rendered window shows the filled bar over the example, in both
      colour schemes (CONTRIBUTING, e2e).

## Risks & Assumptions

- **Additive.** One method and unexported fields; no existing call changes.
  Rollback is removing the method.
- **A popup does not grow with its caption.** It is sized when it opens; a
  caption much longer than the bar is the caller's to keep short, and the doc
  says so.
- **Last call wins with two operations.** The popup describes whichever
  reported last; documented.
- **The direction question.** The design doc's "by design" is the project's
  stated position, and CONTRIBUTING sends interaction changes to a Discussion.
  This change keeps the default and adds an opt-in for the case the position did
  not cover; the maintainer asked for it.

## E2E Test Plan

Rendered window on KDE Plasma 6 under Wayland, `examples/transfer`, started with
`tools/screenshot.sh` so the capture is the example's window and not the active
one. Results are recorded below.

## Technical Notes

- State lives under `busyMu`; the UI-thread widget updates happen inside
  `fyne.Do`, as `busy` does.
- The bar sits in a `container.NewStack` so it can be swapped under a popup
  that is already up without rebuilding the modal.

## Findings

Rendered reading, 2026-11-12: KDE Plasma 6 on Wayland (KWin), Fyne 2.8.1,
`examples/transfer` started by `tools/screenshot.sh`, which finds the window by
the process it started and grabs only that window. The copy starts by itself
under `TRANSFER_AUTOSTART=1` at 150 ms per megabyte so the popup is up long
enough to capture.

- Dark, Breeze Dark: [docs/img/transfer-busy-progress-dark.png](../docs/img/transfer-busy-progress-dark.png).
  The popup reads "Copying 9 of 64 MiB", the bar is filled to 14% with the
  percentage drawn on it, and Cancel is below. The section behind it is dimmed
  as for any busy popup.
- Light, Breeze Light: [docs/img/transfer-busy-progress-light.png](../docs/img/transfer-busy-progress-light.png).
  "Copying 6 of 64 MiB", 9%, Cancel.
- The popup is the size of the one with no end: swapping the bar did not resize
  or move it.
- In the light scheme Fyne's stock `ProgressBar` draws its percentage in white
  on the pale track, which is low contrast. It is Fyne's own text colour for
  the bar, not something this change sets, so it is recorded here and left.
- Not re-rendered: the gallery and the other examples. Nothing they draw
  changed; a program that does not call `BusyProgress` gets the same popup,
  covered by the existing busy tests. Consumers were not rendered: the
  clockwork-orange change that uses it follows this one.
