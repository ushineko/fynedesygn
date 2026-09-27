//go:build cgo && linux && !android && !wasm && !js && !mobile && !test_web_driver

package glance

/*
#cgo pkg-config: x11
#include <X11/Xlib.h>
#include <X11/Xatom.h>
#include <stdlib.h>

// fyne_set_window_opacity writes _NET_WM_WINDOW_OPACITY on a window, which is
// what a compositing window manager reads to fade it. It is the same two lines
// GLFW's own X11 backend runs for glfwSetWindowOpacity, done here because Fyne
// hands back the window handle and not the GLFW window.
static int fyne_set_window_opacity(unsigned long window, double opacity) {
	Display *d = XOpenDisplay(NULL);
	if (d == NULL) {
		return 0;
	}
	unsigned long value = (unsigned long)(0xffffffffu * opacity);
	Atom prop = XInternAtom(d, "_NET_WM_WINDOW_OPACITY", False);
	XChangeProperty(d, (Window)window, prop, XA_CARDINAL, 32, PropModeReplace,
		(unsigned char *)&value, 1);
	XFlush(d);
	XCloseDisplay(d);
	return 1;
}
*/
import "C"

// setWindowOpacity writes the opacity property on an X11 window. It reports
// whether it reached a display at all; the window manager may still ignore the
// property, and there is no way to ask it whether it did.
func setWindowOpacity(handle uintptr, opacity float32) bool {
	return C.fyne_set_window_opacity(C.ulong(handle), C.double(opacity)) == 1
}
