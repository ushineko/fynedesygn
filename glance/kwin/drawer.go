package kwin

import "fmt"

/*
HoldRightEdgeScript is a KWin script that keeps a window's right edge where it
is when the window changes its own width, so a window made wider grows
leftwards. It is what a glance drawer on the left needs (spec 068), where the
cards must stay under the pointer that clicked them and only the compositor
moves a window.

It stays loaded, as WatchGeometryScript does, and acts in the window's
frameGeometryChanged handler: when the width changed and the user is not
moving or resizing the window by hand, it puts the right edge back where it
was. Measured on Plasma 6.7.5 on Wayland: the client's resize and the move are
logged in the same millisecond, growing and shrinking, and a window opened and
closed returns to the pixel it started on. A frame is not painted between the
two, because the move is made inside the commit that applied the resize.
Setting frameGeometry inside the handler is not dropped and needs no timer.

The x KWin gave the window is not consulted (spec 071). Plasma 6.7 left x
where it was when a client changed its own width; Plasma 6.8 keeps the
window's centre instead, moving x by half the difference in the same change.
The script once acted only when x had not moved, and on 6.8 it never acted: a
left drawer moved the cards by half its width. The right edge is worked out
from the geometry before the change, whichever x the compositor chose.

Widths and places are compared to within half a pixel, never with ==. The
script's own move changes the geometry and calls the handler again, and the
width it reports then differs from the one it set in the last digits
(553.3333333333335 against 553.3333333333333 at a scale of 1.5). Compared
exactly, that was read as the client widening again and the move was undone
inside its own handler: a drawer that opened at the centred place and closed
at the held one, and the panel crept sideways by half a drawer at each
opening.

**A put-back of the place alone is undone too** (spec 079). KWin answers a
change of focus with the geometry it knew before, and when that crosses a
resize it can put back the old x while keeping the new width: three drawers
closed at once on Plasma 6.8, then the keyboard handed back, left the cards
two drawers to the left of where they stood, 30 ms after the edge had been
held. Such a change keeps the width, so the handler above passes it by. So
the script remembers the edge it last held, and for heldFor milliseconds a
change that keeps the width but not that edge, made by nobody's hand, is put
back on it. A drag (w.move) forgets the edge at once.

**Load it only while the drawer is on the left**, and tell the drawer with
glance.Drawer.SetEdgeHeld. Loaded for a drawer on the right it would make
that one grow leftwards too: the script cannot know which side was meant.

The geometry is a plain object, as in PositionScript: Qt.rect does not exist
in a KWin script.
*/
func HoldRightEdgeScript(t Target) string {
	return t.match() + fmt.Sprintf(`const heldFor = %d;
function hold(w) {
    if (!matches(w) || !w.frameGeometryChanged) { return; }
    let edge = null, at = 0;
    w.frameGeometryChanged.connect(function(old) {
        const g = w.frameGeometry;
        if (w.move || w.resize) { edge = null; return; }
        if (Math.abs(g.width - old.width) < 0.5) {
            if (edge === null || Date.now() - at > heldFor || Math.abs(g.x + g.width - edge) < 0.5) { return; }
            w.frameGeometry = { x: edge - g.width, y: g.y, width: g.width, height: g.height };
            return;
        }
        const x = old.x + old.width - g.width;
        edge = old.x + old.width;
        at = Date.now();
        if (Math.abs(g.x - x) < 0.5) { return; }
        w.frameGeometry = { x: x, y: g.y, width: g.width, height: g.height };
    });
}`, holdPutBackMillis) + `

for (const w of workspace.windowList()) { hold(w); }
workspace.windowAdded.connect(hold);
`
}

// holdPutBackMillis is how long after holding an edge the hold script undoes
// a put-back of the place alone. The put-back seen came 31 ms after the hold;
// a person's own move in that time is a drag, which the script never undoes.
const holdPutBackMillis = 250

/*
ReportRoomScript is a KWin script that reports how much room a window has on
each side of it, within the area of its screen that windows are placed in
(panels left out), and calls it back to the program over the session bus.

It is how a glance drawer chooses its side on Plasma (glance.Drawer.Choose):
a program cannot read its window's position (quirk 33), and the screen's
usable area is the compositor's to know.

The call carries two ints, in the compositor's logical pixels, which are
Fyne's units: the room to the left of the window and the room to the right.
Either is negative when the window is past that edge.

A report is a fresh load, run and unload, like ReportGeometryScript. Measured
on Plasma 6.7.5 over the session bus: between 1.1 and 2.0 ms from the first
call to the answer, thirty times, so it is asked at each opening and nothing
is kept loaded for it.
*/
func ReportRoomScript(t Target, service, path, iface, method string) string {
	return t.match() + fmt.Sprintf(`for (const w of workspace.windowList()) {
    if (matches(w)) {
        const g = w.frameGeometry;
        const a = workspace.clientArea(KWin.PlacementArea, w);
        callDBus(%q, %q, %q, %q,
                 Math.round(g.x - a.x), Math.round(a.x + a.width - g.x - g.width));
        break;
    }
}
`, service, path, iface, method)
}

/*
YieldFocusScript is a KWin script that gives the keyboard back to the window
a person was in before they clicked a glance panel (spec 068,
glance.Drawer.SetYield).

A Wayland client cannot do it. There is no request that gives the focus
away, a request for it is refused, and a window hidden and shown takes the
focus again three milliseconds later, somewhere else on the screen (quirk
49). Measured on Plasma 6.7.5: this script took between 1.3 and 1.5 ms from
the first call to its effect, and focus-stealing prevention at its default
level did not interfere.

It acts only when the target is the active window, so a panel that never had
the keyboard brings nothing forward. It then activates the first window down
the stacking order that a person could have been typing in: a normal window,
not minimised, shown in the taskbar, taking input, on the current desktop,
and not the target's own. A window activated by a click is raised, so the one
under it is the one that was active before.

**The stacking order is not the focus history.** It names the wrong window
when the one the person was in was not the topmost normal one: a window kept
above that had the focus, or one focused without being raised. An exact
answer needs a script kept loaded to record each activation.

Setting workspace.activeWindow also raises that window. It was already the
top normal one when measured, so nothing moved.

Plasma 6 has no onCurrentDesktop on a window: it reads as undefined and
silently excluded every window. The desktops list is asked instead.

A loaded script runs once, so each yield is a load, run and unload.
*/
func YieldFocusScript(t Target) string {
	return t.match() + `const active = workspace.activeWindow;
if (active && matches(active)) {
    const order = workspace.stackingOrder;
    for (let i = order.length - 1; i >= 0; i--) {
        const w = order[i];
        if (w.resourceClass == target.class || !w.normalWindow || w.minimized ||
            w.skipTaskbar || !w.wantsInput) { continue; }
        if (!(w.onAllDesktops || w.desktops.indexOf(workspace.currentDesktop) >= 0)) { continue; }
        workspace.activeWindow = w;
        break;
    }
}
`
}
