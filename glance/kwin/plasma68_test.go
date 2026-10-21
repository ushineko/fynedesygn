package kwin

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests are inside the package because loadLikeKWin68 rewrites the file
// with the package's own reader and writer.

// plasma68 is a kwinrulesrc as KWin 6.8 saves it: the list in Order, no rules
// or count, and a border policy instead of noborder.
const plasma68 = `[1]
Description=Battery panel always on top
above=true
aboverule=2
wmclass=example-battery-panel
wmclassmatch=1

[21]
Description=Terminal maximised
maximizehoriz=true
maximizehorizrule=4
wmclass=example-terminal
wmclassmatch=1

[3]
Description=Volume OSD
decorationpolicy=none
decorationpolicyrule=2
title=volume-osd
titlematch=1
wmclass=example-osd
wmclassmatch=1

[General]
Order=1,3,21
`

// theirs are the descriptions of the rules in plasma68, none of which this
// package may lose.
var theirs = []string{"Battery panel always on top", "Terminal maximised", "Volume OSD"}

func sandbox68(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	path := filepath.Join(dir, "kwinrulesrc")
	if content != "" {
		require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	}
	return path
}

func contents(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path) // nolint:gosec // a path this test made
	require.NoError(t, err)
	return string(b)
}

/*
loadLikeKWin68 does to the file what RuleBookSettings::usrRead and usrSave do
in KWin v6.8.0 (src/rulebooksettings.cpp) when a reconfigure reloads the
rules:

 1. A non-empty `rules` is an old file to migrate: every rule group it does
    not list is deleted, and the order becomes that list.
 2. Names with no group are dropped; groups not listed are prepended.
 3. A rule with noborderrule gets decorationpolicy instead.
 4. The file is saved with `Order` and without `rules` or `count`.

It is a copy of KWin's behaviour, not KWin, and goes stale if KWin changes.
*/
func loadLikeKWin68(t *testing.T, path string) {
	t.Helper()
	file, err := load(path)
	require.NoError(t, err)
	general := file.ensure(generalSection)
	raw, _ := general.get(orderKey)
	order := splitNames(raw)
	raw, _ = general.get(legacyKey)

	if legacy := splitNames(raw); len(legacy) > 0 {
		var doomed []string
		for _, s := range file.sections {
			if !isMetaData(s.name) && !slices.Contains(legacy, s.name) {
				doomed = append(doomed, s.name)
			}
		}
		for _, name := range doomed {
			file.remove(name)
		}
		order = legacy
	}
	general.unset(legacyKey)
	general.unset(countKey)

	order = slices.DeleteFunc(order, func(name string) bool { return file.section(name) == nil })
	for _, s := range file.sections {
		if !isMetaData(s.name) && !slices.Contains(order, s.name) {
			order = append([]string{s.name}, order...)
		}
	}
	general.set(orderKey, strings.Join(order, ","))

	for _, name := range order {
		sec := file.section(name)
		if rule, ok := sec.get("noborderrule"); ok && rule != "0" {
			policy := "server"
			if v, _ := sec.get("noborder"); v == "true" {
				policy = decorationNone
			}
			sec.set("decorationpolicyrule", rule)
			sec.set("decorationpolicy", policy)
			sec.unset("noborder")
			sec.unset("noborderrule")
		}
	}
	require.NoError(t, save(path, file))
}

func indicator(appID string) Rule {
	return Rule{AppID: appID, Title: appID + "-indicator", AlwaysOnTop: true, NoBorder: true, NoFocus: true}
}

func assertTheirsSurvive(t *testing.T, path string) {
	t.Helper()
	got := contents(t, path)
	for _, d := range theirs {
		assert.Contains(t, got, "Description="+d+"\n", "a rule this package did not come to change was lost")
	}
}

// The file holds every rule the user has. Installing into a 6.8 file once
// wrote rules=1 beside Order, over the user's [1], and KWin deleted the rest.
func TestInstallingIntoAPlasma68FileKeepsEveryRuleAfterKWinLoadsIt(t *testing.T) {
	path := sandbox68(t, plasma68)

	require.NoError(t, Install(indicator("io.example.app")))

	got := contents(t, path)
	assert.NotContains(t, got, "rules=", "a legacy list in a 6.8 file is a list of the only rules to keep")
	assert.NotContains(t, got, "count=")
	assert.Contains(t, got, "Order=1,3,21,22\n")

	loadLikeKWin68(t, path)
	assertTheirsSurvive(t, path)
	r, ok, err := LookupTitled("io.example.app", "io.example.app-indicator")
	require.NoError(t, err)
	require.True(t, ok, "the rule is not found once KWin has saved the file")
	assert.True(t, r.NoBorder)
}

// A file damaged by the version that wrote rules=1 into a 6.8 file is repaired
// by the next install, before KWin reads the stray list.
func TestAStrayLegacyListInAPlasma68FileIsFoldedIntoOrder(t *testing.T) {
	path := sandbox68(t, strings.Replace(plasma68, "Order=1,3,21\n", "Order=1,3,21\ncount=1\nrules=1\n", 1))

	require.NoError(t, Install(indicator("io.example.app")))

	got := contents(t, path)
	assert.NotContains(t, got, "rules=")
	assert.NotContains(t, got, "count=")
	assert.Contains(t, got, "Order=1,3,21,22\n")
	loadLikeKWin68(t, path)
	assertTheirsSurvive(t, path)
}

// Two programs on one desk, each with its own rule, installing in turn with a
// reconfigure between: each once wrote into [1] and took the other's place.
func TestTwoProgramsInstallingInTurnBothKeepTheirRule(t *testing.T) {
	path := sandbox68(t, plasma68)

	for range 2 {
		for _, id := range []string{"io.example.one", "io.example.two"} {
			require.NoError(t, Install(indicator(id)))
			loadLikeKWin68(t, path)
		}
	}

	assertTheirsSurvive(t, path)
	for _, id := range []string{"io.example.one", "io.example.two"} {
		r, ok, err := LookupTitled(id, id+"-indicator")
		require.NoError(t, err)
		require.True(t, ok, id+" lost its rule")
		assert.True(t, r.NoBorder, id+" got its titlebar back")
	}
	assert.Equal(t, 2, strings.Count(contents(t, path), "wmclass=io.example."),
		"a second rule was added for a program that already had one")
}

// KWin 6.8 says "no border" as a decoration policy, and Lookup reads it.
func TestAPlasma68InstallSaysNoBorderAsADecorationPolicy(t *testing.T) {
	path := sandbox68(t, plasma68)
	id := "io.example.app"

	require.NoError(t, Install(indicator(id)))
	got := contents(t, path)
	assert.NotContains(t, got, "noborder")
	assert.Equal(t, 2, strings.Count(got, "decorationpolicy=none\n"))

	off := indicator(id)
	off.NoBorder = false
	require.NoError(t, Install(off))
	r, ok, err := LookupTitled(id, off.Title)
	require.NoError(t, err)
	require.True(t, ok)
	assert.False(t, r.NoBorder)
	assert.Equal(t, 1, strings.Count(contents(t, path), "decorationpolicy=none\n"),
		"turning the border back on took another rule's policy, or left this one's")
}

// A rule written in the legacy format reads back after KWin 6.8 migrates it:
// a settings switch that asks Lookup must not untick itself.
func TestALegacyRuleIsStillFoundAfterKWin68MigratesIt(t *testing.T) {
	path := sandbox68(t, "")
	id := "io.example.app"

	require.NoError(t, Install(indicator(id)))
	assert.Contains(t, contents(t, path), "noborder=true\n", "a new file is written in the legacy format")
	loadLikeKWin68(t, path)

	r, ok, err := LookupTitled(id, id+"-indicator")
	require.NoError(t, err)
	require.True(t, ok)
	assert.True(t, r.NoBorder)
}

func TestAnEmptyFileGetsTheListEveryKWinReads(t *testing.T) {
	path := sandbox68(t, "")

	require.NoError(t, Install(indicator("io.example.app")))

	got := contents(t, path)
	assert.Contains(t, got, "rules=1\n")
	assert.Contains(t, got, "count=1\n")
	assert.NotContains(t, got, "Order=")
}

func TestRemovingFromAPlasma68FileUpdatesOrder(t *testing.T) {
	path := sandbox68(t, plasma68)
	id := "io.example.app"
	require.NoError(t, Install(indicator(id)))

	removed, err := RemoveTitled(id, id+"-indicator")
	require.NoError(t, err)
	require.True(t, removed)

	got := contents(t, path)
	assert.Contains(t, got, "Order=1,3,21\n")
	assert.NotContains(t, got, "rules=")
	assert.NotContains(t, got, id)
	loadLikeKWin68(t, path)
	assertTheirsSurvive(t, path)
}

// Groups the list does not name are rules to KWin 6.8, which prepends them.
// A new rule is numbered past them and the file stays in the 6.8 format.
func TestUnlistedGroupsAreRulesTooAndAreNotWrittenOver(t *testing.T) {
	path := sandbox68(t, `[1]
Description=Listed
wmclass=example-listed

[5]
Description=Unlisted
wmclass=example-unlisted

[General]
Order=1
`)

	require.NoError(t, Install(indicator("io.example.app")))
	got := contents(t, path)
	assert.Contains(t, got, "[6]\n")
	assert.Contains(t, got, "Order=1,5,6\n")
	assert.NotContains(t, got, "rules=")

	loadLikeKWin68(t, path)
	got = contents(t, path)
	assert.Contains(t, got, "Description=Listed\n")
	assert.Contains(t, got, "Description=Unlisted\n")
}

// A file of rule groups and no list at all is one only KWin 6.8 leaves.
func TestRuleGroupsWithNoListAreAPlasma68File(t *testing.T) {
	path := sandbox68(t, `[1]
Description=Listed nowhere
wmclass=example-nowhere
`)

	require.NoError(t, Install(indicator("io.example.app")))
	got := contents(t, path)
	assert.Contains(t, got, "Order=1,2\n")
	assert.NotContains(t, got, "rules=")
	loadLikeKWin68(t, path)
	assert.Contains(t, contents(t, path), "Description=Listed nowhere\n")
}
