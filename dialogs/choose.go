package dialogs

import (
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"

	fynetheme "fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// PickerStart is where a chooser should open: the path already in the field
// if it names a directory, otherwise its parent, otherwise the home directory.
// A leading ~ is the home directory. A field left empty, or holding something
// that no longer exists, must not leave the chooser at whatever directory the
// process happens to be in.
func PickerStart(text string) fyne.ListableURI {
	home, _ := os.UserHomeDir()
	var candidates []string
	if p := expandHome(strings.TrimSpace(text), home); p != "" {
		candidates = append(candidates, p, filepath.Dir(p))
	}
	if home != "" {
		candidates = append(candidates, home)
	}
	for _, c := range candidates {
		if fi, err := os.Stat(c); err != nil || !fi.IsDir() {
			continue
		}
		if lu, err := storage.ListerForURI(storage.NewFileURI(c)); err == nil {
			return lu
		}
	}
	return nil
}

func expandHome(p, home string) string {
	if p == "~" || strings.HasPrefix(p, "~/") || strings.HasPrefix(p, `~\`) {
		if home == "" {
			return ""
		}
		return filepath.Join(home, p[1:])
	}
	return p
}

// showSized shows a file dialog roomy: Fyne's default shows about four names,
// which is a directory read through a slot. See Roomy for the order and why
// it matters.
func showSized(win fyne.Window, d *dialog.FileDialog) { Roomy(d, win) }

// ChooseFile opens a single-file chooser starting near start, filtered when
// filter is not nil, and hands the chosen path to then. Fyne 2.8.1 has no
// multi-select chooser; an "Add..." button reopens this one.
func ChooseFile(win fyne.Window, start string, filter storage.FileFilter, then func(path string)) {
	d := dialog.NewFileOpen(func(rc fyne.URIReadCloser, err error) {
		if err != nil || rc == nil {
			return
		}
		// The chooser hands back an open handle; callers read the file
		// themselves, by path, so close it immediately rather than holding a
		// descriptor open for the life of the dialog.
		path := rc.URI().Path()
		_ = rc.Close()
		then(path)
	}, win)
	if filter != nil {
		d.SetFilter(filter)
	}
	d.SetLocation(PickerStart(start))
	showSized(win, d)
}

// ChooseFolder opens a directory chooser starting near start.
func ChooseFolder(win fyne.Window, start string, then func(path string)) {
	d := dialog.NewFolderOpen(func(lu fyne.ListableURI, err error) {
		if err != nil || lu == nil {
			return
		}
		then(lu.Path())
	}, win)
	d.SetLocation(PickerStart(start))
	showSized(win, d)
}

// BrowseButton is the affordance beside a path field: it opens the chooser
// and writes the chosen path back into the field. The field stays editable: a
// path can still be typed or pasted, which is the only way to reach somewhere
// the chooser will not show. dir picks a directory chooser.
func BrowseButton(win fyne.Window, field *widget.Entry, dir bool) *widget.Button {
	return widget.NewButtonWithIcon("Browse...", fynetheme.FolderOpenIcon(), func() {
		if dir {
			ChooseFolder(win, field.Text, field.SetText)
			return
		}
		ChooseFile(win, field.Text, nil, field.SetText)
	})
}

// WithBrowse lays a path field out with its chooser button on the right.
func WithBrowse(win fyne.Window, field *widget.Entry, dir bool) fyne.CanvasObject {
	return container.NewBorder(nil, nil, nil, BrowseButton(win, field, dir), field)
}
