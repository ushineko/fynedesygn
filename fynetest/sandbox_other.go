//go:build !windows

package fynetest

// platformDirs is empty off Windows: HOME and the XDG variables are all the
// standard library and Fyne consult.
func platformDirs(string) map[string]string {
	return nil
}
