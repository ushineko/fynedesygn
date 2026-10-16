package glance_test

import (
	"slices"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ushineko/fynedesygn/fynetest"
	"github.com/ushineko/fynedesygn/glance"
)

// The drift tests draw a plot of capacity 11 at 100 px, so a step is 10 px,
// and move a fake clock through a 2 s drift.
const (
	driftCap   = 11
	driftEvery = 2 * time.Second
)

// clock is a fake time a test moves by hand.
type clock struct{ t time.Time }

func (c *clock) now() time.Time      { return c.t }
func (c *clock) add(d time.Duration) { c.t = c.t.Add(d) }
func newClock() *clock               { return &clock{t: time.Unix(1_000_000, 0)} }
func ramp(from, n int) (out []float64) { // from, from+1, ... n of them
	for i := range n {
		out = append(out, float64(from+i))
	}
	return out
}

// driftPlot is a plot on a fake clock with drift on and no redraws but the
// test's.
func driftPlot(t *testing.T, sc glance.Scale) (*glance.Sparkline, *clock) {
	t.Helper()
	fynetest.App(t)
	t.Cleanup(glance.HoldDriftTicks())
	c := newClock()
	s := glance.NewSparkline(driftCap)
	s.SetClock(c.now)
	s.SetScale(sc)
	s.AddSeries("a", green, 1)
	s.SetDrift(driftEvery)
	return s, c
}

// drawnNow redraws the plot at 100 by 100 and returns its lines. A test that
// moved the clock must redraw: the redraws a drift starts are held.
func drawnNow(t *testing.T, s *glance.Sparkline) []*canvas.Line {
	t.Helper()
	s.Resize(fyne.NewSize(100, 100))
	s.Refresh()
	return fynetest.All[*canvas.Line](s)
}

// starts are the left ends of the drawn segments, left to right.
func starts(t *testing.T, s *glance.Sparkline) []float32 {
	t.Helper()
	var xs []float32
	for _, l := range drawnNow(t, s) {
		xs = append(xs, l.Position1.X)
	}
	slices.Sort(xs)
	return xs
}

// R1: a filling plot grows from the right edge; its points do not move apart.
func TestAFillingPlotGrowsFromTheRightEdge(t *testing.T) {
	fynetest.App(t)
	s := glance.NewSparkline(driftCap)
	s.AddSeries("a", green, 1)
	for _, v := range []float64{1, 2, 3} {
		s.Add("a", v)
	}

	lines := drawnLines(t, s, fyne.NewSize(100, 100))
	require.Len(t, lines, 2)
	assert.InDelta(t, 80, lines[0].Position1.X, 0.01, "points are a capacity step apart")
	assert.InDelta(t, 90, lines[1].Position1.X, 0.01)
	assert.InDelta(t, 100, lines[1].Position2.X, 0.01, "the newest sample is the right edge")
}

// R2: SetSamples replaces what it names, trims, copies, and leaves the rest.
func TestSetSamplesReplacesTheTracesItNames(t *testing.T) {
	fynetest.App(t)
	s := glance.NewSparkline(3)
	s.AddSeries("a", green, 1)
	s.AddSeries("b", steel, 1)
	s.Add("b", 7)

	in := []float64{1, 2, 3, 4}
	s.SetSamples(map[string][]float64{"a": in, "nobody": {1, 2}})
	in[3] = 99

	assert.Equal(t, []float64{2, 3, 4}, s.Samples("a"), "trimmed to the capacity, and copied")
	assert.Equal(t, []float64{7}, s.Samples("b"), "a trace not named keeps its samples")
	assert.Nil(t, s.Samples("nobody"), "an undeclared key is ignored")
}

// R5: a one-sample shift is drawn a step to the right at the call, half a step
// at half the interval, and in place at the end, with the sample that left
// drawn at the left edge until it has gone.
func TestAShiftedSampleDriftsIntoPlace(t *testing.T) {
	s, c := driftPlot(t, glance.ScaleEach)
	s.SetSamples(map[string][]float64{"a": ramp(0, driftCap)})
	s.SetSamples(map[string][]float64{"a": ramp(1, driftCap)})

	at := starts(t, s)
	require.Len(t, at, 11, "eleven segments: the departed sample and the ten between the eleven held")
	assert.InDelta(t, 0, at[0], 0.01, "the departed sample is at the left edge")
	assert.InDelta(t, 10, at[1], 0.01, "every point is a step right of its place")

	c.add(driftEvery / 2)
	at = starts(t, s)
	require.Len(t, at, 11)
	assert.InDelta(t, 0, at[0], 0.01, "the departed sample's segment is cut at the edge")
	assert.InDelta(t, 5, at[1], 0.01, "half a step right at half the interval")

	c.add(driftEvery / 2)
	at = starts(t, s)
	require.Len(t, at, 10, "at rest the departed sample is gone")
	assert.InDelta(t, 0, at[0], 0.01, "in place at the end of the interval")
	assert.InDelta(t, 10, at[1], 0.01)
}

// R5: nothing is drawn past the right edge while the newest point is outside.
func TestADriftingSegmentIsCutAtTheRightEdge(t *testing.T) {
	s, c := driftPlot(t, glance.ScaleEach)
	s.SetSamples(map[string][]float64{"a": ramp(0, driftCap)})
	s.SetSamples(map[string][]float64{"a": ramp(1, driftCap)})
	c.add(driftEvery / 4)

	for _, l := range drawnNow(t, s) {
		assert.LessOrEqual(t, l.Position2.X, float32(100.001))
		assert.GreaterOrEqual(t, l.Position1.X, float32(-0.001))
	}
}

// R3: a call that changes nothing leaves the drift where it was.
func TestTheSameSamplesAgainDoNotRestartTheDrift(t *testing.T) {
	s, c := driftPlot(t, glance.ScaleEach)
	s.SetSamples(map[string][]float64{"a": ramp(0, driftCap)})
	s.SetSamples(map[string][]float64{"a": ramp(1, driftCap)})
	c.add(driftEvery / 2)

	s.SetSamples(map[string][]float64{"a": ramp(1, driftCap)})

	assert.InDelta(t, 5, starts(t, s)[1], 0.01, "still half a step right")
}

// R6 and R9: anything but a one-sample shift, and Add and Clear, draw in place.
func TestAnythingButAShiftIsDrawnInPlace(t *testing.T) {
	for name, change := range map[string]func(*glance.Sparkline){
		"a reset": func(s *glance.Sparkline) {
			s.SetSamples(map[string][]float64{"a": ramp(50, driftCap)})
		},
		"two new samples": func(s *glance.Sparkline) {
			s.SetSamples(map[string][]float64{"a": ramp(3, driftCap)})
		},
		"Add": func(s *glance.Sparkline) { s.Add("a", 12) },
		"drift off": func(s *glance.Sparkline) {
			s.SetDrift(0)
		},
	} {
		t.Run(name, func(t *testing.T) {
			s, _ := driftPlot(t, glance.ScaleEach)
			s.SetSamples(map[string][]float64{"a": ramp(0, driftCap)})
			s.SetSamples(map[string][]float64{"a": ramp(1, driftCap)})
			require.InDelta(t, 10, starts(t, s)[1], 0.01, "drifting before the change")

			change(s)

			at := starts(t, s)
			require.Len(t, at, 10)
			assert.InDelta(t, 10, at[1], 0.01, "a step apart from the left edge, in place")
		})
	}
}

// R6: a trace that is drifting and not named stops where its place is.
func TestATraceNotNamedStopsDrifting(t *testing.T) {
	s, _ := driftPlot(t, glance.ScaleEach)
	s.AddSeries("b", steel, 1)
	s.SetSamples(map[string][]float64{"a": ramp(0, driftCap), "b": ramp(0, driftCap)})
	s.SetSamples(map[string][]float64{"a": ramp(1, driftCap), "b": ramp(1, driftCap)})

	s.SetSamples(map[string][]float64{"a": ramp(2, driftCap)})

	assert.Len(t, starts(t, s), 21, "a drifts with its departed sample, b is in place")
}

// R7: a drifting trace's range moves from the old to the new over the drift.
func TestADriftingTracesRangeIsBlended(t *testing.T) {
	s, c := driftPlot(t, glance.ScaleEach)
	old := make([]float64, driftCap)
	old[0] = 10 // the only peak, and the next shift drops it
	s.SetSamples(map[string][]float64{"a": old})
	s.SetSamples(map[string][]float64{"a": make([]float64, driftCap)})

	// Old range 0..10; new range all zeros widened to the minimum span 1,
	// so -0.5..0.5. Halfway: low -0.25, size 5.5.
	c.add(driftEvery / 2)
	y := newestY(t, s)
	assert.InDelta(t, 100-0.25/5.5*100, y, 0.05, "zero drawn against the blended range")

	c.add(driftEvery / 2)
	assert.InDelta(t, 50, newestY(t, s), 0.05, "zero is the middle of -0.5..0.5 at rest")
}

// R7: under ScaleShared the one range is blended.
func TestASharedRangeIsBlended(t *testing.T) {
	s, c := driftPlot(t, glance.ScaleShared)
	s.AddSeries("b", steel, 1)
	old := make([]float64, driftCap)
	old[0] = 100
	flat := slices.Repeat([]float64{10}, driftCap)
	s.SetSamples(map[string][]float64{"a": old, "b": flat})
	s.SetSamples(map[string][]float64{"a": make([]float64, driftCap), "b": flat})

	// The one range moves from 0..100 to 0..10 over the drift, and b, flat at
	// 10, is drawn against it. Halfway: 0..55.
	c.add(driftEvery / 2)
	var bY float32
	for _, l := range drawnNow(t, s) {
		if l.StrokeColor == steel {
			bY = l.Position1.Y
		}
	}
	assert.InDelta(t, 100-10.0/55*100, bY, 0.05)
}

// R10: a drifting refresh allocates nothing.
func TestADriftingRefreshAllocatesNothing(t *testing.T) {
	s, c := driftPlot(t, glance.ScaleEach)
	s.AddSeries("b", steel, 1)
	s.Resize(fyne.NewSize(300, 26))
	s.SetSamples(map[string][]float64{"a": ramp(0, driftCap), "b": ramp(0, driftCap)})
	s.SetSamples(map[string][]float64{"a": ramp(1, driftCap), "b": ramp(1, driftCap)})
	c.add(driftEvery / 3)

	got := testing.AllocsPerRun(200, s.Refresh)

	assert.Zero(t, got, "a drifting refresh allocated %v times", got)
}

// Under the test driver an animation runs to its end when it starts, so a
// headless program's drift is put in place at once and nothing redraws the
// plot from another goroutine.
func TestAHeadlessDriftIsInPlaceAtOnce(t *testing.T) {
	s, _ := driftPlot(t, glance.ScaleEach)
	restore := glance.HoldDriftTicks()
	restore()
	s.SetSamples(map[string][]float64{"a": ramp(0, driftCap)})
	s.SetSamples(map[string][]float64{"a": ramp(1, driftCap)})

	assert.False(t, s.Ticking(), "the animation has already ended")
	at := starts(t, s)
	require.Len(t, at, 10)
	assert.InDelta(t, 10, at[1], 0.01, "in place")
}

// Drift reads back what SetDrift set, zero until then.
func TestDriftReadsBackWhatWasSet(t *testing.T) {
	s := glance.NewSparkline(8)
	assert.Zero(t, s.Drift())
	s.SetDrift(driftEvery)
	assert.Equal(t, driftEvery, s.Drift())
}

// newestY is the height the newest sample of a single-trace plot is drawn at.
func newestY(t *testing.T, s *glance.Sparkline) float32 {
	t.Helper()
	var best *canvas.Line
	for _, l := range drawnNow(t, s) {
		if best == nil || l.Position2.X > best.Position2.X {
			best = l
		}
	}
	require.NotNil(t, best)
	return best.Position2.Y
}
