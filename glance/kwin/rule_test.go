package kwin_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ushineko/fynedesygn/glance/kwin"
)

// sandbox points XDG_CONFIG_HOME at a temporary directory and returns the
// rules path inside it. A test that wrote to the developer's own kwinrulesrc
// would be a test that changed their desktop.
func sandbox(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	return filepath.Join(dir, "kwinrulesrc")
}

// existing is a rules file with one rule the user already had, plus the
// comment and the $Version section KDE writes.
const existing = `[$Version]
update_info=kwinrules.upd:replace-placement-string-to-enum

[1]
Description=Krita full screen
above=true
aboverule=2
wmclass=krita
wmclassmatch=1

[General]
count=1
rules=1
`

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path) // nolint:gosec // a path this test made
	require.NoError(t, err)
	return string(b)
}

func TestInstallingIntoAFileThatDoesNotExistYetCreatesIt(t *testing.T) {
	path := sandbox(t)

	require.NoError(t, kwin.Install(kwin.Rule{
		AppID:       "io.ushineko.sensors",
		AlwaysOnTop: true,
		NoBorder:    true,
		Opacity:     95,
	}))

	got := read(t, path)
	assert.Contains(t, got, "wmclass=io.ushineko.sensors")
	assert.Contains(t, got, "wmclassmatch=1", "the match must be exact")
	assert.Contains(t, got, "above=true")
	assert.Contains(t, got, "aboverule=2", "always on top is forced")
	assert.Contains(t, got, "noborder=true")
	assert.Contains(t, got, "opacityactive=95")
	assert.Contains(t, got, "opacityactiverule=4", "opacity applies initially, so the user can override it")
	assert.Contains(t, got, "opacityinactive=95", "an unfocused glance window is the normal case")
}

// The file holds every window rule the user has. A rule installer that
// normalised it would hand them back a file that was not theirs.
func TestInstallingKeepsEveryRuleTheUserAlreadyHad(t *testing.T) {
	path := sandbox(t)
	require.NoError(t, os.WriteFile(path, []byte(existing), 0o600))

	require.NoError(t, kwin.Install(kwin.Rule{AppID: "io.ushineko.sensors", AlwaysOnTop: true}))

	got := read(t, path)
	assert.Contains(t, got, "Description=Krita full screen")
	assert.Contains(t, got, "wmclass=krita")
	assert.Contains(t, got, "[$Version]", "KDE's own section must survive")
	assert.Contains(t, got, "update_info=kwinrules.upd:replace-placement-string-to-enum")
	assert.Contains(t, got, "rules=1,2")
	assert.Contains(t, got, "count=2")
}

// Installing twice updates the rule in place. A second section matching the
// same window class would leave KWin applying two rules to one window.
func TestInstallingTwiceUpdatesTheSameRule(t *testing.T) {
	path := sandbox(t)
	const id = "io.ushineko.sensors"

	require.NoError(t, kwin.Install(kwin.Rule{AppID: id, Opacity: 95}))
	require.NoError(t, kwin.Install(kwin.Rule{AppID: id, Opacity: 80}))

	got := read(t, path)
	assert.Contains(t, got, "opacityactive=80")
	assert.NotContains(t, got, "opacityactive=95")

	found, ok, err := kwin.Lookup(id)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, 80, found.Opacity)
	assert.Contains(t, read(t, path), "count=1", "a second rule was added for the same window")
}

// KWin's position rule selects a screen and snaps to its origin on Wayland. A
// rule that half-works is worse than none, so one an earlier version wrote is
// cleared rather than left behind.
func TestInstallingClearsAStalePositionRule(t *testing.T) {
	path := sandbox(t)
	require.NoError(t, os.WriteFile(path, []byte(`[General]
count=1
rules=1

[1]
wmclass=io.ushineko.sensors
position=100,200
positionrule=4
size=300,400
sizerule=4
`), 0o600))

	require.NoError(t, kwin.Install(kwin.Rule{AppID: "io.ushineko.sensors", AlwaysOnTop: true}))

	got := read(t, path)
	assert.NotContains(t, got, "position=")
	assert.NotContains(t, got, "positionrule=")
	assert.NotContains(t, got, "size=")
}

// Zero opacity means "leave the user's setting alone", which has to clear the
// keys rather than write 0 — a rule with opacity 0 is an invisible window.
func TestZeroOpacityWritesNoOpacityKeysAtAll(t *testing.T) {
	path := sandbox(t)

	require.NoError(t, kwin.Install(kwin.Rule{AppID: "io.ushineko.sensors", AlwaysOnTop: true}))

	got := read(t, path)
	assert.NotContains(t, got, "opacityactive")
	assert.NotContains(t, got, "opacity")
}

func TestTurningOffAFlagRemovesItRatherThanWritingFalse(t *testing.T) {
	path := sandbox(t)
	const id = "io.ushineko.sensors"

	require.NoError(t, kwin.Install(kwin.Rule{AppID: id, AlwaysOnTop: true, NoBorder: true}))
	require.NoError(t, kwin.Install(kwin.Rule{AppID: id, AlwaysOnTop: true}))

	got := read(t, path)
	assert.Contains(t, got, "above=true")
	assert.NotContains(t, got, "noborder",
		"a rule edited down should not keep the part that was removed")
}

func TestRemovingTakesTheRuleOutAndLeavesTheOthers(t *testing.T) {
	path := sandbox(t)
	require.NoError(t, os.WriteFile(path, []byte(existing), 0o600))
	require.NoError(t, kwin.Install(kwin.Rule{AppID: "io.ushineko.sensors", AlwaysOnTop: true}))

	removed, err := kwin.Remove("io.ushineko.sensors")
	require.NoError(t, err)
	assert.True(t, removed)

	got := read(t, path)
	assert.NotContains(t, got, "io.ushineko.sensors")
	assert.Contains(t, got, "wmclass=krita", "the user's own rule must survive")
	assert.Contains(t, got, "rules=1")
	assert.Contains(t, got, "count=1")
}

func TestRemovingARuleThatIsNotThereChangesNothing(t *testing.T) {
	path := sandbox(t)
	require.NoError(t, os.WriteFile(path, []byte(existing), 0o600))

	removed, err := kwin.Remove("io.ushineko.nothing")
	require.NoError(t, err)

	assert.False(t, removed)
	assert.Equal(t, existing, read(t, path), "the file was rewritten for nothing")
}

func TestLookupOnADesktopWithNoRulesFileIsNotAnError(t *testing.T) {
	sandbox(t)

	_, ok, err := kwin.Lookup("io.ushineko.sensors")

	require.NoError(t, err, "a desktop with no custom rules yet is the normal case")
	assert.False(t, ok)
}

func TestARuleNeedsAWindowClassToMatchOn(t *testing.T) {
	sandbox(t)

	assert.Error(t, kwin.Install(kwin.Rule{AppID: "  ", AlwaysOnTop: true}))
}

func TestAnOpacityOutsideAPercentageIsRefused(t *testing.T) {
	sandbox(t)

	assert.Error(t, kwin.Install(kwin.Rule{AppID: "x", Opacity: 140}))
	assert.Error(t, kwin.Install(kwin.Rule{AppID: "x", Opacity: -1}))
}

// A new rule takes the next free number, not the count: a file whose rules
// were removed out of order has gaps, and reusing a number would collide with
// a rule the user still has.
func TestANewRuleTakesTheNextFreeNumberNotTheCount(t *testing.T) {
	path := sandbox(t)
	require.NoError(t, os.WriteFile(path, []byte(`[General]
count=1
rules=7

[7]
wmclass=krita
`), 0o600))

	require.NoError(t, kwin.Install(kwin.Rule{AppID: "io.ushineko.sensors", AlwaysOnTop: true}))

	got := read(t, path)
	assert.Contains(t, got, "[8]")
	assert.Contains(t, got, "rules=7,8")
}

func TestTheDescriptionFallsBackToSomethingTheUserCanRecognise(t *testing.T) {
	sandbox(t)
	require.NoError(t, kwin.Install(kwin.Rule{AppID: "io.ushineko.sensors", AlwaysOnTop: true}))

	found, ok, err := kwin.Lookup("io.ushineko.sensors")
	require.NoError(t, err)
	require.True(t, ok)
	assert.Contains(t, found.Description, "io.ushineko.sensors")
}

func TestReconfigureNamesTheSessionBusMethodThatReloadsTheRules(t *testing.T) {
	c := kwin.ReconfigureCall()

	assert.Equal(t, "org.kde.KWin", c.Destination)
	assert.Equal(t, "/KWin", c.Path)
	assert.Equal(t, "org.kde.KWin.reconfigure", c.Interface+"."+c.Method)
}

// A rule value is written as the rest of its line, so one carrying a newline
// would continue into keys of its own inside a file holding every window rule
// the user has.
func TestAValueWithALineBreakIsRefused(t *testing.T) {
	path := sandbox(t)

	assert.Error(t, kwin.Install(kwin.Rule{
		AppID:       "io.ushineko.sensors\nabove=true\naboverule=2",
		AlwaysOnTop: true,
	}))
	assert.Error(t, kwin.Install(kwin.Rule{
		AppID:       "io.ushineko.sensors",
		Description: "harmless\n[General]\nrules=",
		AlwaysOnTop: true,
	}))

	_, err := os.Stat(path)
	assert.True(t, os.IsNotExist(err), "a refused rule must not have written the file")
}

// realWorld is shaped like an actual kwinrulesrc: sections out of numeric
// order, a gap where a rule was removed, and [General] wherever KDE left it.
const realWorld = `[1]
Description=Mouse battery always on top
above=true
aboverule=2
wmclass=logitech-mouse-battery
wmclassmatch=1

[21]
Description=Terminal maximised
maximizehoriz=true
maximizehorizrule=4
wmclass=alacritty-monitor-0
wmclassmatch=1

[3]
Description=Volume OSD
noborder=true
noborderrule=2
title=volume-osd
titlematch=1

[General]
count=3
rules=1,3,21
`

// A file this package did not come to change must come back exactly as it
// went in. kwinrulesrc holds every window rule the user has.
func TestRewritingAFileWithoutChangingItIsByteIdentical(t *testing.T) {
	path := sandbox(t)
	require.NoError(t, os.WriteFile(path, []byte(realWorld), 0o600))

	// Installing and removing the same rule is the round trip.
	require.NoError(t, kwin.Install(kwin.Rule{AppID: "io.ushineko.sensors", AlwaysOnTop: true}))
	removed, err := kwin.Remove("io.ushineko.sensors")
	require.NoError(t, err)
	require.True(t, removed)

	assert.Equal(t, realWorld, read(t, path),
		"install followed by remove did not restore the file")
}

// Every install used to add one blank line per section, so a file of thirteen
// rules grew thirteen lines a run.
func TestInstallingRepeatedlyDoesNotGrowTheFile(t *testing.T) {
	path := sandbox(t)
	require.NoError(t, os.WriteFile(path, []byte(realWorld), 0o600))

	require.NoError(t, kwin.Install(kwin.Rule{AppID: "io.ushineko.sensors", AlwaysOnTop: true}))
	once := read(t, path)
	for range 5 {
		require.NoError(t, kwin.Install(kwin.Rule{AppID: "io.ushineko.sensors", AlwaysOnTop: true}))
	}

	assert.Equal(t, once, read(t, path), "repeated installs changed the file")
	assert.Equal(t, strings.Count(realWorld, "\n\n"), strings.Count(once, "\n\n")-1,
		"a blank line was added to a section this install did not touch")
}

// The file ends with exactly one newline, the way KDE's writer leaves it.
func TestTheFileEndsWithExactlyOneNewline(t *testing.T) {
	path := sandbox(t)
	require.NoError(t, os.WriteFile(path, []byte(realWorld), 0o600))
	require.NoError(t, kwin.Install(kwin.Rule{AppID: "io.ushineko.sensors", AlwaysOnTop: true}))

	got := read(t, path)
	assert.True(t, strings.HasSuffix(got, "\n"))
	assert.False(t, strings.HasSuffix(got, "\n\n"))
}
