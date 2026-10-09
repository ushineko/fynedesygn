package markdown

import (
	"net/url"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	fynetheme "fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ushineko/fynedesygn/fynetest"
)

func TestSlugIsGitHubs(t *testing.T) {
	for heading, want := range map[string]string{
		"Installing it":             "installing-it",
		"0.9.6 (2026-10-08)":        "096-2026-10-08",
		"What it reads, and how":    "what-it-reads-and-how",
		"snake_case and kebab-case": "snake_case-and-kebab-case",
		"  Trimmed  ":               "trimmed",
		"Ünïcode letters stay":      "ünïcode-letters-stay",
		"Symbols: & / ? ! drop":     "symbols-----drop",
	} {
		require.Equal(t, want, Slug(heading), heading)
	}
}

func TestAnchorsNumberDuplicatesAndSkipFences(t *testing.T) {
	blocks := Blocks("# Top\n\n## Usage\n\ntext\n\n```sh\n# not a heading\n```\n\n## Usage\n\n### `code` and **bold** [link](https://example.com)\n\n#nospace\n")
	got := Anchors(blocks)
	require.Equal(t, map[string]int{
		"top":                0,
		"usage":              1,
		"usage-1":            4,
		"code-and-bold-link": 5,
	}, got)
}

// paneIn is a pane under a header, in a scroller, in a window: the shape of
// an About section, where the pane is not the scroller's content itself
// and not even a direct child of it.
func paneIn(t *testing.T, src string, o Options) (*Pane, *container.Scroll) {
	t.Helper()
	fynetest.App(t)
	p := New(src, o)
	sc := container.NewVScroll(container.NewPadded(container.NewVBox(widget.NewLabel("A header above the document"), p)))
	w := test.NewWindow(sc)
	t.Cleanup(w.Close)
	w.Resize(fyne.NewSize(500, 400))
	p.Follow(sc)
	return p, sc
}

// longDoc has a contents list at the top and its target far down.
func longDoc() string {
	var b strings.Builder
	b.WriteString("# Title\n\n- [Near](#near)\n- [Far away](#far-away)\n\n## Near\n\nShort.\n\n")
	for range 60 {
		b.WriteString("A paragraph of filler that takes up a line or two of the pane.\n\n")
	}
	b.WriteString("## Far away\n\nThe end.\n\n")
	for range 20 {
		b.WriteString("More text after it, so the heading can reach the top.\n\n")
	}
	return b.String()
}

// hyperlink is the drawn hyperlink widget saying text in block i.
func hyperlink(t *testing.T, o fyne.CanvasObject, text string) *widget.Hyperlink {
	t.Helper()
	var found *widget.Hyperlink
	fynetest.WalkRendered(o, func(c fyne.CanvasObject) bool {
		if h, ok := c.(*widget.Hyperlink); ok && h.Text == text {
			found = h
			return true
		}
		return false
	})
	require.NotNil(t, found, "no hyperlink %q drawn", text)
	return found
}

func TestTappingAnAnchorScrollsItsHeadingToTheTop(t *testing.T) {
	p, sc := paneIn(t, longDoc(), Options{})
	require.Zero(t, sc.Offset.Y)

	for _, c := range []struct{ text, frag string }{{"Far away", "far-away"}, {"Near", "near"}} {
		want, ok := p.AnchorY(c.frag)
		require.True(t, ok)
		require.Positive(t, want, "the header above the pane counts")
		tapMiddle(hyperlink(t, p.Visual(1), c.text))
		require.InDelta(t, want, sc.Offset.Y, 0.5, c.text)
		i := p.Anchors()[c.frag]
		require.True(t, p.IsLive(i), "the heading scrolled to is drawn")
		d := fyne.CurrentApp().Driver()
		at := d.AbsolutePositionForObject(p.Visual(i)).Y - d.AbsolutePositionForObject(sc).Y
		require.InDelta(t, 0, at, 0.5, "%s: the heading is at the top of the viewport, not %v below it", c.text, at)
	}
}

func TestAnUnknownAnchorDoesNotScroll(t *testing.T) {
	p, sc := paneIn(t, "[Nowhere](#nowhere)\n\n# Somewhere\n", Options{})
	sc.Offset.Y = 0
	require.False(t, p.ScrollToAnchor("nowhere"))
	require.Zero(t, sc.Offset.Y)
}

func TestAnAbsoluteLinkOpensThroughTheOpener(t *testing.T) {
	var opened []string
	p, _ := paneIn(t, "See [the project](https://example.com/project#readme) and [mail](mailto:someone@example.com).\n",
		Options{OpenURL: func(u *url.URL) error { opened = append(opened, u.String()); return nil }})
	tapMiddle(hyperlink(t, p.Visual(0), "the project"))
	require.Equal(t, []string{"https://example.com/project#readme"}, opened)
}

/*
A link to another document, or to anything the pane cannot open in a
browser, is drawn as text: there is no hyperlink to tap at all. In prose, in
a list, in a table cell.
*/
func TestARelativeLinkIsPlainText(t *testing.T) {
	var opened []string
	src := "Read [the architecture](docs/architecture.md).\n\n- [Credits](../credits.md) in a list\n- [mail](mailto:someone@example.com)\n\n| Doc | Note |\n|---|---|\n| [Guide](guide.md) | relative |\n"
	p, _ := paneIn(t, src, Options{OpenURL: func(u *url.URL) error { opened = append(opened, u.String()); return nil }})
	var said []string
	for i := range p.Blocks() {
		fynetest.WalkRendered(p.Visual(i), func(c fyne.CanvasObject) bool {
			_, link := c.(*widget.Hyperlink)
			require.False(t, link, "block %d drew a hyperlink", i)
			if text, ok := c.(*canvas.Text); ok {
				said = append(said, text.Text)
			}
			return false
		})
	}
	all := strings.Join(said, " ")
	for _, text := range []string{"the architecture", "Credits", "mail", "Guide"} {
		require.Contains(t, all, text, "the link's text is still there")
	}
	require.Empty(t, opened)
}

// RenderBlock on its own has no pane to scroll, so an anchor is text too.
func TestAnAnchorOutsideAPaneIsPlainText(t *testing.T) {
	fynetest.App(t)
	o := RenderBlock("[Somewhere](#somewhere) and [web](https://example.com)", Options{})
	w := test.NewWindow(o)
	defer w.Close()
	var links []string
	fynetest.WalkRendered(o, func(c fyne.CanvasObject) bool {
		if h, ok := c.(*widget.Hyperlink); ok {
			links = append(links, h.Text)
		}
		return false
	})
	require.Equal(t, []string{"web"}, links)
}

// tapMiddle taps a hyperlink over its text: Fyne ignores a tap beside it,
// and a link alone on a line is as wide as the line, its text at the left.
func tapMiddle(h *widget.Hyperlink) {
	text := fyne.MeasureText(h.Text, fynetheme.TextSize(), fyne.TextStyle{})
	test.TapAt(h, fyne.NewPos(fynetheme.InnerPadding()+text.Width/2, h.Size().Height/2))
}

/*
Every part of a link's text takes a click, in a list of links one under
another (spec 064). Fyne's link boxes are bigger than their text and overlap
the line below, and before tight a click on the lower half of an entry went to
the next entry's box, which ignored it.

The click goes through the window's canvas, which finds the topmost tappable
object under the point as a real click does; tapping the widget directly
would skip the very step that was wrong.
*/
func TestEveryPartOfAListedLinkTakesTheClick(t *testing.T) {
	fynetest.App(t)
	var got []string
	entries := []string{"One", "Two two", "Three", "Four four four", "Five"}
	var src strings.Builder
	for _, e := range entries {
		src.WriteString("- [" + e + "](#" + Slug(e) + ")\n")
	}
	o := renderBlock(src.String(), Options{}, linkFixer{anchor: func(f string) { got = append(got, f) }})
	w := test.NewWindow(o)
	defer w.Close()
	w.Resize(fyne.NewSize(400, 300))
	d := fyne.CurrentApp().Driver()

	for _, e := range entries {
		var text *canvas.Text
		fynetest.WalkRendered(hyperlink(t, o, e), func(c fyne.CanvasObject) bool {
			text, _ = c.(*canvas.Text)
			return text != nil
		})
		require.NotNil(t, text, "no text drawn for %q", e)
		at, size := d.AbsolutePositionForObject(text), text.Size()
		for _, fy := range []float32{0.2, 0.5, 0.8} {
			got = nil
			test.TapCanvas(w.Canvas(), at.Add(fyne.NewPos(size.Width/2, size.Height*fy)))
			assert.Equal(t, []string{Slug(e)}, got, "a click %.0f%% down %q", fy*100, e)
		}
	}
}

/*
Canary for quirk 47: Fyne's own RichText, unchanged, still does not give a
click on the lower part of a listed link to that link when another link is
below it. Here the next link takes it; on a real window it was dropped. When
this fails, Fyne no longer overlaps the boxes and linkText can go.
*/
func TestFyneStillOverlapsListedLinkBoxes(t *testing.T) {
	fynetest.App(t)
	rt := widget.NewRichTextFromMarkdown("- [One](#one)\n- [Two](#two)\n")
	var got []string
	for _, item := range rt.Segments[0].(*widget.ListSegment).Items {
		for _, seg := range item.(*widget.ParagraphSegment).Texts {
			if h, ok := seg.(*widget.HyperlinkSegment); ok {
				name := h.Text
				h.OnTapped = func() { got = append(got, name) }
			}
		}
	}
	w := test.NewWindow(rt)
	defer w.Close()
	w.Resize(fyne.NewSize(400, 300))

	var text *canvas.Text
	fynetest.WalkRendered(hyperlink(t, rt, "One"), func(c fyne.CanvasObject) bool {
		text, _ = c.(*canvas.Text)
		return text != nil
	})
	require.NotNil(t, text)
	at := fyne.CurrentApp().Driver().AbsolutePositionForObject(text)
	test.TapCanvas(w.Canvas(), at.Add(fyne.NewPos(text.Size().Width/2, text.Size().Height*0.8)))
	assert.NotEqual(t, []string{"One"}, got,
		"Fyne gave the click to the link under it: the boxes no longer overlap (quirk 47)")
}
