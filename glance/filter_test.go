package glance_test

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ushineko/fynedesygn/fynetest"
	"github.com/ushineko/fynedesygn/glance"
)

// typeText types each character at the filter.
func typeText(f *glance.Filter, s string) {
	for _, r := range s {
		f.TypedRune(r)
	}
}

func key(name fyne.KeyName) *fyne.KeyEvent { return &fyne.KeyEvent{Name: name} }

// passing is the names a filter lets through.
func passing(f *glance.Filter, names ...string) []string {
	var out []string
	for _, n := range names {
		if f.Matches(n) {
			out = append(out, n)
		}
	}
	return out
}

var processes = []string{"firefox", "firefox-bin", "chrome", "chrome_crashpad", "kded6", "kscreen", "zsh", "Xwayland"}

// Spec 068 R17. The filter's key starts it and is not part of it; what is
// typed after narrows the list as it is typed.
func TestTheFiltersKeyStartsItAndTypingNarrows(t *testing.T) {
	fynetest.App(t)
	f := glance.NewFilter()
	changed := 0
	f.OnChanged = func() { changed++ }

	assert.False(t, f.TypedRune('c'), "a key that is not the filter's started it")
	assert.Equal(t, processes, passing(f, processes...))

	assert.True(t, f.TypedRune(glance.DefaultFilterKey))
	assert.True(t, f.Editing())
	assert.Empty(t, f.Text(), "the key that starts the filter is not part of it")
	assert.Zero(t, changed)

	typeText(f, "ch")
	assert.Equal(t, "ch", f.Text())
	assert.Equal(t, []string{"chrome", "chrome_crashpad"}, passing(f, processes...))
	assert.Equal(t, 2, changed)

	f.TypedRune('f')
	assert.Equal(t, "chf", f.Text(), "the filter's own key, typed in a filter, is a character")
}

// R18. The text is a regular expression matched without regard to case.
func TestAFilterIsARegularExpressionWithoutCase(t *testing.T) {
	fynetest.App(t)
	f := glance.NewFilter()
	f.TypedRune('f')
	typeText(f, "^(k|z)|fox$")

	assert.True(t, f.Valid())
	assert.Equal(t, []string{"firefox", "kded6", "kscreen", "zsh"}, passing(f, processes...))

	f.SetText("XWAY")
	assert.Equal(t, []string{"Xwayland"}, passing(f, processes...))
}

// R18. Text that is not an expression is matched as it is written, and says
// so, so a half-typed one never empties the list.
func TestAHalfTypedExpressionIsMatchedAsWritten(t *testing.T) {
	fynetest.App(t)
	f := glance.NewFilter()
	f.TypedRune('f')
	typeText(f, "fire(")

	assert.False(t, f.Valid())
	assert.Empty(t, passing(f, processes...), "nothing is named fire(")

	f.SetText("[kde")
	assert.False(t, f.Valid())
	assert.Equal(t, []string{"[KDE] helper"}, passing(f, "[KDE] helper", "kded6"))

	f.SetText("fire(fox)?")
	assert.True(t, f.Valid())
	assert.Equal(t, []string{"firefox", "firefox-bin"}, passing(f, processes...))
}

// R17. Backspace takes a character, Return keeps the filter, Escape clears
// it; and the keys are not the filter's when there is no filter.
func TestTheFiltersKeys(t *testing.T) {
	fynetest.App(t)
	f := glance.NewFilter()

	assert.False(t, f.TypedKey(key(fyne.KeyEscape)), "Escape with no filter was taken")
	assert.False(t, f.TypedKey(key(fyne.KeyBackspace)))

	f.TypedRune('f')
	typeText(f, "zsh")
	assert.True(t, f.TypedKey(key(fyne.KeyBackspace)))
	assert.Equal(t, "zs", f.Text())
	assert.False(t, f.TypedKey(key(fyne.KeyDown)), "a key the filter has no use for was taken")

	assert.True(t, f.TypedKey(key(fyne.KeyReturn)))
	assert.False(t, f.Editing())
	assert.Equal(t, "zs", f.Text(), "Return did not keep the filter")
	assert.False(t, f.TypedKey(key(fyne.KeyBackspace)), "Backspace changed a filter that was kept")
	assert.False(t, f.TypedRune('x'), "a character was added to a filter that was kept")

	// The key again edits what is there.
	f.TypedRune('f')
	f.TypedRune('h')
	assert.Equal(t, "zsh", f.Text())

	assert.True(t, f.TypedKey(key(fyne.KeyEscape)))
	assert.False(t, f.Editing())
	assert.Empty(t, f.Text())

	// A kept filter is cleared by Escape too.
	f.SetText("zsh")
	assert.True(t, f.TypedKey(key(fyne.KeyEscape)))
	assert.Empty(t, f.Text())

	// Backspace on an empty filter being typed stays in it.
	f.TypedRune('f')
	assert.True(t, f.TypedKey(key(fyne.KeyBackspace)))
	assert.True(t, f.Editing())
}

// A filter can have another key.
func TestAFilterCanHaveAnotherKey(t *testing.T) {
	fynetest.App(t)
	f := glance.NewFilter()
	f.Key = '/'

	assert.False(t, f.TypedRune('f'))
	assert.True(t, f.TypedRune('/'))
	assert.True(t, strings.HasPrefix(f.Line(), "/"))
}

// R16. The line says where the filter is: a hint, the text and a cursor, the
// filter in force. And it is one height in all three, so nothing under it
// moves.
func TestTheFiltersLineIsAlwaysThereAndOneHeight(t *testing.T) {
	fynetest.App(t)
	f := glance.NewFilter()

	hint := f.Object().MinSize().Height
	assert.Equal(t, "f  filter", f.Line())
	require.Positive(t, hint)

	f.TypedRune('f')
	assert.Equal(t, "/_", f.Line())
	typeText(f, "zsh")
	assert.Equal(t, "/zsh_", f.Line())
	assert.InDelta(t, hint, f.Object().MinSize().Height, 0.01)

	f.TypedKey(key(fyne.KeyReturn))
	assert.Equal(t, "/zsh", f.Line())
	assert.InDelta(t, hint, f.Object().MinSize().Height, 0.01)

	f.Clear()
	assert.Equal(t, "f  filter", f.Line())
}

// A long filter is drawn by its end, where the typing is, so the line does
// not run out of the drawer.
func TestALongFilterIsDrawnByItsEnd(t *testing.T) {
	fynetest.App(t)
	f := glance.NewFilter()
	f.TypedRune('f')
	short := "abcdefghij"
	typeText(f, short)
	width := f.Object().MinSize().Width

	typeText(f, strings.Repeat("0123456789", 6)+"END")
	assert.Equal(t, short+strings.Repeat("0123456789", 6)+"END", f.Text(), "the filter itself was cut")
	assert.True(t, strings.HasSuffix(f.Line(), "END_"))
	assert.Contains(t, f.Line(), glance.Ellipsis)
	assert.Less(t, f.Object().MinSize().Width, 4*width)
}

// R17. In a drawer, the window's keys reach the filter while the drawer is
// open and not while it is closed.
func TestKeysReachTheFilterOnlyWhileTheDrawerIsOpen(t *testing.T) {
	w, _, _ := drawerWindow(t, glance.Options{Title: "drawer"})
	d, f := w.Panel().Drawer(), glance.NewFilter()
	d.SetContent(f)
	d.SetFilter(f)
	cv := w.Window().Canvas()
	require.NotNil(t, cv.OnTypedRune())
	require.NotNil(t, cv.OnTypedKey())

	cv.OnTypedRune()('f')
	assert.False(t, f.Editing(), "a key reached the filter of a closed drawer")

	d.SetOpen(true)
	cv.OnTypedRune()('f')
	cv.OnTypedRune()('z')
	assert.Equal(t, "z", f.Text())
	cv.OnTypedKey()(key(fyne.KeyBackspace))
	assert.Empty(t, f.Text())
}

// R20. Return and Escape give the keyboard back; typing and Backspace do
// not. Closing the drawer ends the typing and keeps the text.
func TestEndingAFilterGivesTheKeyboardBack(t *testing.T) {
	w, _, _ := drawerWindow(t, glance.Options{Title: "drawer"})
	d, f := w.Panel().Drawer(), glance.NewFilter()
	d.SetContent(f)
	d.SetFilter(f)
	yields := 0
	d.SetYield(func() { yields++ })
	d.SetOpen(true)

	f.TypedRune('f')
	typeText(f, "zs")
	f.TypedKey(key(fyne.KeyBackspace))
	assert.Zero(t, yields)

	f.TypedKey(key(fyne.KeyReturn))
	assert.Equal(t, 1, yields)

	f.TypedRune('f')
	f.TypedKey(key(fyne.KeyEscape))
	assert.Equal(t, 2, yields)

	f.TypedRune('f')
	typeText(f, "kde")
	d.SetOpen(false)
	assert.Equal(t, 3, yields)
	assert.False(t, f.Editing(), "a closed drawer left its filter being typed")
	assert.Equal(t, "kde", f.Text())
}

// A filter set before the panel has a window is wired when it gets one.
func TestAFilterSetBeforeTheWindowIsWiredWhenThereIsOne(t *testing.T) {
	a := fynetest.App(t)
	p := glance.NewPanel(0)
	f := glance.NewFilter()
	p.Drawer().SetContent(f)
	p.Drawer().SetFilter(f)

	w := a.NewWindow("late")
	p.Attach(w)
	p.Drawer().SetOpen(true)
	require.NotNil(t, w.Canvas().OnTypedRune())
	w.Canvas().OnTypedRune()('f')
	assert.True(t, f.Editing())
}
