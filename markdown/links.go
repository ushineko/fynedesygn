package markdown

import (
	"image/color"
	"net/url"
	"strconv"
	"strings"
	"unicode"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	fynetheme "fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

/*
What a link in a document does (spec 064).

Fyne's Markdown widget makes every link a hyperlink, and a hyperlink that is
tapped hands its URL to the operating system: rundll32's
FileProtocolHandler on Windows, xdg-open on Linux. For an absolute web
address that is right. For `#installing-it` or `docs/architecture.md` it is
a relative string with nothing to be relative to, so nothing useful opened --
on Windows at most an error, on Linux a log line -- and a README's contents
list was a column of links that did nothing.

So each link is looked at once, when its block is rendered:

  - `#fragment` scrolls the pane to the heading with that slug, the way
    GitHub's anchors do. With no pane to scroll (RenderBlock on its own) it
    has nowhere to go and is drawn as text.
  - An absolute http or https address opens in the browser.
  - Anything else -- a path to another document, a scheme the desktop may
    not handle -- is drawn as plain text. A link that looks clickable and is
    not is worse than no link: links to other documents must be absolute
    URLs (docs/markdown.md).
*/

// linkFixer is what a block's links are rewired with: anchor scrolls to a
// fragment and is nil when there is no pane; open opens an absolute address.
type linkFixer struct {
	anchor func(fragment string)
	open   func(*url.URL) error
}

// opener is Options.OpenURL, or the app's own when the caller left it unset.
func opener(o Options) func(*url.URL) error {
	if o.OpenURL != nil {
		return o.OpenURL
	}
	return func(u *url.URL) error {
		a := fyne.CurrentApp()
		if a == nil {
			return nil
		}
		return a.OpenURL(u)
	}
}

// fix rewires every link in rt and returns it.
func (l linkFixer) fix(rt *widget.RichText) *widget.RichText {
	rt.Segments = l.segments(rt.Segments)
	return rt
}

// segments rewires links at any depth: Fyne nests them in paragraphs, list
// items and table cells.
func (l linkFixer) segments(in []widget.RichTextSegment) []widget.RichTextSegment {
	for i, seg := range in {
		switch s := seg.(type) {
		case *widget.HyperlinkSegment:
			in[i] = l.link(s)
		case *widget.ParagraphSegment:
			s.Texts = l.segments(s.Texts)
		case *widget.ListSegment:
			s.Items = l.segments(s.Items)
		case *widget.TableSegment:
			for j := range s.Headers {
				s.Headers[j] = l.segments(s.Headers[j])
			}
			for j := range s.Rows {
				for k := range s.Rows[j] {
					s.Rows[j][k] = l.segments(s.Rows[j][k])
				}
			}
		}
	}
	return in
}

// link is one hyperlink, rewired, or replaced by its text.
func (l linkFixer) link(s *widget.HyperlinkSegment) widget.RichTextSegment {
	u := s.URL
	switch {
	case u == nil:
	case IsAnchor(u) && l.anchor != nil:
		frag := u.Fragment
		s.OnTapped = func() { l.anchor(frag) }
		return s
	case IsWebURL(u):
		open := l.open
		s.OnTapped = func() {
			if err := open(u); err != nil {
				fyne.LogError("opening "+u.String(), err)
			}
		}
		return s
	}
	return &widget.TextSegment{Text: s.Text, Style: widget.RichTextStyleInline}
}

// IsAnchor reports whether u is a link within the same document: a fragment
// and nothing else.
func IsAnchor(u *url.URL) bool {
	return u.Scheme == "" && u.Host == "" && u.Path == "" && u.Opaque == "" &&
		u.RawQuery == "" && u.Fragment != ""
}

// IsWebURL reports whether u is an absolute address a browser opens.
func IsWebURL(u *url.URL) bool {
	return (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

/*
Slug is the anchor GitHub gives a heading: lower case, letters, digits,
spaces, hyphens and underscores kept and everything else dropped, spaces made
hyphens. The text is the heading as drawn, with its Markdown already gone.
Duplicates are numbered by Anchors, not here.
*/
func Slug(heading string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(heading)) {
		switch {
		case r == ' ':
			b.WriteByte('-')
		case r == '-' || r == '_' || unicode.IsLetter(r) || unicode.IsNumber(r):
			b.WriteRune(r)
		}
	}
	return b.String()
}

/*
Anchors maps each heading's slug to the block it is in. A slug used twice
gets -1, -2 and so on, as on GitHub, so the second "Usage" is #usage-1.

Only ATX headings (a line of one to six #, a space, the text) count, and
none inside a code fence: blocks are what Blocks made, so a fence is one
block and its first line is the fence.
*/
func Anchors(blocks []string) map[string]int {
	out := map[string]int{}
	seen := map[string]int{}
	for i, b := range blocks {
		if _, _, code := CodeBlock(b); code {
			continue
		}
		for _, line := range strings.Split(b, "\n") {
			text, ok := headingText(line)
			if !ok {
				continue
			}
			s := Slug(text)
			if n, dup := seen[s]; dup {
				seen[s] = n + 1
				s = s + "-" + strconv.Itoa(n+1)
			} else {
				seen[s] = 0
			}
			if _, taken := out[s]; !taken {
				out[s] = i
			}
		}
	}
	return out
}

// headingText is an ATX heading line's text as it is drawn: the Markdown
// parser removes emphasis, code ticks and link targets.
func headingText(line string) (string, bool) {
	t := strings.TrimLeft(line, " ")
	if len(line)-len(t) > 3 {
		return "", false
	}
	n := 0
	for n < len(t) && t[n] == '#' {
		n++
	}
	if n == 0 || n > 6 || (n < len(t) && t[n] != ' ' && t[n] != '\t') {
		return "", false
	}
	raw := strings.TrimSpace(strings.TrimRight(strings.TrimSpace(t[n:]), "#"))
	return plainText(widget.NewRichTextFromMarkdown(raw).Segments), true
}

// plainText is what a run of segments says, with no styling.
func plainText(segs []widget.RichTextSegment) string {
	var b strings.Builder
	for _, seg := range segs {
		switch s := seg.(type) {
		case *widget.TextSegment:
			b.WriteString(s.Text)
		case *widget.HyperlinkSegment:
			b.WriteString(s.Text)
		case *widget.ParagraphSegment:
			b.WriteString(plainText(s.Texts))
		case *widget.ListSegment:
			b.WriteString(plainText(s.Items))
		}
	}
	return b.String()
}

/*
linkText is a RichText whose links take a click over their own text.

**Fyne's link widgets overlap on consecutive lines.** A hyperlink in a
RichText is a Hyperlink widget the inner padding bigger than its text on
every side, pulled back over the text by the padding (unpadTextWidgetLayout),
so with the theme's 8-point inner padding a link one line tall is two lines
tall. In a list of links -- a README's contents -- each link's box covers the
lower half of the one above it, and Fyne gives a click to the topmost box: a
click on the lower half of "Platform notes" reached "Architecture", whose own
test (is the point over my text?) said no, and nothing happened. Only the top
of each entry worked. A headless tap on the widget itself never sees this; a
click on the window did (hayami's About page, spec 064).

So each link widget is drawn under a theme whose inner padding is zero, which
makes its box its text's box and leaves its text where it was. Fyne applies a
theme override to the objects that exist when it is applied, and a RichText
makes new link widgets on every refresh -- it keeps those inside a list or a
paragraph only until the next one -- so the override is applied by the
renderer, to the links each refresh made, before they are laid out again.
*/
type linkText struct {
	widget.RichText
}

// newLinkText is rt's content, drawn as a linkText.
func newLinkText(rt *widget.RichText) *linkText {
	t := &linkText{}
	t.Segments = rt.Segments
	t.Wrapping = rt.Wrapping
	t.Scroll = rt.Scroll
	t.ExtendBaseWidget(t)
	return t
}

// CreateRenderer implements fyne.Widget.
func (t *linkText) CreateRenderer() fyne.WidgetRenderer {
	return &linkRenderer{WidgetRenderer: t.RichText.CreateRenderer()}
}

// linkRenderer is the RichText's renderer, which unpads each link it makes.
type linkRenderer struct {
	fyne.WidgetRenderer
}

func (r *linkRenderer) Refresh() {
	r.WidgetRenderer.Refresh()
	for _, o := range r.Objects() {
		c, ok := o.(*fyne.Container)
		if !ok || len(c.Objects) != 1 {
			continue
		}
		hl, ok := c.Objects[0].(*widget.Hyperlink)
		if !ok || hl.Theme().Size(fynetheme.SizeNameInnerPadding) == 0 {
			continue
		}
		// The override is the point; the container it returns is not used.
		container.NewThemeOverride(hl, unpadded{})
		c.Refresh() // lays the link out again, under the padding it now has
	}
}

// unpadded is the running theme with no inner padding.
type unpadded struct{}

func (unpadded) current() fyne.Theme { return fyne.CurrentApp().Settings().Theme() }

func (u unpadded) Color(n fyne.ThemeColorName, v fyne.ThemeVariant) color.Color {
	return u.current().Color(n, v)
}

func (u unpadded) Font(s fyne.TextStyle) fyne.Resource { return u.current().Font(s) }

func (u unpadded) Icon(n fyne.ThemeIconName) fyne.Resource { return u.current().Icon(n) }

func (u unpadded) Size(n fyne.ThemeSizeName) float32 {
	if n == fynetheme.SizeNameInnerPadding {
		return 0
	}
	return u.current().Size(n)
}

// live reports whether rt still holds a link a reader can tap.
func live(segs []widget.RichTextSegment) bool {
	for _, seg := range segs {
		switch s := seg.(type) {
		case *widget.HyperlinkSegment:
			return true
		case *widget.ParagraphSegment:
			if live(s.Texts) {
				return true
			}
		case *widget.ListSegment:
			if live(s.Items) {
				return true
			}
		}
	}
	return false
}

// draw is a block's or a cell's RichText, as a linkText when it holds a link.
func draw(rt *widget.RichText) fyne.CanvasObject {
	if live(rt.Segments) {
		return newLinkText(rt)
	}
	return rt
}
