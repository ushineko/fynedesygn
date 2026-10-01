package kwin

import "fmt"

/*
Target is the window a script acts on: every window of Class, or the one of
them whose caption is Caption when that is given.

A class alone is not a window. On Wayland every window of an app carries the
app's ID as its resource class -- the preferences window as much as the
panel -- so a script that matched the class moved and reported both (#154).
hayami's preferences window was saved as the panel's position when dragged,
and started straight into a preferences page the compositor showed that
window 140 ms before the panel, which is the order "the first window of the
class" got wrong. The caption is the window's title, which a program sets
and knows; an empty one keeps the old meaning for a program with one window.
*/
type Target struct {
	Class   string
	Caption string
}

// match is the JavaScript that declares the target and a matches(w) for it,
// shared by every script here so they agree on what a window is.
func (t Target) match() string {
	return fmt.Sprintf(`const target = { class: %q, caption: %q };
function matches(w) {
    return w.resourceClass == target.class &&
        (target.caption == "" || w.caption == target.caption);
}
`, t.Class, t.Caption)
}

// PositionScript is a KWin script that moves the target's windows, and the
// answer to a question a Wayland client cannot answer for itself.
//
// Measured on Plasma 6: `desktop.Window.RequestPosition(900, 400)` moved a
// glance window by nothing at all. Fyne's own doc comment hedges — the request
// "may be ignored (for example Linux Wayland)" — and on this compositor it is
// not ignored sometimes, it is ignored. Qt hits the same wall from the other
// side, which is why the monitor this archetype comes from drives the
// scripting API rather than calling move().
//
// This is not the same as a Rule's position. A rule is applied when the window
// is created and pins it to one screen, so it can say "always open here" and
// nothing else. This runs against a window that is already on screen, so a
// program can put its window on the screen the pointer is on, or back where
// the user last dragged it.
//
// x and y are in the compositor's coordinates, which span every screen. The
// window's size is kept: a glance window is sized by its content and a script
// that set a size would fight Panel.Resize.
//
// The script is returned as text and the calls as data, for the reason
// ReconfigureCall gives. See OpacityScript for the load, run and unload cycle.
func PositionScript(t Target, x, y int) string {
	return t.match() + fmt.Sprintf(`for (const w of workspace.windowList()) {
    if (matches(w)) {
        const g = w.frameGeometry;
        w.frameGeometry = { x: %d, y: %d, width: g.width, height: g.height };
    }
}
`, x, y)
}

/*
PlaceScript is a KWin script that puts a window at x, y as soon as there is
one: now, if a window of the class is on screen, and otherwise the first one
that appears, once.

PositionScript moves a window that is already there, and a program that
restores its position with it has to guess how long its own window takes to
appear. hayami guessed 600 ms; a cold Fyne window on the desk took 2.15 s,
the script found nothing, and the panel opened where the compositor put it
(#152). The compositor knows the moment, and this asks it: the script
connects to workspace.windowAdded and places the first window of the class
it sees, then disconnects.

Once. The target names the window -- with a caption, where a program has
more than one window of its class (#154) -- and the first match is placed
and no other, so a window that matches later, or a second copy of the
program, is not moved onto it.

The script stays loaded until the program unloads it, as WatchGeometryScript
does; once it has placed a window it does nothing more. The size is kept, as
PositionScript keeps it.
*/
func PlaceScript(t Target, x, y int) string {
	return t.match() + fmt.Sprintf(`let placed = false;
function place(w) {
    if (placed || !matches(w)) { return; }
    placed = true;
    const g = w.frameGeometry;
    w.frameGeometry = { x: %d, y: %d, width: g.width, height: g.height };
}
for (const w of workspace.windowList()) { place(w); }
if (!placed) {
    workspace.windowAdded.connect(function added(w) {
        place(w);
        if (placed) { workspace.windowAdded.disconnect(added); }
    });
}
`, x, y)
}

// ReportGeometryScript is a KWin script that reads a window's true geometry
// and calls it back to the program over the session bus.
//
// It exists because nothing else can answer. Fyne exposes no window position
// (quirk 33), and on Wayland a move made by the compositor — which is every
// move, including the user's own Meta-drag — raises no event a toolkit can
// see. A program that wants to reopen where the user left its window has to
// ask the compositor, and this is the asking.
//
// The call carries four ints: x, y, width, height. The program exports
// method on iface at path under the bus name it owns, and that method is
// what receives them. The monitor's kwin_window_position.py does the same
// thing with the same round trip.
//
// A loaded script runs once, so a report is a fresh load, run and unload.
func ReportGeometryScript(t Target, service, path, iface, method string) string {
	return t.match() + fmt.Sprintf(`for (const w of workspace.windowList()) {
    if (matches(w)) {
        const g = w.frameGeometry;
        callDBus(%q, %q, %q, %q, g.x, g.y, g.width, g.height);
    }
}
`, service, path, iface, method)
}

/*
WatchGeometryScript is a KWin script that stays loaded and reports a window's
geometry whenever it changes.

ReportGeometryScript answers "where is it now" once. This answers "tell me
when it moves", which is what a program that wants to reopen where the user
left its window actually needs: on Wayland the move is the compositor's, no
event reaches the toolkit, and polling for an answer that changes twice a day
is a round trip a second spent on nothing.

**A loaded script keeps its signal handlers.** The load runs the body once --
which is what "a script runs once" means, and it is worth being exact because
the natural reading is that the script is then gone. It is not: the handlers
connected by that body keep firing until the script is unloaded. Verified on
Plasma 6 by loading a script that printed on frameGeometryChanged, moving the
window, and reading the print out of the journal.

Two signals, because they answer different halves. interactiveMoveResizeFinished
fires once when the user lets go of a drag, which is the case this exists for
and the one that should not report a hundred times on the way. frameGeometryChanged
catches everything else -- a compositor move, another script, a screen
change -- and fires freely, so the program on the other end saves only when
the value it holds has actually changed.

Windows that appear later are watched too: a panel restarted while the script
is loaded is still a window this wants to hear about. Only the target's,
which with a caption is one window: the preferences window a drag of which
was saved as the panel's position is the reason the target has one (#154).

The call carries four ints: x, y, width, height, like ReportGeometryScript.
*/
func WatchGeometryScript(t Target, service, path, iface, method string) string {
	return t.match() + fmt.Sprintf(`function report(w) {
    const g = w.frameGeometry;
    callDBus(%q, %q, %q, %q, Math.round(g.x), Math.round(g.y),
             Math.round(g.width), Math.round(g.height));
}
function watch(w) {
    if (!matches(w)) { return; }
    if (w.interactiveMoveResizeFinished) {
        w.interactiveMoveResizeFinished.connect(function() { report(w); });
    }
    if (w.frameGeometryChanged) {
        w.frameGeometryChanged.connect(function() { report(w); });
    }
}
for (const w of workspace.windowList()) { watch(w); }
workspace.windowAdded.connect(watch);
`, service, path, iface, method)
}
