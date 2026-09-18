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

// scrollBy moves the scroll the way a wheel notch does, so that OnScrolled
// fires: assigning Offset does not.
func scrollBy(sc *container.Scroll, dy float32) {
	sc.Scrolled(&fyne.ScrollEvent{Scrolled: fyne.NewDelta(0, dy)})
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
	before := p.MinSize().Height
	require.Greater(t, before, float32(500))
	for range 20 {
		scrollBy(sc, -400)
	}
	require.NotZero(t, p.Live())
	require.Equal(t, before, p.MinSize().Height, "the document changed height while scrolling")
}

func TestThePaneMeasuresAgainWhenTheWindowNarrows(t *testing.T) {
	p, _ := paneInScroll(t, 900, 500)
	wide := p.MinSize().Height
	p.Resize(fyne.NewSize(450, p.Size().Height))
	require.Greater(t, p.MinSize().Height, wide, "narrowing makes the document taller")
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
