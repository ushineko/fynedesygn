package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ushineko/fynedesygn/fynetest"
	"github.com/ushineko/fynedesygn/shell"
)

func TestChangesSaveThemselvesAndRevertReadsTheFileBack(t *testing.T) {
	app := fynetest.App(t)
	path := filepath.Join(t.TempDir(), "settings.json")
	d := newDocument(path)
	s := shell.Headless(app, options(d))
	o := s.Current().Build(s)

	sel := fynetest.FindSelect(o)
	require.Equal(t, "safe", sel.Selected)
	sel.SetSelected("fast")
	require.True(t, d.saver.Pending(), "a user change schedules a save")
	d.saver.Flush()
	b, err := os.ReadFile(path)
	require.NoError(t, err)
	var saved map[string]string
	require.NoError(t, json.Unmarshal(b, &saved))
	require.Equal(t, "fast", saved["mode"])

	// Type something (and flush it, so no timer is left to fire mid-test:
	// under the test driver the saver's fyne.Do would draw a banner on its own
	// goroutine), then edit the file behind the program's back and Revert.
	fynetest.FindEntry(o).SetText("typed")
	d.saver.Flush()
	saved["name"] = "from-disk"
	b, _ = json.Marshal(saved)
	require.NoError(t, os.WriteFile(path, b, 0o600))
	revert := fynetest.FindButton(o, "Revert to the file")
	require.NotNil(t, revert)
	revert.OnTapped()
	require.Equal(t, "from-disk", fynetest.FindEntry(o).Text)
	require.Equal(t, "from-disk", d.values["name"])
	require.Equal(t, "Put the fields back to what "+path+" says. Nothing was written.", s.FlashText())

	for _, sec := range s.Sections() {
		require.NotNil(t, sec.Build(s), sec.Title())
	}
}
