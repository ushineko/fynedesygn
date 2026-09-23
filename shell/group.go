package shell

import (
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	fynetheme "fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ushineko/fynedesygn/settings"
)

/*
Folding a run of sections under one heading.

A program whose navigation has grown says so by listing more sections, and
past seven or eight the list stops reading as a set of places and starts
reading as an inventory. The sections that caused it are usually of a kind --
three sources of pictures, four reports, five importers -- and what the reader
wants from the list is the kind, until the moment they want one of them.

A group is a heading in the list with its members indented beneath it, opened
and closed from the heading. It is drawn only in the shape that has a list to
draw it in: icons-only and along-the-top are a strip of buttons, where a tree
has nowhere to go, and there every section is a button as it has always been.

A group is not a section. It has no page, it is not something Select can reach
and it is not counted by Ctrl+1..9, which stay bound to sections in the order
the program listed them -- so folding sections into a group does not renumber
the shortcuts of the ones above it.

See spec 031.
*/

// NavGroup folds sections under one heading in the navigation list.
//
// Members are section titles, matched case-insensitively against
// Options.Sections. A title that names no section is ignored, which is what
// lets a program list a group whose members are built conditionally. Members
// should be contiguous in Sections order: the list is drawn in that order, and
// the heading appears where the first member would have been.
type NavGroup struct {
	Title   string
	Icon    func() fyne.Resource
	Members []string
}

// resource is the group's icon, or nil when it has none.
func (g NavGroup) resource() fyne.Resource {
	if g.Icon == nil {
		return nil
	}
	return g.Icon()
}

// NavGroupsKey is where the closed groups live in the settings store.
const NavGroupsKey = settings.Prefix + "nav.groups"

// navGroupSettings is the stored shape: the groups that are closed, by title.
// Closed rather than open, so a group a program adds later starts open.
type navGroupSettings struct {
	Closed []string `json:"closed"`
}

// navRow is one line of the navigation list: a group heading (sec < 0) or a
// section, which carries its group when it has one so the row can be indented.
type navRow struct {
	group int // index into Options.Groups, or -1
	sec   int // index into Options.Sections, or -1 for a heading
}

// groupOf is the group a section belongs to, or -1.
func (s *Shell) groupOf(sec int) int {
	if sec < 0 || sec >= len(s.opts.Sections) {
		return -1
	}
	title := s.opts.Sections[sec].Title()
	for gi, g := range s.opts.Groups {
		for _, m := range g.Members {
			if strings.EqualFold(m, title) {
				return gi
			}
		}
	}
	return -1
}

// loadGroups reads which groups the user last closed. A stored title that
// names no group is dropped, the same way a withdrawn navigation shape is.
func (s *Shell) loadGroups() {
	s.closed = map[string]bool{}
	var saved navGroupSettings
	if s.store == nil || !s.store.Get(NavGroupsKey, &saved) {
		return
	}
	for _, title := range saved.Closed {
		for _, g := range s.opts.Groups {
			if strings.EqualFold(g.Title, title) {
				s.closed[g.Title] = true
			}
		}
	}
}

// saveGroups records the closed groups, in the order the program listed them
// so the file does not churn on the order a map happened to walk in.
func (s *Shell) saveGroups() {
	if s.store == nil {
		return
	}
	out := navGroupSettings{Closed: []string{}}
	for _, g := range s.opts.Groups {
		if s.closed[g.Title] {
			out.Closed = append(out.Closed, g.Title)
		}
	}
	if err := s.store.Set(NavGroupsKey, out); err != nil {
		s.Report("Saving the navigation's groups", err)
	}
}

/*
buildRows is the list as it stands: every section in the program's order, with
a heading inserted before the first member of each group and the members of a
closed group left out.

Built rather than stored, because it changes for two reasons -- a group opened
or closed, and the shell rebuilding -- and a second copy of the truth is a
second thing to keep in step.
*/
func (s *Shell) buildRows() {
	rows := make([]navRow, 0, len(s.opts.Sections)+len(s.opts.Groups))
	headed := make(map[int]bool, len(s.opts.Groups))
	for i := range s.opts.Sections {
		g := s.groupOf(i)
		if g < 0 {
			rows = append(rows, navRow{group: -1, sec: i})
			continue
		}
		if !headed[g] {
			headed[g] = true
			rows = append(rows, navRow{group: g, sec: -1})
		}
		if s.closed[s.opts.Groups[g].Title] {
			continue
		}
		rows = append(rows, navRow{group: g, sec: i})
	}
	s.rows = rows
}

// rowFor is the list row a section is drawn on, or -1 when its group is
// closed and it is not drawn at all.
func (s *Shell) rowFor(sec int) int {
	for i, r := range s.rows {
		if r.sec == sec {
			return i
		}
	}
	return -1
}

/*
setGroupClosed opens or closes a group and redraws the list around the
selection.

The section on screen does not change. Closing the group that holds it leaves
the reader on the page they were reading with nothing highlighted, which is
honest -- the entry is not on the list to highlight -- and opening the group
again puts the highlight back.
*/
func (s *Shell) setGroupClosed(g int, closed bool) {
	if g < 0 || g >= len(s.opts.Groups) {
		return
	}
	if s.closed[s.opts.Groups[g].Title] == closed {
		return
	}
	s.closed[s.opts.Groups[g].Title] = closed
	s.saveGroups()
	s.buildRows()
	if !s.usesList() {
		// The button navigations are redrawn whole; there is no selection of
		// their own to put back.
		s.redrawNav()
		return
	}
	if s.nav == nil {
		return
	}
	s.nav.Refresh()
	row := s.rowFor(s.current)
	if row < 0 {
		s.nav.UnselectAll()
		return
	}
	// Quietly: the selection is being restored to where it already was, and a
	// swap here would rebuild the section and tell it it had been arrived at
	// because a heading was clicked.
	s.navQuiet = true
	s.nav.Select(row)
	s.navQuiet = false
}

// navIndent is the width a member's row is pushed in by. A rectangle rather
// than padding on the label: the icon is what the eye lines up on, and it is
// the icon that has to move.
func navIndent() *canvas.Rectangle {
	r := canvas.NewRectangle(color.Transparent)
	r.SetMinSize(fyne.NewSize(fynetheme.Padding()*3, 1))
	return r
}

// navTwisty is the heading's open/closed mark.
func navTwisty(closed bool) fyne.Resource {
	if closed {
		return fynetheme.MenuExpandIcon()
	}
	return fynetheme.MenuDropDownIcon()
}

// newNavList is the section list: headings, their members indented beneath
// them, and everything else as it always was.
func (s *Shell) newNavList() *widget.List {
	list := widget.NewList(
		func() int { return len(s.rows) },
		func() fyne.CanvasObject {
			return &fyne.Container{
				Layout: navRowLayout{},
				Objects: []fyne.CanvasObject{
					navIndent(),
					widget.NewIcon(fynetheme.MenuExpandIcon()),
					widget.NewIcon(fynetheme.HomeIcon()),
					widget.NewLabel("placeholder"),
				},
			}
		},
		func(i widget.ListItemID, obj fyne.CanvasObject) {
			if i < 0 || i >= len(s.rows) {
				return
			}
			row := s.rows[i]
			c := obj.(*fyne.Container)
			indent := c.Objects[0].(*canvas.Rectangle)
			twisty := c.Objects[1].(*widget.Icon)
			icon := c.Objects[2].(*widget.Icon)
			label := c.Objects[3].(*widget.Label)

			if row.sec < 0 {
				g := s.opts.Groups[row.group]
				indent.Hide()
				twisty.SetResource(navTwisty(s.closed[g.Title]))
				twisty.Show()
				setNavIcon(icon, g.resource())
				setNavLabel(label, g.Title, true)
				return
			}
			twisty.Hide()
			if row.group >= 0 {
				indent.Show()
			} else {
				indent.Hide()
			}
			setNavIcon(icon, s.opts.Sections[row.sec].Icon())
			setNavLabel(label, s.opts.Sections[row.sec].Title(), false)
		},
	)
	list.OnSelected = func(i widget.ListItemID) {
		if i < 0 || i >= len(s.rows) {
			return
		}
		row := s.rows[i]
		if row.sec < 0 {
			// A heading is not a place. Put the selection back where it was
			// and open or close the group instead.
			s.setGroupClosed(row.group, !s.closed[s.opts.Groups[row.group].Title])
			return
		}
		s.current = row.sec
		if s.navQuiet {
			return
		}
		s.swap(false)
	}
	return list
}

// setNavIcon shows a resource, or takes the icon's room back when there is
// none: a blank square in front of a title reads as an icon that failed.
func setNavIcon(icon *widget.Icon, r fyne.Resource) {
	if r == nil {
		icon.Hide()
		return
	}
	icon.SetResource(r)
	icon.Show()
}

// setNavLabel writes a row's text, bold for a heading.
func setNavLabel(label *widget.Label, text string, bold bool) {
	style := fyne.TextStyle{Bold: bold}
	if label.TextStyle != style {
		label.TextStyle = style
	}
	label.SetText(text)
}

/*
navRowLayout lays a row out the way an HBox would, except that the twisty and
the indent occupy the same column.

An HBox gives every visible object its own width, so a heading's twisty and a
member's indent would sit at different depths and the icons below a heading
would not line up with the heading's own. One column, one width, and whichever
of the two is shown fills it.
*/
type navRowLayout struct{}

// MinSize implements fyne.Layout.
func (navRowLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	pad := fynetheme.Padding()
	w, h := float32(0), float32(0)
	for _, o := range objects {
		if !o.Visible() {
			continue
		}
		m := o.MinSize()
		if w > 0 {
			w += pad
		}
		w += m.Width
		h = max(h, m.Height)
	}
	return fyne.NewSize(w, h)
}

// Layout implements fyne.Layout.
func (navRowLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	pad := fynetheme.Padding()
	x := float32(0)
	for _, o := range objects {
		if !o.Visible() {
			continue
		}
		m := o.MinSize()
		o.Move(fyne.NewPos(x, (size.Height-m.Height)/2))
		o.Resize(fyne.NewSize(m.Width, m.Height))
		x += m.Width + pad
	}
}
