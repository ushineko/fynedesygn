package kwin_test

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ushineko/fynedesygn/glance/kwin"
)

func TestThePositionScriptMovesAndKeepsTheSize(t *testing.T) {
	s := kwin.PositionScript(kwin.Target{Class: "io.ushineko.example"}, 900, 400)

	assert.Contains(t, s, `const target = { class: "io.ushineko.example", caption: "" }`)
	assert.Contains(t, s, "x: 900, y: 400")
	assert.Contains(t, s, "width: g.width, height: g.height",
		"a glance window is sized by its content; a script that set a size would fight Panel.Resize")
	assert.Contains(t, s, "w.resourceClass == target.class")
	assert.Contains(t, s, "if (matches(w))")
}

// Negative coordinates are ordinary: a compositor's origin is the primary
// screen's, and a screen to the left of it has negative x.
func TestThePositionScriptTakesAScreenToTheLeftOfTheOrigin(t *testing.T) {
	assert.Contains(t, kwin.PositionScript(kwin.Target{Class: "x"}, -1920, 0), "x: -1920, y: 0")
}

func TestTheReportScriptCallsBackWithTheFourNumbers(t *testing.T) {
	s := kwin.ReportGeometryScript(kwin.Target{Class: "io.ushineko.example"},
		"org.ushineko.Example", "/Window", "org.ushineko.Window", "Report")

	assert.Contains(t, s, `callDBus("org.ushineko.Example", "/Window", "org.ushineko.Window", "Report", g.x, g.y, g.width, g.height)`)
}

// Every string in both scripts is quoted. They come from the program rather
// than from a user, but a quote that escaped the literal would run whatever
// followed it inside the compositor.
func TestAPositionScriptCannotBeEscapedByItsStrings(t *testing.T) {
	bad := `x"; workspace.windowList()[0].closeWindow(); //`

	for _, s := range []string{
		kwin.PositionScript(kwin.Target{Class: bad}, 0, 0),
		kwin.PositionScript(kwin.Target{Class: "x", Caption: bad}, 0, 0),
		kwin.PlaceScript(kwin.Target{Class: bad}, 0, 0),
		kwin.ReportGeometryScript(kwin.Target{Class: "x"}, bad, "/p", "i", "m"),
		kwin.ReportGeometryScript(kwin.Target{Class: "x"}, "s", "/p", "i", bad),
		kwin.WatchGeometryScript(kwin.Target{Class: "x", Caption: bad}, "s", "/p", "i", "m"),
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
	js := kwin.WatchGeometryScript(kwin.Target{Class: "io.ushineko.hayami"},
		"io.ushineko.hayami", "/Geometry", "io.ushineko.hayami.Geometry", "Report")

	assert.Contains(t, js, "interactiveMoveResizeFinished.connect",
		"a drag has to report once when it ends, not throughout")
	assert.Contains(t, js, "frameGeometryChanged.connect",
		"a move the compositor makes has to be caught too")
	assert.Contains(t, js, "workspace.windowAdded.connect",
		"a window that appears later is still one to watch")
	assert.Contains(t, js, `callDBus("io.ushineko.hayami", "/Geometry"`)
}

// AC. It matches on the target, as every other script here does.
func TestTheWatchScriptMatchesTheTarget(t *testing.T) {
	js := kwin.WatchGeometryScript(kwin.Target{Class: "io.example.thing"}, "s", "/p", "i", "m")
	assert.Contains(t, js, `const target = { class: "io.example.thing", caption: "" }`)
	assert.Contains(t, js, "if (!matches(w)) { return; }")
}

// AC. A target with a caption is one window of its class: the panel, not
// the preferences window that shares its app ID (#154). Without a caption
// the class alone matches, as before, for a program with one window.
func TestATargetWithACaptionIsOneWindowOfItsClass(t *testing.T) {
	js := kwin.WatchGeometryScript(kwin.Target{Class: "io.ushineko.hayami", Caption: "hayami"},
		"s", "/p", "i", "m")
	assert.Contains(t, js, `const target = { class: "io.ushineko.hayami", caption: "hayami" }`)
	assert.Contains(t, js, `w.resourceClass == target.class &&`)
	assert.Contains(t, js, `(target.caption == "" || w.caption == target.caption)`)

	for _, s := range []string{
		kwin.PositionScript(kwin.Target{Class: "c", Caption: "t"}, 0, 0),
		kwin.PlaceScript(kwin.Target{Class: "c", Caption: "t"}, 0, 0),
		kwin.ReportGeometryScript(kwin.Target{Class: "c", Caption: "t"}, "s", "/p", "i", "m"),
	} {
		assert.Contains(t, s, `caption: "t"`, "every script takes the same target")
		assert.Contains(t, s, "matches(w)")
	}
}

// AC. The place script moves a window that is there and otherwise waits for
// one, which is what a program whose window is not up yet needs (#152).
func TestThePlaceScriptPlacesNowOrWhenTheWindowAppears(t *testing.T) {
	s := kwin.PlaceScript(kwin.Target{Class: "io.ushineko.example"}, 3690, 2052)

	assert.Contains(t, s, `const target = { class: "io.ushineko.example", caption: "" }`)
	assert.Contains(t, s, "x: 3690, y: 2052")
	assert.Contains(t, s, "width: g.width, height: g.height", "the size is the window's own")
	assert.Contains(t, s, "for (const w of workspace.windowList()) { place(w); }",
		"a window already on screen is placed at once")
	assert.Contains(t, s, "workspace.windowAdded.connect", "one not yet there is placed when it appears")
}

// AC. The place script places one window and no more: a later match, or a
// second copy of the program, is not moved onto the first.
func TestThePlaceScriptPlacesOnlyTheFirstWindow(t *testing.T) {
	s := kwin.PlaceScript(kwin.Target{Class: "io.ushineko.example"}, 0, 0)

	assert.Contains(t, s, "if (placed || !matches(w)) { return; }")
	assert.Contains(t, s, "workspace.windowAdded.disconnect(added)",
		"the handler lets go once it has placed a window")
	assert.Contains(t, s, "if (!placed) {", "a window placed at once needs no handler")
}
