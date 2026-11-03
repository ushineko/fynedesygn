package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const ellipsis = "…"

/*
fitLabel is a one-line label that cuts its own text to the width it is given,
with an ellipsis, the way a label with TextTruncateEllipsis does.

Fyne's truncation shapes the whole text again through harfbuzz at every
layout, whether or not it fits (Fyne quirk 30). A window drag lays out every
row at every frame, so a card of PlainRows spent half the process's CPU on
text that had not changed. The label here keeps wrapping and truncation off,
which is the path in Fyne's rich text that measures nothing, and measures the
text itself only when the width changes. A drag that changes only the height
does no text work, and a text that fits costs one cached measurement.

Its minimum width is zero, as a truncating label's is, so the text it holds
cannot set the window's width.
*/
type fitLabel struct {
	widget.BaseWidget
	label *widget.Label
	full  string

	// The width the text was last cut for, and the text size and full width
	// it was measured at, so an unchanged layout costs nothing.
	cutFor     float32
	measuredAt float32
	fullWidth  float32
}

func newFitLabel(text string, importance widget.Importance) *fitLabel {
	l := widget.NewLabel(text)
	l.Importance = importance
	l.Wrapping = fyne.TextWrapOff
	l.Truncation = fyne.TextTruncateOff
	f := &fitLabel{label: l, full: text, cutFor: -1}
	f.ExtendBaseWidget(f)
	return f
}

func (f *fitLabel) CreateRenderer() fyne.WidgetRenderer {
	return &fitLabelRenderer{f: f}
}

// fit cuts the text for width, or restores it whole when it fits.
func (f *fitLabel) fit(width float32) {
	size := theme.SizeForWidget(theme.SizeNameText, f)
	if width == f.cutFor && size == f.measuredAt {
		return
	}
	f.cutFor = width
	if size != f.measuredAt {
		f.measuredAt = size
		f.fullWidth = f.measure(f.full, size)
	}
	text := f.full
	if f.fullWidth > width {
		text = f.cut(width, size)
	}
	if text != f.label.Text {
		f.label.Text = text
		f.label.Refresh()
	}
}

// cut finds the longest prefix that fits with the ellipsis after it.
func (f *fitLabel) cut(width, size float32) string {
	runes := []rune(f.full)
	lo, hi := 0, len(runes)
	for lo < hi {
		mid := (lo + hi + 1) / 2
		if f.measure(string(runes[:mid])+ellipsis, size) <= width {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	if lo == 0 {
		return ellipsis
	}
	return string(runes[:lo]) + ellipsis
}

// measure is the width the label needs to draw s on one line, padding included.
func (f *fitLabel) measure(s string, size float32) float32 {
	pad := theme.SizeForWidget(theme.SizeNameInnerPadding, f)
	return fyne.MeasureText(s, size, f.label.TextStyle).Width + 2*pad
}

type fitLabelRenderer struct{ f *fitLabel }

func (r *fitLabelRenderer) Layout(size fyne.Size) {
	r.f.fit(size.Width)
	r.f.label.Resize(size)
}

func (r *fitLabelRenderer) MinSize() fyne.Size {
	return fyne.NewSize(0, r.f.label.MinSize().Height)
}

func (r *fitLabelRenderer) Refresh() {
	// A theme change can change the text size, so the old widths are not
	// trusted. Before the first layout there is no width to cut for.
	r.f.measuredAt = 0
	r.f.cutFor = -1
	if w := r.f.Size().Width; w > 0 {
		r.f.fit(w)
	}
	r.f.label.Refresh()
}

func (r *fitLabelRenderer) Objects() []fyne.CanvasObject { return []fyne.CanvasObject{r.f.label} }

func (r *fitLabelRenderer) Destroy() {}
