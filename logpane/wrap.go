package logpane

import "unicode/utf8"

/*
Wrapping a log line.

The pane draws each row as one line of monospace text in a widget.List, which
gives every row the same height -- that is what makes a log of a thousand lines
scroll like a terminal instead of costing a layout pass per line. A line longer
than the pane was therefore drawn off the right edge and simply lost: an
auto-restore line naming fifteen cheats ended at "tool_reach" with no way to see
the rest.

So the wrapping is done here, in the model's own terms, and a long line becomes
several rows. The font is monospace, so the column count is exact rather than a
measurement per row, and the rows stay uniform.

What the user copies is unaffected: Copy renders the model, not the rows.
*/

/*
Continued marks a row that is the rest of the line above it.

Indented, so the eye can see where a message starts without reading it, and with
a mark, so an indented continuation is not mistaken for a message that happens
to begin with spaces. The ellipsis rather than an arrow: a glyph outside the
bundled font draws from a fallback face and marks the run boundary as missing
(docs/fyne-quirks.md, 19), and the ellipsis is one the font has.
*/
const Continued = "  … "

/*
wrapLine breaks text to fit cols columns, at a space where there is a usable one
and mid-word where there is not.

Every row after the first carries Continued and is that much narrower for it, so
a continuation still ends inside the pane rather than one mark past its edge. A
cols of zero or less means the pane has no width yet, and the line is left alone.
*/
func wrapLine(text string, cols int) []string {
	if cols <= 0 || utf8.RuneCountInString(text) <= cols {
		return []string{text}
	}
	rest := []rune(text)
	width := cols
	out := []string{}

	for len(rest) > width {
		cut := breakAt(rest, width)
		out = append(out, mark(len(out), string(rest[:cut])))
		// The space a line was broken at is consumed with it: a row starting
		// with a space reads as an indent nobody wrote.
		for cut < len(rest) && rest[cut] == ' ' {
			cut++
		}
		rest = rest[cut:]
		width = cols - utf8.RuneCountInString(Continued)
		if width < minColumns/2 {
			width = cols // too narrow to indent into; better long than empty
		}
	}
	if len(rest) > 0 {
		out = append(out, mark(len(out), string(rest)))
	}
	return out
}

// mark prefixes every row but the first.
func mark(row int, text string) string {
	if row == 0 {
		return text
	}
	return Continued + text
}

/*
breakAt is where to cut a line of runes that is longer than cols.

The last space in the first cols runes, so words stay whole; cols itself when
there is none, because a path or a blob of JSON has no spaces and losing its
tail is worse than breaking it.

minWordCol keeps the wrap from degenerating: a break in the first few columns
would leave a nearly empty row and push the whole line down, which is worse to
read than one broken word.
*/
func breakAt(line []rune, cols int) int {
	minWordCol := cols / 4
	for i := cols; i > minWordCol; i-- {
		if line[i-1] == ' ' {
			return i - 1
		}
	}
	return cols
}

// wrapRows is every line of the model as visual rows, in order.
func wrapRows(m *Model, cols int) []Line {
	n := m.Len()
	out := make([]Line, 0, n)
	for i := range n {
		line := m.At(i)
		for _, part := range wrapLine(line.Text, cols) {
			out = append(out, Line{Level: line.Level, Text: part})
		}
	}
	return out
}

// columnsFor is how many monospace characters fit in a width. Deliberately
// short by a couple: wrapping one column early is invisible, and one column
// late clips a character off every long line.
func columnsFor(width, charWidth float32) int {
	if width <= 0 || charWidth <= 0 {
		return 0
	}
	cols := int(width/charWidth) - wrapSlack
	if cols < minColumns {
		return 0 // too narrow to be worth wrapping into
	}
	return cols
}

// The slack, and the width below which wrapping is not attempted.
const (
	wrapSlack  = 2
	minColumns = 16
)
