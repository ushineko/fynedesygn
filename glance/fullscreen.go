package glance

import (
	"context"
	"sync"
)

/*
A glance window can stand aside while a full-screen window is in front of it
(spec 059).

It is always on top, and on Windows that keeps it over a borderless
full-screen window: a browser after F11, most current games, a video player.
An exclusive full-screen game covers it anyway. On KWin an active full-screen
window is in a layer above keep-above windows, so the window manager already
does this, and on every platform but Windows SetHideForFullscreen does
nothing.

The decision is made here, from facts the platform gathers, so it is tested
everywhere: fullscreenInFront says whether one look at the screen shows a
full-screen window in front, and settle turns a run of looks into a hide or a
show.
*/

// foreground is what one look at the window in front found.
type foreground struct {
	// Own is the glance window, or another window of this program: its menu
	// or its preferences in front is not something to stand aside for.
	Own bool
	// Shell is the desktop or the taskbar.
	Shell bool
	// Hidden is a window that is not on the screen: invisible, minimised, or
	// cloaked (on another virtual desktop, or a suspended app).
	Hidden bool
	// Rect is the window's rectangle (GetWindowRect), Frame the frame DWM
	// draws (DWMWA_EXTENDED_FRAME_BOUNDS, zero where it gives none), and
	// Monitor the full rectangle of the monitor the window is on, taskbar
	// included.
	Rect, Frame, Monitor edges
	// SameMonitor is the window being on the glance window's monitor: a film
	// full-screen on the other monitor leaves this one's panel alone.
	SameMonitor bool
	// Maximised and Captioned are the window being maximised and having a
	// titlebar.
	Maximised, Captioned bool
}

/*
fullscreenInFront is whether the window in front is a full-screen window over
the glance window's monitor.

Full screen is covering the whole monitor, taskbar and all. Only geometry decides:
windowed and borderless full screen, which is what browsers (F11, a web
player's full-screen button) and most current games use, is a window sized
to the monitor and nothing else marks it. Windows' own SHQueryUserNotificationState
reports a full-screen app for exclusive Direct3D and presentation mode, and
misses exactly these, so it is not consulted. **A maximised
window with a titlebar is not**, though it can cover the monitor: where the
taskbar hides itself, the work area is the whole monitor and a maximised
window's frame reaches past every edge. A full-screen browser drops its
titlebar, which is what tells the two apart.
*/
func fullscreenInFront(f foreground) bool {
	if f.Own || f.Shell || f.Hidden || !f.SameMonitor {
		return false
	}
	if !contains(f.Rect, f.Monitor) && !contains(f.Frame, f.Monitor) {
		return false
	}
	return !f.Maximised || !f.Captioned
}

// settlePolls is how many looks in a row it takes to hide or show. One look
// would flicker the window through a transition -- a game resizing itself, a
// browser on its way into full screen passes through other sizes -- and two at
// the watch's interval is still under a second.
const settlePolls = 2

// settle is the hidden state, changed only by settlePolls looks in a row that
// agree.
type settle struct {
	hidden bool
	want   bool
	seen   int
}

// next takes one look and says whether the window should now be hidden, and
// whether that is a change.
func (s *settle) next(fullscreen bool) (hidden, changed bool) {
	if fullscreen == s.hidden {
		s.seen = 0
		return s.hidden, false
	}
	if fullscreen != s.want || s.seen == 0 {
		s.want, s.seen = fullscreen, 0
	}
	s.seen++
	if s.seen < settlePolls {
		return s.hidden, false
	}
	s.hidden, s.seen = fullscreen, 0
	return s.hidden, true
}

// fullscreenWatch runs the platform's watch while it is wanted.
type fullscreenWatch struct {
	mu     sync.Mutex
	cancel context.CancelFunc
	done   chan struct{}
}

/*
SetHideForFullscreen asks the window to hide while a full-screen window is in
front on its monitor, and to show again, without taking the focus, when that
stops. It is off until asked for and can be turned on and off at any time;
turned off, a window it hid is shown.

Windows only: elsewhere it does nothing, and on KWin the window manager
already covers a glance window with an active full-screen one. The window is
hidden and shown by the system, not through Fyne: Fyne still thinks of it as
shown, which is what a program that called Show wants, and its place and its
always-on-top are untouched. A program that hides the window itself while
this is on should turn this off first, or the watch may show it again.
*/
func (w *Window) SetHideForFullscreen(on bool) {
	w.fullscreen.mu.Lock()
	defer w.fullscreen.mu.Unlock()
	switch {
	case on && w.fullscreen.cancel == nil && fullscreenSupported:
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan struct{})
		w.fullscreen.cancel, w.fullscreen.done = cancel, done
		go func() {
			defer close(done)
			watchFullscreen(ctx, w)
		}()
	case !on && w.fullscreen.cancel != nil:
		w.fullscreen.cancel()
		<-w.fullscreen.done
		w.fullscreen.cancel, w.fullscreen.done = nil, nil
	}
}

/*
contains is whether r covers the whole of m. Not equality: a borderless or
full-screen window is often a few pixels larger than its monitor, on every
side (a game sized past the edges so no border shows, a window keeping its
invisible resize borders), and that is full screen too.

Two rectangles are asked because neither is always right. GetWindowRect
includes the invisible borders a resizable window keeps, so on its own it
would call a window full screen whose visible frame stops short; the frame DWM
draws leaves them out, but DWM gives none for some windows. Either one
covering the monitor is the window covering it.
*/
func contains(r, m edges) bool {
	if r == (edges{}) {
		return false
	}
	return r.left <= m.left && r.top <= m.top && r.right >= m.right && r.bottom >= m.bottom
}
