package theme

import (
	"image/color"

	"fyne.io/fyne/v2"
	fynetheme "fyne.io/fyne/v2/theme"
)

// WithTransparentBackground returns base with one role changed: the window
// background is fully transparent.
//
// That role is what Fyne's GL painter clears each frame with, alpha included
// (internal/painter/gl/painter.go), so a theme wrapped in this one gives a
// window whose unpainted area is see-through. It is half of what a translucent
// window needs; the other half is a framebuffer with an alpha channel, which
// is a window-creation hint and therefore not something a theme can ask for.
// glance.Options.Translucent asks for both, in the right order.
//
// **Do not wrap a theme with this unless the hint was granted.** A window
// whose framebuffer has no alpha clears to *black*, not to the desktop, and a
// black rectangle is worse than an opaque panel. glance probes for the grant
// before it wraps.
//
// Only ColorNameBackground changes. OverlayBackground in particular does not:
// it is the fill of a popup menu, which has to stay opaque or the only
// interface a glance window has becomes invisible.
func WithTransparentBackground(base fyne.Theme) fyne.Theme {
	return transparentBackground{base}
}

type transparentBackground struct{ fyne.Theme }

func (t transparentBackground) Color(n fyne.ThemeColorName, v fyne.ThemeVariant) color.Color {
	if n == fynetheme.ColorNameBackground {
		return color.NRGBA{}
	}
	return t.Theme.Color(n, v)
}
