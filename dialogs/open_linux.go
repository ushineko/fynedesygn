//go:build !windows && !darwin

package dialogs

func opener(path string) (string, []string) { return "xdg-open", []string{path} }
