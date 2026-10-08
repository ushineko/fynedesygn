package glance

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Spec 058. A window dragged near an edge of the work area lands on it; one
// dragged anywhere else lands where it was dropped.
func TestADraggedWindowSnapsToTheWorkAreasEdges(t *testing.T) {
	// A 1920x1040 work area: a 1920x1080 screen with a 40 px taskbar below.
	work := edges{left: 0, top: 0, right: 1920, bottom: 1040}
	// A 260x200 window.
	at := func(x, y int32) edges { return edges{left: x, top: y, right: x + 260, bottom: y + 200} }

	cases := []struct {
		name string
		r    edges
		want edges
	}{
		{"in the middle: untouched", at(800, 400), at(800, 400)},
		{"near the left", at(7, 400), at(0, 400)},
		{"pushed past the left: goes with the pointer", at(-6, 400), at(-6, 400)},
		{"on the left edge: stays", at(0, 400), at(0, 400)},
		{"pushed past the top: goes with the pointer", at(800, -12), at(800, -12)},
		{"exactly the distance from the left", at(10, 400), at(0, 400)},
		{"just beyond the distance", at(11, 400), at(11, 400)},
		{"near the right", at(1920-260-4, 400), at(1920-260, 400)},
		{"near the top", at(800, 9), at(800, 0)},
		{"near the taskbar, not the screen", at(800, 1040-200-3), at(800, 1040-200)},
		{"near a corner, inside: both axes", at(1920-260-5, 1040-200-8), at(1920-260, 1040-200)},
		{"pushed past a corner: goes with the pointer", at(1920-260+5, 1040-200+8), at(1920-260+5, 1040-200+8)},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, snapToWork(c.r, work, 10), c.name)
	}
}

// A monitor to the left of the primary one has negative coordinates, and its
// edges snap the same way.
func TestTheWorkAreaOfASecondMonitorSnapsInItsOwnCoordinates(t *testing.T) {
	work := edges{left: -2560, top: -200, right: 0, bottom: 1240}
	r := edges{left: -2556, top: -195, right: -2296, bottom: 5}
	assert.Equal(t, edges{left: -2560, top: -200, right: -2300, bottom: 0}, snapToWork(r, work, 10))

	r = edges{left: -264, top: 600, right: -4, bottom: 800}
	assert.Equal(t, edges{left: -260, top: 600, right: 0, bottom: 800}, snapToWork(r, work, 10),
		"the right edge of the left monitor is its own edge, not the primary's")
}

// A window as wide as the work area, near both sides, keeps its left edge.
func TestAWindowNearBothSidesSnapsItsStart(t *testing.T) {
	work := edges{left: 0, top: 0, right: 300, bottom: 300}
	r := edges{left: 4, top: 100, right: 298, bottom: 200}
	assert.Equal(t, edges{left: 0, top: 100, right: 294, bottom: 200}, snapToWork(r, work, 10))
}

// The pull is the same physical distance on a scaled screen.
func TestTheSnapDistanceScalesWithTheMonitorsDPI(t *testing.T) {
	assert.Equal(t, int32(10), snapDistance(96))
	assert.Equal(t, int32(15), snapDistance(144), "150 %")
	assert.Equal(t, int32(20), snapDistance(192), "200 %")
	assert.Equal(t, int32(13), snapDistance(120), "125 %")
	assert.Equal(t, int32(10), snapDistance(0), "no DPI given is 96")
}

/*
The window's free place comes from the pointer and the point it was grabbed
by, so snapping it cannot stick: a window on the top edge whose pointer has
moved 20 px down comes off it, where snapping the rectangle Windows proposed,
built from the snapped one plus a few pixels at a time, held it there.
*/
func TestASnappedWindowFollowsThePointerOffTheEdge(t *testing.T) {
	work := edges{0, 0, 2560, 1400}
	size := edges{0, 0, 260, 180}
	grabbed := point{60, 40}
	const dist = 15

	// Grabbed at (60, 40) and carried to 4 px below the top edge: snapped.
	at := freeRect(size, point{760, 44}, grabbed)
	assert.Equal(t, edges{700, 4, 960, 184}, at, "free place under the pointer")
	assert.Equal(t, int32(0), snapToWork(at, work, dist).top, "4 px below the top snaps onto it")

	// The same drag, 20 px lower: the free place is 24 px down, past the snap
	// distance, and the window is there, not on the edge.
	at = freeRect(size, point{760, 64}, grabbed)
	assert.Equal(t, int32(24), snapToWork(at, work, dist).top, "the window stayed on the edge")

	// The size is the window's whatever rectangle it is given.
	assert.Equal(t, edges{-60, -40, 200, 140}, freeRect(edges{500, 500, 760, 680}, point{0, 0}, grabbed))
}

/*
Dragging across to the next monitor: the window catches on the shared edge
from inside, as on any edge, and goes on with the pointer as soon as it is
pushed past it. The first version pulled an edge back in from outside too, and
a window had to be thrown across the border to get over it.
*/
func TestAWindowDraggedAcrossToTheNextMonitorGoesOnPastTheEdge(t *testing.T) {
	left := edges{left: 0, top: 0, right: 1920, bottom: 1040}
	at := func(x int32) edges { return edges{left: x, top: 400, right: x + 260, bottom: 600} }

	assert.Equal(t, at(1920-260), snapToWork(at(1920-260-8), left, 10), "8 px short of the shared edge: on it")
	assert.Equal(t, at(1920-260+6), snapToWork(at(1920-260+6), left, 10), "6 px over it: with the pointer")
	assert.Equal(t, at(1920-260+120), snapToWork(at(1920-260+120), left, 10), "well over it: with the pointer")
}
