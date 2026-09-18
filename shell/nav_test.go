package shell

import (
	"path/filepath"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	fynetheme "fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/require"

	"github.com/ushineko/fynedesygn/fynetest"
	fdtheme "github.com/ushineko/fynedesygn/theme"
)

// everyShape is a program that allows all of them, which is what the gallery
// does.
func everyShape(o Options) Options {
	o.NavModes = []NavMode{NavLabels, NavIcons, NavHidden}
	o.NavPlacements = []NavPlacement{NavLeft, NavTop}
	return o
}

// navSections are three sections, the last without an icon of its own.
func navSections() []Section {
	return []Section{
		NewSection("One", fynetheme.HomeIcon, blank),
		NewSection("Two", fynetheme.ListIcon, blank),
		NewSection("Three", nil, blank),
	}
}

// shaped is a shell on screen with a settings file of its own.
func shaped(t *testing.T, tweak func(Options) Options) (*Shell, Options) {
	t.Helper()
	o := testOptions(navSections()...)
	o.SettingsPath = filepath.Join(t.TempDir(), "settings.json")
	if tweak != nil {
		o = tweak(o)
	}
	return onScreen(t, o), o
}

// menuLabels is what the shape control offers, in order, with a tick on the
// current one.
func menuLabels(s *Shell) []string {
	var out []string
	for _, item := range s.navMenuItems() {
		switch {
		case item.IsSeparator:
			out = append(out, "---")
		case item.Checked:
			out = append(out, "* "+item.Label)
		default:
			out = append(out, item.Label)
		}
	}
	return out
}

/*
A program that asks for nothing keeps the window it has.

The public API is a contract, and this is a change to a window four programs
already ship -- one of which captures the documentation screenshots from it. So
opting in is a line per program rather than a change imposed on all of them.
*/
func TestAProgramThatAsksForNothingKeepsItsWindow(t *testing.T) {
	s, _ := shaped(t, nil)

	body, ok := s.body().(*container.Split)
	require.True(t, ok, "the same two-child split as before")
	require.Same(t, s.nav, body.Leading)
	require.Same(t, s.content, body.Trailing)

	require.Nil(t, s.navControl(), "and nothing in the header to change it")
	require.NotContains(t, fynetest.Texts(s.header()), "Icons", "no shape menu")

	// The shortcut is not bound either: one that does nothing reads as broken.
	s.toggleNav()
	mode, place := s.NavShape()
	require.Equal(t, NavLabels, mode)
	require.Equal(t, NavLeft, place)
}

/*
The control offers exactly what the program allows, with the current one ticked.

One control whatever the number of choices, rather than a toggle for two and a
menu for three: a control that changes shape according to how many options a
program allows is one the user has to re-learn per program.
*/
func TestTheControlOffersWhatTheProgramAllows(t *testing.T) {
	two, _ := shaped(t, func(o Options) Options {
		o.NavModes = []NavMode{NavLabels, NavHidden}
		return o
	})
	require.NotNil(t, two.navControl())
	require.Equal(t, []string{"* Icons and labels", "Hidden"}, menuLabels(two))

	all, _ := shaped(t, everyShape)
	require.Equal(t, []string{
		"* Icons and labels", "Icons", "Hidden", "---",
		"* Down the left", "Along the top",
	}, menuLabels(all))

	all.SetNavShape(NavIcons, NavTop)
	require.Equal(t, []string{
		"Icons and labels", "* Icons", "Hidden", "---",
		"Down the left", "* Along the top",
	}, menuLabels(all))
}

// A shape the program does not allow is ignored rather than forced: the
// program decided what it offers and a caller cannot widen that from outside.
func TestAShapeTheProgramDoesNotAllowIsIgnored(t *testing.T) {
	s, _ := shaped(t, func(o Options) Options {
		o.NavModes = []NavMode{NavLabels, NavIcons}
		return o
	})
	s.SetNavShape(NavHidden, NavLeft)
	mode, _ := s.NavShape()
	require.Equal(t, NavLabels, mode)

	// All or nothing: a combination the program does not offer is not taken
	// apart and half-applied, because half of a shape nobody allowed is still
	// a shape nobody allowed.
	s.SetNavShape(NavIcons, NavTop)
	mode, place := s.NavShape()
	require.Equal(t, NavLabels, mode)
	require.Equal(t, NavLeft, place)

	s.SetNavShape(NavIcons, NavLeft)
	mode, _ = s.NavShape()
	require.Equal(t, NavIcons, mode, "and one it does offer is taken")
}

/*
Ctrl+B hides the navigation and brings back the shape it had.

And the control that does it is in the header, which is always drawn: a
navigation that can be hidden with no way to bring it back is a broken window,
not a compact one.
*/
func TestHidingTheNavigationAlwaysHasAWayBack(t *testing.T) {
	s, _ := shaped(t, everyShape)
	s.SetNavShape(NavIcons, NavLeft)

	s.toggleNav()
	mode, _ := s.NavShape()
	require.Equal(t, NavHidden, mode)
	require.Same(t, s.content, s.body(), "the content takes the whole window")

	require.NotNil(t, s.navControl(), "the control is still there")
	require.Contains(t, menuLabels(s), "* Hidden")

	s.toggleNav()
	mode, _ = s.NavShape()
	require.Equal(t, NavIcons, mode, "back to the shape it had, not to the default")
}

/*
Each shape lays out as it should, and the content takes the rest.

Icons on the left is a strip rather than a split, because a divider on something
sized to its icons has nothing to give.
*/
func TestEachShapeLaysOutWhereItShould(t *testing.T) {
	s, _ := shaped(t, everyShape)

	s.SetNavShape(NavLabels, NavLeft)
	split, ok := s.body().(*container.Split)
	require.True(t, ok)
	require.False(t, split.Horizontal == false, "the section list is beside the content")
	require.Same(t, s.nav, split.Leading)

	s.SetNavShape(NavIcons, NavLeft)
	body, ok := s.body().(*fyne.Container)
	require.True(t, ok, "a strip beside the content, not a split")
	require.Contains(t, body.Objects, fyne.CanvasObject(s.navHolder))
	require.Contains(t, body.Objects, fyne.CanvasObject(s.content))

	s.SetNavShape(NavIcons, NavTop)
	body, ok = s.body().(*fyne.Container)
	require.True(t, ok)
	require.Contains(t, body.Objects, fyne.CanvasObject(s.navHolder))

	s.SetNavShape(NavLabels, NavTop)
	require.Contains(t, fynetest.Texts(s.body()), "Three", "labels along the top are labels")

	s.SetNavShape(NavHidden, NavTop)
	require.Same(t, s.content, s.body())
}

/*
An icon carries its section's title, and a section with no icon still has one.

Without the title an icons-only navigation is a row of pictures to guess at, and
a blank button is a section nobody can reach.
*/
func TestAnIconsOnlyNavigationStillNamesItsSections(t *testing.T) {
	s, _ := shaped(t, everyShape)
	s.SetNavShape(NavIcons, NavLeft)

	tips := fynetest.Tips(s.body())
	for _, title := range []string{"One", "Two", "Three"} {
		require.Containsf(t, tips, title, "%q is not named anywhere", title)
	}
	require.NotContains(t, fynetest.Texts(s.body()), "One",
		"icons only means icons: the titles are in the tips, not under them")

	// "Three" has no icon of its own and still has a button, because a blank
	// one would be a section nobody could reach.
	require.Nil(t, s.opts.Sections[2].Icon())
	require.Len(t, tips, 3)
}

/*
Changing shape does not change which section is open, or rebuild it.

The shape is the window's, not the section's: a user who narrows the navigation
while reading something is still reading it afterwards.
*/
func TestChangingShapeKeepsTheSectionAndDoesNotRebuildIt(t *testing.T) {
	built := map[string]int{}
	count := func(title string) func(*Shell) fyne.CanvasObject {
		return func(s *Shell) fyne.CanvasObject { built[title]++; return blank(s) }
	}
	o := testOptions(
		NewSection("One", fynetheme.HomeIcon, count("One")),
		NewSection("Two", fynetheme.ListIcon, count("Two")),
		NewSection("Three", nil, count("Three")),
	)
	o.SettingsPath = filepath.Join(t.TempDir(), "settings.json")
	s := onScreen(t, everyShape(o))

	s.Select("Three")
	require.Equal(t, "Three", s.Current().Title())
	was := built["Three"]

	s.SetNavShape(NavIcons, NavTop)
	s.SetNavShape(NavHidden, NavTop)
	s.SetNavShape(NavLabels, NavLeft)

	require.Equal(t, "Three", s.Current().Title())
	require.Equal(t, was, built["Three"], "the section on screen is not rebuilt by a reshape")
}

// Selecting works from a button navigation as well as from the list.
func TestASectionCanBeSelectedFromEitherNavigation(t *testing.T) {
	s, _ := shaped(t, everyShape)
	s.SetNavShape(NavIcons, NavTop)

	s.Select("Two")
	require.Equal(t, "Two", s.Current().Title())
	s.selectIndex(2)
	require.Equal(t, "Three", s.Current().Title())
	s.selectIndex(99)
	require.Equal(t, "Three", s.Current().Title(), "an index nobody has is ignored")
}

/*
A shape chosen in one run is the shape at the start of the next.

And a stored shape the program has since withdrawn opens at the first one it
does allow, rather than in a shape it no longer supports.
*/
func TestAChosenShapeIsKeptAndAWithdrawnOneIsNot(t *testing.T) {
	s, o := shaped(t, everyShape)
	s.SetNavShape(NavIcons, NavTop)
	s.Stop()

	next := Headless(s.App, everyShape(testOptionsAt(o.SettingsPath, navSections()...)))
	mode, place := next.NavShape()
	require.Equal(t, NavIcons, mode)
	require.Equal(t, NavTop, place)

	// The same file, read by a program that no longer offers either.
	narrowed := testOptionsAt(o.SettingsPath, navSections()...)
	narrowed.NavModes = []NavMode{NavLabels, NavHidden}
	after := Headless(s.App, narrowed)
	mode, place = after.NavShape()
	require.Equal(t, NavLabels, mode)
	require.Equal(t, NavLeft, place)
}

// The divider survives a trip through the other shapes: it is the shell's
// position, and the shapes without a divider simply do not draw one.
func TestTheDividerSurvivesAReshape(t *testing.T) {
	s, _ := shaped(t, everyShape)
	splitIn(t, s, NavSplitKey).SetOffset(0.3)

	s.SetNavShape(NavIcons, NavLeft)
	s.SetNavShape(NavHidden, NavTop)
	s.SetNavShape(NavLabels, NavLeft)

	require.InDelta(t, 0.3, splitIn(t, s, NavSplitKey).Offset, 0.001)
}

/*
Every shape draws, in every scheme.

Five layouts is a wider matrix than the suite had, and it is the cheapest place
to catch one that draws wrong: the alternative is opening the window five times
and looking.
*/
func TestEveryShapeRendersInEveryScheme(t *testing.T) {
	shapes := []struct {
		mode  NavMode
		place NavPlacement
	}{
		{NavLabels, NavLeft}, {NavIcons, NavLeft},
		{NavLabels, NavTop}, {NavIcons, NavTop},
		{NavHidden, NavLeft},
	}
	for _, scheme := range fdtheme.Schemes() {
		for _, shape := range shapes {
			name := scheme.Name + "/" + shape.mode.String() + "/" + shape.place.String()
			t.Run(name, func(t *testing.T) {
				o := testOptions(navSections()...)
				o.SettingsPath = filepath.Join(t.TempDir(), "settings.json")
				o.Scheme = scheme.Name
				s := onScreen(t, everyShape(o))
				s.SetNavShape(shape.mode, shape.place)
				s.Window.Resize(fyne.NewSize(900, 400))
				require.NotPanics(t, func() { _ = s.Window.Canvas().Capture() })
			})
		}
	}
}

/*
The shape menu opens under the button that raises it.

A widget's Position is measured from its parent, so the button in the header's
row reports a position near the origin of that row. Opening the menu there put
it in the top-left corner of the window, across the navigation it was offering
to change.
*/
func TestTheShapeMenuOpensUnderItsButton(t *testing.T) {
	s, _ := shaped(t, everyShape)
	s.Window.Resize(fyne.NewSize(900, 400))
	// The button the window is drawing, not a fresh one: navControl builds a
	// new button each time the header is composed, and one that is not in the
	// tree has no absolute position to speak of.
	require.NotNil(t, s.navBtn)

	test.Tap(s.navBtn)
	top := s.Window.Canvas().Overlays().Top()
	require.NotNil(t, top, "the menu is up")

	// The overlay covers the canvas; the menu inside it is what was placed.
	menu := findPopUpMenu(top)
	require.NotNil(t, menu, "the overlay holds a menu")

	at := fyne.CurrentApp().Driver().AbsolutePositionForObject(s.navBtn)
	require.Greater(t, at.X, float32(0), "the button is near the trailing edge")

	// Directly below the button, and at its left edge or a little to the left
	// of it: Fyne slides a menu that would run off the canvas back inside.
	require.InDelta(t, at.Y+s.navBtn.Size().Height, menu.Position().Y, 1)
	require.LessOrEqual(t, menu.Position().X, at.X+1)
	require.Greater(t, menu.Position().X, at.X-menu.Size().Width,
		"beside the button, not in the corner of the window")
}

// findPopUpMenu is the menu inside an overlay, or nil.
func findPopUpMenu(o fyne.CanvasObject) *widget.PopUpMenu {
	switch w := o.(type) {
	case *widget.PopUpMenu:
		return w
	case *fyne.Container:
		for _, child := range w.Objects {
			if got := findPopUpMenu(child); got != nil {
				return got
			}
		}
	case fyne.Widget:
		for _, child := range w.CreateRenderer().Objects() {
			if got := findPopUpMenu(child); got != nil {
				return got
			}
		}
	}
	return nil
}
