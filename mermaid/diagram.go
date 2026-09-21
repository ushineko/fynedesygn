package mermaid

import (
	"bytes"
	"fmt"
	"image"
	_ "image/png" // the diagrams are PNG

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"

	"github.com/ushineko/fynedesygn/imagecache"
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

/*
NewDiagram wraps a PNG resource rendered at scale.

**Decoded through the shared cache, and drawn from the decoded image.** A
`canvas.Image` built from a resource decodes it, and decodes it again on
every refresh -- so a document pane, which measures its blocks by rendering
them, decoded a 1246x2186 diagram at 10.4 MB a time, once per measure and
once per rebuild of the section it was in. Twelve copies were measured alive
at once in hotaru.

The resource's name is the key, which is what makes this safe here: a
diagram's name is a hash of the text it was drawn from (see Set), so a
changed diagram is a different key rather than a stale picture.
*/
func NewDiagram(res fyne.Resource, scale float32) *Diagram {
	decoded, err := imagecache.Shared.Get(res.Name(), func() (image.Image, error) {
		img, _, err := image.Decode(bytes.NewReader(res.Content()))
		if err != nil {
			return nil, fmt.Errorf("decode %s: %w", res.Name(), err)
		}
		return img, nil
	})

	if err != nil {
		// Unreadable: fall back to the resource, so a diagram this package
		// cannot decode still draws whatever Fyne can make of it.
		d := &Diagram{img: canvas.NewImageFromResource(res)}
		d.img.FillMode = canvas.ImageFillContain
		d.img.ScaleMode = canvas.ImageScaleSmooth
		d.ExtendBaseWidget(d)
		return d
	}

	d := &Diagram{img: canvas.NewImageFromImage(decoded)}
	d.img.FillMode = canvas.ImageFillContain
	d.img.ScaleMode = canvas.ImageScaleSmooth
	if scale <= 0 {
		scale = 1
	}
	bounds := decoded.Bounds()
	d.natural = fyne.NewSize(float32(bounds.Dx())/scale, float32(bounds.Dy())/scale)
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
