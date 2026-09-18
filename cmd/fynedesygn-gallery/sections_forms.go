package main

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	fynetheme "fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	fd "github.com/ushineko/fynedesygn"
	"github.com/ushineko/fynedesygn/dialogs"
	"github.com/ushineko/fynedesygn/forms"
	"github.com/ushineko/fynedesygn/logpane"
	"github.com/ushineko/fynedesygn/shell"
	"github.com/ushineko/fynedesygn/widgets"
)

// --- Dialogs -----------------------------------------------------------------

func buildDialogs(s *shell.Shell) fyne.CanvasObject {
	win := s.Window
	path := widget.NewEntry()
	path.SetPlaceHolder("a path to open with the desktop")

	destructive := widget.NewButton("Remove...", func() {
		dialogs.ConfirmDestructive(win, "Remove the example?",
			"The example entry is removed from the list and its cache is deleted. "+
				"The file it points at is not touched.", "Remove",
			func() { s.Flash("Removed. The file is still there.", fd.StatusGood) })
	})
	destructive.Importance = widget.DangerImportance

	withBody := widget.NewButton("Remove with a choice...", func() {
		also := widget.NewCheck("Also delete the file on disk", nil)
		d := dialogs.ConfirmWithBody(win, "Remove the example?", container.NewVBox(
			widgets.Wrapped("The entry goes either way. Tick the box to delete the file too; otherwise it stays where it is."), also,
		), "Remove", func() {
			s.Flash(fmt.Sprintf("Removed. File deleted: %v.", also.Checked), fd.StatusGood)
		})
		d.Show()
	})

	prompt := widget.NewButton("Rename...", func() {
		name := widget.NewEntry()
		name.SetText("example")
		dialogs.Prompt(win, "Rename the example", "Rename", widget.NewForm(widget.NewFormItem("New name", name)),
			func() { s.Flash("Renamed to "+name.Text+".", fd.StatusGood) })
	})

	detail := widget.NewButton("Show details...", func() {
		body := widgets.Card("Checks",
			widgets.FactRow("Syntax", "ok", fd.StatusGood),
			widgets.FactRow("References", "2 unresolved", fd.StatusWarn),
			widgets.Note("A read-only answer to a question just asked; it goes away when read.", fd.StatusInfo),
		)
		dialogs.ShowDetail(win, "Details", body, 600, 360)
	})

	decide := widget.NewButton("Ask from a worker", func() {
		s.Perform("Asking...", func(context.Context) error {
			yes := dialogs.Decide(win, "Continue the job?", "The worker goroutine is blocked on this answer.", "Continue", "Stop")
			fyne.Do(func() { s.Flash(fmt.Sprintf("The worker was told: %v.", yes), fd.StatusInfo) })
			return nil
		})
	})

	open := widget.NewButton("Open with the desktop", func() {
		if err := dialogs.OpenPath(path.Text); err != nil {
			s.Flash(err.Error()+" Nothing else was affected.", fd.StatusWarn)
			return
		}
		s.Flash("Handed to "+dialogs.Opener()+".", fd.StatusGood)
	})
	s.Gate(destructive, withBody, prompt, detail, decide, open)

	return container.NewVScroll(container.NewVBox(
		widgets.Heading("Dialogs", "The shared dialog shapes. Every destructive confirmation says what is not touched; every chooser is resized after it is shown, which is what keeps Fyne 2.8.1 from crashing."),
		widgets.Card("dialogs.ConfirmDestructive / ConfirmWithBody / Prompt / ShowDetail",
			container.NewHBox(destructive, withBody, prompt, detail)),
		widgets.Card("dialogs.WithBrowse / OpenPath",
			dialogs.WithBrowse(win, path, false),
			container.NewHBox(open),
			widgets.DimWrapped("Browse opens the file chooser starting at the field's directory, its parent, or home."),
		),
		widgets.Card("dialogs.Decide",
			widgets.Wrapped("A yes-or-no question asked from a worker goroutine through fyne.Do and a channel. Never called on the UI thread."),
			container.NewHBox(decide),
		),
	))
}

// --- Log ---------------------------------------------------------------------

// logDemo feeds a pane from a fake job while its section is on screen.
type logDemo struct {
	pane *logpane.Pane
	stop func()
}

func newLogDemo() *logDemo { return &logDemo{pane: logpane.New(logpane.NewModel(300))} }

func (d *logDemo) build(s *shell.Shell) fyne.CanvasObject {
	d.detach()
	start := widget.NewButton("Start a noisy job", func() {
		if d.stop != nil {
			return
		}
		stopPump := d.pane.Pump()
		ctx, cancel := context.WithCancel(context.Background())
		d.stop = func() {
			cancel()
			stopPump()
			d.stop = nil
		}
		go func() {
			levels := []logpane.Level{logpane.Info, logpane.Info, logpane.Debug, logpane.Warn, logpane.Info, logpane.Error}
			for i := 0; ; i++ {
				select {
				case <-ctx.Done():
					return
				case <-time.After(50 * time.Millisecond):
					d.pane.Log(levels[i%len(levels)], fmt.Sprintf("step %d: something happened in the job", i))
				}
			}
		}()
	})
	stop := widget.NewButton("Stop", func() {
		if d.stop != nil {
			d.stop()
		}
	})
	return container.NewVBox(
		widgets.Heading("Log", "A fixed-height list of monospace rows drawn on a 100 ms timer, following the tail until you scroll up. The cap here is 300 lines so the drop shows."),
		container.NewHBox(start, stop),
		d.pane.Widget(logpane.Options{
			Title:     "logpane.Pane",
			Clipboard: s.App.Clipboard(),
			Flash:     s.Flash,
			OnClear:   func() {},
		}),
		widgets.DimWrapped("Copy puts the whole log on the clipboard with a line saying how many older rows were dropped; Clear empties it."),
	)
}

func (d *logDemo) detach() {
	d.pane.Detach()
}

// --- Forms -------------------------------------------------------------------

// formDemo is a settings form over an in-memory document with Save and
// Revert, plus the slider and numeric shapes.
type formDemo struct {
	saved   map[string]string
	pending map[string]string
	factor  float64
}

func newFormDemo() *formDemo {
	saved := map[string]string{"name": "example", "mode": "safe", "enabled": "true", "dir": "", "jobs": "4"}
	return &formDemo{saved: saved, pending: cloneMap(saved), factor: 1.5}
}

func cloneMap(m map[string]string) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func (d *formDemo) build(s *shell.Shell) fyne.CanvasObject {
	f := forms.New(
		forms.Entry("name", "Name", "what to call it"),
		forms.Select("mode", "Mode", []string{"fast", "safe"}),
		forms.Check("enabled", "Enabled"),
		forms.Path(s.Window, "dir", "Directory", true),
		forms.Numeric("jobs", "Parallel jobs", 1, 16),
	)
	f.Set(d.pending)
	status := widgets.Dim("Nothing changed.")
	f.OnChange = func(k, v string) {
		d.pending[k] = v
		status.(*widget.Label).SetText(fmt.Sprintf("Changed %s; not saved yet.", k))
	}
	save := widget.NewButtonWithIcon("Save", fynetheme.DocumentSaveIcon(), func() {
		d.saved = cloneMap(f.Values())
		d.pending = cloneMap(d.saved)
		status.(*widget.Label).SetText("Saved.")
		s.Flash("Saved the example settings.", fd.StatusGood)
	})
	save.Importance = widget.HighImportance
	revert := widget.NewButtonWithIcon("Revert", fynetheme.ContentUndoIcon(), func() {
		d.pending = cloneMap(d.saved)
		f.Set(d.pending)
		status.(*widget.Label).SetText("Put the fields back to the saved values. Nothing was written.")
	})
	s.Gate(save, revert)

	slider := forms.NewSliderEntry(forms.SliderOptions{
		Min: 0.1, Max: 10, Step: 0.1, Value: d.factor,
		Format: func(v float64) string { return strconv.FormatFloat(v, 'f', 1, 64) },
		Commit: func(v float64) {
			d.factor = v
			s.Flash(fmt.Sprintf("Committed factor %.1f (one write per gesture).", v), fd.StatusInfo)
		},
		OnInvalid: func(text string) { s.Flash(strconv.Quote(text)+" is not a number.", fd.StatusWarn) },
	})
	reset := widget.NewButtonWithIcon("Reset", fynetheme.ContentUndoIcon(), func() { slider.Set(1.5) })

	numeric := forms.NumericEntry(6, 48, func(n int) {
		status.(*widget.Label).SetText(fmt.Sprintf("Console size %d is valid.", n))
	})
	numeric.SetPlaceHolder("6 to 48")

	return container.NewVScroll(container.NewVBox(
		widgets.Heading("Forms", "Widgets held apart from the section so Save and Revert are testable and a Revert reverts. Programmatic writes never fire the change hook."),
		widgets.Card("forms.Form", f.Widget(), container.NewHBox(save, revert), status),
		forms.Group("forms.SliderEntry", "The slider updates the field while dragging and commits on release; the field commits on Enter.",
			container.NewBorder(nil, nil, widgets.FixedWidth(widgets.Dim("Multiplier"), widgets.LabelWidth),
				container.NewHBox(widgets.Dim("default 1.5"), reset), slider.Widget())),
		forms.Group("forms.NumericEntry", "Validated to a range; the hook fires only for valid values.",
			container.NewHBox(widgets.FixedWidth(numeric, forms.NumericWidth), widgets.Dim("console text size"))),
	))
}
