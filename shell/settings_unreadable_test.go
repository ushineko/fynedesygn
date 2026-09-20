package shell

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/ushineko/fynedesygn/settings"
)

/*
A settings file that does not parse must not stop a window opening, and must not
cost the user their file (spec 019).

The shell already collects the error Open returns and reports it. What this adds
is the guarantee that the rest of the program carries on: appearance and layout
writes fail quietly rather than panicking, and nothing on disk is disturbed by
a program that only started up.
*/
func TestAShellWhoseSettingsDoNotParseStillOpens(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	body := "{not json at all"
	require.NoError(t, os.WriteFile(path, []byte(body), 0o600))

	o := testOptions(NewSection("One", nil, blank))
	o.SettingsPath = path
	s := headless(t, o)

	require.NotNil(t, s.Settings(), "a store is always there to talk to")

	var parse *settings.ParseError
	require.ErrorAs(t, s.Settings().Unreadable(), &parse,
		"the shell knows why, in terms it can report")

	// The library's own writes refuse rather than panic or overwrite.
	require.Error(t, s.Settings().Set(NavKey, navSettings{Mode: "rail"}))

	// And the user's file is exactly where they left it, with no .bad beside it.
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	on, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, body, string(on))
}
