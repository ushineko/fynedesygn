//go:build linux

package theme

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCursorThemeIsReadFromTheDesktopConfigInOrderOfAuthority(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XCURSOR_THEME", "")
	t.Setenv("XCURSOR_SIZE", "")
	require.NoError(t, os.MkdirAll(filepath.Join(home, ".config", "gtk-3.0"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(home, ".config", "gtk-3.0", "settings.ini"),
		[]byte("[Settings]\ngtk-cursor-theme-name=Adwaita\ngtk-cursor-theme-size=32\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(home, ".config", "kcminputrc"),
		[]byte("[Mouse]\ncursorTheme=Breeze_Light\ncursorSize=24\n"), 0o600))

	ApplyCursorTheme()
	require.Equal(t, "Breeze_Light", os.Getenv("XCURSOR_THEME"), "kcminputrc outranks GTK")
	require.Equal(t, "24", os.Getenv("XCURSOR_SIZE"))
}

func TestCursorThemeNeverOverridesWhatTheSessionExported(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XCURSOR_THEME", "Mine")
	t.Setenv("XCURSOR_SIZE", "")
	require.NoError(t, os.WriteFile(filepath.Join(home, ".gtkrc-2.0"),
		[]byte(`gtk-cursor-theme-name="Other"`+"\n"), 0o600))
	ApplyCursorTheme()
	require.Equal(t, "Mine", os.Getenv("XCURSOR_THEME"))
}

func TestGtkrcQuotedValuesAreUnquotedAndSectionsIgnored(t *testing.T) {
	p := filepath.Join(t.TempDir(), "gtkrc")
	require.NoError(t, os.WriteFile(p, []byte("# comment\n[weird]\ngtk-cursor-theme-name = \"Bibata\"\ngtk-cursor-theme-size=\n"), 0o600))
	name, size := readCursorConfig(p, "gtk-cursor-theme-name", "gtk-cursor-theme-size")
	require.Equal(t, "Bibata", name)
	require.Empty(t, size)
	name, _ = readCursorConfig(filepath.Join(t.TempDir(), "missing"), "a", "b")
	require.Empty(t, name)
}
