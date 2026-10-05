//go:build cgo && linux && !android && !test_web_driver && wayland && !x11

package dragout

import "fyne.io/fyne/v2"

// haveX11 is false in a build made with -tags wayland, which has no X11
// support in GLFW either.
const haveX11 = false

func x11Start(fyne.Window, []string) error { return ErrUnsupported }
