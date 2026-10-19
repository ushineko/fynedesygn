package theme

import (
	"sync"

	"fyne.io/fyne/v2"
	"golang.org/x/image/font/sfnt"
)

/*
Glyphs a face can actually draw (spec 073).

A program that wants an arrow in a label picks a character and hopes. The
hope is the bug: U+2190 LEFTWARDS ARROW is not in Adwaita Sans, so it drew
as a missing-glyph box on a Linux desk; U+25C2 BLACK LEFT-POINTING SMALL
TRIANGLE was chosen to fix that, tested on that one face, and drew as a box
on Windows. A glyph verified against one machine's font is not a glyph that
renders, and neither desk can speak for the other.

So ask the face. The theme already knows which file it draws with, and the
file says which characters it has.
*/

// BackGlyphs are ways to point back, best first: a small triangle, a single
// angle quote, then ASCII, which every face has.
var BackGlyphs = []string{"◂", "‹", "<"}

// ForwardGlyphs are ways to point on, best first, as BackGlyphs are. The
// last is ASCII and always drawable.
var ForwardGlyphs = []string{"▸", "›", ">"}

/*
FirstDrawable is the first candidate whose characters the theme's face can
all draw, or the last candidate when none of them can.

The last candidate is the caller's fallback and should be one every face
has, which in practice means ASCII. An empty list answers with "".

The answer is for the face the theme hands out now: a theme whose font has
changed answers again, so ask when the face changes rather than keeping it.
*/
func FirstDrawable(th fyne.Theme, style fyne.TextStyle, candidates ...string) string {
	if len(candidates) == 0 {
		return ""
	}
	if th == nil {
		return candidates[len(candidates)-1]
	}
	face := parseFace(th.Font(style))
	if face == nil {
		// Nothing can be said about a face that will not parse, so the
		// answer is the one that needs no face to be right.
		return candidates[len(candidates)-1]
	}
	var buf sfnt.Buffer
	for _, c := range candidates {
		if drawable(face, &buf, c) {
			return c
		}
	}
	return candidates[len(candidates)-1]
}

// drawable reports whether every character of s has a glyph in the face.
func drawable(face *sfnt.Font, buf *sfnt.Buffer, s string) bool {
	for _, r := range s {
		index, err := face.GlyphIndex(buf, r)
		if err != nil || index == 0 {
			return false
		}
	}
	return true
}

// faces are the parsed fonts, by resource name. Parsing one is reading and
// checking a file, and a label asks about its glyph every time it is drawn.
var faces sync.Map

func parseFace(res fyne.Resource) *sfnt.Font {
	if res == nil {
		return nil
	}
	if got, ok := faces.Load(res.Name()); ok {
		face, _ := got.(*sfnt.Font)
		return face
	}
	face, err := sfnt.Parse(res.Content())
	if err != nil {
		face = nil
	}
	faces.Store(res.Name(), face)
	return face
}
