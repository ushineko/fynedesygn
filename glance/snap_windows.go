package glance

import (
	"sync"
	"syscall"
	"unsafe"
)

/*
A window dragged on Windows snaps to the edges of its monitor's work area
(spec 058).

On Linux the window manager moves a glance window and its border snap zone
pulls it onto an edge; Windows' move loop, which startMove hands the drag to,
has no such pull. So the window procedure is wrapped: during the loop
Windows sends WM_MOVING with the rectangle the window is about to take, and
the wrapper moves that rectangle onto any edge of the work area it is within
snapDistance of, before the window moves.

GLFW owns the window procedure. The wrapper replaces it for the one window
(SetWindowLongPtrW, GWLP_WNDPROC) and passes every message on to it
(CallWindowProcW), so nothing GLFW handles changes. comctl32's
SetWindowSubclass would do the same with less bookkeeping, but it is exported
by name only from comctl32 version 6, which a Go program loads only with a
manifest it does not have. The original procedure is put back on
WM_NCDESTROY, the last message a window receives.
*/

var (
	setWindowLongPtr = user32.NewProc("SetWindowLongPtrW")
	callWindowProc   = user32.NewProc("CallWindowProcW")
	monitorFromRect  = user32.NewProc("MonitorFromRect")
	getMonitorInfo   = user32.NewProc("GetMonitorInfoW")
	getDpiForWindow  = user32.NewProc("GetDpiForWindow")
)

const (
	wmMoving    = 0x0216
	wmNCDestroy = 0x0082
	// monitorDefaultToNearest makes MonitorFromRect answer for a rectangle
	// that is off every screen, as a window dragged past an edge can be.
	monitorDefaultToNearest = 2
)

// gwlpWndProc is GWLP_WNDPROC, -4: a variable because a negative constant
// does not convert to uintptr.
var gwlpWndProc = -4

// snapping holds, for each window wrapped, the procedure it had before.
var (
	snapMu   sync.Mutex
	snapping = map[uintptr]uintptr{}
	// snapProc is made once: the runtime keeps a fixed number of callbacks.
	snapProc = syscall.NewCallback(snapWindowProc)
)

// monitorInfo is MONITORINFO.
type monitorInfo struct {
	size    uint32
	monitor edges
	work    edges
	flags   uint32
}

// snapEdges wraps the window's procedure, once, so a drag snaps to the edges
// of its work area. It runs on the window's thread, from startMove.
func snapEdges(hwnd uintptr) {
	snapMu.Lock()
	defer snapMu.Unlock()
	if _, done := snapping[hwnd]; done {
		return
	}
	prev, _, _ := setWindowLongPtr.Call(hwnd, uintptr(gwlpWndProc), snapProc)
	if prev == 0 {
		return
	}
	snapping[hwnd] = prev
}

// snapWindowProc is the wrapper: it adjusts WM_MOVING's rectangle, unwraps on
// WM_NCDESTROY, and hands every message to the window's own procedure.
func snapWindowProc(hwnd, msg, wparam, lparam uintptr) uintptr {
	snapMu.Lock()
	prev := snapping[hwnd]
	if msg == wmNCDestroy {
		delete(snapping, hwnd)
	}
	snapMu.Unlock()
	if prev == 0 {
		return 0
	}
	if msg == wmNCDestroy {
		_, _, _ = setWindowLongPtr.Call(hwnd, uintptr(gwlpWndProc), prev)
	}
	if msg == wmMoving && lparam != 0 {
		// lParam is a RECT the system owns for the length of the call.
		snapMoving(hwnd, (*edges)(unsafe.Add(nil, lparam)))
	}
	ret, _, _ := callWindowProc.Call(prev, hwnd, msg, wparam, lparam)
	if msg == wmMoving {
		// The message was handled: the rectangle says where to go.
		return 1
	}
	return ret
}

// snapMoving moves the rectangle the window is about to take onto the work
// area's edges it is near.
func snapMoving(hwnd uintptr, r *edges) {
	mon, _, _ := monitorFromRect.Call(uintptr(unsafe.Pointer(r)), monitorDefaultToNearest)
	if mon == 0 {
		return
	}
	info := monitorInfo{size: uint32(unsafe.Sizeof(monitorInfo{}))}
	if ok, _, _ := getMonitorInfo.Call(mon, uintptr(unsafe.Pointer(&info))); ok == 0 {
		return
	}
	var dpi uint32
	if getDpiForWindow.Find() == nil {
		d, _, _ := getDpiForWindow.Call(hwnd)
		dpi = uint32(d)
	}
	*r = snapToWork(*r, info.work, snapDistance(dpi))
}
