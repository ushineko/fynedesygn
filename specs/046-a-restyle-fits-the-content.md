# Spec 046: a restyle fits the content

**Issue**: [#140](https://github.com/ushineko/fynedesygn/issues/140)

## Status: COMPLETE

## Context

`glance.Panel.Resize` has to tell a width the user dragged from a width
nobody chose (spec 016 and the comment in `panel.go`): the first resize
takes the content, and afterwards a width the window has that the panel did
not ask for is remembered as `chosen` and handed back on every later
resize, because the panel resizes on every reading.

A restyle breaks that reading of the width. `SetTheme` replaces the window
content, Fyne fits the window to the new face's minimum on its own (quirk
34 grows and never shrinks), and `Restyle` then calls `Resize`, which finds
a width it did not ask for and records Fyne's growth as the user's choice.
Switching from a wide face to a narrow one afterwards keeps the wide width
as a floor for the life of the program; hayami's user reclaims the space
with Alt+F3 (hayami #85).

The width the window has during a restyle is nobody's choice: everything
was just re-measured. A restyle should fit the content and forget the
remembered width, and a drag after that is honoured as before.

## Requirements

- R1 `Panel.Restyle` (and so `SetTheme`, which ends in it) clears
  `chosen` and resizes to `Size()`: the content's minimum widened to the
  panel's floor. It does not compare the window's current width with
  `asked` on that pass.
- R2 A drag after a restyle is still remembered: the next `Resize` after
  the restyle compares as before.
- R3 `SetArrangement` and `Add`, which also re-measure, behave the same
  way through the same path if they end in `Restyle`; if they call
  `Resize` directly, they keep today's behaviour (a card added should not
  discard a width the user dragged).
- R4 `docs/glance.md` "Size" (or wherever the chosen width is described)
  gains the sentence: a restyle fits the content and forgets a dragged
  width.
- R5 Changelog under `### Unreleased`.

## Acceptance Criteria

- [x] `make test` and `make lint` pass.
- [x] A test on the Fyne test driver: `SetTheme(wideFace)` then
  `SetTheme(narrowFace)` leaves the window at the narrow content's width
  (`p.Size().Width`), where today it stays at the wide one (the test fails
  before the fix and passes after; noted below).
- [x] A test: after a restyle, a window width set by the test (as a drag)
  followed by a `Resize` is kept.
- [x] `docs/glance.md` and the changelog updated in the same commit.

## Risks & Assumptions

- **A user who dragged the panel wider loses that width on a font change.**
  That is the requested behaviour: a font change is a re-measure, and the
  user drags again if they want it wider.
- **Rollback**: revert.

## Verification

Tests, in `glance/panel_test.go` on the Fyne test driver (headless):

- `TestARestyleFitsTheContent` -- a resizable window with no floor,
  `SetTheme` at text size 24 then at 9; the window ends at the narrow
  content's `p.Size().Width`.
- `TestAWidthSetAfterARestyleIsKept` -- `SetTheme`, then the test resizes the
  window 300 px past the content (the drag), then three `Resize` calls; the
  dragged width is kept on each.

Before the fix, `TestARestyleFitsTheContent` failed:

```
--- FAIL: TestARestyleFitsTheContent
    Error: Not equal:
           expected: 112.421875
           actual  : 300
    Messages: the window kept the wide face's width after a narrower restyle
```

The wide face's content measured 293 and the window was 300: the width Fyne
fitted on `SetContent`, which `Resize` then recorded as `chosen`.
`TestAWidthSetAfterARestyleIsKept` passed before the fix as well; it is the
guard that the fix does not over-reach (R2).

After the fix both pass, as does `TestAWidthTheUserSetSurvivesEveryLaterResize`.

The fix: `Resize` became `resize(fit bool)`; `Restyle` calls `resize(true)`,
which sets `chosen` to 0 instead of comparing the window's width with
`asked`. `Add` and `SetArrangement` call `Resize` directly and are unchanged
(R3). The other `Restyle` caller, `Window`'s translucency grant, runs before
the window is shown, where there is no drag to forget.

`make test` passed; `make lint` reported 0 issues. The first `make test` run
failed on a data race in `shell.TestPerformCancellableCancelsThroughTheContext`,
which this change does not touch; it passed on five isolated reruns, on three
runs of `./shell` with the change stashed, and on the full rerun. Intermittent
and pre-existing.
