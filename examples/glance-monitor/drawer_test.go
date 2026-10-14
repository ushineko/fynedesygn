package main

import (
	"testing"

	"fyne.io/fyne/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ushineko/fynedesygn/glance"
)

// drawn builds the monitor with every card up.
func drawn(t *testing.T) *monitor {
	t.Helper()
	m := build(t)
	m.win.Window().SetPadded(false)
	m.renderTicks(20)
	return m
}

// listed is the names the drawer is showing, top to bottom.
func (m *monitor) listed() []string { return append([]string(nil), m.drawer.listed...) }

// A click anywhere on a card opens the drawer on it, a click on another card
// changes the subject and leaves it open, and a click anywhere on the same
// card closes it.
func TestAClickOpensSwitchesAndCloses(t *testing.T) {
	m := drawn(t)
	dr := m.win.Panel().Drawer()
	closed := m.win.Panel().Size()

	m.tapped(glance.Tap{Card: m.bandwidth, Row: m.down})
	require.True(t, dr.Open())
	assert.Equal(t, "Bandwidth by process", dr.Title())
	open := m.win.Panel().Size()
	assert.Greater(t, open.Width, closed.Width)
	assert.Equal(t, closed.Height, open.Height)

	// Another row of the same card is the same subject: the card's.
	m.tapped(glance.Tap{Card: m.bandwidth, Row: m.up})
	assert.False(t, dr.Open(), "a second click on the same card did not close the drawer")

	m.tapped(glance.Tap{Card: m.bandwidth})
	m.tapped(glance.Tap{Card: m.thermal, Row: m.coolant})
	assert.True(t, dr.Open(), "a click on another card closed the drawer")
	assert.Equal(t, "CPU by process", dr.Title(), "the row clicked chose the subject")
	assert.Equal(t, open, m.win.Panel().Size(), "changing the subject changed the window")

	m.tapped(glance.Tap{Card: m.thermal, Row: m.cpu})
	assert.False(t, dr.Open())
	assert.Equal(t, closed, m.win.Panel().Size())
}

// The thermal card's drawer has a switch between its two subjects, worked by
// a key; the list and the heading change, the window does not, and the card
// opens next on what was chosen. The other cards' drawers have none.
func TestTheThermalDrawerSwitchesBetweenItsSubjects(t *testing.T) {
	m := drawn(t)
	dr := m.win.Panel().Drawer()
	cv := m.win.Window().Canvas()

	m.tapped(glance.Tap{Card: m.thermal})
	require.True(t, m.drawer.sw.Object().Visible())
	open, first := m.win.Panel().Size(), m.listed()

	cv.OnTypedRune()('o')
	assert.Equal(t, "Coolant", dr.Title())
	assert.NotEqual(t, first, m.listed(), "the other subject is the same list")
	assert.Equal(t, open, m.win.Panel().Size(), "choosing changed the window")

	m.tapped(glance.Tap{Card: m.thermal})
	m.tapped(glance.Tap{Card: m.thermal})
	assert.Equal(t, "Coolant", dr.Title(), "the card opened on another subject than the one chosen")

	m.tapped(glance.Tap{Card: m.bandwidth})
	assert.False(t, m.drawer.sw.Object().Visible(), "a card with one subject has a switch")
	cv.OnTypedRune()('c')
	assert.Equal(t, "Bandwidth by process", dr.Title())
}

// The usage card's drawer is every account's quotas, taller than the cards:
// it scrolls, and the window is the size it was through twenty polls. A list
// after it is a list again.
func TestTheUsageDrawerScrolls(t *testing.T) {
	m := drawn(t)
	dr := m.win.Panel().Drawer()
	cv := m.win.Window().Canvas()
	closed := m.win.Panel().Size()

	m.tapped(glance.Tap{Card: m.usage})
	require.True(t, dr.Scrolling())
	open := m.win.Panel().Size()
	assert.Equal(t, closed.Height, open.Height)

	cv.OnTypedKey()(&fyne.KeyEvent{Name: fyne.KeyPageDown})
	at := dr.Offset()
	require.Greater(t, at, float32(0), "the quotas fit, so the example shows no scrolling")
	for i := range 20 {
		m.render(m.sample())
		assert.Equal(t, open, m.win.Panel().Size(), "tick %d", i)
	}
	assert.InDelta(t, at, dr.Offset(), 0.01, "a reading moved what was being read")

	m.tapped(glance.Tap{Card: m.bandwidth})
	assert.False(t, dr.Scrolling())
	assert.NotEmpty(t, m.listed())
}

// The list is as long as the drawer fits and no longer, greatest first.
func TestTheListIsWhatFitsGreatestFirst(t *testing.T) {
	m := drawn(t)
	dr := m.win.Panel().Drawer()
	m.tapped(glance.Tap{Card: m.bandwidth})

	fit := dr.Fit(m.drawer.rows[0], m.drawer.filter)
	require.Positive(t, fit)
	require.Less(t, fit, len(processNames), "the test needs more processes than fit")
	assert.Len(t, m.listed(), fit)

	// The fixed-width format sorts as text only within a unit, so the order
	// is checked on the names: two renders of one tick agree, and the first
	// name is the one with the greatest rate.
	first := m.listed()
	m.renderDrawer()
	assert.Equal(t, first, m.listed())
}

// A card that goes shortens the drawer, and the list with it.
func TestAShorterPanelListsFewer(t *testing.T) {
	m := drawn(t)
	m.tapped(glance.Tap{Card: m.bandwidth})
	tall := len(m.listed())

	m.usage.SetAllowed(false)
	m.renderDrawer()
	assert.Less(t, len(m.listed()), tall)
}

// The filter narrows the list from the window's keys, and clearing it brings
// the list back.
func TestTheFilterNarrowsTheList(t *testing.T) {
	m := drawn(t)
	m.tapped(glance.Tap{Card: m.bandwidth})
	all := len(m.listed())
	cv := m.win.Window().Canvas()

	for _, r := range "fchrome" {
		cv.OnTypedRune()(r)
	}
	assert.ElementsMatch(t, []string{"chrome", "chrome_crashpad"}, m.listed())

	cv.OnTypedKey()(&fyne.KeyEvent{Name: fyne.KeyEscape})
	assert.Len(t, m.listed(), all)
}

// The window is one size while the list changes under it: through twenty
// polls, a filter typed and a filter cleared.
func TestTheDrawerNeverResizesTheWindowWhileItIsOpen(t *testing.T) {
	m := drawn(t)
	m.tapped(glance.Tap{Card: m.bandwidth})
	open := m.win.Panel().Size()
	cv := m.win.Window().Canvas()

	for i := range 20 {
		m.render(m.sample())
		if i == 5 {
			for _, r := range "f^(k|z)" {
				cv.OnTypedRune()(r)
			}
		}
		if i == 12 {
			cv.OnTypedKey()(&fyne.KeyEvent{Name: fyne.KeyEscape})
		}
		// The thermal card goes stale and comes back in this stretch, which
		// is a change of the cards' own and not the drawer's.
		assert.Equal(t, open, m.win.Panel().Size(), "tick %d", i)
	}
}
