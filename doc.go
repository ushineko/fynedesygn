/*
Package fynedesygn is a design system and wrapper library for building desktop
user interfaces with Fyne (fyne.io/fyne/v2).

The root package holds only the vocabulary every subpackage shares. The
components live in subpackages: theme (colour schemes, fonts, appearance
preferences), widgets (the small shared primitives), table, logpane, markdown,
mermaid, dialogs, shell (the window skeleton and section lifecycle), glance
(always-on-top status panels) and fynetest (headless test helpers).

There are two window archetypes and two rulebooks. shell builds a window
someone works in, and its rules are in docs/design-system.md. glance builds a
small frameless panel read without being interacted with, and its rules are in
docs/glance.md.
*/
package fynedesygn

import _ "embed"

//go:embed README.md
var readme []byte

// README is this module's README, embedded so a program (the gallery) can
// show it in a window without carrying a second copy of the text.
func README() string { return string(readme) }

// Status ranks a fact for presentation. It is the one presentation type the
// whole library shares: every component that paints a verdict, a banner or a
// table row takes a Status and asks the active colour scheme for the colour.
//
// Status says how to paint what an application's core returns, which is a
// different question from what the core returns. Applications map their own
// domain levels onto it and keep that mapping on their side of the boundary.
type Status int

const (
	// StatusInfo is neutral: a fact with no verdict attached.
	StatusInfo Status = iota
	// StatusGood is a positive verdict.
	StatusGood
	// StatusWarn is a verdict that deserves attention but not alarm.
	StatusWarn
	// StatusBad is a failure. Banners with this status never dismiss themselves.
	StatusBad
)

// String returns the lower-case name of the status.
func (s Status) String() string {
	switch s {
	case StatusGood:
		return "good"
	case StatusWarn:
		return "warn"
	case StatusBad:
		return "bad"
	default:
		return "info"
	}
}
