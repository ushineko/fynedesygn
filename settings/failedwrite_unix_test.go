//go:build !windows

package settings_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ushineko/fynedesygn/settings"
)

/*
A write that fails leaves the settings that were there.

The file is written through a temporary file in the same directory and renamed,
so a crash, a full disk or a killed process cannot leave it truncated -- and a
truncated settings file is one that has lost whatever was in it.

Per platform, because there is no portable way to say "no more files here". A
directory with its write bit off stops the temporary file on Unix and stops
nothing on Windows, where the same rule is a matter of ACLs rather than mode
bits. The Windows twin blocks the rename instead; the assertions are the same in
both.
*/
func TestAFailedWriteLeavesTheOldSettingsIntact(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	st, err := settings.Open(path)
	require.NoError(t, err)
	require.NoError(t, st.Set("jobs", jobs{Parallel: 4}))
	require.NoError(t, st.Flush())
	before, err := os.ReadFile(path)
	require.NoError(t, err)

	require.NoError(t, os.Chmod(dir, 0o500)) // no new files in this directory
	t.Cleanup(func() { _ = os.Chmod(dir, 0o750) })

	require.NoError(t, st.Set("jobs", jobs{Parallel: 99}))
	require.Error(t, st.Flush(), "a write that cannot happen is reported")

	after, err := os.ReadFile(path)
	require.NoError(t, err, "and the file is still readable")
	require.Equal(t, before, after, "with what it had")

	require.NoError(t, os.Chmod(dir, 0o750))
	require.NoError(t, st.Set("jobs", jobs{Parallel: 5}))
	require.NoError(t, st.Flush())

	var got jobs
	st2, err := settings.Open(path)
	require.NoError(t, err)
	require.True(t, st2.Get("jobs", &got))
	require.Equal(t, 5, got.Parallel, "and the next write lands")
}
