//go:build darwin

package dialogs

func opener(path string) (string, []string) { return "open", []string{path} }
