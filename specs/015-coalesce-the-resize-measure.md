# Spec 015: Coalesce the resize measure

**Issue**: ushineko/clockwork-orange#26

## Status: COMPLETE

- **Priority**: Medium
- **Estimated Complexity**: Low
- **Branch**: `fix/coalesce-measure`
- **Asked for by**: clockwork-orange, whose About section is jerky to resize

## Context

Reported against clockwork-orange: the About section is slow to switch to and
jerky to resize. A 20-second CPU profile, taken through spec 017's endpoint
while the window was being dragged:

    Duration: 20s, Total samples = 7590ms (37.95%)

                      cum     cum%
    markdown.(*Pane).Resize    3.18s   41.90%
    markdown.(*Pane).measure   2.40s   31.62%
    glfw.MakeContextCurrent    1.62s   28.32%

A third of all CPU during a drag is this pane re-measuring, and the call
arrives through `glfwPollEvents → cgocallback → processResized`, which is to
say Fyne performs the whole relayout synchronously **inside the event poll**.
The event queue cannot drain while that runs, so the window moves in bursts.

`measure` renders every block to ask its height — `Visual(i)` builds the
block's widget tree — and it does so on every width change. A drag is hundreds
of width changes.

The other two thirds are Fyne's own resize path and are not this module's to
fix.

## Requirements

### R1. A drag is waited out

- R1.1 `Options.SettleResize` is how long a width must stop changing before the
  document is measured again. Zero, the default, measures on every change as
  this pane always has.
- R1.2 **Opt-in, like `logpane.Pump`.** The work runs on a timer and returns to
  the UI thread with `fyne.Do`, which is a real hop in a real program and an
  inline call under the test driver — so a pane that scheduled timers by itself
  would run text shaping on a timer goroutine in every headless test that ever
  resized a window. The library's rule is that work off the UI thread is the
  program's to start.
- R1.3 The timer is restarted on every step, so a drag schedules the work
  repeatedly and performs it once.

### R2. A jump is not a drag

- R2.1 A width change of `DragFraction` (a quarter) or more is measured at
  once. A section being shown, a window being maximised, or a first layout
  arrives as one large jump, and delaying it would show the document at the
  wrong heights — visible, for no gain, because it happens once.
- R2.2 A pane that has never been measured measures immediately: a document
  with no heights has no shape at all.

### R3. Nothing fires into a pane that is gone

- R3.1 `Detach` cancels a pending measure. The shell calls it before replacing
  a section, and a timer firing afterwards would measure a pane nobody is
  looking at and hop to the UI thread to do it.
- R3.2 Measuring directly cancels a pending measure, so nothing fires
  afterwards at a width that has been overtaken.
- R3.3 `Settle` performs a pending measure now and reports whether there was
  one, for a test and for a caller that knows the resizing has stopped.

### R4. Measure each block once

- R4.1 `measure` calls `MinSize` once per block rather than twice. It is the
  expensive call — it shapes the block's text — and the second was asking the
  same question again after a `Resize` that cannot change the answer.

## Acceptance Criteria

- [x] AC1 A drag of forty steps measures once, at the width it ended on (R1.1, R1.3)
  - Verified: `TestADragMeasuresOnceNotOncePerStep`
- [x] AC2 A change of a quarter or more is measured without waiting (R2.1)
  - Verified: `TestALargeChangeIsMeasuredImmediately`
- [x] AC3 An unmeasured pane measures immediately (R2.2)
  - Verified: `TestADragMeasuresOnceNotOncePerStep` — the baseline is positive before the drag begins
- [x] AC4 `Detach` cancels a pending measure (R3.1)
  - Verified: `TestDetachCancelsAPendingMeasure`
- [x] AC5 The same width twice does nothing (R3.2)
  - Verified: `TestMeasuringIsSkippedWhenTheWidthIsUnchanged`
- [x] AC6 `MinSize` is called once per block (R4.1)
  - Verified: `measure` in `pane.go`
- [x] AC7 The default is unchanged behaviour (R1.1)
  - Verified: `SettleResize` zero takes the immediate path; every existing test passes untouched
- [x] AC8 `make test` passes, with the race detector
  - Verified: all packages ok
- [x] AC9 `make lint` is clean
  - Verified: 0 issues
- [x] AC10 `govulncheck ./...` is clean
  - Verified: no vulnerabilities found

## Risks & Assumptions

- **Rollback**: revert the commit. The module is `v0.x` and adopters pin a
  version.
- **Additive and default-off.** A caller that sets nothing behaves exactly as
  before, which is why no existing test changed.
- **The document is briefly laid out for the old width.** While a drag is in
  progress the heights are the ones from before it. That is a smaller wrongness
  than the one it replaces and it lasts until the user lets go.
- **`fyne.Do` from a timer.** Safe in a real program, which is the whole point
  of the call. Under the test driver it runs inline, which is why this is
  opt-in and why the tests set a settle of an hour and drive `Settle`
  explicitly rather than waiting on a timer.
- **The remaining two thirds are Fyne's.** `MakeContextCurrent` and the
  synchronous relayout inside `processResized` are the driver's design. This
  removes the part that is ours.

## Gaps found

- **`Visual(i)` renders a block and keeps it forever.** Measuring the document
  therefore renders all of it, and the pane's virtualisation applies only to
  what is in the widget *tree*, not to what has been built. A document of any
  size costs its whole rendering the first time it is measured. Worth its own
  spec: measure from a cheaper source than a rendered widget, or discard
  renderings outside the viewport.

## Alternatives Considered

- **Estimate the heights of blocks that are off screen.** Rejected: the pane's
  stated invariant is that heights are measured, never estimated, because an
  estimate that came out short clips a paragraph's last line and one that came
  out long leaves a gap.
- **Throttle on the UI thread with no timer at all.** Rejected: it needs no
  goroutine, but a drag that stops produces no further `Resize`, so the final
  width would never be measured and the document would stay laid out for a
  width the window no longer has.
- **Schedule the timer unconditionally.** Rejected: it raced the font cache in
  the gallery's own headless tests, because `fyne.Do` under the test driver
  runs inline on the calling goroutine. Found by the race detector in
  `make test`.
