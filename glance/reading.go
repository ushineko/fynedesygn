package glance

import (
	"image/color"

	fd "github.com/ushineko/fynedesygn"
)

// Reading is one value as it will be drawn: the formatted text, the verdict
// that colours it, and whether the source that produced it has since gone.
//
// Text is expected to come from one of this package's formatters, so it is
// already padded to a fixed width. A Reading built from fmt.Sprintf will draw,
// and will move the column the first time its magnitude changes.
//
// The five states a glance window distinguishes (docs/glance.md, Degradation)
// are reached through this type and through Card:
//
//   - not read yet, and read but absent: Text is the formatter's blank
//     (NoRate, NoPercent and so on), so the row holds its column.
//   - read, with a verdict: Text and Status.
//   - was read, source has gone: Stale, which dims the row and leaves the last
//     value legible rather than replacing it.
//   - source never present: not a Reading at all. The card is not drawn.
type Reading struct {
	// Text is the formatted value, padded to a fixed width.
	Text string
	// Status ranks the value. Info leaves it the theme's foreground colour,
	// which is what an ungraded measurement wants: a colour that is always on
	// is not a signal (docs/glance.md, Colour is the legend).
	Status fd.Status
	// Stale marks a last-known value whose source has stopped answering. It
	// dims the row; it does not blank it.
	Stale bool

	// Colour overrides Status for painting, for a value that has a colour but
	// no verdict. The case it exists for is a trace in a Sparkline: the plot
	// carries no legend, so each trace is drawn in the colour of its row's
	// value, and an ungraded measurement therefore needs a colour that is not
	// one of the four status colours. A graded value leaves this nil and takes
	// its colour from Status, which is how a colour stays a signal.
	//
	// Stale still wins: a value that is no longer being refreshed is dimmed
	// whatever colour it was.
	Colour color.Color
}

// Tinted returns a copy of r painted in an explicit colour. Used for a value
// whose row identifies a sparkline trace rather than ranking anything.
func (r Reading) Tinted(c color.Color) Reading {
	r.Colour = c
	return r
}

// Known is a value that was read, with a verdict.
func Known(text string, st fd.Status) Reading { return Reading{Text: text, Status: st} }

// Measured is a value that was read and is not being graded: a temperature
// with no threshold, a rate, a count. It is the common case and is deliberately
// easy to reach for, so that grading is the decision a caller has to make.
func Measured(text string) Reading { return Reading{Text: text, Status: fd.StatusInfo} }

// Missing is a value the source did not report. blank comes from the matching
// formatter (NoRate, NoPercent, NoQuantity), so the row keeps its width.
//
// Missing is not StatusBad. A reading that could not be taken is not a failure
// reading: the monitor this package is transcribed from states it as "absence
// of evidence is not a stopped pump", after a rendering that could not tell
// "0 rpm" from "no answer" would have masked a real one.
func Missing(blank string) Reading { return Reading{Text: blank, Status: fd.StatusInfo} }

// Dimmed returns a copy of r marked stale. Used when a source that was
// answering stops: the card keeps its rows and dims them, because the reader's
// question is "is this still true", and a dim number answers it without moving
// anything.
func (r Reading) Dimmed() Reading {
	r.Stale = true
	return r
}
