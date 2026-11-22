package glance_test

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"
	fynetheme "fyne.io/fyne/v2/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ushineko/fynedesygn/fynetest"
	"github.com/ushineko/fynedesygn/glance"
)

// clockWindow is a panel holding a card whose heading is a clock, and a card
// with a row under it, at a width wider than either needs.
func clockWindow(t *testing.T) (*glance.Window, *glance.Card, *glance.Card) {
	t.Helper()
	w := newTestWindow(t, glance.Options{Title: "clock", MinWidth: 400})
	w.Window().SetPadded(false)
	clock := glance.NewCard("18:01:18")
	clock.SetIcon(fynetheme.HistoryIcon())
	clock.SetTitleEnd("Sat 2026/11/21", fynetheme.CalendarIcon())
	other := glance.NewCard("Peripherals")
	other.AddRow(named("mouse", "G502"))
	w.Panel().Add(clock, other)
	clock.SetAvailable(true)
	other.SetAvailable(true)
	w.Window().Show()
	w.Panel().Resize()
	return w, clock, other
}

// text finds a drawn text in a card by what it says.
func text(t *testing.T, c *glance.Card, s string) *canvas.Text {
	t.Helper()
	for _, o := range test.LaidOutObjects(c.Object()) {
		if x, ok := o.(*canvas.Text); ok && x.Text == s && x.Visible() {
			return x
		}
	}
	require.Failf(t, "no text", "%q is not drawn", s)
	return nil
}

// Spec 086 R9. The heading's second title is at its right end, its icon
// before it, across from the title at the left: a clock's date across from
// its time.
func TestATitleAtTheHeadingsRightEnd(t *testing.T) {
	_, clock, other := clockWindow(t)
	drv := fyne.CurrentApp().Driver()
	x := func(o fyne.CanvasObject) float32 { return drv.AbsolutePositionForObject(o).X }

	title, end := text(t, clock, "18:01:18"), text(t, clock, "Sat 2026/11/21")
	rowValueEnd := x(other.Rows()[0].Object()) + other.Rows()[0].Object().Size().Width
	assert.InDelta(t, rowValueEnd, x(end)+end.Size().Width, 0.5,
		"the date does not end where the other card's figures do")

	var icons []*canvas.Image
	for _, o := range test.LaidOutObjects(clock.Object()) {
		if img, ok := o.(*canvas.Image); ok && img.Visible() {
			icons = append(icons, img)
		}
	}
	require.Len(t, icons, 2)
	assert.InDelta(t, x(text(t, other, "Peripherals")), x(icons[0]), 0.01,
		"the clock does not start where the other card's title does")
	assert.Less(t, x(icons[0]), x(title), "the time's icon is not before it")
	assert.Less(t, x(icons[1]), x(end), "the date's icon is not before it")
	assert.Greater(t, x(icons[1]), x(title)+title.Size().Width, "the date's icon is not at the right end")
	assert.Equal(t, "Sat 2026/11/21", clock.TitleEnd())

	clock.SetTitleEnd("", nil)
	assert.False(t, end.Visible())
	assert.False(t, icons[1].Visible())
}

// Spec 086 R10. A title that changes is held at the widest it has been, so a
// narrower time does not narrow the card.
func TestATitleThatChangesNeverNarrowsItsCard(t *testing.T) {
	_ = fynetest.App(t)
	c := glance.NewCard("WWWWWWWW")
	c.SetTitleEnd("WWWW", nil)
	wide := c.Object().MinSize().Width

	c.SetTitle("11:11:11")
	c.SetTitleEnd("i", nil)
	assert.InDelta(t, wide, c.Object().MinSize().Width, 0.01, "a narrower title narrowed the card")
	assert.Equal(t, "11:11:11", c.Title())

	c.SetTitle("WWWWWWWWWWWW")
	assert.Greater(t, c.Object().MinSize().Width, wide, "a wider title was clipped")
}

// Spec 086 R11. Drawn as one line per reading, a card keeps its heading when
// the heading is all it has, and leaves it out when it has rows.
func TestACardThatIsOnlyItsHeadingKeepsItInLines(t *testing.T) {
	w, clock, other := clockWindow(t)
	w.Panel().SetArrangement(glance.Lines)
	assert.True(t, visibleInTree(clock.Object(), text(t, clock, "18:01:18")),
		"a card that is only its heading lost it in Lines")
	assert.Greater(t, clock.Object().MinSize().Height, glance.CardPadTop()+glance.CardPadBottom()+1)
	for _, o := range test.LaidOutObjects(other.Object()) {
		if x, ok := o.(*canvas.Text); ok && x.Text == "Peripherals" {
			assert.False(t, visibleInTree(other.Object(), x), "a card with rows kept its heading in Lines")
		}
	}
}

// visibleInTree reports whether o and every container above it in root are
// shown.
func visibleInTree(root fyne.CanvasObject, o fyne.CanvasObject) bool {
	var walk func(n fyne.CanvasObject) (found, visible bool)
	walk = func(n fyne.CanvasObject) (bool, bool) {
		if n == o {
			return true, n.Visible()
		}
		if c, ok := n.(*fyne.Container); ok {
			for _, ch := range c.Objects {
				if f, v := walk(ch); f {
					return true, v && n.Visible()
				}
			}
		}
		return false, false
	}
	_, v := walk(root)
	return v
}
