package markdown

import (
	"strings"

	"fyne.io/fyne/v2"
	fynetheme "fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

/*
grid is a table's column widths: one value every row and the frame share, so
a row's cells and the rules between them agree on where each column is.

The weights say how the width is shared; the floors say how narrow a column may
get. The weights alone were the first version, and they count characters: a
column of one-word labels beside two columns of paragraphs came to an eleventh
of 191 characters, which on hayami's About page was narrower than "Section" in
bold plus the cell's padding, and Fyne's word wrap broke it as "Sectio" / "n"
(#194). A character count cannot know the face, the size or the padding; the
floor is measured in all three.
*/
type grid struct {
	weights []float32
	// cells holds every cell's text by column, for measuring the floors.
	cells [][]*widget.RichText

	// floors is each column's narrowest width, in the text size it was
	// measured at.
	floors   []float32
	floorsAt float32
}

// add records a cell's text as being in column i.
func (g *grid) add(i int, rt *widget.RichText) {
	for len(g.cells) <= i {
		g.cells = append(g.cells, nil)
	}
	g.cells[i] = append(g.cells[i], rt)
}

/*
width is column i's width in a table w wide, after the gaps between columns
are taken out.

Every column first gets its floor. If the floors do not fit, there is no width
that keeps every word whole, and they are shared in proportion to themselves so
that what breaks breaks evenly. Otherwise a column whose weighted share is
under its floor is held at the floor and the rest is shared again among the
others by their weights, until none is under. A row with no weights -- one
longer than its header -- takes the whole width.
*/
func (g *grid) width(i int, w float32) float32 {
	cols := len(g.weights)
	if cols == 0 || i >= cols {
		return w
	}
	avail := w - float32(cols-1)*tableGap
	return g.widths(avail)[i]
}

func (g *grid) widths(avail float32) []float32 {
	cols := len(g.weights)
	floors := g.measure()

	var floorSum float32
	for _, f := range floors {
		floorSum += f
	}
	out := make([]float32, cols)
	if floorSum >= avail {
		for i, f := range floors {
			if floorSum > 0 {
				out[i] = avail * f / floorSum
			}
		}
		return out
	}

	held := make([]bool, cols)
	for range cols {
		left, weight := avail, float32(0)
		for i := range cols {
			if held[i] {
				left -= floors[i]
			} else {
				weight += g.weights[i]
			}
		}
		grew := false
		for i := range cols {
			if held[i] {
				out[i] = floors[i]
				continue
			}
			out[i] = left * g.weights[i] / weight
			if out[i] < floors[i] {
				held[i], grew = true, true
			}
		}
		if !grew {
			break
		}
	}
	return out
}

/*
measure is each column's floor: its longest word, in the style the word is
drawn in, with the cell's padding on both sides.

Taken again when the text size changes, because the floor is in pixels and a
larger text needs a wider column for the same word.
*/
func (g *grid) measure() []float32 {
	size := fynetheme.TextSize()
	if g.floors != nil && g.floorsAt == size {
		return g.floors
	}
	floors := make([]float32, len(g.weights))
	pad := 2 * fynetheme.Size(fynetheme.SizeNameInnerPadding)
	for i := range floors {
		if i >= len(g.cells) {
			continue
		}
		for _, rt := range g.cells[i] {
			floors[i] = max(floors[i], longestWord(rt.Segments)+pad)
		}
	}
	g.floors, g.floorsAt = floors, size
	return floors
}

// longestWord is the width of the widest word in some segments, in the face
// and size each is drawn in.
func longestWord(segments []widget.RichTextSegment) float32 {
	var widest float32
	for _, segment := range segments {
		switch s := segment.(type) {
		case *widget.TextSegment:
			widest = max(widest, widestIn(s.Text, s.Style))
		case *widget.HyperlinkSegment:
			widest = max(widest, widestIn(s.Text, widget.RichTextStyleInline))
		case *widget.ParagraphSegment:
			widest = max(widest, longestWord(s.Texts))
		case *widget.ListSegment:
			widest = max(widest, longestWord(s.Items))
		}
	}
	return widest
}

/*
widestIn is the widest word of text in a style.

Inline code is drawn on a chip with padding of its own on each side, which a
bare measurement of the letters would leave out.
*/
func widestIn(text string, style widget.RichTextStyle) float32 {
	size := fynetheme.TextSize()
	if style.SizeName != "" {
		size = fynetheme.Size(style.SizeName)
	}
	var chip float32
	if style.TextStyle.Monospace && style.Inline {
		chip = 2 * fynetheme.Padding()
	}
	var widest float32
	for _, word := range strings.Fields(text) {
		widest = max(widest, fyne.MeasureText(word, size, style.TextStyle).Width+chip)
	}
	return widest
}
