package glance

import (
	"image/color"

	"fyne.io/fyne/v2"
	fynetheme "fyne.io/fyne/v2/theme"
)

// The design system's categorical set: the second, third and fourth series
// colours, after the scheme's link colour. They are fixed rather than read from
// the palette because the palette cannot supply them. Its status-free tokens
// are too few and too alike: the selection accent is StatusInfo's colour,
// positive is StatusGood's, hover equals the selection accent in the Breeze
// and Adwaita schemes, and link equals it in the Windows pair. Violet, teal
// and magenta are hues no status takes (blue, green, amber, red), so a series
// never reads as a verdict. Each comes in two weights: lighter and more
// saturated on a dark background, darker on a light one, chosen against Breeze
// Dark and Breeze Light.
var (
	// Violet, teal and magenta on a dark background.
	seriesDark = [3]color.NRGBA{
		{R: 0xb1, G: 0x97, B: 0xfc, A: 0xff}, // violet  #b197fc
		{R: 0x4f, G: 0xd1, B: 0xc5, A: 0xff}, // teal    #4fd1c5
		{R: 0xe0, G: 0x7b, B: 0xd0, A: 0xff}, // magenta #e07bd0
	}
	// The same three on a light background.
	seriesLight = [3]color.NRGBA{
		{R: 0x7c, G: 0x4d, B: 0xbd, A: 0xff}, // violet  #7c4dbd
		{R: 0x0f, G: 0x8a, B: 0x80, A: 0xff}, // teal    #0f8a80
		{R: 0xb0, G: 0x36, B: 0x9a, A: 0xff}, // magenta #b0369a
	}
)

// seriesCycle is how many colours SeriesColour has before it wraps.
const seriesCycle = 1 + len(seriesDark)

// SeriesColour is the colour of the i-th trace of a plot whose traces are
// told apart by colour rather than by the colour of a row's value. Series 0 is
// the theme's link colour, so the first trace follows the scheme's accent;
// series 1, 2 and 3 are the design system's categorical violet, teal and
// magenta, in the weight that suits the theme's background; the fifth wraps
// to the first. None of them is a status colour. A nil th is the application's
// theme.
func SeriesColour(th fyne.Theme, i int) color.Color {
	i %= seriesCycle
	if i < 0 {
		i += seriesCycle
	}
	t := &themed{th: th}
	if i == 0 {
		// Opaque through NRGBA, in case a theme's link is translucent;
		// theme.Alpha would read the premultiplied channels and darken it.
		c, _ := color.NRGBAModel.Convert(t.colour(fynetheme.ColorNameHyperlink)).(color.NRGBA)
		c.A = 0xff
		return c
	}
	if isDark(t.colour(fynetheme.ColorNameBackground)) {
		return seriesDark[i-1]
	}
	return seriesLight[i-1]
}

// isDark reports whether a background is dark, by its Rec. 709 luma. The
// theme's variant is not asked: the design system's themes ignore it, and a
// glance panel's background is its own.
func isDark(c color.Color) bool {
	r, g, b, _ := color.NRGBAModel.Convert(c).RGBA()
	luma := 0.2126*float64(r) + 0.7152*float64(g) + 0.0722*float64(b)
	return luma < 0.5*0xffff
}

// fadedAlpha is the share of its alpha a faded colour keeps. Half is the
// point at which the secondary line of a pair (an interface's upload beside
// its download) is plainly the lesser of the two at 1.5 px on a dark or a
// light card, and still plainly the same hue, which is what makes it read as
// that interface's rather than as another trace.
const fadedAlpha = 0.5

// Faded is c with its alpha reduced to fadedAlpha of what it was, keeping the
// hue, for the secondary trace drawn as a pair with a primary one in c.
func Faded(c color.Color) color.Color {
	n, _ := color.NRGBAModel.Convert(c).(color.NRGBA)
	n.A = uint8(float64(n.A) * fadedAlpha)
	return n
}
