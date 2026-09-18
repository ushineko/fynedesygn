package mermaid

import (
	"bytes"
	"image"
	_ "image/png" // the diagrams are PNG

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
)

// Diagram draws a rendered diagram contained in the width it is given, at
// its natural size divided by the render scale when there is room. Its
// minimum height follows from its width, so a document pane can measure it
// like any other block.
type Diagram struct {
	widget.BaseWidget
	img     *canvas.Image
	natural fyne.Size // at 1x
	width   float32
}

// NewDiagram wraps a PNG resource rendered at scale.
func NewDiagram(res fyne.Resource, scale float32) *Diagram {
	d := &Diagram{img: canvas.NewImageFromResource(res)}
	d.img.FillMode = canvas.ImageFillContain
	d.img.ScaleMode = canvas.ImageScaleSmooth
	if scale <= 0 {
		scale = 1
	}
	if cfg, _, err := image.DecodeConfig(bytes.NewReader(res.Content())); err == nil {
		d.natural = fyne.NewSize(float32(cfg.Width)/scale, float32(cfg.Height)/scale)
	}
	d.ExtendBaseWidget(d)
	return d
}

// CreateRenderer implements fyne.Widget.
func (d *Diagram) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(d.img)
}

// Resize records the width so MinSize can derive the height.
func (d *Diagram) Resize(s fyne.Size) {
	d.width = s.Width
	d.BaseWidget.Resize(s)
}

// MinSize is the natural height, reduced in proportion when the width given
// is narrower than the natural width. Zero when the image could not be read.
func (d *Diagram) MinSize() fyne.Size {
	if d.natural.Width <= 0 || d.natural.Height <= 0 {
		return fyne.NewSize(0, 0)
	}
	h := d.natural.Height
	if d.width > 0 && d.width < d.natural.Width {
		h = d.natural.Height * d.width / d.natural.Width
	}
	return fyne.NewSize(0, h)
}

// Natural is the diagram's size at 1x.
func (d *Diagram) Natural() fyne.Size { return d.natural }
