package glance

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	fd "github.com/ushineko/fynedesygn"
	"github.com/ushineko/fynedesygn/widgets"
)

// BarHeight is the height of a meter's bar. Small: it is a proportion, and the
// number beside it is what anyone reads precisely.
const BarHeight float32 = 8

// BarRadius rounds the bar's ends.
const BarRadius float32 = 4

// Meter is a proportion of something that has a limit: a quota used, a budget
// spent, a window of time elapsed. A short label, a caption that carries the
// detail, and a bar coloured by how close to the limit it is.
//
// It is the one place in this package where a bar is the right answer. A
// reading with no limit — a temperature, a rate — has nothing to fill up, and
// drawing it as a bar invents a maximum. The busy indicator from the design
// system is not this either: that one is indeterminate on purpose because a
// bar that filled steadily would be inventing a number. Here the number is
// real.
//
// The caption does the work the bar cannot. A bar says "most of it"; the
// caption says which window, how much of it, and when it resets — the things
// someone glancing at the panel actually wants. So the caption is not optional
// decoration and the bar is not the primary reading.
//
// **The caption is a changing value and is held to the same rule as any
// other.** It sets the meter's minimum width, so a caption written with
// fmt.Sprintf("%.0f%%", pct) goes from "5%" to "100%" and takes the window's
// width with it. Format it through Percent, Quantity or Pad, the same as a
// row's value.
type Meter struct {
	label   *canvas.Text
	caption *canvas.Text
	bar     *bar
	box     *fyne.Container

	status fd.Status
}

// NewMeter builds a meter with a label and nothing filled. labelWidth pins the
// label column so several meters stacked together line their captions up; 0
// lets each label take its own width.
func NewMeter(label string, labelWidth float32) *Meter {
	m := &Meter{
		label:   canvas.NewText(label, theme.Color(theme.ColorNameForeground)),
		caption: canvas.NewText("", theme.Color(theme.ColorNameDisabled)),
		bar:     newBar(),
	}
	m.label.TextSize = theme.TextSize()
	m.caption.TextSize = theme.Size(theme.SizeNameCaptionText)
	// Monospace, for the same reason a Row's value is. Padding a caption to a
	// fixed character count only holds its width if the characters are the
	// same width as each other: in a proportional face a space is narrower
	// than a digit, so Percent's "  9 %" is visibly narrower than "100 %" and
	// the meter's minimum width — and the window's — moves with the value.
	m.caption.TextStyle = fyne.TextStyle{Monospace: true}

	var head fyne.CanvasObject = container.NewHBox(m.label, m.caption, layout.NewSpacer())
	if labelWidth > 0 {
		head = container.NewHBox(
			widgets.FixedWidth(m.label, labelWidth), m.caption, layout.NewSpacer())
	}
	m.box = container.NewVBox(head, m.bar)
	return m
}

// Set fills the meter. fraction is 0..1 and is clamped, because a bar wider
// than its track is a bar that has left the layout. caption is the detail line
// and st colours the fill.
//
// Usage against a quota is one of the things that does have a threshold, so
// grading it is a signal rather than decoration (docs/glance.md, Colour is the
// legend).
func (m *Meter) Set(fraction float64, caption string, st fd.Status) {
	m.status = st
	m.caption.Text = caption
	m.caption.Color = theme.Color(theme.ColorNameDisabled)
	m.bar.set(clamp01(fraction), m.fillColour())
	m.caption.Refresh()
}

// SetLabel replaces the meter's label.
func (m *Meter) SetLabel(s string) {
	m.label.Text = s
	m.label.Refresh()
}

// Fraction is how full the meter is, for a test that would rather ask than
// measure a rectangle.
func (m *Meter) Fraction() float64 { return m.bar.fraction }

// Caption is the detail line the meter currently shows.
func (m *Meter) Caption() string { return m.caption.Text }

// Status is the verdict the fill is drawn in.
func (m *Meter) Status() fd.Status { return m.status }

// fillColour maps the verdict onto a theme role. An ungraded meter takes the
// scheme's primary rather than the foreground: a bar drawn in the text colour
// reads as a block of text.
func (m *Meter) fillColour() fyne.ThemeColorName {
	switch m.status {
	case fd.StatusGood:
		return theme.ColorNameSuccess
	case fd.StatusWarn:
		return theme.ColorNameWarning
	case fd.StatusBad:
		return theme.ColorNameError
	default:
		return theme.ColorNamePrimary
	}
}

// Restyle repaints the meter in the current theme.
func (m *Meter) Restyle() {
	m.label.TextSize = theme.TextSize()
	m.label.Color = theme.Color(theme.ColorNameForeground)
	m.caption.TextSize = theme.Size(theme.SizeNameCaptionText)
	m.caption.Color = theme.Color(theme.ColorNameDisabled)
	m.caption.TextStyle = fyne.TextStyle{Monospace: true}
	m.bar.set(m.bar.fraction, m.fillColour())
	m.label.Refresh()
	m.caption.Refresh()
}

// Object is the meter's content.
func (m *Meter) Object() fyne.CanvasObject { return m.box }

// clamp01 holds a fraction inside the track.
func clamp01(v float64) float64 {
	switch {
	case v < 0:
		return 0
	case v > 1:
		return 1
	default:
		return v
	}
}

// bar is the track and its fill. It is a widget rather than two rectangles in
// a container because the fill's width is a proportion of the track's, which
// only the layout knows.
type bar struct {
	widget.BaseWidget

	fraction float64
	fill     fyne.ThemeColorName
}

func newBar() *bar {
	b := &bar{fill: theme.ColorNamePrimary}
	b.ExtendBaseWidget(b)
	return b
}

func (b *bar) set(fraction float64, fill fyne.ThemeColorName) {
	b.fraction, b.fill = fraction, fill
	b.Refresh()
}

// MinSize reserves the bar's height and no width: the card decides how wide
// the panel is.
func (b *bar) MinSize() fyne.Size { return fyne.NewSize(0, BarHeight) }

func (b *bar) CreateRenderer() fyne.WidgetRenderer {
	r := &barRenderer{bar: b}
	r.track = canvas.NewRectangle(theme.Color(theme.ColorNameDisabledButton))
	r.track.CornerRadius = BarRadius
	r.value = canvas.NewRectangle(theme.Color(b.fill))
	r.value.CornerRadius = BarRadius
	return r
}

type barRenderer struct {
	bar   *bar
	track *canvas.Rectangle
	value *canvas.Rectangle
	size  fyne.Size
}

func (r *barRenderer) Layout(size fyne.Size) {
	r.size = size
	r.track.Resize(size)
	r.value.Resize(fyne.NewSize(size.Width*float32(r.bar.fraction), size.Height))
}

func (r *barRenderer) MinSize() fyne.Size { return r.bar.MinSize() }

func (r *barRenderer) Refresh() {
	r.track.FillColor = theme.Color(theme.ColorNameDisabledButton)
	r.value.FillColor = theme.Color(r.bar.fill)
	r.Layout(r.size)
	r.track.Refresh()
	r.value.Refresh()
}

func (r *barRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.track, r.value}
}

func (r *barRenderer) Destroy() {}
