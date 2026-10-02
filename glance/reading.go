package glance

import (
	"image/color"
	"strings"

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

	// Parts draws the value as pieces side by side, each in its own colour,
	// for a value that is two readings at once: a bandwidth row's down and up
	// rates, each graded by its own size. Painting the whole value by the
	// stronger one coloured a 2 KiB/s download orange because the upload was
	// 40 MiB/s (hayami spec 031).
	//
	// Text is still the whole value -- the parts' concatenation -- and is what
	// the row is measured by, so a parted value is exactly as wide as the
	// same text unparted. Parted builds a reading that way. Status and Colour
	// still describe the row as a whole (a sparkline trace reads them); each
	// part is painted from its own. Stale dims every part.
	Parts []Part
}

// Part is one piece of a parted value. It is painted like a Reading: Colour
// when set, the foreground for Info, the status colour otherwise.
type Part struct {
	// Text is this piece of the value, padded like any other: the parts are
	// laid end to end and their total width is the row's.
	Text string
	// Status ranks this piece.
	Status fd.Status
	// Colour overrides Status for painting, as Reading.Colour does.
	Colour color.Color
	// Bold draws this piece bold, where the theme has a bold monospace face
	// (quirk 35); where it has none the piece is regular.
	Bold bool
}

// Parted is a value drawn in pieces. Its Text is the pieces' concatenation,
// which is what the row measures, and its Status is Info: the grading is in
// the parts.
func Parted(parts ...Part) Reading {
	return Reading{Text: partsText(parts), Status: fd.StatusInfo, Parts: parts}
}

// partsText is the whole value a set of parts spells.
func partsText(parts []Part) string {
	var b strings.Builder
	for _, p := range parts {
		b.WriteString(p.Text)
	}
	return b.String()
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
