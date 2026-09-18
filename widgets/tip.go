package widgets

import (
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

// TipDelay is how long the pointer rests on a control before its tip appears.
// Long enough that moving across a form does not flash a tip per control.
const TipDelay = 500 * time.Millisecond

// tipMaxWidth is how wide a tip is allowed to get before it wraps. A tip is a
// sentence or two; a wider one reads as a paragraph pinned to the pointer.
const tipMaxWidth float32 = 420

// tipOffset is how far below and right of the pointer the tip sits, so it does
// not appear under the cursor itself.
var tipOffset = fyne.NewPos(12, 20)

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

// tipArea is the transparent catcher: it draws nothing and exists to notice the
// pointer.
type tipArea struct {
	widget.BaseWidget
	text  string
	under fyne.CanvasObject

	mu    sync.Mutex
	timer *time.Timer
	pop   *widget.PopUp
	at    fyne.Position
}

// CreateRenderer implements fyne.Widget. Nothing is drawn: the control beneath
// is the thing being looked at.
func (t *tipArea) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(canvas.NewRectangle(nil))
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

// show puts the tip up at the pointer, clamped to the canvas. On the UI thread.
func (t *tipArea) show() {
	d := fyne.CurrentApp().Driver()
	c := d.CanvasForObject(t)
	if c == nil {
		return // detached between the wait starting and ending
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.timer == nil || t.pop != nil {
		return // the pointer left, or a tip is already up
	}

	label := widget.NewLabel(t.text)
	label.Wrapping = fyne.TextWrapWord
	body := container.NewPadded(label)
	pop := widget.NewPopUp(body, c)

	pop.Resize(tipSize(t.text, label, body, c.Size()))
	pop.ShowAtPosition(clampToCanvas(
		d.AbsolutePositionForObject(t).Add(t.at).Add(tipOffset), pop.Size(), c.Size()))
	t.pop = pop
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

// hide takes the tip down and cancels a pending one.
func (t *tipArea) hide() {
	t.mu.Lock()
	timer, pop := t.timer, t.pop
	t.timer, t.pop = nil, nil
	t.mu.Unlock()
	if timer != nil {
		timer.Stop()
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
