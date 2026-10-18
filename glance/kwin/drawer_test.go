package kwin_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ushineko/fynedesygn/glance/kwin"
)

var panel = kwin.Target{Class: "io.ushineko.example", Caption: "Monitor"}

// Spec 068 R9. The script moves the window back by what it grew, in the
// handler, and leaves a move or a resize by hand alone.
func TestTheHoldScriptKeepsTheRightEdge(t *testing.T) {
	s := kwin.HoldRightEdgeScript(panel)

	assert.Contains(t, s, `const target = { class: "io.ushineko.example", caption: "Monitor" }`)
	assert.Contains(t, s, "w.frameGeometryChanged.connect(function(old)")
	assert.Contains(t, s, "const x = old.x + old.width - g.width;")
	assert.Contains(t, s, "x: x, y: g.y, width: g.width, height: g.height")
	assert.Contains(t, s, "w.move || w.resize", "a window dragged or resized by hand was moved back")
	assert.Contains(t, s, "|| Math.abs(g.width - old.width) < 0.5)", "a change that kept the width was acted on")
	assert.Contains(t, s, "if (Math.abs(g.x - x) < 0.5) { return; }", "an edge already held was set again")
	assert.Contains(t, s, "workspace.windowAdded.connect(hold)", "a panel restarted while it is loaded was not held")
	assert.NotContains(t, s, "Qt.rect", "Qt.rect does not exist in a KWin script")
}

// R10. The room is two numbers, measured in the area windows are placed in.
func TestTheRoomScriptReportsBothSides(t *testing.T) {
	s := kwin.ReportRoomScript(panel, "org.ushineko.Example", "/Window", "org.ushineko.Window", "Room")

	assert.Contains(t, s, "workspace.clientArea(KWin.PlacementArea, w)")
	assert.Contains(t, s, `callDBus("org.ushineko.Example", "/Window", "org.ushineko.Window", "Room",`)
	assert.Contains(t, s, "Math.round(g.x - a.x), Math.round(a.x + a.width - g.x - g.width)")
	assert.Contains(t, s, "if (matches(w))")
}

// R22. The keyboard goes back only from the panel, to a window a person
// could have been typing in, on this desktop.
func TestTheYieldScriptActsOnlyFromThePanel(t *testing.T) {
	s := kwin.YieldFocusScript(panel)

	assert.Contains(t, s, "if (active && matches(active))",
		"a panel that never had the keyboard would bring a window forward")
	assert.Contains(t, s, "workspace.stackingOrder")
	assert.Contains(t, s, "for (let i = order.length - 1; i >= 0; i--)", "the order is walked from the top")
	assert.Contains(t, s, "w.resourceClass == target.class", "the keyboard was handed to the program's own window")
	for _, skip := range []string{"!w.normalWindow", "w.minimized", "w.skipTaskbar", "!w.wantsInput"} {
		assert.Contains(t, s, skip)
	}
	assert.Contains(t, s, "w.onAllDesktops || w.desktops.indexOf(workspace.currentDesktop) >= 0")
	assert.NotContains(t, s, "onCurrentDesktop", "Plasma 6 has no onCurrentDesktop; it excludes every window")
	assert.Contains(t, s, "workspace.activeWindow = w;")
}

// The strings in each script are quoted, as in the position scripts.
func TestADrawerScriptCannotBeEscapedByItsStrings(t *testing.T) {
	bad := `x"; workspace.windowList()[0].closeWindow(); //`
	target := kwin.Target{Class: bad, Caption: bad}

	for name, s := range map[string]string{
		"hold":  kwin.HoldRightEdgeScript(target),
		"room":  kwin.ReportRoomScript(target, bad, bad, bad, bad),
		"yield": kwin.YieldFocusScript(target),
	} {
		assert.NotContains(t, s, `x"; workspace`, name)
		assert.Contains(t, s, `x\"; workspace`, name)
	}
}

// Spec 071. Plasma 6.8 moves x by half of what the window grew, in the same
// change; a script that waited for x to stay put never acted there.
func TestTheHoldScriptActsWhereTheCompositorMovedTheWindow(t *testing.T) {
	s := kwin.HoldRightEdgeScript(panel)

	assert.NotContains(t, s, "g.x != old.x", "a widening the compositor also moved was left alone")
	// The script's own move reports a width a few ulps from the one set;
	// compared exactly, the move was undone in its own handler.
	assert.NotContains(t, s, "g.width == old.width", "the script's own move was read as a widening")
	assert.NotContains(t, s, "g.x == x", "a held edge a few ulps off was set again")
}
