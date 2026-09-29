package glance_test

import (
	"testing"

	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
	fd "github.com/ushineko/fynedesygn"
	"github.com/ushineko/fynedesygn/glance"
)

// carded is a card with one row, ready to be dimmed.
func carded(t *testing.T) (*glance.Card, *glance.Row) {
	t.Helper()
	test.NewTempApp(t)
	c := glance.NewCard("Peripherals")
	r := glance.NewRow("G502 X PLUS", "--")
	r.Set(glance.Known("76 %", fd.StatusGood))
	c.AddRow(r)
	return c, r
}

// AC. Last-known dims the rows, because the values are as provisional as a
// stale card's.
func TestLastKnownDimsTheRows(t *testing.T) {
	c, r := carded(t)
	c.SetLastKnown(true)

	assert.True(t, r.Reading().Stale, "a last-known value was drawn as a current one")
	assert.True(t, c.Stale())
}

// AC. And does not say the source is unavailable, because nothing is: the
// panel has not asked yet.
func TestLastKnownDoesNotMarkTheHeader(t *testing.T) {
	c, _ := carded(t)
	c.SetLastKnown(true)

	assert.False(t, marked(t, c), "a card that had only just started called itself unavailable")
}

// AC. A source that was answering and has stopped still says so.
func TestStaleStillMarksTheHeader(t *testing.T) {
	c, _ := carded(t)
	c.SetStale(true)

	assert.True(t, marked(t, c))
}

// AC. And the marker clears when a live reading lands, from either state.
func TestTheMarkerClearsOnRecovery(t *testing.T) {
	for _, from := range []struct {
		name string
		set  func(*glance.Card)
	}{
		{"stale", func(c *glance.Card) { c.SetStale(true) }},
		{"last known", func(c *glance.Card) { c.SetLastKnown(true) }},
	} {
		t.Run(from.name, func(t *testing.T) {
			c, r := carded(t)
			from.set(c)
			c.SetStale(false)

			assert.False(t, marked(t, c))
			assert.False(t, r.Reading().Stale)
		})
	}
}

// AC. Going from last-known to stale marks the header: the dimming is
// already on, and the marker must not be skipped because of it.
func TestLastKnownThenStaleStillMarks(t *testing.T) {
	c, _ := carded(t)
	c.SetLastKnown(true)
	c.SetStale(true)

	assert.True(t, marked(t, c), "the marker was skipped because the card was already dim")
}

// marked reports whether the card's header shows the gone marker.
func marked(t *testing.T, c *glance.Card) bool {
	t.Helper()
	for _, o := range test.LaidOutObjects(c.Object()) {
		txt, ok := o.(*canvas.Text)
		if ok && txt.Text == glance.GoneMarker {
			return txt.Visible()
		}
	}
	return false
}
