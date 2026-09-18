/*
Command settings is a fynedesygn example: a form over a JSON document in the
user's configuration directory, saved automatically a second after the last
change through forms.Saver, with Revert, the standard Appearance section, and
a pending save flushed before the process restarts.
*/
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	fynetheme "fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	fd "github.com/ushineko/fynedesygn"
	"github.com/ushineko/fynedesygn/forms"
	"github.com/ushineko/fynedesygn/shell"
	"github.com/ushineko/fynedesygn/widgets"
)

func main() {
	dir, err := os.UserConfigDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	shell.Run(options(newDocument(filepath.Join(dir, "fynedesygn-settings-example", "settings.json"))))
}

// document is the settings file: a flat map, which is what a form reads and
// writes. Unknown keys survive a save.
type document struct {
	path   string
	values map[string]string
	saver  *forms.Saver
	save   func() // set by options once there is a shell to flash on
}

func defaults() map[string]string {
	return map[string]string{"name": "example", "mode": "safe", "enabled": "true", "dir": "", "jobs": "4"}
}

func newDocument(path string) *document {
	d := &document{path: path, values: defaults()}
	d.load()
	return d
}

// load reads the file; a missing file means the defaults.
func (d *document) load() {
	b, err := os.ReadFile(d.path)
	if err != nil {
		return
	}
	var v map[string]string
	if json.Unmarshal(b, &v) == nil {
		for k, val := range v {
			d.values[k] = val
		}
	}
}

// write saves the document, creating its directory.
func (d *document) write() error {
	if err := os.MkdirAll(filepath.Dir(d.path), 0o750); err != nil {
		return fmt.Errorf("create %s: %w", filepath.Dir(d.path), err)
	}
	b, err := json.MarshalIndent(d.values, "", "  ")
	if err != nil {
		return fmt.Errorf("encode settings: %w", err)
	}
	if err := os.WriteFile(d.path, b, 0o600); err != nil {
		return fmt.Errorf("write %s: %w", d.path, err)
	}
	return nil
}

func options(d *document) shell.Options {
	return shell.Options{
		AppID:   "io.ushineko.fynedesygn.example.settings",
		Name:    "settings",
		OnStart: func(s *shell.Shell) { wire(s, d) },
		OnStop:  func(*shell.Shell) { d.saver.Flush() },
		Sections: []shell.Section{
			shell.NewSection("Settings", fynetheme.SettingsIcon, func(s *shell.Shell) fyne.CanvasObject {
				wire(s, d)
				return buildSettings(s, d)
			}),
			shell.AppearanceSection("settings.json written 1 s after the last change"),
		},
		StatusBar: func(*shell.Shell) []fyne.CanvasObject {
			return []fyne.CanvasObject{widgets.Dim("file"), widget.NewLabel(d.path)}
		},
	}
}

// wire connects the saver to the shell once; Headless shells skip OnStart, so
// the section does it too.
func wire(s *shell.Shell, d *document) {
	if d.saver != nil {
		return
	}
	d.save = func() {
		if err := d.write(); err != nil {
			s.Report("Saving", err)
			return
		}
		s.Flash("Saved.", fd.StatusGood)
	}
	d.saver = forms.NewSaver(0, d.save)
}

func buildSettings(s *shell.Shell, d *document) fyne.CanvasObject {
	f := forms.New(
		forms.Entry("name", "Name", "what to call it"),
		forms.Select("mode", "Mode", []string{"fast", "safe"}),
		forms.Check("enabled", "Enabled"),
		forms.Path(s.Window, "dir", "Directory", true),
		forms.Numeric("jobs", "Parallel jobs", 1, 16),
	)
	f.Set(d.values)
	// Every user change lands in the document and schedules one coalesced
	// save; Set above went through the guard and scheduled nothing.
	f.OnChange = func(k, v string) {
		d.values[k] = v
		d.saver.Schedule()
	}
	revert := widget.NewButtonWithIcon("Revert to the file", fynetheme.ContentUndoIcon(), func() {
		d.values = defaults()
		d.load()
		f.Set(d.values)
		s.Flash("Put the fields back to what "+d.path+" says. Nothing was written.", fd.StatusInfo)
	})
	saveNow := widget.NewButtonWithIcon("Save now", fynetheme.DocumentSaveIcon(), func() {
		d.saver.Schedule()
		d.saver.Flush()
	})
	s.Gate(revert, saveNow)
	return container.NewVScroll(container.NewVBox(
		widgets.Heading("Settings", "A form over a JSON file. Changes save themselves a second after you stop; Revert reads the file back into the fields."),
		f.Widget(),
		container.NewHBox(saveNow, revert),
		widgets.Dim("The form's widgets are held apart from the section, so Revert can put values back at once instead of waiting for a rebuild."),
	))
}
