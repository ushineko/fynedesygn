package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	fynetheme "fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ushineko/fynedesygn/logpane"
	"github.com/ushineko/fynedesygn/shell"
	"github.com/ushineko/fynedesygn/widgets"
	"github.com/ushineko/fynedesygn/wizard"
)

/*
The Wizard section.

A wizard owns its window, as a glance window does, so the gallery opens one
beside itself rather than drawing one inside a section. wizard.NewIn puts it
over the gallery's app and leaves the gallery's theme alone, so the sample
is drawn in whatever scheme the Appearance section has chosen.
*/
func buildWizard(s *shell.Shell) fyne.CanvasObject {
	open := func(fail bool) func() {
		return func() { wizard.NewIn(s.App, sampleWizard(fail)).Window.Show() }
	}
	run := widget.NewButtonWithIcon("Open a sample wizard", fynetheme.MediaPlayIcon(), open(false))
	run.Importance = widget.HighImportance
	return container.NewVBox(
		widgets.Heading("Wizard",
			"The third window archetype: pages in a fixed order with Back, Next and Cancel. Next waits for a "+
				"valid page, the progress page is the point of no return, and the message line and fixed-width "+
				"buttons keep the window still while a job runs. See docs/wizard.md."),
		container.NewHBox(run, widget.NewButton("Open one whose job fails", open(true))),
		widgets.Card("Pages",
			widgets.PlainRow("wizard.Welcome", "Markdown that says what will happen"),
			widgets.PlainRow("wizard.Licence", "Markdown and an \"I accept\" check that Next waits for"),
			widgets.PlainRow("wizard.Directory", "an entry with Browse, checked as it is typed"),
			widgets.PlainRow("wizard.Form", "a forms.Form of options"),
			widgets.PlainRow("wizard.Summary", "facts read each time the page is entered"),
			widgets.PlainRow("wizard.Progress", "a step list beside a log, with a cancellable job"),
			widgets.PlainRow("wizard.Finish", "Markdown and checks that the Result reports"),
		),
	)
}

func sampleWizard(fail bool) wizard.Options {
	stepNames := []string{"Prepare", "Copy", "Record"}
	return wizard.Options{
		Name: "Sample wizard",
		Pages: []wizard.Page{
			wizard.Welcome("Welcome", "A sample wizard from the gallery. It **installs nothing**."),
			wizard.Directory("Location", "Pretend to install into:", "/opt/sample", nil),
			wizard.Summary("Ready", "Install", func() []wizard.Fact {
				return []wizard.Fact{{Label: "Steps", Value: fmt.Sprint(len(stepNames))}}
			}),
			wizard.Progress("Working", stepNames, "Done.", func(ctx context.Context, r *wizard.Reporter) error {
				for i, name := range stepNames {
					r.Advance(i, "")
					select {
					case <-ctx.Done():
						return fmt.Errorf("%s: %w", name, ctx.Err())
					case <-time.After(700 * time.Millisecond):
					}
					r.Log(logpane.Info, name+" done")
					if fail && i == 1 {
						return errors.New("the sample was told to fail")
					}
					r.Finish(i, "ok")
				}
				return nil
			}),
			wizard.Finish("Finished", "The sample is finished.", widget.NewCheck("Open the gallery again", nil)),
		},
	}
}
