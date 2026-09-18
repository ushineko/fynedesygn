# Spec 007: Second adopter feedback

> **Note**: This work has no associated issue tracker ticket. The repository
> is a personal public project without an issue tracker.

## Status: COMPLETE

## Executive Summary

Five changes nmsbonker's adoption (its spec 012) recorded as gaps:
`steps.Advance` never reopens a finished step and `steps.NewSteps` carries
standing notes that `Reset` restores; `logpane.Pane.SetFollowing` re-arms
the tail for a new job; `forms.SliderEntry` hands `Commit` the value as
typed so a program can clamp and say so; `shell.Load` is the loader shape
beside `Perform`; `shell.About.URLText` captions the link. Released as
v0.1.3.

## Context

The build section of nmsbonker is the most involved job view among the
three programs: a step list with standing notes, a log that must follow
the tail on every run, several loads running beside an operation, and a
parameter slider whose out-of-range values core clamps and reports. Each
gap below is that section meeting a library shape written from the simpler
job in clockwork-orange.

## Requirements

- R1 `steps.Advance(i, note)` leaves a step that is Done, Failed or
  Cancelled alone (earlier pending or running steps still become Done).
- R2 `steps.NewSteps(steps ...Step)` creates a list whose notes are standing
  notes; `Reset` restores them. `New(names...)` is unchanged.
- R3 `logpane.Pane.SetFollowing(bool)` sets the state, forgets the remembered
  offset when turning on, and redraws so the Follow box agrees.
- R4 `forms.SliderEntry` shows a typed value clamped but passes it to
  `Commit` as typed; a program calls `Set` with what it stored.
- R5 `shell.Load(what, fn)`: inline when off screen, else a goroutine with
  `Busy`, never refused while `Working()`; errors to `Report` on the UI
  thread.
- R6 `shell.About.URLText`.

Recorded and left as is: the shell's busy popup is modal, so a toolbar
Cancel is reachable only before the popup appears; a job that must be
cancellable mid-run uses `PerformCancellable`, whose Cancel lives on the
popup. (Superseded by spec 008: a job that holds the indicator itself uses
`BusyCancellable` and gets the same Cancel.)

## Acceptance Criteria

- [x] AC1 A late `Advance` for a Done step leaves it Done; `Reset` restores
  the standing notes (R1, R2).
- [x] AC2 `SetFollowing(true)` after the box was unticked follows again and
  the box shows it (R3).
- [x] AC3 Typing 500 into a 0..100 pair commits 500 and shows 100 (R4).
- [x] AC4 `Load` runs while `Busy` is held and reports an error as a banner
  (R5); `URLText` shows (R6).
- [x] AC5 Tests, lint and vet clean; changelog entry.

## Risks & Assumptions

- R4 changes what `Commit` receives for out-of-range typed values. The one
  caller (the gallery's Forms demo) stores what it is given; a program that
  relied on the clamp gets the raw value now. v0 API; noted in the changelog.
- Rollback: `git revert`.
