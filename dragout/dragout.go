/*
Package dragout lets the user drag files out of a Fyne window into another
application: a file manager, a browser's upload field, a chat window.

Fyne receives drops (`Window.SetOnDropped`) but cannot start one, and GLFW
has no drag-source API on any platform (quirk 44). This package starts the
operating system's own drag, behind build tags: the core Wayland protocol
on Wayland, XDND on X11. A build or session without a backend reports
[Supported] false, and a [Source] there is a plain container.

The drag offers files and asks for a copy, so the drop target copies the
file and the original is never moved. The package does no other file I/O.
*/
package dragout

import (
	"errors"
	"math"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

// Threshold is how far, in Fyne units, the pointer must travel with the
// button held before a drag leaves the window. It is a little more than
// Fyne's own two-pixel drag threshold, the way GTK and Qt ask for about eight
// before they start one, so a click with a shaky hand stays a click.
const Threshold = 8

// ErrUnsupported is what [Source.OnFailed] receives when this build or
// session has no way to start a drag. A [Source] does not report it for every
// gesture: it reports it once, on the first one.
var ErrUnsupported = errors.New("dragout: dragging out of the window is not supported here")

// backend is the platform's drag source. Both methods run on the UI thread.
type backend interface {
	// supports reports whether a drag can start from this window.
	supports(w fyne.Window) bool
	// prepare readies w for a drag before the press it will start from.
	prepare(w fyne.Window)
	// start begins a drag of paths (absolute, existing regular files) from
	// w. The platform owns the pointer once it returns nil.
	start(w fyne.Window, paths []string) error
	// release tells Fyne the button it saw pressed is up, because the
	// platform took the pointer before the release (quirk 45).
	release(w fyne.Window)
}

// platform is this build's backend; tests replace it.
var platform = newBackend()

// Supported reports whether a drag can start from any of this application's
// windows: false under the test driver, in a build without cgo, and on a
// platform with no backend yet.
func Supported() bool {
	app := fyne.CurrentApp()
	if app == nil {
		return false
	}
	for _, w := range app.Driver().AllWindows() {
		if platform.supports(w) {
			return true
		}
	}
	return false
}

/*
Source wraps a canvas object so that dragging it drags files out of the
window.

The files are asked for when the drag starts, not when the Source is made,
so a pane that shows one file at a time passes a function that returns the
current one. Taps, hovers and right-clicks still reach the wrapped object;
only a drag is taken.
*/
type Source struct {
	widget.BaseWidget

	// OnFailed is called on the UI thread when a drag could not start. Nil
	// ignores failures. The library does not log.
	OnFailed func(error)

	content fyne.CanvasObject
	paths   func() []string

	// travelled is how far this gesture has moved; started is whether it
	// has already left the window, so one gesture is one drag.
	travelled float32
	started   bool
	// toldUnsupported is set once OnFailed has had ErrUnsupported.
	toldUnsupported bool
}

// New wraps content so that dragging it drags the files paths returns. A
// paths that returns nil or nothing usable means there is nothing to drag,
// and the gesture does nothing.
func New(content fyne.CanvasObject, paths func() []string) *Source {
	s := &Source{content: content, paths: paths}
	s.ExtendBaseWidget(s)
	return s
}

// CreateRenderer implements fyne.Widget.
func (s *Source) CreateRenderer() fyne.WidgetRenderer {
	s.prepare()
	return widget.NewSimpleRenderer(s.content)
}

/*
MouseIn implements desktop.Hoverable, and readies the platform for a drag
before the press it will start from.

Wayland needs it: the press's serial is only seen by a pointer bound before
the press (quirk 44). CreateRenderer covers a Source built into a window
already on screen; this covers one built before its window was shown. A
hoverable object inside content still gets its own hover events: Fyne sends
them to the innermost hoverable under the pointer.
*/
func (s *Source) MouseIn(*desktop.MouseEvent) { s.prepare() }

// MouseMoved implements desktop.Hoverable.
func (s *Source) MouseMoved(*desktop.MouseEvent) {}

// MouseOut implements desktop.Hoverable.
func (s *Source) MouseOut() {}

func (s *Source) prepare() {
	if w := s.window(); w != nil {
		platform.prepare(w)
	}
}

// Dragged implements fyne.Draggable. Once the gesture has travelled
// [Threshold], the drag leaves the window.
func (s *Source) Dragged(e *fyne.DragEvent) {
	if s.started {
		return
	}
	s.travelled += float32(math.Hypot(float64(e.Dragged.DX), float64(e.Dragged.DY)))
	if s.travelled < Threshold {
		return
	}
	s.started = true

	w := s.window()
	if w == nil || !platform.supports(w) {
		if !s.toldUnsupported {
			s.toldUnsupported = true
			s.fail(ErrUnsupported)
		}
		return
	}
	paths := usable(s.paths())
	if len(paths) == 0 {
		return
	}
	if err := platform.start(w, paths); err != nil {
		s.fail(err)
		return
	}
	// Queued, not called: Fyne is inside this Dragged call and marks the
	// drag as started after it returns, which would undo a release sent now.
	fyne.Do(func() { platform.release(w) })
}

// DragEnd implements fyne.Draggable.
func (s *Source) DragEnd() {
	s.travelled = 0
	s.started = false
}

func (s *Source) fail(err error) {
	if s.OnFailed != nil {
		s.OnFailed(err)
	}
}

// window is the window the Source is drawn in, or nil before it is shown.
func (s *Source) window() fyne.Window {
	app := fyne.CurrentApp()
	if app == nil {
		return nil
	}
	c := app.Driver().CanvasForObject(s)
	if c == nil {
		return nil
	}
	for _, w := range app.Driver().AllWindows() {
		if w.Canvas() == c {
			return w
		}
	}
	return nil
}

// usable keeps the absolute paths that name an existing regular file now,
// following symbolic links, in the order given.
func usable(paths []string) []string {
	var out []string
	for _, p := range paths {
		if !filepath.IsAbs(p) {
			continue
		}
		if fi, err := os.Stat(p); err == nil && fi.Mode().IsRegular() {
			out = append(out, p)
		}
	}
	return out
}

// uriList is paths as a text/uri-list (RFC 2483): file URIs (RFC 8089),
// percent-encoded, each line ended by CRLF.
func uriList(paths []string) string {
	var b strings.Builder
	for _, p := range paths {
		u := url.URL{Scheme: "file", Path: filepath.ToSlash(p)}
		b.WriteString(u.String())
		b.WriteString("\r\n")
	}
	return b.String()
}
