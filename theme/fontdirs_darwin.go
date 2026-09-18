//go:build darwin

package theme

import (
	"os"
	"path/filepath"
)

// platformFontDirs are the directories scanned for fonts on macOS: the
// system's, the machine's, and the user's Library.
func platformFontDirs() []string {
	dirs := []string{"/System/Library/Fonts", "/Library/Fonts"}
	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, "Library", "Fonts"))
	}
	return dirs
}
