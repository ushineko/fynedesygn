package widgets

import (
	"sort"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

/*
PickList is a list whose rows can be ticked, for an action on several at once.

Fyne's list selects one row. That is right for a list you are reading and wrong
for one you are editing: taking six items off a list one at a time, with a
confirmation and a reload between each, is six times the work for one decision.

The tick is a checkbox in the row rather than a modifier held while clicking.
A checkbox says what it does by being there; ctrl-click is a convention the user
has to already know, it is invisible until they try it, and Fyne does not pass
the modifier to a list's selection callback anyway.

The caller builds and fills the rest of the row, exactly as with widget.List:

	picks := widgets.NewPickList(
	    func() int { return len(items) },
	    func() fyne.CanvasObject { return widget.NewLabel("") },
	    func(i int, row fyne.CanvasObject) { row.(*widget.Label).SetText(items[i]) },
	)
	picks.OnPicked = func(n int) { remove.SetText(fmt.Sprintf("Remove %d", n)) }

Picks are row numbers as the list stands. A caller that changes the list calls
ClearPicks, because row three of a shorter list is a different thing.
*/
type PickList struct {
	widget.BaseWidget

	// OnPicked is called with how many rows are ticked, whenever that changes,
	// so a button can say what it is about to do.
	OnPicked func(count int)

	list   *widget.List
	picked map[int]bool
	// create and update are the caller's, kept so a row can be built and
	// filled outside a layout pass -- which is how the rows are tested, since
	// a list draws none until something lays it out.
	create func() fyne.CanvasObject
	update func(id int, row fyne.CanvasObject)
}

/*
NewPickList makes a list of tickable rows.

length, create and update are widget.List's, and mean the same: how many rows
there are, what an empty one looks like, and how to fill one. The tick is added
in front of what create returns and is not the caller's to draw.
*/
func NewPickList(length func() int, create func() fyne.CanvasObject,
	update func(id int, row fyne.CanvasObject)) *PickList {
	p := &PickList{picked: map[int]bool{}, create: create, update: update}
	p.list = widget.NewList(length, p.blank, func(id widget.ListItemID, row fyne.CanvasObject) {
		p.fill(id, row)
	})
	p.ExtendBaseWidget(p)
	return p
}

// blank is one empty row: the caller's, with a tick in front of it.
func (p *PickList) blank() fyne.CanvasObject {
	return container.NewBorder(nil, nil, widget.NewCheck("", nil), nil, p.create())
}

// fill puts row number id into a row, ticking it if it is picked.
func (p *PickList) fill(id int, row fyne.CanvasObject) {
	box, ok := row.(*fyne.Container)
	if !ok || len(box.Objects) < 2 {
		return
	}
	// The caller's part is the Border's centre, which it keeps at index 0; the
	// tick is the piece added after it.
	tick, ok := box.Objects[1].(*widget.Check)
	if !ok {
		return
	}
	p.update(id, box.Objects[0])
	p.dress(tick, id)
}

/*
dress sets a recycled row's tick without the setting firing the handler.

widget.Check.SetChecked calls OnChanged, so a row scrolled into view would tick
whatever item it had been recycled from (docs/fyne-quirks.md, 2). The handler is
cleared first and put back after.
*/
func (p *PickList) dress(tick *widget.Check, id int) {
	tick.OnChanged = nil
	tick.SetChecked(p.picked[id])
	tick.OnChanged = func(on bool) {
		if on {
			p.picked[id] = true
		} else {
			delete(p.picked, id)
		}
		if p.OnPicked != nil {
			p.OnPicked(len(p.picked))
		}
	}
}

// CreateRenderer implements fyne.Widget.
func (p *PickList) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(p.list)
}

// Picked is the ticked rows, lowest first, as row numbers of the list as it
// stands.
func (p *PickList) Picked() []int {
	out := make([]int, 0, len(p.picked))
	for id := range p.picked {
		out = append(out, id)
	}
	sort.Ints(out)
	return out
}

// PickedCount is how many rows are ticked.
func (p *PickList) PickedCount() int { return len(p.picked) }

// ClearPicks unticks everything. Called by a caller whose list has changed,
// because a row number means something else once the list is different.
func (p *PickList) ClearPicks() {
	p.picked = map[int]bool{}
	if p.OnPicked != nil {
		p.OnPicked(0)
	}
	p.list.Refresh()
}

// Refresh redraws the rows, for a list whose contents changed under it.
func (p *PickList) Refresh() {
	p.list.Refresh()
	p.BaseWidget.Refresh()
}

// ScrollToTop puts the list back at its first row.
func (p *PickList) ScrollToTop() { p.list.ScrollToTop() }
