package wizard

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	fd "github.com/ushineko/fynedesygn"
	"github.com/ushineko/fynedesygn/dialogs"
	"github.com/ushineko/fynedesygn/forms"
	"github.com/ushineko/fynedesygn/markdown"
	"github.com/ushineko/fynedesygn/widgets"
)

// The standard pages (R7). Each is built only from components the module
// already has, so a wizard reads like the rest of a program's windows.

// document renders Markdown block by block. It sits in the wizard's one
// scroller, so it is not a markdown.Pane, which would bring its own.
func document(src string) fyne.CanvasObject {
	box := container.NewVBox()
	for _, b := range markdown.Blocks(src) {
		box.Add(markdown.RenderBlock(b, markdown.Options{}))
	}
	return box
}

type textPage struct {
	title, body string
}

func (p *textPage) Title() string                   { return p.title }
func (p *textPage) Build(*Wizard) fyne.CanvasObject { return document(p.body) }

// Welcome is a page of Markdown that says what the wizard will do.
func Welcome(title, body string) Page { return &textPage{title: title, body: body} }

// LicencePage shows a licence and holds Next until it is accepted.
type LicencePage struct {
	body   string
	accept *widget.Check
}

// LicenceAccept is the text of the licence page's check.
const LicenceAccept = "I accept the terms of the licence"

// Licence is a page with a licence in Markdown and an "I accept" check. It
// is valid only while the check is set.
func Licence(body string) *LicencePage { return &LicencePage{body: body} }

// Title implements Page.
func (p *LicencePage) Title() string { return "Licence" }

// Valid implements Validator.
func (p *LicencePage) Valid() bool { return p.accept != nil && p.accept.Checked }

// Build implements Page.
func (p *LicencePage) Build(w *Wizard) fyne.CanvasObject {
	p.accept = widget.NewCheck(LicenceAccept, func(bool) { w.Revalidate() })
	return container.NewVBox(document(p.body), widget.NewSeparator(), p.accept)
}

// DirectoryPage asks for a directory.
type DirectoryPage struct {
	title, label string
	check        func(string) error
	entry        *widget.Entry
	problem      *widget.Label
	initial      string
}

// Directory is a page with one directory entry and a Browse button. It is
// valid while check returns nil for the entry's text, and check's error
// shows below the entry.
func Directory(title, label, initial string, check func(string) error) *DirectoryPage {
	return &DirectoryPage{title: title, label: label, initial: initial, check: check}
}

// Title implements Page.
func (p *DirectoryPage) Title() string { return p.title }

// Value is the directory entered.
func (p *DirectoryPage) Value() string {
	if p.entry == nil {
		return p.initial
	}
	return p.entry.Text
}

// Changed implements Changer: the user typed something else.
func (p *DirectoryPage) Changed() bool { return p.Value() != p.initial }

// Valid implements Validator.
func (p *DirectoryPage) Valid() bool { return p.check == nil || p.check(p.Value()) == nil }

// Build implements Page.
func (p *DirectoryPage) Build(w *Wizard) fyne.CanvasObject {
	p.entry = widget.NewEntry()
	p.entry.SetText(p.initial)
	p.problem = widget.NewLabel("")
	p.problem.Wrapping = fyne.TextWrapWord
	p.entry.OnChanged = func(string) { p.recheck(); w.Revalidate() }
	p.recheck()
	field := fyne.CanvasObject(p.entry)
	if w.Window != nil {
		field = dialogs.WithBrowse(w.Window, p.entry, true)
	}
	return container.NewVBox(widget.NewLabel(p.label), field, p.problem)
}

func (p *DirectoryPage) recheck() {
	var err error
	if p.check != nil {
		err = p.check(p.entry.Text)
	}
	if err != nil {
		p.problem.Importance = widgets.ImportanceFor(fd.StatusBad)
		p.problem.SetText(err.Error())
		return
	}
	p.problem.SetText("")
}

// FormPage is a page of options.
type FormPage struct {
	title  string
	form   *forms.Form
	values map[string]string
	valid  func() bool
	start  map[string]string
}

/*
Form is a page holding f, with values set into it when the page is built.

The values are applied in Build and not by the caller, because the pages are
made before Run has created the app, and setting a widget refreshes it,
which Fyne cannot do without an app. It is valid while valid returns true
(nil means always), which is asked again after every change to the form.
*/
func Form(title string, f *forms.Form, values map[string]string, valid func() bool) *FormPage {
	return &FormPage{title: title, form: f, values: values, valid: valid}
}

// Title implements Page.
func (p *FormPage) Title() string { return p.title }

// Valid implements Validator.
func (p *FormPage) Valid() bool { return p.valid == nil || p.valid() }

// Changed implements Changer: a value differs from when the page was built.
func (p *FormPage) Changed() bool {
	for k, v := range p.form.Values() {
		if p.start[k] != v {
			return true
		}
	}
	return false
}

// Build implements Page.
func (p *FormPage) Build(w *Wizard) fyne.CanvasObject {
	if p.values != nil {
		p.form.Set(p.values)
	}
	p.start = p.form.Values()
	prev := p.form.OnChange
	p.form.OnChange = func(key, value string) {
		if prev != nil {
			prev(key, value)
		}
		w.Revalidate()
	}
	return p.form.Widget()
}

// Fact is one row of a summary.
type Fact struct{ Label, Value string }

// SummaryPage shows what is about to happen.
type SummaryPage struct {
	title, next string
	facts       func() []Fact
	box         *fyne.Container
}

// Summary is a page of facts read each time it becomes current, so it
// shows the choices made on the pages before it. next labels its Next
// button, such as "Install".
func Summary(title, next string, facts func() []Fact) *SummaryPage {
	return &SummaryPage{title: title, next: next, facts: facts}
}

// Title implements Page.
func (p *SummaryPage) Title() string { return p.title }

// NextLabel implements Labeller.
func (p *SummaryPage) NextLabel() string { return p.next }

// Build implements Page.
func (p *SummaryPage) Build(*Wizard) fyne.CanvasObject {
	p.box = container.NewVBox()
	return p.box
}

// Enter implements Enterer. The rows change only when a choice did, so
// they are rebuilt here and not on a timer.
func (p *SummaryPage) Enter(*Wizard) {
	rows := make([]fyne.CanvasObject, 0)
	for _, f := range p.facts() {
		rows = append(rows, widgets.PlainRow(f.Label, f.Value))
	}
	p.box.Objects = rows
	p.box.Refresh()
}

// FinishPage is the last page.
type FinishPage struct {
	title, body string
	checks      []*widget.Check
}

// Finish is the last page: Markdown, and checks such as "Launch now" whose
// state Result.Checks reports after the window closes.
func Finish(title, body string, checks ...*widget.Check) *FinishPage {
	return &FinishPage{title: title, body: body, checks: checks}
}

// Title implements Page.
func (p *FinishPage) Title() string { return p.title }

// Build implements Page.
func (p *FinishPage) Build(*Wizard) fyne.CanvasObject {
	box := container.NewVBox(document(p.body))
	for _, c := range p.checks {
		box.Add(c)
	}
	return box
}
