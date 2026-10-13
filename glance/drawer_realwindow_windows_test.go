package glance_test

import (
	"os"
	"testing"
	"time"
	"unsafe"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	rwSetWindowPos = rwUser32.NewProc("SetWindowPos")
	rwPostMessage  = rwUser32.NewProc("PostMessageW")
)

// settle is how long the window is given to act on a click or a key.
const settle = 600 * time.Millisecond

// realWindow starts the example and gives it time to draw every card: the
// last arrives after twelve polls of half a second.
func realWindow(t *testing.T, args ...string) uintptr {
	t.Helper()
	_, _, _ = rwSetDpiAwareness.Call(^uintptr(3)) // DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2 (-4)
	hwnd := runGlanceMonitor(t, args...)
	time.Sleep(8 * time.Second)
	return hwnd
}

// keepPointer puts the pointer back where it was when the test ends.
func keepPointer(t *testing.T) {
	t.Helper()
	var saved struct{ X, Y int32 }
	_, _, _ = rwGetCursorPos.Call(uintptr(unsafe.Pointer(&saved)))
	t.Cleanup(func() { _, _, _ = rwSetCursorPos.Call(uintptr(saved.X), uintptr(saved.Y)) })
}

// place moves the window's top-left to (x, y) and leaves its size.
func place(hwnd uintptr, x, y int32) {
	const noSize, noZOrder, noActivate = 0x0001, 0x0004, 0x0010
	_, _, _ = rwSetWindowPos.Call(hwnd, 0, uintptr(x), uintptr(y), 0, 0, noSize|noZOrder|noActivate)
	time.Sleep(300 * time.Millisecond)
}

// click presses and releases the real pointer at a point of the screen,
// without moving between the two.
func click(x, y int32) {
	const leftDown, leftUp = 0x0002, 0x0004
	_, _, _ = rwSetCursorPos.Call(uintptr(x), uintptr(y))
	time.Sleep(150 * time.Millisecond)
	_, _, _ = rwMouseEvent.Call(leftDown, 0, 0, 0, 0)
	time.Sleep(60 * time.Millisecond)
	_, _, _ = rwMouseEvent.Call(leftUp, 0, 0, 0, 0)
	time.Sleep(settle)
}

// bandwidthCard is a point on the second card of the example, which is the
// Bandwidth card: a third of the way down the cards, 60 px in from their
// left edge. cards is the cards' rectangle on the screen.
func bandwidthCard(cards rwRect) (x, y int32) {
	return cards.L + 60, cards.T + (cards.B-cards.T)*30/100
}

// typeKeys posts characters to the window, as its own keyboard input.
func typeKeys(hwnd uintptr, s string) {
	const wmChar = 0x0102
	for _, r := range s {
		_, _, _ = rwPostMessage.Call(hwnd, wmChar, uintptr(r), 1)
		time.Sleep(60 * time.Millisecond)
	}
	time.Sleep(settle)
}

// pressKey posts a key by its virtual key and scan code.
func pressKey(hwnd, vk, scan uintptr) {
	const wmKeyDown, wmKeyUp = 0x0100, 0x0101
	_, _, _ = rwPostMessage.Call(hwnd, wmKeyDown, vk, 1|scan<<16)
	time.Sleep(40 * time.Millisecond)
	_, _, _ = rwPostMessage.Call(hwnd, wmKeyUp, vk, 1|scan<<16|0xC0000000)
	time.Sleep(settle)
}

func foreground() uintptr {
	h, _, _ := rwGetForegroundWindow.Call()
	return h
}

/*
Spec 068 R4, R8, R9, on Windows. A click on a card opens the drawer beside the
cards and a second click closes it, and the cards do not move on the screen:
on the right the window's left edge stays, on the left its right edge does,
and either way it comes back to the rectangle it had.

With the side left to the room, a window in the middle of the screen opens to
the right and one near the right edge of the work area opens to the left.

The clicks are real ones, so this also says that a press which does not move
is a click and not the start of a move (R12).
*/
func TestADrawerOpensBesideTheCardsOnWindows(t *testing.T) {
	if os.Getenv(realWindowEnv) == "" {
		t.Skip("moves the real pointer; set " + realWindowEnv + "=1 to run it")
	}
	keepPointer(t)

	for _, c := range []struct {
		name     string
		args     []string
		atEdge   bool
		wantLeft bool
	}{
		{"named right", []string{"-drawer", "right"}, false, false},
		{"named left", []string{"-drawer", "left"}, false, true},
		{"by room, mid-screen", nil, false, false},
		{"by room, near the right edge", nil, true, true},
		{"named right, near the right edge", []string{"-drawer", "right"}, true, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			hwnd := realWindow(t, c.args...)
			if c.atEdge {
				work, r := workArea(hwnd), windowRect(hwnd)
				place(hwnd, work.R-(r.R-r.L)-40, r.T)
			}
			closed := windowRect(hwnd)
			x, y := bandwidthCard(closed)

			click(x, y)
			open := windowRect(hwnd)
			t.Logf("closed %+v, open %+v", closed, open)
			require.Greater(t, open.R-open.L, (closed.R-closed.L)*3/2, "the click did not open the drawer")
			assert.Equal(t, closed.T, open.T)
			assert.Equal(t, closed.B, open.B, "the drawer changed the window's height")
			if c.wantLeft {
				assert.Equal(t, closed.R, open.R, "the cards moved: the right edge did not stay")
				assert.Less(t, open.L, closed.L)
			} else {
				assert.Equal(t, closed.L, open.L, "the cards moved: the left edge did not stay")
				assert.Greater(t, open.R, closed.R)
			}

			// The pointer has not moved and the cards are where they were.
			click(x, y)
			assert.Equal(t, closed, windowRect(hwnd), "a second click did not put the window back")
		})
	}
}

/*
R12, R13. A drag still moves the window, and is not a click: the drawer is
closed after it. A click after the drag is a click.
*/
func TestADragMovesTheWindowAndIsNotAClick(t *testing.T) {
	if os.Getenv(realWindowEnv) == "" {
		t.Skip("moves the real pointer; set " + realWindowEnv + "=1 to run it")
	}
	keepPointer(t)
	hwnd := realWindow(t, "-drawer", "right")

	start := windowRect(hwnd)
	x, y := start.L-180, start.T-120
	drag(t, hwnd, x, y)
	moved := windowRect(hwnd)
	require.Equal(t, x, moved.L, "the drag did not move the window")
	require.Equal(t, y, moved.T)
	assert.Equal(t, start.R-start.L, moved.R-moved.L, "the drag was taken for a click and opened the drawer")

	cx, cy := bandwidthCard(moved)
	click(cx, cy)
	open := windowRect(hwnd)
	assert.Greater(t, open.R-open.L, (moved.R-moved.L)*3/2, "a click after a drag was not a click")
}

/*
R20, R21. The click that opens the drawer makes the panel the foreground
window, and the keyboard goes back to the window that had it: when the drawer
closes, when Return keeps a filter and when Escape clears one.

The window that had it is a second copy of the example, clicked first so that
it is the foreground window by the person's own hand.
*/
func TestTheKeyboardGoesBackToTheWindowThatHadIt(t *testing.T) {
	if os.Getenv(realWindowEnv) == "" {
		t.Skip("moves the real pointer; set " + realWindowEnv + "=1 to run it")
	}
	keepPointer(t)
	other := realWindow(t, "-drawer", "right")
	work := workArea(other)
	place(other, work.L+80, work.T+80)
	panel := runGlanceMonitor(t, "-drawer", "right")
	time.Sleep(8 * time.Second)

	otherRect, panelRect := windowRect(other), windowRect(panel)
	require.Less(t, otherRect.R+700, panelRect.L, "the two windows would overlap with their drawers out")

	// On the other window's heading, which is a click on a card and so
	// opens or closes its drawer. Closing gives the keyboard away, as it
	// should, so a click that closed it is followed by one that opens it:
	// all that matters here is that the other window ends with the keyboard.
	activate := func() {
		click(otherRect.L+60, otherRect.T+12)
		if foreground() != other {
			click(otherRect.L+60, otherRect.T+12)
		}
		require.Equal(t, other, foreground(), "the other window did not take the foreground")
	}
	x, y := bandwidthCard(panelRect)

	activate()
	click(x, y)
	require.Equal(t, panel, foreground(), "the click did not give the panel the keyboard")
	click(x, y)
	assert.Equal(t, panelRect, windowRect(panel))
	assert.Equal(t, other, foreground(), "closing the drawer did not give the keyboard back")

	activate()
	click(x, y)
	typeKeys(panel, "fzsh")
	assert.Equal(t, panel, foreground(), "typing a filter gave the keyboard away")
	pressKey(panel, 0x0D, 0x1C) // Return
	assert.Equal(t, other, foreground(), "Return did not give the keyboard back")

	click(x+4, y) // on the same card: closes
	activate()
	click(x, y)
	typeKeys(panel, "f")
	pressKey(panel, 0x1B, 0x01) // Escape
	assert.Equal(t, other, foreground(), "Escape did not give the keyboard back")
}

// R16, R17 on a real window: a filter typed, edited and cleared leaves the
// window the size it was.
func TestAFilterTypedDoesNotResizeTheWindow(t *testing.T) {
	if os.Getenv(realWindowEnv) == "" {
		t.Skip("moves the real pointer; set " + realWindowEnv + "=1 to run it")
	}
	keepPointer(t)
	hwnd := realWindow(t, "-drawer", "right")
	closed := windowRect(hwnd)
	x, y := bandwidthCard(closed)
	click(x, y)
	open := windowRect(hwnd)
	require.Greater(t, open.R-open.L, closed.R-closed.L)

	typeKeys(hwnd, "f^(k|z)|fox$")
	assert.Equal(t, open, windowRect(hwnd))
	pressKey(hwnd, 0x08, 0x0E) // Backspace
	assert.Equal(t, open, windowRect(hwnd))
	pressKey(hwnd, 0x1B, 0x01) // Escape
	assert.Equal(t, open, windowRect(hwnd))
}
