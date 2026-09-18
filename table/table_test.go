package table

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/require"

	fd "github.com/ushineko/fynedesygn"
)

func fourRows() *Detail {
	d := New()
	d.Header("Name", "Verdict", "Detail")
	d.Row(fd.StatusInfo, "a", "info", "one")
	d.Row(fd.StatusGood, "b", "good", "two")
	d.Row(fd.StatusWarn, "c", "warn", "three")
	d.Row(fd.StatusBad, "d", "bad", "four")
	return d
}

func TestCellsCarryTheRowStatusNotJustTheVerdictColumn(t *testing.T) {
	d := fourRows()
	require.Equal(t, 4, d.Len())
	want := []widget.Importance{
		widget.MediumImportance, widget.SuccessImportance,
		widget.WarningImportance, widget.DangerImportance,
	}
	for row, imp := range want {
		for col := range 3 {
			text, got := d.Cell(row, col)
			require.NotEmpty(t, text)
			require.Equal(t, imp, got, "row %d col %d", row, col)
		}
	}
}

func TestCellOutOfRangeIsEmptyAndMedium(t *testing.T) {
	d := fourRows()
	for _, id := range [][2]int{{-1, 0}, {4, 0}, {0, -1}, {0, 3}} {
		text, imp := d.Cell(id[0], id[1])
		require.Empty(t, text, "cell %v", id)
		require.Equal(t, widget.MediumImportance, imp, "cell %v", id)
	}
}

func TestHeaderTextGuardsTheCornerCell(t *testing.T) {
	d := fourRows()
	require.Equal(t, "", d.HeaderText(-1))
	require.Equal(t, "", d.HeaderText(3))
	require.Equal(t, "Name", d.HeaderText(0))
	require.Equal(t, "Detail", d.HeaderText(2))
}

func TestMeasureClampsBetween70And460(t *testing.T) {
	d := New()
	d.Header("x", "long", "")
	d.Row(fd.StatusInfo, "y", string(bytes.Repeat([]byte("w"), 200)), "")
	require.Equal(t, float32(70), d.Measure(0), "a one-rune column sits at the floor")
	require.Equal(t, float32(460), d.Measure(1), "a 200-rune column sits at the ceiling")
	require.Equal(t, float32(70), d.Measure(2), "an empty column sits at the floor")
	require.Equal(t, float32(70), d.Measure(-1), "an out-of-range column is measured as empty")
	d2 := New()
	d2.Header("ten chars!")
	require.Equal(t, float32(10*7+24), d2.Measure(0), "7 px per rune plus 24 in between the clamps")
}

// tinyPNG is a valid 2x2 PNG for exercising the thumbnail column.
func tinyPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.NRGBA{R: 255, A: 255})
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}

func TestWidgetRendersHeadlessly(t *testing.T) {
	test.NewTempApp(t)
	t.Run("without thumbnails", func(t *testing.T) {
		d := fourRows()
		d.SetWidths(120)
		require.NotPanics(t, func() {
			tbl := d.Widget()
			r := test.WidgetRenderer(tbl)
			tbl.Resize(fyne.NewSize(600, 300))
			r.Refresh()
		})
	})
	t.Run("with thumbnails", func(t *testing.T) {
		d := New()
		d.Header("Thumb", "Name")
		d.Row(fd.StatusGood, "", "with image")
		d.Row(fd.StatusInfo, "", "without image")
		d.SetThumbnails(0, [][]byte{tinyPNG(t), nil})
		require.NotPanics(t, func() {
			tbl := d.Widget()
			r := test.WidgetRenderer(tbl)
			tbl.Resize(fyne.NewSize(600, 300))
			r.Refresh()
			// Drive the cell callbacks directly as well, so the image path is
			// exercised even if the headless table lays out no visible cells.
			cell := tbl.CreateCell()
			tbl.UpdateCell(widget.TableCellID{Row: 0, Col: 0}, cell)
			tbl.UpdateCell(widget.TableCellID{Row: 1, Col: 0}, cell)
			tbl.UpdateCell(widget.TableCellID{Row: 0, Col: 1}, cell)
		})
		require.NotNil(t, d.thumb(0))
		require.Same(t, d.thumb(0), d.thumb(0), "decoded thumbnails are cached per row")
		require.Nil(t, d.thumb(1))
	})
}

// TestUpdateHeaderVisitsCornerCell is canary #4 in docs/fyne-quirks.md:
// widget.Table calls UpdateHeader with Col == -1 for the corner cell even when
// row headers are off. The table guards it; this test drives the callback
// with the corner id directly so a change in Fyne's calling convention shows
// up here rather than as a panic in three apps.
func TestUpdateHeaderVisitsCornerCell(t *testing.T) {
	test.NewTempApp(t)
	tbl := fourRows().Widget()
	require.NotPanics(t, func() {
		h := tbl.CreateHeader()
		tbl.UpdateHeader(widget.TableCellID{Row: -1, Col: -1}, h)
		require.Equal(t, "", h.(*widget.Label).Text)
		tbl.UpdateHeader(widget.TableCellID{Row: -1, Col: 1}, h)
		require.Equal(t, "Verdict", h.(*widget.Label).Text)
	})
}

// textColour digs the canvas.Text out of a label's renderer tree.
func textColour(t *testing.T, l *widget.Label) color.Color {
	t.Helper()
	var found *canvas.Text
	var walk func(o fyne.CanvasObject)
	walk = func(o fyne.CanvasObject) {
		if found != nil {
			return
		}
		switch v := o.(type) {
		case *canvas.Text:
			found = v
		case *fyne.Container:
			for _, c := range v.Objects {
				walk(c)
			}
		case fyne.Widget:
			for _, c := range test.WidgetRenderer(v).Objects() {
				walk(c)
			}
		}
	}
	walk(l)
	require.NotNil(t, found, "label renders a canvas.Text")
	return found.Color
}

// TestImportanceAfterSetTextKeepsOldColour is canary #3 in docs/fyne-quirks.md.
// Label.SetText refreshes the label, and the refresh (syncSegments) is where
// Importance becomes a colour. Setting Importance after SetText therefore
// leaves the previous colour on screen until something else refreshes the
// label. The table sets Importance first; this test pins the Fyne behaviour
// that makes the ordering matter. If it fails after a Fyne bump, the ordering
// comment in Widget can be relaxed.
func TestImportanceAfterSetTextKeepsOldColour(t *testing.T) {
	test.NewTempApp(t)
	variant := fyne.CurrentApp().Settings().ThemeVariant()
	danger := theme.Color(theme.ColorNameError)
	success := theme.Color(theme.ColorNameSuccess)
	require.NotEqual(t, danger, success, "the theme tells error and success apart (variant %v)", variant)

	l := widget.NewLabel("")
	test.WidgetRenderer(l) // create the renderer; before that Refresh is a no-op
	l.Importance = widget.DangerImportance
	l.SetText("a")
	require.Equal(t, danger, textColour(t, l))

	// The wrong order: text first, importance second, no further refresh.
	l.SetText("b")
	l.Importance = widget.SuccessImportance
	require.Equal(t, danger, textColour(t, l), "importance set after SetText is not painted")

	// The right order paints the new colour on the same refresh.
	l.Importance = widget.WarningImportance
	l.SetText("c")
	require.Equal(t, theme.Color(theme.ColorNameWarning), textColour(t, l))
}

func TestSortTitleAppendsTheArrowForTheSortedDirection(t *testing.T) {
	require.Equal(t, "Name", SortTitle("Name", Unsorted))
	require.Equal(t, "Name  ↑", SortTitle("Name", Ascending))
	require.Equal(t, "Name  ↓", SortTitle("Name", Descending))
}

func TestSortHeaderIsALowImportanceLeadingButtonThatTaps(t *testing.T) {
	test.NewTempApp(t)
	tapped := 0
	b := SortHeader("Size", Descending, func() { tapped++ })
	require.Equal(t, "Size  ↓", b.Text)
	require.Equal(t, widget.LowImportance, b.Importance)
	require.Equal(t, widget.ButtonAlignLeading, b.Alignment)
	test.Tap(b)
	require.Equal(t, 1, tapped)
}
