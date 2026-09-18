# Spec 008: Cancel on the busy popup

> **Note**: This work has no associated issue tracker ticket. The repository
> is a personal public project without an issue tracker.

## Status: COMPLETE

## Executive Summary

`shell.BusyCancellable(what, cancel)` is `Busy` with a Cancel button on the
busy popup, for a job that holds the indicator itself instead of running
through `PerformCancellable`. The popup is modal, so a Cancel left enabled in
a toolbar behind it is on screen and unclickable; nmsbonker's build had been
in that state since before it adopted the library. The button follows the
operation rather than the popup, dies when pressed, and `PerformCancellable`
now runs through the same path. Reviewers should read `shell/busy.go`:
`busy`, `showBusy` and `busyPopUp`. Released as v0.1.4.

## Context

The busy popup is modal, and spec 007 recorded why: the window runs one
operation at a time and every button that starts work is disabled while one
runs, so a popup that also swallows clicks takes nothing away. That reasoning
has a hole, and nmsbonker's build section is standing in it. A cancellable job
leaves one button live — Cancel — and the modal swallows clicks on that one
too. From 300 ms into a build until it ends, the button is on screen, enabled,
and unreachable.

Spec 007 left this as is and pointed cancellable jobs at
`shell.PerformCancellable`, whose Cancel lives on the popup. That answer does
not fit a job that is more than one call: nmsbonker's build runs a log pump
around `core.Build`, keeps its own cancellation bookkeeping (a `cancelled`
flag that decides the summary banner), and reports its own outcome rather than
`Perform`'s. It holds the indicator with `shell.Busy` and there is no way to
give `Busy` a cancel.

This is not a regression from the adoption: nmsbonker's own `showBusy` before
the port was the same modal with the same comment, and the button was just as
unreachable. It is a gap the second adopter found by looking, and it belongs
to the library now.

## Requirements

- R1 `shell.BusyCancellable(what string, cancel context.CancelFunc) func()`
  is `Busy` with a Cancel button on the popup. The button calls `cancel`.
- R2 The button follows the cancel, not the moment the popup was built. A
  popup already on screen without one gains it when a cancellable operation
  starts beneath it, and loses it when that operation ends.
- R3 Pressing Cancel disables the button and leaves the popup up: cancelling
  a build is not instant, and a live button is a button pressed twice. The
  caption stays the caller's.
- R4 The cancel is forgotten when the busy count reaches zero, so the next
  plain `Busy` shows no Cancel.
- R5 `PerformCancellable` runs through `BusyCancellable`; the shell has one
  place that decides what the popup holds.
- R6 The gallery's Shell section drives it, `docs/design-system.md` records
  the rule, and the changelog names the addition.

## Acceptance Criteria

- [x] AC1 `BusyCancellable` puts a button captioned Cancel on the popup, and
  pressing it calls the cancel that was handed in (R1). Found on the popup and
  tapped, not by reaching for the field.
- [x] AC2 A plain `Busy` holds the popup, a `BusyCancellable` starts beneath
  it and the button appears; when the cancellable one finishes and the plain
  one still holds, the button goes (R2).
- [x] AC3 Pressing Cancel disables the button; the popup is still up (R3).
- [x] AC4 After every operation ends, a plain `Busy` shows a popup with no
  Cancel (R4).
- [x] AC5 `PerformCancellable` still cancels through the context, and its
  button is the same one (R5).
- [x] AC6 Gallery card, design-system entry, changelog; tests, lint and vet
  clean (R6).

Where each is verified, in `shell/shell_test.go`:
`TestBusyCancellablePutsTheJobsOwnCancelOnThePopup` (AC1, AC3),
`TestTheCancelButtonFollowsTheCancellableOperationNotThePopup` (AC2, AC4),
`TestPerformCancellableCancelsThroughTheContext` (AC5, rewritten to tap the
button on the popup rather than reach for the field behind it). AC6: the
Shell section's card in `cmd/fynedesygn-gallery/sections.go`, "Progress and
results" in `docs/design-system.md`, 0.1.4 in the README changelog; `go test
./...`, `make lint` (0 issues) and `go vet ./...` clean, and
`govulncheck -mode binary` on the gallery reports no vulnerabilities.

## Risks & Assumptions

- **Assumption**: rebuilding the popup's content when the cancel appears or
  goes is cheap enough to do inline; it happens at most twice per operation.
- **Risk**: `busyCancel` is read and written on the UI thread only. `Busy`
  hops there itself, so `BusyCancellable` must set the cancel inside that hop
  rather than before it, as `perform` does today.
- **Rollback**: additive. `git revert`; `Busy` and `PerformCancellable` keep
  their signatures and `BusyCancellable` is new.
- Supersedes spec 007's "recorded and left as is" on the modal popup.

## Alternatives Considered

- Making the popup non-modal, so the toolbar's Cancel is clickable. Rejected:
  every other button behind it is disabled, and a non-modal popup invites
  clicks that do nothing. The reachable Cancel should be the one the user is
  already looking at.
- Reworking nmsbonker's `startBuild` onto `PerformCancellable` and leaving the
  library alone. Rejected: it fits one caller by bending the job around the
  runner, and the next program with a cancellable multi-call job meets the
  same wall.
