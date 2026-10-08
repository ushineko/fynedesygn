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

Each axis is decided alone: an edge of the window within dist of the same
edge of the work area is moved onto it, the window keeping its size, so a
window near a corner snaps on both axes. Near the left and the right at once
(a window as wide as the work area, near enough), the left wins, and the top
over the bottom, so the window's start is the edge that stays put. Anywhere
else r is returned as it is and the window follows the pointer exactly.
*/
func snapToWork(r, work edges, dist int32) edges {
	shift := func(lo, hi, workLo, workHi int32) int32 {
		switch {
		case abs32(lo-workLo) <= dist:
			return workLo - lo
		case abs32(hi-workHi) <= dist:
			return workHi - hi
		default:
			return 0
		}
	}
	dx := shift(r.left, r.right, work.left, work.right)
	dy := shift(r.top, r.bottom, work.top, work.bottom)
	return edges{left: r.left + dx, top: r.top + dy, right: r.right + dx, bottom: r.bottom + dy}
}

func abs32(v int32) int32 {
	if v < 0 {
		return -v
	}
	return v
}
