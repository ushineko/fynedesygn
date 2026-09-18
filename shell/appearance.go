package shell

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	fynetheme "fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	fd "github.com/ushineko/fynedesygn"
	fdtheme "github.com/ushineko/fynedesygn/theme"
	"github.com/ushineko/fynedesygn/widgets"
)

// AppearanceSection is the standard scheme, font, text size and scale picker.
// Fyne draws its own widgets, so these settings are the whole of what makes
// the window look like it belongs on the user's desktop, which is why they are
// a section rather than a line in a preferences dialog. monoSample is the
// monospace line the live sample shows; a log line that looks like the
// program's own is the useful preview.
func AppearanceSection(monoSample string) Section {
	return NewSection("Appearance", fynetheme.ColorPaletteIcon, func(s *Shell) fyne.CanvasObject {
		return buildAppearance(s, monoSample)
	})
}

func buildAppearance(s *Shell, monoSample string) fyne.CanvasObject {
	a := s.Appearance()
	apply := func() { s.SetAppearance(a) }

	scheme := widget.NewSelect(fdtheme.SchemeNames(), func(name string) { a.Scheme = name; apply() })
	scheme.SetSelected(a.Scheme)

	font := widget.NewSelect(fdtheme.FontNames(), func(name string) { a.Font = name; apply() })
	font.SetSelected(a.Font)

	mono := widget.NewSelect(fdtheme.FontNames(), func(name string) { a.Mono = name; apply() })
	mono.SetSelected(a.Mono)

	sizes := make([]string, 0, len(fdtheme.TextSizes()))
	for _, v := range fdtheme.TextSizes() {
		sizes = append(sizes, fmt.Sprintf("%g", v))
	}
	size := widget.NewSelect(sizes, func(v string) {
		for _, t := range fdtheme.TextSizes() {
			if fmt.Sprintf("%g", t) == v {
				a.TextSize = t
				apply()
				return
			}
		}
	})
	size.SetSelected(fmt.Sprintf("%g", a.TextSize))

	restart := widget.NewButtonWithIcon("Restart the window now", fynetheme.ViewRefreshIcon(), func() { s.Restart() })
	restart.Hide()
	scales := make([]string, 0, len(fdtheme.ScaleChoices()))
	for _, v := range fdtheme.ScaleChoices() {
		scales = append(scales, fdtheme.ScaleLabel(v))
	}
	scale := widget.NewSelect(scales, nil)
	scale.SetSelected(fdtheme.ScaleLabel(a.Scale))
	scale.OnChanged = func(v string) {
		a.Scale = fdtheme.ScaleValue(v)
		apply()
		restart.Show()
		s.Flash("Interface scale saved. It applies when the window next opens.", fd.StatusInfo)
	}

	reset := widget.NewButton("Reset to defaults", func() {
		a = fdtheme.DefaultAppearance()
		scheme.SetSelected(a.Scheme)
		font.SetSelected(a.Font)
		mono.SetSelected(a.Mono)
		size.SetSelected(fmt.Sprintf("%g", a.TextSize))
		scale.SetSelected(fdtheme.ScaleLabel(a.Scale))
		apply()
	})

	form := widget.NewForm(
		widget.NewFormItem("Colour scheme", scheme),
		widget.NewFormItem("Font", font),
		widget.NewFormItem("Monospace font", mono),
		widget.NewFormItem("Text size", size),
		widget.NewFormItem("Interface scale", container.NewHBox(scale, restart)),
	)

	return container.NewVScroll(container.NewVBox(
		widgets.Heading("Appearance",
			"How this window looks. Fyne draws its own widgets, so this is what decides whether it sits well next to the rest of your desktop."),
		form,
		container.NewHBox(reset),
		widgets.DimWrapped("These settings are kept in Fyne's own preference store and apply to this program only. "+
			"Every other setting lives in the program's configuration, where its command line can see it too."),
		widget.NewSeparator(),
		fdtheme.Sample(monoSample),
		widget.NewSeparator(),
		widgets.DimWrapped("The KDE schemes are transcribed from the desktop's colour-scheme files, the Adwaita ones from "+
			"libadwaita's named colours, the Windows and macOS ones from their published design tokens. They are "+
			"compiled in, so the window does not follow the desktop's current scheme and needs no desktop installed."),
		widgets.DimWrapped("Fonts are read from the system font directories. Fyne draws its own text and does not consult "+
			"fontconfig, so this list is what was found on disk rather than what the desktop is configured to use. "+
			"A family with no bold or italic face is drawn in its regular face for those styles."),
		widgets.DimWrapped("Interface scale enlarges everything in the window, text included, on top of the desktop's own "+
			"scale. Fyne draws text without hinting, which on a fractionally scaled desktop reads soft at the "+
			"default size; 1.2 is usually enough. It takes effect when the window is opened."),
	))
}
