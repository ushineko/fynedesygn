package glance

import (
	"context"
	"os"
	"syscall"
	"time"
	"unsafe"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver"
)

// fullscreenSupported is true: Windows keeps an always-on-top window over a
// borderless full-screen one, so the glance window stands aside itself.
const fullscreenSupported = true

// fullscreenInterval is how often the window in front is looked at. Polled
// rather than hooked: a browser going full screen with F11 changes its size
// and not which window is in front, so a foreground-change hook alone misses
// it, and a look is a handful of cheap calls.
const fullscreenInterval = 500 * time.Millisecond

var (
	getForegroundWindow      = user32.NewProc("GetForegroundWindow")
	getWindowThreadProcessID = user32.NewProc("GetWindowThreadProcessId")
	getShellWindow           = user32.NewProc("GetShellWindow")
	getClassName             = user32.NewProc("GetClassNameW")
	isWindowVisible          = user32.NewProc("IsWindowVisible")
	isIconic                 = user32.NewProc("IsIconic")
	isZoomed                 = user32.NewProc("IsZoomed")
	isWindow                 = user32.NewProc("IsWindow")
	getWindowLongPtr         = user32.NewProc("GetWindowLongPtrW")
	getWindowRectFS          = user32.NewProc("GetWindowRect")
	monitorFromWindow        = user32.NewProc("MonitorFromWindow")
	showWindowAsync          = user32.NewProc("ShowWindowAsync")
	dwmapi                   = syscall.NewLazyDLL("dwmapi.dll")
	dwmGetWindowAttribute    = dwmapi.NewProc("DwmGetWindowAttribute")
)

const (
	swHide           = 0
	swShowNoActivate = 4
	gwlStyle         = -16
	wsCaption        = 0x00C00000
	dwmwaCloaked     = 14
	dwmwaFrameBounds = 9
)

// shellClasses are the desktop and the taskbars: in front of everything when
// the desktop is clicked, and never a reason to hide.
var shellClasses = map[string]bool{
	"Progman": true, "WorkerW": true, "Shell_TrayWnd": true, "Shell_SecondaryTrayWnd": true,
}

/*
watchFullscreen looks at the window in front every fullscreenInterval and
hides or shows the glance window as settle decides, until ctx ends or the
window is destroyed. Ending, it shows a window it hid.

The calls are Win32's and are safe from this goroutine. The hide and the show
are ShowWindowAsync, which posts to the window's own thread rather than
waiting on it: the window is Fyne's, its thread is Fyne's main loop, and a
synchronous call from here could wait on a loop that is waiting on this. The
show is SW_SHOWNOACTIVATE, so coming back never takes the focus from whatever
the person has just switched to, and neither call moves the window or changes
its always-on-top; GetWindowRect answers the same for a hidden window, so the
position code that polls it, and the snap code, see nothing change.
*/
func watchFullscreen(ctx context.Context, w *Window) {
	tick := time.NewTicker(fullscreenInterval)
	defer tick.Stop()
	var hwnd uintptr
	var s settle
	defer func() {
		if s.hidden && hwnd != 0 {
			_, _, _ = showWindowAsync.Call(hwnd, swShowNoActivate)
		}
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		}
		if hwnd == 0 {
			// The window has a handle only once it is shown; until then there
			// is nothing to hide.
			hwnd = nativeHandle(w)
			if hwnd == 0 {
				continue
			}
		}
		if ok, _, _ := isWindow.Call(hwnd); ok == 0 {
			return
		}
		hidden, changed := s.next(fullscreenInFront(lookInFront(hwnd)))
		if !changed {
			continue
		}
		cmd := uintptr(swShowNoActivate)
		if hidden {
			cmd = swHide
		}
		_, _, _ = showWindowAsync.Call(hwnd, cmd)
	}
}

// nativeHandle is the glance window's HWND, or 0 before it is shown.
func nativeHandle(w *Window) uintptr {
	native, ok := w.win.(driver.NativeWindow)
	if !ok {
		return 0
	}
	var hwnd uintptr
	fyne.DoAndWait(func() {
		native.RunNative(func(ctx any) {
			if wc, ok := ctx.(driver.WindowsWindowContext); ok {
				hwnd = wc.HWND
			}
		})
	})
	return hwnd
}

// lookInFront gathers what fullscreenInFront decides on, for the window in
// front of the glance window self.
func lookInFront(self uintptr) foreground {
	fg, _, _ := getForegroundWindow.Call()
	if fg == 0 {
		return foreground{Hidden: true}
	}
	var f foreground
	var pid uint32
	_, _, _ = getWindowThreadProcessID.Call(fg, uintptr(unsafe.Pointer(&pid)))
	f.Own = fg == self || int(pid) == os.Getpid()

	shell, _, _ := getShellWindow.Call()
	f.Shell = fg == shell || shellClasses[className(fg)]

	visible, _, _ := isWindowVisible.Call(fg)
	iconic, _, _ := isIconic.Call(fg)
	var cloaked uint32
	_, _, _ = dwmGetWindowAttribute.Call(fg, dwmwaCloaked, uintptr(unsafe.Pointer(&cloaked)), unsafe.Sizeof(cloaked))
	f.Hidden = visible == 0 || iconic != 0 || cloaked != 0

	_, _, _ = getWindowRectFS.Call(fg, uintptr(unsafe.Pointer(&f.Rect)))
	// The frame DWM draws, without the invisible resize borders GetWindowRect
	// includes; zero where DWM has none to give.
	_, _, _ = dwmGetWindowAttribute.Call(fg, dwmwaFrameBounds, uintptr(unsafe.Pointer(&f.Frame)), unsafe.Sizeof(f.Frame))
	mon, _, _ := monitorFromWindow.Call(fg, monitorDefaultToNearest)
	ownMon, _, _ := monitorFromWindow.Call(self, monitorDefaultToNearest)
	f.SameMonitor = mon != 0 && mon == ownMon
	info := monitorInfo{size: uint32(unsafe.Sizeof(monitorInfo{}))}
	if ok, _, _ := getMonitorInfo.Call(mon, uintptr(unsafe.Pointer(&info))); ok == 0 {
		f.SameMonitor = false
	}
	f.Monitor = info.monitor

	zoomed, _, _ := isZoomed.Call(fg)
	style, _, _ := getWindowLongPtr.Call(fg, uintptr(gwlStyleValue))
	f.Maximised = zoomed != 0
	f.Captioned = style&wsCaption == wsCaption
	return f
}

// gwlStyleValue is GWL_STYLE as a variable: a negative constant does not
// convert to uintptr.
var gwlStyleValue = gwlStyle

// className is a window's class name.
func className(hwnd uintptr) string {
	buf := make([]uint16, 64)
	n, _, _ := getClassName.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	return syscall.UTF16ToString(buf[:n])
}
