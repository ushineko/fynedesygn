//go:build windows

package dialogs

func opener(path string) (string, []string) { return "explorer", []string{path} }
