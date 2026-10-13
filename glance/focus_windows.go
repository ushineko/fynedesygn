package glance

import (
	"unsafe"

	"fyne.io/fyne/v2"
)

var (
	setForegroundWindow = user32.NewProc("SetForegroundWindow")
	getWindowNext       = user32.NewProc("GetWindow")
	getWindowTextLength = user32.NewProc("GetWindowTextLengthW")
)

const (
	wmMouseActivate = 0x0021
	gwHwndNext      = 2
	wsExToolWindow  = 0x00000080
	wsExNoActivate  = 0x08000000
)

// gwlExStyle is GWL_EXSTYLE, -20: a variable for the reason gwlpWndProc is.
var gwlExStyle = -20

// before holds, for each window wrapped, the window that was in the
// foreground when a click last activated it. Guarded by snapMu.
var before = map[uintptr]uintptr{}

// hookWindow wraps the window's procedure as soon as there is a window to
// wrap, so the click that gives the panel the keyboard is seen. The wrapper
// is the one a drag installs (snapEdges); a panel that is clicked before it
// is ever dragged needs it sooner.
func hookWindow(win fyne.Window) {
	if hwnd := hwndOf(win); hwnd != 0 {
		snapEdges(hwnd)
	}
}

// noteForeground records the window in the foreground as a click arrives on
// one that is not: WM_MOUSEACTIVATE is sent before the activation, so the
// foreground window is still the one the person was in.
func noteForeground(hwnd uintptr) {
	fg, _, _ := getForegroundWindow.Call()
	if fg == 0 || fg == hwnd {
		return
	}
	snapMu.Lock()
	before[hwnd] = fg
	snapMu.Unlock()
}

/*
yieldFocus gives the keyboard back to the window the person was in before a
click on the panel took it, and reports that this platform has its own way.

It does nothing unless this window is the foreground one: a panel that was
never given the keyboard has nothing to give back, and must not pull another
window forward.

The window handed to is the one that was in the foreground when the click
arrived, recorded on WM_MOUSEACTIVATE. Where none was recorded, or it has
gone, it is the first window under the panel in the Z-order that a person
could have been typing in: a window activated by a click goes to the top of
the order, so the one under it is the one that was active before.
*/
func yieldFocus(win fyne.Window) bool {
	hwnd := hwndOf(win)
	if hwnd == 0 {
		// No native window: a test driver's, which has no foreground to give.
		return false
	}
	if fg, _, _ := getForegroundWindow.Call(); fg != hwnd {
		return true
	}

	snapMu.Lock()
	to := before[hwnd]
	delete(before, hwnd)
	snapMu.Unlock()

	if !typedInto(to, hwnd) {
		to = 0
		for h, _, _ := getWindowNext.Call(hwnd, gwHwndNext); h != 0; h, _, _ = getWindowNext.Call(h, gwHwndNext) {
			if typedInto(h, hwnd) {
				to = h
				break
			}
		}
	}
	if to != 0 {
		_, _, _ = setForegroundWindow.Call(to)
	}
	return true
}

// typedInto reports whether h is a window a person could have been typing
// in: there, visible, not minimised, not cloaked, titled, not a tool window,
// and not one of this program's.
func typedInto(h, self uintptr) bool {
	if h == 0 || h == self {
		return false
	}
	if ok, _, _ := isWindow.Call(h); ok == 0 {
		return false
	}
	if v, _, _ := isWindowVisible.Call(h); v == 0 {
		return false
	}
	if v, _, _ := isIconic.Call(h); v != 0 {
		return false
	}
	if style, _, _ := getWindowLongPtr.Call(h, uintptr(gwlExStyle)); style&(wsExToolWindow|wsExNoActivate) != 0 {
		return false
	}
	if n, _, _ := getWindowTextLength.Call(h); n == 0 {
		return false
	}
	var cloaked uint32
	_, _, _ = dwmGetWindowAttribute.Call(h, dwmwaCloaked, uintptr(unsafe.Pointer(&cloaked)), unsafe.Sizeof(cloaked))
	if cloaked != 0 {
		return false
	}
	var own, pid uint32
	_, _, _ = getWindowThreadProcessID.Call(self, uintptr(unsafe.Pointer(&own)))
	_, _, _ = getWindowThreadProcessID.Call(h, uintptr(unsafe.Pointer(&pid)))
	return pid != own
}
