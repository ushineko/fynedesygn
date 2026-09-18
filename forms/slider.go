package forms

import (
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/ushineko/fynedesygn/widgets"
)

// SliderOptions describe a SliderEntry.
type SliderOptions struct {
	Min, Max, Step, Value float64
	// Format renders a value for the entry; nil uses %g.
	Format func(float64) string
	// Commit receives the value once per gesture: slider release or Enter.
	Commit func(float64)
	// OnInvalid is told about text that is not a number; nil ignores it. The
	// entry is restored to the current value either way.
	OnInvalid func(text string)
	// EntryWidth is the entry's fixed width; 0 means NumericWidth.
	EntryWidth float32
}

/*
SliderEntry is a slider and a field that agree, with Reset-friendly Set.

Both controls, not one. The slider is how you find a value you like by feel;
the field is how you type 250000 into a range that goes to a million without
dragging across nine hundred thousand of it. They are kept in step, and only
one of them commits: the slider on release, the field on Enter, so a drag is
one write to the settings rather than four hundred.
*/
type SliderEntry struct {
	opts   SliderOptions
	value  float64
	slider *widget.Slider
	entry  *widget.Entry
	body   fyne.CanvasObject
}

// NewSliderEntry builds the pair at o.Value.
func NewSliderEntry(o SliderOptions) *SliderEntry {
	if o.Format == nil {
		o.Format = func(v float64) string { return strconv.FormatFloat(v, 'g', -1, 64) }
	}
	if o.EntryWidth <= 0 {
		o.EntryWidth = NumericWidth
	}
	s := &SliderEntry{opts: o, value: o.Value}
	s.entry = widget.NewEntry()
	s.entry.SetText(o.Format(o.Value))
	s.slider = widget.NewSlider(o.Min, o.Max)
	s.slider.Step = o.Step
	s.slider.Value = o.Value

	// Live text while dragging, one write on release. OnChanged fires per
	// pixel, and a settings file written per pixel is a settings file written
	// four hundred times to move one multiplier.
	s.slider.OnChanged = func(v float64) { s.entry.SetText(o.Format(v)) }
	s.slider.OnChangeEnded = s.commit
	s.entry.OnSubmitted = func(text string) {
		v, err := strconv.ParseFloat(strings.TrimSpace(text), 64)
		if err != nil {
			if o.OnInvalid != nil {
				o.OnInvalid(text)
			}
			s.entry.SetText(o.Format(s.value))
			return
		}
		s.commit(v)
	}
	s.body = container.NewBorder(nil, nil, nil, widgets.FixedWidth(s.entry, o.EntryWidth), s.slider)
	return s
}

// commit clamps, records and reports one value, and brings both controls to it.
func (s *SliderEntry) commit(v float64) {
	v = min(max(v, s.opts.Min), s.opts.Max)
	s.Set(v)
	if s.opts.Commit != nil {
		s.opts.Commit(v)
	}
}

// Set moves both controls to v without committing, for Reset and for a value
// the program adjusted after the commit.
func (s *SliderEntry) Set(v float64) {
	s.value = v
	if text := s.opts.Format(v); s.entry.Text != text {
		s.entry.SetText(text)
	}
	if s.slider.Value != v {
		s.slider.Value = v
		s.slider.Refresh()
	}
}

// Value is the last committed or set value.
func (s *SliderEntry) Value() float64 { return s.value }

// Widget is the pair laid out: the slider filling, the entry on the right.
func (s *SliderEntry) Widget() fyne.CanvasObject { return s.body }

// Slider and Entry expose the controls for tests and for Gate.
func (s *SliderEntry) Slider() *widget.Slider { return s.slider }

// Entry is the pair's text field.
func (s *SliderEntry) Entry() *widget.Entry { return s.entry }
