//go:build windows

package fynetest

import "path/filepath"

// platformDirs is the Windows half of Sandbox: the variables os.UserHomeDir,
// os.UserConfigDir and os.UserCacheDir read on Windows instead of HOME and
// XDG_*.
func platformDirs(home string) map[string]string {
	return map[string]string{
		"USERPROFILE":  home,
		"APPDATA":      filepath.Join(home, "AppData", "Roaming"),
		"LOCALAPPDATA": filepath.Join(home, "AppData", "Local"),
	}
}
