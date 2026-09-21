package markdown

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
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

/*
rowHeights is each row's drawn height, skipping the rules.

Two levels down: the table is a frame holding the rows and the column rules
over them, and the rows are a stack of row containers and the horizontal
rules between them.
*/
func rowHeights(t *testing.T, drawn fyne.CanvasObject) []float32 {
	t.Helper()
	frame, ok := drawn.(*fyne.Container)
	require.True(t, ok)
	require.NotEmpty(t, frame.Objects)
	body, ok := frame.Objects[0].(*fyne.Container)
	require.True(t, ok, "the frame's first child should be the rows")

	var out []float32
	for _, o := range body.Objects {
		if row, ok := o.(*fyne.Container); ok {
			out = append(out, row.Size().Height)
		}
	}
	return out
}

func TestAnEmptyHeaderCellIsNotAHorizontalRule(t *testing.T) {
	/*
		A two-column table of label and description is often written with an
		empty header -- `| | |` -- and the first version made a header bold by
		wrapping the cell in asterisks, so an empty cell became `****`, which
		Markdown renders as a thematic break. Two empty header cells drew two
		short horizontal rules above the table.

		Editing somebody's Markdown to change how it looks is the fault. A
		cell that already carried emphasis, or a pipe, or nothing at all was
		going to produce something nobody wrote.
	*/
	test.NewTempApp(t)

	row := tableRow([]string{"", ""}, []float32{0.5, 0.5}, true)
	box, ok := row.(*fyne.Container)
	require.True(t, ok)

	for _, cell := range box.Objects {
		rt, ok := cell.(*widget.RichText)
		require.True(t, ok)
		for _, segment := range rt.Segments {
			require.IsType(t, &widget.TextSegment{}, segment,
				"an empty header cell rendered as something other than text")
		}
	}
}

func TestAHeaderIsBoldWithoutBeingRewritten(t *testing.T) {
	test.NewTempApp(t)

	row := tableRow([]string{"Key"}, []float32{1}, true)
	box, _ := row.(*fyne.Container)
	rt, ok := box.Objects[0].(*widget.RichText)
	require.True(t, ok)

	/*
		Every text segment bold, and none of them rewritten.

		A cell of one word comes back as two segments -- the word and a
		trailing empty one Fyne uses to close the paragraph -- so this reads
		what is there rather than assuming a count.
	*/
	var said string
	for _, segment := range rt.Segments {
		text, ok := segment.(*widget.TextSegment)
		require.True(t, ok)
		require.True(t, text.Style.TextStyle.Bold, "a header cell is not bold")
		said += text.Text
	}
	require.Equal(t, "Key", said, "a header cell's text was rewritten")
}

func TestATableIsRuledOnEverySideAReaderFollows(t *testing.T) {
	// Under the header, between the rows, under the last one, and down the
	// column boundaries.
	test.NewTempApp(t)

	drawn := renderTable(lopsided())
	frame, _ := drawn.(*fyne.Container)
	body, _ := frame.Objects[0].(*fyne.Container)

	var horizontals int
	for _, o := range body.Objects {
		if _, ok := o.(*canvas.Rectangle); ok {
			horizontals++
		}
	}
	// A roof, a rule under the header, and one under each row including the
	// last: one more than there are rows.
	require.Equal(t, len(lopsided())+1, horizontals,
		"a table is ruled above, under the header, between the rows and below")

	require.Len(t, frame.Objects[1:], 1, "one vertical per column boundary")
}

func TestATableWrittenWithoutHeadersHasNoHeaderRow(t *testing.T) {
	/*
		A pipe table always has a first row and a `|---|---|` under it, so a
		table written without headers is written with empty ones -- which is
		how hotaru's README writes its two-column tables, and how most people
		do. Both forms are ordinary Markdown.

		Drawn as a header, an empty row is a blank strip above the table.
	*/
	test.NewTempApp(t)

	headerless := [][]string{
		{"", ""},
		{"Lighting", "every device OpenRGB can see"},
		{"Scenes", "a named set of colour assignments"},
	}
	with := [][]string{
		{"What", "What it does"},
		{"Lighting", "every device OpenRGB can see"},
		{"Scenes", "a named set of colour assignments"},
	}

	require.Equal(t, 2, drawnRows(t, renderTable(headerless)),
		"an empty header was drawn as a row")
	require.Equal(t, 3, drawnRows(t, renderTable(with)),
		"a header was not drawn")
}

func TestAHeaderOfOneNamedColumnIsStillAHeader(t *testing.T) {
	// Partly empty is not empty: a table naming only its second column has a
	// header, and the blank cell is the author's choice.
	require.True(t, titled([]string{"", "What it does"}))
	require.False(t, titled([]string{"", "  "}))
}

// drawnRows counts the row containers in a rendered table.
func drawnRows(t *testing.T, drawn fyne.CanvasObject) int {
	t.Helper()
	frame, ok := drawn.(*fyne.Container)
	require.True(t, ok)
	body, ok := frame.Objects[0].(*fyne.Container)
	require.True(t, ok)

	var rows int
	for _, o := range body.Objects {
		if _, ok := o.(*fyne.Container); ok {
			rows++
		}
	}
	return rows
}
