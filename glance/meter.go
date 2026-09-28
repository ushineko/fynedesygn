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
	themed

	label   *canvas.Text
	caption *canvas.Text
	bar     *bar
	box     *fyne.Container

	// trailing is the header's right-hand value, against the meter's own
	// width. Empty for a meter that does not use one, and its row lays out
	// exactly as it did before this existed.
	trailing *canvas.Text

	// statsLeft and statsRight are the row under the bar. The container is
	// kept so the row can be taken away entirely when both are empty: an
	// empty row is a gap, and a gap under one meter in a stack of them is
	// read as a missing reading.
	statsLeft  *canvas.Text
	statsRight *canvas.Text
	stats      *fyne.Container

	status fd.Status
}

// NewMeter builds a meter with a label and nothing filled. labelWidth pins the
// label column so several meters stacked together line their captions up; 0
// lets each label take its own width.
func NewMeter(label string, labelWidth float32) *Meter {
	m := &Meter{
		label:   canvas.NewText(label, nil),
		caption: canvas.NewText("", nil),
		bar:     newBar(),
	}
	m.label.Color = m.colour(theme.ColorNameForeground)
	m.caption.Color = m.colour(theme.ColorNameDisabled)
	m.label.TextSize = m.textSize()
	m.caption.TextSize = m.size(theme.SizeNameCaptionText)
	// Monospace, for the same reason a Row's value is. Padding a caption to a
	// fixed character count only holds its width if the characters are the
	// same width as each other: in a proportional face a space is narrower
	// than a digit, so Percent's "  9 %" is visibly narrower than "100 %" and
	// the meter's minimum width — and the window's — moves with the value.
	m.caption.TextStyle = fyne.TextStyle{Monospace: true}

	// The trailing value and the stats share the caption's face and size for
	// the reason the caption has them: they are changing values in a column,
	// and in a proportional face a space is narrower than a digit, so a value
	// that changes moves the one opposite it.
	m.trailing = m.detailText()
	m.statsLeft = m.detailText()
	m.statsRight = m.detailText()

	var head fyne.CanvasObject = m.label
	if labelWidth > 0 {
		head = widgets.FixedWidth(m.label, labelWidth)
	}

	// The spacer between the caption and the trailing value is what makes the
	// meter as wide as its widest *pair* rather than as wide as everything in
	// it laid end to end. That is the whole of this spec: the archetype's own
	// rows are a widget, a stretch and a widget, and three figures laid out
	// that way cost the width of the longest two.
	top := container.NewHBox(head, m.caption, layout.NewSpacer(), m.trailing)

	m.stats = container.NewHBox(m.statsLeft, layout.NewSpacer(), m.statsRight)
	m.stats.Hide()

	m.box = container.NewVBox(top, m.bar, m.stats)
	return m
}

// detailText builds one of the quiet values: the trailing header value, or an
// end of the stats row.
func (m *Meter) detailText() *canvas.Text {
	t := canvas.NewText("", m.colour(theme.ColorNameDisabled))
	t.TextSize = m.size(theme.SizeNameCaptionText)
	t.TextStyle = fyne.TextStyle{Monospace: true}
	return t
}

/*
SetTrailing puts a value at the right-hand end of the meter's header.

It is where the archetype puts the reset — the one figure a reader looks for in
the same place on every line, which is exactly what a column is for and what a
sentence is not.

It is **held to the no-jitter rule**, like the caption: format it through
Percent, Quantity or Pad. A trailing value that changes width does not merely
move itself, it drags the caption opposite it, which is worse than a wide
meter because it moves while being read.

Empty removes it.
*/
func (m *Meter) SetTrailing(s string) {
	m.trailing.Text = s
	m.trailing.Color = m.colour(theme.ColorNameDisabled)
	m.trailing.Refresh()
}

/*
SetStats puts a value at each end of a row under the bar.

The figures that would otherwise lengthen the caption go here. Both empty
takes the row away rather than leaving it blank: a gap under one meter in a
stack of them reads as a reading that failed.

Held to the no-jitter rule, for the reason SetTrailing gives.
*/
func (m *Meter) SetStats(left, right string) {
	m.statsLeft.Text = left
	m.statsRight.Text = right
	m.statsLeft.Color = m.colour(theme.ColorNameDisabled)
	m.statsRight.Color = m.colour(theme.ColorNameDisabled)

	if left == "" && right == "" {
		m.stats.Hide()
	} else {
		m.stats.Show()
	}
	m.statsLeft.Refresh()
	m.statsRight.Refresh()
}

// Trailing is the header's right-hand value, for a test that would rather ask.
func (m *Meter) Trailing() string { return m.trailing.Text }

// Stats are the two ends of the row under the bar.
func (m *Meter) Stats() (left, right string) { return m.statsLeft.Text, m.statsRight.Text }

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
	m.caption.Color = m.colour(theme.ColorNameDisabled)
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
// SetTheme gives the meter and its bar a theme of their own. A panel calls it.
func (m *Meter) SetTheme(th fyne.Theme) {
	m.themed.SetTheme(th)
	if m.bar != nil {
		m.bar.SetTheme(th)
	}
	// And repaint in it. A canvas.Text holds a literal size and colour rather
	// than asking the theme when it draws, so being told is not enough.
	m.Restyle()
}

func (m *Meter) Restyle() {
	m.label.TextSize = m.textSize()
	m.label.Color = m.colour(theme.ColorNameForeground)
	m.caption.TextSize = m.size(theme.SizeNameCaptionText)
	m.caption.Color = m.colour(theme.ColorNameDisabled)
	m.caption.TextStyle = fyne.TextStyle{Monospace: true}
	m.bar.set(m.bar.fraction, m.fillColour())

	for _, t := range []*canvas.Text{m.trailing, m.statsLeft, m.statsRight} {
		t.TextSize = m.size(theme.SizeNameCaptionText)
		t.Color = m.colour(theme.ColorNameDisabled)
		t.TextStyle = fyne.TextStyle{Monospace: true}
		t.Refresh()
	}

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
	themed

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
	r.track = canvas.NewRectangle(b.colour(theme.ColorNameDisabledButton))
	r.track.CornerRadius = BarRadius
	r.value = canvas.NewRectangle(b.colour(b.fill))
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
	r.track.FillColor = r.bar.colour(theme.ColorNameDisabledButton)
	r.value.FillColor = r.bar.colour(r.bar.fill)
	r.Layout(r.size)
	r.track.Refresh()
	r.value.Refresh()
}

func (r *barRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.track, r.value}
}

func (r *barRenderer) Destroy() {}
