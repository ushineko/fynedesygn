package yamlcodec_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ushineko/fynedesygn/settings"
	_ "github.com/ushineko/fynedesygn/settings/yamlcodec"
)

// listTimeout bounds the toolchain query below; it reads the module cache and
// compiles nothing.
const listTimeout = 2 * time.Minute

// window is a section with the json tags a caller would already have.
type window struct {
	Mode   string  `json:"mode"`
	Width  int     `json:"width"`
	Offset float64 `json:"offset"`
	Shown  bool    `json:"shown"`
}

/*
Importing the codec is what makes a .yaml settings file YAML.

The extension chooses the format, and the codec registers itself for its own
extensions, so nothing in the core has to name this package.
*/
func TestAYamlPathIsYamlOnceTheCodecIsImported(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.yaml")
	st, err := settings.Open(path)
	require.NoError(t, err)
	require.NoError(t, st.Set("window", window{Mode: "icons", Width: 220, Offset: 0.16, Shown: true}))
	require.NoError(t, st.Flush())

	b, err := os.ReadFile(path)
	require.NoError(t, err)
	text := string(b)
	require.Contains(t, text, "window:")
	require.Contains(t, text, "mode: icons")
	require.NotContains(t, text, "{", "this is YAML, not JSON in a YAML file")

	again, err := settings.Open(path)
	require.NoError(t, err)
	var got window
	require.True(t, again.Get("window", &got))
	require.Equal(t, window{Mode: "icons", Width: 220, Offset: 0.16, Shown: true}, got)

	require.Contains(t, settings.Registered(), ".yaml")
	require.Contains(t, settings.Registered(), ".yml")
}

/*
The two formats hold the same keys.

The YAML codec encodes through JSON, so one set of json struct tags names the
fields in both -- where a second set of yaml tags would be two sets of names
kept in step by hand, and one silent difference the first time they drift.
*/
func TestTheSameSettingsHaveTheSameKeysInEitherFormat(t *testing.T) {
	dir := t.TempDir()
	value := window{Mode: "labels", Width: 300, Offset: 0.25, Shown: false}

	asJSON := filepath.Join(dir, "settings.json")
	js, err := settings.Open(asJSON)
	require.NoError(t, err)
	require.NoError(t, js.Set("window", value))
	require.NoError(t, js.Flush())

	asYAML := filepath.Join(dir, "settings.yaml")
	ys, err := settings.Open(asYAML)
	require.NoError(t, err)
	require.NoError(t, ys.Set("window", value))
	require.NoError(t, ys.Flush())

	for _, key := range []string{"mode", "width", "offset", "shown"} {
		require.Containsf(t, read(t, asJSON), `"`+key+`"`, "%s is missing from the JSON", key)
		require.Containsf(t, read(t, asYAML), key+":", "%s is missing from the YAML", key)
	}

	// And each reads back what the other wrote, once renamed.
	swapped := filepath.Join(dir, "swapped.yaml")
	require.NoError(t, os.WriteFile(swapped, []byte(read(t, asYAML)), 0o600))
	st, err := settings.Open(swapped)
	require.NoError(t, err)
	var got window
	require.True(t, st.Get("window", &got))
	require.Equal(t, value, got)
}

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(b)
}

/*
A whole number stays whole.

encoding/json decodes every number as a float64 unless asked otherwise, and a
lifetime of 216000 ticks written through one comes out as 2.16e+05 -- a settings
file holding numbers nobody set.
*/
func TestAWholeNumberIsWrittenWhole(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.yaml")
	st, err := settings.Open(path)
	require.NoError(t, err)
	require.NoError(t, st.Set("window", window{Width: 216000, Offset: 0.5}))
	require.NoError(t, st.Flush())

	text := read(t, path)
	require.Contains(t, text, "width: 216000")
	require.NotContains(t, text, "e+")
	require.Contains(t, text, "offset: 0.5", "and a fraction stays a fraction")
}

// A file that is not YAML is reported like any other unreadable settings file,
// rather than being read as an empty one -- and, since spec 019, left where it
// is, with the parser's own complaint carried in the error.
func TestAFileThatIsNotYamlIsReportedAndLeftAlone(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.yaml")
	body := "\tthis: [is not\n\t  yaml"
	require.NoError(t, os.WriteFile(path, []byte(body), 0o600))

	_, err := settings.Open(path)
	require.Error(t, err)

	var parse *settings.ParseError
	require.ErrorAs(t, err, &parse)
	require.Equal(t, path, parse.Path)

	require.NoFileExists(t, path+".bad")
	require.Equal(t, body, read(t, path), "the user's file was modified")
}

/*
The core does not reach the YAML package.

A subpackage is only worth having if a program that writes JSON does not carry a
parser it never calls, so this asks the toolchain rather than trusting the
imports as read.
*/
func TestTheCoreDoesNotLinkTheYamlPackage(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), listTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, "go", "list", "-deps",
		"github.com/ushineko/fynedesygn/settings",
		"github.com/ushineko/fynedesygn/shell").CombinedOutput()
	require.NoErrorf(t, err, "asking the toolchain: %s", out)
	for _, dep := range strings.Fields(string(out)) {
		require.NotContainsf(t, dep, "yaml", "%s reaches a YAML package", dep)
	}
}
