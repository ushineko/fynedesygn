//go:build !cgo || wasm || js || android || ios || mobile || test_web_driver

package glance

import "fyne.io/fyne/v2"

// repoint cannot reach a GLFW window here; the panel reads the next click
// itself (Panel.tapped).
func repoint(fyne.Window, *pointer, float32) bool { return false }
