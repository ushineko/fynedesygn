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

	fynetheme "fyne.io/fyne/v2/theme"

	"github.com/ushineko/fynedesygn/shell"
	fdtheme "github.com/ushineko/fynedesygn/theme"
)

// version is stamped by the build; "dev" otherwise.
var version = "dev" //nolint:gochecknoglobals // set through -ldflags

// appID names the preference store. Wayland matches the window to a desktop
// entry by this string, so a desktop file for the gallery would carry the same
// basename.
const appID = "io.ushineko.fynedesygn.gallery"

func main() {
	names := shell.Names(sections())
	section := flag.String("section", "", "open on this section: "+strings.Join(names, ", "))
	scheme := flag.String("scheme", "", "colour scheme for this run, not saved: "+strings.Join(fdtheme.SchemeNames(), ", "))
	showVersion := flag.Bool("version", false, "print the version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println("fynedesygn-gallery", version)
		return
	}
	if *section != "" && !known(names, *section) {
		fmt.Fprintf(os.Stderr, "unknown section %q; one of: %s\n", *section, strings.Join(names, ", "))
		os.Exit(2)
	}

	shell.Run(options(*section, *scheme))
}

// options describes the gallery to the shell.
func options(section, scheme string) shell.Options {
	demo := newJobDemo()
	return shell.Options{
		AppID:       appID,
		Name:        "fynedesygn",
		Version:     version,
		Sections:    sectionsWith(demo),
		Section:     section,
		Scheme:      scheme,
		StatusBar:   demo.statusBar,
		AlsoWorking: func() bool { return false },
		// Every shape, because this is the reference program: the control in
		// the header is the demonstration, and a component is judged here
		// before it is judged in a program.
		NavModes:      []shell.NavMode{shell.NavLabels, shell.NavIcons, shell.NavHidden},
		NavPlacements: []shell.NavPlacement{shell.NavLeft, shell.NavTop},
		// And a group, for the same reason: spec 031 is judged here first.
		// Three sections of one kind is exactly the shape a group is for.
		Groups: []shell.NavGroup{{
			Title:   "Components",
			Icon:    fynetheme.ViewFullScreenIcon,
			Members: []string{"Widgets", "Table", "Fonts"},
		}},
	}
}

// known reports whether name is one of names, case-insensitively.
func known(names []string, name string) bool {
	for _, n := range names {
		if strings.EqualFold(n, name) {
			return true
		}
	}
	return false
}
