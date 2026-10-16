# Spec 070: a sparkline that drifts between samples

**Issue**: [#211](https://github.com/ushineko/fynedesygn/issues/211)

## Status: COMPLETE

## Context

`glance.Sparkline` moves only when a sample arrives, and then every point moves
one step left in one frame. At 60 samples in a 268 px plot a step is about
4.5 px, and hayami's samples arrive every 2 s (bandwidth) and 5 s (cooler), so
the trace reads as a series of jolts. Two more jolts come with it: the scale is
recomputed in one frame when an extreme enters or leaves the window, and a plot
that is still filling re-spaces every point on every sample, because points are
spaced by the number held rather than by the capacity.

hayami asked about polling faster to hide it. That costs reads and makes the
data worse: a byte counter sampled faster than its driver updates it reports
noise, and a Kraken serves a cached status for 250 ms. The motion belongs in
the plot.

A prototype tried three ways, on a 2 s and a 5 s feed, on an i9-14900K, one
process per variant, measured with `getrusage` over 90–120 s:

| Variant | CPU, one core |
|---|---|
| The plot as it is (`Clear` + 60 `Add` per sample, as hayami feeds it) | 0.42 % |
| A 350 ms ease-out slide of one step | 1.1 % |
| A constant drift across the interval, 10 frames a second | 1.25 % |
| The same at 20 frames a second | 2.0 % |
| The same at the display's rate | 4.0 % |
| The display's rate with monotone cubic smoothing (four segments a step) | 10.4 % |

The ease-out slide moves 4.5 px in its first 100 ms and is not seen as motion.
The drift is, and the person reviewing it could not tell 10 frames a second
from 20: at 4.5 px per 2 s a frame moves the line a fraction of a pixel.
Smoothing costs four times the segments and was not asked for.

The prototype also found a trap worth recording: a widget wrapped in another
struct and put in a window draws once and never again, because Fyne finds the
renderer by the pointer `Refresh` is called on.

## Requirements

### Spacing

- R1 Points are spaced by the capacity, not by the number held:
  `width / (capacity-1)`, with the newest sample at the right edge. A plot
  that is filling grows from the right; its points do not move apart.

### Setting the samples

- R2 `Sparkline.SetSamples(map[string][]float64)` replaces the samples of the
  traces it names, in one refresh. A trace given more than the capacity keeps
  the newest. A key that was not declared with `AddSeries` is ignored, as
  `Add` ignores it. A trace not named keeps its samples. The slices are
  copied.
- R3 A call that changes nothing does not refresh.

### Drift

- R4 `Sparkline.SetDrift(every time.Duration)` turns drift on: `every` is the
  time the program expects between samples. Zero turns it off, and off is the
  default, so a plot never told is drawn as before.
- R5 With drift on, a trace that `SetSamples` gave exactly one more sample
  at the end (the old samples, the same, followed by one; or, when full, the
  old samples less the oldest followed by one) is drawn one step to the
  right of its place and moves left at a constant speed, reaching its place
  `every` after the call. The sample that left a full trace is drawn at the
  left edge until it has moved out. Segments past either edge are cut at it.
- R6 Any other change to a trace (a reset, a gap, more than one new sample)
  is drawn in place at once, as is every trace not named by the call. A
  trace that is drifting and is not named stops where its place is.
- R7 While a trace drifts its range moves with it: the low edge and the span
  are blended at the same constant rate from what was drawn when the sample
  arrived to the range of the new samples. Under `ScaleShared` the one range
  is blended. A trace that does not drift is drawn against its new range at
  once.
- R8 The plot redraws `SparklineDriftRate` (10) times a second while a
  trace is drifting and not at all when none is. A plot hidden with `Hide`
  is not redrawn. A sample that arrives early cuts the drift short; one that arrives
  late leaves the plot at rest until it does.
- R9 `Add` and `Clear` stop the drift and draw in place, as before.

### What does not change

- R10 Scaling, minimum spans, colours and the pooled segments are as spec 016
  and its successors left them. A refresh of a full plot that is not drifting
  allocates nothing (`TestRefreshingAPlotAllocatesNothing`); a drifting
  refresh allocates nothing either.

### Docs and gallery

- R11 `docs/glance.md` says when to use drift and what it costs, with the
  table above; the numbers table gains the drift rate. The README's changelog
  says what changed, the spacing change among it.
- R12 The gallery shows a drifting plot fed every 2 s through `SetSamples`.

## Acceptance Criteria

- [x] A filling plot draws its newest point at the right edge and its points
      `width/(capacity-1)` apart (R1), tested headless.
- [x] `SetSamples` replaces named traces, trims to capacity, ignores
      undeclared keys, keeps unnamed traces, copies, and does not refresh when
      nothing changed (R2, R3), tested headless.
- [x] With drift on, a one-sample shift is drawn one step right at the call,
      half a step right at half of `every`, and in place at `every`; the
      departed sample is drawn at the left edge until then; segments past
      the edges are cut (R5), tested headless against a fake clock.
- [x] A reset, a non-shift change, `Add` and `Clear` draw in place at once
      (R6, R9), tested headless.
- [x] The range is blended over the drift, for `ScaleEach` and `ScaleShared`
      (R7), tested headless.
- [x] A drifting refresh allocates nothing (R10), tested.
- [x] Drift off draws exactly what it drew before, the existing sparkline
      tests unchanged (R4, R10).
- [x] On a real window the plot moves between two captures 300 ms apart while
      drifting, and does not between two captures once at rest (R8), checked
      against spectacle captures of a KDE window drawing `glance.Sparkline`
      from this branch (the prototype program, `-variant lib`): every capture
      differed while samples arrived every 2 s and 5 s, none did with no
      samples. hayami's adoption repeats the check on its own window.
- [x] CPU of a drifting plot at 10 frames a second is measured and recorded in
      the commit (library rule: performance is measured): 1.18–1.29 % of one
      core against 0.46–0.51 % for the stepping plot, the same machine and
      feeds, 120 s each.
- [x] docs/glance.md, the numbers table, the README changelog and the gallery
      are updated (R11, R12).

## Risks & Assumptions

- The drift lags the plot by up to one interval: the newest sample is fully
  in place only as the next is due. The rows above the plot show the value at
  once, which is where values are read. Accepted by the person who asked for
  this, against the alternative of pinning the newest point, which brings the
  jolt back.
- A goroutine per drifting plot ticks at 10 Hz and stops when the drift ends,
  so a plot removed from a window without being destroyed stops within one
  interval.
- The spacing change (R1) is visible to every consumer while a plot fills.
  It is a fix, but it is a change, and the changelog names it.
- Rollback: revert the commit; consumers that call `SetSamples` or
  `SetDrift` would need the same revert, and hayami is the only one.

## Alternatives Considered

- Polling faster in the consumer: rejected for cost and for noisier data.
- A short eased slide per sample: measured, and not seen as motion.
- Drift at the display's rate: three times the cost of 10 Hz with no
  difference the reviewer could see.
- Monotone cubic smoothing: four times the segments, not asked for, and a
  softening of "the plot says what was measured".
