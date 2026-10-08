package glance

// edges is a rectangle by its four edges, in screen pixels, right and bottom
// exclusive: the shape Windows gives a window's rectangle and a monitor's
// work area (spec 058). It is platform-free so the arithmetic is tested
// everywhere; only Windows asks for it.
type edges struct {
	left, top, right, bottom int32
}

// snapReach is how close, in pixels at 96 DPI, an edge of a dragged window
// comes to an edge of the work area before it is pulled onto it. KWin's
// default border snap zone is 10 px, and a glance window on Linux is dragged
// by KWin, so the two desktops feel the same.
const snapReach = 10

// snapDistance is snapReach at a monitor's DPI, so the pull is the same
// physical distance on a scaled screen. A DPI the system did not give is
// taken as 96.
func snapDistance(dpi uint32) int32 {
	if dpi == 0 {
		dpi = 96
	}
	return int32((snapReach*dpi + 48) / 96)
}

/*
snapToWork is where a window being dragged to r should go, given the work area
it is over and the snap distance.

Each axis is decided alone: an edge of the window inside the work area and
within dist of the same edge is moved onto it, the window keeping its size, so a
window near a corner snaps on both axes. Near the left and the right at once
(a window as wide as the work area, near enough), the left wins, and the top
over the bottom, so the window's start is the edge that stays put. Anywhere
else r is returned as it is and the window follows the pointer exactly.

**Only from inside.** An edge already past the work area's is left where the
pointer put it: pushing on through an edge takes the window part way off the
screen, or on to the next monitor, as the person dragging it meant. Pulling it
back in from outside, as the first version did, fought every drag across a
monitor's edge.
*/
func snapToWork(r, work edges, dist int32) edges {
	shift := func(lo, hi, workLo, workHi int32) int32 {
		switch {
		case lo >= workLo && lo-workLo <= dist:
			return workLo - lo
		case hi <= workHi && workHi-hi <= dist:
			return workHi - hi
		default:
			return 0
		}
	}
	dx := shift(r.left, r.right, work.left, work.right)
	dy := shift(r.top, r.bottom, work.top, work.bottom)
	return edges{left: r.left + dx, top: r.top + dy, right: r.right + dx, bottom: r.bottom + dy}
}

// point is a position in screen pixels: the pointer, or an offset from a
// window's top-left.
type point struct {
	x, y int32
}

// freeRect is r moved, keeping its size, so the point it was grabbed by
// (grabbed, an offset from its top-left) is under the pointer: where a drag
// puts the window before any snapping (spec 058).
func freeRect(r edges, pointer, grabbed point) edges {
	left, top := pointer.x-grabbed.x, pointer.y-grabbed.y
	return edges{left: left, top: top, right: left + (r.right - r.left), bottom: top + (r.bottom - r.top)}
}
