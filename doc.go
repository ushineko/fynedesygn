/*
Package fynedesygn is a design system and wrapper library for building desktop
user interfaces with Fyne (fyne.io/fyne/v2).

The root package holds only the vocabulary every subpackage shares. The
components live in subpackages: theme (colour schemes, fonts, appearance
preferences), widgets (the small shared primitives), table, logpane, markdown,
mermaid, dialogs, shell (the window skeleton and section lifecycle) and
fynetest (headless test helpers).

The rules the components follow are written down in docs/design-system.md.
*/
package fynedesygn

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
