package settings_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ushineko/fynedesygn/settings"
)

// jobs is a program's own section, with the json tags a caller would write.
type jobs struct {
	Parallel int    `json:"parallel"`
	KeepLogs bool   `json:"keepLogs"`
	Label    string `json:"label,omitempty"`
}

// counting wraps a codec and counts the writes, which is how a test tells one
// write from three without reading the clock.
type counting struct {
	settings.Codec
	writes atomic.Int64
}

func (c *counting) Marshal(v any) ([]byte, error) {
	c.writes.Add(1)
	return c.Codec.Marshal(v)
}

/*
A program that has never been configured is not an error.

It is the ordinary first run, and a window that refused to open because it had
no settings file would be a window nobody could ever open.
*/
func TestAMissingFileIsTheDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "settings.json")
	st, err := settings.Open(path)
	require.NoError(t, err)

	var got jobs
	require.False(t, st.Get("jobs", &got), "nothing has been set")
	require.Equal(t, jobs{}, got)

	require.NoError(t, st.Set("jobs", jobs{Parallel: 4, KeepLogs: true}))
	require.NoError(t, st.Flush())

	b, err := os.ReadFile(path)
	require.NoError(t, err, "the file and its directory are created by the first write")
	require.Contains(t, string(b), `"parallel": 4`)

	again, err := settings.Open(path)
	require.NoError(t, err)
	require.True(t, again.Get("jobs", &got))
	require.Equal(t, jobs{Parallel: 4, KeepLogs: true}, got)
}

/*
A section this build never asks for survives a load and a save.

An older binary reading a file a newer one wrote must not quietly delete the
setting it could not decode. That is the failure this rule exists for, and it is
why sections are held as bytes until something asks for one.
*/
func TestASectionThisBuildDoesNotKnowIsKept(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	require.NoError(t, os.WriteFile(path, []byte(`{
	  "fromTheFuture": {"nested": {"deep": [1, 2, 3]}, "kept": true},
	  "jobs": {"parallel": 1}
	}`), 0o600))

	st, err := settings.Open(path)
	require.NoError(t, err)
	require.NoError(t, st.Set("jobs", jobs{Parallel: 8}))
	require.NoError(t, st.Flush())

	var doc map[string]json.RawMessage
	b, err := os.ReadFile(path)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(b, &doc))
	require.Contains(t, doc, "fromTheFuture")

	var future struct {
		Nested struct {
			Deep []int `json:"deep"`
		} `json:"nested"`
		Kept bool `json:"kept"`
	}
	require.NoError(t, json.Unmarshal(doc["fromTheFuture"], &future))
	require.Equal(t, []int{1, 2, 3}, future.Nested.Deep)
	require.True(t, future.Kept)
}

/*
A write that fails leaves the settings that were there.

The file is written through a temporary file in the same directory and renamed,
so a crash, a full disk or a killed process cannot leave it truncated -- and a
truncated settings file is one that has lost whatever was in it.
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
	require.Len(t, ls(t, dir), 1, "no temporary file is left behind")
}

// ls names the files in a directory.
func ls(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		out = append(out, e.Name())
	}
	return out
}

/*
A run of changes is one write.

A slider dragged across its range or a divider dragged across the window is one
settings file on disk, not one per pixel.
*/
func TestChangesInOneMomentAreOneWrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	c := &counting{Codec: settings.JSON}
	st, err := settings.OpenWith(path, c)
	require.NoError(t, err)
	st.SetDelay(30 * time.Millisecond)

	for i := range 5 {
		require.NoError(t, st.Set("jobs", jobs{Parallel: i}))
	}
	require.True(t, st.Pending())
	require.Zero(t, c.writes.Load(), "nothing is written while the changes are still coming")

	require.Eventually(t, func() bool { return !st.Pending() }, time.Second, 10*time.Millisecond)
	require.Equal(t, int64(1), c.writes.Load())

	// Flush writes what is waiting, at once, and leaves nothing pending.
	require.NoError(t, st.Set("jobs", jobs{Parallel: 9}))
	require.NoError(t, st.Flush())
	require.False(t, st.Pending())
	require.Equal(t, int64(2), c.writes.Load())

	// And a flush with nothing waiting does not rewrite the file.
	require.NoError(t, st.Flush())
	require.Equal(t, int64(2), c.writes.Load())
}

// Setting a section to what it already holds is not a change, so a section
// rebuilt from the same values does not rewrite the file.
func TestSettingTheSameValueIsNotAChange(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	st, err := settings.Open(path)
	require.NoError(t, err)
	require.NoError(t, st.Set("jobs", jobs{Parallel: 4}))
	require.NoError(t, st.Flush())

	require.NoError(t, st.Set("jobs", jobs{Parallel: 4}))
	require.False(t, st.Pending())
}

/*
A file that does not parse is moved aside, not deleted and not obeyed.

It is the user's file, so it is kept where they can look at it. It is not left
in place, because every save from then on would have to either refuse or
overwrite it without saying so.
*/
func TestAFileThatDoesNotParseIsMovedAside(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	require.NoError(t, os.WriteFile(path, []byte("{this is not json"), 0o600))

	st, err := settings.Open(path)
	require.Error(t, err, "the caller is told rather than left to guess")
	require.Contains(t, err.Error(), ".bad")
	require.NotNil(t, st, "and the store still works, on defaults")

	aside, err := os.ReadFile(path + ".bad")
	require.NoError(t, err)
	require.Equal(t, "{this is not json", string(aside), "kept exactly as it was")

	var got jobs
	require.False(t, st.Get("jobs", &got))
	require.NoError(t, st.Set("jobs", jobs{Parallel: 2}))
	require.NoError(t, st.Flush())

	again, err := settings.Open(path)
	require.NoError(t, err)
	require.True(t, again.Get("jobs", &got))
}

// A section that no longer decodes into the caller's type reads as absent. A
// program that changed the shape of its settings gets its defaults rather than
// an error it has no way to act on.
func TestASectionThatNoLongerFitsReadsAsAbsent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"jobs": "used to be a string"}`), 0o600))
	st, err := settings.Open(path)
	require.NoError(t, err)

	var got jobs
	require.False(t, st.Get("jobs", &got))
	require.Equal(t, jobs{}, got)
}

// The library's sections and a program's live in one file without either
// writing the other's key.
func TestTheLibraryAndTheProgramShareOneFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	st, err := settings.Open(path)
	require.NoError(t, err)

	require.NoError(t, st.Set(settings.Prefix+"nav", map[string]string{"mode": "icons"}))
	require.NoError(t, st.Set("jobs", jobs{Parallel: 3}))
	require.NoError(t, st.Flush())

	var doc map[string]json.RawMessage
	b, err := os.ReadFile(path)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(b, &doc))
	require.Contains(t, doc, "fynedesygn.nav")
	require.Contains(t, doc, "jobs")
	require.Len(t, doc, 2)
}

// A value that cannot be encoded is an error at the call that made it, not a
// silent failure a second later with nobody left to tell.
func TestAValueThatCannotBeEncodedFailsAtTheCall(t *testing.T) {
	st, err := settings.Open(filepath.Join(t.TempDir(), "settings.json"))
	require.NoError(t, err)
	require.Error(t, st.Set("bad", func() {}))
	require.False(t, st.Pending())
}

/*
An extension nothing has registered is a refusal that names the cure.

A JSON document written to a file called settings.yaml is worse than a refusal:
nothing downstream would notice, and the first person to open it would find a
format nobody chose.
*/
func TestAnUnknownExtensionIsRefusedAndSaysWhatIsMissing(t *testing.T) {
	_, err := settings.Open(filepath.Join(t.TempDir(), "settings.toml"))
	require.Error(t, err)
	require.Contains(t, err.Error(), ".toml")

	require.Contains(t, settings.Registered(), ".json", "the core registers its own")
}

// The store is written to from a timer and read from the interface at once, so
// it must be safe from any goroutine.
func TestTheStoreIsSafeFromSeveralGoroutines(t *testing.T) {
	st, err := settings.Open(filepath.Join(t.TempDir(), "settings.json"))
	require.NoError(t, err)
	st.SetDelay(time.Millisecond)

	var wg sync.WaitGroup
	for i := range 4 {
		wg.Add(2)
		go func() { defer wg.Done(); _ = st.Set("jobs", jobs{Parallel: i}) }()
		go func() {
			defer wg.Done()
			var got jobs
			st.Get("jobs", &got)
			st.Has("jobs")
		}()
	}
	wg.Wait()
	require.NoError(t, st.Flush())
}

// A failed write reaches whoever asked to hear about them: the save happens on
// a timer, long after the call that caused it, so there is nobody to return it
// to.
func TestAFailedWriteIsReported(t *testing.T) {
	dir := t.TempDir()
	st, err := settings.Open(filepath.Join(dir, "settings.json"))
	require.NoError(t, err)
	st.SetDelay(10 * time.Millisecond)

	var mu sync.Mutex
	var said []string
	st.OnError(func(err error) { mu.Lock(); said = append(said, err.Error()); mu.Unlock() })

	require.NoError(t, os.Chmod(dir, 0o500))
	t.Cleanup(func() { _ = os.Chmod(dir, 0o750) })
	require.NoError(t, st.Set("jobs", jobs{Parallel: 1}))

	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(said) > 0
	}, time.Second, 10*time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	require.True(t, strings.Contains(said[0], "settings.json"), "the message names the file")
}
