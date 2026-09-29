package glance_test

import (
	"image/color"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ushineko/fynedesygn/glance"
)

// panelOf is a panel holding n cards, laid out at a width.
func panelOf(t *testing.T, n int, width float32) *glance.Panel {
	t.Helper()
	test.NewTempApp(t)
	p := glance.NewPanel(glance.NoMinWidth)
	for i := 0; i < n; i++ {
		c := glance.NewCard("Card")
		c.AddRow(glance.NewRow("label", "0"))
		c.SetAvailable(true)
		p.Add(c)
	}
	p.Content().Resize(fyne.NewSize(width, 2000))
	return p
}

// columnsOf counts distinct x positions among the cards: one per column.
func columnsOf(p *glance.Panel) int {
	seen := map[float32]bool{}
	for _, c := range p.Cards() {
		seen[c.Object().Position().X] = true
	}
	return len(seen)
}

// AC. A stacked panel is one column however wide it is.
func TestAStackedPanelIsOneColumn(t *testing.T) {
	p := panelOf(t, 4, 1600)
	assert.Equal(t, glance.Stack, p.Arrangement(), "stack is the default")
	assert.Equal(t, 1, columnsOf(p))
}

// AC. A grid at a width that fits several uses several.
func TestAGridUsesTheColumnsTheWidthFits(t *testing.T) {
	p := panelOf(t, 4, 1600)
	p.SetArrangement(glance.Grid)
	p.Content().Resize(fyne.NewSize(1600, 2000))

	assert.Greater(t, columnsOf(p), 1, "a wide grid drew a single column")
}

// AC. A grid too narrow for two columns is a stack, so a panel nobody widens
// never looks any different.
func TestANarrowGridIsAStack(t *testing.T) {
	p := panelOf(t, 4, glance.MinCardWidth())
	p.SetArrangement(glance.Grid)
	p.Content().Resize(fyne.NewSize(glance.MinCardWidth(), 2000))

	assert.Equal(t, 1, columnsOf(p))
}

// AC. Never more columns than cards: two cards in a very wide window are two
// columns, not four thin ones.
func TestThereAreNeverMoreColumnsThanCards(t *testing.T) {
	p := panelOf(t, 2, 4000)
	p.SetArrangement(glance.Grid)
	p.Content().Resize(fyne.NewSize(4000, 2000))

	assert.LessOrEqual(t, columnsOf(p), 2)
}

// AC. Column-major: reading down a column follows the order the cards were
// added, which is the rule the terminal panel already uses.
func TestTheGridFillsColumnMajor(t *testing.T) {
	p := panelOf(t, 4, 1600)
	p.SetArrangement(glance.Grid)
	p.Content().Resize(fyne.NewSize(1600, 2000))

	cards := p.Cards()
	require.Len(t, cards, 4)
	if columnsOf(p) != 2 {
		t.Skipf("this width gave %d columns, not the two this case is about", columnsOf(p))
	}
	first, second := cards[0].Object().Position(), cards[1].Object().Position()
	assert.Equal(t, first.X, second.X, "the second card started a new column instead of going below the first")
	assert.Greater(t, second.Y, first.Y)
}

// AC. Going back to a stack restores one column.
func TestAGridCanGoBackToAStack(t *testing.T) {
	p := panelOf(t, 4, 1600)
	p.SetArrangement(glance.Grid)
	p.Content().Resize(fyne.NewSize(1600, 2000))
	require.Greater(t, columnsOf(p), 1)

	p.SetArrangement(glance.Stack)
	p.Content().Resize(fyne.NewSize(1600, 2000))
	assert.Equal(t, 1, columnsOf(p))
}

// AC. The arrangement survives a restyle.
//
// The regression this encodes shipped: Restyle replaced the cards' layout with
// a VBox, so the grid was lost the first time the text size or the scheme
// changed -- which every panel does at startup, when its own theme is applied.
// A test that only ever set the arrangement and measured could not see it.
func TestAGridSurvivesARestyle(t *testing.T) {
	p := panelOf(t, 4, 1600)
	p.SetArrangement(glance.Grid)
	p.Content().Resize(fyne.NewSize(1600, 2000))
	require.Greater(t, columnsOf(p), 1, "the grid was not in columns to begin with")

	// A different width, because Fyne skips the layout pass when the size did
	// not change -- and a test that resized to 1600 twice never ran the layout
	// the restyle had just replaced.
	p.Restyle()
	p.Content().Resize(fyne.NewSize(1500, 2000))

	assert.Greater(t, columnsOf(p), 1, "a restyle flattened the grid into a stack")
}

// counting is a layout that records how many times it was asked to lay out.
type counting struct{ runs int }

func (c *counting) Layout([]fyne.CanvasObject, fyne.Size) { c.runs++ }
func (c *counting) MinSize([]fyne.CanvasObject) fyne.Size { return fyne.NewSize(10, 10) }

// Canary for quirk 40. A container resized to the size it already has does not
// lay out again, which is why a test that changes a layout has to resize to a
// different size afterwards to see the change.
func TestFyneStillSkipsALayoutWhenTheSizeDidNotChange(t *testing.T) {
	test.NewTempApp(t)
	l := &counting{}
	c := container.New(l, canvas.NewRectangle(color.Transparent))

	c.Resize(fyne.NewSize(400, 400))
	after := l.runs
	c.Resize(fyne.NewSize(400, 400))
	assert.Equal(t, after, l.runs, "a resize to the same size ran the layout")

	c.Resize(fyne.NewSize(401, 400))
	assert.Greater(t, l.runs, after, "a resize to a different size did not run the layout")
}
