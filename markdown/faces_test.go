package markdown

import (
	"os"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/require"
)

/*
Fyne dereferences a font the theme does not have.

The canary for quirk 35. Bold around a code span parses to a segment that is
bold *and* monospace, and a theme without that face returns nothing, which the
painter then reads through. The day Fyne falls back instead, this test fails
and `drawable` can go.
*/
func TestFyneStillPanicsOnAFaceTheThemeDoesNotHave(t *testing.T) {
	a := test.NewTempApp(t)
	both := fyne.TextStyle{Bold: true, Monospace: true}
	require.Nil(t, a.Settings().Theme().Font(both),
		"the test theme grew a bold monospace face; quirk 35 may be gone")

	// Straight to Fyne, with no drawable in the way.
	raw := widget.NewRichTextFromMarkdown("**bold with `code` inside**")
	require.Panics(t, func() { raw.Refresh(); _ = raw.MinSize() },
		"Fyne no longer panics on a missing face; drop quirk 35 and drawable")
}

// rendered puts a document in a scroll and lays it out, which is where the
// painter is asked for faces.
func rendered(t *testing.T, doc string) {
	t.Helper()
	test.NewTempApp(t)
	p := New(doc, Options{})
	sc := container.NewScroll(container.NewVBox(p))
	win := test.NewWindow(sc)
	t.Cleanup(win.Close)
	win.Resize(fyne.NewSize(900, 500))
	p.Follow(sc)
}

func TestBoldAroundACodeSpanRenders(t *testing.T) {
	/*
		How anybody emphasises an identifier in prose, and it took the README
		down. Each of these renders on its own; only the combination asked for
		a face the test theme has not got.
	*/
	for _, doc := range []string{
		"plain `code` here\n",
		"**bold** here\n",
		"**bold with `code` inside**\n",
		"a paragraph naming **`Build`** in passing\n",
	} {
		require.NotPanics(t, func() { rendered(t, doc) }, "rendering %q", doc)
	}
}

func TestATableCellKeepsItsFacesDrawable(t *testing.T) {
	// The other place this package builds a RichText from Markdown.
	require.NotPanics(t, func() {
		rendered(t, "| A | B |\n|---|---|\n| **`Build`** | plain |\n")
	})
}

func TestTheReadmeRenders(t *testing.T) {
	// The test that found the crash, said directly: this module's own front
	// page is a document this module has to be able to draw.
	b, err := os.ReadFile("../README.md")
	require.NoError(t, err)
	require.NotPanics(t, func() { rendered(t, string(b)) })
}

func TestABoldCodeSpanKeepsTheCodeAndLosesTheEmphasis(t *testing.T) {
	/*
		Monospace is the one that carries: it says the run *is* code, which is
		why the author reached for backticks. Emphasis that cannot be drawn is
		the smaller loss.
	*/
	test.NewTempApp(t)
	rt := drawable(widget.NewRichTextFromMarkdown("**`Build`**"))

	var checked int
	for _, seg := range rt.Segments {
		text, ok := seg.(*widget.TextSegment)
		if !ok || text.Text == "" {
			continue
		}
		require.True(t, text.Style.TextStyle.Monospace, "%q stopped being code", text.Text)
		require.False(t, text.Style.TextStyle.Bold, "%q kept a face the theme has not got", text.Text)
		checked++
	}
	require.Positive(t, checked, "no text segment to check")
}

func TestAThemeWithTheFaceKeepsBoth(t *testing.T) {
	/*
		Conditional on the theme rather than always. fynedesygn's own theme
		has a bold monospace face, and a program using it should get bold
		monospace -- stripping unconditionally would make every real program
		pay for a gap in a test fixture.
	*/
	a := test.NewTempApp(t)
	a.Settings().SetTheme(bothFaces{a.Settings().Theme()})

	rt := drawable(widget.NewRichTextFromMarkdown("**`Build`**"))
	var seen bool
	for _, seg := range rt.Segments {
		if text, ok := seg.(*widget.TextSegment); ok && text.Text != "" {
			require.True(t, text.Style.TextStyle.Bold, "bold was dropped by a theme that has the face")
			seen = true
		}
	}
	require.True(t, seen)
}

// bothFaces is a theme that answers for every style, standing in for the ones
// that do have a bold monospace face.
type bothFaces struct{ fyne.Theme }

func (b bothFaces) Font(style fyne.TextStyle) fyne.Resource {
	if f := b.Theme.Font(style); f != nil {
		return f
	}
	return b.Theme.Font(fyne.TextStyle{Monospace: true})
}
