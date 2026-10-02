package shell

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
)

/*
VScroll is a vertical scroller whose position survives the section being
rebuilt.

A section with affixed controls puts its body in a scroller of its own under a
Border, so the controls stay put while the body moves. The shell keeps the
content pane's offset on a rebuild in place, but the section's own scroller is
a new object every build, and every operation rebuilds the section (OK
invalidates, the busy tail regates). A checkbox pressed halfway down a long
page threw the page back to the top under the pointer.

So the shell holds the live scroller for each name and gives its offset to the
one that replaces it, which is what it already does for split dividers. Unlike
a divider the offset is not stored: navigating to the section, by the list,
Select or a Tabs strip, starts at the top, as the content pane does.

name identifies the scroller within the program ("tweaks/Ships"). Two scrollers
built under one name in one build share nothing useful; give each its own.

The offset is assigned rather than scrolled to: ScrollToOffset on a scroller
that has not been laid out clamps against a zero size and throws the value
away, while an assigned offset is clamped by the first real layout. See spec
054.
*/
func (s *Shell) VScroll(name string, content fyne.CanvasObject) *container.Scroll {
	sc := container.NewVScroll(content)
	if old, ok := s.scrolls[name]; ok && old != nil {
		sc.Offset = old.Offset
	}
	if s.scrolls == nil {
		s.scrolls = map[string]*container.Scroll{}
	}
	s.scrolls[name] = sc
	return sc
}
