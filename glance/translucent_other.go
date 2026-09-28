//go:build !cgo || wasm || js || android || ios || mobile || test_web_driver

package glance

// grantTranslucent reports that a translucent window is not available. A build
// without cgo has no GLFW to ask, and the other platforms here have no desktop
// behind the window to show through it.
func grantTranslucent() bool { return false }

// clearTranslucent has nothing to clear on a build with no GLFW.
func clearTranslucent() {}
