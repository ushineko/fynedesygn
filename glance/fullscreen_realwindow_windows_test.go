package glance_test

import (
	"os"
	"runtime"
	"syscall"
	"testing"
	"time"
	"unsafe"

	"github.com/stretchr/testify/require"
)

var (
	rwCreateWindowEx      = rwUser32.NewProc("CreateWindowExW")
	rwDestroyWindow       = rwUser32.NewProc("DestroyWindow")
	rwPeekMessage         = rwUser32.NewProc("PeekMessageW")
	rwTranslateMessage    = rwUser32.NewProc("TranslateMessage")
	rwDispatchMessage     = rwUser32.NewProc("DispatchMessageW")
	rwGetForegroundWindow = rwUser32.NewProc("GetForegroundWindow")
	rwSetForegroundWindow = rwUser32.NewProc("SetForegroundWindow")
	rwAttachThreadInput   = rwUser32.NewProc("AttachThreadInput")
	rwBringWindowToTop    = rwUser32.NewProc("BringWindowToTop")
	rwKernel32            = syscall.NewLazyDLL("kernel32.dll")
	rwGetCurrentThreadID  = rwKernel32.NewProc("GetCurrentThreadId")
)

/*
Spec 059. With the option on, a glance window hides while a full-screen
window is in front on its monitor, and comes back, in the same place and
without the focus, when it goes.

The full-screen window is a plain borderless popup this test opens: the shape
of a browser after F11 or a game in windowed full screen, and nothing else is
put in front. Two sizes, because a full-screen window is often a few pixels
larger than its monitor: exactly the monitor, and 8 px past it on every side.

The test takes the foreground for a few seconds. It skips when a full-screen
window is already in front -- a game or a film someone is watching -- rather
than take it over.
*/
func TestAGlanceWindowStandsAsideForAFullScreenWindow(t *testing.T) {
	if os.Getenv(realWindowEnv) == "" {
		t.Skip("puts a full-screen window in front; set " + realWindowEnv + "=1 to run it")
	}
	_, _, _ = rwSetDpiAwareness.Call(^uintptr(3)) // DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2 (-4)
	skipOverAFullScreenWindow(t)

	hwnd := runGlanceMonitor(t, "-hide-fullscreen")
	start := windowRect(hwnd)
	mon := monitorRect(hwnd)

	for _, c := range []struct {
		name string
		r    rwRect
	}{
		{"exactly the monitor", mon},
		{"8 px past every side", rwRect{mon.L - 8, mon.T - 8, mon.R + 8, mon.B + 8}},
	} {
		began := time.Now()
		closeCover := openCover(t, c.r)
		require.True(t, waitVisible(hwnd, false, 3*time.Second), "%s: the glance window did not hide", c.name)
		t.Logf("%s: hidden after %v", c.name, time.Since(began).Round(10*time.Millisecond))

		began = time.Now()
		closeCover()
		require.True(t, waitVisible(hwnd, true, 3*time.Second), "%s: the glance window did not come back", c.name)
		t.Logf("%s: back after %v", c.name, time.Since(began).Round(10*time.Millisecond))
		// Its place, not its size: the example's cards come and go as its
		// generated readings do, and the window is the size of its cards.
		back := windowRect(hwnd)
		require.Equal(t, [2]int32{start.L, start.T}, [2]int32{back.L, back.T}, "%s: the glance window came back somewhere else", c.name)
		fg, _, _ := rwGetForegroundWindow.Call()
		require.NotEqual(t, hwnd, fg, "%s: coming back took the focus", c.name)
	}
}

// Without the option, a full-screen window in front leaves the glance window
// as it was: the hiding is the option's and nothing else's.
func TestAGlanceWindowWithoutTheOptionStaysForAFullScreenWindow(t *testing.T) {
	if os.Getenv(realWindowEnv) == "" {
		t.Skip("puts a full-screen window in front; set " + realWindowEnv + "=1 to run it")
	}
	_, _, _ = rwSetDpiAwareness.Call(^uintptr(3))
	skipOverAFullScreenWindow(t)

	hwnd := runGlanceMonitor(t)
	closeCover := openCover(t, monitorRect(hwnd))
	defer closeCover()
	require.False(t, waitVisible(hwnd, false, 2500*time.Millisecond), "the glance window hid without the option")
}

// skipOverAFullScreenWindow skips the test when what is in front already
// covers its monitor: someone is playing or watching something.
func skipOverAFullScreenWindow(t *testing.T) {
	t.Helper()
	fg, _, _ := rwGetForegroundWindow.Call()
	if fg == 0 {
		return
	}
	r, m := windowRect(fg), monitorRect(fg)
	if r.L <= m.L && r.T <= m.T && r.R >= m.R && r.B >= m.B {
		t.Skip("a full-screen window is in front (a game or a film); not taking it over")
	}
}

// monitorRect is the full rectangle of the monitor a window is on.
func monitorRect(hwnd uintptr) rwRect {
	mon, _, _ := rwMonitorFromWindow.Call(hwnd, 2) // MONITOR_DEFAULTTONEAREST
	info := rwMonitorInfo{Size: uint32(unsafe.Sizeof(rwMonitorInfo{}))}
	_, _, _ = rwGetMonitorInfo.Call(mon, uintptr(unsafe.Pointer(&info)))
	return info.Monitor
}

// waitVisible waits up to d for the window to be visible (or not), and says
// whether it got there.
func waitVisible(hwnd uintptr, visible bool, d time.Duration) bool {
	for deadline := time.Now().Add(d); time.Now().Before(deadline); time.Sleep(50 * time.Millisecond) {
		v, _, _ := rwIsWindowVisible.Call(hwnd)
		if (v != 0) == visible {
			return true
		}
	}
	return false
}

/*
openCover opens a borderless popup over r and brings it to the front, and
returns what closes it.

The popup lives on a thread of its own, which pumps its messages, because a
window belongs to the thread that made it. It is brought to the front by
joining the input of the thread in front for a moment, the usual way a
program that is not in front may hand the foreground on; pressing a key to
unlock it would type into whatever was in front.
*/
func openCover(t *testing.T, r rwRect) func() {
	t.Helper()
	const (
		wsPopup   = 0x80000000
		wsVisible = 0x10000000
		pmRemove  = 1
	)
	made := make(chan uintptr, 1)
	stop := make(chan struct{})
	gone := make(chan struct{})
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		defer close(gone)
		class, _ := syscall.UTF16PtrFromString("STATIC")
		h, _, _ := rwCreateWindowEx.Call(0, uintptr(unsafe.Pointer(class)), 0, wsPopup|wsVisible,
			uintptr(r.L), uintptr(r.T), uintptr(r.R-r.L), uintptr(r.B-r.T), 0, 0, 0, 0)
		made <- h
		if h == 0 {
			return
		}
		fg, _, _ := rwGetForegroundWindow.Call()
		other, _, _ := rwGetWindowThreadProcess.Call(fg, 0)
		self, _, _ := rwGetCurrentThreadID.Call()
		if other != 0 && other != self {
			_, _, _ = rwAttachThreadInput.Call(self, other, 1)
		}
		_, _, _ = rwBringWindowToTop.Call(h)
		_, _, _ = rwSetForegroundWindow.Call(h)
		if other != 0 && other != self {
			_, _, _ = rwAttachThreadInput.Call(self, other, 0)
		}
		var msg [48]byte
		for {
			select {
			case <-stop:
				_, _, _ = rwDestroyWindow.Call(h)
				return
			default:
			}
			for {
				got, _, _ := rwPeekMessage.Call(uintptr(unsafe.Pointer(&msg[0])), 0, 0, 0, pmRemove)
				if got == 0 {
					break
				}
				_, _, _ = rwTranslateMessage.Call(uintptr(unsafe.Pointer(&msg[0])))
				_, _, _ = rwDispatchMessage.Call(uintptr(unsafe.Pointer(&msg[0])))
			}
			time.Sleep(10 * time.Millisecond)
		}
	}()
	h := <-made
	require.NotZero(t, h, "the full-screen popup was not made")
	time.Sleep(200 * time.Millisecond)
	fg, _, _ := rwGetForegroundWindow.Call()
	if fg != h {
		t.Logf("the popup is not in front (foreground %#x); the test reads only the glance window's state", fg)
	}
	var once bool
	closeIt := func() {
		if once {
			return
		}
		once = true
		close(stop)
		<-gone
	}
	t.Cleanup(closeIt)
	return closeIt
}
