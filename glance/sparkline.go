package glance

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
)

// SparklineHeight is the default height of a plot. It is small on purpose: a
// glance window has room for a shape, not for a chart.
const SparklineHeight float32 = 26

// Sparkline is a fixed-capacity line plot of one or more traces, right
// anchored so the newest sample is at the right edge.
//
// Three rules it enforces, each of which was a defect first:
//
//   - **Each trace is scaled to its own range, not to a shared axis.** In one
//     recorded session a CPU trace ranged 65–98 °C while the coolant trace it
//     was drawn beside moved 45.8–46.6 °C, roughly 40:1. On a shared axis the
//     coolant trace is two pixels tall. Heights are therefore not comparable
//     between traces, and the rows above the plot are where values are read.
//   - **Each trace has a minimum span**, so an idle flat line stays flat
//     instead of amplifying sensor noise into a mountain range.
//   - **There is no legend.** Each trace is drawn in the colour of its row's
//     value, and that is the mapping. A legend in a 260 px window is a second
//     copy of the row labels.
//
// The x axis is sample index, not time. A poll that read nothing records
// nothing, so a gap compresses rather than being interpolated across: the plot
// says what was measured and does not invent the rest.
type Sparkline struct {
	widget.BaseWidget

	capacity int
	height   float32
	order    []string
	series   map[string]*series
}

// series is one trace: its samples, its colour, and its own vertical scale.
type series struct {
	colour  color.Color
	width   float32
	minSpan float64
	samples []float64
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
	ser.samples = append(ser.samples, v)
	if len(ser.samples) > s.capacity {
		ser.samples = ser.samples[len(ser.samples)-s.capacity:]
	}
	s.Refresh()
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
	for _, ser := range s.series {
		ser.samples = nil
	}
	s.Refresh()
}

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

type sparklineRenderer struct {
	plot    *Sparkline
	objects []fyne.CanvasObject
	size    fyne.Size
}

func (r *sparklineRenderer) Layout(size fyne.Size) {
	r.size = size
	r.rebuild()
}

func (r *sparklineRenderer) MinSize() fyne.Size { return r.plot.MinSize() }

func (r *sparklineRenderer) Refresh() { r.rebuild() }

func (r *sparklineRenderer) Objects() []fyne.CanvasObject { return r.objects }

func (r *sparklineRenderer) Destroy() {}

// rebuild draws every trace into the current size. Each trace is scaled to its
// own bounds; see the type's comment for why they are not shared.
func (r *sparklineRenderer) rebuild() {
	r.objects = r.objects[:0]
	w, h := r.size.Width, r.size.Height
	if w <= 0 || h <= 0 {
		return
	}

	for _, key := range r.plot.order {
		ser := r.plot.series[key]
		n := len(ser.samples)
		if n < 2 {
			continue
		}
		low, span := ser.bounds()
		step := w / float32(n-1)

		at := func(i int) fyne.Position {
			y := h - float32((ser.samples[i]-low)/span)*h
			return fyne.NewPos(float32(i)*step, y)
		}
		for i := range n - 1 {
			l := canvas.NewLine(ser.colour)
			l.StrokeWidth = ser.width
			l.Position1 = at(i)
			l.Position2 = at(i + 1)
			r.objects = append(r.objects, l)
		}
	}
	canvas.Refresh(r.plot)
}
