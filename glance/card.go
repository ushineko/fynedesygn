package glance

import (
	"errors"
	"fmt"
	"image/color"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"github.com/ushineko/fynedesygn/widgets"
)

// CardRadius is the corner radius of a card, in pixels.
//
// Deliberately not the scheme's own radius token. That token is a *control*
// radius — an entry, a button — and the schemes set it to 2 (Breeze, Oxygen,
// Adwaita), 4 (Fluent) or 8 (macOS). At 2 px a card the size of a glance
// window's reads as a square, which is not what any of those desktops draws
// for a floating panel: the monitor this archetype comes from uses 8 for a
// section and 12 for the window around them.
//
// A card is a surface, not a control, so it takes a surface's radius. It is
// one number rather than a per-scheme token because the schemes do not
// disagree about panels the way they disagree about buttons.
const CardRadius float32 = 8

// CardPadH is a card's inner margin at the sides. With CardPadTop,
// CardPadBottom and CardGap it sets the card's geometry, and each of the four
// is a factor of the text size rather than a pixel count. A margin set for one
// size is proportionally huge at a smaller one: the monitor multiplies every
// margin by its font scale for that reason, and this is the same arithmetic
// with the scale already applied by the theme.
//
// The factors come from the monitor's own numbers at its 11 px face: 15 px at
// the sides, 8 above the title, 10 under the last row, and 11 between one
// section and the next. A card padded with the scheme's padding token instead
// (3 px in Breeze) sits its rows against its own border, which is the single
// largest difference between a glance window drawn by this package and the
// program the archetype comes from.
func CardPadH() float32 { return theme.TextSize() * 1.25 }

// CardPadTop is the margin above a card's title. See CardPadH.
func CardPadTop() float32 { return theme.TextSize() * 0.67 }

// CardPadBottom is the margin under a card's last row. It is larger than the
// top margin because the title's own ascent already reads as space. See
// CardPadH.
func CardPadBottom() float32 { return theme.TextSize() * 0.83 }

// CardGap is the space between two cards in the stack. See CardPadH.
func CardGap() float32 { return theme.TextSize() * 0.9 }

// padH, padTop, padBottom and gap are the four above measured in whatever face
// the object has been given, which is what every layout here uses. The
// exported ones stay for a consumer doing arithmetic of its own.
func (t *themed) padH() float32      { return t.textSize() * 1.25 }
func (t *themed) padTop() float32    { return t.textSize() * 0.67 }
func (t *themed) padBottom() float32 { return t.textSize() * 0.83 }
func (t *themed) gap() float32       { return t.textSize() * 0.9 }

// IconScale is a card icon's edge, in text sizes.
//
// A little over the cap height, so the glyph reads as the title's equal
// rather than as a bullet before it. A factor rather than a pixel count, for
// the reason CardPadH gives: a size set for one face is wrong at another, and
// a glance panel's text size is a setting.
const IconScale float32 = 1.15

// GoneMarker is what a card's header says when its source was answering and
// has stopped. It is short because it shares the header with the title, and it
// is a state rather than a mechanism: the reason belongs in the log.
const GoneMarker = "(unavailable)"

// Card is one metric's panel: a title, some rows, and optionally something
// drawn under them such as a Sparkline.
//
// A card is drawn only when the user allows it and its source has something to
// say. The two are separate on purpose — a card the user hid and a card with
// no data look the same and arrive by different paths, and a program that
// conflated them would bring a hidden card back the moment its sensor
// appeared.
//
// A card that is not drawn costs nothing. Stop its timer in the callback
// handed to OnDrawnChanged: a glance window on a machine without the sensor it
// watches should be indistinguishable from one built without the card.
type Card struct {
	themed

	title  *canvas.Text
	icon   *canvas.Image
	mark   *canvas.Text
	header *fyne.Container

	// end and endIcon are the heading's other end (SetTitleEnd), drawn at
	// the right before the gone marker. Hidden until given something.
	end     *canvas.Text
	endIcon *canvas.Image

	// clear is the room after the heading's other end that keeps it out from
	// under the panel's tuck glyph (spec 088). Shown with the end.
	clear *canvas.Rectangle

	// titleSize is the theme size the heading is drawn at (SetTitleSize):
	// the ordinary text size unless the consumer asks for another.
	titleSize fyne.ThemeSizeName

	// titleWidth and endWidth hold the two titles at the widest they have
	// been, so a title that changes -- a clock's time -- never narrows the
	// card and the panel with it.
	titleWidth, endWidth *widest
	rows                 []*Row
	body                 *fyne.Container

	// objects are what AddObject was given, kept so a change of theme can
	// reach them: the body holds them as plain canvas objects and a
	// container cannot be asked which of its children have a theme.
	objects []fyne.CanvasObject

	// pieces are what Add was given: the things in the card that have a face
	// of their own and can be told when it changes.
	pieces []Piece
	inner  *fyne.Container
	face   *canvas.Rectangle
	frame  *fyne.Container

	allowed   bool
	available bool
	stale     bool
	drawn     bool

	// lines is whether the card draws one line per reading (Lines), and
	// tucked is what it hid to do so, so leaving Lines shows exactly that
	// and not something its consumer had hidden itself.
	lines  bool
	tucked []fyne.CanvasObject

	// meters are the card's meters, which share columns in Lines.
	meters []*Meter

	// scrollers are the runs of rows given to Add (spec 086), whose rows are
	// the card's to dim, restyle and report a click on as its own are.
	scrollers []*Scroller

	// OnDrawnChanged is called when the card starts or stops being drawn, so
	// the owner can start and stop its poll. It is called on the UI thread.
	OnDrawnChanged func(drawn bool)
}

// NewCard builds a card with a title and no rows. It is allowed by default and
// not available: a card is drawn once its source answers, so a window does not
// flash an empty panel on the way up.
func NewCard(title string) *Card {
	c := &Card{
		title:     canvas.NewText(title, nil),
		mark:      canvas.NewText(GoneMarker, nil),
		allowed:   true,
		titleSize: theme.SizeNameText,
	}
	c.title.TextStyle = fyne.TextStyle{Bold: true}
	c.title.Color = c.colour(theme.ColorNamePlaceHolder)
	c.mark.Color = c.colour(theme.ColorNameDisabled)
	c.title.TextSize = c.headingSize()
	c.mark.TextSize = c.textSize()
	c.mark.Hide()

	c.icon = canvas.NewImageFromResource(nil)
	c.icon.FillMode = canvas.ImageFillContain
	c.icon.Hide()
	c.sizeIcon()

	c.end = canvas.NewText("", nil)
	c.end.TextStyle = c.title.TextStyle
	c.end.Color = c.title.Color
	c.end.TextSize = c.title.TextSize
	c.end.Hide()
	c.endIcon = canvas.NewImageFromResource(nil)
	c.endIcon.FillMode = canvas.ImageFillContain
	c.endIcon.Hide()
	c.clear = canvas.NewRectangle(color.Transparent)
	c.clear.Hide()
	c.sizeClear()
	c.sizeIcon()

	c.titleWidth, c.endWidth = &widest{}, &widest{}
	c.header = container.NewHBox(c.icon, container.New(c.titleWidth, c.title), layout.NewSpacer(),
		c.endIcon, container.New(c.endWidth, c.end), c.clear, c.mark)
	c.body = container.NewVBox()

	// A card is a surface of its own, not a run of rows. Fyne cannot draw a
	// translucent window (quirk 32), so the separation a glance window gets
	// from alpha in a toolkit that has it has to come from contrast here: a
	// fill one step from the window's own, and a hairline border one step
	// again. Both are palette tokens, so a card stays a card in every scheme
	// rather than being a grey that happens to work in one.
	c.face = canvas.NewRectangle(c.colour(theme.ColorNameButton))
	c.face.StrokeColor = c.colour(theme.ColorNameSeparator)
	c.face.StrokeWidth = 1
	c.face.CornerRadius = CardRadius

	c.inner = container.New(
		layout.NewCustomPaddedLayout(c.padTop(), c.padBottom(), c.padH(), c.padH()),
		container.NewVBox(c.header, c.body))
	// Tipped rather than WithTip: a card's note is a reading like any other
	// and comes and goes, so the catcher is there from the start and says
	// nothing until there is something to say. See SetTip.
	c.frame = widgets.Tipped(container.NewStack(c.face, c.inner), "").(*fyne.Container)
	c.frame.Hide()
	return c
}

/*
showHeader draws the heading, or leaves it out in Lines.

Lines leaves a card's heading out, since its rows carry the readings -- except
for a card that has nothing but its heading (spec 086). A clock's time and
date are its heading (SetTitle, SetTitleEnd), and without it the card would
be a line with nothing on it.
*/
func (c *Card) showHeader() {
	if c.lines && len(c.body.Objects) > 0 {
		c.header.Hide()
	} else {
		c.header.Show()
	}
}

/*
SetTitle replaces the card's title, for a heading that is a reading: a
clock's time (spec 086). It is drawn in the title's face, which is not
monospace, so the title is held at the widest it has been: a narrower time
leaves the card's width alone, and nothing under it moves.
*/
func (c *Card) SetTitle(text string) {
	c.title.Text = text
	c.refit(c.title)
	c.header.Refresh()
}

// Title is the card's title.
func (c *Card) Title() string { return c.title.Text }

/*
SetTitleEnd puts a second title at the heading's right end, with an icon
before it, in the title's face: a clock's date across from its time (spec
086). It is held at the widest it has been, as the title is. An empty text and
a nil icon take it away.

The icon is drawn as given, as SetIcon's is: pass a resource coloured as the
program's other icons are.
*/
func (c *Card) SetTitleEnd(text string, icon fyne.Resource) {
	c.end.Text = text
	c.end.Hidden = text == ""
	c.endIcon.Resource = icon
	c.endIcon.Hidden = icon == nil
	c.clear.Hidden = text == "" && icon == nil
	c.refit(c.end)
	c.endIcon.Refresh()
	c.header.Refresh()
}

// TitleEnd is the heading's right-hand title, or empty.
func (c *Card) TitleEnd() string { return c.end.Text }

/*
SetTitleSize draws the heading -- the title, the title end and their icons --
at a theme size other than the ordinary text's: theme.SizeNameSubHeadingText
for a heading that is the reading, a clock's time and date (spec 088). The
card's margins, rows and gone marker keep the ordinary size, so a larger
heading makes the card taller by the heading's own growth and nothing else.
*/
func (c *Card) SetTitleSize(name fyne.ThemeSizeName) {
	c.titleSize = name
	c.Restyle()
}

// headingSize is the size the heading is drawn at, in this card's face.
func (c *Card) headingSize() float32 { return c.size(c.titleSize) }

/*
sizeClear sizes the room after the title end, so the end stops short of the
panel's tuck glyph (spec 081) by a quarter of a text size. The glyph stands
half a side margin in from the card's right edge and is glyphEms wide; the
heading ends a whole side margin in, so the glyph reaches glyphEms less half
a margin past it. The header's own padding before the room is the rest of
the gap.

Every card with a title end keeps the room, not only the one under the glyph:
a date that moved when its card was moved down the stack would be a
transient reflow of its own.
*/
func (c *Card) sizeClear() {
	ts := c.textSize()
	w := ts*glyphEms - c.padH()/2 + ts/4 - c.size(theme.SizeNamePadding)
	c.clear.SetMinSize(fyne.NewSize(max(0, w), 0))
}

// widest is a layout that is as wide as the widest its object has asked to
// be since it was last reset, so a text that changes does not narrow what
// holds it.
type widest struct{ width float32 }

func (w *widest) MinSize(objects []fyne.CanvasObject) fyne.Size {
	var size fyne.Size
	for _, o := range objects {
		if o.Visible() {
			size = size.Max(o.MinSize())
		}
	}
	w.width = max(w.width, size.Width)
	return fyne.NewSize(w.width, size.Height)
}

func (w *widest) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	for _, o := range objects {
		o.Move(fyne.NewPos(0, 0))
		o.Resize(fyne.NewSize(o.MinSize().Width, size.Height))
	}
}

// AddRow appends rows in the order they will be drawn.
//
// It panics on a row whose ID the card already holds: two rows answering to
// one name is a fault in the program building the card, and finding it on
// the first build is better than a lookup returning whichever came first.
func (c *Card) AddRow(rows ...*Row) {
	defer c.showHeader()
	for _, r := range rows {
		if err := c.unique(r); err != nil {
			panic(err)
		}
		r.SetTheme(c.th)
		c.rows = append(c.rows, r)
		c.body.Add(r.Object())
	}
}

// ErrRowID is a row whose ID its card already holds.
var ErrRowID = errors.New("glance: the card already has a row with that ID")

// ErrRowIndex is a position outside a card's rows.
var ErrRowIndex = errors.New("glance: no such row position")

/*
InsertRow puts r among the card's rows so that it is drawn at position at:
0 is above every row, len(Rows()) is under the last one.

**Among the rows, never under what follows them.** A card's pieces go into
one column in the order they were added, and a row appended after a plot used
to land under the plot (hayami's "Coolant  no cooler" drawn through its trend).
InsertRow places by the rows, not by the column: at the end it goes straight
after the last row, so a plot, a meter or a grid added after the rows stays
under it. With no rows yet it goes to the top of the card.

It is a change of shape, not of value, and the card grows by the row's height:
call the panel's Resize afterwards, as after SetShown. The row is given the
card's face, and dimmed if the card is showing last-known values. Call it on
the UI thread, as every other method here.

It returns ErrRowIndex for a position outside 0..len(Rows()), and ErrRowID for
an ID the card already holds; the card is unchanged either way.
*/
func (c *Card) InsertRow(at int, r *Row) error {
	defer c.showHeader()
	if at < 0 || at > len(c.rows) {
		return fmt.Errorf("inserting at %d of %d rows: %w", at, len(c.rows), ErrRowIndex)
	}
	if err := c.unique(r); err != nil {
		return err
	}

	// Where in the column: in place of the row now at `at`, or straight after
	// the last row, or -- with no rows at all -- at the top.
	pos := 0
	switch {
	case at < len(c.rows):
		pos = c.position(c.rows[at])
	case len(c.rows) > 0:
		pos = c.position(c.rows[len(c.rows)-1]) + 1
	}

	r.SetTheme(c.th)
	if c.stale {
		rd := r.Reading()
		rd.Stale = true
		r.Set(rd)
	}
	c.rows = slices.Insert(c.rows, at, r)
	c.body.Objects = slices.Insert(c.body.Objects, pos, r.Object())
	c.body.Refresh()
	return nil
}

// RemoveRow takes out the row with id, and reports whether there was one. Like
// InsertRow it changes the card's shape: call the panel's Resize afterwards.
func (c *Card) RemoveRow(id string) bool {
	if id == "" {
		return false
	}
	for i, r := range c.rows {
		if r.id != id {
			continue
		}
		c.rows = slices.Delete(c.rows, i, i+1)
		if pos := c.position(r); pos >= 0 {
			c.body.Objects = slices.Delete(c.body.Objects, pos, pos+1)
		}
		c.body.Refresh()
		return true
	}
	return false
}

// RowByID is the row with id, or nil. An empty id finds nothing.
func (c *Card) RowByID(id string) *Row {
	if id == "" {
		return nil
	}
	for _, r := range c.rows {
		if r.id == id {
			return r
		}
	}
	return nil
}

// unique is ErrRowID when r's ID is already one of the card's.
func (c *Card) unique(r *Row) error {
	if r.id != "" && c.RowByID(r.id) != nil {
		return fmt.Errorf("row %q: %w", r.id, ErrRowID)
	}
	return nil
}

// position is where r's object stands in the card's column, or -1.
func (c *Card) position(r *Row) int {
	return slices.Index(c.body.Objects, r.Object())
}

// AddObject appends something that is not a row — a Sparkline, a progress bar
// — under the rows added so far.
func (c *Card) AddObject(o fyne.CanvasObject) {
	defer c.showHeader()
	c.themeOne(o)
	c.objects = append(c.objects, o)
	c.body.Add(o)
	if c.lines && o.Visible() {
		o.Hide()
		c.tucked = append(c.tucked, o)
	}
}

/*
Piece is something a card holds that has a face of its own to keep in step: a
Meter, a CellGrid, a Row.

It exists because AddObject is given an object and an object cannot be asked
for its theme. A meter handed over as meter.Object() is a *fyne.Container by
the time the card sees it, so the card has no way to tell it that the panel's
face changed -- which left a meter and a grid of cells as the two pieces still
wearing the application's.
*/
type Piece interface {
	Object() fyne.CanvasObject
	SetTheme(fyne.Theme)
}

// Add puts pieces in the card, in the order they will be drawn, and keeps them
// in step with the card's face.
//
// Prefer it to AddObject for anything this package builds. AddObject stays for
// a consumer's own object, which is its own to style.
func (c *Card) Add(pieces ...Piece) {
	defer c.showHeader()
	for _, p := range pieces {
		p.SetTheme(c.th)
		c.pieces = append(c.pieces, p)
		if m, ok := p.(*Meter); ok {
			c.meters = append(c.meters, m)
			m.peers = &c.meters
		}
		if s, ok := p.(*Scroller); ok {
			c.scrollers = append(c.scrollers, s)
			if c.stale {
				s.setStale(true)
			}
		}
		c.body.Add(p.Object())
		if c.lines {
			if l, ok := p.(liner); ok {
				l.setLines(true)
			} else if _, isRow := p.(*Row); !isRow && p.Object().Visible() {
				p.Object().Hide()
				c.tucked = append(c.tucked, p.Object())
			}
		}
	}
}

/*
SetTheme gives the card, its rows and whatever has been added to it a theme of
its own, and repaints in it. Nil returns it to the application's.

A panel calls this for every card in it; there is nothing for a consumer to do.
It exists because the application's theme belongs to the window that has
overlays -- see themed.
*/
func (c *Card) SetTheme(th fyne.Theme) {
	c.themed.SetTheme(th)
	for _, r := range c.rows {
		r.SetTheme(th)
	}
	for _, o := range c.objects {
		c.themeOne(o)
	}
	for _, p := range c.pieces {
		p.SetTheme(th)
	}
	c.Restyle()
}

// themeOne hands the card's theme to something added to it, when it is
// something that can take one.
//
// A meter, a sparkline and a grid of cells all can. A plain canvas object
// cannot and is left alone: a consumer that adds its own object is responsible
// for how it draws, which is the same bargain AddObject has always offered.
func (c *Card) themeOne(o fyne.CanvasObject) {
	if t, ok := o.(interface{ SetTheme(fyne.Theme) }); ok {
		t.SetTheme(c.th)
	}
}

// Rows are the card's rows, in order, for a caller that keeps no handles of
// its own and for tests.
func (c *Card) Rows() []*Row { return c.rows }

// SetAllowed records the user's toggle: whether this card is wanted at all.
// It is the context menu's half of the decision.
func (c *Card) SetAllowed(allowed bool) {
	c.allowed = allowed
	c.apply()
}

// Allowed reports the user's toggle.
func (c *Card) Allowed() bool { return c.allowed }

// SetAvailable records whether the source has anything to say. It is the
// poll's half of the decision: true on the first answer, and left true
// afterwards, because a source that has answered once and stopped is stale
// rather than absent. Use SetStale for that.
func (c *Card) SetAvailable(available bool) {
	c.available = available
	c.apply()
}

// Available reports whether the source has anything to say.
func (c *Card) Available() bool { return c.available }

/*
SetIcon puts a glyph before the card's title. A nil resource takes it away.

Before the title and not after it, because the icon is what the eye finds
first when it is scanning a stack of cards for one of them -- which is the
whole reason a card has an icon rather than making do with its name.

It is sized from the text, not in pixels: an icon set for one face is
proportionally wrong at another, and a glance panel's text size is a setting
the user moves. IconScale is the factor.
*/
func (c *Card) SetIcon(res fyne.Resource) {
	c.icon.Resource = res
	if res == nil {
		c.icon.Hide()
		return
	}
	c.sizeIcon()
	c.icon.Show()
	c.icon.Refresh()
}

// Icon is the card's glyph, or nil.
func (c *Card) Icon() fyne.Resource { return c.icon.Resource }

// sizeIcon measures the icon against the card's own face.
func (c *Card) sizeIcon() {
	edge := c.headingSize() * IconScale
	for _, img := range []*canvas.Image{c.icon, c.endIcon} {
		if img == nil {
			continue
		}
		img.SetMinSize(fyne.NewSize(edge, edge))
		img.Resize(fyne.NewSize(edge, edge))
	}
}

// SetStale dims every row and marks the header, for a source that was
// answering and has stopped. The card keeps its last values: a reader's
// question is whether they are still true, and a dim number answers it without
// moving anything. Recovery clears the marker.
func (c *Card) SetStale(stale bool) {
	c.setStale(stale, stale)
}

/*
SetLastKnown dims every row without marking the header.

For values that are the last ones heard but whose source has not *stopped* --
a panel showing what it knew when it was last running, before this run's first
reading has landed. The values are as provisional as a stale card's and are
dimmed for the same reason, but GoneMarker would be a lie: nothing is
unavailable, nothing has been asked yet.

The distinction is the consumer's to make and it is a real one. A card that
said "(unavailable)" two seconds after the program started would be reporting
a fault where there is only a device that has not woken up.
*/
func (c *Card) SetLastKnown(dim bool) {
	c.setStale(dim, false)
}

// setStale is the two halves of staleness, which are separate because a card
// can want the dimming without the word.
func (c *Card) setStale(dim, mark bool) {
	if c.stale != dim {
		c.stale = dim
		for _, r := range c.rows {
			rd := r.Reading()
			rd.Stale = dim
			r.Set(rd)
		}
		for _, s := range c.scrollers {
			s.setStale(dim)
		}
	}
	if mark {
		c.mark.Show()
	} else {
		c.mark.Hide()
	}
}

// Stale reports whether the card is showing last-known values.
func (c *Card) Stale() bool { return c.stale }

// Drawn reports whether the card is on screen: allowed by the user and
// available from its source.
func (c *Card) Drawn() bool { return c.drawn }

// apply reconciles the two halves and tells the owner when the answer changes.
func (c *Card) apply() {
	drawn := c.allowed && c.available
	if drawn == c.drawn {
		return
	}
	c.drawn = drawn
	if drawn {
		c.frame.Show()
	} else {
		c.frame.Hide()
	}
	if c.OnDrawnChanged != nil {
		c.OnDrawnChanged(drawn)
	}
}

// Restyle repaints the card and its rows in the current theme.
func (c *Card) Restyle() {
	// The icon is measured from the text, so a change of face moves it too.
	c.sizeIcon()
	c.sizeClear()
	c.title.TextSize = c.headingSize()
	c.title.Color = c.colour(theme.ColorNamePlaceHolder)
	c.mark.TextSize = c.textSize()
	c.mark.Color = c.colour(theme.ColorNameDisabled)
	c.end.TextSize = c.headingSize()
	c.end.Color = c.title.Color
	// A new face measures the titles again from nothing: the widest they
	// were was in the old one.
	c.titleWidth.width, c.endWidth.width = 0, 0
	c.face.FillColor = c.colour(theme.ColorNameButton)
	c.face.StrokeColor = c.colour(theme.ColorNameSeparator)
	c.face.CornerRadius = CardRadius
	// The inner margins are a factor of the text size, so a size change moves
	// them. A layout is a value here, not a live object: replacing it and
	// refreshing is what re-measures the card (a Refresh alone keeps the old
	// one's numbers).
	c.inner.Layout = layout.NewCustomPaddedLayout(
		c.padTop(), c.padBottom(), c.padH(), c.padH())
	c.inner.Refresh()
	c.face.Refresh()
	c.title.Refresh()
	c.mark.Refresh()
	c.refit(c.title, c.end, c.mark)
	c.header.Refresh()
	for _, r := range c.rows {
		r.Restyle()
	}
	for _, sc := range c.scrollers {
		sc.Restyle()
	}
}

// Object is the card's content, for a caller assembling a panel by hand.
func (c *Card) Object() fyne.CanvasObject { return c.frame }

/*
SetTip is what the card says when the pointer rests on it: detail there is no
room to draw.

A glance card is small on purpose and grows with what it holds, so a section
with more to report than fits has two bad choices -- widen the card, which
moves everything beside it, or drop the detail. This is the third: the card
keeps its size and the detail is a hover away. hayami's peripherals section is
the case it was built for, where a third pair of headphones would otherwise
have made the card a third wider.

An empty text says nothing, so a note that comes and goes needs no branch.

**The panel has to have a tip layer** or the note is drawn as an overlay, which
takes every pointer event in the window while it is up (quirk 26). NewPanel
puts one in.
*/
func (c *Card) SetTip(text string) { widgets.SetTip(c.frame, text) }

// Tip is what the card says on hover, for a test that would otherwise have to
// hover to find out.
func (c *Card) Tip() string { return widgets.TipText(c.frame) }

// liner is a piece that draws itself differently as one line per reading: a
// Meter, a CellGrid.
type liner interface{ setLines(on bool) }

/*
SetArrangement draws the card for a panel arrangement. A panel calls it for
each of its cards; a card shown outside a panel (the gallery) can be told
directly. Stack and Grid draw the card as it was built.

Lines (spec 057) draws one line per reading: the heading is left out, rows
stay as they are, a meter and a grid of cells draw as lines, and everything
else -- a plot, anything added with AddObject -- is hidden, because a line per
reading has no room for what is not a reading. It is a change of shape: call
the panel's Resize afterwards, as Panel.SetArrangement does. Rows are not
touched, so their IDs (spec 056) and the consumer's handles survive the switch.

The marker GoneMarker lives in the heading and goes with it; the rows of a
stale card are still dimmed, which is how the card says it in Lines.
*/
func (c *Card) SetArrangement(a Arrangement) {
	lines := a == Lines
	if c.lines == lines {
		return
	}
	c.lines = lines
	c.showHeader()
	if lines {
		c.tucked = c.tucked[:0]
		tuck := func(o fyne.CanvasObject) {
			if o.Visible() {
				o.Hide()
				c.tucked = append(c.tucked, o)
			}
		}
		for _, o := range c.objects {
			tuck(o)
		}
		for _, p := range c.pieces {
			if l, ok := p.(liner); ok {
				l.setLines(true)
				continue
			}
			if _, isRow := p.(*Row); !isRow {
				tuck(p.Object())
			}
		}
	} else {
		for _, o := range c.tucked {
			o.Show()
		}
		c.tucked = c.tucked[:0]
		for _, p := range c.pieces {
			if l, ok := p.(liner); ok {
				l.setLines(false)
			}
		}
	}
	c.body.Refresh()
	c.inner.Refresh()
}
