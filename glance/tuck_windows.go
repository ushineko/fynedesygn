package glance

import (
	"math"
	"unsafe"

	"fyne.io/fyne/v2"
)

/*
moveCorner moves the window so that the bottom corner on the side given stays
where it is while its size goes from before to after, in Fyne's units, and
keeps the window inside its monitor's work area (spec 081 R5c). It leaves the
size alone: the caller resizes, before the move when shrinking and after it
when growing.

The pixels are the sizes as Fyne makes them, rounded up, as pixelsMoved reads
a width (#238), and not the difference rounded on its own.
*/
func moveCorner(win fyne.Window, corner Side, before, after fyne.Size) {
	hwnd, scale := hwndOf(win), win.Canvas().Scale()
	if hwnd == 0 || scale == 0 {
		return
	}
	var r edges
	if ok, _, _ := getWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&r))); ok == 0 {
		return
	}
	x, y := r.left, r.top
	if corner == Right {
		x += pixelsMoved(scale, before.Width, after.Width)
	}
	y += pixelsMoved(scale, before.Height, after.Height)

	// Kept on the monitor the window is on: a square moved into a corner of
	// the screen unfolds into a panel that would run off it.
	w := int32(math.Ceil(float64(after.Width * scale)))
	h := int32(math.Ceil(float64(after.Height * scale)))
	mon, _, _ := monitorFromRect.Call(uintptr(unsafe.Pointer(&r)), monitorDefaultToNearest)
	info := monitorInfo{size: uint32(unsafe.Sizeof(monitorInfo{}))}
	if got, _, _ := getMonitorInfo.Call(mon, uintptr(unsafe.Pointer(&info))); got != 0 {
		x = max(info.work.left, min(x, info.work.right-w))
		y = max(info.work.top, min(y, info.work.bottom-h))
	}
	const noSize, noZOrder, noActivate = 0x0001, 0x0004, 0x0010
	_, _, _ = setWindowPos.Call(hwnd, 0, uintptr(x), uintptr(y), 0, 0, noSize|noZOrder|noActivate)
}
