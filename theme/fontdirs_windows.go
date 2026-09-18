//go:build windows

package theme

import (
	"os"
	"path/filepath"
)

// platformFontDirs are the directories scanned for fonts on Windows: the
// system fonts folder and the per-user folder that Windows 10 and later use
// for fonts installed without administrator rights.
func platformFontDirs() []string {
	var dirs []string
	windir := os.Getenv("WINDIR")
	if windir == "" {
		windir = `C:\Windows`
	}
	dirs = append(dirs, filepath.Join(windir, "Fonts"))
	if local := os.Getenv("LOCALAPPDATA"); local != "" {
		dirs = append(dirs, filepath.Join(local, "Microsoft", "Windows", "Fonts"))
	}
	return dirs
}
