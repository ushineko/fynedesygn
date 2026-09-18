package theme

import (
	"image/color"

	"fyne.io/fyne/v2"
	fynetheme "fyne.io/fyne/v2/theme"
)

// DefaultTextSize is smaller than Fyne's own 14. The windows this package
// serves are dense and information-heavy (tables, reports, listings) and 14 pt
// makes them feel like a phone application. 12 is close to what Breeze and
// Adwaita use for interface text at a normal DPI.
const DefaultTextSize float32 = 12

// TextSizes are the text sizes the Appearance picker offers.
func TextSizes() []float32 { return []float32{10, 11, 12, 13, 14, 16, 18} }

// Options are the user's choices layered over a Palette.
type Options struct {
	// Font is the interface family; nil means the font Fyne ships with.
	Font *Font
	// Mono is the family for monospace text (log panes, code); nil means
	// Fyne's monospace face. It is never taken from Font.
	Mono *Font
	// TextSize in points; 0 means DefaultTextSize.
	TextSize float32
}

// Theme adapts a Palette and Options to fyne.Theme. Icons come from Fyne's
// default theme: a colour scheme says nothing about them, and following the
// desktop's icon theme would mean reading it at runtime.
type Theme struct {
	palette Palette
	font    *Font
	mono    *Font
	text    float32
}

var _ fyne.Theme = Theme{}

// New builds a theme from a palette and the user's options.
func New(p Palette, o Options) Theme {
	return Theme{palette: p, font: o.Font, mono: o.Mono, text: o.TextSize}
}

// Palette returns the scheme the theme paints with.
func (t Theme) Palette() Palette { return t.palette }

// TextSize is the base text size in points.
func (t Theme) TextSize() float32 {
	if t.text > 0 {
		return t.text
	}
	return DefaultTextSize
}

// Font implements fyne.Theme.
//
// Monospace is never taken from the interface family: a proportional family
// will not have a monospace face, and substituting one silently would misalign
// the places that asked for monospace precisely because alignment mattered. It
// comes from Options.Mono when one is chosen, else Fyne's own.
func (t Theme) Font(s fyne.TextStyle) fyne.Resource {
	if s.Monospace {
		if r := t.mono.Face(s); r != nil {
			return r
		}
		return fynetheme.DefaultTheme().Font(s)
	}
	if r := t.font.Face(s); r != nil {
		return r
	}
	return fynetheme.DefaultTheme().Font(s)
}

// Icon implements fyne.Theme with Fyne's default icons.
func (t Theme) Icon(n fyne.ThemeIconName) fyne.Resource {
	return fynetheme.DefaultTheme().Icon(n)
}

// Size implements fyne.Theme. Corner radius and padding come from the palette
// (Breeze is tight and square, Fluent and Liquid Glass are rounder); the text
// sizes scale from the chosen base.
func (t Theme) Size(n fyne.ThemeSizeName) float32 {
	switch n {
	case fynetheme.SizeNameInputRadius, fynetheme.SizeNameSelectionRadius:
		if t.palette.Radius > 0 {
			return t.palette.Radius
		}
	case fynetheme.SizeNamePadding:
		if t.palette.Padding > 0 {
			return t.palette.Padding
		}
	case fynetheme.SizeNameText:
		return t.TextSize()
	case fynetheme.SizeNameHeadingText:
		return t.TextSize() * 1.3
	case fynetheme.SizeNameSubHeadingText:
		return t.TextSize() * 1.15
	case fynetheme.SizeNameCaptionText:
		return t.TextSize() * 0.85
	}
	return fynetheme.DefaultTheme().Size(n)
}

// Color implements fyne.Theme.
//
// The variant argument is ignored on purpose. Fyne passes the variant the app
// is set to, but the palette already decided whether it is dark; honouring
// both would let a mismatch paint dark text on a dark window.
func (t Theme) Color(n fyne.ThemeColorName, _ fyne.ThemeVariant) color.Color {
	p := t.palette
	switch n {
	case fynetheme.ColorNameBackground:
		return p.WindowBG
	case fynetheme.ColorNameForeground:
		return p.WindowFG
	case fynetheme.ColorNameForegroundOnPrimary:
		return p.SelectionFG
	case fynetheme.ColorNameInputBackground, fynetheme.ColorNameMenuBackground:
		return p.ViewBG
	case fynetheme.ColorNameOverlayBackground:
		return p.WindowBG
	case fynetheme.ColorNameButton:
		return p.ButtonBG
	case fynetheme.ColorNamePrimary, fynetheme.ColorNameSelection:
		return p.SelectionBG
	case fynetheme.ColorNameFocus:
		return Alpha(p.Focus, 0x66)
	case fynetheme.ColorNameHover:
		return Alpha(p.Hover, 0x33)
	case fynetheme.ColorNamePressed:
		return Alpha(p.SelectionBG, 0x55)
	case fynetheme.ColorNameDisabled, fynetheme.ColorNamePlaceHolder:
		return p.InactiveFG
	case fynetheme.ColorNameDisabledButton:
		return Alpha(p.ButtonBG, 0x88)
	case fynetheme.ColorNameError:
		return p.Negative
	case fynetheme.ColorNameSuccess:
		return p.Positive
	case fynetheme.ColorNameWarning:
		return p.Neutral
	case fynetheme.ColorNameHyperlink:
		return p.Link
	case fynetheme.ColorNameSeparator, fynetheme.ColorNameInputBorder:
		return p.Separator
	case fynetheme.ColorNameHeaderBackground:
		return p.ViewAltBG
	case fynetheme.ColorNameScrollBar:
		return Alpha(p.InactiveFG, 0x99)
	case fynetheme.ColorNameShadow:
		return color.NRGBA{A: 0x66}
	}
	return fynetheme.DefaultTheme().Color(n, t.Variant())
}

// Variant is the variant the palette declares, for the few Fyne calls that
// need one.
func (t Theme) Variant() fyne.ThemeVariant {
	if t.palette.Dark {
		return fynetheme.VariantDark
	}
	return fynetheme.VariantLight
}

// MappedColors lists every colour role Color answers from the palette. Tests
// and the gallery use it to check that no palette leaves a role nil.
func MappedColors() []fyne.ThemeColorName {
	return []fyne.ThemeColorName{
		fynetheme.ColorNameBackground, fynetheme.ColorNameForeground,
		fynetheme.ColorNameForegroundOnPrimary, fynetheme.ColorNameInputBackground,
		fynetheme.ColorNameMenuBackground, fynetheme.ColorNameOverlayBackground,
		fynetheme.ColorNameButton, fynetheme.ColorNamePrimary, fynetheme.ColorNameSelection,
		fynetheme.ColorNameFocus, fynetheme.ColorNameHover, fynetheme.ColorNamePressed,
		fynetheme.ColorNameDisabled, fynetheme.ColorNamePlaceHolder,
		fynetheme.ColorNameDisabledButton, fynetheme.ColorNameError,
		fynetheme.ColorNameSuccess, fynetheme.ColorNameWarning, fynetheme.ColorNameHyperlink,
		fynetheme.ColorNameSeparator, fynetheme.ColorNameInputBorder,
		fynetheme.ColorNameHeaderBackground, fynetheme.ColorNameScrollBar,
		fynetheme.ColorNameShadow,
	}
}
