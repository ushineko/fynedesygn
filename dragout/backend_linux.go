//go:build cgo && linux && !android && !test_web_driver

package dragout

import (
	"fyne.io/fyne/v2"
	"github.com/go-gl/glfw/v3.4/glfw"

	"github.com/ushineko/fynedesygn/internal/glfwwin"
)

// glfwBackend is the drag source for Fyne's GLFW driver on Linux: Wayland or
// X11, whichever GLFW chose.
type glfwBackend struct{}

func newBackend() backend { return glfwBackend{} }

// supports needs Fyne's GLFW window, which a test window does not have, and
// a GLFW running on a platform this package drives.
func (glfwBackend) supports(w fyne.Window) bool {
	if glfwwin.ViewportOf(w) == nil {
		return false
	}
	switch glfw.GetPlatform() {
	case glfw.PlatformWayland:
		return haveWayland
	case glfw.PlatformX11:
		return haveX11
	}
	return false
}

// prepare binds what the Wayland drag needs before the press it will start
// from. X11 needs nothing in advance.
func (b glfwBackend) prepare(w fyne.Window) {
	if b.supports(w) && glfw.GetPlatform() == glfw.PlatformWayland {
		_ = waylandPrepare() // a failure is reported by start
	}
}

func (glfwBackend) start(w fyne.Window, paths []string) error {
	if glfw.GetPlatform() == glfw.PlatformWayland {
		return waylandStart(w, paths)
	}
	return x11Start(w, paths)
}

/*
release sends Fyne the left button's release that it will never receive.
Quirk 45.

Once the platform owns the pointer, the release goes to the drop target,
not to this window. Fyne would go on believing the button is down: it would
call Dragged on the next hover and swallow the next click as the end of a
drag. GLFW's callback is Fyne's own handler, so calling it is the same path
a real release takes.
*/
func (glfwBackend) release(w fyne.Window) {
	view := glfwwin.ViewportOf(w)
	if view == nil {
		return
	}
	handler := view.SetMouseButtonCallback(nil)
	view.SetMouseButtonCallback(handler)
	if handler != nil {
		handler(view, glfw.MouseButtonLeft, glfw.Release, 0)
	}
}
