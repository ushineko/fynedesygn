//go:build !cgo || wasm || js || android || ios || mobile || test_web_driver

package widgets

import "fyne.io/fyne/v2"

// watchLeave does nothing on a build with no GLFW: there is no window for the
// pointer to leave that this could hear about.
func watchLeave(fyne.Canvas) {}
