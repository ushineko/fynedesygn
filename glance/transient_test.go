package glance

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ushineko/fynedesygn/fynetest"
)

func transientWindow(t *testing.T, hold time.Duration) (*Transient, int) {
	t.Helper()
	a := fynetest.App(t)
	w := NewWindow(a, Options{Title: "test-indicator", Secondary: true})
	card := NewCard("Volume")
	card.AddRow(NewRow("Level", "--"))
	w.Panel().Add(card)
	shows := 0
	tr := NewTransient(w, hold)
	tr.OnShow = func() { shows++ }
	return tr, shows
}

// TestAnIndicatorGoesAwayOnItsOwn: after the hold with no further change,
// the window is hidden without anyone touching it.
func TestAnIndicatorGoesAwayOnItsOwn(t *testing.T) {
	tr, _ := transientWindow(t, 40*time.Millisecond)
	tr.Show()
	require.True(t, tr.Shown())
	require.Eventually(t, func() bool { return !tr.Shown() }, time.Second, 5*time.Millisecond)
}

// TestChangesCloseTogetherKeepTheIndicatorUp: each Show restarts the hold,
// and OnShow runs once for the whole run, not once per change.
func TestChangesCloseTogetherKeepTheIndicatorUp(t *testing.T) {
	// The hold is long against the spacing, so no timer can fire while the
	// loop runs: under the test driver a firing timer hides the window on
	// its own goroutine, and a test that raced it would be testing luck.
	tr, _ := transientWindow(t, 400*time.Millisecond)
	shows := 0
	tr.OnShow = func() { shows++ }
	for range 5 {
		tr.Show()
		time.Sleep(10 * time.Millisecond)
	}
	require.True(t, tr.Shown(), "the indicator blinked between changes")
	require.Equal(t, 1, shows)
	require.Eventually(t, func() bool { return !tr.Shown() }, time.Second, 5*time.Millisecond)
	tr.Show()
	require.Equal(t, 2, shows, "a new run after a hide did not run OnShow")
}

// TestHideStopsTheTimer: a Hide followed by a Show must not be undone by
// the earlier run's timer.
func TestHideStopsTheTimer(t *testing.T) {
	tr, _ := transientWindow(t, 100*time.Millisecond)
	tr.Show()
	tr.Hide()
	require.False(t, tr.Shown())
	tr.Show()
	time.Sleep(30 * time.Millisecond)
	require.True(t, tr.Shown(), "the first run's timer hid the second run")
	tr.Hide()
	tr.Hide()
}

// TestASecondaryWindowIsNotTheMaster: closing an indicator must not end the
// program that owns a main window.
func TestASecondaryWindowIsNotTheMaster(t *testing.T) {
	a := fynetest.App(t)
	main := NewWindow(a, Options{Title: "main"})
	ind := NewWindow(a, Options{Title: "indicator", Secondary: true})
	require.NotNil(t, main)
	require.NotNil(t, ind)
	ind.Window().Close()
}
