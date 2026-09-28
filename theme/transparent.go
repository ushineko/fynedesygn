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

/*
KeepTransparentBackground returns next, wrapped when the application's current
background is already clear.

A transparent application background is what makes a glance panel's gaps alpha
0 -- Fyne clears every window's framebuffer from that one role -- and anything
that sets the app's theme afterwards would put an opaque background back and
fill the panel in. That is not hypothetical: opening a second window is enough,
because a window that does not draw in its own appearance sets the
application's theme as it is built.

So every place in this library that sets the app theme goes through here, and
the wrap survives. Detected rather than remembered: the window that asked for
it is a different one and may not exist yet, so an alpha of zero on the current
background is the whole of the signal. A program that never had a translucent
window is unaffected, because its background was never clear.

**A consumer that sets the app's theme itself has the same problem** and the
same answer: wrap with this, or let shell.SetAppearance do it.
*/
func KeepTransparentBackground(app fyne.App, next fyne.Theme) fyne.Theme {
	if app == nil {
		return next
	}
	current := app.Settings().Theme()
	if current == nil {
		return next
	}
	_, _, _, alpha := current.Color(
		fynetheme.ColorNameBackground, app.Settings().ThemeVariant()).RGBA()
	if alpha == 0 {
		return WithTransparentBackground(next)
	}
	return next
}
