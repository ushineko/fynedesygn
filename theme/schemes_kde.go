package theme

// The KDE and GNOME size tokens. Breeze is a tighter, squarer style than
// Fyne's default; pulling the corner radius and padding in is what stops the
// window reading as "Fyne wearing KDE colours". The Adwaita palettes keep the
// same tokens because the three programs this package came from drew them
// that way and their users are used to it.
const (
	kdeRadius  float32 = 2
	kdePadding float32 = 3
)

// BreezeDark is /usr/share/color-schemes/BreezeDark.colors.
var BreezeDark = Palette{
	Name: "Breeze Dark", Dark: true,
	WindowBG: RGB(32, 35, 38), WindowFG: RGB(252, 252, 252),
	ViewBG: RGB(20, 22, 24), ViewAltBG: RGB(29, 31, 34), ViewFG: RGB(252, 252, 252),
	ButtonBG: RGB(41, 44, 48), ButtonFG: RGB(252, 252, 252),
	SelectionBG: RGB(61, 174, 233), SelectionFG: RGB(252, 252, 252),
	TooltipBG:  RGB(41, 44, 48),
	InactiveFG: RGB(161, 169, 177),
	Negative:   RGB(218, 68, 83), Positive: RGB(39, 174, 96), Neutral: RGB(246, 116, 0),
	Link: RGB(29, 153, 243), Focus: RGB(61, 174, 233), Hover: RGB(61, 174, 233),
	Separator: RGB(60, 64, 69),
	Radius:    kdeRadius, Padding: kdePadding,
}

// BreezeLight is /usr/share/color-schemes/BreezeLight.colors.
var BreezeLight = Palette{
	Name: "Breeze Light", Dark: false,
	WindowBG: RGB(239, 240, 241), WindowFG: RGB(35, 38, 41),
	ViewBG: RGB(255, 255, 255), ViewAltBG: RGB(247, 247, 247), ViewFG: RGB(35, 38, 41),
	ButtonBG: RGB(252, 252, 252), ButtonFG: RGB(35, 38, 41),
	SelectionBG: RGB(61, 174, 233), SelectionFG: RGB(255, 255, 255),
	TooltipBG:  RGB(247, 247, 247),
	InactiveFG: RGB(112, 125, 138),
	Negative:   RGB(218, 68, 83), Positive: RGB(39, 174, 96), Neutral: RGB(246, 116, 0),
	Link: RGB(41, 128, 185), Focus: RGB(61, 174, 233), Hover: RGB(61, 174, 233),
	Separator: RGB(200, 203, 207),
	Radius:    kdeRadius, Padding: kdePadding,
}

// OxygenDark is /usr/share/color-schemes/OxygenDark.colors. Oxygen is the warm
// scheme: its greys carry red, its foregrounds are off-white towards peach, and
// its selection is orange rather than blue. Transcribing it faithfully rather
// than tinting Breeze Dark is the point: a scheme that reads as "the same dark
// grey again" would not show whether Fyne can carry a scheme at all.
var OxygenDark = Palette{
	Name: "Oxygen Dark", Dark: true,
	WindowBG: RGB(38, 36, 35), WindowFG: RGB(255, 232, 223),
	ViewBG: RGB(30, 29, 29), ViewAltBG: RGB(33, 31, 30), ViewFG: RGB(255, 230, 222),
	ButtonBG: RGB(57, 53, 50), ButtonFG: RGB(255, 234, 225),
	SelectionBG: RGB(247, 159, 82), SelectionFG: RGB(38, 36, 35),
	TooltipBG:  RGB(24, 21, 19),
	InactiveFG: RGB(137, 136, 135),
	// Oxygen's own negative/positive are very dark (191,3,3 / 0,110,40) because
	// the scheme was authored for a light window. On the dark variant they are
	// close to unreadable against the view background, so they are lightened
	// here. This is the one place the transcription is not literal, and it is a
	// legibility fix, not a taste preference.
	Negative: RGB(232, 87, 82), Positive: RGB(94, 189, 110), Neutral: RGB(226, 170, 61),
	Link: RGB(88, 172, 255), Focus: RGB(240, 213, 194), Hover: RGB(255, 229, 208),
	Separator: RGB(64, 60, 57),
	Radius:    kdeRadius, Padding: kdePadding,
}

// Adwaita is GNOME's scheme rather than KDE's, and unlike the three above it is
// not transcribed from a file on disk: libadwaita ships named colours compiled
// into the library, not a .colors document. These values are its documented
// named colours (window_bg_color, view_bg_color, accent_bg_color, and the
// destructive/success/warning triple). Where libadwaita specifies a translucent
// colour over another (its buttons are black at 5% over the window) the
// composited result is used, because Fyne asks for an opaque fill there.

// AdwaitaLight is libadwaita's light scheme.
var AdwaitaLight = Palette{
	Name: "Adwaita Light", Dark: false,
	WindowBG: RGB(250, 250, 250), WindowFG: RGB(46, 52, 54),
	ViewBG: RGB(255, 255, 255), ViewAltBG: RGB(246, 245, 244), ViewFG: RGB(46, 52, 54),
	ButtonBG: RGB(239, 239, 239), ButtonFG: RGB(46, 52, 54),
	SelectionBG: RGB(53, 132, 228), SelectionFG: RGB(255, 255, 255),
	TooltipBG:  RGB(53, 57, 60),
	InactiveFG: RGB(146, 149, 149),
	Negative:   RGB(224, 27, 36), Positive: RGB(38, 162, 105), Neutral: RGB(229, 165, 10),
	Link: RGB(28, 113, 216), Focus: RGB(53, 132, 228), Hover: RGB(53, 132, 228),
	Separator: RGB(205, 199, 194),
	Radius:    kdeRadius, Padding: kdePadding,
}

// AdwaitaDark is libadwaita's dark scheme.
var AdwaitaDark = Palette{
	Name: "Adwaita Dark", Dark: true,
	WindowBG: RGB(36, 36, 36), WindowFG: RGB(255, 255, 255),
	ViewBG: RGB(30, 30, 30), ViewAltBG: RGB(48, 48, 48), ViewFG: RGB(255, 255, 255),
	ButtonBG: RGB(56, 56, 56), ButtonFG: RGB(255, 255, 255),
	SelectionBG: RGB(53, 132, 228), SelectionFG: RGB(255, 255, 255),
	TooltipBG:  RGB(56, 56, 56),
	InactiveFG: RGB(154, 153, 150),
	Negative:   RGB(255, 122, 116), Positive: RGB(46, 194, 126), Neutral: RGB(245, 194, 17),
	Link: RGB(120, 174, 237), Focus: RGB(53, 132, 228), Hover: RGB(53, 132, 228),
	Separator: RGB(61, 61, 61),
	Radius:    kdeRadius, Padding: kdePadding,
}
