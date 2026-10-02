package shell

import (
	"path/filepath"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	fynetheme "fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/require"

	"github.com/ushineko/fynedesygn/fynetest"
)

// arrivingSection is a logSection that also logs arrivals.
type arrivingSection struct{ logSection }

func (a *arrivingSection) Arrive() { *a.log = append(*a.log, "arrive "+a.title) }

func part(title string, log *[]string) *arrivingSection {
	return &arrivingSection{logSection{title, log}}
}

// tabbed is a shell on screen with one plain section and a Tabs of three,
// over a settings file a test can reopen.
func tabbed(t *testing.T, path string) (*Shell, *Tabs, *[]string) {
	t.Helper()
	log := &[]string{}
	tabs := NewTabs("Tweaks", fynetheme.SettingsIcon,
		part("Economy", log), part("Movement", log), part("Ships", log))
	o := testOptionsAt(path, &logSection{"Overview", log}, tabs)
	return onScreen(t, o), tabs, log
}

func strip(t *testing.T, s *Shell) []*widget.Button {
	t.Helper()
	border, ok := s.content.Content.(*fyne.Container)
	require.True(t, ok, "a Tabs builds a container")
	var out []*widget.Button
	for _, o := range border.Objects {
		box, ok := o.(*fyne.Container)
		if !ok {
			continue
		}
		for _, b := range box.Objects {
			if button, ok := b.(*widget.Button); ok {
				out = append(out, button)
			}
		}
	}
	return out
}

func TestTabsDrawOneButtonPerPartWithTheChosenOneHigh(t *testing.T) {
	s, _, _ := tabbed(t, filepath.Join(t.TempDir(), "settings.json"))
	s.Select("Tweaks")

	buttons := strip(t, s)
	require.Len(t, buttons, 3)
	for i, want := range []string{"Economy", "Movement", "Ships"} {
		require.Equal(t, want, buttons[i].Text)
	}
	require.Equal(t, widget.HighImportance, buttons[0].Importance)
	require.Equal(t, widget.MediumImportance, buttons[1].Importance)
	require.Equal(t, widget.MediumImportance, buttons[2].Importance)
	require.NotNil(t, fynetest.FindLabel(s.content.Content, "Economy"), "the first part is drawn")
}

func TestASwapDetachesEveryPartAndArrivalReachesOnlyTheOneShowing(t *testing.T) {
	s, _, log := tabbed(t, filepath.Join(t.TempDir(), "settings.json"))
	*log = nil
	s.Select("Tweaks")

	require.Subset(t, *log, []string{"detach Economy", "detach Movement", "detach Ships"},
		"a part that is not showing may still hold something the shell has to let go of")
	require.Contains(t, *log, "arrive Economy")
	require.NotContains(t, *log, "arrive Movement")
	require.NotContains(t, *log, "arrive Ships")
}

/*
Choosing a tab is arriving, not refreshing.

hotaru's copy called Invalidate, which runs the program's OnInvalidate and
throws away what it loaded -- a tab change is not a reason to refetch the game
directory.
*/
func TestChoosingATabArrivesWithoutInvalidating(t *testing.T) {
	log := &[]string{}
	tabs := NewTabs("Tweaks", nil, part("Economy", log), part("Movement", log))
	o := testOptions(tabs)
	invalidated := 0
	o.OnInvalidate = func(*Shell) { invalidated++ }
	s := onScreen(t, o)
	s.Select("Tweaks")

	*log = nil
	test.Tap(strip(t, s)[1])
	require.Equal(t, "Movement", tabs.Showing())
	require.Contains(t, *log, "build Movement")
	require.Contains(t, *log, "arrive Movement")
	require.Zero(t, invalidated)

	*log = nil
	test.Tap(strip(t, s)[1])
	require.Empty(t, *log, "pressing the tab already showing does nothing")
}

func TestTheChosenTabSurvivesNavigationAndARestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	s, tabs, _ := tabbed(t, path)
	s.Select("Tweaks")
	test.Tap(strip(t, s)[2])

	s.Select("Overview")
	s.Select("Tweaks")
	require.Equal(t, "Ships", tabs.Showing(), "leaving and coming back")

	s.Stop()
	next, again, _ := tabbed(t, path)
	next.Select("Tweaks")
	require.Equal(t, "Ships", again.Showing(), "a restart")
}

func TestAStoredTabThatNamesNoPartFallsBackToTheFirst(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	s, _, _ := tabbed(t, path)
	require.NoError(t, s.store.Set(TabsKey, map[string]string{"Tweaks": "Retired"}))
	s.Stop()

	next, tabs, _ := tabbed(t, path)
	next.Select("Tweaks")
	require.Equal(t, "Economy", tabs.Showing())
}

func TestShowIsCaseInsensitiveAndRefusesAnUnknownTitle(t *testing.T) {
	tabs := NewTabs("T", nil, NewSection("Economy", nil, blank), NewSection("Ships", nil, blank))
	require.True(t, tabs.Show("ships"))
	require.Equal(t, "Ships", tabs.Showing())
	require.False(t, tabs.Show("Nowhere"))
	require.Equal(t, "Ships", tabs.Showing(), "a refused title changes nothing")
}

func TestTabsWithNoPartsBuildsNothing(t *testing.T) {
	tabs := NewTabs("Empty", nil)
	s := onScreen(t, testOptions(tabs))
	require.NotPanics(t, func() { s.Select("Empty") })
	require.Empty(t, tabs.Showing())
	require.False(t, tabs.Wide())
}

func TestTabsAreAsWideAsThePartShowing(t *testing.T) {
	narrow := NewSection("Prose", nil, blank)
	wide := NewSection("Table", nil, blank).WideContent()
	tabs := NewTabs("T", nil, narrow, wide)
	require.False(t, tabs.Wide())
	tabs.Show("Table")
	require.True(t, tabs.Wide())
}
