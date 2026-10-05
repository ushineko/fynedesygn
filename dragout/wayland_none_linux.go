//go:build cgo && linux && !android && !test_web_driver && x11 && !wayland

package dragout

import "fyne.io/fyne/v2"

// haveWayland is false in a build made with -tags x11, which has no Wayland
// support in GLFW either.
const haveWayland = false

func waylandPrepare() error                    { return ErrUnsupported }
func waylandStart(fyne.Window, []string) error { return ErrUnsupported }
