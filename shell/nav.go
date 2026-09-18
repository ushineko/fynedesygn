package shell

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	fynetheme "fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ushineko/fynedesygn/settings"
	"github.com/ushineko/fynedesygn/widgets"
)

/*
The navigation's shape.

One list of icons and labels down the left side is a reasonable default and was
the only option. A section that wants the horizontal room -- a grid, a document,
a diagram -- had no way to ask for it, and a screenshot of one spent a sixth of
its width on a list of seven words.

Two axes: how much of each entry is drawn, and which edge it is drawn on. A
program declares which combinations it allows, and one that declares nothing
keeps exactly the window it has today.

See spec 013.
*/

// NavMode is how much of a navigation entry is drawn.
type NavMode uint8

// The three modes. NavLabels is the default and the window as it has always
// been.
const (
	NavLabels NavMode = iota // icon and title
	NavIcons                 // icon alone
	NavHidden                // no navigation; the content takes the window
)

// NavPlacement is which edge the navigation is drawn on.
type NavPlacement uint8

// The two placements. NavLeft is the default.
const (
	NavLeft NavPlacement = iota
	NavTop
)

// NavKey is where the chosen shape lives in the settings store.
const NavKey = settings.Prefix + "nav"

// String names a mode as it is written in the settings file and in the menu.
func (m NavMode) String() string {
	switch m {
	case NavIcons:
		return "Icons"
	case NavHidden:
		return "Hidden"
	}
	return "Icons and labels"
}

// String names a placement as it is written in the settings file and the menu.
func (p NavPlacement) String() string {
	if p == NavTop {
		return "Along the top"
	}
	return "Down the left"
}

// navSettings is the stored shape. Written as names rather than numbers, so a
// person reading the settings file can see what it says and a later release can
// reorder the constants.
type navSettings struct {
	Mode      string `json:"mode"`
	Placement string `json:"placement"`
}

// navModeByName is the reverse of NavMode.String.
func navModeByName(name string) (NavMode, bool) {
	for _, m := range []NavMode{NavLabels, NavIcons, NavHidden} {
		if m.String() == name {
			return m, true
		}
	}
	return NavLabels, false
}

// navPlacementByName is the reverse of NavPlacement.String.
func navPlacementByName(name string) (NavPlacement, bool) {
	for _, p := range []NavPlacement{NavLeft, NavTop} {
		if p.String() == name {
			return p, true
		}
	}
	return NavLeft, false
}

/*
allowedModes is what this program permits, always at least one.

Empty means today's window: one shape, no control, nothing to choose. A public
API is a contract, and this is a change to a window four programs already ship,
so opting in is a line per program rather than a change imposed on all of them.
*/
func (s *Shell) allowedModes() []NavMode {
	if len(s.opts.NavModes) == 0 {
		return []NavMode{NavLabels}
	}
	return s.opts.NavModes
}

// allowedPlacements is the same for the edge the navigation is drawn on.
func (s *Shell) allowedPlacements() []NavPlacement {
	if len(s.opts.NavPlacements) == 0 {
		return []NavPlacement{NavLeft}
	}
	return s.opts.NavPlacements
}

// allowsMode reports whether a mode is one the program offers.
func (s *Shell) allowsMode(m NavMode) bool {
	for _, got := range s.allowedModes() {
		if got == m {
			return true
		}
	}
	return false
}

// allowsPlacement reports whether a placement is one the program offers.
func (s *Shell) allowsPlacement(p NavPlacement) bool {
	for _, got := range s.allowedPlacements() {
		if got == p {
			return true
		}
	}
	return false
}

/*
loadNav settles the shape this run starts in.

The program's own choice, then whatever the user last chose, then a fall back to
the first shape the program allows -- because a stored choice the program has
since withdrawn must not open a window in a shape it no longer supports.
*/
func (s *Shell) loadNav() {
	s.navMode, s.navPlace = s.opts.Nav, s.opts.NavPlace

	var saved navSettings
	if s.store.Get(NavKey, &saved) {
		if m, ok := navModeByName(saved.Mode); ok {
			s.navMode = m
		}
		if p, ok := navPlacementByName(saved.Placement); ok {
			s.navPlace = p
		}
	}
	if !s.allowsMode(s.navMode) {
		s.navMode = s.allowedModes()[0]
	}
	if !s.allowsPlacement(s.navPlace) {
		s.navPlace = s.allowedPlacements()[0]
	}
	if s.navMode != NavHidden {
		s.navShown = s.navMode
	}
}

// NavShape is the navigation as it stands: how much of each entry is drawn, and
// on which edge.
func (s *Shell) NavShape() (NavMode, NavPlacement) { return s.navMode, s.navPlace }

/*
SetNavShape changes the navigation and remembers the choice.

A shape the program does not allow is ignored rather than forced: the program
decided what it offers, and a caller cannot widen that from outside.
*/
func (s *Shell) SetNavShape(m NavMode, p NavPlacement) {
	if !s.allowsMode(m) || !s.allowsPlacement(p) {
		return
	}
	if m == s.navMode && p == s.navPlace {
		return
	}
	s.navMode, s.navPlace = m, p
	if m != NavHidden {
		// Remembered so Ctrl+B has something to come back to.
		s.navShown = m
	}
	if err := s.store.Set(NavKey, navSettings{Mode: m.String(), Placement: p.String()}); err != nil {
		s.Report("Saving the navigation's shape", err)
	}
	s.layout()
}

/*
toggleNav hides the navigation, or brings back the shape it had.

Bound to Ctrl+B, and only when the program allows hiding: a shortcut that does
nothing is worse than no shortcut, because it reads as a broken one.
*/
func (s *Shell) toggleNav() {
	if !s.allowsMode(NavHidden) {
		return
	}
	if s.navMode == NavHidden {
		back := s.navShown
		if back == NavHidden || !s.allowsMode(back) {
			back = s.allowedModes()[0]
		}
		s.SetNavShape(back, s.navPlace)
		return
	}
	s.SetNavShape(NavHidden, s.navPlace)
}

/*
navControl is the stock button that changes the shape.

Shown only when there is more than one shape to choose. One control whatever the
number of choices, rather than a toggle for two and a menu for three: a control
that changes shape according to how many options a program allows is one the
user has to re-learn per program.

It lives in the header, which is always drawn, so hiding the navigation always
has a way back. A control that disappears with the thing it hides is the one
state that needs it.
*/
func (s *Shell) navControl() fyne.CanvasObject {
	if len(s.allowedModes())+len(s.allowedPlacements()) < 3 {
		return nil // one mode and one placement: nothing to choose
	}

	items := s.navMenuItems()
	btn := widget.NewButtonWithIcon("", fynetheme.MenuIcon(), nil)
	s.navBtn = btn
	btn.OnTapped = func() {
		// The button's own tip first: opening a menu leaves the pointer where
		// it was, so nothing else would take it down and it would sit behind
		// the menu until the pointer moved.
		widgets.HideTips()
		// Relative to the button, not at its Position: a widget's Position is
		// measured from its parent, so a button in the header's row reports
		// something near the origin of that row and the menu opened in the
		// corner of the window.
		widget.ShowPopUpMenuAtRelativePosition(fyne.NewMenu("", items...), s.Window.Canvas(),
			fyne.NewPos(0, btn.Size().Height), btn)
	}
	tip := "Where the section list is drawn, and how much of it."
	if s.allowsMode(NavHidden) {
		tip += " Ctrl+B hides it and brings it back."
	}
	return widgets.WithTip(btn, tip)
}

/*
navMenuItems is what the control offers: every mode the program allows, then
every placement, with the current one of each ticked.
*/
func (s *Shell) navMenuItems() []*fyne.MenuItem {
	var items []*fyne.MenuItem
	for _, m := range s.allowedModes() {
		item := fyne.NewMenuItem(m.String(), func() { s.SetNavShape(m, s.navPlace) })
		item.Checked = m == s.navMode
		items = append(items, item)
	}
	if places := s.allowedPlacements(); len(places) > 1 {
		items = append(items, fyne.NewMenuItemSeparator())
		for _, p := range places {
			item := fyne.NewMenuItem(p.String(), func() { s.SetNavShape(s.navMode, p) })
			item.Checked = p == s.navPlace
			items = append(items, item)
		}
	}
	return items
}

/*
body is the region between the header and the status bar.

One shape per combination. Icons on the left is a strip rather than a split,
because a divider on something sized to its icons has nothing to give.
*/
func (s *Shell) body() fyne.CanvasObject {
	if s.navMode == NavHidden {
		return s.content
	}
	if s.navPlace == NavTop {
		return container.NewBorder(s.navHolderFor(true), nil, nil, nil, s.content)
	}
	if s.navMode == NavIcons {
		return container.NewBorder(nil, nil, s.navHolderFor(false), nil, s.content)
	}
	return s.HSplit(NavSplitKey, NavOffset, s.nav, s.content)
}

// navHolderFor is the button navigation, in a holder the shell can redraw when
// the selection moves without rebuilding the window around it.
func (s *Shell) navHolderFor(horizontal bool) fyne.CanvasObject {
	s.navHolder = container.NewStack(s.navButtons(horizontal))
	return s.navHolder
}

// redrawNav repaints the button navigation, so the current section is the one
// that looks current. A no-op in the shape that uses the list.
func (s *Shell) redrawNav() {
	if s.navHolder == nil {
		return
	}
	s.navHolder.Objects[0] = s.navButtons(s.navPlace == NavTop)
	s.navHolder.Refresh()
}

/*
navButtons is one button per section, laid out along the edge it is drawn on.

Each carries its section's title as a tip, which is not decoration in an
icons-only navigation: without it the window is a row of pictures to guess at.
A section with no icon of its own gets a generic one, because a blank button is
a section nobody can reach.
*/
func (s *Shell) navButtons(horizontal bool) fyne.CanvasObject {
	labels := s.navMode == NavLabels
	items := make([]fyne.CanvasObject, 0, len(s.opts.Sections))
	for i, sec := range s.opts.Sections {
		icon := sec.Icon()
		if icon == nil {
			icon = fynetheme.RadioButtonIcon()
		}
		title := ""
		if labels {
			title = sec.Title()
		}
		btn := widget.NewButtonWithIcon(title, icon, func() { s.selectIndex(i) })
		if i == s.current {
			btn.Importance = widget.HighImportance
		}
		items = append(items, widgets.WithTip(btn, sec.Title()))
	}

	if horizontal {
		row := container.NewHScroll(container.NewHBox(items...))
		return row
	}
	// Down the left: the buttons at their natural height, a spacer under them
	// so a short list does not stretch, and a scroller for a long one.
	return container.NewVScroll(container.NewVBox(items...))
}

// selectIndex moves to a section from whichever navigation is drawn.
func (s *Shell) selectIndex(i int) {
	if i < 0 || i >= len(s.opts.Sections) {
		return
	}
	if s.usesList() && s.nav != nil {
		s.nav.Select(i) // the list's own selection drives the swap
		return
	}
	s.current = i
	s.swap(false)
	s.redrawNav()
}

// usesList reports whether the current shape draws the section list, which is
// the one shape with a selection of its own to keep in step.
func (s *Shell) usesList() bool { return s.navMode == NavLabels && s.navPlace == NavLeft }
