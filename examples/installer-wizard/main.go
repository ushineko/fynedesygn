/*
Command installer-wizard is a fynedesygn example: a wizard that goes through
every standard page (welcome, licence, directory, options, summary,
progress and finish) to install nothing. The job is fake and slow enough to
cancel; -fail makes it fail at its third step, and -scheme chooses the colour
scheme, for tools/screenshot.sh.

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
	flag.Parse()
	o := options(400*time.Millisecond, *fail)
	if *scheme != "" {
		ap := fdtheme.DefaultAppearance()
		ap.Scheme = *scheme
		o.Appearance = &ap
	}
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
			wizard.Progress("Installing", stepNames, "Example is installed.", job(tick, fail)),
			wizard.Finish("Done", "Example 1.0 is installed.", widget.NewCheck(launchNow, nil)),
		},
	}
}

// job pretends to install: each step logs three parts, a tick apart.
func job(tick time.Duration, fail bool) wizard.Job {
	return func(ctx context.Context, r *wizard.Reporter) error {
		for i, name := range stepNames {
			r.Advance(i, "working")
			for k := range 3 {
				select {
				case <-ctx.Done():
					r.Log(logpane.Warn, "stopped during "+name)
					return fmt.Errorf("%s: %w", name, ctx.Err())
				case <-time.After(tick):
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
