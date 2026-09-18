package theme

import "image/color"

/*
Palette is one colour scheme.

The field names follow KDE's .colors vocabulary rather than Fyne's, so a
transcription can be checked against its source file without a translation
step; Theme.Color does the translation. Only the roles a window paints are
carried; a scheme has more (Complementary, Header, a per-role Inactive set)
and carrying them without a widget that uses them would be noise.

Radius and Padding are the two size tokens that differ between desktop styles.
Breeze is tight and square; Fluent and Liquid Glass are rounder. A zero value
means Fyne's own default for that size.
*/
type Palette struct {
	// Name is what the Appearance picker shows and what the preference stores.
	Name string
	// Dark reports whether text is light on a dark window. Theme reports this
	// as the variant and ignores the one Fyne passes.
	Dark bool

	WindowBG    color.Color // [Colors:Window]    BackgroundNormal
	WindowFG    color.Color // [Colors:Window]    ForegroundNormal
	ViewBG      color.Color // [Colors:View]      BackgroundNormal: inputs, menus, lists
	ViewAltBG   color.Color // [Colors:View]      BackgroundAlternate: table headers
	ViewFG      color.Color // [Colors:View]      ForegroundNormal
	ButtonBG    color.Color // [Colors:Button]    BackgroundNormal
	ButtonFG    color.Color // [Colors:Button]    ForegroundNormal
	SelectionBG color.Color // [Colors:Selection] BackgroundNormal: also the primary/accent colour
	SelectionFG color.Color // [Colors:Selection] ForegroundNormal
	TooltipBG   color.Color // [Colors:Tooltip]   BackgroundNormal
	InactiveFG  color.Color // ForegroundInactive: disabled text and placeholders
	Negative    color.Color // ForegroundNegative: errors, StatusBad
	Positive    color.Color // ForegroundPositive: success, StatusGood
	Neutral     color.Color // ForegroundNeutral: warnings, StatusWarn
	Link        color.Color // ForegroundLink
	Focus       color.Color // DecorationFocus
	Hover       color.Color // DecorationHover
	Separator   color.Color // no KDE equivalent: derived from the window colours

	// Radius is the corner radius of inputs and selections.
	Radius float32
	// Padding is the theme padding between widgets.
	Padding float32
}

// RGB returns an opaque colour. It is exported so a program can define a
// palette of its own in the same notation the built-in ones use.
func RGB(r, g, b uint8) color.Color { return color.NRGBA{R: r, G: g, B: b, A: 0xff} }

// Alpha returns c at the given opacity. Fyne asks for translucent colours in
// places a desktop scheme has no role for (hover fills, the scrollbar, the
// modal overlay, a banner's tint) and deriving them from a scheme colour keeps
// them in the scheme's hue instead of dropping a neutral grey onto a warm
// palette.
func Alpha(c color.Color, a uint8) color.Color {
	// RGBA returns 16-bit channels; the high byte is the 8-bit value, so the
	// shift cannot overflow and the conversion is exact.
	r, g, b, _ := c.RGBA()
	return color.NRGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: a} //nolint:gosec // r,g,b are 16-bit; >>8 fits a byte
}

// schemes lists the palettes in the order the Appearance picker offers them:
// the KDE and GNOME schemes first because the design system grew up on
// Plasma, then the Windows and macOS pairs.
func schemes() []Palette {
	return []Palette{
		BreezeDark, BreezeLight, OxygenDark,
		AdwaitaDark, AdwaitaLight,
		WindowsDark, WindowsLight,
		MacOSDark, MacOSLight,
	}
}

// Schemes returns the built-in palettes in the order they are offered.
func Schemes() []Palette { return schemes() }

// SchemeNames lists the built-in scheme names in the order they are offered.
func SchemeNames() []string {
	all := schemes()
	names := make([]string, 0, len(all))
	for _, p := range all {
		names = append(names, p.Name)
	}
	return names
}

// SchemeByName returns the named scheme, falling back to DefaultScheme. A
// missing name means a stale preference, not an error worth a dialog.
func SchemeByName(name string) Palette {
	for _, p := range schemes() {
		if p.Name == name {
			return p
		}
	}
	return DefaultScheme()
}
