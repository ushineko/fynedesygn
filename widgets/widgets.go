/*
Package widgets holds the small shared primitives every section is assembled
from: dimmed and wrapped labels, status-coloured text and markers, headings,
cards, fact rows, action blocks, and the fixed-size wrappers that keep a
section's layout stable while it is being looked at.

Reach for one of these before writing a new arrangement of labels, so that
every section states the same kind of thing the same way. Colour always comes
from the active theme through a Status, never from a literal, so the widgets
stay legible in every colour scheme.

Ported from the shared widgets of angou and nmsbonker (same author, MIT).
*/
package widgets

import (
	"fmt"
	"image/color"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	fd "github.com/ushineko/fynedesygn"
)

// LabelWidth is the width of the label column in fact rows, so the values in
// a card line up regardless of how long each label is.
const LabelWidth float32 = 190

// MarkerGutter is the width PlainRow reserves in place of a status marker, so
// unranked values line up with the ranked rows above and below them.
const MarkerGutter float32 = 26

// Dim is a low-importance label, for text that supports a value rather than
// being one.
func Dim(s string) fyne.CanvasObject {
	l := widget.NewLabel(s)
	l.Importance = widget.LowImportance
	return l
}

// Sep is the dot that separates the facts on a status line.
func Sep() fyne.CanvasObject { return widget.NewLabel("·") }

// Wrapped is a paragraph that reflows rather than running off the edge. Used
// for the sentences in dialogs, which are the ones that state consequences.
func Wrapped(text string) fyne.CanvasObject {
	l := widget.NewLabel(text)
	l.Wrapping = fyne.TextWrapWord
	return l
}

// StatusText colours a value by its status. The colour is drawn from the
// active theme's success, warning and error roles, so it stays legible in
// every scheme. An info status leaves the label at its default colour.
func StatusText(s string, st fd.Status) fyne.CanvasObject {
	l := widget.NewLabel(s)
	switch st {
	case fd.StatusGood:
		l.Importance = widget.SuccessImportance
	case fd.StatusWarn:
		l.Importance = widget.WarningImportance
	case fd.StatusBad:
		l.Importance = widget.DangerImportance
	case fd.StatusInfo:
	}
	return l
}

// Marker is the icon that ranks a row at a glance.
func Marker(st fd.Status) fyne.CanvasObject {
	switch st {
	case fd.StatusGood:
		return widget.NewIcon(theme.ConfirmIcon())
	case fd.StatusWarn:
		return widget.NewIcon(theme.WarningIcon())
	case fd.StatusBad:
		return widget.NewIcon(theme.ErrorIcon())
	case fd.StatusInfo:
	}
	return widget.NewIcon(theme.InfoIcon())
}

// Heading is a section title with a dimmed, wrapping blurb and a rule under
// both. Every section opens with one.
func Heading(title, blurb string) fyne.CanvasObject {
	h := widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	b := widget.NewLabel(blurb)
	b.Wrapping = fyne.TextWrapWord
	b.Importance = widget.LowImportance
	return container.NewVBox(h, b, widget.NewSeparator())
}

// Card is a titled block of facts with a rule under the title. Overview
// sections are made of these.
func Card(title string, body ...fyne.CanvasObject) fyne.CanvasObject {
	head := widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	return container.NewVBox(append([]fyne.CanvasObject{head, widget.NewSeparator()}, body...)...)
}

// Note is a wrapped paragraph under a row, for the things a value cannot say
// on its own. It is indented past the label column so it reads as belonging
// to the value above it. Warn and bad notes take the matching colour; good
// and info notes are dimmed, because a note that merely elaborates should not
// compete with the value it elaborates.
func Note(text string, st fd.Status) fyne.CanvasObject {
	l := widget.NewLabel(text)
	l.Wrapping = fyne.TextWrapWord
	switch st {
	case fd.StatusWarn:
		l.Importance = widget.WarningImportance
	case fd.StatusBad:
		l.Importance = widget.DangerImportance
	case fd.StatusGood, fd.StatusInfo:
		l.Importance = widget.LowImportance
	}
	return container.NewBorder(nil, nil, FixedWidth(widget.NewLabel(""), LabelWidth), nil, l)
}

// FactRow is one "label   value" line with a status marker, the shape a
// terminal would print as `label: value`.
func FactRow(label, value string, st fd.Status) fyne.CanvasObject {
	l := widget.NewLabel(label)
	l.Importance = widget.LowImportance
	return container.NewBorder(nil, nil, FixedWidth(l, LabelWidth), nil,
		container.NewHBox(Marker(st), StatusText(value, st)))
}

// PlainRow is FactRow without the marker, for facts that carry no verdict. The
// marker column is still reserved so the values line up with the ranked rows
// above and below them.
func PlainRow(label, value string) fyne.CanvasObject {
	l := widget.NewLabel(label)
	l.Importance = widget.LowImportance
	v := widget.NewLabel(value)
	v.Truncation = fyne.TextTruncateEllipsis
	return container.NewBorder(nil, nil, FixedWidth(l, LabelWidth), nil,
		container.NewBorder(nil, nil, FixedWidth(widget.NewLabel(""), MarkerGutter), nil, v))
}

// RowWithAction puts one button on a fact row's trailing edge, for a fact that
// names a place the desktop can open. The button is always drawn and disables
// with the fact behind it rather than appearing and disappearing, so the card
// keeps its shape when the state changes.
func RowWithAction(row fyne.CanvasObject, btn *widget.Button) fyne.CanvasObject {
	return container.NewBorder(nil, nil, nil,
		container.NewPadded(container.NewVBox(btn)), row)
}

// Action renders one operation as a titled block with its consequences
// stated, rather than as a bare button: a window has room to say what will
// happen. The button is returned as well so the caller can enable and disable
// it. danger paints the button in the theme's error colour.
func Action(title, blurb, button string, danger bool, tapped func()) (fyne.CanvasObject, *widget.Button) {
	t := widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	b := widget.NewLabel(blurb)
	b.Wrapping = fyne.TextWrapWord
	b.Importance = widget.LowImportance
	btn := widget.NewButton(button, tapped)
	if danger {
		btn.Importance = widget.DangerImportance
	}
	// Padded so the button does not sit flush against the window edge; the
	// border layout gives the trailing object exactly its minimum width.
	row := container.NewBorder(nil, nil, nil, container.NewPadded(container.NewVBox(btn)),
		container.NewVBox(t, b))
	return row, btn
}

// AboutNote is a bold title over a dimmed, wrapping paragraph, the shape an
// About section uses to state one thing the window promises or does not.
func AboutNote(title, detail string) fyne.CanvasObject {
	t := widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	d := widget.NewLabel(detail)
	d.Wrapping = fyne.TextWrapWord
	d.Importance = widget.LowImportance
	return container.NewVBox(t, d)
}

// FixedHeight pins a widget's minimum height. Fyne's list and table take all
// the space they are given; this keeps a section's layout stable while it is
// being looked at, which is what stops a growing log from stretching the
// window a line at a time.
func FixedHeight(o fyne.CanvasObject, h float32) fyne.CanvasObject {
	pad := canvas.NewRectangle(nil)
	pad.SetMinSize(fyne.NewSize(0, h))
	return container.New(layout.NewStackLayout(), pad, o)
}

// FixedWidth pins a widget's minimum width. An HBox hands a truncating label
// its minimum size, which for a truncating label is nothing; this gives the
// label a width to truncate within.
func FixedWidth(o fyne.CanvasObject, w float32) fyne.CanvasObject {
	pad := canvas.NewRectangle(nil)
	pad.SetMinSize(fyne.NewSize(w, 0))
	return container.New(layout.NewStackLayout(), pad, o)
}

// HumanSize formats a byte count in binary units (GiB, MiB, KiB, B) with one
// decimal, the way a file manager would.
func HumanSize(n int64) string {
	switch {
	case n >= 1<<30:
		return fmt.Sprintf("%.1f GiB", float64(n)/(1<<30))
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MiB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.1f KiB", float64(n)/(1<<10))
	}
	return fmt.Sprintf("%d B", n)
}

// HumanAgo says how long ago t was, coarsely: "just now", "5m ago", "3h ago",
// "3d ago". The zero time reads "never".
func HumanAgo(t time.Time) string { return HumanAgoAt(t, time.Now()) }

// HumanAgoAt is HumanAgo with the clock injected, so the output can be
// asserted without waiting.
func HumanAgoAt(t, now time.Time) string {
	if t.IsZero() {
		return "never"
	}
	d := now.Sub(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 48*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	}
	return fmt.Sprintf("%dd ago", int(d.Hours()/24))
}

// OrNone substitutes a stand-in for an empty value, so a blank cell never
// reads as a rendering fault.
func OrNone(s, fallback string) string {
	if strings.TrimSpace(s) == "" {
		return fallback
	}
	return s
}

// ImportanceFor maps a status onto the label importance that paints it from
// the active theme's roles. Info maps to MediumImportance, the default colour.
func ImportanceFor(st fd.Status) widget.Importance {
	switch st {
	case fd.StatusGood:
		return widget.SuccessImportance
	case fd.StatusWarn:
		return widget.WarningImportance
	case fd.StatusBad:
		return widget.DangerImportance
	case fd.StatusInfo:
	}
	return widget.MediumImportance
}

// StatusColor is the colour the active theme assigns to a status: its
// success, warning and error roles for good, warn and bad, and its primary
// colour for info. Components that paint a status themselves (a banner
// background, a marker tint) use this rather than a literal so they follow
// the scheme.
func StatusColor(st fd.Status) color.Color {
	switch st {
	case fd.StatusGood:
		return theme.Color(theme.ColorNameSuccess)
	case fd.StatusWarn:
		return theme.Color(theme.ColorNameWarning)
	case fd.StatusBad:
		return theme.Color(theme.ColorNameError)
	case fd.StatusInfo:
	}
	return theme.Color(theme.ColorNamePrimary)
}
