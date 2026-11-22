package glance

import (
	"fmt"
	"math"
	"slices"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
)

/*
Scroller is a run of rows in a card that is never taller than a number of
them (spec 086). Past that number the wheel scrolls it, a row at a time.

For a list whose length follows what is read and whose order does not: the
VPNs that are up, under a card's interfaces. A card that grew a line for each
of them would push every card under it down each time one came up; held to a
height, the card keeps its size and the rest are a turn of the wheel away.

Add it to a card with Card.Add, between what is above it and what is below:
a sparkline added after it stays under it and does not scroll.

  - **Its height is that of the rows it shows**: Visible rows when it holds
    more, all of them when it holds fewer. A row added or taken away past
    Visible changes nothing outside it.
  - **The wheel moves it by whole rows**, while the pointer is over it.
  - **The bar is in the card's right margin** and takes no width, as the
    drawer's is (spec 069); it is not dragged, since a press on a panel is
    the catcher's.
  - **A row set in place does not move it**, and a row added or taken away
    leaves the first row shown where it was while that row is still there.
  - **A click lands on the row under the pointer**, after scrolling; a row
    scrolled out of sight is never the one clicked.

Do not use it for a list sorted by a reading. Its order moves under the eye;
cut it to the height instead and give it a filter, as a drawer does.
*/
type Scroller struct {
	themed

	rows    []*Row
	visible int

	list   *fyne.Container
	pad    *fyne.Container
	scroll *container.Scroll
	box    *fyne.Container

	// first is the index, among the shown rows, of the row at the top, and
	// top that row itself, kept so a row added or taken away above it does
	// not change which row is at the top.
	first int
	top   *Row

	// acc is the wheel travel not yet turned into a row, and accAt when it
	// was last added to: a touchpad sends travel in slivers, a wheel in
	// notches, and a row should take about a notch either way.
	acc      float32
	accAt    time.Time
	snapping bool

	// stale is whether the card is showing last-known values, so a row
	// added while it is comes in dimmed.
	stale bool
}

// accReset is how long a pause in the wheel forgets the travel that had not
// yet made a row.
const accReset = 300 * time.Millisecond

/*
wheelRun is how many rows' travel one wheel event must carry before it moves
more than one row.

A notch is a row. Fyne turns a notch into its ScrollSpeed (25 on Linux and
Windows, 10 on macOS) times the wheel's own count, which on a scaled desktop
is more than one -- 37.5 at 1.5 -- so a notch carries one to two rows' travel
depending on the desk and the text size. Rounding that moved two rows a notch
on one desk and one on another. Only a spin Fyne has accelerated (125 times a
notch) carries more than four rows, and that moves as many as it carries over
four.
*/
const wheelRun = 4

// NewScroller builds an empty run that shows at most visible rows. A visible
// below one is one.
func NewScroller(visible int) *Scroller {
	s := &Scroller{visible: max(1, visible)}
	s.list = container.NewVBox()
	s.pad = container.New(s.rightPad(), s.list)
	s.scroll = container.NewVScroll(s.pad)
	s.scroll.OnScrolled = s.scrolled
	s.box = container.New(&scrollerLayout{s: s}, s.scroll)
	return s
}

// rightPad keeps the rows off the right margin the scroll reaches into, so
// the bar is drawn in the margin and not over a figure.
func (s *Scroller) rightPad() fyne.Layout {
	return layout.NewCustomPaddedLayout(0, 0, 0, s.padH())
}

// SetVisible changes how many rows show at most. A number below one is one.
func (s *Scroller) SetVisible(n int) {
	s.visible = max(1, n)
	s.changed()
}

// Visible is how many rows show at most.
func (s *Scroller) Visible() int { return s.visible }

// AddRow appends rows, in the order they will be drawn. It panics on a row
// whose ID the run already holds, as Card.AddRow does.
func (s *Scroller) AddRow(rows ...*Row) {
	for _, r := range rows {
		if err := s.unique(r); err != nil {
			panic(err)
		}
		r.SetTheme(s.th)
		s.dim(r)
		s.rows = append(s.rows, r)
		s.list.Add(r.Object())
	}
	s.changed()
}

// InsertRow puts a row before the one at index at; at equal to the number of
// rows appends it.
func (s *Scroller) InsertRow(at int, r *Row) error {
	if at < 0 || at > len(s.rows) {
		return fmt.Errorf("inserting at %d of %d rows: %w", at, len(s.rows), ErrRowIndex)
	}
	if err := s.unique(r); err != nil {
		return err
	}
	r.SetTheme(s.th)
	s.dim(r)
	s.rows = slices.Insert(s.rows, at, r)
	s.list.Objects = slices.Insert(s.list.Objects, at, r.Object())
	s.changed()
	return nil
}

// RemoveRow takes out the row with the ID, and reports whether there was one.
func (s *Scroller) RemoveRow(id string) bool {
	if id == "" {
		return false
	}
	for i, r := range s.rows {
		if r.ID() != id {
			continue
		}
		s.rows = slices.Delete(s.rows, i, i+1)
		s.list.Objects = slices.Delete(s.list.Objects, i, i+1)
		if s.top == r {
			s.top = nil
		}
		s.changed()
		return true
	}
	return false
}

// RowByID is the row with the ID, or nil.
func (s *Scroller) RowByID(id string) *Row {
	if id == "" {
		return nil
	}
	for _, r := range s.rows {
		if r.ID() == id {
			return r
		}
	}
	return nil
}

// Rows are the run's rows, in order, shown or not.
func (s *Scroller) Rows() []*Row { return s.rows }

func (s *Scroller) unique(r *Row) error {
	if r.ID() != "" && s.RowByID(r.ID()) != nil {
		return fmt.Errorf("row %q: %w", r.ID(), ErrRowID)
	}
	return nil
}

// First is the index, among the rows shown, of the row at the top.
func (s *Scroller) First() int { return s.first }

// ScrollTo puts the shown row at index i at the top, within what there is to
// scroll.
func (s *Scroller) ScrollTo(i int) {
	s.place(i)
}

// Object is the run, for Card.Add.
func (s *Scroller) Object() fyne.CanvasObject { return s.box }

// SetTheme gives the run and its rows a face of their own and repaints them.
func (s *Scroller) SetTheme(th fyne.Theme) {
	s.themed.SetTheme(th)
	for _, r := range s.rows {
		r.SetTheme(th)
	}
	s.Restyle()
}

// Restyle repaints the rows and measures the run again, after a change of
// face or size.
func (s *Scroller) Restyle() {
	for _, r := range s.rows {
		r.Restyle()
	}
	s.pad.Layout = s.rightPad()
	s.changed()
}

// setLines keeps the run in a card drawn as one line per reading (Lines):
// its rows are lines like any other, and it still shows at most Visible.
func (s *Scroller) setLines(bool) {}

// shown are the rows drawn, in order.
func (s *Scroller) shown() []*Row {
	out := make([]*Row, 0, len(s.rows))
	for _, r := range s.rows {
		if r.Shown() {
			out = append(out, r)
		}
	}
	return out
}

// pitch is the distance from one row's top to the next's.
func (s *Scroller) pitch() float32 {
	for _, r := range s.rows {
		if r.Shown() {
			return r.Object().MinSize().Height + theme.Padding()
		}
	}
	return 0
}

// height is the room the run takes: the rows it shows and the gaps between.
func (s *Scroller) height() float32 {
	n := min(s.visible, len(s.shown()))
	if n == 0 {
		return 0
	}
	return float32(n)*s.pitch() - theme.Padding()
}

// most is the highest index the top row can have.
func (s *Scroller) most() int { return max(0, len(s.shown())-s.visible) }

// changed lays the run out again after its rows or its size changed, and
// keeps the row that was at the top there.
func (s *Scroller) changed() {
	s.list.Refresh()
	s.box.Refresh()
	at := 0
	if s.top != nil {
		if i := slices.Index(s.shown(), s.top); i >= 0 {
			at = i
		} else {
			at = s.first
		}
	} else {
		at = s.first
	}
	s.place(at)
}

// place puts the shown row at index i at the top.
func (s *Scroller) place(i int) {
	i = max(0, min(i, s.most()))
	s.first = i
	if shown := s.shown(); i < len(shown) {
		s.top = shown[i]
	} else {
		s.top = nil
	}
	s.snapping = true
	s.scroll.ScrollToOffset(fyne.NewPos(0, float32(i)*s.pitch()))
	s.snapping = false
}

// scrolled turns the scroll's move into whole rows. The scroll has moved by
// the wheel's travel when this is called; it is put back to a row's top
// before it is drawn.
func (s *Scroller) scrolled(p fyne.Position) {
	if s.snapping {
		return
	}
	pitch := s.pitch()
	if pitch <= 0 {
		return
	}
	now := time.Now()
	if now.Sub(s.accAt) > accReset {
		s.acc = 0
	}
	s.accAt = now
	s.acc += p.Y - float32(s.first)*pitch
	step := 0
	if a := float32(math.Abs(float64(s.acc))); a >= pitch/4 {
		step = max(1, int(a/(wheelRun*pitch)))
		if s.acc < 0 {
			step = -step
		}
		s.acc = 0
	}
	s.place(s.first + step)
}

// inside reports whether a canvas position is in the room the run shows.
func (s *Scroller) inside(at fyne.Position) bool {
	if !s.box.Visible() {
		return false
	}
	pos := fyne.CurrentApp().Driver().AbsolutePositionForObject(s.box)
	size := s.box.Size()
	return at.X >= pos.X && at.X < pos.X+size.Width && at.Y >= pos.Y && at.Y < pos.Y+size.Height
}

// rowAt is the shown row under a canvas position, or nil. A row scrolled
// out of the room is never under it.
func (s *Scroller) rowAt(at fyne.Position) *Row {
	if !s.inside(at) {
		return nil
	}
	for _, r := range s.rows {
		if rowAt(r, at) {
			return r
		}
	}
	return nil
}

// scrollerLayout gives the run its height, and gives the scroll the card's
// right margin as well as the run's width, so the bar is drawn in the margin.
type scrollerLayout struct{ s *Scroller }

func (l *scrollerLayout) MinSize([]fyne.CanvasObject) fyne.Size {
	return fyne.NewSize(l.s.list.MinSize().Width, l.s.height())
}

func (l *scrollerLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	for _, o := range objects {
		o.Move(fyne.NewPos(0, 0))
		o.Resize(fyne.NewSize(size.Width+l.s.padH(), size.Height))
	}
}

// setStale dims the run's rows, or brightens them, with their card.
func (s *Scroller) setStale(dim bool) {
	s.stale = dim
	for _, r := range s.rows {
		s.dim(r)
	}
}

// dim gives a row the run's staleness.
func (s *Scroller) dim(r *Row) {
	if rd := r.Reading(); rd.Stale != s.stale {
		rd.Stale = s.stale
		r.Set(rd)
	}
}
