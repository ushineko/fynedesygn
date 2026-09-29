package markdown

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/require"

	"github.com/ushineko/fynedesygn/fynetest"
	"github.com/ushineko/fynedesygn/mermaid"
	fdtheme "github.com/ushineko/fynedesygn/theme"
)

// readme is this module's README: the long document the pane exists for.
func readme(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile("../README.md")
	require.NoError(t, err)
	return string(b)
}

// paneInScroll is a pane holding the README, laid out inside a scroll the
// size of a section pane, which is how an About section uses it.
func paneInScroll(t *testing.T, w, h float32) (*Pane, *container.Scroll) {
	t.Helper()
	test.NewTempApp(t)
	p := New(readme(t), Options{})
	sc := container.NewScroll(container.NewVBox(p))
	win := test.NewWindow(sc)
	t.Cleanup(win.Close)
	win.Resize(fyne.NewSize(w, h))
	p.Follow(sc)
	return p, sc
}

/*
scrollBy moves the scroll a wheel notch's worth and tells the pane, without
going through Scroll.Scrolled.

Quirk 41. Where the scrollbars auto-hide -- macOS -- Scrolled starts a 500 ms
timer that refreshes the bars through fyne.Do, and the test driver runs
fyne.Do inline, on that timer's goroutine. A notch over this document takes
longer than the delay on a CI runner, so the refresh lands in the middle of
the next notch and measures text while the test is measuring it. Linux and
Windows never see it: scrollBarAlwaysVisible is a constant true there and the
timer is never started.

Waiting the timer out afterwards does not help, because the collision is
inside the loop. What these tests are about is the pane's reaction to the
scroll position and not Fyne's scrollbar fade, so they move the offset and
call the hook, which is what updateOffset does either way.
*/
func scrollBy(sc *container.Scroll, dy float32) {
	limit := max(sc.Content.MinSize().Height-sc.Size().Height, 0)
	y := min(max(sc.Offset.Y-dy, 0), limit)
	if y == sc.Offset.Y {
		return
	}
	sc.Offset.Y = y
	if sc.OnScrolled != nil {
		sc.OnScrolled(sc.Offset)
	}
	sc.Refresh()
}

// Canary #7 (docs/fyne-quirks.md): one RichText over a long document repaints
// every segment per wheel notch, so the pane keeps distant blocks out of the
// tree. If Fyne ever virtualises RichText itself, this is the test to revisit.
func TestDocumentRendersOnlyNearTheViewport(t *testing.T) {
	p, _ := paneInScroll(t, 900, 500)
	require.Greater(t, p.Blocks(), 20, "the README should split into many blocks")
	require.Less(t, p.Live(), p.Blocks(), "a document taller than the viewport should not render all of itself at once")
	require.False(t, p.IsLive(p.Blocks()-1), "the last block is screens away")
	require.True(t, p.IsLive(0), "the first block is on screen")
}

/*
blockHeights is what the pane has recorded for each block.

The pane's own numbers, not Fyne's total. A block that is taken out of the
tree leaves its spacer behind at the height the block measured, which is the
whole mechanism by which the document keeps its length while most of it is
not rendered -- so these are the values the contract is actually about.

Asserting on p.MinSize().Height instead put Fyne's text shaping inside the
assertion, which is how a test of our virtualisation came to fail on one
platform and not the others. See docs/style.md.
*/
func blockHeights(p *Pane) []float32 {
	out := make([]float32, len(p.spacers))
	for i, sp := range p.spacers {
		out[i] = sp.MinSize().Height
	}
	return out
}

func TestBlocksRenderAsTheyAreScrolledTo(t *testing.T) {
	p, sc := paneInScroll(t, 900, 500)
	last := p.Blocks() - 1
	for i := 0; i < 400 && !p.IsLive(last); i++ {
		scrollBy(sc, -400)
	}
	require.True(t, p.IsLive(last), "the end of the document renders once scrolled to")
	require.False(t, p.IsLive(0), "the top has been released by then")
}

func TestDocumentHeightIsTheSameWhicheverBlocksAreRendered(t *testing.T) {
	p, sc := paneInScroll(t, 900, 500)
	before := blockHeights(p)
	require.NotEmpty(t, before)

	for range 20 {
		scrollBy(sc, -400)
	}

	// The test is worth nothing if nothing was released: a pane that kept
	// every block in the tree would pass it without virtualising at all.
	require.NotZero(t, p.Live())
	require.Less(t, p.Live(), p.Blocks(), "no block was released, so nothing was under test")

	require.Equal(t, before, blockHeights(p),
		"a block changed the height it reserves depending on whether it is rendered")
}

func TestThePaneMeasuresAgainWhenTheWindowNarrows(t *testing.T) {
	p, _ := paneInScroll(t, 900, 500)
	wide := blockHeights(p)

	p.Resize(fyne.NewSize(450, p.Size().Height))

	narrow := blockHeights(p)
	require.Len(t, narrow, len(wide))
	// Some block wraps onto more lines, and none gets shorter. Which blocks
	// grow is the font's business; that the pane measured again is ours.
	taller := 0
	for i := range narrow {
		require.GreaterOrEqual(t, narrow[i], wide[i], "block %d got shorter when the pane narrowed", i)
		if narrow[i] > wide[i] {
			taller++
		}
	}
	require.NotZero(t, taller, "the pane did not measure again when it narrowed")
}

func TestDetachStopsWatchingTheScroll(t *testing.T) {
	p, sc := paneInScroll(t, 900, 500)
	require.NotNil(t, sc.OnScrolled)
	p.Detach()
	require.Nil(t, sc.OnScrolled)
}

func TestWithNoViewportEverythingRenders(t *testing.T) {
	test.NewTempApp(t)
	p := New("# One\n\nTwo\n\nThree", Options{})
	require.NotPanics(t, func() { p.Follow(nil) })
	p.Resize(fyne.NewSize(400, 100))
	require.Equal(t, 3, p.Blocks())
	require.Equal(t, 3, p.Live())
}

func TestBlocksKeepFencedCodeWholeAndEveryLine(t *testing.T) {
	require.Equal(t, []string{"intro", "```\nfirst\n\nsecond\n```", "after"},
		Blocks("intro\n\n```\nfirst\n\nsecond\n```\n\nafter"))

	// A Windows checkout may carry CRLF; Blocks normalises it, so the
	// expectation must too.
	src := strings.ReplaceAll(readme(t), "\r\n", "\n")
	var got []string
	for _, b := range Blocks(src) {
		got = append(got, strings.Split(b, "\n")...)
	}
	var want []string
	for _, line := range strings.Split(src, "\n") {
		if strings.TrimSpace(line) != "" {
			want = append(want, line)
		}
	}
	require.Equal(t, want, got)
}

func TestCodeBlockReportsTheLanguageAndStripsFencesAndIndents(t *testing.T) {
	lang, code, ok := CodeBlock("```mermaid\nflowchart LR\n  A --> B\n```")
	require.True(t, ok)
	require.Equal(t, "mermaid", lang)
	require.Equal(t, "flowchart LR\n  A --> B", code)

	lang, code, ok = CodeBlock("```sh\nmake build\n```")
	require.True(t, ok)
	require.Equal(t, "sh", lang)
	require.Equal(t, "make build", code)

	lang, code, ok = CodeBlock("    make build\n    make test")
	require.True(t, ok)
	require.Empty(t, lang)
	require.Equal(t, "make build\nmake test", code)

	_, _, ok = CodeBlock("A paragraph that happens to mention    spaces.")
	require.False(t, ok)
	_, _, ok = CodeBlock("- a list item\n  continued here")
	require.False(t, ok)
}

func TestImageBlockRecognisesOnlyWholeBlockImages(t *testing.T) {
	alt, p, ok := ImageBlock("![The window](img/window.png)")
	require.True(t, ok)
	require.Equal(t, "The window", alt)
	require.Equal(t, "img/window.png", p)
	_, p, ok = ImageBlock(`![x](a.png "title")`)
	require.True(t, ok)
	require.Equal(t, "a.png", p)
	_, _, ok = ImageBlock("Text then ![x](a.png)")
	require.False(t, ok)
	_, _, ok = ImageBlock("![x](a.png) and more")
	require.False(t, ok)
	require.True(t, isRelative("img/a.png"))
	require.False(t, isRelative("https://example.invalid/a.png"))
	require.False(t, isRelative("/abs/a.png"))
}

func TestNothingInTheDocumentsTakesTheWheel(t *testing.T) {
	test.NewTempApp(t)
	docs, err := os.ReadFile("../docs/markdown.md")
	require.NoError(t, err)
	for name, src := range map[string]string{"README": readme(t), "markdown.md": string(docs)} {
		p := New(src, Options{})
		p.Resize(fyne.NewSize(900, 700))
		for i := range p.Blocks() {
			require.Falsef(t, fynetest.ScrollableIn(p.Visual(i)), "%s block %d takes the wheel", name, i)
		}
	}
}

func pngBytes(t *testing.T, w, h int) []byte {
	t.Helper()
	var b bytes.Buffer
	require.NoError(t, png.Encode(&b, image.NewNRGBA(image.Rect(0, 0, w, h))))
	return b.Bytes()
}

func TestMermaidFencesRenderTheVariantForThePaletteOrTheSource(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()
	const flow = "flowchart LR\n  A --> B"
	fsys := fstest.MapFS{
		"diagrams/" + mermaid.FileName(flow, true):  {Data: pngBytes(t, 40, 20)},
		"diagrams/" + mermaid.FileName(flow, false): {Data: pngBytes(t, 40, 20)},
	}
	opts := Options{Diagrams: mermaid.NewSet(fsys, "")}
	block := "```mermaid\n" + flow + "\n```"

	app.Settings().SetTheme(fdtheme.New(fdtheme.BreezeDark, fdtheme.Options{}))
	d, ok := RenderBlock(block, opts).(*mermaid.Diagram)
	require.True(t, ok)
	require.Equal(t, fyne.NewSize(20, 10), d.Natural())

	app.Settings().SetTheme(fdtheme.New(fdtheme.BreezeLight, fdtheme.Options{}))
	_, ok = RenderBlock(block, opts).(*mermaid.Diagram)
	require.True(t, ok)

	// No image: the source under a caption, and nothing scrollable.
	o := RenderBlock("```mermaid\nsequenceDiagram\n  A->>B: x\n```", opts)
	text := fynetest.Text(o)
	require.Contains(t, text, NotRenderedCaption)
	require.Contains(t, text, "sequenceDiagram")
	require.False(t, fynetest.ScrollableIn(o))
	o = RenderBlock(block, Options{})
	require.Contains(t, fynetest.Text(o), NotRenderedCaption, "a nil set has no images")
}

// Canary #21 (docs/fyne-quirks.md): Fyne's image segment takes a URI and
// cannot read an embedded file, so whole-block images resolve through the
// document's own FS.
func TestRelativeImagesResolveThroughTheFS(t *testing.T) {
	test.NewTempApp(t)
	fsys := fstest.MapFS{"docs/img/window.png": {Data: pngBytes(t, 30, 10)}}
	opts := Options{FS: fsys, Dir: "docs"}
	d, ok := RenderBlock("![The window](img/window.png)", opts).(*mermaid.Diagram)
	require.True(t, ok)
	require.Equal(t, fyne.NewSize(30, 10), d.Natural())

	o := RenderBlock("![Gone](img/missing.png)", opts)
	require.Contains(t, fynetest.Text(o), "[Gone]")
	_, isRich := RenderBlock("![Remote](https://example.invalid/a.png)", opts).(*widget.RichText)
	require.True(t, isRich, "URLs stay with Fyne")
}

func TestCodePanelShowsEveryLineOfTheCode(t *testing.T) {
	test.NewTempApp(t)
	code := "make build\nmake test"
	c := NewCodePanel(code)
	c.Resize(fyne.NewSize(600, 100))
	require.Contains(t, fynetest.Text(c), code)
	require.Equal(t, code, c.Code())
	require.NotPanics(t, c.Refresh)
}

func TestTableBlockParsesPipeTables(t *testing.T) {
	rows, ok := TableBlock("| Package | Purpose |\n|---|---|\n| `a` | one \\| two |\n| b |\n")
	require.True(t, ok)
	require.Equal(t, [][]string{{"Package", "Purpose"}, {"`a`", "one | two"}, {"b", ""}}, rows)
	_, ok = TableBlock("| not a table\nno separator")
	require.False(t, ok)
	_, ok = TableBlock("plain paragraph")
	require.False(t, ok)
}

// Canary #22 (docs/fyne-quirks.md): Fyne draws a Markdown table inside a
// scroller. When this fails, Fyne has changed and renderTable can go.
func TestFyneStillDrawsMarkdownTablesInsideAScroll(t *testing.T) {
	test.NewTempApp(t)
	rt := widget.NewRichTextFromMarkdown("| a | b |\n|---|---|\n| 1 | 2 |\n")
	rt.Resize(fyne.NewSize(400, 100))
	require.True(t, fynetest.ScrollableIn(rt))

	o := RenderBlock("| a | b |\n|---|---|\n| **1** | `2` |\n", Options{})
	require.False(t, fynetest.ScrollableIn(o))
	text := fynetest.Text(o)
	require.Contains(t, text, "a")
	require.Contains(t, text, "2")
}

/*
A block is measured after it has been given its width (spec 021).

Fyne's RichText reports the size of unwrapped text until it has been resized,
and a mermaid diagram derives its height from the width it was given. Asking
either for its height before telling it how wide it is reserves one line for a
paragraph that draws as a dozen, so the document comes out short and every
block below sits above where it will be drawn.

The symptom in hotaru: the About section is rebuilt on a poll, which builds a
new pane. The new pane measured itself several screens shorter than the one it
replaced, so the document under the reader's scroll position moved -- which
reads as the page leaping to the top on the way past the diagram.
*/
func TestAFreshPaneIsTheSameHeightAsOneThatHasBeenResized(t *testing.T) {
	test.NewTempApp(t)
	src := readme(t)

	settled, sc := paneInScroll(t, 900, 600)
	for range 6 {
		scrollBy(sc, -400)
	}
	was := blockHeights(settled)

	fresh := New(src, Options{})
	sc.Content = container.NewVBox(fresh)
	sc.Refresh()
	fresh.Follow(sc)

	// Block by block rather than on the total, so a failure says which block
	// was measured differently -- which is the whole question when a document
	// comes out short -- instead of only that the two disagree.
	now := blockHeights(fresh)
	require.Len(t, now, len(was))
	for i := range was {
		require.Equal(t, was[i], now[i],
			"block %d is a different height in a rebuilt document, so everything below it moved", i)
	}
}
