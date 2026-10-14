package glance

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
)

/*
Rule is a horizontal line across a card or a drawer, between groups of
readings that are of different things: one account's quotas and the next's
(spec 069).

It is the colour of a card's border and one unit thick, in the middle of half
a line of room, so the groups stand apart by more than the gap between two
rows. Use it between groups and not between rows: a rule under every reading
is a table, and a glance surface is not one.
*/
type Rule struct {
	themed

	line *canvas.Rectangle
	box  *fyne.Container
}

// NewRule builds a rule.
func NewRule() *Rule {
	r := &Rule{line: canvas.NewRectangle(nil)}
	r.box = container.New(&ruleLayout{rule: r}, r.line)
	r.Restyle()
	return r
}

// Object is the rule, for a card's or a drawer's content.
func (r *Rule) Object() fyne.CanvasObject { return r.box }

// SetTheme gives the rule a face of its own and repaints in it.
func (r *Rule) SetTheme(th fyne.Theme) {
	r.themed.SetTheme(th)
	r.Restyle()
}

// Restyle repaints the rule in the current theme.
func (r *Rule) Restyle() {
	r.line.FillColor = r.colour(theme.ColorNameSeparator)
	r.line.Refresh()
	r.box.Refresh()
}

// ruleLayout draws the line across the middle of the rule's room.
type ruleLayout struct{ rule *Rule }

func (l *ruleLayout) MinSize([]fyne.CanvasObject) fyne.Size {
	return fyne.NewSize(0, l.rule.textSize()/2)
}

func (l *ruleLayout) Layout(_ []fyne.CanvasObject, size fyne.Size) {
	l.rule.line.Move(fyne.NewPos(0, (size.Height-1)/2))
	l.rule.line.Resize(fyne.NewSize(size.Width, 1))
}
