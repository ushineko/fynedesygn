package theme

/*
The Windows palettes are transcribed from the Windows 11 Fluent design tokens
(the WinUI 3 common theme resources): SolidBackgroundFillColorBase for the
window, LayerFillColorDefault and CardBackgroundFillColorSecondary for views,
ControlFillColorDefault for buttons, TextFillColorPrimary and
TextFillColorDisabled for text, the AccentFillColorDefault of each theme for
selection, and SystemFillColorCritical/Success/Caution for the status triple.

Fluent expresses most fills as white or black at an alpha over the base; Fyne
wants opaque fills, so the composited results over the base colour are used.
The accent is the default Windows accent (SystemAccentColor 0078D4); in the
dark theme Fluent uses its Light2 tint (60CDFF) with dark text on it, and in
the light theme its Dark1 tint (005FB8) with white text, which is why the two
selections differ.

Control corners are 4 px in Fluent; padding is set to match.
*/
const (
	windowsRadius  float32 = 4
	windowsPadding float32 = 4
)

// WindowsDark is the Windows 11 dark theme.
var WindowsDark = Palette{
	Name: "Windows Dark", Dark: true,
	WindowBG: RGB(32, 32, 32), WindowFG: RGB(255, 255, 255),
	ViewBG: RGB(43, 43, 43), ViewAltBG: RGB(50, 50, 50), ViewFG: RGB(255, 255, 255),
	ButtonBG: RGB(45, 45, 45), ButtonFG: RGB(255, 255, 255),
	SelectionBG: RGB(96, 205, 255), SelectionFG: RGB(0, 0, 0),
	TooltipBG:  RGB(43, 43, 43),
	InactiveFG: RGB(113, 113, 113),
	Negative:   RGB(255, 153, 164), Positive: RGB(108, 203, 95), Neutral: RGB(252, 225, 0),
	Link: RGB(96, 205, 255), Focus: RGB(255, 255, 255), Hover: RGB(255, 255, 255),
	Separator: RGB(51, 51, 51),
	Radius:    windowsRadius, Padding: windowsPadding,
}

// WindowsLight is the Windows 11 light theme.
var WindowsLight = Palette{
	Name: "Windows Light", Dark: false,
	WindowBG: RGB(243, 243, 243), WindowFG: RGB(27, 27, 27),
	ViewBG: RGB(255, 255, 255), ViewAltBG: RGB(246, 246, 246), ViewFG: RGB(27, 27, 27),
	ButtonBG: RGB(251, 251, 251), ButtonFG: RGB(27, 27, 27),
	SelectionBG: RGB(0, 95, 184), SelectionFG: RGB(255, 255, 255),
	TooltipBG:  RGB(249, 249, 249),
	InactiveFG: RGB(156, 156, 156),
	Negative:   RGB(196, 43, 28), Positive: RGB(15, 123, 15), Neutral: RGB(157, 93, 0),
	Link: RGB(0, 95, 184), Focus: RGB(27, 27, 27), Hover: RGB(0, 0, 0),
	Separator: RGB(224, 224, 224),
	Radius:    windowsRadius, Padding: windowsPadding,
}
