package kwin

import "fmt"

/*
HoldCornerScript is a KWin script that keeps a bottom corner of a window
where it is when the window changes its own size, so a glance panel tucks
into a square at that corner and unfolds from it again (spec 081): the right
corner when right is set, the left otherwise.

It is the hold-edge scripts' handler with the bottom edge held as well, and
for the same reason: Plasma 6.8 keeps a window's centre when a client resizes
itself (#236), and a Wayland client cannot place its own window (quirk 33).
Widths and places are compared to within half a pixel, and a put-back of the
place alone within holdPutBackMillis of a hold is undone, as there.

**The window is kept inside the work area.** A square moved by Meta-drag into
a corner of the screen unfolds from it into a panel that would run off the
screen: the window is moved in until it does not, and the corner held from
then on is the one it is at. **Less than half a unit over the edge is left
where it is.** A panel placed against the screen's edge at a scale of 1.5
can stand a third of a unit past it, and pulled in, it tucked and unfolded a
third of a unit from where it had been (found on the desk).

A drag (w.move) or a resize by hand forgets the corner: the next change of
size holds the corner the window has then.

**The window unfolds from the corner it shrank from, not from where KWin put
the square.** KWin puts a window on whole device pixels, and a corner at
4000.333 at a scale of 1.5 is half a pixel: the square went to 3970 and not
3970.333, and the panel unfolded from it a third of a unit up and to the
left of where it had been. So the corner a window shrinks from is kept as
home, and the next growth is held to home, unless the window was moved by
hand in between.

**Load it only while tucking, tucked and unfolding**, in place of the
hold-edge script, never beside it: it holds the bottom edge, and a panel
whose cards change while it is loaded would grow upwards. Load it before
glance.Panel.Tuck and replace it with the hold-edge script after Untuck.
*/
func HoldCornerScript(t Target, right bool) string {
	edge, x := "old.x", "edge"
	if right {
		edge, x = "old.x + old.width", "edge - g.width"
	}
	return t.match() + fmt.Sprintf(`const heldFor = %d;
function hold(w) {
    if (!matches(w) || !w.frameGeometryChanged) { return; }
    let edge = null, bottom = null, at = 0, home = null;
    w.frameGeometryChanged.connect(function(old) {
        const g = w.frameGeometry;
        if (w.move || w.resize) { edge = null; home = null; return; }
        const same = Math.abs(g.width - old.width) < 0.5 && Math.abs(g.height - old.height) < 0.5;
        if (same) {
            if (edge === null || Date.now() - at > heldFor) { return; }
        } else {
            edge = %s;
            bottom = old.y + old.height;
            if (g.width * g.height < old.width * old.height) {
                home = { edge: edge, bottom: bottom };
            } else if (home !== null) {
                edge = home.edge;
                bottom = home.bottom;
                home = null;
            }
            at = Date.now();
        }
        const a = workspace.clientArea(KWin.PlacementArea, w);
        let x = %s, y = bottom - g.height;
        if (x < a.x - 0.5) { x = a.x; }
        if (x + g.width > a.x + a.width + 0.5) { x = a.x + a.width - g.width; }
        if (y < a.y - 0.5) { y = a.y; }
        if (y + g.height > a.y + a.height + 0.5) { y = a.y + a.height - g.height; }
        if (Math.abs(g.x - x) < 0.5 && Math.abs(g.y - y) < 0.5) { return; }
        w.frameGeometry = { x: x, y: y, width: g.width, height: g.height };
    });
}`, holdPutBackMillis, edge, x) + `

for (const w of workspace.windowList()) { hold(w); }
workspace.windowAdded.connect(hold);
`
}
