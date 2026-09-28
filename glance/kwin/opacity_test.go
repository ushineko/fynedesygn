package kwin_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ushineko/fynedesygn/glance/kwin"
)

func TestTheOpacityScriptMatchesTheAppIDAndSetsTheValue(t *testing.T) {
	s := kwin.OpacityScript("io.ushineko.example", 0.7)

	assert.Contains(t, s, `const target = "io.ushineko.example"`)
	assert.Contains(t, s, "w.opacity = 0.700")
	assert.Contains(t, s, "workspace.windowList()",
		"the script has to walk the windows; there is no lookup by app id")
	assert.Contains(t, s, "w.resourceClass == target",
		"resourceClass is the Wayland app id, which is what a Rule matches on too")
}

// A fraction outside the range is a window that is invisible or a script KWin
// refuses, and neither is what the caller meant.
func TestTheOpacityScriptClampsToAFraction(t *testing.T) {
	assert.Contains(t, kwin.OpacityScript("x", 4), "w.opacity = 1.000")
	assert.Contains(t, kwin.OpacityScript("x", -1), "w.opacity = 0.000")
}

// The app id is interpolated into JavaScript. It comes from the program and
// not from a user, but a quote that escapes the string literal would run
// whatever followed it inside the compositor.
func TestTheOpacityScriptCannotBeEscapedByAnAppID(t *testing.T) {
	s := kwin.OpacityScript(`x"; workspace.windowList()[0].closeWindow(); //`, 0.5)

	first := strings.Split(s, "\n")[0]
	require.True(t, strings.HasSuffix(first, `";`),
		"the app id line does not end with its own literal: %s", first)
	assert.Contains(t, first, `\"`, "the quote in the app id was not escaped")
	assert.NotContains(t, s, "\ncloseWindow", "a statement escaped the literal")
	// Everything the app id carried stays inside the one line that declares it.
	assert.NotContains(t, strings.SplitN(s, "\n", 2)[1], "closeWindow")
}

func TestTheScriptingCallsNameKWinsOwnInterface(t *testing.T) {
	load := kwin.LoadScriptCall()
	assert.Equal(t, "org.kde.KWin", load.Destination)
	assert.Equal(t, "/Scripting", load.Path)
	assert.Equal(t, "org.kde.kwin.Scripting", load.Interface)
	assert.Equal(t, "loadScript", load.Method)

	// The id the load returns is in the path, not an argument.
	assert.Equal(t, "/Scripting/Script7", kwin.RunCall(7).Path)
	assert.Equal(t, "org.kde.kwin.Script", kwin.RunCall(7).Interface)

	assert.Equal(t, "unloadScript", kwin.UnloadScriptCall().Method)
	assert.Equal(t, "/Scripting", kwin.UnloadScriptCall().Path)
}

// The decoration script sets what the rule forces, on windows already open.
//
// A rule reaches only windows KWin creates after it reads it, so without this
// a control that turns "frameless and on top" on and off changes a file and
// nothing the user can see.
func TestTheDecorationScriptSetsBothProperties(t *testing.T) {
	script := kwin.DecorationScript("io.example.app", "panel", true, true)

	assert.Contains(t, script, `"io.example.app"`)
	assert.Contains(t, script, `"panel"`)
	assert.Contains(t, script, "w.noBorder = true")
	assert.Contains(t, script, "w.keepAbove = true")
}

// Turning it off says so, rather than leaving the property alone.
func TestTheDecorationScriptPutsTheDecorationBack(t *testing.T) {
	script := kwin.DecorationScript("io.example.app", "panel", false, false)

	assert.Contains(t, script, "w.noBorder = false")
	assert.Contains(t, script, "w.keepAbove = false")
}

// It matches the title as well as the app ID.
//
// Every window in a program carries the same Wayland app_id, so a script keyed
// on that alone would strip the decoration off a program's other windows —
// which is the bug Rule.Title exists to avoid.
func TestTheDecorationScriptMatchesTheTitleToo(t *testing.T) {
	script := kwin.DecorationScript("io.example.app", "panel", true, true)

	assert.Contains(t, script, "w.resourceClass == target")
	assert.Contains(t, script, "w.caption == caption")
}

// Neither string can escape its literal and run as code in the compositor.
func TestTheDecorationScriptCannotBeEscaped(t *testing.T) {
	script := kwin.DecorationScript(`a"; workspace.slotToggleShowDesktop(); //`,
		`b"; workspace.slotToggleShowDesktop(); //`, true, true)

	assert.NotContains(t, script, `slotToggleShowDesktop();
`, "an unescaped quote let the payload out of its string")
	assert.Contains(t, script, `\"`)
}
