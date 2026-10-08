package glance_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
	"time"
	"unsafe"

	"github.com/stretchr/testify/require"
)

// The real-window tests drag a real glance window with the real pointer, so
// they run only when asked: they move the mouse on the desk they run on.
const realWindowEnv = "FYNEDESYGN_WINDOW_TEST"

var (
	rwUser32                 = syscall.NewLazyDLL("user32.dll")
	rwEnumWindows            = rwUser32.NewProc("EnumWindows")
	rwGetWindowThreadProcess = rwUser32.NewProc("GetWindowThreadProcessId")
	rwIsWindowVisible        = rwUser32.NewProc("IsWindowVisible")
	rwGetWindowRect          = rwUser32.NewProc("GetWindowRect")
	rwMonitorFromWindow      = rwUser32.NewProc("MonitorFromWindow")
	rwGetMonitorInfo         = rwUser32.NewProc("GetMonitorInfoW")
	rwGetCursorPos           = rwUser32.NewProc("GetCursorPos")
	rwSetCursorPos           = rwUser32.NewProc("SetCursorPos")
	rwMouseEvent             = rwUser32.NewProc("mouse_event")
	rwSetDpiAwareness        = rwUser32.NewProc("SetProcessDpiAwarenessContext")
)

type rwRect struct{ L, T, R, B int32 }

type rwMonitorInfo struct {
	Size    uint32
	Monitor rwRect
	Work    rwRect
	Flags   uint32
}

/*
Spec 058. A glance window dragged on Windows snaps to the edges of its
monitor's work area, and one dropped away from them stays where it was
dropped.

The window is fynedesygn's own glance-monitor example, built here and run
with its profile folders in the test's temporary directory. The drag is the
real one: the pointer is pressed on the window, which hands it to Windows'
move loop (spec 052), moved in steps and released, and the window's rectangle
is read afterwards. The pointer is put back where it was.
*/
func TestADraggedGlanceWindowSnapsToTheWorkAreaOnWindows(t *testing.T) {
	if os.Getenv(realWindowEnv) == "" {
		t.Skip("moves the real pointer; set " + realWindowEnv + "=1 to run it")
	}
	// Coordinates in physical pixels, as the window's own are.
	_, _, _ = rwSetDpiAwareness.Call(^uintptr(3)) // DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2 (-4)

	hwnd := runGlanceMonitor(t)
	var saved struct{ X, Y int32 }
	_, _, _ = rwGetCursorPos.Call(uintptr(unsafe.Pointer(&saved)))
	t.Cleanup(func() { _, _, _ = rwSetCursorPos.Call(uintptr(saved.X), uintptr(saved.Y)) })

	work := workArea(hwnd)
	start := windowRect(hwnd)
	w, h := start.R-start.L, start.B-start.T
	t.Logf("work area %+v, window %+v", work, start)

	// Near the top-left corner: 6 px in from the left, 4 px down from the top.
	drag(t, hwnd, work.L+6, work.T+4)
	got := windowRect(hwnd)
	t.Logf("dropped at (%d, %d), window at (%d, %d)", work.L+6, work.T+4, got.L, got.T)
	require.Equal(t, work.L, got.L, "a window dropped 6 px from the left edge is on it")
	require.Equal(t, work.T, got.T, "a window dropped 4 px below the top edge is on it")

	// Away from every edge: where it was dropped.
	x, y := work.L+(work.R-work.L-w)/2, work.T+(work.B-work.T-h)/2
	drag(t, hwnd, x, y)
	got = windowRect(hwnd)
	t.Logf("dropped at (%d, %d), window at (%d, %d)", x, y, got.L, got.T)
	require.Equal(t, x, got.L, "a window dropped away from the edges stays where it was dropped")
	require.Equal(t, y, got.T, "a window dropped away from the edges stays where it was dropped")
}

// drag presses the pointer on the window, moves it so the window's top-left
// would be at (x, y), and releases.
func drag(t *testing.T, hwnd uintptr, x, y int32) {
	t.Helper()
	const (
		leftDown = 0x0002
		leftUp   = 0x0004
	)
	r := windowRect(hwnd)
	// Where spec 052 pressed: inside the window, clear of its edges.
	px, py := r.L+60, r.T+40
	_, _, _ = rwSetCursorPos.Call(uintptr(px), uintptr(py))
	time.Sleep(150 * time.Millisecond)
	_, _, _ = rwMouseEvent.Call(leftDown, 0, 0, 0, 0)
	time.Sleep(150 * time.Millisecond)
	dx, dy := x-r.L, y-r.T
	const steps = 20
	for i := int32(1); i <= steps; i++ {
		_, _, _ = rwSetCursorPos.Call(uintptr(px+dx*i/steps), uintptr(py+dy*i/steps))
		time.Sleep(20 * time.Millisecond)
	}
	time.Sleep(100 * time.Millisecond)
	_, _, _ = rwMouseEvent.Call(leftUp, 0, 0, 0, 0)
	time.Sleep(400 * time.Millisecond)
}

func windowRect(hwnd uintptr) rwRect {
	var r rwRect
	_, _, _ = rwGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&r)))
	return r
}

func workArea(hwnd uintptr) rwRect {
	mon, _, _ := rwMonitorFromWindow.Call(hwnd, 2) // MONITOR_DEFAULTTONEAREST
	info := rwMonitorInfo{Size: uint32(unsafe.Sizeof(rwMonitorInfo{}))}
	_, _, _ = rwGetMonitorInfo.Call(mon, uintptr(unsafe.Pointer(&info)))
	return info.Work
}

// runGlanceMonitor builds and starts the glance-monitor example and returns
// its window. It is killed when the test ends.
func runGlanceMonitor(t *testing.T, args ...string) uintptr {
	t.Helper()
	dir := t.TempDir()
	bin := filepath.Join(dir, "glance-monitor.exe")
	_, file, _, _ := runtime.Caller(0)
	build := exec.CommandContext(t.Context(), "go", "build", "-o", bin, "./examples/glance-monitor")
	build.Dir = filepath.Join(filepath.Dir(file), "..")
	build.Env = append(os.Environ(), "CGO_ENABLED=1")
	out, err := build.CombinedOutput()
	require.NoError(t, err, "%s", out)

	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, bin, args...)
	profile := filepath.Join(dir, "profile")
	cmd.Env = append(os.Environ(),
		"APPDATA="+filepath.Join(profile, "Roaming"),
		"LOCALAPPDATA="+filepath.Join(profile, "Local"),
		"USERPROFILE="+profile)
	require.NoError(t, cmd.Start())
	t.Cleanup(func() {
		cancel()
		_ = cmd.Wait()
	})

	pid := uint32(cmd.Process.Pid)
	var found uintptr
	cb := syscall.NewCallback(func(h, _ uintptr) uintptr {
		var owner uint32
		_, _, _ = rwGetWindowThreadProcess.Call(h, uintptr(unsafe.Pointer(&owner)))
		if owner == pid {
			if v, _, _ := rwIsWindowVisible.Call(h); v != 0 {
				found = h
				return 0
			}
		}
		return 1
	})
	for deadline := time.Now().Add(20 * time.Second); found == 0 && time.Now().Before(deadline); time.Sleep(200 * time.Millisecond) {
		_, _, _ = rwEnumWindows.Call(cb, 0)
	}
	require.NotZero(t, found, "the glance window did not appear")
	time.Sleep(time.Second)
	return found
}

/*
A window snapped onto an edge leaves it as soon as the pointer carries it
past the snap distance, in the same drag, and comes off it under the point
that was grabbed.

The first version snapped the rectangle Windows proposed, and Windows builds
each proposal from the last one as it was left, plus the pointer's movement
since. Once snapped, every small movement started from the edge, fell within
the snap distance and was snapped back: the window stuck to the edge while the
pointer ran on, and had to be released and dragged again, several times, to
come off (hayami's 0.9.2 sit test).
*/
func TestASnappedGlanceWindowComesOffTheEdgeInTheSameDrag(t *testing.T) {
	if os.Getenv(realWindowEnv) == "" {
		t.Skip("moves the real pointer; set " + realWindowEnv + "=1 to run it")
	}
	_, _, _ = rwSetDpiAwareness.Call(^uintptr(3)) // DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2 (-4)

	hwnd := runGlanceMonitor(t)
	var saved struct{ X, Y int32 }
	_, _, _ = rwGetCursorPos.Call(uintptr(unsafe.Pointer(&saved)))
	t.Cleanup(func() { _, _, _ = rwSetCursorPos.Call(uintptr(saved.X), uintptr(saved.Y)) })

	work := workArea(hwnd)
	start := windowRect(hwnd)
	w, h := start.R-start.L, start.B-start.T
	mid := work.L + (work.R-work.L-w)/2

	// One drag: up onto the top edge (4 px short of it, which snaps), then on
	// down to the middle of the screen without letting go.
	end := work.T + (work.B-work.T-h)/2
	dragThrough(t, hwnd, [][2]int32{{mid, work.T + 4}, {mid, end}})
	got := windowRect(hwnd)
	t.Logf("through the top edge to (%d, %d): window at (%d, %d)", mid, end, got.L, got.T)
	require.Equal(t, mid, got.L, "the window did not follow the pointer off the edge")
	require.Equal(t, end, got.T, "the window stayed on the edge, or came off it away from the point grabbed")
}

// dragThrough presses on the window and moves it, without letting go, so its
// top-left passes each point in turn, then releases.
func dragThrough(t *testing.T, hwnd uintptr, points [][2]int32) {
	t.Helper()
	const (
		leftDown = 0x0002
		leftUp   = 0x0004
	)
	r := windowRect(hwnd)
	px, py := r.L+60, r.T+40
	_, _, _ = rwSetCursorPos.Call(uintptr(px), uintptr(py))
	time.Sleep(150 * time.Millisecond)
	_, _, _ = rwMouseEvent.Call(leftDown, 0, 0, 0, 0)
	time.Sleep(150 * time.Millisecond)
	from := [2]int32{r.L, r.T}
	for _, p := range points {
		dx, dy := p[0]-from[0], p[1]-from[1]
		// A few pixels a step, as a hand moves: a step longer than the snap
		// distance jumps clear of an edge in one message and hides the fault.
		steps := max(abs32(dx), abs32(dy))/3 + 1
		for i := int32(1); i <= steps; i++ {
			_, _, _ = rwSetCursorPos.Call(uintptr(px+dx*i/steps), uintptr(py+dy*i/steps))
			time.Sleep(4 * time.Millisecond)
		}
		px, py = px+dx, py+dy
		from = p
	}
	time.Sleep(100 * time.Millisecond)
	_, _, _ = rwMouseEvent.Call(leftUp, 0, 0, 0, 0)
	time.Sleep(400 * time.Millisecond)
}

func abs32(v int32) int32 {
	if v < 0 {
		return -v
	}
	return v
}
