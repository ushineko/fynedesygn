package widgets

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
)

/*
Swatch is a block of colour somebody can click.

It exists because the obvious way to build one -- a `canvas.Rectangle` with an
invisible `widget.Button` stacked over it -- is five objects and an expensive
measurement, and a palette is hundreds of them.

A profile of hotaru's scene editor being resized put `buttonRenderer.MinSize`
at 14% of all samples: each call does a theme lookup, a padding calculation
and a `RichText.MinSize` for a label that is the empty string. A scroller lays
out everything it holds, so every drag step paid that for every block on the
page.

This is one widget, and its minimum size is the number it was given.
*/
type Swatch struct {
	widget.BaseWidget

	// Fill is the colour of the block.
	Fill color.Color
	// Stroke and StrokeWidth are its border, which is how a selected swatch
	// says so.
	Stroke      color.Color
	StrokeWidth float32
	// OnTapped is called when somebody clicks it. Nil makes it a block of
	// colour that does nothing, which is a legitimate thing to want.
	OnTapped func()

	size fyne.Size
	rect *canvas.Rectangle
}

// NewSwatch is a swatch of a fixed size. The size is the point: a palette is
// a grid of identical blocks, and a block that measured itself from its
// contents would have none.
func NewSwatch(size fyne.Size) *Swatch {
	s := &Swatch{size: size}
	s.ExtendBaseWidget(s)
	return s
}

// MinSize is the size it was made with. No theme lookup, no padding, no text
// to shape: this is the whole reason the type exists.
func (s *Swatch) MinSize() fyne.Size { return s.size }

// Tapped implements fyne.Tappable.
func (s *Swatch) Tapped(*fyne.PointEvent) {
	if s.OnTapped != nil {
		s.OnTapped()
	}
}

// CreateRenderer implements fyne.Widget.
func (s *Swatch) CreateRenderer() fyne.WidgetRenderer {
	s.rect = canvas.NewRectangle(s.Fill)
	s.rect.StrokeColor = s.Stroke
	s.rect.StrokeWidth = s.StrokeWidth
	return widget.NewSimpleRenderer(s.rect)
}

/*
Refresh redraws the block after its colours have been set.

The fields are public and a caller sets them directly, which is the shape a
palette wants: build the grid once and recolour it, rather than rebuilding a
grid of widgets every time one of them changes.
*/
func (s *Swatch) Refresh() {
	if s.rect != nil {
		s.rect.FillColor = s.Fill
		s.rect.StrokeColor = s.Stroke
		s.rect.StrokeWidth = s.StrokeWidth
		s.rect.Refresh()
	}
	s.BaseWidget.Refresh()
}
