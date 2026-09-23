package shell

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	fynetheme "fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/require"

	"github.com/ushineko/fynedesygn/fynetest"
)

// grouped is the shape spec 031 was written for: one section above a group of
// three, and two below it.
func grouped(t *testing.T) (*Shell, *[]string) {
	t.Helper()
	log := &[]string{}
	o := testOptions(
		&logSection{"Service", log},
		&logSection{"Local", log},
		&logSection{"Wallhaven", log},
		&logSection{"DuckDuckGo", log},
		&logSection{"History", log},
		&logSection{"About", log},
	)
	o.Groups = []NavGroup{{
		Title:   "Plugins",
		Icon:    fynetheme.StorageIcon,
		Members: []string{"Local", "Wallhaven", "DuckDuckGo"},
	}}
	// Every shape, so a test can move between them. Left first: it is the
	// default, and allowedPlacements()[0] is what a shell falls back to.
	o.NavModes = []NavMode{NavLabels, NavIcons, NavHidden}
	o.NavPlacements = []NavPlacement{NavLeft, NavTop}
	return onScreen(t, o), log
}

// titles is what the navigation list draws, top to bottom, with a heading
// marked so a test can tell it from a section of the same name.
func titles(s *Shell) []string {
	out := make([]string, 0, len(s.rows))
	for _, r := range s.rows {
		if r.sec < 0 {
			out = append(out, "["+s.opts.Groups[r.group].Title+"]")
			continue
		}
		out = append(out, s.opts.Sections[r.sec].Title())
	}
	return out
}

func TestAGroupDrawsItsMembersUnderAHeading(t *testing.T) {
	s, _ := grouped(t)
	require.Equal(t,
		[]string{"Service", "[Plugins]", "Local", "Wallhaven", "DuckDuckGo", "History", "About"},
		titles(s), "the heading sits where the first member was")
	require.Len(t, s.opts.Sections, 6, "a group is not a section")
	require.Equal(t, "Service", s.Current().Title())
}

func TestClosingAGroupHidesItsMembersAndKeepsThePage(t *testing.T) {
	s, log := grouped(t)
	s.Select("Wallhaven")
	require.Equal(t, "Wallhaven", s.Current().Title())

	*log = nil
	s.nav.Select(s.rowFor(1) - 1) // the heading, which is the row above Local
	require.Equal(t, []string{"Service", "[Plugins]", "History", "About"}, titles(s))
	require.Equal(t, "Wallhaven", s.Current().Title(),
		"closing a group changed the page under the reader")
	require.Empty(t, *log, "and rebuilt a section to do it")
	require.Equal(t, -1, s.rowFor(s.current), "nothing to highlight, which is honest")
}

func TestTheHeadingIsNeverWhatIsHighlighted(t *testing.T) {
	/*
		A second click on the heading has to close what the first one opened.

		It is also the only way to see, from outside widget.List, that the
		shell moved the highlight off the heading: List.Select returns early
		when the row is already the selected one, so if the heading stayed
		highlighted the second click would do nothing at all.
	*/
	s, _ := grouped(t)
	const heading = 1
	all := []string{"Service", "[Plugins]", "Local", "Wallhaven", "DuckDuckGo", "History", "About"}
	folded := []string{"Service", "[Plugins]", "History", "About"}

	s.nav.Select(heading)
	require.Equal(t, folded, titles(s))
	s.nav.Select(heading)
	require.Equal(t, all, titles(s), "the heading kept the highlight and went deaf")

	// And from inside the group, where the shell has no row to hand the
	// highlight back to and unselects instead.
	s.Select("Wallhaven")
	s.nav.Select(heading)
	require.Equal(t, folded, titles(s))
	s.nav.Select(heading)
	require.Equal(t, all, titles(s))
	require.Equal(t, "Wallhaven", s.Current().Title())
}

func TestSelectingIntoAClosedGroupOpensIt(t *testing.T) {
	s, _ := grouped(t)
	s.nav.Select(1) // close Plugins while reading Service
	require.Equal(t, []string{"Service", "[Plugins]", "History", "About"}, titles(s))

	// A "View report" button, --section, Ctrl+3: a member is still a place
	// the program can send the reader.
	s.Select("DuckDuckGo")
	require.Equal(t, "DuckDuckGo", s.Current().Title())
	require.Equal(t, []string{"Service", "[Plugins]", "Local", "Wallhaven", "DuckDuckGo", "History", "About"},
		titles(s), "the group did not open to show where the reader was sent")
}

func TestNumberKeysCountSectionsNotRows(t *testing.T) {
	// Ctrl+1..9 are bound to sections in the program's order. Folding three
	// of them under a heading must not renumber the ones below, or a heading
	// nobody clicked changes what Ctrl+5 does.
	s, _ := grouped(t)
	s.selectIndex(4)
	require.Equal(t, "History", s.Current().Title())
	s.selectIndex(5)
	require.Equal(t, "About", s.Current().Title())
}

func TestAClosedGroupIsRememberedAndAnUnknownOneIsDropped(t *testing.T) {
	s, _ := grouped(t)
	s.nav.Select(1)
	require.True(t, s.closed["Plugins"])
	var saved navGroupSettings
	require.True(t, s.store.Get(NavGroupsKey, &saved))
	require.Equal(t, []string{"Plugins"}, saved.Closed)

	// A title the program no longer groups is dropped rather than kept as a
	// key nothing reads, the way a withdrawn navigation shape is.
	require.NoError(t, s.store.Set(NavGroupsKey, navGroupSettings{Closed: []string{"Plugins", "Gone"}}))
	s.loadGroups()
	require.True(t, s.closed["Plugins"])
	require.False(t, s.closed["Gone"])
	require.Len(t, s.closed, 1)
}

func TestAGroupNamingNoSectionDrawsNothing(t *testing.T) {
	// A program builds its sections conditionally -- a plugin that is not
	// installed, a pane behind a flag -- and lists the group either way.
	log := &[]string{}
	o := testOptions(&logSection{"Service", log}, &logSection{"About", log})
	o.Groups = []NavGroup{{Title: "Plugins", Members: []string{"Local", "Wallhaven"}}}
	s := onScreen(t, o)
	require.Equal(t, []string{"Service", "About"}, titles(s), "an empty group drew a heading")
}

// navLabels is what a button navigation draws, in order.
func navLabels(o fyne.CanvasObject) []string {
	var out []string
	fynetest.Walk(o, func(c fyne.CanvasObject) bool {
		if b, ok := c.(*widget.Button); ok {
			out = append(out, b.Text)
		}
		return false
	})
	return out
}

func TestTheButtonNavigationsDrawTheGroupToo(t *testing.T) {
	// A row of buttons is the same list drawn sideways. A group that folded
	// in one shape and not the other would be two behaviours to learn -- and
	// the top placement is the one shape where the sections were most in the
	// way, because they are all on one line.
	s, _ := grouped(t)
	s.SetNavShape(NavLabels, NavTop)
	require.ElementsMatch(t,
		[]string{"Service", "Plugins", "Local", "Wallhaven", "DuckDuckGo", "History", "About"},
		navLabels(s.navButtons(true)), "the group is a button and its members are still drawn")

	s.setGroupClosed(0, true)
	require.Equal(t, []string{"Service", "Plugins", "History", "About"},
		navLabels(s.navButtons(true)), "a closed group still drew its members")
}

func TestTheTopRowKeepsTheGroupsMembersOnASecondRow(t *testing.T) {
	// The members go under the button they belong to, not in the middle of
	// the row the reader picks a group from.
	s, _ := grouped(t)
	s.SetNavShape(NavLabels, NavTop)
	rows, ok := s.navButtons(true).(*fyne.Container)
	require.True(t, ok, "an open group drew one row")
	require.Len(t, rows.Objects, 2)
	require.Equal(t, []string{"Service", "Plugins", "History", "About"}, navLabels(rows.Objects[0]))
	require.Equal(t, []string{"Local", "Wallhaven", "DuckDuckGo"}, navLabels(rows.Objects[1]))

	// Closed, there is no second row and the top row is all there is.
	s.setGroupClosed(0, true)
	require.Equal(t, []string{"Service", "Plugins", "History", "About"},
		navLabels(s.navButtons(true)))
}

func TestTheGroupButtonOpensAndClosesRatherThanNavigating(t *testing.T) {
	s, log := grouped(t)
	s.SetNavShape(NavLabels, NavTop)
	s.Select("History")
	*log = nil

	btn := fynetest.FindButton(s.navButtons(true), "Plugins")
	require.NotNil(t, btn)
	btn.Tapped(nil)
	require.True(t, s.closed["Plugins"])
	require.Equal(t, "History", s.Current().Title(), "the group button went somewhere")
	require.Empty(t, *log, "and rebuilt a section to do it")
}

func TestTheGroupButtonIsLitWhileThePageIsOneOfItsMembers(t *testing.T) {
	// Open, the branch and the leaf are both lit, which is the pair the list
	// lights too. Closed, the group is the only thing left to say where the
	// reader is -- and a button that lit only then would be changing
	// appearance for a reason nobody can see.
	s, _ := grouped(t)
	s.SetNavShape(NavLabels, NavTop)
	require.Equal(t, widget.MediumImportance,
		fynetest.FindButton(s.navButtons(true), "Plugins").Importance, "on Service")

	s.Select("Wallhaven")
	require.Equal(t, widget.HighImportance,
		fynetest.FindButton(s.navButtons(true), "Plugins").Importance, "open, on a member")
	require.Equal(t, widget.HighImportance,
		fynetest.FindButton(s.navButtons(true), "Wallhaven").Importance, "and the member itself")

	s.setGroupClosed(0, true)
	require.Equal(t, widget.HighImportance,
		fynetest.FindButton(s.navButtons(true), "Plugins").Importance, "closed, on a member")

	s.Select("History")
	require.Equal(t, widget.MediumImportance,
		fynetest.FindButton(s.navButtons(true), "Plugins").Importance, "off the group again")
}

func TestTheHeaderActionsDoNotGrowWithASecondNavRow(t *testing.T) {
	// A Border stretches what it is given to the height of its row, and the
	// row is two buttons tall while a group is open along the top. Refresh
	// drawn twice as tall as every other button reads as twice the control.
	s, _ := grouped(t)
	s.SetNavShape(NavLabels, NavTop)

	win := test.NewWindow(s.header())
	t.Cleanup(win.Close)
	win.Resize(fyne.NewSize(1200, 300))

	refresh := fynetest.FindButton(win.Content(), "Refresh")
	require.NotNil(t, refresh)
	require.Equal(t, refresh.MinSize().Height, refresh.Size().Height,
		"the header actions grew with the navigation")

	// And on the first row, which is where a window's own controls live
	// however many rows the navigation has grown to.
	first := fynetest.FindButton(win.Content(), "Service")
	require.NotNil(t, first)
	at := fyne.CurrentApp().Driver().AbsolutePositionForObject
	require.Equal(t, at(first).Y, at(refresh).Y, "Refresh sank below the first row")

	// And the navigation really is two rows tall, or the test proves nothing.
	nav := fynetest.FindButton(win.Content(), "Wallhaven")
	require.NotNil(t, nav)
	require.Greater(t, s.navHolder.Size().Height, nav.MinSize().Height*1.5)
}
