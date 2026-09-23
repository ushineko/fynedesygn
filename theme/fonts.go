package theme

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"golang.org/x/image/font"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

/*
Font discovery.

Fyne draws its own text and does not consult fontconfig, so a font chooser
means finding the files ourselves and handing the bytes to the theme. This
scans the platform's font directories once, groups faces into families, and
keeps only families that have a regular face: a family we cannot render at
normal weight is not one we can offer.

The scan reads font files and nothing else. Together with the cursor
configuration on Linux it is the only filesystem access this package performs.
*/

// DefaultFontName is the sentinel for "use the font Fyne ships with".
const DefaultFontName = "Fyne default"

// family holds the faces of one family as paths. Missing faces stay empty, and
// the theme falls back to the regular face for them: a family shipping only a
// regular weight renders bold text unbolded, which is better than rendering it
// in an unrelated font.
type family struct {
	name       string
	regular    string
	bold       string
	italic     string
	boldItalic string
}

var (
	fontsMu   sync.Mutex
	fontDirs  []string // nil means the platform's defaults
	fontList  []family
	scanned   bool
	fontCache = map[string]*Font{}
)

// RescanFonts discards the cached scan and, on the next request, scans the
// given directories instead of the platform's. Called with no arguments it
// restores the platform's directories. Programs call it after installing a
// font; tests call it with a temporary directory.
func RescanFonts(dirs ...string) {
	fontsMu.Lock()
	defer fontsMu.Unlock()
	fontDirs = dirs
	fontList = nil
	scanned = false
	fontCache = map[string]*Font{}
}

// families returns the installed families, discovered once. The caller holds
// fontsMu.
func families() []family {
	if !scanned {
		dirs := fontDirs
		if len(dirs) == 0 {
			dirs = platformFontDirs()
		}
		fontList = scanFonts(dirs)
		scanned = true
	}
	return fontList
}

func scanFonts(dirs []string) []family {
	byName := map[string]*family{}

	for _, dir := range dirs {
		_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil //nolint:nilerr // an unreadable font directory is not worth failing over
			}
			ext := strings.ToLower(filepath.Ext(path))
			if ext != ".ttf" && ext != ".otf" {
				return nil
			}
			name, style := splitFace(filepath.Base(path))
			if name == "" {
				return nil
			}
			f := byName[name]
			if f == nil {
				f = &family{name: name}
				byName[name] = f
			}
			switch style {
			case "regular":
				f.regular = path
			case "bold":
				f.bold = path
			case "italic", "oblique":
				f.italic = path
			case "bolditalic", "boldoblique":
				f.boldItalic = path
			}
			return nil
		})
	}

	out := make([]family, 0, len(byName))
	for _, f := range byName {
		if f.regular != "" {
			out = append(out, *f)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].name < out[j].name })
	return out
}

// splitFace derives a family name and a style from a font filename.
// "DejaVuSans-BoldOblique.ttf" becomes ("DejaVu Sans", "boldoblique");
// "arial.ttf" becomes ("arial", "regular").
func splitFace(base string) (string, string) {
	stem := strings.TrimSuffix(base, filepath.Ext(base))
	style := "regular"
	if i := strings.LastIndex(stem, "-"); i > 0 {
		style = strings.ToLower(stem[i+1:])
		stem = stem[:i]
	}
	// Variable fonts carry an axis list rather than a weight and cannot be
	// rendered at a fixed style here; skip them rather than offering a family
	// that draws wrong.
	if strings.Contains(strings.ToLower(stem), "[") || strings.Contains(style, "[") {
		return "", ""
	}
	return spaceCamel(stem), style
}

// spaceCamel turns "DejaVuSansMono" into "DejaVu Sans Mono" so the picker reads
// like a font menu rather than a directory listing.
func spaceCamel(s string) string {
	var b strings.Builder
	runes := []rune(s)
	for i, r := range runes {
		if i == 0 {
			b.WriteRune(r)
			continue
		}
		prevUpper := runes[i-1] >= 'A' && runes[i-1] <= 'Z'
		if r >= 'A' && r <= 'Z' && !prevUpper {
			b.WriteRune(' ')
		}
		b.WriteRune(r)
	}
	return b.String()
}

// FontNames lists the families offered, with the bundled default first.
func FontNames() []string {
	fontsMu.Lock()
	defer fontsMu.Unlock()
	names := []string{DefaultFontName}
	for _, f := range families() {
		names = append(names, f.name)
	}
	return names
}

// Font holds the loaded faces of a chosen family. A nil *Font means the
// bundled default, and every method tolerates a nil receiver.
type Font struct {
	regular    fyne.Resource
	bold       fyne.Resource
	italic     fyne.Resource
	boldItalic fyne.Resource
}

// LoadFont reads a family's faces. It returns nil for the default name and
// nil for anything it cannot read: an unreadable font falls back rather than
// failing, because a missing font is not a reason to refuse to draw a window.
func LoadFont(name string) *Font {
	if name == "" || name == DefaultFontName {
		return nil
	}
	fontsMu.Lock()
	defer fontsMu.Unlock()
	if f, ok := fontCache[name]; ok {
		return f
	}
	f := readFamily(name)
	fontCache[name] = f
	return f
}

/*
PreviewFont reads a family's faces without keeping them, for showing somebody
what a font looks like before they choose it.

LoadFont caches for the life of the process, which is right for the two or
three families a program actually draws in and wrong for browsing. A font is
its file: this machine carries 311 families, and reading every one costs
778 MB and 731 ms (2.5 MB and 2.4 ms each, measured by
TestLoadingEveryFamilyIsNotFree). A picker that previewed through LoadFont
would grow the process by a family's worth of memory for every name somebody
scrolled past, and never give any of it back.

So a preview reads the file each time and lets the result go. One font is
alive at a time, and the cost of looking through all of them is the cost of
looking at one.
*/
func PreviewFont(name string) *Font {
	if name == "" || name == DefaultFontName {
		return nil
	}
	fontsMu.Lock()
	defer fontsMu.Unlock()
	if f, ok := fontCache[name]; ok {
		// Already resident because the program draws in it: no reason to
		// read it again.
		return f
	}
	return readFamily(name)
}

// readFamily reads a family's faces from disk. Call with fontsMu held.
func readFamily(name string) *Font {
	var fam *family
	all := families()
	for i := range all {
		if all[i].name == name {
			fam = &all[i]
			break
		}
	}
	if fam == nil {
		return nil
	}

	read := func(path string) fyne.Resource {
		if path == "" {
			return nil
		}
		b, err := os.ReadFile(path) //nolint:gosec // a font path this package discovered itself
		if err != nil {
			return nil
		}
		return fyne.NewStaticResource(filepath.Base(path), b)
	}

	lf := &Font{
		regular:    read(fam.regular),
		bold:       read(fam.bold),
		italic:     read(fam.italic),
		boldItalic: read(fam.boldItalic),
	}
	if lf.regular == nil {
		// A family with no regular face is one nothing can be drawn in.
		return nil
	}
	return lf
}

// Face picks the resource for a text style, falling back to regular for any
// face the family does not ship. A nil Font returns nil, meaning Fyne's own.
func (f *Font) Face(s fyne.TextStyle) fyne.Resource {
	if f == nil {
		return nil
	}
	var r fyne.Resource
	switch {
	case s.Bold && s.Italic:
		r = f.boldItalic
	case s.Bold:
		r = f.bold
	case s.Italic:
		r = f.italic
	}
	if r == nil {
		r = f.regular
	}
	return r
}

/*
Missing counts the runes in text that this family has no glyph for.

Many of the families on a Linux machine are script fonts: Noto ships one per
writing system, and most of them carry punctuation and digits and not a single
Latin letter. Drawing a Latin sample in one of those shows a line of fallback
glyphs from some other font — which is a preview that lies twice, once by
looking like every other script font and once by looking like nothing
changed.

So a preview asks first. A nil family, or one whose face cannot be parsed,
answers with the whole length: it can draw none of it.
*/
func (f *Font) Missing(text string) int {
	if f == nil || f.regular == nil {
		return len([]rune(text))
	}
	face, err := sfnt.Parse(f.regular.Content())
	if err != nil {
		return len([]rune(text))
	}

	var buf sfnt.Buffer
	missing := 0
	for _, r := range text {
		index, err := face.GlyphIndex(&buf, r)
		if err != nil || index == 0 {
			missing++
		}
	}
	return missing
}

/*
Covered is the runes of text this family can draw, in order.

For showing somebody a script font's real face: it carries digits and
punctuation even when it carries no letters, and those in the family's own
shapes are worth more than a line of somebody else's glyphs.
*/
func (f *Font) Covered(text string) string {
	if f == nil || f.regular == nil {
		return ""
	}
	face, err := sfnt.Parse(f.regular.Content())
	if err != nil {
		return ""
	}

	var buf sfnt.Buffer
	out := make([]rune, 0, len(text))
	for _, r := range text {
		index, err := face.GlyphIndex(&buf, r)
		if err != nil || index == 0 {
			continue
		}
		out = append(out, r)
	}
	return string(out)
}

/*
IsMonospace reports whether every letter in this family is the same width.

For a font being chosen as a program's monospace face, where a proportional
family is not a matter of taste but a mistake: columns stop lining up, and the
places that asked for monospace did so because alignment was the point.

Measured rather than guessed from the name. "Liberation Sans" is obviously
proportional and "Meslo LGLDZ Nerd Font Propo" is not obviously anything, yet
the second is the one that will misalign a log pane.
*/
func (f *Font) IsMonospace() bool {
	if f == nil || f.regular == nil {
		return false
	}
	face, err := sfnt.Parse(f.regular.Content())
	if err != nil {
		return false
	}

	var buf sfnt.Buffer
	const ppem = 64
	width := fixed.Int26_6(0)
	for _, r := range "iMW.1" {
		index, err := face.GlyphIndex(&buf, r)
		if err != nil || index == 0 {
			continue
		}
		advance, err := face.GlyphAdvance(&buf, index, ppem, font.HintingNone)
		if err != nil {
			return false
		}
		if width == 0 {
			width = advance
			continue
		}
		if advance != width {
			return false
		}
	}
	return width != 0
}
