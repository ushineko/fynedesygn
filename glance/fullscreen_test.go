package glance

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// A 3840x2160 monitor whose taskbar takes the bottom 48 px when it shows.
var screen = edges{0, 0, 3840, 2160}

// inFront is a full-screen window on the glance window's monitor, which each
// case changes one thing about.
func inFront(r edges) foreground {
	return foreground{Rect: r, Monitor: screen, SameMonitor: true}
}

func TestAFullScreenWindowIsOneThatCoversItsMonitor(t *testing.T) {
	maximised := inFront(edges{-8, -8, 3848, 2120}) // work area plus borders, taskbar showing
	maximised.Maximised, maximised.Captioned = true, true

	autohide := inFront(edges{-8, -8, 3848, 2168}) // taskbar hidden: past every edge
	autohide.Maximised, autohide.Captioned = true, true

	f11 := inFront(screen) // a browser after F11: maximised, no titlebar
	f11.Maximised = true

	cases := []struct {
		name string
		f    foreground
		want bool
	}{
		{"exactly the monitor", inFront(screen), true},
		// Terraria in "windowed full screen", measured: WS_POPUP, not maximised,
		// window, client and DWM frame all 0,0-3840,2160, the taskbar's work
		// area 0,0-3840,2088.
		{"a borderless game window exactly the monitor's size, not maximised", terraria(), true},
		{"8 px larger on every side", inFront(edges{-8, -8, 3848, 2168}), true},
		{"1 px short on the right", inFront(edges{0, 0, 3839, 2160}), false},
		{"1 px short at the bottom", inFront(edges{0, 0, 3840, 2159}), false},
		{"maximised, taskbar showing", maximised, false},
		{"maximised with a titlebar, taskbar hidden", autohide, false},
		{"maximised without a titlebar (F11)", f11, true},
		{"in the middle of the screen", inFront(edges{800, 400, 2400, 1400}), false},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, fullscreenInFront(c.f), c.name)
	}
}

// DWM's frame leaves out the invisible borders GetWindowRect counts: either
// one covering the monitor is full screen, and neither covering it is not.
func TestEitherRectangleCoveringTheMonitorIsFullScreen(t *testing.T) {
	f := inFront(edges{0, 0, 3839, 2160})
	f.Frame = screen
	assert.True(t, fullscreenInFront(f), "the frame covers it")

	f = inFront(screen)
	f.Frame = edges{7, 0, 3833, 2153}
	assert.True(t, fullscreenInFront(f), "the window rectangle covers it")

	f = inFront(edges{0, 0, 3839, 2160})
	f.Frame = edges{7, 0, 3833, 2153}
	assert.False(t, fullscreenInFront(f), "neither covers it")
}

func TestWhatIsNotAReasonToHide(t *testing.T) {
	cases := map[string]func(*foreground){
		"this program's own window":    func(f *foreground) { f.Own = true },
		"the desktop or the taskbar":   func(f *foreground) { f.Shell = true },
		"hidden, minimised or cloaked": func(f *foreground) { f.Hidden = true },
		"on another monitor":           func(f *foreground) { f.SameMonitor = false },
	}
	for name, change := range cases {
		f := inFront(screen)
		change(&f)
		assert.False(t, fullscreenInFront(f), name)
	}
}

// Two looks in a row decide; a look that disagrees starts the count again, so
// a window passing through full screen does not flicker the panel.
func TestTheWindowHidesAndShowsOnlyOnTwoLooksInARow(t *testing.T) {
	var s settle
	step := func(fs bool) (bool, bool) { return s.next(fs) }

	hidden, changed := step(true)
	assert.False(t, hidden)
	assert.False(t, changed, "one look does not hide")
	hidden, changed = step(true)
	assert.True(t, hidden)
	assert.True(t, changed, "two looks hide")
	hidden, changed = step(true)
	assert.True(t, hidden)
	assert.False(t, changed, "already hidden")

	_, changed = step(false)
	assert.False(t, changed, "one look does not show")
	_, changed = step(true)
	assert.False(t, changed, "a look back in full screen starts the count again")
	_, changed = step(false)
	assert.False(t, changed)
	hidden, changed = step(false)
	assert.False(t, hidden)
	assert.True(t, changed, "two looks show")
}

// terraria is a borderless game in windowed full screen, as measured.
func terraria() foreground {
	f := inFront(screen)
	f.Frame = screen
	return f
}
