//go:build !linux

package theme

// ApplyCursorTheme is a no-op away from Linux. The problem it works around is
// specific to GLFW's Wayland backend; the macOS and Windows backends use the
// system cursor and need no help.
func ApplyCursorTheme() {}
