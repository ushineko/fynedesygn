# Spec 029: a dialog can be dismissed from its corner

**Issue**: TBD — to be filed before the PR.

## Status: COMPLETE

## Executive Summary

Every dialog this package raises now carries a close X in its upper right, in
addition to whatever buttons it was given. Fyne's dialog draws a title and its
buttons and nothing else, so until now a dialog could only be left by finding
the right button among the others — fine for a confirmation, where choosing is
the point, and wrong for a long read-only answer somebody opened to look at.
Reviewers should read `dialogs/dialogs.go`: `closable`, and the four helpers
that wrap their bodies in it.

## Context

Reported from jira-viewer, which shows ticket detail through
`dialogs.ShowDetail`: a document several screens long whose only way out is a
`Close` button below the fold of the action bar, in a row of six other
buttons. The reporter's words: "modal popups should also have an 'X' in the
upper right corner to close, in addition to the regular 'close' button."

Every window on every desktop closes from its corner. A dialog that does not
is one people hunt around in, and the hunt is worst exactly where the dialog
is largest.

Fyne 2.8.1's `dialog` package has no close affordance and no way to add one:
`dialog.NewCustom` and friends build the title bar themselves and expose
nothing of it. `base.go` has `Dismiss`, `SetOnClosed` and the buttons, and no
title-bar hook. So the X cannot go beside the title; it goes at the top right
of the *content*, which is the same corner a reader reaches for.

## Requirements

- R1. Every dialog in this package can be closed from its upper right corner.
- R2. The X is a dismissal and never an answer: it must not run a
  confirmation's action, confirm a prompt, or answer a blocking question yes.
- R3. A blocking question (`Decide`) answers exactly once however it is
  closed, the X included, or the worker waiting on it never wakes.
- R4. The X reads as an affordance, not as one of the choices.

## Acceptance Criteria

- [x] AC1. `ShowDetail`, `ConfirmDestructive`, `ConfirmWithBody`, `Prompt` and
  `Decide` all draw a close X above their body (`closable`).
- [x] AC2. Tapping it closes the dialog and leaves no overlay
  (`TestEveryDialogClosesFromItsCorner`).
- [x] AC3. It does not run `ConfirmDestructive`'s action, and does not confirm
  a `Prompt` (same test, `ran` stays zero).
- [x] AC4. It answers `Decide` false rather than leaving the caller waiting
  (same test, with a one-second deadline so a regression fails rather than
  hangs).
- [x] AC5. Low importance, icon only, so it is not mistaken for a third answer
  to a two-answer question.
- [x] AC6. `make lint` is clean and the full suite passes.

## Risks & Assumptions

- The X sits under the title rather than in the title bar, which is Fyne's.
  If a future Fyne exposes the title bar, the button moves and nothing else
  changes — `closable` is the only place that knows.
- No API changed. Every helper keeps its signature, so consumers get the X by
  upgrading and no call site moves.
- Rollback is a revert: the affordance is additive and nothing depends on it.

## Related finding, not addressed here

A banner raised while a dialog is up is drawn *behind* it, and dimmed with
everything else under the modal. That is by construction: `flashLayout` puts
the banner in a layer of the window's content precisely so it does not take
every click in the window (quirk 26), and an overlay above the dialog would
take the dialog's clicks instead. The design notes already answer it from the
other direction — a refusal inside a dialog belongs in the dialog — and
jira-viewer now says such things in the dialog rather than through `Flash`.
Whether this package should offer that line as a component, so each consumer
does not hand-roll it, is worth its own issue.
