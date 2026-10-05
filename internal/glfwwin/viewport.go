//go:build cgo && !wasm && !js && !android && !ios && !mobile && !test_web_driver

// Package glfwwin reaches the GLFW window behind a Fyne desktop window, for
// the components that need what Fyne does not expose: widgets (quirk 42) and
// dragout (quirks 44 and 45).
package glfwwin

import (
	"reflect"
	"unsafe"

	"fyne.io/fyne/v2"
	"github.com/go-gl/glfw/v3.4/glfw"
)

// glfwWindowType is the type of the field ViewportOf reads.
var glfwWindowType = reflect.TypeOf((*glfw.Window)(nil))

/*
ViewportOf is the GLFW window behind a Fyne window, or nil.

Read from the unexported field `viewport` of Fyne's desktop window, by
reflection, because no API returns it (driver.NativeWindow gives the X11 or
Wayland handle, which GLFW cannot be asked about). The field's name and type
are both checked, so a window that is not Fyne's GLFW window, or a Fyne that
changed it, is nil and not a crash.
*/
func ViewportOf(w fyne.Window) *glfw.Window {
	v := reflect.ValueOf(w)
	for v.Kind() == reflect.Interface || v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct || !v.CanAddr() {
		return nil
	}
	f := v.FieldByName("viewport")
	if !f.IsValid() || f.Type() != glfwWindowType {
		return nil
	}
	return *(**glfw.Window)(unsafe.Pointer(f.UnsafeAddr())) //nolint:gosec // see the comment above
}
