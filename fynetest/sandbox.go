package fynetest

import (
	"os"
	"path/filepath"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
)

// Sandbox points HOME, XDG_CONFIG_HOME, XDG_DATA_HOME and XDG_CACHE_HOME (and
// on Windows USERPROFILE, APPDATA and LOCALAPPDATA) at a fresh t.TempDir()
// through t.Setenv, creates the directories, and returns the home directory.
// The XDG and Windows variables get the subdirectories they would have under
// a real home (.config, .local/share, .cache; AppData/Roaming, AppData/Local)
// so config, data and cache do not collide. A test whose result depends on the
// developer's machine is not a test.
func Sandbox(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	dirs := map[string]string{
		"HOME":            home,
		"XDG_CONFIG_HOME": filepath.Join(home, ".config"),
		"XDG_DATA_HOME":   filepath.Join(home, ".local", "share"),
		"XDG_CACHE_HOME":  filepath.Join(home, ".cache"),
	}
	for name, dir := range platformDirs(home) {
		dirs[name] = dir
	}
	for name, dir := range dirs {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatalf("fynetest.Sandbox: create %s: %v", name, err)
		}
		t.Setenv(name, dir)
	}
	return home
}

// App is test.NewApp with t.Cleanup(app.Quit), run inside Sandbox so nothing
// the app or the code under test writes lands outside t.TempDir().
func App(t *testing.T) fyne.App {
	t.Helper()
	Sandbox(t)
	app := test.NewApp()
	t.Cleanup(app.Quit)
	return app
}
