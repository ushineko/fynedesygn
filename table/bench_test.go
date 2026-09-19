package table_test

import (
	"fmt"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	fd "github.com/ushineko/fynedesygn"
	"github.com/ushineko/fynedesygn/table"
)

/*
BenchmarkResize is what a window drag does to a table: resize it, over and over,
with the contents unchanged.

It exists because that path was 738 microseconds per resize on a 300-row table,
and a drag is a resize per frame. Fyne resizes every visible cell on layout, and
a Label with wrapping or truncation on re-shapes its text through harfbuzz each
time. Taking Fyne's no-measure path and cutting the text here instead brought it
to 27 microseconds. Keep it under 100.
*/
func BenchmarkResize(b *testing.B) {
	t := table.New()
	t.Header("", "Name", "Kind", "Damage", "Defense", "Life", "Rarity", "ID")
	t.SetWidths(table.ThumbCellSize, 320, 150, 90, 90, 90, 90, 80)
	for i := range 300 {
		t.Row(fd.StatusInfo, "", fmt.Sprintf("Some Item Name %d", i), "Sword",
			"95", "0", "—", "8", fmt.Sprintf("%d", i))
	}
	w := t.Widget()
	win := test.NewWindow(w)
	defer win.Close()
	win.Resize(fyne.NewSize(1000, 700))

	b.ResetTimer()
	for i := 0; b.Loop(); i++ {
		w.Resize(fyne.NewSize(float32(900+i%60), 700))
	}
}
