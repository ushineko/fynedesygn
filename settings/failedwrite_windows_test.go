//go:build windows

package settings_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ushineko/fynedesygn/settings"
)

/*
A write that fails leaves the settings that were there. The Windows twin of the
test in failedwrite_unix_test.go; the comment there says why there are two.

Windows has no directory write bit to take away -- os.Chmod moves the read-only
attribute and nothing else, and a read-only directory still accepts new files.
What it does have is a read-only file that cannot be replaced, which stops the
rename the store finishes its write with. Either way the write fails and the
file that was there is still the file that is there.
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

	require.NoError(t, os.Chmod(path, 0o444)) // the read-only attribute
	t.Cleanup(func() { _ = os.Chmod(path, 0o644) })

	require.NoError(t, st.Set("jobs", jobs{Parallel: 99}))
	require.Error(t, st.Flush(), "a write that cannot happen is reported")

	after, err := os.ReadFile(path)
	require.NoError(t, err, "and the file is still readable")
	require.Equal(t, before, after, "with what it had")

	require.NoError(t, os.Chmod(path, 0o644))
	require.NoError(t, st.Set("jobs", jobs{Parallel: 5}))
	require.NoError(t, st.Flush())

	var got jobs
	st2, err := settings.Open(path)
	require.NoError(t, err)
	require.True(t, st2.Get("jobs", &got))
	require.Equal(t, 5, got.Parallel, "and the next write lands")
}
