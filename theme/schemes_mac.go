package theme

/*
The macOS palettes are transcribed from Apple's dynamic system colours as
documented in the Human Interface Guidelines and AppKit (windowBackgroundColor,
textBackgroundColor, alternatingContentBackgroundColors, controlColor,
labelColor, placeholderTextColor, separatorColor, linkColor, and systemBlue,
systemRed, systemGreen and systemOrange for the accent and status triple).

AppKit gives several of these as an alpha over whatever is behind them; the
values here are composited over the window or view colour because Fyne asks for
an opaque fill. The accent is the default blue; a user's chosen accent colour
is not read.

The corner radius is the one visible concession to macOS 26 Tahoe's Liquid
Glass: its controls are rounder than the Big Sur through Sequoia style. Fyne
cannot draw translucency or refraction, so roundness and the system colours
are what carry the style.
*/
const (
	macRadius  float32 = 8
	macPadding float32 = 4
)

// MacOSDark is the macOS dark appearance.
var MacOSDark = Palette{
	Name: "macOS Dark", Dark: true,
	WindowBG: RGB(50, 50, 50), WindowFG: RGB(224, 224, 224),
	ViewBG: RGB(30, 30, 30), ViewAltBG: RGB(42, 42, 42), ViewFG: RGB(224, 224, 224),
	ButtonBG: RGB(101, 101, 101), ButtonFG: RGB(224, 224, 224),
	SelectionBG: RGB(10, 132, 255), SelectionFG: RGB(255, 255, 255),
	TooltipBG:  RGB(60, 60, 60),
	InactiveFG: RGB(110, 110, 110),
	Negative:   RGB(255, 69, 58), Positive: RGB(48, 209, 88), Neutral: RGB(255, 159, 10),
	Link: RGB(65, 156, 255), Focus: RGB(10, 132, 255), Hover: RGB(255, 255, 255),
	Separator: RGB(70, 70, 70),
	Radius:    macRadius, Padding: macPadding,
}

// MacOSLight is the macOS light appearance.
var MacOSLight = Palette{
	Name: "macOS Light", Dark: false,
	WindowBG: RGB(236, 236, 236), WindowFG: RGB(38, 38, 38),
	ViewBG: RGB(255, 255, 255), ViewAltBG: RGB(244, 245, 245), ViewFG: RGB(38, 38, 38),
	ButtonBG: RGB(255, 255, 255), ButtonFG: RGB(38, 38, 38),
	SelectionBG: RGB(0, 122, 255), SelectionFG: RGB(255, 255, 255),
	TooltipBG:  RGB(240, 240, 240),
	InactiveFG: RGB(172, 172, 172),
	Negative:   RGB(255, 59, 48), Positive: RGB(52, 199, 89), Neutral: RGB(255, 149, 0),
	Link: RGB(0, 104, 218), Focus: RGB(0, 122, 255), Hover: RGB(0, 0, 0),
	Separator: RGB(212, 212, 212),
	Radius:    macRadius, Padding: macPadding,
}
