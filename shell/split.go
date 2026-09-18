package shell

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"

	"github.com/ushineko/fynedesygn/settings"
)

/*
Dividers the user can drag, whose positions outlive the sections that draw them.

Fyne's split has an Offset it can be set to and read from, and no callback: the
divider writes the field and nothing is told. A section is rebuilt on every
refresh, every selection and every status-driven redraw, so a split a section
owns is discarded and replaced several times a minute -- and with it the position
the user dragged it to.

So the shell holds the positions. It reads each live split just before the
content that holds it is replaced, which is the last moment that split exists,
and keeps the result in the settings file under one section.

See spec 012.
*/

// SplitsKey is where the dragged positions live in the settings store.
const SplitsKey = settings.Prefix + "splits"

// NavSplitKey is the shell's own divider, between the section list and the
// content. It goes through the same machinery as a program's, so the width of
// the navigation is something the user can set and keep.
const NavSplitKey = "nav"

/*
VSplit puts top over bottom with a bar between them the user can drag.

key names the divider within the program ("output", "preview"); offset is where
it sits before anyone has moved it. The position is remembered across a section
rebuild and across runs.

Two splits with the same key share one position, on purpose: a log pane under
every section should not jump when the section changes.
*/
func (s *Shell) VSplit(key string, offset float64, top, bottom fyne.CanvasObject) fyne.CanvasObject {
	return s.split(container.NewVSplit(top, bottom), key, offset)
}

// HSplit is VSplit laid out side by side.
func (s *Shell) HSplit(key string, offset float64, leading, trailing fyne.CanvasObject) fyne.CanvasObject {
	return s.split(container.NewHSplit(leading, trailing), key, offset)
}

// split positions a new divider and registers it as the live one for its key.
func (s *Shell) split(sp *container.Split, key string, offset float64) fyne.CanvasObject {
	if got, ok := s.splitPos[key]; ok && usableOffset(got) {
		offset = got
	}
	sp.SetOffset(offset)
	if s.splits == nil {
		s.splits = map[string]*container.Split{}
	}
	// The previous split for this key is read before it is forgotten: a rebuild
	// that happened without going through swap would otherwise lose the drag.
	s.readSplit(key)
	s.splits[key] = sp
	return sp
}

/*
usableOffset rejects a position that would open the window with one pane
invisible.

Zero and one are what a split reports when a pane has been dragged shut. Storing
one is fine; obeying it on the next run is not, because the pane that is gone is
often the one holding the control that would bring it back.
*/
func usableOffset(v float64) bool { return v > 0 && v < 1 }

// readSplit takes one live split's position into the remembered map.
func (s *Shell) readSplit(key string) {
	sp, ok := s.splits[key]
	if !ok || sp == nil {
		return
	}
	if s.splitPos == nil {
		s.splitPos = map[string]float64{}
	}
	if usableOffset(sp.Offset) {
		s.splitPos[key] = sp.Offset
	}
}

/*
rememberSplits reads every live divider and writes the positions to the settings
store.

Called just before the content is replaced, and in the stop path. The store's
own write is debounced and a section that is set to what it already holds is not
a change, so calling this on every section swap costs nothing.
*/
func (s *Shell) rememberSplits() {
	for key := range s.splits {
		s.readSplit(key)
	}
	if len(s.splitPos) == 0 {
		return
	}
	if err := s.store.Set(SplitsKey, s.splitPos); err != nil {
		s.Report("Saving the divider positions", err)
	}
}

// loadSplits reads the remembered positions at start.
func (s *Shell) loadSplits() {
	s.splitPos = map[string]float64{}
	s.store.Get(SplitsKey, &s.splitPos)
}
