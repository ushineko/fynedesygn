package glance

import (
	"image/color"
	"math"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ushineko/fynedesygn/widgets"
)

// HeatCellHeight is the height of a heat strip's cells: a line of text's, less
// a little, so a strip in a list of rows is the height of a row.
const HeatCellHeight float32 = 12

// HeatCellGap is the space between two cells.
const HeatCellGap float32 = 2

/*
HeatStrip is many proportions of one kind side by side, each a cell coloured by
how full it is: a processor's cores, each one's load (spec 083). A label at the
left, the cells sharing the rest of the width.

It is for when the reader asks which of many is busy, not how busy each is to
the percent: a meter per core would be thirty-two lines. The colour runs from
the empty track to the scheme's primary, so an idle cell is the track and a
full one is the bar a meter fills with.

A cell whose value is not a number (NaN) is drawn empty, as a gap in the
row's colour, not left out: a core that is offline keeps its place, and the
others do not move. The strip asks for no width, as a meter's bar does, and
its height is the same for any number of cells and any values: nothing it
shows reflows the panel.
*/
type HeatStrip struct {
	themed

	label *canvas.Text
	cells *heatCells
	box   *fyne.Container
}

// NewHeatStrip builds a strip with a label and no cells. labelWidth pins the
// label column, as NewMeter's does.
func NewHeatStrip(label string, labelWidth float32) *HeatStrip {
	h := &HeatStrip{label: canvas.NewText(label, nil), cells: newHeatCells()}
	var head fyne.CanvasObject = h.label
	if labelWidth > 0 {
		head = widgets.FixedWidth(h.label, labelWidth)
	}
	h.box = container.NewBorder(nil, nil, head, nil, h.cells)
	h.Restyle()
	return h
}

// Set draws the strip's cells, one per value, each a fraction from 0 to 1;
// NaN is a cell with nothing to show.
func (h *HeatStrip) Set(values []float64) {
	h.cells.values = append(h.cells.values[:0], values...)
	h.cells.Refresh()
}

// Values are the fractions the cells are drawn from.
func (h *HeatStrip) Values() []float64 { return append([]float64(nil), h.cells.values...) }

// SetLabel replaces the label.
func (h *HeatStrip) SetLabel(s string) {
	h.label.Text = s
	h.label.Refresh()
}

// Label is the strip's label.
func (h *HeatStrip) Label() string { return h.label.Text }

// SetTheme gives the strip a theme of its own. A panel calls it.
func (h *HeatStrip) SetTheme(th fyne.Theme) {
	h.themed.SetTheme(th)
	h.cells.SetTheme(th)
	h.Restyle()
}

// Restyle repaints the strip in the current theme.
func (h *HeatStrip) Restyle() {
	h.label.TextSize = h.textSize()
	h.label.Color = h.colour(theme.ColorNameForeground)
	h.label.Refresh()
	h.cells.Refresh()
	h.box.Refresh()
}

// Object is the strip's content.
func (h *HeatStrip) Object() fyne.CanvasObject { return h.box }

/*
SetCellLabels draws a word under each cell, centred on it: a day's initial
under a day (spec 085). A strip whose cells are days or hours is read as
such only when each says which it is, and a label beside the strip cannot.

From the first call with labels the strip is a line taller, and that height
does not change with what the labels say or how many there are. nil takes
the line away again.
*/
func (h *HeatStrip) SetCellLabels(labels []string) {
	h.cells.labels = append([]string(nil), labels...)
	h.cells.Refresh()
	h.box.Refresh()
}

/*
SetSelected marks cell i as the one chosen: an outline in the foreground
colour, and its label bold in the same colour (spec 085). -1, or an index
with no cell, marks none. Marking a cell changes no size.
*/
func (h *HeatStrip) SetSelected(i int) {
	h.cells.selected = i
	h.cells.Refresh()
}

// Selected is the marked cell, or -1.
func (h *HeatStrip) Selected() int {
	if h.cells.selected < 0 || h.cells.selected >= len(h.cells.values) {
		return -1
	}
	return h.cells.selected
}

/*
CellAt is the cell under a position on the window's canvas, or -1 off the
cells: on the label, on the labels' line, outside the strip (spec 085). The
gap after a cell is that cell's, so a click between two lands on one.
*/
func (h *HeatStrip) CellAt(at fyne.Position) int {
	n := len(h.cells.values)
	if n == 0 || !h.box.Visible() || !h.cells.Visible() {
		return -1
	}
	pos := fyne.CurrentApp().Driver().AbsolutePositionForObject(h.cells)
	size := h.cells.Size()
	if at.X < pos.X || at.X >= pos.X+size.Width || at.Y < pos.Y || at.Y >= pos.Y+HeatCellHeight {
		return -1
	}
	_, step := cellSpan(size.Width, n)
	return min(int((at.X-pos.X)/step), n-1)
}

// cellSpan is a cell's width and the distance from one cell's left edge to
// the next's, for n cells across width.
func cellSpan(width float32, n int) (w, step float32) {
	w = max((width-HeatCellGap*float32(n-1))/float32(n), 0)
	return w, w + HeatCellGap
}

// CellColours are the colours the cells are drawn in, for a test that reads
// them; NaN's cell is nil.
func (h *HeatStrip) CellColours() []color.Color {
	out := make([]color.Color, len(h.cells.values))
	for i, v := range h.cells.values {
		if !math.IsNaN(v) {
			out[i] = h.cells.heat(v)
		}
	}
	return out
}

type heatCells struct {
	widget.BaseWidget
	themed

	values   []float64
	labels   []string
	selected int
}

func newHeatCells() *heatCells {
	c := &heatCells{selected: -1}
	c.ExtendBaseWidget(c)
	return c
}

// MinSize reserves the cells' height, and the labels' line when there are
// labels, and no width.
func (c *heatCells) MinSize() fyne.Size {
	if c.labels == nil {
		return fyne.NewSize(0, HeatCellHeight)
	}
	return fyne.NewSize(0, HeatCellHeight+heatLabelGap+c.labelLine())
}

// heatLabelGap is the space between the cells and their labels.
const heatLabelGap float32 = 1

// labelSize is the labels' text size: smaller than a row's, as a caption is.
func (c *heatCells) labelSize() float32 { return c.textSize() * 0.85 }

// labelLine is the height of the labels' line, measured with a bold capital
// so that a marked label is no taller than the others.
func (c *heatCells) labelLine() float32 {
	return fyne.MeasureText("M", c.labelSize(), fyne.TextStyle{Bold: true}).Height
}

// heat is a fraction's colour: the track's, mixed toward the primary.
func (c *heatCells) heat(v float64) color.Color {
	v = min(max(v, 0), 1)
	from := color.NRGBAModel.Convert(c.colour(theme.ColorNameDisabledButton)).(color.NRGBA)
	to := color.NRGBAModel.Convert(c.colour(theme.ColorNamePrimary)).(color.NRGBA)
	mix := func(a, b uint8) uint8 { return uint8(math.Round(float64(a) + (float64(b)-float64(a))*v)) }
	return color.NRGBA{R: mix(from.R, to.R), G: mix(from.G, to.G), B: mix(from.B, to.B), A: mix(from.A, to.A)}
}

func (c *heatCells) CreateRenderer() fyne.WidgetRenderer { return &heatRenderer{cells: c} }

type heatRenderer struct {
	cells *heatCells
	rects []*canvas.Rectangle
	words []*canvas.Text
	size  fyne.Size
}

func (r *heatRenderer) Layout(size fyne.Size) {
	r.size = size
	n := len(r.rects)
	if n == 0 {
		return
	}
	w, step := cellSpan(size.Width, n)
	// With labels the cells sit at the top and the labels under them;
	// without, the cells are centred in whatever height they were given.
	y := (size.Height - HeatCellHeight) / 2
	if r.cells.labels != nil {
		y = 0
	}
	for i, rect := range r.rects {
		rect.Move(fyne.NewPos(float32(i)*step, y))
		rect.Resize(fyne.NewSize(w, HeatCellHeight))
	}
	for i, t := range r.words {
		ts := t.MinSize()
		t.Move(fyne.NewPos(float32(i)*step+(w-ts.Width)/2, HeatCellHeight+heatLabelGap))
		t.Resize(ts)
	}
}

func (r *heatRenderer) MinSize() fyne.Size { return r.cells.MinSize() }

func (r *heatRenderer) Refresh() {
	for len(r.rects) < len(r.cells.values) {
		rect := canvas.NewRectangle(color.Transparent)
		rect.CornerRadius = 2
		r.rects = append(r.rects, rect)
	}
	r.rects = r.rects[:len(r.cells.values)]
	for i, v := range r.cells.values {
		if math.IsNaN(v) {
			r.rects[i].FillColor = color.Transparent
			r.rects[i].StrokeColor = r.cells.colour(theme.ColorNameDisabledButton)
			r.rects[i].StrokeWidth = 1
		} else {
			r.rects[i].FillColor = r.cells.heat(v)
			r.rects[i].StrokeWidth = 0
		}
		if i == r.cells.selected {
			r.rects[i].StrokeColor = r.cells.colour(theme.ColorNameForeground)
			r.rects[i].StrokeWidth = 2
		}
		r.rects[i].Refresh()
	}
	// A label per cell: labels past the last cell have nowhere to go.
	n := min(len(r.cells.labels), len(r.cells.values))
	for len(r.words) < n {
		r.words = append(r.words, canvas.NewText("", nil))
	}
	r.words = r.words[:n]
	for i, t := range r.words {
		t.Text = r.cells.labels[i]
		t.TextSize = r.cells.labelSize()
		t.Color = r.cells.colour(theme.ColorNamePlaceHolder)
		t.TextStyle = fyne.TextStyle{}
		if i == r.cells.selected {
			t.Color = r.cells.colour(theme.ColorNameForeground)
			t.TextStyle.Bold = true
		}
		t.Refresh()
	}
	r.Layout(r.size)
}

func (r *heatRenderer) Objects() []fyne.CanvasObject {
	out := make([]fyne.CanvasObject, 0, len(r.rects)+len(r.words))
	for _, rect := range r.rects {
		out = append(out, rect)
	}
	for _, t := range r.words {
		out = append(out, t)
	}
	return out
}

func (r *heatRenderer) Destroy() {}
