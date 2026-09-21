package markdown

import (
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

// renderTable draws a pipe table. rows[0] is the header.
func renderTable(rows [][]string) fyne.CanvasObject {
	if len(rows) == 0 {
		return widget.NewLabel("")
	}
	weights := columnWeights(rows)

	out := make([]fyne.CanvasObject, 0, len(rows)*2)
	for r, row := range rows {
		out = append(out, tableRow(row, weights, r == 0))
		switch {
		case r == 0:
			out = append(out, rule(fynetheme.ColorNameSeparator, headerRule))
		case r < len(rows)-1:
			out = append(out, rule(fynetheme.ColorNameInputBorder, rowRule))
		}
	}
	return container.New(&stack{}, out...)
}

// tableRow is one row, laid out to the table's column weights.
func tableRow(cells []string, weights []float32, header bool) fyne.CanvasObject {
	drawn := make([]fyne.CanvasObject, 0, len(cells))
	for _, cell := range cells {
		if header {
			cell = "**" + cell + "**"
		}
		rt := widget.NewRichTextFromMarkdown(cell)
		rt.Wrapping = fyne.TextWrapWord
		drawn = append(drawn, rt)
	}
	return container.New(&columns{weights: weights}, drawn...)
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
