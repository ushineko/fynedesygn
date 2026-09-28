package glance_test

import (
	"image/color"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	fd "github.com/ushineko/fynedesygn"
	"github.com/ushineko/fynedesygn/fynetest"
	"github.com/ushineko/fynedesygn/glance"
)

// battery is a cell with a name, a reading and a state, which is all three
// pieces a cell has.
func battery(t *testing.T, name, level, note string) *glance.Cell {
	t.Helper()
	c := glance.NewCell(name, "-- %")
	c.Set(glance.Known(level, fd.StatusGood))
	c.SetNote(note)
	return c
}

// The reading is larger than the name and the state, and that is the whole of
// what makes a cell a cell rather than a row. Drawn at one size the figure is
// no more the subject than the name over it.
func TestACellsReadingIsLargerThanItsNameAndItsState(t *testing.T) {
	_ = fynetest.App(t)
	c := battery(t, "G502 X PLUS", "67 %", "Discharging")

	sizes := textSizes(c.Object())

	require.Len(t, sizes, 3, "a cell is a name, a reading and a state")
	assert.Greater(t, sizes[1], sizes[0], "the reading is not larger than the name")
	assert.Greater(t, sizes[1], sizes[2], "the reading is not larger than the state")
	assert.InDelta(t, theme.TextSize()*glance.CellValueScale, sizes[1], 0.01)
}

// A cell carries the same Reading a row does, colour rules and all.
func TestACellTakesItsColourFromItsReading(t *testing.T) {
	_ = fynetest.App(t)
	c := glance.NewCell("Arctis", "-- %")

	c.Set(glance.Known("15 %", fd.StatusBad))
	bad := valueColour(c.Object())

	c.Set(glance.Known("85 %", fd.StatusGood))
	good := valueColour(c.Object())

	assert.NotEqual(t, bad, good, "the verdict did not reach the reading")
}

// Stale wins over the verdict, exactly as it does for a row: a warning nobody
// is refreshing should not keep shouting.
func TestAStaleReadingIsDimmedWhateverItsVerdict(t *testing.T) {
	_ = fynetest.App(t)
	c := glance.NewCell("Arctis", "-- %")

	c.Set(glance.Known("15 %", fd.StatusBad))
	rd := c.Reading()
	rd.Stale = true
	c.Set(rd)

	assert.Equal(t, theme.Color(theme.ColorNameDisabled), valueColour(c.Object()))
}

/*
A device name is as transient as the device -- a mouse waking up, a headset
changing what it calls itself -- so a name may ask for width up to
CellNameBudget and no further.

A cell sized by its reading alone is forty pixels wide and four device names
drawn into four such columns run into each other, which is what the first
capture of this showed. A cell sized by its name reflows the panel when a
peripheral connects, which is the one thing a glance window must not do. The
budget is the compromise, and what this asserts is that it is a *bound*.
*/
func TestALongNameCannotWidenACellPastItsBudget(t *testing.T) {
	_ = fynetest.App(t)
	c := battery(t, "G502", "67 %", "Discharging")

	c.SetName("Arctis Nova Pro Wireless Base Station Edition")
	wide := c.Object().MinSize().Width

	c.SetName("Arctis Nova Pro Wireless Base Station Edition, And Then Some More Of It")
	assert.InDelta(t, wide, c.Object().MinSize().Width, 0.01,
		"a longer name kept widening the cell")
	assert.LessOrEqual(t, wide, glance.CellNameBudget(),
		"a device name set the width of the cell")
}

// And the grid is bounded with it, which is the size that actually reaches the
// card and the window.
func TestALongNameCannotWidenTheGridPastItsBudget(t *testing.T) {
	_ = fynetest.App(t)
	g := glance.NewCellGrid()
	c := battery(t, "G502", "67 %", "Discharging")
	g.Add(c, battery(t, "B", "20 %", "Discharging"))

	c.SetName("Arctis Nova Pro Wireless Base Station Edition")

	assert.LessOrEqual(t, g.Object().MinSize().Width, glance.CellNameBudget(),
		"a device waking up with a long name would resize the panel")
}

// The name that does not fit is elided rather than drawn over its neighbour.
// The first capture of this drew "G502 X PLUSKeychron K4 HE AirPods Pro" as
// one unbroken line, which is what a canvas.Text does when nobody cuts it.
func TestANameTooLongForItsCellIsElided(t *testing.T) {
	_ = fynetest.App(t)
	c := battery(t, "Arctis Nova Pro Wireless Base Station", "67 %", "Discharging")
	o := c.Object()
	o.Resize(o.MinSize())

	drawn := drawnName(o)

	assert.NotEqual(t, c.Name(), drawn, "the name was drawn in full in a cell too narrow for it")
	assert.Contains(t, drawn, glance.Ellipsis)
	assert.True(t, strings.HasPrefix(c.Name(), strings.TrimSuffix(drawn, glance.Ellipsis)),
		"the elided name is not a prefix of the real one: %q", drawn)
}

/*
A name set *after* the cell has a size is elided too.

A poll sets a name and a poll does not re-lay-out -- the container measures
when it is resized, not when a canvas.Text inside it changes -- so a cell whose
device was renamed drew the new name in full, straight over its neighbour.
That is what the panel showed the first time a real device went into one, and
no test above catches it because they all set the name before the resize.
*/
func TestANameSetAfterTheCellHasASizeIsElidedToo(t *testing.T) {
	_ = fynetest.App(t)
	c := battery(t, "G502", "67 %", "Discharging")
	o := c.Object()
	o.Resize(o.MinSize())

	c.SetName("Arctis Nova Pro Wireless Base Station")

	assert.Contains(t, drawnName(o), glance.Ellipsis,
		"a name set on a poll was drawn in full, over its neighbour")
}

// And so is a state, for the same reason.
func TestANoteSetAfterTheCellHasASizeIsElidedToo(t *testing.T) {
	_ = fynetest.App(t)
	c := battery(t, "G502", "67 %", "Discharging")
	o := c.Object()
	o.Resize(o.MinSize())

	c.SetNote("L 80  R 90  case 50  Charging from the case")

	assert.Contains(t, canvasTexts(o)[2].Text, glance.Ellipsis)
}

// A name that fits is left alone, ellipsis and all.
func TestANameThatFitsIsNotElided(t *testing.T) {
	_ = fynetest.App(t)
	c := battery(t, "G502", "67 %", "Discharging")
	o := c.Object()
	o.Resize(o.MinSize())

	assert.Equal(t, "G502", drawnName(o))
}

// Name and Note report what was set, not what was drawn. A caller that read
// back an elided name and set it again would lose a character per layout.
func TestACellReportsTheNameItWasGivenNotTheOneItDrew(t *testing.T) {
	_ = fynetest.App(t)
	full := "Arctis Nova Pro Wireless Base Station"
	c := battery(t, full, "67 %", "Discharging")
	o := c.Object()
	o.Resize(o.MinSize())

	assert.Equal(t, full, c.Name())
	assert.Equal(t, "Discharging", c.Note())
}

// Set repaints and does not re-lay-out: the text it is given is the same
// width as the text it had, which is what the formatters guarantee. A cell
// that re-measured on every poll would reflow the panel several times a
// minute.
func TestSettingAReadingOfTheSameWidthDoesNotResizeTheCell(t *testing.T) {
	_ = fynetest.App(t)
	c := battery(t, "G502", glance.Percent(67), "Discharging")
	before := c.Object().MinSize()

	c.Set(glance.Known(glance.Percent(100), fd.StatusGood))

	assert.Equal(t, before, c.Object().MinSize())
}

// A longer reading may widen it, because a reading is what the cell is for and
// is formatted to a fixed width by the caller.
func TestALongerReadingIsAllowedToWidenTheCell(t *testing.T) {
	_ = fynetest.App(t)
	c := glance.NewCell("G502", "--")
	narrow := c.Object().MinSize().Width

	c.Set(glance.Measured("100000 mAh"))

	assert.Greater(t, c.Object().MinSize().Width, narrow)
}

// Restyle brings a cell up to date after a text size change. A canvas.Text
// holds a literal size rather than asking the theme at paint time, so nothing
// else does.
func TestRestyleBringsACellUpToDateAfterATextSizeChange(t *testing.T) {
	a := fynetest.App(t)
	c := battery(t, "G502", "67 %", "Discharging")
	before := textSizes(c.Object())

	a.Settings().SetTheme(bigTheme{Theme: a.Settings().Theme()})
	c.Restyle()

	after := textSizes(c.Object())
	for i := range before {
		assert.Greater(t, after[i], before[i], "piece %d did not follow the theme", i)
	}
}

// A grid puts as many cells on a line as the width fits and wraps the rest.
func TestAGridFitsWhatItCanOnALineAndWraps(t *testing.T) {
	_ = fynetest.App(t)
	g := glance.NewCellGrid()
	for _, n := range []string{"A", "B", "C", "D"} {
		g.Add(battery(t, n, "67 %", "Discharging"))
	}
	o := g.Object()
	cell := o.MinSize().Width

	// Room for two across, which should be two lines of two.
	o.Resize(fyne.NewSize(cell*2+glance.CellGridGap(), o.MinSize().Height))
	assert.Equal(t, 2, linesIn(g), "four cells in a width of two should be two lines")

	// Room for four across is one line.
	o.Resize(fyne.NewSize(cell*4+3*glance.CellGridGap(), o.MinSize().Height))
	assert.Equal(t, 1, linesIn(g))
}

// Every cell in a grid is the same width. A column that fitted its own
// contents would put the readings at a different place on every line, and a
// column of figures that do not line up is what the fixed-width formatters
// exist to prevent.
func TestEveryCellInAGridIsTheSameWidth(t *testing.T) {
	_ = fynetest.App(t)
	g := glance.NewCellGrid()
	g.Add(
		battery(t, "A", "5 %", "Discharging"),
		battery(t, "B", "100 %", "Charged"),
		battery(t, "C", "--", "No reading"),
	)

	o := g.Object()
	o.Resize(fyne.NewSize(o.MinSize().Width*3+2*glance.CellGridGap(), o.MinSize().Height))

	widths := map[float32]bool{}
	for _, c := range g.Cells() {
		widths[c.Object().Size().Width] = true
	}
	assert.Len(t, widths, 1, "the cells are not one column width: %v", widths)
}

// A grid narrower than two cells draws one to a line rather than clipping. The
// panel has a minimum width of its own and this is inside it.
func TestAGridTooNarrowForTwoCellsDrawsOne(t *testing.T) {
	_ = fynetest.App(t)
	g := glance.NewCellGrid()
	g.Add(battery(t, "A", "67 %", "Discharging"), battery(t, "B", "20 %", "Discharging"))

	o := g.Object()
	o.Resize(fyne.NewSize(o.MinSize().Width, o.MinSize().Height))

	assert.Equal(t, 2, linesIn(g))
}

// A hidden cell costs no space rather than leaving a gap where a device used
// to be.
func TestAHiddenCellLeavesNoGap(t *testing.T) {
	_ = fynetest.App(t)
	g := glance.NewCellGrid()
	a := battery(t, "A", "67 %", "Discharging")
	g.Add(a, battery(t, "B", "20 %", "Discharging"))
	both := g.Object().MinSize().Height

	a.SetShown(false)

	assert.Less(t, g.Object().MinSize().Height, both)
}

// A grid asks for one cell's width, not two. A grid that demanded room for a
// pair would set the width of a panel that only ever holds one device.
func TestAGridAsksForOneCellOfWidth(t *testing.T) {
	_ = fynetest.App(t)
	one := glance.NewCellGrid()
	one.Add(battery(t, "A", "67 %", "Discharging"))

	four := glance.NewCellGrid()
	for _, n := range []string{"A", "B", "C", "D"} {
		four.Add(battery(t, n, "67 %", "Discharging"))
	}

	assert.InDelta(t, one.Object().MinSize().Width, four.Object().MinSize().Width, 0.01)
}

// A grid restyles its cells, because the card cannot: a grid goes in through
// AddObject, which takes a plain canvas object and cannot know it has a
// Restyle of its own. This is the same gap the sparkline has.
func TestAGridRestylesItsCells(t *testing.T) {
	a := fynetest.App(t)
	g := glance.NewCellGrid()
	c := battery(t, "A", "67 %", "Discharging")
	g.Add(c)
	before := textSizes(c.Object())

	a.Settings().SetTheme(bigTheme{Theme: a.Settings().Theme()})
	g.Restyle()

	assert.Greater(t, textSizes(c.Object())[1], before[1])
}

// A card takes a grid like any other object, which is how a consumer puts one
// in a panel.
func TestACardTakesAGrid(t *testing.T) {
	_ = fynetest.App(t)
	g := glance.NewCellGrid()
	g.Add(battery(t, "G502 X PLUS", "67 %", "Discharging"))

	c := glance.NewCard("Peripherals")
	c.AddObject(g.Object())

	assert.Contains(t, fynetest.Texts(c.Object()), "G502 X PLUS")
	assert.Contains(t, fynetest.Texts(c.Object()), "67 %")
	assert.Contains(t, fynetest.Texts(c.Object()), "Discharging")
}

// textSizes are the sizes of a cell's three pieces, in the order they are
// drawn. A canvas.Text is what a cell is made of, so this is the object tree
// rather than a rendering -- which is the right question for "did Restyle
// reach it".
func textSizes(o fyne.CanvasObject) []float32 {
	var out []float32
	for _, t := range canvasTexts(o) {
		out = append(out, t.TextSize)
	}
	return out
}

// drawnName is the name a cell actually draws, which is the elided one when
// the cell is narrower than the words.
func drawnName(o fyne.CanvasObject) string {
	texts := canvasTexts(o)
	if len(texts) == 0 {
		return ""
	}
	return texts[0].Text
}

// valueColour is the colour of a cell's reading, which is the middle piece.
func valueColour(o fyne.CanvasObject) color.Color {
	texts := canvasTexts(o)
	if len(texts) < 2 {
		return nil
	}
	return texts[1].Color
}

// canvasTexts walks an object tree and collects its canvas.Texts in order.
func canvasTexts(o fyne.CanvasObject) []*canvas.Text {
	var out []*canvas.Text
	// false keeps walking: Walk's visit returns true to *stop*.
	fynetest.Walk(o, func(obj fyne.CanvasObject) bool {
		if t, ok := obj.(*canvas.Text); ok {
			out = append(out, t)
		}
		return false
	})
	return out
}

// linesIn is how many rows of cells a grid laid out, counted from where it put
// them rather than from what it was asked for.
func linesIn(g *glance.CellGrid) int {
	tops := map[float32]bool{}
	for _, c := range g.Cells() {
		if c.Shown() {
			tops[c.Object().Position().Y] = true
		}
	}
	return len(tops)
}

// bigTheme is the current theme with a larger face, for asking whether a
// Restyle actually reached a canvas.Text.
type bigTheme struct{ fyne.Theme }

func (t bigTheme) Size(n fyne.ThemeSizeName) float32 {
	if n == theme.SizeNameText {
		return t.Theme.Size(n) * 2
	}
	return t.Theme.Size(n)
}

// A grid never lays out more columns than it has cells. Room for three and two
// devices in it used to put them in the first two thirds and leave the last
// third empty, which reads as a panel that does not fit its own window.
func TestAGridNeverUsesMoreColumnsThanItHasCells(t *testing.T) {
	_ = fynetest.App(t)
	g := glance.NewCellGrid()
	g.Add(battery(t, "A", "67 %", "Discharging"), battery(t, "B", "20 %", "Discharging"))

	o := g.Object()
	wide := o.MinSize().Width*4 + 3*glance.CellGridGap()
	o.Resize(fyne.NewSize(wide, o.MinSize().Height))

	cells := g.Cells()
	require.Len(t, cells, 2)
	assert.Equal(t, 1, linesIn(g), "two cells in a width of four should be one line")

	right := cells[1].Object().Position().X + cells[1].Object().Size().Width
	assert.InDelta(t, wide, right, 0.5,
		"the cells did not reach the right-hand edge: %v", right)
}
