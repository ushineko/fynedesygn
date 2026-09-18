//go:build !windows && !darwin

package theme

import (
	"os"
	"path/filepath"
)

// platformFontDirs are the directories scanned for fonts on Linux and the
// BSDs: the system directories and the XDG user directory, plus the older
// ~/.fonts that many users still populate.
func platformFontDirs() []string {
	dirs := []string{"/usr/share/fonts", "/usr/local/share/fonts"}
	if data := os.Getenv("XDG_DATA_HOME"); data != "" {
		dirs = append(dirs, filepath.Join(data, "fonts"))
	}
	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, ".local", "share", "fonts"), filepath.Join(home, ".fonts"))
	}
	return dirs
}
