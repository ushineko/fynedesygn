package shell

import (
	"path/filepath"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/require"

	"github.com/ushineko/fynedesygn/logpane"
)

// blank is a section with nothing in it, for a navigation entry a test selects
// but does not look at.
func blank(*Shell) fyne.CanvasObject { return widget.NewLabel("") }

// splitIn finds the split a shell handed out for a key.
func splitIn(t *testing.T, s *Shell, key string) *container.Split {
	t.Helper()
	sp, ok := s.splits[key]
	require.Truef(t, ok, "no live split for %q", key)
	return sp
}

// withSettings is a shell whose settings go to a file this test owns.
func withSettings(t *testing.T, sections ...Section) (*Shell, Options) {
	t.Helper()
	o := testOptions(sections...)
	o.SettingsPath = filepath.Join(t.TempDir(), "settings.json")
	return headless(t, o), o
}

/*
A divider opens where it was left, or where the caller asked.

The caller's offset is the one for a program nobody has configured; a position
the user dragged it to wins from then on.
*/
func TestASplitOpensAtTheStoredPositionOrTheDefault(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	fresh := headless(t, testOptionsAt(path, NewSection("A", nil, blank)))
	body := fresh.VSplit("output", 0.8, widget.NewLabel("section"), widget.NewLabel("log"))
	require.InDelta(t, 0.8, body.(*container.Split).Offset, 0.001,
		"a program nobody has configured opens where the caller asked")

	// Dragged, written, and opened again by a second shell over the same file.
	body.(*container.Split).SetOffset(0.42)
	fresh.Stop()

	next := Headless(fresh.App, testOptionsAt(path, NewSection("A", nil, blank)))
	again := next.VSplit("output", 0.8, widget.NewLabel("section"), widget.NewLabel("log"))
	require.InDelta(t, 0.42, again.(*container.Split).Offset, 0.001)
}

/*
A drag survives the section being rebuilt.

Sections rebuild on every refresh and every selection, and each rebuild makes a
new split. Without the shell holding the position, a drag would be undone by the
next status-driven redraw -- which is exactly what happened in the program that
asked for this.
*/
func TestADragSurvivesASectionRebuild(t *testing.T) {
	built := 0
	sec := NewSection("A", nil, func(s *Shell) fyne.CanvasObject {
		built++
		return s.VSplit("output", 0.8, widget.NewLabel("section"), widget.NewLabel("log"))
	})
	o := testOptions(sec, NewSection("B", nil, blank))
	o.SettingsPath = filepath.Join(t.TempDir(), "settings.json")
	s := onScreen(t, o)
	require.Positive(t, built)

	splitIn(t, s, "output").SetOffset(0.35) // the user drags the bar down

	s.Refresh()
	require.InDelta(t, 0.35, splitIn(t, s, "output").Offset, 0.001, "rebuilt where it was left")

	// And away to another section and back, which is a build rather than a
	// refresh, and runs the detach and arrive paths on the way.
	s.Select("B")
	s.Select("A")
	require.InDelta(t, 0.35, splitIn(t, s, "output").Offset, 0.001)
}

/*
A drag survives the process.

The positions are one section of the settings file, written when the window
closes like everything else.
*/
func TestADragSurvivesTheProcess(t *testing.T) {
	sec := func() Section {
		return NewSection("A", nil, func(s *Shell) fyne.CanvasObject {
			return s.VSplit("output", 0.8, widget.NewLabel("section"), widget.NewLabel("log"))
		})
	}
	o := testOptions(sec())
	o.SettingsPath = filepath.Join(t.TempDir(), "settings.json")
	s := onScreen(t, o)

	splitIn(t, s, "output").SetOffset(0.28)
	s.Stop()

	next := Headless(s.App, testOptionsAt(o.SettingsPath, sec()))
	body := sec().Build(next)
	require.InDelta(t, 0.28, body.(*container.Split).Offset, 0.001)
}

// testOptionsAt is testOptions pointed at a settings file.
func testOptionsAt(path string, sections ...Section) Options {
	o := testOptions(sections...)
	o.SettingsPath = path
	return o
}

/*
A position that would hide a pane is ignored.

Zero and one are what a split reports when a pane has been dragged shut. Obeying
one on the next run opens a window with a pane missing, and often the missing
pane is the one holding the control that would bring it back.
*/
func TestAPositionThatWouldHideAPaneIsIgnored(t *testing.T) {
	s, _ := withSettings(t, NewSection("A", nil, nil))
	for _, bad := range []float64{0, 1, -0.5, 4} {
		s.splitPos["output"] = bad
		got := s.VSplit("output", 0.75, widget.NewLabel("a"), widget.NewLabel("b"))
		require.InDeltaf(t, 0.75, got.(*container.Split).Offset, 0.001, "%v was obeyed", bad)
	}
}

/*
Two sections sharing a key share one position.

A log pane under every section is one pane as far as the user is concerned, and
it should not jump when the section changes.
*/
func TestTwoSectionsWithOneKeyShareOnePosition(t *testing.T) {
	build := func(s *Shell) fyne.CanvasObject {
		return s.VSplit("output", 0.8, widget.NewLabel("section"), widget.NewLabel("log"))
	}
	o := testOptions(NewSection("A", nil, build), NewSection("B", nil, build))
	o.SettingsPath = filepath.Join(t.TempDir(), "settings.json")
	s := onScreen(t, o)

	splitIn(t, s, "output").SetOffset(0.33)
	s.Select("B")
	require.InDelta(t, 0.33, splitIn(t, s, "output").Offset, 0.001)
}

/*
The navigation's own divider is one of these.

It goes through the same call as a program's, under a reserved key, so the width
of the section list is something the user can set and keep -- which it was not
before, because the shell applied a constant and never read it back.
*/
func TestTheNavigationDividerIsRememberedToo(t *testing.T) {
	o := testOptions(NewSection("A", nil, blank), NewSection("B", nil, blank))
	o.SettingsPath = filepath.Join(t.TempDir(), "settings.json")
	s := onScreen(t, o)

	nav := splitIn(t, s, NavSplitKey)
	require.InDelta(t, NavOffset, nav.Offset, 0.001)

	nav.SetOffset(0.3)
	s.Select("B")
	require.InDelta(t, 0.3, splitIn(t, s, NavSplitKey).Offset, 0.001,
		"the navigation is not rebuilt, so its divider is the same one")

	s.Stop()
	next := onScreen(t, testOptionsAt(o.SettingsPath, NewSection("A", nil, blank)))
	require.InDelta(t, 0.3, splitIn(t, next, NavSplitKey).Offset, 0.001)
}

/*
A log pane inside a divider grows, and stops at its minimum.

The pane's height was a fixed region, which is the one thing wrong with it: the
section where the output matters is whichever one is failing, and a fixed pane
gives it no more room.
*/
func TestALogPaneInADividerGrowsAndStopsAtItsMinimum(t *testing.T) {
	s, _ := withSettings(t, NewSection("A", nil, nil))
	pane := logpane.New(logpane.NewModel(logpane.DefaultMaxLines))
	paneWidget := pane.Widget(logpane.Options{Title: "Output", Height: 80})
	body := s.VSplit("output", 0.5, widget.NewLabel("section"), paneWidget)

	win := test.NewWindow(body)
	t.Cleanup(win.Close)
	win.Resize(fyne.NewSize(800, 900))

	tall := paneWidget.Size().Height
	require.Greater(t, tall, paneWidget.MinSize().Height,
		"half of a tall window is more than the pane's minimum")

	body.(*container.Split).SetOffset(0.95) // drag the bar down, onto the pane
	body.Refresh()
	require.GreaterOrEqual(t, paneWidget.Size().Height, paneWidget.MinSize().Height,
		"and it stops at its minimum rather than disappearing")
	require.Less(t, paneWidget.Size().Height, tall)
}
