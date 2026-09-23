package markdown

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	fynetheme "fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

/*
A pipe table, drawn as a table.

The first version handed the cells to `container.NewGridWithColumns`, which
gives every cell the same size -- so every row was as tall as the tallest row
in the table, and every column as wide as the widest, whatever was in them. A
two-column table of a short label and a paragraph came out as two columns of
floating text with an inch of nothing between the rows, and read as neither a
list nor a table.

Three things make it a table instead: columns proportioned to what is in them,
rows each as tall as their own content, and a rule under the header with a
fainter one between rows so the eye has something to follow across.

Fyne's own table widget is not the answer here. It lives in a scroller, and a
scroller inside a document takes the wheel from the page (see CodePanel for
the same trap).
*/

// tableGap is the space between columns, and the inset a rule is drawn with.
const tableGap = 12

// tableRowPad is the space above and below a row's text. Enough that rows do
// not touch, small enough that a table stays a table rather than becoming a
// list of paragraphs.
const tableRowPad = 6

// headerRule and rowRule are how solid the lines under the header and between
// the rows are. The header's is the divider; the rows' is a hint.
const (
	headerRule float32 = 1
	rowRule    float32 = 1
)

/*
renderTable draws a pipe table. rows[0] is the header.

Ruled on every side a reader follows: under the header, between the rows,
under the last one, and down the column boundaries. A grid of lines is what
tells somebody the thing they are looking at is a table before they have read
a word of it, and the version without them read as two columns of floating
text.
*/
func renderTable(rows [][]string) fyne.CanvasObject {
	if len(rows) == 0 {
		return widget.NewLabel("")
	}
	weights := columnWeights(rows)

	/*
		A roof, because a table is closed.

		Without it the first row hangs off the rule under the header and the
		top of the table is open, which reads as the page having run out
		rather than as the table having started.
	*/
	out := []fyne.CanvasObject{rule(fynetheme.ColorNameSeparator, headerRule)}

	/*
		The first row is the header either way -- that is what the `|---|`
		under it makes it -- so it never becomes a body row. An empty one is
		simply not drawn.
	*/
	if header := rows[0]; titled(header) {
		out = append(out,
			tableRow(header, weights, true),
			rule(fynetheme.ColorNameSeparator, headerRule))
	}
	body := rows[1:]

	for _, row := range body {
		out = append(out, tableRow(row, weights, false),
			rule(fynetheme.ColorNameInputBorder, rowRule))
	}

	stacked := container.New(&stack{}, out...)

	/*
		The verticals: both edges and every boundary between them.

		Drawn over the rows rather than between them, so they run the height
		of the table without every row having to know about them -- and
		including the outer two, because a table ruled on the inside and open
		at the sides is a table somebody has to infer the shape of.
	*/
	lines := make([]fyne.CanvasObject, 0, len(weights)+1)
	for range len(weights) + 1 {
		lines = append(lines, vertical())
	}
	return container.New(&framed{weights: weights}, append([]fyne.CanvasObject{stacked}, lines...)...)
}

/*
titled reports whether a table's first row is a header or an empty one.

A pipe table always has a first row and a `|---|---|` under it -- that is what
makes it a table rather than text with pipes in it -- so a table written
without headers is written with *empty* ones:

	| | |
	|---|---|
	| Lighting | every device OpenRGB can see |

Both forms are ordinary Markdown and both should draw as what they are. An
empty header drawn as a header is a blank row above the table, which is what
the gap at the top of hotaru's was.
*/
func titled(header []string) bool {
	for _, cell := range header {
		if strings.TrimSpace(cell) != "" {
			return true
		}
	}
	return false
}

/*
tableRow is one row, laid out to the table's column weights.

The header is emphasised by **style**, not by wrapping the text in asterisks.
That is what the first version did, and a table whose header cells are empty
-- which is how a two-column table of label and description is often written
-- turned each of them into `****`, which Markdown renders as a thematic
break. Two empty header cells drew two short horizontal rules.

Editing somebody's Markdown to change how it looks is the fault; a cell that
already carried emphasis, or a pipe, or nothing at all, was going to produce
something nobody wrote.
*/
func tableRow(cells []string, weights []float32, header bool) fyne.CanvasObject {
	drawn := make([]fyne.CanvasObject, 0, len(cells))
	for _, cell := range cells {
		rt := drawable(widget.NewRichTextFromMarkdown(cell))
		rt.Wrapping = fyne.TextWrapWord
		if header {
			embolden(rt)
		}
		drawn = append(drawn, rt)
	}
	return container.New(&columns{weights: weights}, drawn...)
}

// embolden makes every piece of text in a cell bold, leaving what it is --
// code, a link, a heading somebody wrote in a cell -- alone.
func embolden(rt *widget.RichText) {
	for i, segment := range rt.Segments {
		text, ok := segment.(*widget.TextSegment)
		if !ok {
			continue
		}
		text.Style.TextStyle.Bold = true
		rt.Segments[i] = text
	}
}

// vertical is a line down a column boundary.
func vertical() fyne.CanvasObject {
	line := canvas.NewRectangle(fynetheme.Color(fynetheme.ColorNameInputBorder))
	line.SetMinSize(fyne.NewSize(rowRule, 0))
	return line
}

/*
framed puts the column rules over the table.

The rows are laid out first and fill the frame; each vertical is then drawn
the full height of it, in the gap between two columns. Over rather than
between, so a row does not have to know how many lines are in the table or
where they go.
*/
type framed struct{ weights []float32 }

func (f *framed) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) == 0 {
		return
	}
	body := objects[0]
	body.Move(fyne.NewPos(0, 0))
	body.Resize(size)

	at := f.verticals(size.Width)
	for i, line := range objects[1:] {
		if i >= len(at) {
			return
		}
		line.Move(fyne.NewPos(at[i], 0))
		line.Resize(fyne.NewSize(rowRule, size.Height))
	}
}

/*
verticals is where the column rules go: the left edge, each boundary between
two columns, and the right.

The boundaries sit in the middle of the gap between columns rather than
against either one, so a line is equally far from the text on both sides of
it.
*/
func (f *framed) verticals(w float32) []float32 {
	inner := &columns{weights: f.weights}
	out := make([]float32, 0, len(f.weights)+1)
	out = append(out, 0)

	x := float32(0)
	for i := range max(len(f.weights)-1, 0) {
		x += inner.width(i, w)
		out = append(out, x+tableGap/2-rowRule/2)
		x += tableGap
	}
	return append(out, w-rowRule)
}

func (f *framed) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) == 0 {
		return fyne.Size{}
	}
	return objects[0].MinSize()
}

// rule is a hairline across the table.
func rule(name fyne.ThemeColorName, height float32) fyne.CanvasObject {
	line := canvas.NewRectangle(fynetheme.Color(name))
	line.SetMinSize(fyne.NewSize(0, height))
	return line
}

/*
columnWeights is how wide each column should be, as fractions of the width.

From the longest cell in each column, because that is what a reader's eye
uses and what a browser's table layout approximates. Clamped: a column of
one-word labels beside a column of paragraphs would otherwise be drawn at
about a twentieth of the width and wrap every label, and a table of two
similar columns should come out even rather than tipping on one long cell.
*/
func columnWeights(rows [][]string) []float32 {
	cols := 0
	for _, row := range rows {
		cols = max(cols, len(row))
	}
	if cols == 0 {
		return nil
	}

	longest := make([]float32, cols)
	for _, row := range rows {
		for i, cell := range row {
			longest[i] = max(longest[i], float32(len([]rune(cell))))
		}
	}

	var total float32
	for i := range longest {
		longest[i] = min(max(longest[i], minColumnChars), maxColumnChars)
		total += longest[i]
	}
	if total == 0 {
		return even(cols)
	}

	out := make([]float32, cols)
	for i, length := range longest {
		out[i] = length / total
	}
	return out
}

/*
The clamps, in characters.

A column narrower than a few words wraps every cell into a column of single
words; one wider than this reads as a paragraph that happens to have a label
beside it, and stops being a column at all.
*/
const (
	minColumnChars = 8
	maxColumnChars = 90
)

func even(cols int) []float32 {
	out := make([]float32, cols)
	for i := range out {
		out[i] = 1 / float32(cols)
	}
	return out
}

/*
columns lays a row out in proportioned columns.

**Width first, then height.** A cell's height is a property of the cell *and*
the width it is given, so each is resized to its column before it is asked how
tall it is -- spec 021's finding, and a layout is where it bites hardest,
because `MinSize` is asked without a width at all. So the measurement is taken
during `Layout`, where the width is known, and kept for `MinSize` to answer
with.

The first version asked each cell its height before resizing it, and a
wrapping RichText answers that question with the height of one line. Every row
came out 47 pixels tall whether it held a word or a paragraph.
*/
type columns struct {
	weights []float32
	// at is the width the measurement was taken at, and tall is what it came
	// to. Zero means never laid out.
	at, tall float32
}

func (c *columns) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if size.Width != c.at {
		c.measure(objects, size.Width)
	}
	x := float32(0)
	for i, o := range objects {
		w := c.width(i, size.Width)
		o.Move(fyne.NewPos(x, tableRowPad))
		o.Resize(fyne.NewSize(w, o.MinSize().Height))
		x += w + tableGap
	}
}

// measure gives each cell its column's width and then asks how tall it is.
func (c *columns) measure(objects []fyne.CanvasObject, w float32) {
	var tall float32
	for i, o := range objects {
		o.Resize(fyne.NewSize(c.width(i, w), o.MinSize().Height))
		tall = max(tall, o.MinSize().Height)
	}
	c.at, c.tall = w, tall+2*tableRowPad
}

func (c *columns) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if c.at > 0 {
		return fyne.NewSize(0, c.tall)
	}
	// Never laid out, so there is no width to wrap against: one line each is
	// the only answer available, and Layout corrects it.
	var height float32
	for _, o := range objects {
		height = max(height, o.MinSize().Height)
	}
	return fyne.NewSize(0, height+2*tableRowPad)
}

// width is column i's share of w, after the gaps between columns are taken
// out. A row with no weights -- one longer than its header -- divides evenly.
func (c *columns) width(i int, w float32) float32 {
	gaps := float32(len(c.weights)-1) * tableGap
	if len(c.weights) == 0 || i >= len(c.weights) {
		return w
	}
	return (w - gaps) * c.weights[i]
}

/*
stack is a vertical layout whose children each keep their own height.

`container.NewVBox` would do, except that it gives a minimum width of the
widest child -- and a table's children are as wide as the table, so the
document's minimum width would become the table's natural width and the page
would scroll sideways. This asks for no width at all and lets the pane decide.
*/
type stack struct{}

func (s *stack) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	y := float32(0)
	for _, o := range objects {
		/*
			Given its width before it is asked its height, for the reason in
			`columns`: a row of wrapping cells cannot say how tall it is
			until it knows how wide it is, and this resize is what tells it.
		*/
		o.Resize(fyne.NewSize(size.Width, o.MinSize().Height))
		h := o.MinSize().Height
		o.Move(fyne.NewPos(0, y))
		o.Resize(fyne.NewSize(size.Width, h))
		y += h
	}
}

func (s *stack) MinSize(objects []fyne.CanvasObject) fyne.Size {
	var height float32
	for _, o := range objects {
		height += o.MinSize().Height
	}
	return fyne.NewSize(0, height)
}
