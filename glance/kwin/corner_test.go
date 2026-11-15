package kwin_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ushineko/fynedesygn/glance/kwin"
)

// Spec 081 R9. The corner script holds the bottom edge and the side named,
// with the hold-edge scripts' guards, and keeps the window in the work area.
func TestHoldCornerScript(t *testing.T) {
	for name, c := range map[string]struct {
		right   bool
		edge, x string
	}{
		"right": {true, "edge = old.x + old.width;", "let x = edge - g.width, y = bottom - g.height;"},
		"left":  {false, "edge = old.x;", "let x = edge, y = bottom - g.height;"},
	} {
		s := kwin.HoldCornerScript(panel, c.right)
		assert.Contains(t, s, `const target = { class: "io.ushineko.example", caption: "Monitor" }`, name)
		assert.Contains(t, s, c.edge, "%s: the side is not the one held", name)
		assert.Contains(t, s, "bottom = old.y + old.height;", "%s: the bottom is not held", name)
		assert.Contains(t, s, c.x, name)
		assert.Contains(t, s, "if (w.move || w.resize) { edge = null; home = null; return; }", "%s: a drag was undone, or the place before it kept", name)
		// Found on the desk: KWin put the square on whole device pixels, and
		// the panel unfolded half a pixel from where it had been.
		assert.Contains(t, s, "if (g.width * g.height < old.width * old.height) {\n                home = { edge: edge, bottom: bottom };", "%s: the corner shrunk from is not kept", name)
		assert.Contains(t, s, "edge = home.edge;\n                bottom = home.bottom;\n                home = null;", "%s: the window does not unfold from the corner it shrank from", name)
		assert.Contains(t, s, "Math.abs(g.height - old.height) < 0.5", "%s: a change of height alone is not a change of size", name)
		assert.Contains(t, s, "workspace.clientArea(KWin.PlacementArea, w)", "%s: the work area is not read", name)
		assert.Contains(t, s, "if (x + g.width > a.x + a.width + 0.5) { x = a.x + a.width - g.width; }", "%s: the window may leave the screen sideways", name)
		assert.Contains(t, s, "if (x < a.x - 0.5) { x = a.x; }", name)
		assert.Contains(t, s, "if (y < a.y - 0.5) { y = a.y; }", "%s: the window may leave the screen upwards", name)
		assert.Contains(t, s, "if (y + g.height > a.y + a.height + 0.5) { y = a.y + a.height - g.height; }", name)
		assert.NotContains(t, s, "print(", "%s: a debugging print was left in", name)
		assert.Contains(t, s, "if (edge === null || Date.now() - at > heldFor) { return; }", "%s: a put-back is not undone", name)
		assert.Equal(t, 1, strings.Count(s, "w.frameGeometry = {"), name)
		assert.Contains(t, s, "workspace.windowAdded.connect(hold)", name)
		assert.NotContains(t, s, "Qt.rect", name)
		assert.NotContains(t, s, "g.width == old.width", name)
	}
}

// The corner script's target is quoted as the others' are.
func TestTheCornerScriptCannotBeEscapedByItsStrings(t *testing.T) {
	bad := `x"; workspace.windowList()[0].closeWindow(); //`
	s := kwin.HoldCornerScript(kwin.Target{Class: bad, Caption: bad}, true)
	assert.NotContains(t, s, `x"; workspace`)
	assert.Contains(t, s, `x\"; workspace`)
}
