package widgets

import (
	"runtime"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/require"
)

// bigText is the default theme with a text size no theme ships, so a widget
// that reports it is drawing in this theme and in no other.
type bigText struct{ fyne.Theme }

const bigTextSize = 31

func (b bigText) Size(n fyne.ThemeSizeName) float32 {
	if n == theme.SizeNameText {
		return bigTextSize
	}
	return b.Theme.Size(n)
}

// liveHeap is the heap still reachable after two collections, in bytes.
func liveHeap() uint64 {
	runtime.GC()
	runtime.GC()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.HeapAlloc
}

// styledText is text in the four styles a program draws in, so a scope costs
// what it costs in a real window rather than one style's worth.
func styledText() *widget.RichText {
	return widget.NewRichTextFromMarkdown("plain **bold** *italic* `mono`")
}

// heapGrowth is what refreshing o a hundred times, drawing after each, leaves
// on the heap.
func heapGrowth(w fyne.Window, refresh func()) uint64 {
	w.Canvas().Capture()
	before := liveHeap()
	for range 100 {
		refresh()
		w.Canvas().Capture()
	}
	after := liveHeap()
	if after < before {
		return 0
	}
	return after - before
}

func TestAThemeOverrideDrawsItsContentInItsTheme(t *testing.T) {
	test.NewTempApp(t)
	label := widget.NewLabel("x")
	NewThemeOverride(label, bigText{theme.DefaultTheme()})

	require.Equal(t, float32(bigTextSize), label.Theme().Size(theme.SizeNameText))
}

func TestARefreshedThemeOverrideKeepsNoFontsBehind(t *testing.T) {
	a := test.NewTempApp(t)
	w := a.NewWindow("")
	o := NewThemeOverride(styledText(), bigText{theme.DefaultTheme()})
	w.SetContent(container.NewStack(o))
	w.Resize(fyne.NewSize(400, 100))

	// A hundred of Fyne's own refreshes keep about 55 MB; a stray megabyte is
	// the collector, not a scope.
	require.Less(t, heapGrowth(w, o.Refresh), uint64(4<<20))
}

func TestARefreshReachingTheOverrideThroughItsContainerKeepsNoFontsBehind(t *testing.T) {
	a := test.NewTempApp(t)
	w := a.NewWindow("")
	around := container.NewStack(NewThemeOverride(styledText(), bigText{theme.DefaultTheme()}))
	w.SetContent(around)
	w.Resize(fyne.NewSize(400, 100))

	require.Less(t, heapGrowth(w, around.Refresh), uint64(4<<20))
}

func TestAnObjectAddedLaterIsCoveredOnceReapplied(t *testing.T) {
	test.NewTempApp(t)
	box := container.NewVBox(widget.NewLabel("first"))
	o := NewThemeOverride(box, bigText{theme.DefaultTheme()})

	later := widget.NewLabel("later")
	box.Add(later)
	o.Refresh()
	require.NotEqual(t, float32(bigTextSize), later.Theme().Size(theme.SizeNameText),
		"a refresh applies nothing; that is the contract")

	o.Reapply()
	require.Equal(t, float32(bigTextSize), later.Theme().Size(theme.SizeNameText))
}

func TestSetContentDrawsTheNewContentInTheTheme(t *testing.T) {
	test.NewTempApp(t)
	o := NewThemeOverride(widget.NewLabel("old"), bigText{theme.DefaultTheme()})

	next := widget.NewLabel("new")
	o.SetContent(next)

	require.Same(t, next, o.Content())
	require.Equal(t, float32(bigTextSize), next.Theme().Size(theme.SizeNameText))
}

func TestSetThemeRedrawsTheContentInTheNewTheme(t *testing.T) {
	test.NewTempApp(t)
	label := widget.NewLabel("x")
	o := NewThemeOverride(label, theme.DefaultTheme())

	o.SetTheme(bigText{theme.DefaultTheme()})

	require.Equal(t, float32(bigTextSize), label.Theme().Size(theme.SizeNameText))
}

/*
Canary #51 from docs/fyne-quirks.md, not a contract: Fyne's ThemeOverride
makes a theme scope on every Refresh, and the font cache keeps an entry per
scope for the life of the program (fyne-io/fyne#6640). ThemeOverride above
exists to avoid that. When this fails, Fyne has released the fix
(fyne-io/fyne#6642 or its successor) and the pin can move to that tag.
*/
func TestFyneStillKeepsFontsForEveryThemeScope(t *testing.T) {
	a := test.NewTempApp(t)
	w := a.NewWindow("")
	o := container.NewThemeOverride(styledText(), bigText{theme.DefaultTheme()})
	w.SetContent(o)
	w.Resize(fyne.NewSize(400, 100))

	require.Greater(t, heapGrowth(w, o.Refresh), uint64(20<<20))
}
