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
		container.NewHBox(run, widget.NewButton("Open one whose job fails", open(true)),
			widget.NewButton("Open a confirm window", func() { wizard.NewConfirmIn(s.App, sampleConfirm()).Window.Show() })),
		widgets.Card("Pages",
			widgets.PlainRow("wizard.Welcome", "Markdown that says what will happen"),
			widgets.PlainRow("wizard.Licence", "Markdown and an \"I accept\" check that Next waits for"),
			widgets.PlainRow("wizard.Directory", "an entry with Browse, checked as it is typed"),
			widgets.PlainRow("wizard.Form", "a forms.Form of options"),
			widgets.PlainRow("wizard.Summary", "facts read each time the page is entered"),
			widgets.PlainRow("wizard.Progress", "a step list beside a log, with a cancellable job"),
			widgets.PlainRow("wizard.Finish", "Markdown and checks that the Result reports"),
		),
		widgets.Card("One question",
			widgets.PlainRow("wizard.RunConfirm", "ask once, run the job, report: an uninstaller's shape"),
			widgets.PlainRow("wizard.ConfirmStep", "a second question after the job, built from what it found"),
		),
	)
}

// sampleConfirm is the wizard package's other shape: ask once, run, report.
func sampleConfirm() wizard.ConfirmOptions {
	return wizard.ConfirmOptions{
		Name: "Sample confirm window", Question: "Remove the sample?",
		Detail: "A sample from the gallery. It **removes nothing**.",
		Action: "Remove", Destructive: true, Done: "The sample was removed.",
		Job: sampleJob,
		Then: &wizard.ConfirmStep{
			Action: "Remove it too", Decline: "Keep it",
			Ask: func() (string, string, bool) {
				return "The sample left a file it made.", "- `sample-cache.db` (nothing is removed)", true
			},
			Job: sampleJob, Done: "The sample was removed, with the file it made.",
		},
	}
}

func sampleJob(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("removing: %w", ctx.Err())
	case <-time.After(time.Second):
		return nil
	}
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
				defer r.Progress(1, fmt.Sprintf("%d of %d steps", len(stepNames), len(stepNames)), "")
				for i, name := range stepNames {
					r.Advance(i, "")
					select {
					case <-ctx.Done():
						return fmt.Errorf("%s: %w", name, ctx.Err())
					case <-time.After(700 * time.Millisecond):
					}
					r.Log(logpane.Info, name+" done")
					r.Progress(float64(i+1)/float64(len(stepNames)), fmt.Sprintf("%d of %d steps", i+1, len(stepNames)), name)
					if fail && i == 1 {
						return errors.New("the sample was told to fail")
					}
					r.Finish(i, "ok")
				}
				return nil
			}).WithBar(),
			wizard.Finish("Finished", "The sample is finished.", widget.NewCheck("Open the gallery again", nil)),
		},
	}
}
