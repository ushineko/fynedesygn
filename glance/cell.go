package glance

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"

	fd "github.com/ushineko/fynedesygn"
)

// CellValueScale is how much larger a cell's reading is than its name.
//
// It is the whole of what distinguishes a cell from a row, so it is a number
// worth being able to see rather than measure. The archetype draws a
// peripheral's percentage at roughly half again the size of the name above it,
// which is large enough to be the thing the eye lands on and small enough that
// four cells still fit a panel 260 px wide.
//
// A factor and not a pixel count, for the reason CardPadH gives: a size set
// for one face is proportionally wrong at another.
const CellValueScale float32 = 1.5

// CellPad is the space between the three pieces of a cell, in the
// application's face. A cell with a face of its own measures its own.
func CellPad() float32 { return theme.TextSize() * 0.2 }

// CellGridGap is the space between two cells in a grid, across and down, in
// the application's face. A grid with a face of its own measures its own.
func CellGridGap() float32 { return theme.TextSize() * 1.2 }

/*
CellNameBudget is the most width a cell claims on account of its name.

A cell is sized by its reading, because that is the reading and because a
device name is as transient as the device. But sizing by the reading *alone*
puts a cell at the width of "67 %" and packs four of them into a panel, and
then four device names draw into four columns forty pixels wide and run into
each other. That is what the first capture of this showed: "G502 X
PLUSKeychron K4 HE AirPods Pro" as one unbroken line.

So a name is allowed to ask for width, up to this much, and is elided beyond
it. Eight times the text size is around ten characters, which is enough for
"Discharging" and for most device names to be recognisable, and small enough
that two cells still fit a panel 260 px wide -- which is the archetype's own
two-up.
*/
func CellNameBudget() float32 { return theme.TextSize() * 8 }

// pad, gridGap and nameBudget are the three above measured in whatever face
// the object has been given, which is what every layout in this file uses. The
// exported ones stay for a consumer that is doing arithmetic of its own.
func (t *themed) pad() float32        { return t.textSize() * 0.2 }
func (t *themed) gridGap() float32    { return t.textSize() * 1.2 }
func (t *themed) nameBudget() float32 { return t.textSize() * 8 }

/*
Cell is one reading as a block: a name, a reading and a state, stacked and
centred, with the reading drawn larger than the other two.

**A reading with a name worth less than its number is a cell; everything else
is a Row.** A rate, a temperature and a fan speed are two facts of equal
weight, and a row -- label left, value right, slack between -- reads down a
column properly. A battery is not: the percentage is the reading and the device
name is only which battery it belongs to. Drawn as a row the figure is a small
number at the right margin in the same weight as the name, so the eye reads the
name and then hunts for the figure; drawn as a cell the figure is what it lands
on.

The state under the reading is the third of the block and is not optional
furniture. A panel that said what the battery was doing only when it was
charging left the ordinary case looking exactly like a device nobody had heard
from, and a blank third of a cell reads as one that has not finished loading.

Like Row, this is a live handle and not a build function. A card is redrawn
from a poll several times a minute and rebuilding the tree at that rate is what
"build the tree once; update it in place" exists to stop.

There is no icon. The archetype draws a battery glyph beside the percentage and
it is deliberately not carried here: an icon per device type means this module
knowing what a mouse is. A consumer that wants one puts it in the name.
*/
type Cell struct {
	themed

	name  *canvas.Text
	value *canvas.Text
	note  *canvas.Text
	box   *fyne.Container

	reading Reading
	shown   bool

	// fullName and fullNote are what was set, kept because the canvas.Text
	// holds what is *drawn* -- which is elided when the cell is narrower than
	// the words. Re-eliding on every layout from an already-elided string
	// would eat a character each time.
	fullName string
	fullNote string
}

// NewCell builds a cell with a name and no reading yet.
//
// blank is what the reading reads as until Set is called, at the width blank
// implies, so the cell does not change size when its first reading lands --
// the same contract NewRow has and for the same reason.
func NewCell(name, blank string) *Cell {
	c := &Cell{
		name:  canvas.NewText(name, nil),
		value: canvas.NewText(blank, nil),
		note:  canvas.NewText("", nil),
		shown: true,
	}
	for _, t := range []*canvas.Text{c.name, c.value, c.note} {
		t.Alignment = fyne.TextAlignCenter
	}

	c.reading = Measured(blank)
	c.fullName = name
	c.restyleText()
	c.box = container.New(&cellLayout{cell: c}, c.name, c.value, c.note)
	return c
}

// Set replaces the cell's reading. It repaints; it does not re-lay-out,
// because the text it is given is the same width as the text it had.
func (c *Cell) Set(rd Reading) {
	c.reading = rd
	c.value.Text = rd.Text
	c.value.Color = c.readingColour()
	c.value.Refresh()
}

// SetName replaces the name over the reading, for a cell whose subject can
// change -- a slot showing whichever device is connected.
func (c *Cell) SetName(s string) {
	c.fullName = s
	c.redraw(c.name, s)
}

// SetNote replaces the state under the reading.
func (c *Cell) SetNote(s string) {
	c.fullNote = s
	c.redraw(c.note, s)
}

// SetTheme gives the cell a face of its own and repaints in it. A canvas.Text
// holds a literal size and colour rather than asking the theme when it draws,
// so being told is not enough.
func (c *Cell) SetTheme(th fyne.Theme) {
	c.themed.SetTheme(th)
	c.Restyle()
}

// Reading is the value the cell currently holds, which is what a test asks for
// rather than walking the object tree.
func (c *Cell) Reading() Reading { return c.reading }

/*
redraw puts a string on one of the cell's texts, elided to the width the cell
currently has.

Eliding here and not only in Layout, because a name is set on a poll and a poll
does not re-lay-out: the container measures when it is resized, not when a
canvas.Text inside it changes. A cell whose device was renamed therefore drew
the new name in full, straight over its neighbour -- which is what the panel
showed the first time a real device went into one.

Before the first layout the cell has no width and the string is kept whole;
Layout elides it when the size arrives.
*/
func (c *Cell) redraw(t *canvas.Text, s string) {
	if w := c.box.Size().Width; w > 0 {
		elide(t, s, w)
		return
	}
	t.Text = s
	t.Refresh()
}

// Note is the state under the reading, as it was set rather than as it is
// drawn: a cell narrower than the words draws them elided.
func (c *Cell) Note() string { return c.fullNote }

// Name is the name over the reading, as it was set rather than as it is drawn.
func (c *Cell) Name() string { return c.fullName }

// colour is the status colour, or the disabled colour when the value is stale.
// Dimming wins over the verdict, as it does for a Row: a warning that is no
// longer being refreshed should not keep shouting.
func (c *Cell) readingColour() color.Color {
	if c.reading.Stale {
		return c.colour(theme.ColorNameDisabled)
	}
	if c.reading.Colour != nil {
		return c.reading.Colour
	}
	if c.reading.Status == fd.StatusInfo {
		return c.colour(theme.ColorNameForeground)
	}
	return c.statusColour(c.reading.Status)
}

// Restyle repaints the cell in the current theme, after a scheme or text size
// change. A canvas.Text holds a literal colour and size rather than asking the
// theme at paint time, so nothing else brings it up to date.
func (c *Cell) Restyle() {
	c.restyleText()
	c.name.Refresh()
	c.value.Refresh()
	c.note.Refresh()
	c.refit(c.name, c.value, c.note)
	c.box.Refresh()
}

// restyleText applies the current theme's sizes and colours without
// refreshing, for the constructor, which has nothing to refresh yet.
func (c *Cell) restyleText() {
	c.name.TextSize = c.textSize()
	c.value.TextSize = c.textSize() * CellValueScale
	c.note.TextSize = c.textSize()

	/*
		Bold, because the reading is the whole point of a cell.
		CellValueScale already says so in size; weight says it from further
		away, which is the distance a battery percentage is actually read
		from. The name above and the state below stay regular, so the three
		pieces of a cell read in the order they matter.

		Monospace with it, so the digits still hold their columns when a level
		goes from 9 to 10. Through bold rather than set outright, because a
		mono family with no bold is a panic in the painter and not a lighter
		weight -- see quirk 35. It is here and not in the constructor because
		a cell can be given a different face later, and the new family may
		answer differently.
	*/
	c.value.TextStyle = c.bold(fyne.TextStyle{Monospace: true})

	c.name.Color = c.colour(theme.ColorNameForeground)
	c.value.Color = c.readingColour()
	c.note.Color = c.colour(theme.ColorNameDisabled)
}

// SetShown draws or hides the cell. A device that has gone away hides its own
// cell and leaves the grid standing for the ones that are still there.
func (c *Cell) SetShown(shown bool) {
	if c.shown == shown {
		return
	}
	c.shown = shown
	if shown {
		c.box.Show()
	} else {
		c.box.Hide()
	}
}

// Shown reports whether the cell is currently drawn.
func (c *Cell) Shown() bool { return c.shown }

// Object is the cell's content, for a caller assembling a card by hand.
func (c *Cell) Object() fyne.CanvasObject { return c.box }

/*
cellLayout stacks a cell's three pieces, centred, and decides its width.

The reading sets the width and the name is elided into it, up to
CellNameBudget. A device name is as transient as the device -- a mouse waking
up, a headset changing what it calls itself -- so a cell that sized to its name
would reflow the panel when one did; but a cell sized to "67 %" alone is forty
pixels wide and four device names drawn into four such columns run into each
other, which is what the first capture of this showed. The budget is the
compromise and it is a number rather than a judgement.

Eliding happens in Layout, because it depends on the width the cell was
actually given, which is the grid's to decide and not the cell's.
*/
type cellLayout struct{ cell *Cell }

// padding is the gap between a cell's three pieces, in the cell's own face.
func (l *cellLayout) padding() float32 {
	if l.cell == nil {
		return CellPad()
	}
	return l.cell.pad()
}

// MinSize is the width the reading needs, widened for the name and the state
// up to their budget, and the three pieces' heights.
func (l *cellLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) < 3 {
		return fyne.Size{}
	}
	value := objects[1].MinSize()

	width := value.Width
	if c := l.cell; c != nil {
		budget := c.nameBudget()
		width = max(width, min(textWidth(c.name, c.fullName), budget))
		width = max(width, min(textWidth(c.note, c.fullNote), budget))
	}

	height := objects[0].MinSize().Height + value.Height + objects[2].MinSize().Height
	return fyne.NewSize(width, height+2*l.padding())
}

// Layout centres each piece on its own line, eliding the two that may be
// longer than the cell is wide.
func (l *cellLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) < 3 {
		return
	}
	// A cell that has not been given a width yet keeps its words. Eliding
	// against nothing blanks them, and a card assembled but not yet laid out
	// is an ordinary state -- it is what a test sees, and what the first
	// frame is built from.
	if c := l.cell; c != nil && size.Width > 0 {
		elide(c.name, c.fullName, size.Width)
		elide(c.note, c.fullNote, size.Width)
	}

	y := float32(0)
	for _, o := range objects {
		h := o.MinSize().Height
		o.Move(fyne.NewPos(0, y))
		o.Resize(fyne.NewSize(size.Width, h))
		y += h + l.padding()
	}
}

// Ellipsis is what a name too long for its cell ends with.
//
// The character and not three dots: three dots is three characters of a width
// that is already short, and every desktop's own file manager uses this one.
const Ellipsis = "…"

// elide sets a text to as much of s as fits a width, ending with an ellipsis
// when it had to cut.
//
// It measures rather than counting characters. A proportional face makes "WWW"
// three times the width of "iii", and a device name is exactly the kind of
// string that is all capitals or all lower case.
func elide(t *canvas.Text, s string, width float32) {
	if textWidth(t, s) <= width {
		if t.Text != s {
			t.Text = s
			t.Refresh()
		}
		return
	}

	runes := []rune(s)
	for len(runes) > 0 {
		runes = runes[:len(runes)-1]
		candidate := string(runes) + Ellipsis
		if textWidth(t, candidate) <= width {
			if t.Text != candidate {
				t.Text = candidate
				t.Refresh()
			}
			return
		}
	}

	// Narrower than one character and an ellipsis. The ellipsis alone still
	// says "there is more here", which is more than a blank does.
	if t.Text != Ellipsis {
		t.Text = Ellipsis
		t.Refresh()
	}
}

// textWidth measures a string in a canvas.Text's own face and size.
//
// Its own, which fyne.MeasureText cannot give: that passes a nil font source
// and so answers in the application's font whatever this text is drawn in.
// The source set by refit is the face the glyphs will actually have, and
// measuring in anything else is how a label loses its last letter.
func textWidth(t *canvas.Text, s string) float32 {
	a := fyne.CurrentApp()
	if a == nil || a.Driver() == nil {
		return 0
	}
	out, _ := a.Driver().RenderedTextSize(s, t.TextSize, t.TextStyle, t.FontSource)
	return out.Width
}

/*
CellGrid lays cells out across the width it is given, as many to a line as fit,
wrapping.

Every cell is the same width, which is the widest cell's minimum. A column that
fitted its own contents would put the readings at a different place on every
line, and a column of figures that do not line up is what the fixed-width
formatters exist to prevent.

The count follows the width. The archetype draws two to a line because its
panel is 260 px wide and it has four fixed slots; this fits what it can, which
is the same answer at that width and a better one at any other.

It is not layout.NewGridWrapLayout. That layout takes one fixed size for every
cell and is *told* it rather than measuring, so a text size change would leave
it laying out at the old one -- and a glance panel's text size is a setting the
user can move.
*/
type CellGrid struct {
	themed

	cells  []*Cell
	box    *fyne.Container
	layout *cellGridLayout
}

// NewCellGrid builds an empty grid. Cells are added with Add.
func NewCellGrid() *CellGrid {
	g := &CellGrid{}
	g.layout = &cellGridLayout{grid: g}
	g.box = container.New(g.layout)
	return g
}

// SetTheme gives the grid and every cell in it a theme of its own. A panel
// calls it; there is nothing for a consumer to do.
func (g *CellGrid) SetTheme(th fyne.Theme) {
	g.themed.SetTheme(th)
	for _, c := range g.cells {
		c.SetTheme(th)
	}
	g.box.Refresh()
}

// Add puts cells in the grid, in the order they are to be drawn.
func (g *CellGrid) Add(cells ...*Cell) {
	for _, c := range cells {
		c.SetTheme(g.th)
		g.cells = append(g.cells, c)
		g.box.Add(c.Object())
	}
	g.box.Refresh()
}

// Cells are the grid's cells, for a caller that keeps handles to them anyway
// and a test that would rather not.
func (g *CellGrid) Cells() []*Cell { return g.cells }

// Restyle repaints every cell in the current theme and re-measures the grid,
// which a card cannot do for it when the grid went in through AddObject: that
// takes a plain canvas object and cannot know it has a Restyle of its own. A
// grid added with Card.Add is kept in step by the card.
func (g *CellGrid) Restyle() {
	for _, c := range g.cells {
		c.Restyle()
	}
	g.box.Refresh()
}

// Object is the grid's content, for adding to a card.
func (g *CellGrid) Object() fyne.CanvasObject { return g.box }

/*
cellGridLayout puts as many equal-width cells on a line as the width fits.

It remembers the width it was last laid out at, because MinSize has to answer
in lines and Fyne does not tell it the width. A grid that always reported the
height of a single column reserved four rows and drew one, which in a card is
four rows of empty panel under the readings. Fyne measures again after a
resize, so the remembered width is the one the grid is actually at by the time
anybody sees it; before the first layout it is zero, which reports one column
and is the honest answer for a grid nobody has given a size to yet.
*/
type cellGridLayout struct {
	grid  *CellGrid
	width float32
}

// gap is the space between two cells, in the grid's own face.
func (l *cellGridLayout) gap() float32 {
	if l.grid == nil {
		return CellGridGap()
	}
	return l.grid.gridGap()
}

// MinSize is one cell wide and as many lines as the last width implies.
//
// One cell wide, because a grid that demanded room for a pair would set the
// width of a panel that only ever holds one device.
func (l *cellGridLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	shown := visible(objects)
	if len(shown) == 0 {
		return fyne.Size{}
	}
	cell := cellSize(shown)

	gap := l.gap()
	lines := lineCount(len(shown), min(perLine(l.width, cell.Width, gap), len(shown)))
	return fyne.NewSize(cell.Width, float32(lines)*cell.Height+float32(lines-1)*gap)
}

// Layout fills each line before starting the next.
func (l *cellGridLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	shown := visible(objects)
	if len(shown) == 0 {
		return
	}
	l.width = size.Width

	cell := cellSize(shown)
	gap := l.gap()

	// Never more columns than there are cells. A grid with room for three and
	// two devices in it used to lay them out in the first two thirds and
	// leave the last third empty, which reads as a panel that does not fit
	// its own window -- and on a card whose other rows stretch to the edge it
	// is the one thing that looks unfinished.
	across := min(perLine(size.Width, cell.Width, gap), len(shown))

	width := (size.Width - float32(across-1)*gap) / float32(across)
	for i, o := range shown {
		col, row := i%across, i/across
		o.Resize(fyne.NewSize(width, cell.Height))
		o.Move(fyne.NewPos(float32(col)*(width+gap), float32(row)*(cell.Height+gap)))
	}
}

// lineCount is how many rows n cells take at a given number across.
func lineCount(n, across int) int {
	if across < 1 {
		across = 1
	}
	return (n + across - 1) / across
}

// perLine is how many cells of a width fit across a space, at least one.
//
// At least one, because a grid narrower than a cell still has to draw it: a
// card clipped to nothing is worse than a card whose reading runs to its edge,
// and the panel has a minimum width of its own that this is inside.
func perLine(space, cell, gap float32) int {
	if cell <= 0 {
		return 1
	}
	n := 1
	for (float32(n+1))*cell+float32(n)*gap <= space {
		n++
	}
	return n
}

// cellSize is the size every cell is drawn at: the widest and tallest of them,
// so they line up with each other rather than each with itself.
func cellSize(objects []fyne.CanvasObject) fyne.Size {
	var out fyne.Size
	for _, o := range objects {
		m := o.MinSize()
		out.Width = max(out.Width, m.Width)
		out.Height = max(out.Height, m.Height)
	}
	return out
}

// visible is the objects that are drawn, so a hidden cell costs no space
// rather than leaving a gap where a device used to be.
func visible(objects []fyne.CanvasObject) []fyne.CanvasObject {
	out := make([]fyne.CanvasObject, 0, len(objects))
	for _, o := range objects {
		if o.Visible() {
			out = append(out, o)
		}
	}
	return out
}
