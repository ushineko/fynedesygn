package kwin

import "fmt"

// OpacityScript is a KWin script that sets the opacity of every window of one
// app id, and the answer to a question a Wayland client cannot answer for
// itself.
//
// On X11 a client sets its own opacity: the window manager reads
// _NET_WM_WINDOW_OPACITY off the window, and glance.Window.SetOpacity writes
// it. Wayland has no such protocol, and GLFW says so plainly — its Wayland
// backend answers GLFW_FEATURE_UNAVAILABLE, "the platform does not support
// setting the window opacity". The compositor is the only thing that can, and
// on Plasma the scripting API is how you ask it.
//
// This is not the same as the opacity in a Rule. A rule is applied when the
// window is created; this runs against a window that is already on screen, so
// a program can offer the opacity menu the archetype has.
//
// Windows are matched on resourceClass, which is the Wayland app ID — the same
// string a Rule matches on, and the one Fyne takes from fyne.App.UniqueID.
//
// The script is returned as text and the calls as data, for the reason
// ReconfigureCall gives: a D-Bus client is not Fyne, and this module's
// packages take no dependency beyond it. A program writes the script to a
// file, calls LoadScriptCall, calls RunCall with the id it got back, and calls
// UnloadScriptCall. A loaded script runs once.
func OpacityScript(appID string, opacity float32) string {
	if opacity < 0 {
		opacity = 0
	}
	if opacity > 1 {
		opacity = 1
	}
	// The app ID is quoted into JavaScript with %q, which escapes the quote
	// and the backslash exactly as a JavaScript string literal needs. It comes
	// from the program rather than from a user, but a rule that only holds
	// while nobody is careless is not a rule: an unescaped quote here would
	// run whatever followed it inside the compositor.
	return fmt.Sprintf(`const target = %q;
for (const w of workspace.windowList()) {
    if (w.resourceClass == target) {
        w.opacity = %.3f;
    }
}
`, appID, opacity)
}

// LoadScriptCall is the session-bus method that loads a script file. It takes
// the script's absolute path and a plugin name, and returns the script's id.
func LoadScriptCall() DBusCall {
	return DBusCall{
		Destination: "org.kde.KWin",
		Path:        "/Scripting",
		Interface:   "org.kde.kwin.Scripting",
		Method:      "loadScript",
	}
}

// RunCall is the session-bus method that runs a loaded script. The path
// carries the id LoadScriptCall returned.
func RunCall(id int) DBusCall {
	return DBusCall{
		Destination: "org.kde.KWin",
		Path:        fmt.Sprintf("/Scripting/Script%d", id),
		Interface:   "org.kde.kwin.Script",
		Method:      "run",
	}
}

// UnloadScriptCall is the session-bus method that unloads a script by the
// plugin name it was loaded under. A script that ran and was not unloaded
// stays registered with KWin.
func UnloadScriptCall() DBusCall {
	return DBusCall{
		Destination: "org.kde.KWin",
		Path:        "/Scripting",
		Interface:   "org.kde.kwin.Scripting",
		Method:      "unloadScript",
	}
}
