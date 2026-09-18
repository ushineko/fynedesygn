package shell

import "fyne.io/fyne/v2"

// Section is one entry in the navigation.
//
// Title must be computable before a Fyne app exists: flag help lists the
// titles, and a theme icon constructed that early makes Fyne log "Attempt to
// access current Fyne app when none is started". Icon is therefore only called
// once the app exists. Build is stateless: the shell calls it again on every
// change and throws the previous tree away.
type Section interface {
	Title() string
	Icon() fyne.Resource
	Build(s *Shell) fyne.CanvasObject
}

// Detacher is implemented by a section that holds live widgets (a list a
// worker writes into) or a scroll callback. The shell calls Detach before it
// builds the replacement, never after: built first and dropped afterwards, a
// section registers its brand-new list and then has it thrown away by the
// very call that put it on screen, so a running job streams into nothing.
type Detacher interface {
	Detach()
}

/*
Arriver is implemented by a section that needs to know when the navigation
arrives at it, as opposed to when the shell rebuilds it where it stands.

The shell builds a section for both reasons and, without this, tells it
neither: navigating to a section builds it, and so does every rebuild while it
is on screen — an operation starting or finishing, a load landing, F5.

A section that refetches on arrival needs the difference. Refetching on every
build instead is a loop, because the fetch finishing rebuilds the section that
started it. Arrive is called on navigation only: the list, Select, and the
section the window opens on.

The mirror of Detacher, which is the shell telling a section it is about to be
replaced.
*/
type Arriver interface {
	Arrive()
}

// FuncSection is a Section made of closures, for programs that do not want a
// type per section.
type FuncSection struct {
	title  string
	icon   func() fyne.Resource
	build  func(*Shell) fyne.CanvasObject
	detach func()
	arrive func()
}

// NewSection builds a section from a title, a deferred icon and a builder.
// icon may be nil for a section without one.
func NewSection(title string, icon func() fyne.Resource, build func(*Shell) fyne.CanvasObject) *FuncSection {
	return &FuncSection{title: title, icon: icon, build: build}
}

// OnDetach registers the hook the shell calls before this section is
// replaced, and returns the section for chaining.
func (f *FuncSection) OnDetach(fn func()) *FuncSection {
	f.detach = fn
	return f
}

// OnArrive registers the hook the shell calls when the navigation reaches
// this section, and returns the section for chaining.
func (f *FuncSection) OnArrive(fn func()) *FuncSection {
	f.arrive = fn
	return f
}

// Title implements Section.
func (f *FuncSection) Title() string { return f.title }

// Icon implements Section; nil when the section has no icon.
func (f *FuncSection) Icon() fyne.Resource {
	if f.icon == nil {
		return nil
	}
	return f.icon()
}

// Build implements Section.
func (f *FuncSection) Build(s *Shell) fyne.CanvasObject { return f.build(s) }

// Detach implements Detacher; a no-op without a hook.
func (f *FuncSection) Detach() {
	if f.detach != nil {
		f.detach()
	}
}

// Arrive implements Arriver; a no-op without a hook.
func (f *FuncSection) Arrive() {
	if f.arrive != nil {
		f.arrive()
	}
}

// Names lists the titles in order. It reads titles only, so it is safe to
// call while parsing flags, before there is an app to hang an icon on.
func Names(sections []Section) []string {
	out := make([]string, 0, len(sections))
	for _, s := range sections {
		out = append(out, s.Title())
	}
	return out
}
