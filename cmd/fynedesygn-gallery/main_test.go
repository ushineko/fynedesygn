package main

import (
	"testing"

	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/require"

	"github.com/ushineko/fynedesygn/fynetest"
	"github.com/ushineko/fynedesygn/markdown"
	"github.com/ushineko/fynedesygn/shell"
	fdtheme "github.com/ushineko/fynedesygn/theme"
)

func testShell(t *testing.T) *shell.Shell {
	t.Helper()
	fdtheme.RescanFonts(t.TempDir())
	t.Cleanup(func() { fdtheme.RescanFonts() })
	return shell.Headless(fynetest.App(t), options("", ""))
}

func TestSectionNamesNeedNoApp(t *testing.T) {
	// Canary #8 (docs/fyne-quirks.md): the names are read before any Fyne app
	// exists, as --help and --version do.
	require.Equal(t, []string{"Appearance", "Widgets", "Table", "Fonts", "Shell", "Dialogs", "Log", "Forms", "Glance", "Documents", "About"}, shell.Names(sections()))
	require.True(t, known(shell.Names(sections()), "table"))
	require.False(t, known(shell.Names(sections()), "nope"))
}

func TestEverySectionRendersHeadlesslyInEveryScheme(t *testing.T) {
	s := testShell(t)
	for _, scheme := range fdtheme.SchemeNames() {
		a := s.Appearance()
		a.Scheme = scheme
		s.SetAppearance(a)
		for _, sec := range s.Sections() {
			o := sec.Build(s)
			require.NotPanics(t, func() {
				w := test.NewWindow(o)
				defer w.Close()
				w.Resize(o.MinSize())
			}, "%s in %s", sec.Title(), scheme)
		}
	}
}

func TestTheWidgetsSectionNamesEveryPrimitive(t *testing.T) {
	s := testShell(t)
	text := fynetest.Text(buildWidgets(s))
	for _, name := range []string{"widgets.Heading", "widgets.Card", "widgets.Action", "widgets.AboutNote", "widgets.FixedHeight"} {
		require.Contains(t, text, name)
	}
}

func TestTheShellSectionRunsItsJobInlineWhenHeadless(t *testing.T) {
	s := testShell(t)
	demo := newJobDemo()
	o := demo.build(s)
	// A headless Perform runs inline, so the button returns with the job done.
	// The demo job sleeps a tenth of a second per tick; ask for zero seconds.
	quick := fynetest.FindButton(o, "Run and fail (2 s)")
	require.NotNil(t, quick)
	text := fynetest.Text(o)
	require.Contains(t, text, "State: idle")
	require.Contains(t, text, "shell.Flash")
}

func TestTheStatusBarSegmentFollowsTheJobState(t *testing.T) {
	s := testShell(t)
	demo := newJobDemo()
	demo.lastState, demo.runs = "finished", 3
	text := ""
	for _, seg := range demo.statusBar(s) {
		text += fynetest.Text(seg) + " "
	}
	require.Contains(t, text, "finished")
	require.Contains(t, text, "3")
}

func TestTheDocumentsSectionShowsTheEmbeddedGuideWithItsDiagram(t *testing.T) {
	s := testShell(t)
	var doc shell.Section
	for _, sec := range s.Sections() {
		if sec.Title() == "Documents" {
			doc = sec
		}
	}
	require.NotNil(t, doc)
	o := doc.Build(s)
	text := fynetest.Text(o)
	require.Contains(t, text, "Rendering Markdown")
	require.NotContains(t, text, markdown.NotRenderedCaption, "the flowchart's PNG pair is embedded")
	require.False(t, fynetest.ScrollableIn(o), "nothing in the document takes the wheel")
	_, isDetacher := doc.(shell.Detacher)
	require.True(t, isDetacher)
}

func TestTheAboutSectionShowsTheReadme(t *testing.T) {
	s := testShell(t)
	about := s.Sections()[len(s.Sections())-1]
	require.Equal(t, "About", about.Title())
	text := fynetest.Text(about.Build(s))
	require.Contains(t, text, "README")
	require.Contains(t, text, "design system and wrapper library")
}

/*
The gallery follows its own affixing rule (spec 014).

The Log section broke it: Start and Stop sat in the scrolling half of a split
whose other half is a log pane, which is the shape the rule is written against
-- below the fold at an ordinary window height, and unreachable rather than
merely hidden, because the wheel goes to whichever scrollable is under the
pointer.

The other sections are exempt on purpose and the exemption is the interesting
part: a gallery card's buttons are the exhibit the card's prose describes, so
they are the material and travel with it. The rule is about controls that act
on a section, not about a demonstration of a control.
*/
func TestTheLogSectionsControlsAreAffixed(t *testing.T) {
	s := testShell(t)
	for _, sec := range s.Sections() {
		if sec.Title() != "Log" {
			continue
		}
		scrolled := fynetest.ScrolledButtons(sec.Build(s))
		require.NotContains(t, scrolled, "Start a noisy job",
			"a control that starts work must not scroll away from the section it acts on")
		require.NotContains(t, scrolled, "Stop")
		return
	}
	t.Fatal("no Log section")
}
