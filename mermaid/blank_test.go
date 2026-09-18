package mermaid

import (
	"image"
	"image/color"
)

// blank is a solid image of the given size for tests.
func blank(w, h int) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for i := range img.Pix {
		img.Pix[i] = 0x80
	}
	_ = color.Transparent
	return img
}
