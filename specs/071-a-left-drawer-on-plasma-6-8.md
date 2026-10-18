# Spec 071: a left drawer on Plasma 6.8

**Issue**: [#213](https://github.com/ushineko/fynedesygn/issues/213)

## Status: COMPLETE

## Executive Summary

`kwin.HoldRightEdgeScript` keeps a left drawer's cards in place on Plasma
6.8 again. KWin 6.8 moves a window's x when the client changes its width,
and the script skipped any change where x had moved, so it never acted. The
script now works out the held x from the geometry before the change, and
compares widths and places within half a pixel, because its own move reports
a width a few ulps from the one it set. Reviewers should look at the guard in
`glance/kwin/drawer.go` first.

## Context

Spec 068 R9: a drawer on the left grows the window leftwards, so the cards
stay under the pointer that clicked them. On Plasma only the compositor moves
a window, and `HoldRightEdgeScript`, loaded while the drawer is on the left,
moves it back by what it grew, in the window's `frameGeometryChanged`
handler. It acted only when the width changed and x did not:

```js
if (w.move || w.resize || g.width == old.width || g.x != old.x) { return; }
```

hayami 0.10.5 on Plasma 6.8.0 reported the cards moving sideways when the
drawer opened. A log of the panel's geometry from a script connected to the
same signal:

```
open:   x 3696 -> 3571.3   width 304 -> 553.3
close:  x 3571.3 -> 3696   width 553.3 -> 304
```

The window grew by 249.3 and moved left by 124.7: KWin 6.8 keeps the
window's centre when a client changes its own width, and moves x in the same
change. The script returned at `g.x != old.x`. On 6.7.5 KWin left x alone.

The first fix dropped the x condition and kept `g.width == old.width`. On
the desk it held the edge on the first opening only, and after that the
panel crept sideways by half a drawer at each opening. A traced copy showed
why: the script's own move calls the handler again, and the width it reports
there is 553.3333333333335 against the 553.3333333333333 that was set. Read
as a widening, the handler moved x back to where KWin had put it, inside the
handler that had just moved it. The `g.x != old.x` condition had been
stopping that second call on 6.7 as a side effect.

## Requirements

- R1 The script holds the right edge when the client changes its width,
  whatever x the compositor gave the window in the same change.
- R2 A change that keeps the width, within half a pixel, is left alone. This
  covers the script's own move and every move that is not a resize.
- R3 An edge already where it is held, within half a pixel, is not set
  again.
- R4 A move or a resize by hand is left alone, as before.

## Acceptance Criteria

- [x] R1: the guard no longer reads x (`g.x != old.x` is gone) and x is
      worked out as `old.x + old.width - g.width`
      (`TestTheHoldScriptActsWhereTheCompositorMovedTheWindow`,
      `TestTheHoldScriptKeepsTheRightEdge`; falsified by putting the x
      condition back).
- [x] R2, R3: widths and places are compared with `Math.abs(...) < 0.5`,
      never `==` (same tests; falsified by restoring `g.width == old.width`).
- [x] R4: `w.move || w.resize` still returns (`TestTheHoldScriptKeepsTheRightEdge`).
- [x] Real desk, Plasma 6.8.0, Wayland, scale 1.5: with the script loaded
      against hayami 0.10.5, the drawer opened and closed on the left several
      times with the cards in place, confirmed by eye; a traced copy logged
      x 3696 -> 3446.7 on each opening and back to 3696 on each closing, the
      right edge at 4000 throughout.

## Risks & Assumptions

- The script is a string checked by string tests; what it does is checked
  only on a desk. There is no JavaScript engine in the test suite and adding
  one for a dozen lines is not worth the dependency.
- Not run on Plasma 6.7: the desk is on 6.8. On 6.7, where x does not move,
  the held x is the same number the old script computed; the script's own
  move keeps the width, so R2 stops the second call there as the x condition
  did.
- While loaded, the script now also holds the edge when the compositor moves
  and widens the window in one change for its own reasons (a screen
  rearranged). It is loaded only while a left drawer is in use.
- Rollback: revert the commit; consumers pin a version.

## E2E Test Plan

- Environment: Plasma 6.8 on Wayland, a consumer with a left drawer (hayami).
- Steps: open and close the drawer on the left several times (AC 4); a panel
  near the right edge of the screen, so the left drawer is chosen on auto.
- Expected: the cards do not move on opening or on closing, and the panel is
  where it started after the last closing.
- Done for this spec against hayami 0.10.5 with the script swapped in; to be
  repeated with hayami built against the release.
