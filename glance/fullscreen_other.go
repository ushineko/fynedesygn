//go:build !windows

package glance

import "context"

// fullscreenSupported is false where the window manager already covers a
// glance window with an active full-screen one (KWin), or nothing is done.
const fullscreenSupported = false

// watchFullscreen does nothing here; see fullscreen.go.
func watchFullscreen(context.Context, *Window) {}
