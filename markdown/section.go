package markdown

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"

	"github.com/ushineko/fynedesygn/shell"
)

// Section is a shell section that is one document. The pane follows the
// shell's content scroller so one scrollbar governs the whole section, and
// it detaches when the section is replaced. header, when set, is built above
// the document on every rebuild.
func Section(title string, icon func() fyne.Resource, src string, o Options, header func(s *shell.Shell) fyne.CanvasObject) shell.Section {
	sec := &docSection{title: title, icon: icon, src: src, opts: o, header: header}
	return sec
}

type docSection struct {
	title  string
	icon   func() fyne.Resource
	src    string
	opts   Options
	header func(s *shell.Shell) fyne.CanvasObject
	pane   *Pane
}

func (d *docSection) Title() string { return d.title }

func (d *docSection) Icon() fyne.Resource {
	if d.icon == nil {
		return nil
	}
	return d.icon()
}

func (d *docSection) Build(s *shell.Shell) fyne.CanvasObject {
	d.pane = New(d.src, d.opts)
	d.pane.Follow(s.Scroller())
	if d.header == nil {
		return d.pane
	}
	return container.NewVBox(d.header(s), d.pane)
}

// Detach implements shell.Detacher.
func (d *docSection) Detach() {
	if d.pane != nil {
		d.pane.Detach()
		d.pane = nil
	}
}
