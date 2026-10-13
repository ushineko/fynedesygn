package glance

import (
	"math"
	"unsafe"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver"
)

// movesItself is true where a window can move itself and read where it is,
// which is what a drawer on the left and a drawer that chooses its side
// need. Elsewhere the compositor does both; see Drawer.SetEdgeHeld and
// Drawer.Choose.
const movesItself = true

var (
	getWindowRect = user32.NewProc("GetWindowRect")
	setWindowPos  = user32.NewProc("SetWindowPos")
)

// hwndOf is the window's handle, or 0 before it is shown.
func hwndOf(win fyne.Window) uintptr {
	native, ok := win.(driver.NativeWindow)
	if !ok {
		return 0
	}
	var hwnd uintptr
	native.RunNative(func(ctx any) {
		if wc, ok := ctx.(driver.WindowsWindowContext); ok {
			hwnd = wc.HWND
		}
	})
	return hwnd
}

// moveWindowBy moves the window sideways by dx, in Fyne's units, and leaves
// its size alone.
func moveWindowBy(win fyne.Window, dx float32) {
	hwnd := hwndOf(win)
	if hwnd == 0 {
		return
	}
	// Rounded, not truncated: the window's width is rounded when Fyne sets
	// it, and a move a pixel short of it leaves the far edge a pixel out.
	px := int32(math.Round(float64(dx * win.Canvas().Scale())))
	var r edges
	if ok, _, _ := getWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&r))); ok == 0 {
		return
	}
	const noSize, noZOrder, noActivate = 0x0001, 0x0004, 0x0010
	_, _, _ = setWindowPos.Call(hwnd, 0,
		uintptr(r.left+px), uintptr(r.top), 0, 0, noSize|noZOrder|noActivate)
}

// roomBeside is the room between the window and each side of its monitor's
// work area, the taskbar left out, in Fyne's units.
func roomBeside(win fyne.Window) (left, right float32, ok bool) {
	hwnd, scale := hwndOf(win), win.Canvas().Scale()
	if hwnd == 0 || scale == 0 {
		return 0, 0, false
	}
	var r edges
	if got, _, _ := getWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&r))); got == 0 {
		return 0, 0, false
	}
	mon, _, _ := monitorFromRect.Call(uintptr(unsafe.Pointer(&r)), monitorDefaultToNearest)
	info := monitorInfo{size: uint32(unsafe.Sizeof(monitorInfo{}))}
	if got, _, _ := getMonitorInfo.Call(mon, uintptr(unsafe.Pointer(&info))); got == 0 {
		return 0, 0, false
	}
	return float32(r.left-info.work.left) / scale, float32(info.work.right-r.right) / scale, true
}
