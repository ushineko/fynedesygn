//go:build !cgo || !linux || android || wasm || js || mobile || test_web_driver

package glance

// setWindowOpacity reports that there is no X11 window to write the property
// on. See opacity_x11.go.
func setWindowOpacity(uintptr, float32) bool { return false }
