//go:build cgo && linux && !android && !test_web_driver && (wayland || !x11)

package dragout

/*
#cgo pkg-config: wayland-client
#include <stdlib.h>
#include "wayland_linux.h"
*/
import "C"

import (
	"errors"
	"fmt"
	"unsafe"

	"fyne.io/fyne/v2"
	"github.com/go-gl/glfw/v3.4/glfw"

	"github.com/ushineko/fynedesygn/internal/glfwwin"
)

// haveWayland is true in a build whose GLFW can run on Wayland.
const haveWayland = true

var (
	errNoSeat   = errors.New("dragout: the compositor offers no seat with a pointer and a data device")
	errNoSerial = errors.New("dragout: no button press to start the drag from")
)

// waylandDisplay is GLFW's connection, the one Fyne's windows live on.
func waylandDisplay() *C.struct_wl_display {
	return (*C.struct_wl_display)(unsafe.Pointer(glfw.GetWaylandDisplay()))
}

// waylandPrepare binds the seat, pointer and data device, once. It has to
// happen before the press a drag starts from, or the pointer never hears of
// that press.
func waylandPrepare() error {
	return waylandErr(C.dragout_wl_prepare(waylandDisplay()))
}

// waylandStart starts a drag of paths from w's surface.
func waylandStart(w fyne.Window, paths []string) error {
	if err := waylandPrepare(); err != nil {
		return err
	}
	view := glfwwin.ViewportOf(w)
	if view == nil {
		return ErrUnsupported
	}
	surface := (*C.struct_wl_surface)(unsafe.Pointer(view.GetWaylandWindow()))
	data := uriList(paths)
	cdata := C.CString(data)
	defer C.free(unsafe.Pointer(cdata))
	return waylandErr(C.dragout_wl_start(waylandDisplay(), surface, cdata, C.size_t(len(data))))
}

func waylandErr(code C.int) error {
	switch code {
	case C.DRAGOUT_OK:
		return nil
	case C.DRAGOUT_NO_SEAT:
		return errNoSeat
	case C.DRAGOUT_NO_SERIAL:
		return errNoSerial
	default:
		return fmt.Errorf("dragout: wayland error %d", int(code))
	}
}
