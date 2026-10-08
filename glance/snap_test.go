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
		{"past the left", at(-6, 400), at(0, 400)},
		{"exactly the distance from the left", at(10, 400), at(0, 400)},
		{"just beyond the distance", at(11, 400), at(11, 400)},
		{"near the right", at(1920-260-4, 400), at(1920-260, 400)},
		{"near the top", at(800, 9), at(800, 0)},
		{"near the taskbar, not the screen", at(800, 1040-200-3), at(800, 1040-200)},
		{"near a corner: both axes", at(1920-260+5, 1040-200+8), at(1920-260, 1040-200)},
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

	r = edges{left: -265, top: 600, right: -5, bottom: 800}
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
