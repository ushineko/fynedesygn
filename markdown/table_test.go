package markdown

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/require"

	"github.com/ushineko/fynedesygn/fynetest"
)

// lopsided is the shape that showed the fault: a column of short labels
// beside a column of paragraphs.
func lopsided() [][]string {
	return [][]string{
		{"What", "What it does"},
		{"Lighting", strings.Repeat("every device OpenRGB can see, ", 6)},
		{"Scenes", "a named set of colour assignments"},
		{"Hotkeys", strings.Repeat("nine shipped on the numpad, ", 9)},
	}
}

func TestEachRowIsAsTallAsItsOwnContent(t *testing.T) {
	/*
		The fault. container.NewGridWithColumns gives every cell the same
		size, so every row was as tall as the tallest row in the table: a
		two-column table of a label and a paragraph came out with an inch of
		nothing between the rows and read as neither a list nor a table.
	*/
	test.NewTempApp(t)

	drawn := renderTable(lopsided())
	win := test.NewWindow(drawn)
	t.Cleanup(win.Close)
	win.Resize(fyne.NewSize(900, 700))

	rows := rowHeights(t, drawn)
	require.Len(t, rows, 4)

	short, long := rows[2], rows[3]
	require.Less(t, short, long, "a one-line row is as tall as a four-line one")
}

func TestAColumnOfLabelsIsNotHalfTheTable(t *testing.T) {
	// Proportioned to what is in them, which is what a reader's eye uses.
	weights := columnWeights(lopsided())
	require.Len(t, weights, 2)
	require.Less(t, weights[0], weights[1], "the label column is as wide as the prose")
	require.InDelta(t, 1, weights[0]+weights[1], 0.001)
}

func TestColumnsOfTheSameLengthComeOutEven(t *testing.T) {
	// Proportioning by content has to leave an even table even.
	weights := columnWeights([][]string{
		{"Before", "After"},
		{"thirteen char", "thirteen char"},
		{"also thirteen", "also thirteen"},
	})
	require.InDelta(t, weights[0], weights[1], 0.001)
}

func TestAColumnOfLongKeysIsWiderThanOneOfShortNames(t *testing.T) {
	// And an uneven one uneven, in the direction the content says.
	weights := columnWeights([][]string{
		{"Key", "Scene"},
		{"Ctrl+Alt+Num+1", "red"},
		{"Ctrl+Alt+Num+2", "green"},
	})
	require.Greater(t, weights[0], weights[1])
}

func TestANarrowColumnStillFitsAWord(t *testing.T) {
	/*
		Without the floor, a column of one-word labels beside a column of
		paragraphs is drawn at about a twentieth of the width and wraps every
		label into a column of single letters.
	*/
	weights := columnWeights([][]string{
		{"A", "B"},
		{"x", strings.Repeat("long ", 200)},
	})
	require.Greater(t, weights[0], float32(0.05))
}

func TestATableBringsNoScroller(t *testing.T) {
	// Fyne's own table widget lives in one, and a scroller inside a document
	// takes the wheel from the page.
	test.NewTempApp(t)
	require.False(t, fynetest.ScrollableIn(renderTable(lopsided())))
}

func TestATableSaysWhatItSays(t *testing.T) {
	test.NewTempApp(t)
	said := fynetest.Text(renderTable(lopsided()))
	require.Contains(t, said, "Lighting")
	require.Contains(t, said, "a named set of colour assignments")
}

func TestARowLongerThanItsHeaderDoesNotPanic(t *testing.T) {
	// Somebody's Markdown, not this program's.
	test.NewTempApp(t)
	require.NotPanics(t, func() {
		renderTable([][]string{{"one"}, {"a", "b", "c"}})
	})
}

// rowHeights is each row's drawn height, skipping the rules between them.
func rowHeights(t *testing.T, drawn fyne.CanvasObject) []float32 {
	t.Helper()
	box, ok := drawn.(*fyne.Container)
	require.True(t, ok)

	var out []float32
	for _, o := range box.Objects {
		if row, ok := o.(*fyne.Container); ok {
			out = append(out, row.Size().Height)
		}
	}
	return out
}
