package widgets

import (
	"fmt"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/require"
)

/*
The rows are exercised directly rather than through a laid-out list.

widget.List builds no rows at all until something lays it out, and a list as a
window's whole content is not laid out by the test driver -- a plain
widget.List in that position draws nothing either, so it is not this widget. The
row path is what matters here and it is reached the same way the list reaches
it: build a blank row, fill it with a row number.

That the ticks reach the screen is covered where a real caller draws them, in
terrariabonker's sell list.
*/
func pickList(names []string) (*PickList, []fyne.CanvasObject) {
	p := NewPickList(
		func() int { return len(names) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(i int, row fyne.CanvasObject) { row.(*widget.Label).SetText(names[i]) },
	)
	rows := make([]fyne.CanvasObject, len(names))
	for i := range names {
		rows[i] = p.blank()
		p.fill(i, rows[i])
	}
	return p, rows
}

// tickOf is a row's checkbox, and label its text.
func tickOf(row fyne.CanvasObject) *widget.Check {
	return row.(*fyne.Container).Objects[1].(*widget.Check)
}

func labelOf(row fyne.CanvasObject) *widget.Label {
	return row.(*fyne.Container).Objects[0].(*widget.Label)
}

/*
Several rows can be ticked, and the caller is told how many.

Fyne's list selects one row, which is right for a list being read and wrong for
one being edited: taking six things off a list one at a time, with a reload
between each, is six times the work for one decision.
*/
func TestSeveralRowsCanBePicked(t *testing.T) {
	p, rows := pickList([]string{"Terra Blade", "Wood", "Meowmere", "Zenith"})
	require.Equal(t, "Meowmere", labelOf(rows[2]).Text, "the caller still fills its own row")

	var told []int
	p.OnPicked = func(n int) { told = append(told, n) }
	require.Empty(t, p.Picked())

	tickOf(rows[2]).SetChecked(true)
	tickOf(rows[0]).SetChecked(true)
	require.Equal(t, []int{0, 2}, p.Picked(), "in row order, whichever order they were ticked")
	require.Equal(t, 2, p.PickedCount())
	require.Equal(t, []int{1, 2}, told,
		"the caller hears each change, so a button can say what it will do")

	tickOf(rows[0]).SetChecked(false)
	require.Equal(t, []int{2}, p.Picked())
	require.Equal(t, []int{1, 2, 1}, told)
}

/*
A row scrolled back into view does not tick what it was recycled from.

widget.Check.SetChecked calls OnChanged (docs/fyne-quirks.md, 2), so filling a
recycled row from a ticked item's state would tick the item now in it -- and
this list's whole purpose is to say which items an action is about to happen to.
*/
func TestARecycledRowDoesNotPickWhatItHeld(t *testing.T) {
	many := make([]string, 200)
	for i := range many {
		many[i] = fmt.Sprintf("item %d", i)
	}
	p, rows := pickList(many)

	tickOf(rows[0]).SetChecked(true)
	require.Equal(t, []int{0}, p.Picked())

	// The same row widget, recycled for another item, as scrolling does.
	row := rows[0]
	p.fill(150, row)
	require.Equal(t, []int{0}, p.Picked(), "recycling picked nothing")
	require.False(t, tickOf(row).Checked, "and the row it now holds is not ticked")

	// And back to the picked one, which must show as picked without doubling.
	p.fill(0, row)
	require.True(t, tickOf(row).Checked)
	require.Equal(t, []int{0}, p.Picked())
}

/*
A caller whose list changed clears the picks.

Row three of a shorter list is a different thing, so picks do not survive a
change: they are row numbers, not items, and the alternative is an action
happening to something nobody chose.
*/
func TestPicksAreClearedWhenTheListChanges(t *testing.T) {
	p, rows := pickList([]string{"one", "two", "three"})
	var told []int
	p.OnPicked = func(n int) { told = append(told, n) }

	tickOf(rows[1]).SetChecked(true)
	require.Equal(t, []int{1}, p.Picked())

	p.ClearPicks()
	require.Empty(t, p.Picked())
	require.Zero(t, p.PickedCount())
	require.Equal(t, []int{1, 0}, told, "the caller is told it is back to none")

	p.fill(1, rows[1])
	require.False(t, tickOf(rows[1]).Checked, "and a redrawn row is clear too")
}

// A row that is not the shape this list makes is left alone rather than
// panicking: a caller cannot supply one, but a future change to the row could.
func TestAnUnexpectedRowIsIgnored(t *testing.T) {
	p, _ := pickList([]string{"one"})
	require.NotPanics(t, func() { p.fill(0, widget.NewLabel("not a row")) })
	require.Empty(t, p.Picked())
}
