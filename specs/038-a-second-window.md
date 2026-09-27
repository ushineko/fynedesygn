# Spec 038: a second window

**Issue**: [#80](https://github.com/ushineko/fynedesygn/issues/80)

## Status: COMPLETE

## Executive Summary

`shell.NewIn(app, options)` builds a shell window over an app the caller
already has. `New` creates the app and is therefore the wrong way in for a
program that already has one, which Fyne allows exactly one of. Reviewers
should look at the doc comment on `NewIn`, which is where the boundary between
the two is drawn.

## Context

`shell.New` calls `app.NewWithID`. That makes it the thing that creates the
application, and a program that already has one cannot use it: Fyne allows a
single app per process. `Headless` builds over an existing app and makes no
window, because it exists for tests.

The case that found this is hayami. Its panel is a glance window and its
preferences are a shell window in the same process; they share a settings store
and a change made in one has to reach the other as it is made. Two binaries
would mean a file watcher and a race for the last write.

Two archetypes in one process is not a new idea here — `docs/glance.md` already
describes `Options.Secondary` for a glance window belonging to a program with a
main window of its own. What was missing was the other direction.

## Requirements

- R1 `NewIn(a fyne.App, o Options) *Shell` does everything `New` does except
  create the app.
- R2 `New` is `NewIn` over an app it creates, so there is one construction
  path and not two that drift.
- R3 The cursor theme stays with `New`. It has to be applied before the
  toolkit starts, which is the app-creating caller's moment and not this one.

## Acceptance Criteria

- [x] AC1 `NewIn` over a test app builds a window, runs `OnStart` and reports its settings error, which the existing `New` tests cover by construction. (R1)
- [x] AC2 `New` is three lines and delegates; nothing in it is duplicated. (R2)
- [x] AC3 `ApplyCursorTheme` is called by `New` and not by `NewIn`. (R3)

## Risks & Assumptions

- **Two shells in one process would fight over the app's theme.** `NewIn` sets
  it from its own appearance, as `New` does. That is not a shape this
  supports, and the doc comment says so: a program wanting two should ask why
  the second is not a section of the first.
- A glance window and a shell window do not conflict. The glance window draws
  from the same theme and follows it.
- Rollback: revert. `New` keeps its behaviour and nothing else calls `NewIn`
  yet.

## Alternatives Considered

- Exporting a variant of `Headless` that takes a window. Rejected: `Headless`
  is for tests and its contract says `OnScreen` reports false; giving it a
  window would make it two things.
