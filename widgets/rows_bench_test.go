package widgets

import (
	"fmt"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"

	fd "github.com/ushineko/fynedesygn"
)

/*
BenchmarkRowsResize is what a window drag does to a card of fact rows: resize
it, over and over, both ways at once, as a corner drag does. nmsbonker's
Overview is 20-odd of these rows, and a drag of it spent half the process's
CPU shaping their values again through harfbuzz (Fyne quirk 30).
*/
func BenchmarkRowsResize(b *testing.B) {
	test.NewTempApp(b)
	rows := container.NewVBox()
	for i := range 12 {
		rows.Add(PlainRow(fmt.Sprintf("Plain %d", i),
			fmt.Sprintf("/home/someone/.local/share/Steam/steamapps/common/Some Game/GAMEDATA/PCBANKS/%d", i)))
		rows.Add(FactRow(fmt.Sprintf("Fact %d", i),
			fmt.Sprintf("current: 97 paks, 120000 files, %d minutes ago", i), fd.StatusGood))
	}
	w := container.NewVScroll(rows)
	win := test.NewWindow(w)
	defer win.Close()
	win.Resize(fyne.NewSize(1000, 700))

	b.ResetTimer()
	for i := 0; b.Loop(); i++ {
		w.Resize(fyne.NewSize(float32(500+i%600), float32(500+i%300)))
	}
}
