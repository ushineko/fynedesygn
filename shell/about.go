package shell

import (
	"net/url"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	fynetheme "fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ushineko/fynedesygn/widgets"
)

// aboutLogoSize is the edge of the logo in the About section.
const aboutLogoSize float32 = 72

// About describes a program for AboutSection.
type About struct {
	// Icon is drawn at 72 px beside the name; nil draws none.
	Icon fyne.Resource
	// Name and Version head the section; Blurb is one wrapped paragraph.
	Name, Version, Blurb string
	// URL, when set, is a hyperlink under the blurb; URLText is its caption,
	// the URL itself when empty.
	URL, URLText string
	// Notes are the "What it does" blocks: a bold title over a dim paragraph.
	Notes []Note
	// Facts are the "Facts" form: licence, configuration path, and the like.
	Facts []Fact
	// Extra, when set, is appended under the facts: the embedded README, for
	// instance. It receives the shell so it can follow the content scroller.
	Extra func(s *Shell) fyne.CanvasObject
}

// Note is one titled paragraph in About.Notes.
type Note struct{ Title, Detail string }

// Fact is one label and value in About.Facts.
type Fact struct{ Label, Value string }

// AboutSection is the standard About page: logo, name, version, blurb, link,
// notes and facts, the shape the consuming programs share. Long-form
// documentation belongs in the README; a shortened restatement here would
// become a second, less careful copy to keep in sync. Use Extra to show the
// README itself; OnDetach on the returned section releases what Extra holds.
func AboutSection(a About) *FuncSection {
	return NewSection("About", fynetheme.HelpIcon, func(s *Shell) fyne.CanvasObject {
		return buildAbout(s, a)
	})
}

func buildAbout(s *Shell, a About) fyne.CanvasObject {
	name := widget.NewLabelWithStyle(a.Name, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	name.TextStyle = fyne.TextStyle{Bold: true}
	head := container.NewVBox(name, widgets.Dim(a.Version), widgets.Wrapped(a.Blurb))
	var top fyne.CanvasObject = head
	if a.Icon != nil {
		logo := canvas.NewImageFromResource(a.Icon)
		logo.FillMode = canvas.ImageFillContain
		logo.SetMinSize(fyne.NewSize(aboutLogoSize, aboutLogoSize))
		top = container.NewBorder(nil, nil, container.NewPadded(logo), nil, head)
	}

	items := []fyne.CanvasObject{top}
	if a.URL != "" {
		if u, err := url.Parse(a.URL); err == nil {
			text := a.URLText
			if text == "" {
				text = a.URL
			}
			items = append(items, widget.NewHyperlink(text, u))
		}
	}
	if len(a.Notes) > 0 {
		items = append(items, widget.NewSeparator(),
			widget.NewLabelWithStyle("What it does", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
		for _, n := range a.Notes {
			items = append(items, widgets.AboutNote(n.Title, n.Detail))
		}
	}
	if len(a.Facts) > 0 {
		form := widget.NewForm()
		for _, f := range a.Facts {
			form.Append(f.Label, widget.NewLabel(f.Value))
		}
		items = append(items, widget.NewSeparator(),
			widget.NewLabelWithStyle("Facts", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), form)
	}
	if a.Extra != nil {
		items = append(items, widget.NewSeparator(), a.Extra(s))
	}
	/*
		Content, not a scroller.

		The shell puts what a section builds inside Shell.Scroller(), so
		wrapping again is a scroll inside a scroll -- the rule this module
		states, broken by this module. It also made About.Extra's own purpose
		impossible: a document pane following the content scroller was
		following the outer one while sitting in the inner one, and rendered
		the first screenful and nothing after it (spec 020).
	*/
	return container.NewVBox(items...)
}
