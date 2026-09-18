package theme

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
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

	var fam *family
	all := families()
	for i := range all {
		if all[i].name == name {
			fam = &all[i]
			break
		}
	}
	if fam == nil {
		fontCache[name] = nil
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
		lf = nil
	}
	fontCache[name] = lf
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
