package glance

import (
	"syscall"
	"unsafe"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver"
)

var (
	user32         = syscall.NewLazyDLL("user32.dll")
	releaseCapture = user32.NewProc("ReleaseCapture")
	sendMessage    = user32.NewProc("SendMessageW")
	postMessage    = user32.NewProc("PostMessageW")
	getCursorPos   = user32.NewProc("GetCursorPos")
	screenToClient = user32.NewProc("ScreenToClient")
)

const (
	wmLButtonUp     = 0x0202
	wmNCLButtonDown = 0x00A1
	htCaption       = 2
)

// beginMove moves the window with the drag that has just begun, as a drag on
// a titlebar would.
//
// It runs on the first movement of a press and not on the press, so a press
// that does not move is a click (spec 068). Windows has no Alt-drag and no
// window rule to fall back on: a glance window there, with no titlebar,
// opened in the middle of the screen, on top of everything, and stayed there
// (spec 052).
func beginMove(win fyne.Window) { startMove(win) }

/*
startMove hands the drag to Windows as a press on the caption, which is how a
window with no titlebar is dragged there. SendMessage runs the system's move
loop and returns when the button is released.

Two things around it are not obvious. GLFW captured the mouse on the press,
and a captured mouse keeps the move loop from ever starting, so the capture is
released first. And the loop eats the release, so GLFW -- and Fyne behind it
-- would believe the button is still down and read the next plain move as a
drag; the release is posted back to the window after the loop.
*/
func startMove(win fyne.Window) {
	native, ok := win.(driver.NativeWindow)
	if !ok {
		return
	}
	native.RunNative(func(ctx any) {
		wc, ok := ctx.(driver.WindowsWindowContext)
		if !ok || wc.HWND == 0 {
			return
		}
		// The loop sends WM_MOVING to the window; snapEdges is what answers
		// it (spec 058).
		snapEdges(wc.HWND)
		_, _, _ = releaseCapture.Call()
		_, _, _ = sendMessage.Call(wc.HWND, wmNCLButtonDown, htCaption, 0)

		var pt struct{ X, Y int32 }
		_, _, _ = getCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
		_, _, _ = screenToClient.Call(wc.HWND, uintptr(unsafe.Pointer(&pt)))
		lparam := uintptr(uint16(pt.X)) | uintptr(uint16(pt.Y))<<16
		_, _, _ = postMessage.Call(wc.HWND, wmLButtonUp, 0, lparam)
	})
}
