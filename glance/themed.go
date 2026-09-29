package glance

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"

	fd "github.com/ushineko/fynedesygn"
	"github.com/ushineko/fynedesygn/widgets"
)

/*
themed is what every piece of a panel embeds so it can be given a face of its
own instead of taking the application's.

**The reason is overlays.** A card, a row, a meter and a cell are canvas
objects, so a container.ThemeOverride does not reach them -- which is why a
program with a panel and a settings window used to hand the *application's*
theme to the panel and the override to the window. That arrangement has a hole
in it: a dialog, a widget.Select's dropdown and a context menu are added to the
canvas's overlay stack, not to the window's content, so nothing overrides them
and they fall through to the application's theme. In a program arranged that
way, the application's theme is the panel's -- so opening a font chooser from
the settings window drew the whole dialog in the panel's face and the panel's
card opacity.

There is nothing to patch around it. Fyne's overlay stack is not reachable from
a content-level override, and Select's dropdown is Fyne's own widget. So the
arrangement is turned the right way round: the panel carries its own theme
explicitly, and the application's is left to the window that has overlays.

It is the same lesson this package has already learned twice -- the panel's
background is its own and not the app's theme, and so is the fade on its cards.
The face is the third and last of them.

A nil theme is the application's, which is what every panel that has not asked
for anything else wants and is what they all did before.
*/
type themed struct{ th fyne.Theme }

// SetTheme gives this object a theme of its own. Nil returns it to the
// application's.
//
// A panel sets this on everything in it; there is nothing for a consumer to
// do, and nothing that reads it until the object is restyled.
func (t *themed) SetTheme(th fyne.Theme) { t.th = th }

// Theme is the theme this object draws in: its own, or the application's.
func (t *themed) Theme() fyne.Theme {
	if t.th != nil {
		return t.th
	}
	if a := fyne.CurrentApp(); a != nil {
		return a.Settings().Theme()
	}
	return theme.DefaultTheme()
}

// variant is the light/dark variant to resolve colours in.
//
// The application's, always, even for an object with a theme of its own: a
// panel and a window may differ in face and size, and a program whose two
// windows disagreed about whether it is night would be a bug rather than a
// setting. The design system's own themes ignore the argument anyway and let
// the palette decide, which this cannot know and does not need to.
func (t *themed) variant() fyne.ThemeVariant {
	if a := fyne.CurrentApp(); a != nil {
		return a.Settings().ThemeVariant()
	}
	return theme.VariantDark
}

// colour resolves a colour in this object's theme.
func (t *themed) colour(n fyne.ThemeColorName) color.Color {
	return t.Theme().Color(n, t.variant())
}

// size resolves a size in this object's theme.
func (t *themed) size(n fyne.ThemeSizeName) float32 { return t.Theme().Size(n) }

// textSize is the ordinary text size in this object's theme, which is the one
// every margin in a card is a factor of.
func (t *themed) textSize() float32 { return t.size(theme.SizeNameText) }

// statusColour is widgets.StatusColor resolved in this object's theme rather
// than the application's.
//
// The mapping is the widgets package's and is not repeated here: the roles a
// status takes are a decision of the design system, and a second copy of them
// would be a second place to be wrong.
func (t *themed) statusColour(st fd.Status) color.Color {
	return t.colour(widgets.StatusColorName(st))
}

/*
refit gives texts this object's face and resizes them to what they then
measure.

**A canvas.Text draws inside its Size and clips what does not fit.** A
container gives it that size from its MinSize when the container is laid out,
and a Refresh does not re-run a parent's layout -- so a text that grew from
eight points to nine kept the width it was measured at and lost its last
letter. It showed as card titles reading "Peripheral" and "Bandwidtl".

**And a text measures in the application's font unless it is told otherwise,
whatever face it is drawn in.** That is Fyne's arrangement and it is not
obvious: canvas.Text.MinSize passes the text's own FontSource, which is nil
until somebody sets it, and a nil source means the application's theme. The
painter, meanwhile, draws the text in whatever theme covers it. A panel with a
face of its own therefore measured every label in one family and drew it in
another, and where the drawn family was the wider of the two the last glyph
fell off the end -- "CPU" as "CPL", "tailscale0" as "tailscale". The size was
never the problem; it was the width of an "0" in one font against another.

So the face is set on the text, not merely on the theme above it. Then
MinSize measures what will actually be drawn, and the resize below reserves
the right width.

Every Restyle here ends with this, because every Restyle can change either.
*/
func (t *themed) refit(texts ...*canvas.Text) {
	th := t.Theme()
	for _, x := range texts {
		if x == nil {
			continue
		}
		x.FontSource = th.Font(x.TextStyle)
		x.Resize(x.MinSize())
	}
}

/*
measure is the width of a string in this object's face.

fyne.MeasureText cannot answer it: it passes a nil font source, so it measures
in the application's font however the caller is drawn. This asks the driver
the same question with the face the text will actually use.
*/
func (t *themed) measure(s string, size float32, style fyne.TextStyle) float32 {
	a := fyne.CurrentApp()
	if a == nil || a.Driver() == nil {
		return 0
	}
	out, _ := a.Driver().RenderedTextSize(s, size, style, t.Theme().Font(style))
	return out.Width
}
