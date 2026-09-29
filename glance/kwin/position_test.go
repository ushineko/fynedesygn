package kwin_test

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ushineko/fynedesygn/glance/kwin"
)

func TestThePositionScriptMovesAndKeepsTheSize(t *testing.T) {
	s := kwin.PositionScript("io.ushineko.example", 900, 400)

	assert.Contains(t, s, `const target = "io.ushineko.example"`)
	assert.Contains(t, s, "x: 900, y: 400")
	assert.Contains(t, s, "width: g.width, height: g.height",
		"a glance window is sized by its content; a script that set a size would fight Panel.Resize")
	assert.Contains(t, s, "w.resourceClass == target")
}

// Negative coordinates are ordinary: a compositor's origin is the primary
// screen's, and a screen to the left of it has negative x.
func TestThePositionScriptTakesAScreenToTheLeftOfTheOrigin(t *testing.T) {
	assert.Contains(t, kwin.PositionScript("x", -1920, 0), "x: -1920, y: 0")
}

func TestTheReportScriptCallsBackWithTheFourNumbers(t *testing.T) {
	s := kwin.ReportGeometryScript("io.ushineko.example",
		"org.ushineko.Example", "/Window", "org.ushineko.Window", "Report")

	assert.Contains(t, s, `callDBus("org.ushineko.Example", "/Window", "org.ushineko.Window", "Report", g.x, g.y, g.width, g.height)`)
}

// Every string in both scripts is quoted. They come from the program rather
// than from a user, but a quote that escaped the literal would run whatever
// followed it inside the compositor.
func TestAPositionScriptCannotBeEscapedByItsStrings(t *testing.T) {
	bad := `x"; workspace.windowList()[0].closeWindow(); //`

	for _, s := range []string{
		kwin.PositionScript(bad, 0, 0),
		kwin.ReportGeometryScript("x", bad, "/p", "i", "m"),
		kwin.ReportGeometryScript("x", "s", "/p", "i", bad),
	} {
		// The property is that the injected text stays inside a string
		// literal. Asserting on the raw script cannot say that — the payload
		// appears either way — so the literals are removed and what is left
		// is the code the compositor would run.
		assert.NotContains(t, withoutStringLiterals(s), "closeWindow",
			"a statement escaped its literal:\n%s", s)
		assert.Contains(t, s, `\"`, "the quote was not escaped")
	}
}

// jsString matches a JavaScript double-quoted literal, honouring backslash
// escapes, so an escaped quote does not end the match.
var jsString = regexp.MustCompile(`"(?:\\.|[^"\\])*"`)

// withoutStringLiterals is the script with every quoted literal removed.
func withoutStringLiterals(s string) string {
	return jsString.ReplaceAllString(s, `""`)
}

// AC. The watch script connects handlers rather than reporting once, which
// is what lets it stay loaded and answer when the window actually moves.
func TestTheWatchScriptConnectsRatherThanReporting(t *testing.T) {
	js := kwin.WatchGeometryScript("io.ushineko.hayami",
		"io.ushineko.hayami", "/Geometry", "io.ushineko.hayami.Geometry", "Report")

	assert.Contains(t, js, "interactiveMoveResizeFinished.connect",
		"a drag has to report once when it ends, not throughout")
	assert.Contains(t, js, "frameGeometryChanged.connect",
		"a move the compositor makes has to be caught too")
	assert.Contains(t, js, "workspace.windowAdded.connect",
		"a window that appears later is still one to watch")
	assert.Contains(t, js, `callDBus("io.ushineko.hayami", "/Geometry"`)
}

// AC. It matches on the app id, as every other script here does.
func TestTheWatchScriptMatchesTheAppID(t *testing.T) {
	js := kwin.WatchGeometryScript("io.example.thing", "s", "/p", "i", "m")
	assert.Contains(t, js, `const target = "io.example.thing"`)
	assert.Contains(t, js, "w.resourceClass != target")
}
