/*
Command fynedesygn-gallery is the reference program for the fynedesygn design
system: every component in every colour scheme, in one window.

It exists for three reasons. It is the documentation a reader can point at. It
is the visual regression check: a change to a component is judged here before
it is judged in a consuming program. And it is the proof nothing was lost in
the extraction from the programs the components came from.

Flags:

	--section NAME   open on this section (see --help for the names)
	--scheme NAME    use this colour scheme for this run without saving it
	--version        print the version and exit

--section and --scheme are the screenshot contract: a capture script drives the
gallery through them, and they are computable before a Fyne app exists so that
--help and --version never start the toolkit.
*/
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	fdtheme "github.com/ushineko/fynedesygn/theme"
)

// version is stamped by the build; "dev" otherwise.
var version = "dev" //nolint:gochecknoglobals // set through -ldflags

// appID names the preference store. Wayland matches the window to a desktop
// entry by this string, so a desktop file for the gallery would carry the same
// basename.
const appID = "io.ushineko.fynedesygn.gallery"

func main() {
	section := flag.String("section", "", "open on this section: "+strings.Join(sectionNames(), ", "))
	scheme := flag.String("scheme", "", "colour scheme for this run, not saved: "+strings.Join(fdtheme.SchemeNames(), ", "))
	showVersion := flag.Bool("version", false, "print the version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println("fynedesygn-gallery", version)
		return
	}
	if *section != "" && sectionIndex(*section) < 0 {
		fmt.Fprintf(os.Stderr, "unknown section %q; one of: %s\n", *section, strings.Join(sectionNames(), ", "))
		os.Exit(2)
	}

	run(*section, *scheme)
}

// run builds the window. It is the whole of the gallery's shell until spec 002
// replaces it with the shared one.
func run(section, scheme string) {
	// Before the toolkit starts: the cursor fix reads config and exports
	// variables GLFW consults at init.
	fdtheme.ApplyCursorTheme()

	a := app.NewWithID(appID)
	g := &gallery{app: a, appearance: fdtheme.LoadAppearance(a.Preferences())}
	if scheme != "" {
		// A one-run override for screenshots: applied, never saved.
		g.appearance.Scheme = fdtheme.SchemeByName(scheme).Name
		g.oneRunScheme = true
	}
	fdtheme.ApplyScale(g.appearance.Scale)
	a.Settings().SetTheme(g.appearance.Theme())

	g.win = a.NewWindow("fynedesygn gallery " + version)
	g.content = container.NewScroll(widget.NewLabel(""))

	names := sectionNames()
	g.nav = widget.NewList(
		func() int { return len(names) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(i widget.ListItemID, o fyne.CanvasObject) { o.(*widget.Label).SetText(names[i]) },
	)
	g.nav.OnSelected = func(i widget.ListItemID) { g.show(i) }

	split := container.NewHSplit(g.nav, g.content)
	split.SetOffset(0.16)

	header := container.NewVBox(
		container.NewPadded(widget.NewLabelWithStyle("fynedesygn", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})),
		widget.NewSeparator(),
	)
	g.win.SetContent(container.NewBorder(header, nil, nil, nil, split))
	g.win.Resize(fyne.NewSize(1180, 760))
	g.win.SetMaster()

	start := 0
	if i := sectionIndex(section); i >= 0 {
		start = i
	}
	g.nav.Select(start)
	g.win.ShowAndRun()
}

// gallery is the state the sections share: the app, the window, the content
// pane and the appearance being previewed.
type gallery struct {
	app          fyne.App
	win          fyne.Window
	nav          *widget.List
	content      *container.Scroll
	current      int
	appearance   fdtheme.Appearance
	oneRunScheme bool
}

// show swaps the content pane to a section and scrolls to its top.
func (g *gallery) show(i int) {
	g.current = i
	g.content.Content = sections()[i].build(g)
	g.content.Refresh()
	g.content.ScrollToTop()
}

// applyAppearance sets the theme from the current choices and saves them,
// except when the scheme is a one-run override.
func (g *gallery) applyAppearance() {
	if g.oneRunScheme {
		g.app.Settings().SetTheme(g.appearance.Theme())
		saved := g.appearance
		saved.Scheme = fdtheme.LoadAppearance(g.app.Preferences()).Scheme
		saved.Save(g.app.Preferences())
		return
	}
	g.appearance.Apply(g.app)
}
