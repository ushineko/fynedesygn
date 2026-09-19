/*
Package table holds the read-only detail table every section reaches for
where a terminal would print a fixed-width listing: candidates, report rows,
installed tools, archives.

Detail is deliberately not a widget. Fyne's table asks for its contents
through callbacks, and every section that wanted one was writing the same
three closures over a different slice of strings; Detail holds the strings and
hands back the widget.Table. Rows carry a Status so the verdict is coloured
from the active theme rather than from a hard-coded green.

Sorting is not built in. Tables where order is a question handle it
themselves with SortHeader, because in some of them re-ordering the rows would
misrepresent what the program will do.

Ported from nmsbonker's detailTable with clockwork-orange's thumbnail column
(same author, MIT).
*/
package table

import (
	"fmt"
	"unicode/utf8"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	fd "github.com/ushineko/fynedesygn"
	"github.com/ushineko/fynedesygn/widgets"
)

// ThumbCellSize is the edge, in pixels, of a thumbnail cell.
const ThumbCellSize float32 = 64

/*
The metric a column's width and a cell's own truncation share: text is assumed
perRune wide per rune, plus cellPadding for the cell's own inset.

Approximate on purpose. Fyne cannot measure text without shaping it, and a table
that shaped every cell on every layout is what made a window drag stall.
*/
const (
	perRune     float32 = 7.0
	cellPadding float32 = 24.0
	ellipsis            = "…"
)

// Detail is a listing of string rows, one Status per row, that renders as a
// widget.Table. Build it with Header and Row, then call Widget.
type Detail struct {
	head   []string
	rows   [][]string
	stat   []fd.Status
	widths []float32
	// thumbCol, when >= 0, is a column drawn from thumbs (one encoded image
	// per row) instead of from the row's text. New leaves it at -1, none.
	thumbCol int
	thumbs   [][]byte
	// images caches the decoded thumbnails by row, so scrolling does not
	// decode the same image again for every recycled cell.
	images map[int]fyne.Resource
}

// New returns an empty table with no thumbnail column.
func New() *Detail { return &Detail{thumbCol: -1} }

// Header sets the column titles. The number of titles is the number of
// columns the table draws.
func (t *Detail) Header(cols ...string) { t.head = cols }

// Row adds a line. st colours every cell in it: these tables have one verdict
// per row rather than per cell, and colouring only the verdict column made
// the eye hunt for which row the colour belonged to.
func (t *Detail) Row(st fd.Status, cols ...string) {
	t.rows = append(t.rows, cols)
	t.stat = append(t.stat, st)
}

// SetWidths pins the column widths. Without it the measured widths are used,
// which is right for a listing of unknown content and wrong for one whose
// first column is a fixed vocabulary.
func (t *Detail) SetWidths(w ...float32) { t.widths = w }

// Len is the number of rows.
func (t *Detail) Len() int { return len(t.rows) }

// SetThumbnails marks col as the thumbnail column and supplies one encoded
// image (JPEG or PNG) per row, indexed like the rows. A row with no image
// draws an empty cell. Passing no images turns the column off again.
func (t *Detail) SetThumbnails(col int, images [][]byte) {
	t.thumbCol = col
	t.thumbs = images
	t.images = nil
}

// hasThumbs reports whether a thumbnail column is in use.
func (t *Detail) hasThumbs() bool { return len(t.thumbs) > 0 && t.thumbCol >= 0 }

// thumb is the row's thumbnail as a resource, or nil. The resource name has
// no extension: Fyne only consults the name to recognise SVG, and sniffs
// raster formats from the bytes.
func (t *Detail) thumb(row int) fyne.Resource {
	if row < 0 || row >= len(t.thumbs) || len(t.thumbs[row]) == 0 {
		return nil
	}
	if t.images == nil {
		t.images = map[int]fyne.Resource{}
	}
	if r, ok := t.images[row]; ok {
		return r
	}
	r := fyne.NewStaticResource(fmt.Sprintf("thumb-%d", row), t.thumbs[row])
	t.images[row] = r
	return r
}

// Measure derives a column width from the widest cell, clamped to 70..460.
// Fyne cannot ask a label how wide its text will be without a canvas, so this
// counts runes and multiplies: approximate on purpose, and the truncation
// ellipsis covers the cases where it is short.
func (t *Detail) Measure(col int) float32 {
	const (
		minW = 70.0
		maxW = 460.0
	)
	widest := 0
	if col >= 0 && col < len(t.head) {
		widest = utf8.RuneCountInString(t.head[col])
	}
	for _, r := range t.rows {
		if col >= 0 && col < len(r) {
			if n := utf8.RuneCountInString(r[col]); n > widest {
				widest = n
			}
		}
	}
	w := float32(widest)*perRune + cellPadding
	return min(max(w, minW), maxW)
}

// Cell is the text and importance a body cell shows. Out of range, in either
// dimension, it is empty text at MediumImportance, so a recycled cell that
// scrolls past the data is painted neutral rather than in a stale colour.
func (t *Detail) Cell(row, col int) (text string, imp widget.Importance) {
	if row < 0 || row >= len(t.rows) {
		return "", widget.MediumImportance
	}
	r := t.rows[row]
	if col < 0 || col >= len(r) {
		return "", widget.MediumImportance
	}
	return r[col], widgets.ImportanceFor(t.stat[row])
}

// HeaderText is the title of column col. Row headers are off in these tables,
// but UpdateHeader is still called with Col == -1 for the corner cell, so a
// negative or oversized column returns "" rather than indexing the header
// slice out of range.
func (t *Detail) HeaderText(col int) string {
	if col < 0 || col >= len(t.head) {
		return ""
	}
	return t.head[col]
}

// Widget renders the table. Column widths come from SetWidths where given
// and from Measure otherwise.
func (t *Detail) Widget() *widget.Table {
	cols := len(t.head)
	// Measured once here rather than per cell: the cells need them to cut their
	// own text, and Measure walks every row.
	colWidths := make([]float32, cols)
	for i := range cols {
		if i < len(t.widths) {
			colWidths[i] = t.widths[i]
			continue
		}
		colWidths[i] = t.Measure(i)
	}

	table := widget.NewTable(
		func() (int, int) { return len(t.rows), cols },
		func() fyne.CanvasObject {
			l := widget.NewLabel("")
			/*
				Wrapping and truncation both off is the one path in Fyne's rich
				text that returns without measuring anything (lineBounds, the
				first branch). Everything else shapes the cell's text through
				harfbuzz -- and Fyne resizes every visible cell on every layout,
				so during a window drag that is the whole table, every frame:
				profiled at 42% of the process's CPU, with the GC taking another
				36% clearing up after it.

				The ellipsis is still wanted, so fit() does the cut here against
				the same rune metric the column widths come from.
			*/
			l.Wrapping = fyne.TextWrapOff
			l.Truncation = fyne.TextTruncateOff
			if !t.hasThumbs() {
				return l
			}
			img := canvas.NewImageFromResource(nil)
			img.FillMode = canvas.ImageFillContain
			img.SetMinSize(fyne.NewSize(ThumbCellSize, ThumbCellSize))
			img.Hide()
			return container.NewStack(img, l)
		},
		func(id widget.TableCellID, o fyne.CanvasObject) {
			l, img := cellParts(o)
			if img != nil {
				if id.Col == t.thumbCol {
					img.Resource = t.thumb(id.Row)
					img.Show()
					img.Refresh()
					l.Hide()
					return
				}
				img.Hide()
				l.Show()
			}
			text, imp := t.Cell(id.Row, id.Col)
			text = fit(text, colWidths[id.Col])
			/*
				Nothing to do when the cell already says this.

				Fyne calls UpdateCell for every visible cell on every layout,
				and Label.SetText refreshes unconditionally -- which re-shapes
				the text through harfbuzz. During a window drag that is every
				cell on screen, every frame, for text that has not changed:
				profiled on a seven-column table it was 35% of the process's
				CPU in updateRowBounds, and another 41% in the GC clearing up
				after it. Dragging the window stalled for seconds.

				A recycled cell that scrolls onto a different row does change,
				and still refreshes.
			*/
			if l.Text == text && l.Importance == imp {
				return
			}
			// Importance before SetText: SetText is what refreshes the label,
			// and the refresh is where importance becomes a colour. The other
			// way round, a scrolled table paints each recycled cell in the
			// colour of the row it last held.
			l.Importance = imp
			l.SetText(text)
		},
	)
	table.ShowHeaderRow = true
	if t.hasThumbs() {
		table.SetRowHeight(-1, ThumbCellSize+8) // the default for every row
		for i := range t.rows {
			table.SetRowHeight(i, ThumbCellSize+8)
		}
	}
	table.CreateHeader = func() fyne.CanvasObject {
		l := widget.NewLabel("")
		l.TextStyle = fyne.TextStyle{Bold: true}
		// The header is refreshed with the body on every layout, so it takes
		// the same no-measure path and cuts its own text. See the cell
		// template above.
		l.Wrapping = fyne.TextWrapOff
		l.Truncation = fyne.TextTruncateOff
		return l
	}
	table.UpdateHeader = func(id widget.TableCellID, o fyne.CanvasObject) {
		l := o.(*widget.Label)
		// Row headers are off in these tables, but UpdateHeader is still called
		// with Col == -1 for the corner cell. HeaderText guards it rather than
		// indexing the header slice with a negative number.
		title := t.HeaderText(id.Col)
		if id.Col >= 0 && id.Col < len(colWidths) {
			title = fit(title, colWidths[id.Col])
		}
		if l.Text == title {
			return
		}
		l.SetText(title)
	}
	for i := range cols {
		table.SetColumnWidth(i, colWidths[i])
	}
	return table
}

/*
fit cuts text to what a column of this width can show, with an ellipsis.

The same approximation as Measure -- a rune is perRune wide -- because the
alternative is measuring the text, which is the cost this exists to avoid. A
proportional font makes the cut a rune early or late on occasion; the ellipsis
is there either way, so nobody reads a word that was silently halved.
*/
func fit(text string, width float32) string {
	if width <= 0 || text == "" {
		return text
	}
	room := int((width - cellPadding) / perRune)
	if room < 1 {
		room = 1
	}
	if utf8.RuneCountInString(text) <= room {
		return text
	}
	runes := []rune(text)
	if room == 1 {
		return ellipsis
	}
	return string(runes[:room-1]) + ellipsis
}

// cellParts splits a template cell into its label and, when the table has a
// thumbnail column, its image.
func cellParts(o fyne.CanvasObject) (*widget.Label, *canvas.Image) {
	if l, ok := o.(*widget.Label); ok {
		return l, nil
	}
	stack := o.(*fyne.Container)
	return stack.Objects[1].(*widget.Label), stack.Objects[0].(*canvas.Image)
}
