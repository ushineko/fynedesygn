package shell

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/ushineko/fynedesygn/settings"
)

/*
Tabs is a section made of sections, one of them drawn at a time under a strip
of buttons.

It is for a section that has grown into several pages of one activity: the
reader thinks of it as one place, so it is one entry in the navigation, and
the strip picks which page of it is showing. hotaru's Create section (Scenes,
Pictures, Dashboards) is where the shape came from, and nmsbonker's Tweaks
(one page per subject) is the second program that needed it.

A navigation group (NavGroup) is the other way to fold sections together, and
the two answer different questions. A group is for sections of a kind that
are each a place of their own -- three picture sources, four reports -- and it
puts them all in the navigation. Tabs are for pages of one place, and keep
them out of it. A group also vanishes in the icons-only and along-the-top
shapes, where a strip of tabs stays.

Choosing a tab is arriving somewhere, so it does what the navigation does:
every section is detached, the new part is told it arrived, and the content
pane goes back to the top. It is not a refresh, so the program's
OnInvalidate is not called and nothing it loaded is thrown away.

The chosen tab is remembered across rebuilds, navigation and restarts, by part
title, in the settings store. See spec 053.
*/
type Tabs struct {
	title string
	icon  func() fyne.Resource
	parts []Section
	// at is the part showing. It outlives a rebuild, because a rebuild is not
	// somebody changing tabs.
	at int
	// loaded is whether the stored choice has been read. Read on the first
	// Build, because that is the first time there is a shell to read it from.
	loaded bool
}

// TabsKey is where the chosen tabs live in the settings store: one object,
// keyed by the Tabs section's title, whose values are part titles.
const TabsKey = settings.Prefix + "tabs"

// NewTabs builds a section from its parts, which are drawn in the order given.
// icon may be nil for a section without one.
func NewTabs(title string, icon func() fyne.Resource, parts ...Section) *Tabs {
	return &Tabs{title: title, icon: icon, parts: parts}
}

// Title implements Section.
func (t *Tabs) Title() string { return t.title }

// Icon implements Section; nil when the section has no icon.
func (t *Tabs) Icon() fyne.Resource {
	if t.icon == nil {
		return nil
	}
	return t.icon()
}

// Parts are the pages, in the order they are drawn.
func (t *Tabs) Parts() []Section { return t.parts }

// Showing is the title of the part showing, or "" when there are no parts.
func (t *Tabs) Showing() string {
	if part := t.current(); part != nil {
		return part.Title()
	}
	return ""
}

/*
Show chooses a part by title, case-insensitively, and reports whether there
was one. For deep links and tests: it records the choice but does not redraw,
so a caller that is already on screen follows it with Refresh.
*/
func (t *Tabs) Show(title string) bool {
	i := t.index(title)
	if i < 0 {
		return false
	}
	t.at, t.loaded = i, true
	return true
}

/*
Detach tells every part, not only the one showing.

The shell detaches every section on every swap for the same reason: a part
that was showing a moment ago may still hold a scroll callback or a list a
worker writes into, and "it is not on screen" is exactly when that matters.
*/
func (t *Tabs) Detach() {
	for _, part := range t.parts {
		if d, ok := part.(Detacher); ok {
			d.Detach()
		}
	}
}

// Arrive tells the part showing that the navigation reached it.
func (t *Tabs) Arrive() {
	if a, ok := t.current().(Arriver); ok {
		a.Arrive()
	}
}

// Wide asks the part showing, because that is what the content pane holds.
func (t *Tabs) Wide() bool {
	w, ok := t.current().(Wide)
	return ok && w.Wide()
}

// Build draws the strip and the part showing. The strip is the Border's top,
// so a part that fills its space (affixed controls over its own scroller)
// still fills everything under the strip.
func (t *Tabs) Build(s *Shell) fyne.CanvasObject {
	t.load(s)
	part := t.current()
	if part == nil {
		return container.NewStack()
	}
	strip := make([]fyne.CanvasObject, 0, len(t.parts))
	for i, p := range t.parts {
		button := widget.NewButton(p.Title(), func() { t.choose(s, i) })
		if i == t.at {
			button.Importance = widget.HighImportance
		}
		strip = append(strip, button)
	}
	return container.NewBorder(container.NewHBox(strip...), nil, nil, nil, part.Build(s))
}

// choose is a tab being pressed: the navigation moving, without the
// navigation list moving with it.
func (t *Tabs) choose(s *Shell, i int) {
	if i == t.at || i < 0 || i >= len(t.parts) {
		return
	}
	t.at = i
	t.save(s)
	if s.Current() == t {
		s.swap(false)
		return
	}
	// Built somewhere other than the content pane (a test, a program
	// nesting one): there is no navigation to imitate, only a redraw.
	s.Refresh()
}

// current is the part showing, clamped in case the list ever shortens.
func (t *Tabs) current() Section {
	if len(t.parts) == 0 {
		return nil
	}
	if t.at < 0 || t.at >= len(t.parts) {
		t.at = 0
	}
	return t.parts[t.at]
}

func (t *Tabs) index(title string) int {
	for i, p := range t.parts {
		if strings.EqualFold(p.Title(), title) {
			return i
		}
	}
	return -1
}

// load reads the stored choice once. A title that names no part is ignored,
// which is what lets a program rename or drop a page.
func (t *Tabs) load(s *Shell) {
	if t.loaded {
		return
	}
	t.loaded = true
	if s == nil || s.store == nil {
		return
	}
	var saved map[string]string
	if !s.store.Get(TabsKey, &saved) {
		return
	}
	if i := t.index(saved[t.title]); i >= 0 {
		t.at = i
	}
}

// save records the choice beside any other Tabs section's.
func (t *Tabs) save(s *Shell) {
	if s == nil || s.store == nil {
		return
	}
	saved := map[string]string{}
	s.store.Get(TabsKey, &saved)
	if saved == nil {
		saved = map[string]string{}
	}
	saved[t.title] = t.Showing()
	if err := s.store.Set(TabsKey, saved); err != nil {
		s.Report("Saving the chosen tab", err)
	}
}
