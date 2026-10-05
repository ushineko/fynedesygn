//go:build cgo && !wasm && !js && !android && !ios && !mobile && !test_web_driver

package widgets

import (
	"sync"

	"fyne.io/fyne/v2"
	"github.com/go-gl/glfw/v3.4/glfw"

	"github.com/ushineko/fynedesygn/internal/glfwwin"
)

/*
watchLeave takes every tip down when the pointer leaves the window a canvas
belongs to. Quirk 42.

Fyne sends MouseOut only when the pointer moves onto another object inside the
window; it registers no GLFW cursor-enter callback, so leaving the window
sends nothing at all. A tip waiting on a control the pointer left the window
from appeared a second later anyway, with the pointer elsewhere on the
desktop, and stayed until the pointer came back. On a small window at the
edge of the screen -- a panel -- leaving straight from a control is the usual
way out.

GLFW does report it, so this registers the callback Fyne leaves unset, once
per window. Fyne does not hand out its GLFW window, so it is read from the
window's unexported field (glfwwin.ViewportOf). Where that cannot be done -- a test
window, another driver, a Fyne that renamed the field -- nothing is
registered and tips behave as they did; TestFyneStillKeepsItsGLFWWindowWhereATipLooks
is what notices a rename. Called on the main thread, from MouseIn.
*/
func watchLeave(c fyne.Canvas) {
	app := fyne.CurrentApp()
	if c == nil || app == nil {
		return
	}
	for _, w := range app.Driver().AllWindows() {
		if w.Canvas() != c {
			continue
		}
		view := glfwwin.ViewportOf(w)
		if view == nil {
			return
		}
		watchedMu.Lock()
		seen := watched[view]
		watched[view] = true
		watchedMu.Unlock()
		if seen {
			return
		}
		var previous glfw.CursorEnterCallback
		previous = view.SetCursorEnterCallback(func(win *glfw.Window, entered bool) {
			if !entered {
				HideTips()
			}
			if previous != nil {
				previous(win, entered)
			}
		})
		return
	}
}

// watched is every GLFW window that has the callback, so it is registered
// once however many tips the window has.
var (
	watchedMu sync.Mutex
	watched   = map[*glfw.Window]bool{}
)
