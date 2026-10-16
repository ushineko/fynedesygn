package glance

import (
	"image/color"
	"slices"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
)

// SparklineHeight is the default height of a plot. It is small on purpose: a
// glance window has room for a shape, not for a chart.
const SparklineHeight float32 = 26

// SparklineDriftRate is how many times a second a drifting plot is redrawn
// (spec 070). At 60 samples in 268 px a step is about 4.5 px, so at 2 s a
// sample a frame moves the line under a quarter of a pixel; the person who
// reviewed it could not tell 10 from 20, and 20 cost 2 % of a core to 10's
// 1.25 %.
const SparklineDriftRate = 10

// Sparkline is a fixed-capacity line plot of one or more traces, right
// anchored so the newest sample is at the right edge.
//
// Three rules it enforces, each of which was a defect first:
//
//   - **Each trace is scaled to its own range, not to a shared axis**, unless
//     the plot is told otherwise with SetScale. In one recorded session a CPU
//     trace ranged 65–98 °C while the coolant trace it was drawn beside moved
//     45.8–46.6 °C, roughly 40:1. On a shared axis the coolant trace is two
//     pixels tall. Heights are therefore not comparable between traces, and
//     the rows above the plot are where values are read. A plot whose traces
//     are the same quantity and are meant to be compared (bandwidth) asks for
//     ScaleShared instead.
//   - **Each trace has a minimum span**, so an idle flat line stays flat
//     instead of amplifying sensor noise into a mountain range.
//   - **There is no legend.** Each trace is drawn in the colour of its row's
//     value, and that is the mapping. A legend in a 260 px window is a second
//     copy of the row labels.
//
// The x axis is sample index, not time. A poll that read nothing records
// nothing, so a gap compresses rather than being interpolated across: the plot
// says what was measured and does not invent the rest.
//
// Points are spaced by the capacity, so a plot that is filling grows from the
// right edge rather than re-spacing every point on every sample.
//
// A plot told the time between its samples with SetDrift and fed through
// SetSamples moves at a constant speed between them instead of jumping a step
// when one arrives (spec 070). The newest sample is then fully in place only
// as the next is due: the plot is up to one interval behind the rows above
// it, which are where values are read.
type Sparkline struct {
	widget.BaseWidget

	capacity int
	height   float32
	scale    Scale
	order    []string
	series   map[string]*series

	// drift is the time between samples, zero for a plot that does not
	// drift. start is when the last sample that drifts arrived, and
	// sharedFrom the range a ScaleShared plot was drawn against then.
	drift      time.Duration
	start      time.Time
	sharedFrom span
	now        func() time.Time

	// stop ends the goroutine that redraws a drifting plot, nil when none
	// runs. It is owned by the UI thread.
	stop chan struct{}
}

// span is a range a trace is drawn against: its low edge and its height.
type span struct{ low, size float64 }

// toward is the range a fraction p of the way from r to to.
func (r span) toward(to span, p float64) span {
	return span{r.low + (to.low-r.low)*p, r.size + (to.size-r.size)*p}
}

// Scale is how a plot's traces are fitted to its height.
type Scale int

const (
	// ScaleEach draws every trace against its own range, widened to its
	// minimum span about the middle. It is the default, and the right plot
	// for traces of different quantities read for their shape: coolant at
	// 40 °C beside a processor at 90 °C.
	ScaleEach Scale = iota
	// ScaleShared draws every trace against one range: zero at the bottom and
	// the greatest sample of any trace in the window at the top, widened to
	// the largest of the traces' minimum spans when the peak is below it. It
	// is the right plot for traces of one quantity that are meant to be
	// compared, such as the down and up rates of several interfaces, where a
	// 1 KiB/s trace must be a flat line beside a 20 MiB/s one. It assumes the
	// samples are not negative.
	ScaleShared
)

// series is one trace: its samples, its colour, and its own vertical scale.
type series struct {
	colour  color.Color
	width   float32
	minSpan float64
	samples []float64

	// drifting is true while the trace's last sample is moving into place;
	// dropped is the sample that left a full trace when it arrived, drawn at
	// the left edge until it has moved out; from is the range it was drawn
	// against when it arrived.
	drifting bool
	hasDrop  bool
	dropped  float64
	from     span
}

// bounds are the low edge and the span this trace alone is drawn against,
// widened to its minimum span about the middle of its range.
func (s *series) bounds() (low, span float64) {
	if len(s.samples) == 0 {
		return 0, s.minSpan
	}
	lo, hi := s.samples[0], s.samples[0]
	for _, v := range s.samples[1:] {
		if v < lo {
			lo = v
		}
		if v > hi {
			hi = v
		}
	}
	span = hi - lo
	if span < s.minSpan {
		lo -= (s.minSpan - span) / 2
		span = s.minSpan
	}
	return lo, span
}

// sharedBounds are the low edge and the span every trace is drawn against
// under ScaleShared: zero, and the greatest sample across all traces, widened
// to the largest minimum span when the peak is below it.
func (s *Sparkline) sharedBounds() (low, span float64) {
	for _, ser := range s.series {
		if ser.minSpan > span {
			span = ser.minSpan
		}
		for _, v := range ser.samples {
			if v > span {
				span = v
			}
		}
	}
	if span <= 0 {
		// Every sample is zero and no trace asked for a span: draw the
		// traces along the bottom rather than dividing by zero.
		span = 1
	}
	return 0, span
}

// NewSparkline builds an empty plot holding at most capacity samples per
// trace. Capacity times the poll interval is the window the plot covers: 60
// samples at 5 s is five minutes.
func NewSparkline(capacity int) *Sparkline {
	if capacity < 2 {
		capacity = 2
	}
	s := &Sparkline{
		capacity: capacity,
		height:   SparklineHeight,
		series:   map[string]*series{},
		now:      time.Now,
	}
	s.ExtendBaseWidget(s)
	return s
}

// AddSeries declares a trace. colour should be the colour its row's value is
// painted in, because that is the plot's only legend. minSpan is the smallest
// range the trace is drawn against, in the trace's own units.
func (s *Sparkline) AddSeries(key string, colour color.Color, minSpan float64) {
	if _, ok := s.series[key]; !ok {
		s.order = append(s.order, key)
	}
	s.series[key] = &series{colour: colour, width: 1.5, minSpan: minSpan}
}

// Add records a sample, dropping the oldest when the trace is full. A key that
// was never declared is ignored rather than panicking: a poll should not be
// able to take the window down.
func (s *Sparkline) Add(key string, v float64) {
	ser, ok := s.series[key]
	if !ok {
		return
	}
	s.settle()
	ser.samples = append(ser.samples, v)
	if len(ser.samples) > s.capacity {
		ser.samples = ser.samples[len(ser.samples)-s.capacity:]
	}
	s.Refresh()
}

/*
SetSamples replaces the samples of the traces it names, in one refresh: the
call for a program that keeps its own history and draws the plot from it.

A trace given more than the capacity keeps the newest. A key that was not
declared with AddSeries is ignored, and a trace not named keeps its samples.
The slices are copied. A call that changes nothing does not refresh.

With drift on (SetDrift), a trace given exactly one more sample at the end
moves into place over the drift's interval; any other change, and every trace
not named, is drawn in place at once.
*/
func (s *Sparkline) SetSamples(samples map[string][]float64) {
	changed := false
	for key, next := range samples {
		if ser, ok := s.series[key]; ok && !slices.Equal(ser.samples, s.newest(next)) {
			changed = true
		}
	}
	if !changed {
		return
	}

	// The ranges as drawn now, mid-drift included, so a sample that arrives
	// early blends from where the plot is rather than from where it was.
	p := s.progress()
	sharedFrom := s.sharedRange(p)
	drifts := false
	for _, key := range s.order {
		ser := s.series[key]
		next, named := samples[key]
		if !named {
			ser.drifting = false
			continue
		}
		next = s.newest(next)
		ser.from = s.rangeOf(ser, p)
		ser.drifting = s.drift > 0 && s.isShift(ser.samples, next)
		ser.hasDrop = ser.drifting && len(ser.samples) == s.capacity
		if ser.hasDrop {
			ser.dropped = ser.samples[0]
		}
		ser.samples = append(ser.samples[:0], next...)
		drifts = drifts || ser.drifting
	}
	s.sharedFrom = sharedFrom
	if drifts {
		s.start = s.now()
		s.tick()
	} else {
		s.stopTicking()
	}
	s.Refresh()
}

// newest is the last capacity samples of v.
func (s *Sparkline) newest(v []float64) []float64 {
	if len(v) > s.capacity {
		return v[len(v)-s.capacity:]
	}
	return v
}

// isShift reports whether next is old with one sample added at the end: the
// same samples followed by one, or, when old is full, all but its oldest
// followed by one.
func (s *Sparkline) isShift(old, next []float64) bool {
	switch {
	case len(old) == 0:
		return false
	case len(next) == len(old)+1:
		return slices.Equal(old, next[:len(old)])
	case len(next) == len(old) && len(old) == s.capacity:
		return slices.Equal(old[1:], next[:len(next)-1])
	}
	return false
}

// SetDrift sets the time the program expects between samples, which turns
// drift on: a sample given by SetSamples moves into place over that time
// instead of arriving in one step. Zero turns it off, which is the default.
func (s *Sparkline) SetDrift(every time.Duration) {
	if every == s.drift {
		return
	}
	s.drift = every
	s.settle()
	s.Refresh()
}

// progress is how far the current drift has gone, 0 to 1; 1 when there is
// none.
func (s *Sparkline) progress() float64 {
	if s.drift <= 0 {
		return 1
	}
	p := float64(s.now().Sub(s.start)) / float64(s.drift)
	return min(max(p, 0), 1)
}

// drifting reports whether any trace is moving into place.
func (s *Sparkline) drifting() bool {
	for _, ser := range s.series {
		if ser.drifting {
			return true
		}
	}
	return false
}

// rangeOf is the range a trace is drawn against at drift progress p.
func (s *Sparkline) rangeOf(ser *series, p float64) span {
	if s.scale == ScaleShared {
		return s.sharedRange(p)
	}
	low, size := ser.bounds()
	to := span{low, size}
	if ser.drifting && p < 1 {
		return ser.from.toward(to, p)
	}
	return to
}

// sharedRange is the one range a ScaleShared plot is drawn against at drift
// progress p.
func (s *Sparkline) sharedRange(p float64) span {
	low, size := s.sharedBounds()
	to := span{low, size}
	if p < 1 && s.drifting() {
		return s.sharedFrom.toward(to, p)
	}
	return to
}

// settle draws every trace in place and stops the redraws.
func (s *Sparkline) settle() {
	for _, ser := range s.series {
		ser.drifting = false
	}
	s.stopTicking()
}

// driftTicks is false in a test, which moves the clock itself.
var driftTicks = true

// tick starts the redraws of a drifting plot, if they are not running.
func (s *Sparkline) tick() {
	if s.stop != nil || !driftTicks {
		return
	}
	stop := make(chan struct{})
	s.stop = stop
	go func() {
		t := time.NewTicker(time.Second / SparklineDriftRate)
		defer t.Stop()
		for {
			select {
			case <-stop:
				return
			case <-t.C:
				fyne.Do(func() { s.frame(stop) })
			}
		}
	}()
}

// frame is one redraw of a drifting plot, on the UI thread. The drift that
// has reached its end settles, which stops the redraws until the next sample.
func (s *Sparkline) frame(stop chan struct{}) {
	if s.stop != stop {
		return // a frame queued before the drift was stopped
	}
	if s.progress() >= 1 {
		s.settle()
	}
	if s.Visible() {
		s.Refresh()
	}
}

// stopTicking ends the redraws.
func (s *Sparkline) stopTicking() {
	if s.stop != nil {
		close(s.stop)
		s.stop = nil
	}
}

// SetColour repaints a trace, for a value whose status band changed.
func (s *Sparkline) SetColour(key string, colour color.Color) {
	if ser, ok := s.series[key]; ok && ser.colour != colour {
		ser.colour = colour
		s.Refresh()
	}
}

// Samples are a trace's samples, oldest first. Returned as a copy so a caller
// cannot edit the plot's own history.
func (s *Sparkline) Samples(key string) []float64 {
	ser, ok := s.series[key]
	if !ok {
		return nil
	}
	out := make([]float64, len(ser.samples))
	copy(out, ser.samples)
	return out
}

// HasData reports whether any trace has enough samples to draw a line. The
// plot stays up as long as one does, and a card hides it when none does.
func (s *Sparkline) HasData() bool {
	for _, ser := range s.series {
		if len(ser.samples) > 1 {
			return true
		}
	}
	return false
}

// Clear drops every sample and keeps the traces. History is in memory and
// starts empty after a restart; it is not a state worth persisting.
func (s *Sparkline) Clear() {
	s.settle()
	for _, ser := range s.series {
		ser.samples = nil
	}
	s.Refresh()
}

// SetScale chooses how the traces are fitted to the plot's height. ScaleEach
// is the default; see Scale for when each is right.
func (s *Sparkline) SetScale(sc Scale) {
	if s.scale != sc {
		s.scale = sc
		s.Refresh()
	}
}

// Scale reports how the traces are fitted, so a consumer's test can assert
// the plot it built is the kind it meant without measuring pixels.
func (s *Sparkline) Scale() Scale { return s.scale }

// SetHeight changes the plot's height, which a card scales with the text size.
func (s *Sparkline) SetHeight(h float32) {
	s.height = h
	s.Refresh()
}

// MinSize reserves the plot's height and no width. The card is what decides
// how wide the window is; a plot that demanded a width would be the thing
// setting the window's size, which is the reverse of the rule.
func (s *Sparkline) MinSize() fyne.Size { return fyne.NewSize(0, s.height) }

// CreateRenderer builds the renderer. The lines are rebuilt on every layout
// because the number of segments changes with the number of samples.
func (s *Sparkline) CreateRenderer() fyne.WidgetRenderer {
	return &sparklineRenderer{plot: s}
}

// sparklineRenderer draws the traces as line segments.
//
// The segments are pooled and reused rather than rebuilt. A plot of two traces
// over sixty samples is a hundred and eighteen segments, and Add refreshes on
// every sample, so allocating them per refresh made the cost of drawing a plot
// proportional to how often it was fed — and every one of those allocations
// was garbage a tenth of a second later.
//
// BenchmarkSparklineRefresh: 2397 ns and 118 allocations before, 357 ns and
// none after. TestRefreshingAPlotAllocatesNothing pins the allocation count so
// the pool cannot quietly stop being one.
type sparklineRenderer struct {
	plot *Sparkline

	// pool holds every segment ever made. objects is the same slice as
	// fyne.CanvasObject, kept in step so Objects can hand out a prefix
	// without converting or allocating.
	pool    []*canvas.Line
	objects []fyne.CanvasObject
	used    int

	size fyne.Size
}

func (r *sparklineRenderer) Layout(size fyne.Size) {
	r.size = size
	r.rebuild()
}

func (r *sparklineRenderer) MinSize() fyne.Size { return r.plot.MinSize() }

func (r *sparklineRenderer) Refresh() { r.rebuild() }

// Objects is the segments in use, which is a prefix of the pool. The ones past
// it are kept for the next refresh and must not be drawn: they hold the
// positions of a plot that had more samples than this one.
func (r *sparklineRenderer) Objects() []fyne.CanvasObject { return r.objects[:r.used] }

func (r *sparklineRenderer) Destroy() {}

// segment returns the next pooled line, making one only when the pool has run
// out. Colour and width are set here because a trace's colour changes when its
// status band does.
func (r *sparklineRenderer) segment(colour color.Color, width float32) *canvas.Line {
	if r.used < len(r.pool) {
		l := r.pool[r.used]
		l.StrokeColor = colour
		l.StrokeWidth = width
		r.used++
		return l
	}
	l := canvas.NewLine(colour)
	l.StrokeWidth = width
	r.pool = append(r.pool, l)
	r.objects = append(r.objects, l)
	r.used++
	return l
}

// rebuild draws every trace into the current size. Under ScaleEach each trace
// is scaled to its own bounds, and under ScaleShared all of them to one; see
// the type's comment for why the default is not shared.
//
// Points are spaced by the capacity with the newest at the right edge. A
// drifting trace is drawn a fraction of a step to the right of its place, with
// the sample that left it in front, and the segments past either edge are cut
// at it (spec 070).
func (r *sparklineRenderer) rebuild() {
	r.used = 0
	w, h := r.size.Width, r.size.Height
	if w <= 0 || h <= 0 {
		return
	}

	plot := r.plot
	p := plot.progress()
	step := w / float32(plot.capacity-1)
	for _, key := range plot.order {
		ser := plot.series[key]
		moving := ser.drifting && p < 1
		lead := 0
		if moving && ser.hasDrop {
			lead = 1
		}
		total := len(ser.samples) + lead
		if total < 2 {
			continue
		}
		rng := plot.rangeOf(ser, p)
		var offset float32
		if moving {
			offset = float32(1-p) * step
		}

		point := func(i int) fyne.Position {
			v := ser.dropped
			if i >= lead {
				v = ser.samples[i-lead]
			}
			x := w - float32(total-1-i)*step + offset
			return fyne.NewPos(x, h-float32((v-rng.low)/rng.size)*h)
		}
		prev := point(0)
		for i := 1; i < total; i++ {
			next := point(i)
			r.line(ser, prev, next, w)
			prev = next
		}
	}
	canvas.Refresh(r.plot)
}

// line draws the part of the segment from a to b that is inside 0..w.
func (r *sparklineRenderer) line(ser *series, a, b fyne.Position, w float32) {
	if (a.X < 0 && b.X < 0) || (a.X > w && b.X > w) {
		return
	}
	cut := func(x float32) fyne.Position {
		f := (x - a.X) / (b.X - a.X)
		return fyne.NewPos(x, a.Y+(b.Y-a.Y)*f)
	}
	if a.X < 0 {
		a = cut(0)
	}
	if b.X > w {
		b = cut(w)
	}
	l := r.segment(ser.colour, ser.width)
	l.Position1, l.Position2 = a, b
}
