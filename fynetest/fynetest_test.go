package fynetest_test

import (
	"os"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/require"

	"github.com/ushineko/fynedesygn/fynetest"
)

// A scrolled section is a widget, not a container, and so is a tab set; a
// walker that only follows containers never sees the button inside either.
func TestWalkReachesInsideScrollsAndTabs(t *testing.T) {
	test.NewTempApp(t)
	scrolled := widget.NewButton("scrolled", nil)
	tabbed := widget.NewButton("tabbed", nil)
	tree := container.NewVBox(
		widget.NewLabel("head"),
		container.NewVScroll(container.NewHBox(scrolled)),
		container.NewAppTabs(
			container.NewTabItem("first", widget.NewLabel("shown")),
			container.NewTabItem("second", container.NewVBox(tabbed)),
		),
	)

	require.Same(t, scrolled, fynetest.FindButton(tree, "scrolled"))
	require.Same(t, tabbed, fynetest.FindButton(tree, "tabbed"), "an unselected tab is still part of the tree")
	require.Nil(t, fynetest.FindButton(tree, "absent"))
}

func TestWalkReachesInsideDocTabs(t *testing.T) {
	test.NewTempApp(t)
	entry := widget.NewEntry()
	tree := container.NewDocTabs(
		container.NewTabItem("a", widget.NewLabel("a")),
		container.NewTabItem("b", entry),
	)

	require.Same(t, entry, fynetest.FindEntry(tree))
}

func TestWalkStopsWhenVisitSaysSo(t *testing.T) {
	tree := container.NewVBox(widget.NewLabel("one"), widget.NewLabel("two"))
	var seen int
	stopped := fynetest.Walk(tree, func(o fyne.CanvasObject) bool {
		_, isLabel := o.(*widget.Label)
		if isLabel {
			seen++
		}
		return isLabel
	})

	require.True(t, stopped)
	require.Equal(t, 1, seen)
	require.False(t, fynetest.Walk(tree, func(fyne.CanvasObject) bool { return false }))
}

func TestFindReturnsZeroValueWhenAbsent(t *testing.T) {
	tree := container.NewVBox(widget.NewLabel("only a label"))

	require.Nil(t, fynetest.Find[*widget.Slider](tree))
	require.Nil(t, fynetest.FindCheck(tree))
	require.Nil(t, fynetest.FindEntry(tree))
	require.Nil(t, fynetest.FindSlider(tree))
	require.Nil(t, fynetest.FindSelect(tree))
	require.Nil(t, fynetest.FindLabel(tree, "some other text"))
	require.Nil(t, fynetest.Find[*widget.Label](nil))
}

func TestFindReturnsTheFirstMatchInWalkOrder(t *testing.T) {
	first := widget.NewSlider(0, 1)
	second := widget.NewSlider(0, 2)
	sel := widget.NewSelect([]string{"x"}, nil)
	label := widget.NewLabel("wanted")
	tree := container.NewVBox(first, container.NewHBox(second, sel), widget.NewLabel("other"), label)

	require.Same(t, first, fynetest.Find[*widget.Slider](tree))
	require.Same(t, first, fynetest.FindSlider(tree))
	require.Same(t, sel, fynetest.FindSelect(tree))
	require.Same(t, label, fynetest.FindLabel(tree, "wanted"))
	require.Nil(t, fynetest.FindLabel(tree, "want"), "FindLabel matches the whole text, not a prefix")
}

// Text is how a test asks "is this fact on screen" without knowing the layout.
func TestTextListsEveryStringTheTreeShowsOnce(t *testing.T) {
	test.NewTempApp(t)
	tree := container.NewVBox(
		widget.NewLabel("a"),
		widget.NewButton("b", nil),
		widget.NewCheck("c", nil),
	)

	require.Equal(t, []string{"a", "b", "c"}, fynetest.Texts(tree))
	require.Equal(t, "a\nb\nc", fynetest.Text(tree))
}

func TestTextSeesRichTextSegments(t *testing.T) {
	test.NewTempApp(t)
	rt := widget.NewRichTextFromMarkdown("hello **world**")

	text := fynetest.Text(rt)
	require.Contains(t, text, "hello")
	require.Contains(t, text, "world")
}

func TestTextSeesHyperlinksAndTextInsideWidgets(t *testing.T) {
	test.NewTempApp(t)
	tree := container.NewVBox(
		widget.NewHyperlink("link", nil),
		container.NewVScroll(widget.NewCard("title", "subtitle", widget.NewLabel("body"))),
	)

	text := fynetest.Text(tree)
	require.Contains(t, text, "link")
	require.Contains(t, text, "title", "a Card hides its children from Walk; Text opens the renderer")
	require.Contains(t, text, "body")
}

func TestScrollableInFindsAScrollAndNotALabel(t *testing.T) {
	test.NewTempApp(t)

	require.True(t, fynetest.ScrollableIn(container.NewVScroll(widget.NewLabel("x"))))
	require.True(t, fynetest.ScrollableIn(container.NewVBox(widget.NewLabel("x"), container.NewHScroll(widget.NewLabel("y")))))
	require.False(t, fynetest.ScrollableIn(widget.NewLabel("x")))
	require.False(t, fynetest.ScrollableIn(container.NewVBox(widget.NewLabel("x"))))
}

/*
Canary #6 from docs/fyne-quirks.md, not a contract: Fyne 2.8.1's
richCodeBlock wraps its label in an HScroll, which takes the wheel from the
section around it. markdown.CodePanel exists to avoid that. When this fails,
Fyne has changed and CodePanel can be reconsidered.
*/
func TestFyneStillDrawsMarkdownCodeInsideAScroll(t *testing.T) {
	test.NewTempApp(t)
	rt := widget.NewRichTextFromMarkdown("    code\n")
	rt.Resize(fyne.NewSize(400, 100))

	require.True(t, fynetest.ScrollableIn(rt))
}

func TestSandboxMovesHomeIntoATempDir(t *testing.T) {
	before, err := os.UserHomeDir()
	require.NoError(t, err)

	home := fynetest.Sandbox(t)

	after, err := os.UserHomeDir()
	require.NoError(t, err)
	require.Equal(t, home, after)
	require.NotEqual(t, before, after)
	require.DirExists(t, os.Getenv("XDG_CONFIG_HOME"))
	require.DirExists(t, os.Getenv("XDG_DATA_HOME"))
	require.DirExists(t, os.Getenv("XDG_CACHE_HOME"))
}

func TestAppIsACurrentTestAppInASandbox(t *testing.T) {
	app := fynetest.App(t)

	require.Same(t, app, fyne.CurrentApp())
	home, err := os.UserHomeDir()
	require.NoError(t, err)
	require.DirExists(t, home)
	require.Equal(t, home, os.Getenv("HOME"))
}
