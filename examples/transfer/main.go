/*
Command transfer is a fynedesygn example: a job that knows how far along it is,
run through PerformCancellable, reporting its real progress to the busy popup
with BusyProgress. The bar fills with the bytes copied, the caption says how many,
and Cancel stops it.

A job that cannot name a real number does not call BusyProgress, and its popup
keeps the bar with no end; see examples/job-runner for a job with steps instead.
*/
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	fynetheme "fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	fd "github.com/ushineko/fynedesygn"
	"github.com/ushineko/fynedesygn/shell"
	"github.com/ushineko/fynedesygn/widgets"
)

// total is the size of the pretend transfer, in bytes.
const total = 64 << 20

func main() {
	o := options(40 * time.Millisecond)
	if os.Getenv("TRANSFER_AUTOSTART") != "" {
		// For tools/screenshot.sh: a slow copy that starts by itself, so the
		// busy popup is on screen long enough to be captured.
		o = options(150 * time.Millisecond)
		o.Scheme = os.Getenv("TRANSFER_SCHEME") // one run only, never saved
		o.OnStart = func(s *shell.Shell) {
			go func() {
				time.Sleep(time.Second)
				copyWithPopup(s, 150*time.Millisecond)
			}()
		}
	}
	shell.Run(o)
}

// copyWithPopup runs the copy behind a cancellable busy popup.
func copyWithPopup(s *shell.Shell, tick time.Duration) {
	s.PerformCancellable("Copying the file...", func(ctx context.Context) error {
		return copyBytes(ctx, s, tick)
	})
}

// copyBytes pretends to copy total bytes, a megabyte per tick, reporting real
// progress after each. A real program calls BusyProgress from its read loop.
func copyBytes(ctx context.Context, s *shell.Shell, tick time.Duration) error {
	for done := 0; done < total; done += 1 << 20 {
		// Checked first: with a zero tick both cases of the select below are
		// ready and Go picks one at random, so a cancelled copy could still
		// take a few steps.
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("transfer stopped at %d of %d MiB: %w", done>>20, total>>20, err)
		}
		select {
		case <-ctx.Done():
		case <-time.After(tick):
		}
		got := done + 1<<20
		s.BusyProgress(float64(got)/total, fmt.Sprintf("Copying %d of %d MiB", got>>20, total>>20))
	}
	fyne.Do(func() { s.Flash("The transfer finished.", fd.StatusGood) })
	return nil
}

func options(tick time.Duration) shell.Options {
	return shell.Options{
		AppID: "io.ushineko.fynedesygn.example.transfer",
		Name:  "transfer",
		Sections: []shell.Section{
			shell.NewSection("Transfer", fynetheme.DownloadIcon, func(s *shell.Shell) fyne.CanvasObject {
				start := widget.NewButtonWithIcon("Copy 64 MiB", fynetheme.DownloadIcon(), func() {
					copyWithPopup(s, tick)
				})
				start.Importance = widget.HighImportance
				s.Gate(start)
				return container.NewVBox(
					widgets.Heading("Transfer", "A job that knows how far along it is. The popup's bar fills with the bytes copied, and Cancel stops the copy."),
					container.NewHBox(start),
				)
			}),
		},
	}
}
