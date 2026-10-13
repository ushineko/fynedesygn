//go:build cgo && !wasm && !js && !android && !ios && !mobile && !test_web_driver

package glance

import (
	"fyne.io/fyne/v2"

	"github.com/ushineko/fynedesygn/internal/glfwwin"
)

/*
repoint tells Fyne where the pointer is after the compositor moved the window
under it, and reports whether it could. Quirk 48.

On Wayland a pointer's position is the last motion event's, in the surface's
own coordinates, and the compositor sends none when it moves a window under a
pointer that stays where it is. Fyne then believes the pointer is where it
was on the window, which is dx away from where it is; a click lands that far
off, and when the window has also become narrower the believed place can be
outside the window altogether, where Fyne finds nothing and the click is
lost.

The motion event that was not sent is made up here: Fyne's own cursor
callback is called with the position GLFW last heard, moved by everything the
window has moved since. GLFW's position is not changed by this, so it is also
how a real movement is told from none: while GLFW still reports the position
it reported at the last call, the pointer has not moved and the moves add up;
once it reports another, the pointer has been heard from and they start again
from nothing.

Fyne does not hand out its GLFW window, so it is read from the window's
unexported field (glfwwin.ViewportOf), as the tips' leave watcher reads it.
Where that cannot be done -- a test window, another driver, a Fyne that
renamed the field -- nothing is called and the panel falls back to reading
the next click itself (Panel.tapped). Called on the main thread.
*/
func repoint(win fyne.Window, at *pointer, dx float32) bool {
	view := glfwwin.ViewportOf(win)
	if view == nil {
		return false
	}
	x, y := view.GetCursorPos()
	if !at.heard || x != at.x || y != at.y {
		at.moved = 0
	}
	at.x, at.y, at.heard = x, y, true
	at.moved += dx

	// The callback is Fyne's, the only one registered. Reading it means
	// taking it off and putting it back, which is two calls on this thread
	// with nothing between them.
	moved := view.SetCursorPosCallback(nil)
	view.SetCursorPosCallback(moved)
	if moved == nil {
		return false
	}
	// Fyne divides a GLFW coordinate by the canvas's scale to make its own.
	moved(view, x+float64(at.moved*win.Canvas().Scale()), y)
	return true
}
