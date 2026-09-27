package kwin

import "fmt"

// PositionScript is a KWin script that moves every window of one app id, and
// the answer to a question a Wayland client cannot answer for itself.
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
func PositionScript(appID string, x, y int) string {
	return fmt.Sprintf(`const target = %q;
for (const w of workspace.windowList()) {
    if (w.resourceClass == target) {
        const g = w.frameGeometry;
        w.frameGeometry = { x: %d, y: %d, width: g.width, height: g.height };
    }
}
`, appID, x, y)
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
func ReportGeometryScript(appID, service, path, iface, method string) string {
	return fmt.Sprintf(`const target = %q;
for (const w of workspace.windowList()) {
    if (w.resourceClass == target) {
        const g = w.frameGeometry;
        callDBus(%q, %q, %q, %q, g.x, g.y, g.width, g.height);
    }
}
`, appID, service, path, iface, method)
}
