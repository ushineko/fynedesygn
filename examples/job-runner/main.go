/*
Command job-runner is a fynedesygn example: a job with real steps beside a
log, run through PerformCancellable so the busy popup carries a Cancel button,
with the step list and the log updated in place while the job runs and one
banner summarising the result.
*/
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

	fd "github.com/ushineko/fynedesygn"
	"github.com/ushineko/fynedesygn/logpane"
	"github.com/ushineko/fynedesygn/shell"
	"github.com/ushineko/fynedesygn/steps"
	"github.com/ushineko/fynedesygn/widgets"
)

func main() {
	shell.Run(options(newJob(300*time.Millisecond), true))
}

// job is the program's state: the step list, the log, and the outcome.
type job struct {
	steps   *steps.List
	pane    *logpane.Pane
	tick    time.Duration
	runs    int
	outcome string
}

var stepNames = []string{"Fetch sources", "Compile", "Run checks", "Package"} //nolint:gochecknoglobals // a fixed list

func newJob(tick time.Duration) *job {
	return &job{steps: steps.New(stepNames...), pane: logpane.New(logpane.NewModel(500)), tick: tick, outcome: "not run"}
}

// run is the fake job. It is called off the UI thread by Perform, so every
// step and log update hops through fyne.Do; the pump draws the log.
func (j *job) run(ctx context.Context, s *shell.Shell, fail bool) error {
	stopPump := j.pane.Pump()
	defer stopPump()
	fyne.Do(func() { j.steps.Reset(); j.outcome = "running"; s.RedrawStatus() })
	for i, name := range stepNames {
		fyne.Do(func() { j.steps.Advance(i, "working") })
		j.pane.Log(logpane.Info, "starting "+name)
		for k := range 3 {
			// Check before waiting as well as while waiting: a select with a
			// ready timer and a ready cancellation picks either at random, and
			// a job that was cancelled before it started must not do work.
			if ctx.Err() == nil {
				select {
				case <-ctx.Done():
				case <-time.After(j.tick):
				}
			}
			if err := ctx.Err(); err != nil {
				fyne.Do(func() { j.steps.Stop(steps.Cancelled, "stopped"); j.outcome = "cancelled"; s.RedrawStatus() })
				j.pane.Log(logpane.Warn, "cancelled during "+name)
				return fmt.Errorf("job cancelled during %s: %w", name, err)
			}
			j.pane.Log(logpane.Debug, fmt.Sprintf("%s: part %d of 3", name, k+1))
		}
		if fail && i == 2 {
			fyne.Do(func() { j.steps.Stop(steps.Failed, "2 checks failed"); j.outcome = "failed"; s.RedrawStatus() })
			j.pane.Log(logpane.Error, "2 checks failed")
			return errors.New("2 checks failed")
		}
		fyne.Do(func() { j.steps.Finish(i, "ok") })
		j.pane.Log(logpane.Info, name+" done")
	}
	fyne.Do(func() {
		j.runs++
		j.outcome = "finished"
		s.Flash("The job finished.", fd.StatusGood)
		s.RedrawStatus()
	})
	return nil
}

func options(j *job, cancellable bool) shell.Options {
	return shell.Options{
		AppID: "io.ushineko.fynedesygn.example.jobrunner",
		Name:  "job-runner",
		Sections: []shell.Section{
			shell.NewSection("Job", fynetheme.MediaPlayIcon, func(s *shell.Shell) fyne.CanvasObject {
				return buildJob(s, j, cancellable)
			}).OnDetach(func() { j.steps.Detach(); j.pane.Detach() }),
		},
		StatusBar: func(*shell.Shell) []fyne.CanvasObject {
			st := fd.StatusInfo
			switch j.outcome {
			case "finished":
				st = fd.StatusGood
			case "cancelled":
				st = fd.StatusWarn
			case "failed":
				st = fd.StatusBad
			}
			return []fyne.CanvasObject{widgets.Dim("job"), widgets.StatusText(j.outcome, st), widgets.Sep(), widgets.Dim("runs"), widget.NewLabel(fmt.Sprintf("%d", j.runs))}
		},
	}
}

func buildJob(s *shell.Shell, j *job, cancellable bool) fyne.CanvasObject {
	start := func(fail bool) func() {
		return func() {
			fn := func(ctx context.Context) error { return j.run(ctx, s, fail) }
			if cancellable {
				s.PerformCancellable("Running the job...", fn)
			} else {
				s.Perform("Running the job...", fn)
			}
		}
	}
	runBtn := widget.NewButtonWithIcon("Run", fynetheme.MediaPlayIcon(), start(false))
	runBtn.Importance = widget.HighImportance
	failBtn := widget.NewButton("Run and fail", start(true))
	s.Gate(runBtn, failBtn)

	left := container.NewVBox(
		widget.NewLabelWithStyle("Steps", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		j.steps.Widget(),
	)
	right := j.pane.Widget(logpane.Options{Title: "Output", Height: 300, Clipboard: s.App.Clipboard(), Flash: s.Flash})
	return container.NewBorder(
		container.NewVBox(
			widgets.Heading("Job", "A job with real steps: the list on the left advances, the log on the right streams, and the busy popup carries Cancel. Nothing in this section moves while it runs."),
			container.NewHBox(runBtn, failBtn),
		),
		nil, nil, nil,
		container.NewBorder(nil, nil, widgets.FixedWidth(left, 260), nil, right),
	)
}
