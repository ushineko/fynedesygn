//go:build cgo && !wasm && !js && !android && !ios && !mobile && !test_web_driver

package glance

import "github.com/go-gl/glfw/v3.4/glfw"

// grantTranslucent asks GLFW for a transparent framebuffer and reports whether
// this desktop and driver will actually give one.
//
// It is two steps and both are necessary. The *probe* makes a one-pixel hidden
// window with the hint, reads back what it got and throws it away; the hint is
// a request, and a window that asked and was refused clears to black rather
// than to the desktop. Refusal is not hypothetical: it is reported on AMD
// drivers on Windows (glfw#2731) and on dedicated laptop GPUs (glfw#1288).
// Fyne hands back no glfw.Window, so a one-pixel window of our own is the only
// way to ask. The *hint* is then left set for the window Fyne is about to
// create: hints are sticky global state and Fyne never calls
// glfw.DefaultWindowHints, so the next window created inherits it.
//
// Both must run on the main thread, after GLFW is initialised and before Fyne
// creates the window. That is a narrow gap and Window.ShowAndRun is what fits
// in it.
func grantTranslucent() bool {
	glfw.WindowHint(glfw.Visible, glfw.False)
	glfw.WindowHint(glfw.TransparentFramebuffer, glfw.True)

	probe, err := glfw.CreateWindow(1, 1, "", nil, nil)
	if err != nil || probe == nil {
		glfw.WindowHint(glfw.TransparentFramebuffer, glfw.False)
		return false
	}
	granted := probe.GetAttrib(glfw.TransparentFramebuffer) == glfw.True
	probe.Destroy()

	if !granted {
		glfw.WindowHint(glfw.TransparentFramebuffer, glfw.False)
	}
	return granted
}
