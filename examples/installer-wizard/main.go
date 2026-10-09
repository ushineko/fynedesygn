/*
Command installer-wizard is a fynedesygn example: a wizard that goes through
every standard page (welcome, licence, directory, options, summary,
progress and finish) to install nothing. The job is fake and slow enough to
cancel; -fail makes it fail at its third step, and -scheme chooses the colour
scheme, for tools/screenshot.sh. -confirm shows the other shape instead: a
window that asks once ("Uninstall Example 1.0?"), runs, and reports.
-yes with -confirm skips the question. -leftovers with -confirm asks a
second question after it: whether to remove the files the program left.

Not a shell program: wizard.Run owns its window.
*/
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2/widget"

	"github.com/ushineko/fynedesygn/forms"
	"github.com/ushineko/fynedesygn/logpane"
	fdtheme "github.com/ushineko/fynedesygn/theme"
	"github.com/ushineko/fynedesygn/wizard"
)

func main() {
	fail := flag.Bool("fail", false, "make the job fail at its third step")
	scheme := flag.String("scheme", "", "colour scheme; default the platform's")
	confirm := flag.Bool("confirm", false, "show the confirm window, as an uninstaller would")
	yes := flag.Bool("yes", false, "with -confirm, do not ask")
	leftovers := flag.Bool("leftovers", false, "with -confirm, ask to remove the files the program left")
	flag.Parse()
	var ap *fdtheme.Appearance
	if *scheme != "" {
		a := fdtheme.DefaultAppearance()
		a.Scheme = *scheme
		ap = &a
	}
	if *confirm {
		o := confirmOptions(400*time.Millisecond, *fail)
		o.Appearance, o.SkipQuestion = ap, *yes
		if *leftovers {
			o.Then = leftoversStep(400 * time.Millisecond)
		}
		r := wizard.RunConfirm(o)
		fmt.Println("outcome:", r.Outcome)
		return
	}
	o := options(400*time.Millisecond, *fail)
	o.Appearance = ap
	r := wizard.Run(o)
	fmt.Println("outcome:", r.Outcome, "launch:", r.Checks[launchNow])
}

const launchNow = "Launch Example now"

var stepNames = []string{"Check the directory", "Copy files", "Add the launcher entry", "Write the record"} //nolint:gochecknoglobals // a fixed list

const licence = `MIT License

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software, to deal in the software without restriction.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND.`

func options(tick time.Duration, fail bool) wizard.Options {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "/tmp"
	}
	dir := wizard.Directory("Location", "Install Example into:", filepath.Join(home, ".local", "share", "example"),
		func(s string) error {
			if !filepath.IsAbs(s) {
				return errors.New("choose an absolute path")
			}
			return nil
		})
	opts := forms.New(forms.Check("menu", "Add a launcher entry"), forms.Check("path", "Add a link in ~/.local/bin"))

	return wizard.Options{
		AppID: "io.ushineko.fynedesygn.example.installerwizard",
		Name:  "Example installer",
		Pages: []wizard.Page{
			wizard.Welcome("Welcome", "This installs **Example 1.0**, which does nothing.\n\nNothing is written anywhere: the job is a pretence."),
			wizard.Licence(licence),
			dir,
			wizard.Form("Options", opts, map[string]string{"menu": "true", "path": "true"}, nil),
			wizard.Summary("Ready", "Install", func() []wizard.Fact {
				v := opts.Values()
				return []wizard.Fact{{Label: "Directory", Value: dir.Value()}, {Label: "Launcher entry", Value: v["menu"]}, {Label: "Link", Value: v["path"]}}
			}),
			wizard.Progress("Installing", stepNames, "Example is installed.", job(tick, fail)).WithBar(),
			wizard.Finish("Done", "Example 1.0 is installed.", widget.NewCheck(launchNow, nil)),
		},
	}
}

// confirmOptions are a pretend uninstaller: one question, a short job, a
// result.
func confirmOptions(tick time.Duration, fail bool) wizard.ConfirmOptions {
	return wizard.ConfirmOptions{
		AppID:    "io.ushineko.fynedesygn.example.installerwizard.uninstall",
		Name:     "Uninstall Example",
		Question: "Uninstall Example 1.0?",
		Detail: "It removes `~/.local/share/example` and puts back anything the install replaced.\n\n" +
			"Your settings in `~/.config/example` are left where they are.",
		Action: "Uninstall", Destructive: true,
		Done: "Example 1.0 was removed.",
		Job: func(ctx context.Context) error {
			for range 3 {
				select {
				case <-ctx.Done():
					return fmt.Errorf("removing: %w", ctx.Err())
				case <-time.After(tick):
				}
			}
			if fail {
				return errors.New("the launcher directory is read-only")
			}
			return nil
		},
	}
}

// leftovers are the files the pretend program made, which the pretend
// uninstall did not remove because the install did not create them.
var leftovers = []string{ //nolint:gochecknoglobals // a fixed list
	"~/.local/share/example/cache/thumbnails.db",
	"~/.local/share/example/cache/index.json",
	"~/.local/share/example/plugins/user-theme/",
}

// leftoversStep asks, after the uninstall, whether to remove the files the
// program made too, as fynstall's uninstaller does.
func leftoversStep(tick time.Duration) *wizard.ConfirmStep {
	return &wizard.ConfirmStep{
		Action: "Remove them too", Destructive: true, Decline: "Keep them",
		Ask: func() (string, string, bool) {
			detail := "The install did not create these, so they were left:\n\n"
			for _, l := range leftovers {
				detail += "- `" + l + "`\n"
			}
			return fmt.Sprintf("Example left %d files it made.", len(leftovers)), detail, true
		},
		Job: func(ctx context.Context) error {
			select {
			case <-ctx.Done():
				return fmt.Errorf("removing: %w", ctx.Err())
			case <-time.After(tick):
				return nil
			}
		},
		Done: "Example 1.0 was removed, with the files it made.",
	}
}

// pretendFiles is how many files the job pretends to copy, for the bar.
const pretendFiles = 3072

// job pretends to install: each step logs three parts, a tick apart, and
// the bar counts pretend files through all of them, once per file, as a
// real installer reports.
func job(tick time.Duration, fail bool) wizard.Job {
	return func(ctx context.Context, r *wizard.Reporter) error {
		parts := len(stepNames) * 3
		perPart := pretendFiles / parts
		done := 0
		for i, name := range stepNames {
			r.Advance(i, "working")
			for k := range 3 {
				select {
				case <-ctx.Done():
					r.Log(logpane.Warn, "stopped during "+name)
					return fmt.Errorf("%s: %w", name, ctx.Err())
				case <-time.After(tick):
				}
				for range perPart {
					done++
					r.Progress(float64(done)/pretendFiles,
						fmt.Sprintf("%d of %d files · %d KB of %d KB", done, pretendFiles, done*12, pretendFiles*12),
						fmt.Sprintf("lib/example/%s/module_%04d.py", strings.ToLower(strings.Fields(name)[0]), done))
				}
				r.Log(logpane.Info, fmt.Sprintf("%s: part %d of 3", name, k+1))
			}
			if fail && i == 2 {
				r.Log(logpane.Error, "the launcher directory is read-only")
				return errors.New("the launcher directory is read-only")
			}
			r.Finish(i, "ok")
		}
		return nil
	}
}
