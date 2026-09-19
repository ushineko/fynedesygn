package glance_test

import (
	"image/color"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ushineko/fynedesygn/fynetest"
	"github.com/ushineko/fynedesygn/glance"
)

var (
	green = color.NRGBA{R: 0x27, G: 0xae, B: 0x60, A: 0xff}
	steel = color.NRGBA{R: 0x5b, G: 0x8f, B: 0xc0, A: 0xff}
)

func TestAFullSparklineDropsItsOldestSample(t *testing.T) {
	s := glance.NewSparkline(3)
	s.AddSeries("coolant", green, 5)

	for _, v := range []float64{1, 2, 3, 4} {
		s.Add("coolant", v)
	}

	assert.Equal(t, []float64{2, 3, 4}, s.Samples("coolant"),
		"the plot is right anchored: the newest sample is the right edge")
}

func TestAnUndeclaredSeriesIsIgnoredRatherThanPanicking(t *testing.T) {
	s := glance.NewSparkline(10)

	assert.NotPanics(t, func() { s.Add("nothing-declared-this", 1) })
	assert.Nil(t, s.Samples("nothing-declared-this"))
}

func TestAPlotWithOneSampleHasNoLineToDraw(t *testing.T) {
	s := glance.NewSparkline(10)
	s.AddSeries("coolant", green, 5)

	assert.False(t, s.HasData(), "an empty plot has nothing to say")
	s.Add("coolant", 46)
	assert.False(t, s.HasData(), "one sample is a point, not a trend")
	s.Add("coolant", 46.2)
	assert.True(t, s.HasData())
}

// The rule: each trace is scaled to its own range. On a shared axis a coolant
// trace moving 0.8 °C beside a CPU trace moving 33 °C would be about two
// pixels tall.
func TestEachTraceIsScaledToItsOwnRange(t *testing.T) {
	a := fynetest.App(t)
	_ = a

	s := glance.NewSparkline(10)
	s.AddSeries("coolant", green, 0.1)
	s.AddSeries("cpu", steel, 0.1)
	for _, v := range []float64{45.8, 46.6} {
		s.Add("coolant", v)
	}
	for _, v := range []float64{65, 98} {
		s.Add("cpu", v)
	}

	lines := drawnLines(t, s, fyne.NewSize(100, 26))
	require.Len(t, lines, 2, "one segment per trace for two samples each")

	// Both traces span the full height, because each is drawn against its own
	// min and max. On a shared axis the coolant trace would be flat.
	for _, l := range lines {
		assert.InDelta(t, 26.0, float64(l.Position1.Y-l.Position2.Y), 0.5,
			"a trace should use the full height of its own range")
	}
}

// A minimum span keeps an idle flat line flat instead of amplifying sensor
// jitter into a mountain range.
func TestAMinimumSpanKeepsAnIdleTraceFlat(t *testing.T) {
	a := fynetest.App(t)
	_ = a

	s := glance.NewSparkline(10)
	s.AddSeries("coolant", green, 5) // 5 °C minimum span
	s.Add("coolant", 46.0)
	s.Add("coolant", 46.1) // 0.1 °C of real movement

	lines := drawnLines(t, s, fyne.NewSize(100, 26))
	require.Len(t, lines, 1)

	rise := float64(lines[0].Position1.Y - lines[0].Position2.Y)
	assert.Less(t, rise, 1.0,
		"0.1 °C against a 5 °C minimum span should be under a pixel, got %v", rise)
}

func TestClearDropsTheHistoryAndKeepsTheTraces(t *testing.T) {
	s := glance.NewSparkline(10)
	s.AddSeries("coolant", green, 5)
	s.Add("coolant", 1)
	s.Add("coolant", 2)
	require.True(t, s.HasData())

	s.Clear()

	assert.False(t, s.HasData())
	assert.Empty(t, s.Samples("coolant"))
	s.Add("coolant", 3) // the trace still exists
	assert.Len(t, s.Samples("coolant"), 1)
}

// The plot reserves height and no width: the cards decide how wide the window
// is, and a plot that demanded a width would be setting the window's size.
func TestThePlotReservesHeightAndNoWidth(t *testing.T) {
	s := glance.NewSparkline(60)

	assert.Equal(t, float32(0), s.MinSize().Width)
	assert.Equal(t, glance.SparklineHeight, s.MinSize().Height)
}

// drawnLines renders the plot at a size and returns the lines it drew.
func drawnLines(t *testing.T, s *glance.Sparkline, size fyne.Size) []*canvas.Line {
	t.Helper()
	s.Resize(size)
	return fynetest.All[*canvas.Line](s)
}

// Add refreshes on every sample, and a plot of two traces over sixty samples
// is 118 segments. Allocating them per refresh made the cost of drawing a plot
// proportional to how often it was fed; the segments are pooled instead.
//
// Steady state means the plot is full, which is what a glance window's plot is
// for all but its first minute.
func TestRefreshingAPlotAllocatesNothing(t *testing.T) {
	_ = fynetest.App(t)

	s := glance.NewSparkline(60)
	s.AddSeries("coolant", green, 5)
	s.AddSeries("cpu", steel, 5)
	s.Resize(fyne.NewSize(300, 26))
	for i := range 60 {
		s.Add("coolant", 46+float64(i%5)*0.2)
		s.Add("cpu", 62+float64(i%11))
	}

	got := testing.AllocsPerRun(200, func() {
		s.Add("coolant", 46.4)
		s.Add("cpu", 70)
	})

	assert.Zero(t, got, "a refresh on a full plot allocated %v times", got)
}

// The pool is reused, not merely cleared: a plot that shrank must not draw the
// segments of the longer one it used to be.
func TestAShrunkPlotDoesNotDrawItsOldSegments(t *testing.T) {
	_ = fynetest.App(t)

	s := glance.NewSparkline(60)
	s.AddSeries("coolant", green, 5)
	s.Resize(fyne.NewSize(300, 26))
	for i := range 40 {
		s.Add("coolant", 46+float64(i%3))
	}
	require.Len(t, fynetest.All[*canvas.Line](s), 39, "39 segments for 40 samples")

	s.Clear()
	for i := range 5 {
		s.Add("coolant", 46+float64(i))
	}

	assert.Len(t, fynetest.All[*canvas.Line](s), 4,
		"the plot drew segments left over from when it was longer")
}

func BenchmarkSparklineRefresh(b *testing.B) {
	test.NewApp()
	b.Cleanup(func() { test.NewApp() })

	s := glance.NewSparkline(60)
	s.AddSeries("coolant", green, 5)
	s.AddSeries("cpu", steel, 5)
	s.Resize(fyne.NewSize(300, 26))
	for i := range 60 {
		s.Add("coolant", 46+float64(i%5)*0.2)
		s.Add("cpu", 62+float64(i%11))
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		s.Add("coolant", 46.4)
	}
}
