# Spec 010: A hover tip

> **Note**: This work has no associated issue tracker ticket. The repository
> is a personal public project without an issue tracker.

## Status: COMPLETE

## Executive Summary

`widgets.WithTip(o, text)` shows a note while the pointer rests on o. Fyne has
no tooltip, so the guidance a control cannot fit in its label had nowhere to go
but under it, which turns a row into a paragraph. The tip is a transparent
catcher stacked over the control: Fyne gives a pointer event to the last match
in its tree walk, so the catcher wins the hover, and it is not `Tappable`, so
taps walk past it to the control underneath, which keeps its own behaviour.
Asked for by terrariabonker's port. Released as v0.1.6.

## Context

Fyne 2.8.1 has no tooltip. It has `desktop.Hoverable` — `MouseIn`, `MouseMoved`,
`MouseOut` — and nothing built on it.

terrariabonker's port (its spec 050) is what wants one. The Qt panel it replaces
keeps every piece of guidance in a tooltip: what a cheat does, why a fishing
spot matters, which other cheat a setting depends on. Twenty-odd of them. With
no tip the port has two bad choices — drop the guidance, or print it under every
control, which turns a compact row into a paragraph and makes a section three
times taller than the one it replaces.

The other three programs have the same shape of text and the same absence. This
belongs in the design system rather than in one consumer.

## Design

### Why a transparent catcher rather than a wrapper

Fyne finds the object for a pointer event by walking the visible tree and
keeping the **last** match — `FindObjectAtPositionMatching` assigns on every
match and does not stop early — so the deepest, latest object wins. A widget
that merely contains a `Check` therefore never sees the hover: the `Check` is
walked after it and takes it.

A tip is instead a transparent object stacked **over** the thing it describes.
It is walked last, so it wins the hover; it does not implement `Tappable`, so
the tap search walks past it to the control underneath. The control keeps its
own behaviour and does not know it has been annotated.

The catcher forwards `MouseIn`/`MouseOut` to the wrapped object when that object
is itself `Hoverable`, so a control that highlights under the pointer still does.

### Shape

	widgets.WithTip(o fyne.CanvasObject, text string) fyne.CanvasObject

One call, any object, no new widget type per control. An empty text returns the
object unchanged, so a caller may annotate conditionally without a branch.

### Rules it follows

- **It floats.** A popup over the content, never inserted into it: the design
  system's standing rule that nothing transient may reflow the interface.
- **It waits.** `TipDelay` before it appears, so moving the pointer across a
  form does not flash a tip per control.
- **It stays inside the window.** Positioned at the pointer, then clamped so a
  tip near the right or bottom edge is not drawn off the canvas.
- **It is not a dialog.** No buttons, no focus, dismissed by leaving.

## Requirements

- R1 `widgets.WithTip(o, text)` returns an object that shows text on hover and
  behaves as `o` otherwise. Empty text returns `o` itself.
- R2 Taps, drags and keys reach the wrapped control unchanged.
- R3 A wrapped control that is `Hoverable` still receives `MouseIn`/`MouseOut`.
- R4 The tip appears after `TipDelay`, and not at all if the pointer leaves
  first.
- R5 The tip is a popup over the content and never changes the layout.
- R6 The tip is clamped to the canvas.
- R7 The gallery shows it, `docs/design-system.md` records it, and the changelog
  names it.

## Acceptance Criteria

- [x] AC1 `WithTip` with text returns something that is not the bare object;
  with empty text it returns the object itself (R1).
- [x] AC2 Hovering shows the text after the delay; leaving before the delay
  shows nothing (R4).
- [x] AC3 Leaving after it is shown takes it down (R4).
- [x] AC4 A tap on a wrapped button still fires it (R2).
- [x] AC5 A wrapped `Hoverable` still gets `MouseIn` and `MouseOut` (R3).
- [x] AC6 A tip near the canvas edge is moved to stay inside it (R6).
- [x] AC7 Gallery card, design-system entry, changelog; tests, lint and vet
  clean (R7).

Verified in `widgets/tip_test.go`: `TestWithTipAnnotatesAndEmptyTextDoesNothing`
(AC1), `TestATipWaitsAndThenAppears` and `TestLeavingBeforeTheDelayShowsNothing`
(AC2), `TestLeavingTakesTheTipDown` (AC3), `TestAWrappedControlStillWorks`
(AC4), `TestAWrappedHoverableStillHears` (AC5),
`TestATipIsClampedToTheCanvas` (AC6), and
`TestFyneStillGivesAPointerEventToTheLastMatch`, the canary for the dispatch
rule the design rests on. AC7: the "widgets.WithTip" card in the gallery's
Shell section, "Guidance" in `docs/design-system.md`, quirks 23 and 24 in
`docs/fyne-quirks.md`, 0.1.6 in the README changelog.

## Risks & Assumptions

- **Assumption**: the last-match dispatch rule holds. It is read from Fyne
  2.8.1's `FindObjectAtPositionMatching`, which assigns `found` on every match
  rather than returning at the first, so the deepest object wins. A canary test
  names this, so an upstream change to that order shows up as one failing test
  rather than tips that stop appearing.
- **Risk**: the catcher covers the control, so anything dispatched by "topmost
  match" rather than "last match" would be intercepted. Taps are the ones that
  matter and AC4 pins them.
- **Rollback**: additive. `git revert`; nothing existing changes shape.
