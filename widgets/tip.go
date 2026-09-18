package widgets

import (
	"image/color"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	fynetheme "fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	fdtheme "github.com/ushineko/fynedesygn/theme"
)

/*
TipDelay is how long the pointer rests on a control before its tip appears.

A second, not half of one. Half a second is about how long it takes to move the
pointer onto a button and press it, so a tip arrived at the moment of the click
on controls nobody was asking about -- and the note that was meant to explain a
control was instead something to look past on the way to using it.
*/
const TipDelay = time.Second

// tipMaxWidth is how wide a tip is allowed to get before it wraps. A tip is a
// sentence or two; a wider one reads as a paragraph pinned to the pointer.
const tipMaxWidth float32 = 420

// tipOffset is how far below and right of the pointer the tip sits, so it does
// not appear under the cursor itself.
var tipOffset = fyne.NewPos(12, 20)

// The tip's box, drawn here rather than by a popup: a hairline border so the
// tip reads as a separate thing over whatever it covers, and the same corner as
// the rest of the scheme.
const (
	tipStroke float32 = 1
	tipRadius float32 = 4
)

/*
WithTip returns o with a note that appears while the pointer rests on it.

For the guidance a control cannot say in its own label: what a cheat actually
does, which other setting it depends on, why a value matters. Fyne has no
tooltip of its own, and the alternative -- printing the note under every control
-- turns a row into a paragraph and a form into something three times taller.

An empty text returns o unchanged, so a caller can annotate conditionally
without branching.

How it works, because it is not obvious: the note is a transparent object
stacked over o rather than a widget wrapping it. Fyne finds the object for a
pointer event by walking the visible tree and keeping the last match, so the
deepest object wins -- a widget merely containing a Check would never see the
hover, because the Check is walked after it. Stacked over it instead, the
catcher is walked last and takes the hover; it is not Tappable, so the tap
search walks past it to the control underneath, which keeps its own behaviour
and does not know it has been annotated.
*/
func WithTip(o fyne.CanvasObject, text string) fyne.CanvasObject {
	if text == "" {
		return o
	}
	t := &tipArea{text: text, under: o}
	t.ExtendBaseWidget(t)
	return container.NewStack(o, t)
}

/*
TipText is the note a wrapped object carries, or "" when it carries none.

A tip is a popup raised on hover, so its text is nowhere in the tree until
somebody hovers -- which leaves no way to check that a control which shows only
an icon still says what it is. This is that way.
*/
func TipText(o fyne.CanvasObject) string {
	box, ok := o.(*fyne.Container)
	if !ok {
		return ""
	}
	for _, child := range box.Objects {
		if t, is := child.(interface{ Tip() string }); is {
			return t.Tip()
		}
	}
	return ""
}

/*
Tip is the note this catcher shows, which makes the text readable without
hovering.

An interface method rather than an exported field, so a walker can find a tip
without importing this package -- which is what keeps the test helpers out of
the import graph of the package they help test.
*/
func (t *tipArea) Tip() string { return t.text }

// tipArea is the transparent catcher: it draws nothing and exists to notice the
// pointer.
type tipArea struct {
	widget.BaseWidget
	text  string
	under fyne.CanvasObject

	mu    sync.Mutex
	timer *time.Timer
	// shown is the tip in the window's tip layer, and layer the layer it is
	// in. pop is the fallback for a window that has no layer.
	shown *fyne.Container
	layer *fyne.Container
	pop   *widget.PopUp
	at    fyne.Position
}

// CreateRenderer implements fyne.Widget. Nothing is drawn: the control beneath
// is the thing being looked at.
//
// Transparent rather than a nil colour. A nil fill is invisible under the GL
// painter and a nil dereference under the software one, so a nil here is a
// widget that works in the window and crashes anything that renders it to an
// image -- a test, or a screenshot.
func (t *tipArea) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(canvas.NewRectangle(color.Transparent))
}

// MouseIn starts the wait. The tip is not shown yet: a pointer crossing a form
// should not leave a trail of them.
func (t *tipArea) MouseIn(e *desktop.MouseEvent) {
	t.mu.Lock()
	t.at = e.Position
	if t.timer != nil {
		t.timer.Stop()
	}
	t.timer = time.AfterFunc(TipDelay, func() { fyne.Do(t.show) })
	t.mu.Unlock()
	arm(t)

	// The control underneath still highlights: it is only the hover routing
	// that has been taken from it, not its behaviour.
	if h, ok := t.under.(desktop.Hoverable); ok {
		h.MouseIn(e)
	}
}

// MouseMoved follows the pointer so the tip appears where it came to rest.
func (t *tipArea) MouseMoved(e *desktop.MouseEvent) {
	t.mu.Lock()
	t.at = e.Position
	t.mu.Unlock()
	if h, ok := t.under.(desktop.Hoverable); ok {
		h.MouseMoved(e)
	}
}

// MouseOut cancels the wait and takes down whatever is showing.
func (t *tipArea) MouseOut() {
	t.hide()
	if h, ok := t.under.(desktop.Hoverable); ok {
		h.MouseOut()
	}
}

/*
NewTipLayer returns the container a window's tips are drawn in.

A window that wants tips puts one over its content, last, so it is drawn on top:

	container.NewStack(everythingElse, widgets.NewTipLayer())

The shell does this for every program it builds, so a program on the shell needs
nothing. A program that assembles its own window does it itself.

**Why a layer rather than a popup.** A popup is an overlay, and Fyne routes
pointer events to the top overlay *only*: while one is up, the walk that finds
what was clicked never reaches the window's content, so every click goes to the
overlay -- a non-modal one dismisses itself and swallows the click, which is a
button that needs two clicks whenever its own tip is showing. A tip drawn into
the content instead is found by the same walk, and skipped by it, because the
things it is made of are not tappable. See quirk 26.
*/
func NewTipLayer() *fyne.Container { return container.New(tipLayout{}) }

/*
tipLayout is what marks a container as a tip layer, and what keeps it out of the
way.

It lays nothing out: a tip is placed at an absolute position in canvas
coordinates, and the layer exists only to hold it above everything else. Its
minimum is zero so that being in a Stack costs the window nothing.
*/
type tipLayout struct{}

func (tipLayout) Layout([]fyne.CanvasObject, fyne.Size) {}
func (tipLayout) MinSize([]fyne.CanvasObject) fyne.Size { return fyne.Size{} }

/*
HideTips takes down every tip that is showing or waiting.

For the moment a control is used rather than merely hovered: clicking a button
that opens a menu, a dialog or anything else over the window leaves the tip
where it was, because the pointer never left the control and nothing said it
had. Whatever opens something calls this first.

It is not the same as the pointer leaving: a waiting tip is cancelled too, so
one that was a moment from appearing does not arrive on top of the menu that
was just opened.

Every tip rather than one window's, because a pointer rests on one control at a
time: whatever is showing is the tip for the control that was just used.
*/
func HideTips() {
	armedMu.Lock()
	on := make([]*tipArea, 0, len(armed))
	for t := range armed {
		on = append(on, t)
	}
	armedMu.Unlock()

	for _, t := range on {
		t.hide()
	}
}

// armed is every tip that is waiting or showing. Small -- one or two at a time
// in practice -- and emptied by hide, which every path ends in.
var (
	armedMu sync.Mutex
	armed   = map[*tipArea]struct{}{}
)

// arm and disarm keep that set.
func arm(t *tipArea) {
	armedMu.Lock()
	armed[t] = struct{}{}
	armedMu.Unlock()
}

func disarm(t *tipArea) {
	armedMu.Lock()
	delete(armed, t)
	armedMu.Unlock()
}

// TipLayerIn is a canvas's tip layer, or nil when its window has none. For a
// window that assembles itself and wants to check it did.
func TipLayerIn(c fyne.Canvas) *fyne.Container { return tipLayerIn(c) }

// tipLayerIn is the tip layer of a canvas, or nil when the window has none.
func tipLayerIn(c fyne.Canvas) *fyne.Container {
	if c == nil {
		return nil
	}
	return findTipLayer(c.Content())
}

/*
findTipLayer walks a tree for the marked container.

Containers only, and deliberately: a tip layer is put in a window's content by
whoever assembles it, so it is always a structural child and never inside a
widget's renderer. Descending into renderers would mean calling CreateRenderer
on every widget in the window -- which *builds* one, a whole parallel tree, every
time a tip is about to be shown.
*/
func findTipLayer(o fyne.CanvasObject) *fyne.Container {
	box, ok := o.(*fyne.Container)
	if !ok {
		return nil
	}
	if _, ok := box.Layout.(tipLayout); ok {
		return box
	}
	for _, child := range box.Objects {
		if got := findTipLayer(child); got != nil {
			return got
		}
	}
	return nil
}

// show puts the tip up at the pointer, clamped to the canvas. On the UI thread.
func (t *tipArea) show() {
	d := fyne.CurrentApp().Driver()
	c := d.CanvasForObject(t)
	if c == nil {
		return // detached between the wait starting and ending
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.timer == nil || t.shown != nil {
		return // the pointer left, or a tip is already up
	}

	label := widget.NewLabel(t.text)
	label.Wrapping = fyne.TextWrapWord
	body := container.NewPadded(label)
	size := tipSize(t.text, label, body, c.Size())

	back := canvas.NewRectangle(tipBackground())
	back.StrokeColor = tipBorder()
	back.StrokeWidth = tipStroke
	back.CornerRadius = tipRadius
	tip := container.NewStack(back, body)
	tip.Resize(size)

	at := clampToCanvas(d.AbsolutePositionForObject(t).Add(t.at).Add(tipOffset), size, c.Size())

	layer := tipLayerIn(c)
	if layer == nil {
		// No layer in this window: fall back to a popup, which is a tip that
		// eats the next click. Better than no guidance at all, and the reason
		// the shell always provides one.
		pop := widget.NewPopUp(body, c)
		pop.Resize(size)
		pop.ShowAtPosition(at)
		t.pop = pop
		return
	}

	// The layer is inside the content, which the window pads, so the tip is
	// placed relative to the layer rather than to the canvas.
	tip.Move(at.Subtract(d.AbsolutePositionForObject(layer)))
	layer.Add(tip)
	layer.Refresh()
	t.shown, t.layer = tip, layer
}

// tipBackground and tipBorder are the tooltip's own colours from the active
// scheme, which is what the Tooltip role in the palette is for. A theme that is
// not this library's falls back to Fyne's overlay colours.
func tipBackground() color.Color {
	if p, ok := palette(); ok {
		return p.TooltipBG
	}
	return fynetheme.Color(fynetheme.ColorNameOverlayBackground)
}

func tipBorder() color.Color {
	if p, ok := palette(); ok {
		return p.Separator
	}
	return fynetheme.Color(fynetheme.ColorNameSeparator)
}

// palette is the active scheme when the app is themed by this library.
func palette() (fdtheme.Palette, bool) {
	app := fyne.CurrentApp()
	if app == nil {
		return fdtheme.Palette{}, false
	}
	th, ok := app.Settings().Theme().(fdtheme.Theme)
	if !ok {
		return fdtheme.Palette{}, false
	}
	return th.Palette(), true
}

/*
tipSize measures the tip at the width it will be drawn at.

Two things make this less obvious than it looks.

A wrapping label reports a minimum width of about one character -- it is built to
fill whatever width it is given, so asking it how wide the text is returns
nothing useful. The natural width comes from a label that does not wrap.

And a wrapping label does not know its own height until it has a width. So the
width is chosen first, the body is resized to it, and only then does MinSize
report the height the wrapping produced. Skipping that pass is what drew the
first version one line tall with its text running out of the box and off the
window.
*/
func tipSize(text string, label, body fyne.CanvasObject, canvasSize fyne.Size) fyne.Size {
	natural := widget.NewLabel(text).MinSize().Width
	padding := body.MinSize().Width - label.MinSize().Width
	width := natural + padding

	limit := tipMaxWidth
	// Never wider than the window either: tipMaxWidth is a preference, and a
	// narrow window is a smaller limit than it.
	if edge := canvasSize.Width - tipOffset.X*2; edge > 0 && edge < limit {
		limit = edge
	}
	if width > limit {
		width = limit
	}

	body.Resize(fyne.NewSize(width, body.MinSize().Height))
	return fyne.NewSize(width, body.MinSize().Height)
}

/*
tip is what is on screen, whichever way it is drawn, or nil.

Both paths draw the same box; only where it is parented differs, so a test that
wants to read a tip or measure it should not have to know which one is in use.
*/
func (t *tipArea) tip() fyne.CanvasObject {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.shown != nil {
		return t.shown
	}
	if t.pop != nil {
		return t.pop
	}
	return nil
}

// hide takes the tip down and cancels a pending one.
func (t *tipArea) hide() {
	t.mu.Lock()
	timer, shown, layer, pop := t.timer, t.shown, t.layer, t.pop
	t.timer, t.shown, t.layer, t.pop = nil, nil, nil, nil
	t.mu.Unlock()
	disarm(t)
	if timer != nil {
		timer.Stop()
	}
	if shown != nil && layer != nil {
		layer.Remove(shown)
		layer.Refresh()
	}
	if pop != nil {
		pop.Hide()
	}
}

// clampToCanvas keeps a tip inside the window. Near the right or bottom edge it
// is pulled back rather than drawn off the canvas, where it would be a note
// nobody can read.
func clampToCanvas(at fyne.Position, size, canvasSize fyne.Size) fyne.Position {
	if right := at.X + size.Width; right > canvasSize.Width {
		at.X = canvasSize.Width - size.Width
	}
	if bottom := at.Y + size.Height; bottom > canvasSize.Height {
		at.Y = canvasSize.Height - size.Height
	}
	if at.X < 0 {
		at.X = 0
	}
	if at.Y < 0 {
		at.Y = 0
	}
	return at
}
