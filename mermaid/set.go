package mermaid

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"path"
	"strings"

	"fyne.io/fyne/v2"
)

// DefaultDir is the directory, beside the sources, that rendered PNGs live in.
const DefaultDir = "diagrams"

// Scale is the device scale the PNGs are rendered at. 2x reads well on
// high-density displays and is halved when drawn at 1x.
const Scale float32 = 2

// Hash keys a diagram by its text: SHA-256 of the source with surrounding
// whitespace trimmed and CRLF normalised, first 16 hex characters. Trimming
// and normalising mean an editor's trailing newline or a Windows checkout
// does not invalidate every image.
func Hash(src string) string {
	norm := strings.TrimSpace(strings.ReplaceAll(src, "\r\n", "\n"))
	sum := sha256.Sum256([]byte(norm))
	return hex.EncodeToString(sum[:])[:16]
}

// FileName is the PNG name for a source and variant.
func FileName(src string, dark bool) string {
	variant := "light"
	if dark {
		variant = "dark"
	}
	return Hash(src) + "-" + variant + ".png"
}

// Set is a set of pre-rendered diagrams in a filesystem, usually an embedded
// one.
type Set struct {
	fsys fs.FS
	dir  string
}

// NewSet reads diagrams from dir within fsys; an empty dir means DefaultDir.
func NewSet(fsys fs.FS, dir string) *Set {
	if dir == "" {
		dir = DefaultDir
	}
	return &Set{fsys: fsys, dir: dir}
}

// Lookup returns the rendered image for a diagram source in the wanted
// variant, falling back to the other variant when only one exists. The second
// result is false when neither is present: the source was edited, or generate
// was never run.
func (s *Set) Lookup(src string, dark bool) (fyne.Resource, bool) {
	if s == nil {
		return nil, false
	}
	for _, variant := range []bool{dark, !dark} {
		name := FileName(src, variant)
		b, err := fs.ReadFile(s.fsys, path.Join(s.dir, name))
		if err == nil {
			return fyne.NewStaticResource(name, b), true
		}
	}
	return nil, false
}

// Has reports whether at least one variant of the diagram is present.
func (s *Set) Has(src string) bool {
	_, ok := s.Lookup(src, true)
	return ok
}
