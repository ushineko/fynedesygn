//go:build cgo && linux && !android && !test_web_driver && (x11 || !wayland)

package dragout

/*
#cgo pkg-config: x11
#include <stdlib.h>
#include "x11_linux.h"
*/
import "C"

import (
	"errors"
	"fmt"
	"runtime"
	"unsafe"

	"fyne.io/fyne/v2"
	"github.com/go-gl/glfw/v3.4/glfw"
)

// haveX11 is true in a build whose GLFW can run on X11.
const haveX11 = true

var errNoGrab = errors.New("dragout: another program holds the pointer")

// x11Start takes the pointer from GLFW and runs the drag on a thread of its
// own, so the UI keeps drawing while the user is over another window.
func x11Start(_ fyne.Window, paths []string) error {
	// GLFW's connection holds the implicit grab of the press that started
	// the gesture; the drag's connection cannot take the pointer until it
	// is let go.
	C.dragout_x11_release_grab((*C.Display)(unsafe.Pointer(glfw.GetX11Display())))

	data := uriList(paths)
	cdata := C.CString(data)
	defer C.free(unsafe.Pointer(cdata))
	var code C.int
	d := C.dragout_x11_begin(cdata, C.size_t(len(data)), &code)
	switch code {
	case C.DRAGOUT_X11_OK:
	case C.DRAGOUT_X11_NO_BUTTON:
		return nil // released before the drag began: a click, not a failure
	case C.DRAGOUT_X11_NO_DISPLAY:
		return ErrUnsupported
	case C.DRAGOUT_X11_NO_GRAB:
		return errNoGrab
	default:
		return fmt.Errorf("dragout: x11 error %d", int(code))
	}
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		C.dragout_x11_run(d)
	}()
	return nil
}
