package logpane

import (
	"fmt"
	"image/color"
	"io"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	fynetheme "fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	fd "github.com/ushineko/fynedesygn"
	"github.com/ushineko/fynedesygn/widgets"
)

/*
DefaultHeight is the pane's height unless Options says otherwise.

A minimum rather than a maximum. In a fixed region -- a Border's bottom, a VBox
-- a minimum is the height, which is what a pane has always been. Inside a
draggable divider (Shell.VSplit) it is as small as the pane may be dragged, and
the pane takes whatever else the divider gives it.
*/
const DefaultHeight float32 = 360

// PumpInterval is how often a pane redraws while lines arrive.
const PumpInterval = 100 * time.Millisecond

// Options shape the pane's widget.
type Options struct {
	// Title is the bold label at the head of the pane.
	Title string
	// Height is the list's minimum height; 0 means DefaultHeight. In a fixed
	// region that minimum is the height; inside Shell.VSplit it is how small
	// the pane may be dragged.
	Height float32
	// TextSize of the rows; 0 means the theme's text size. Programs with a
	// console font size pass it here.
	TextSize float32
	// OnClear, when set, adds a Clear button that resets the model and calls
	// it. Nil for a pane whose content comes from a file rather than from
	// this process.
	OnClear func()
	// Clipboard receives the log on Copy; nil hides the button.
	Clipboard fyne.Clipboard
	// Flash reports Copy's result; nil reports nothing.
	Flash func(text string, st fd.Status)
}

/*
Pane is a log model plus the live widgets drawing it. The widgets are updated
in place: rebuilding the section on every line is exactly the reflow the
design rule forbids. It implements the shell's Detacher shape.
*/
type Pane struct {
	log *Model

	// follow is the auto-scroll state: true until the user scrolls up.
	follow bool
	// wantOffset is the scroll offset left behind by the last auto-scroll.
	wantOffset float32

	root      *fyne.Container
	box       *fyne.Container
	list      *widget.List
	counter   *widget.Label
	followBox *widget.Check
	stamp     *widget.Label
	updated   time.Time

	// rows is the model wrapped to the pane's width: one row per drawn line,
	// which is what keeps every row the same height. wrapCols, wrapLen and
	// wrapDropped are what they were built from, so a tick that changed
	// nothing rebuilds nothing.
	rows        []Line
	wrapCols    int
	wrapLen     int
	wrapDropped int
	textSize    float32
	charW       float32
	needsReflow bool
}

// New makes a pane over model; nil makes a model with the default cap.
func New(model *Model) *Pane {
	if model == nil {
		model = NewModel(0)
	}
	return &Pane{log: model, follow: true}
}

// Model is the pane's log, for programs that feed it directly.
func (p *Pane) Model() *Model { return p.log }

// Following reports whether the pane is scrolling to the end.
func (p *Pane) Following() bool { return p.follow }

// SetFollowing turns following on or off and redraws, so a program can
// re-arm the tail when a new job starts after the reader scrolled up during
// the previous one. The Follow box shows the new state.
func (p *Pane) SetFollowing(on bool) {
	p.follow = on
	if on {
		p.wantOffset = 0
	}
	p.Draw()
}

// Detach forgets the live widgets. Called when the content pane is replaced.
func (p *Pane) Detach() {
	p.root, p.box, p.list = nil, nil, nil
	p.counter, p.followBox, p.stamp = nil, nil, nil
	// The rows describe a width the next pane has not got yet.
	p.rows, p.wrapCols, p.wrapLen, p.wrapDropped = nil, 0, 0, 0
	// The next list starts at the top, so the offset the last automatic scroll
	// left behind describes a widget that no longer exists.
	p.wantOffset = 0
}

// Log appends a timestamped line: "15:04:05 LEVEL message". Safe from any
// goroutine.
func (p *Pane) Log(level Level, msg string) {
	p.log.Append(level, time.Now().Format("15:04:05")+" "+level.String()+" "+msg)
}

// Writer is an io.Writer that turns each written line into a row at level,
// for a process's stdout or a logger's output. Partial lines are held until
// their newline arrives.
func (p *Pane) Writer(level Level) io.Writer { return &lineWriter{p: p, level: level} }

type lineWriter struct {
	p     *Pane
	level Level
	mu    sync.Mutex
	rest  string
}

func (w *lineWriter) Write(b []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	text := w.rest + string(b)
	lines := strings.Split(text, "\n")
	w.rest = lines[len(lines)-1]
	for _, line := range lines[:len(lines)-1] {
		w.p.log.Append(w.level, line)
	}
	return len(b), nil
}

// Widget builds the pane: a header row with the title, the counter, the
// Follow box and Copy / Clear, over the fixed-height list.
func (p *Pane) Widget(o Options) fyne.CanvasObject {
	log := p.log
	height := o.Height
	if height <= 0 {
		height = DefaultHeight
	}
	size := o.TextSize
	if size <= 0 {
		size = fynetheme.TextSize()
	}
	p.textSize = size
	// Nothing is wrapped yet: the pane has no width until it is laid out, and
	// the first thing the list does is ask how many rows there are.
	p.rows, p.wrapCols, p.wrapLen, p.wrapDropped = nil, -1, -1, -1
	list := p.buildList()

	p.counter = widget.NewLabel("")
	p.counter.Importance = widget.LowImportance
	p.stamp = widget.NewLabel("")
	p.stamp.Importance = widget.LowImportance

	follow := widget.NewCheck("Follow the tail", func(b bool) {
		p.follow = b
		if b {
			p.Draw()
		}
	})
	follow.SetChecked(p.follow)
	p.followBox = follow

	controls := container.NewHBox(p.counter, follow)
	if o.Clipboard != nil {
		controls.Add(widget.NewButtonWithIcon("Copy", fynetheme.ContentCopyIcon(), func() {
			o.Clipboard.SetContent(log.Text())
			if o.Flash != nil {
				o.Flash(fmt.Sprintf("Copied %d line(s) to the clipboard.", log.Len()), fd.StatusGood)
			}
		}))
	}
	if o.OnClear != nil {
		controls.Add(widget.NewButtonWithIcon("Clear", fynetheme.ContentClearIcon(), func() {
			log.Reset()
			o.OnClear()
			p.Draw()
		}))
	}
	bar := container.NewBorder(nil, nil,
		widget.NewLabelWithStyle(o.Title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		controls, widget.NewLabel(""))
	p.Draw()
	// The list inside a container whose layout notices the width. A layout's
	// Layout runs on every resize, which is the only resize callback Fyne
	// offers (docs/fyne-quirks.md, 14) -- and without it the pane wraps to
	// whatever width it was first drawn at and never reflows.
	watched := newWrapWatch(p, list)
	p.root = container.NewBorder(bar, p.stamp, nil, nil, widgets.FixedHeight(watched, height))
	return p.root
}

// rowColor paints a row from the active scheme; Info and Debug are plain
// foreground so the log reads as text with the problems standing out.
func rowColor(l Level) color.Color {
	if l == Debug || l == Info {
		return fynetheme.Color(fynetheme.ColorNameForeground)
	}
	return widgets.StatusColor(l.Status())
}

/*
Draw refreshes the pane and decides whether to stay at the end. Called on the
UI thread. The offset dance is the whole of the auto-scroll rule; see
FollowTail.
*/
func (p *Pane) Draw() {
	if p.list == nil {
		return
	}
	if p.counter != nil {
		text := fmt.Sprintf("%d line(s)", p.log.Len())
		if d := p.log.Dropped(); d > 0 {
			text = fmt.Sprintf("%d line(s), %d older dropped", p.log.Len(), d)
		}
		p.counter.SetText(text)
	}
	if p.stamp != nil && !p.updated.IsZero() {
		p.stamp.SetText(fmt.Sprintf("Last updated %s", p.updated.Format("15:04:05")))
	}
	p.follow = FollowTail(p.follow, p.list.GetScrollOffset(), p.wantOffset)
	// The box is the state, so it has to say what the state is.
	if p.followBox != nil && p.followBox.Checked != p.follow {
		p.followBox.SetChecked(p.follow)
	}
	p.list.Refresh()
	if !p.follow {
		return
	}
	p.list.ScrollToBottom()
	p.wantOffset = p.list.GetScrollOffset()
}

/*
rewrap rebuilds the visual rows when the pane's width or its content changed.

Called from the list's own length function, which is the one place reached
however the pane is driven. It cannot hang off the pump: a pane that is only
written to and never pumped -- which is most of them, because a section that
rebuilds on its own redraws the log with it -- would never wrap at all. That is
exactly what shipped in 0.1.16 and did nothing.

The width comes from the list, which is the object the rows are drawn in, and it
is read here rather than on a resize because Fyne has no resize callback
(docs/fyne-quirks.md, 14). Rebuilt only when something moved, so a log that is
not changing costs three comparisons per refresh.
*/
func (p *Pane) rewrap() {
	if p.list == nil {
		return
	}
	cols := columnsFor(p.list.Size().Width, p.charWidth())
	if cols == p.wrapCols && p.log.Len() == p.wrapLen && p.log.Dropped() == p.wrapDropped {
		return
	}
	if cols != p.wrapCols {
		// Noted rather than acted on: rewrap is called from the list's own
		// length function, and a list cannot be replaced from inside it.
		p.needsReflow = true
	}
	p.wrapCols, p.wrapLen, p.wrapDropped = cols, p.log.Len(), p.log.Dropped()
	p.rows = wrapRows(p.log, cols)
}

/*
buildList makes the row list.

A function because the pane makes a new one when the wrapping changes: see
reflow, which explains why an old list cannot be told about it.
*/
func (p *Pane) buildList() *widget.List {
	size := p.textSize
	list := widget.NewList(
		func() int {
			p.rewrap()
			return len(p.rows)
		},
		func() fyne.CanvasObject {
			t := canvas.NewText("", fynetheme.Color(fynetheme.ColorNameForeground))
			t.TextStyle = fyne.TextStyle{Monospace: true}
			t.TextSize = size
			return t
		},
		func(i widget.ListItemID, obj fyne.CanvasObject) {
			t := obj.(*canvas.Text)
			if i < 0 || i >= len(p.rows) {
				t.Text = ""
				t.Refresh()
				return
			}
			line := p.rows[i]
			t.Color = rowColor(line.Level)
			t.TextSize = size
			t.Text = line.Text
			t.Refresh()
		},
	)
	// A canvas.Text row is one text line high; without separators the list
	// packs them like a terminal.
	list.HideSeparators = true
	p.list = list
	return list
}

/*
wrapWatch wraps the list in a widget that notices how wide it has become.

Fyne offers no resize callback for a window (docs/fyne-quirks.md, 14), but every
CanvasObject is told its own new size, so a widget around the list is where a
pane can learn that it has to reflow.

A widget rather than a container layout, and the difference is not cosmetic: a
layout learns the width in the middle of a layout pass, and a Refresh asked for
there is swallowed -- the rows are rebuilt and the list goes on drawing the old
ones, which is a pane that reflows everywhere except on screen.
*/
type wrapWatch struct {
	widget.BaseWidget
	pane *Pane
	// box holds the list, so the pane can put a new one in it without a new
	// renderer.
	box *fyne.Container
}

func newWrapWatch(p *Pane, list *widget.List) *wrapWatch {
	w := &wrapWatch{pane: p, box: container.NewStack(list)}
	w.ExtendBaseWidget(w)
	p.box = w.box
	return w
}

func (w *wrapWatch) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(w.box)
}

// Resize reflows the pane to the width it has just been given.
func (w *wrapWatch) Resize(size fyne.Size) {
	changed := size.Width != w.Size().Width
	w.BaseWidget.Resize(size)
	w.box.Resize(size)
	if changed {
		w.pane.resized()
	}
}

/*
resized rewraps to a width that has just changed, and repaints if it did.

Only when the column count actually moved: a drag of a few pixels usually leaves
the columns where they were, and rebuilding a list per frame of a drag is work
for nothing.

The change is noticed by rewrap rather than compared here, because the list asks
for the row count while it is being resized -- so by the time this runs, the
rewrap has often already happened and there is nothing left to compare against.
That is what made the first version of this do nothing at all.
*/
func (p *Pane) resized() {
	if p.list == nil {
		return
	}
	p.rewrap()
	if p.needsReflow {
		p.needsReflow = false
		p.reflow()
	}
}

/*
reflow puts a new list in, because an old one cannot be told the rows changed.

widget.List reuses the widget it already made for a row index, and neither
Refresh nor RefreshItem re-runs the update for one while the resize that changed
it is still settling: the rows that are new appear -- a narrower pane needs more
of them -- and the ones whose index already existed keep the text they had. What
that looks like is a wrap that half worked, with the tail of a line underneath it
and the head of it still running off the edge.

A list with no rows yet has nothing to reuse. It is virtualised, so building one
costs the visible rows and nothing else, and it happens when the column count
changes, which is a resize rather than a keystroke. The scroll position is
carried over so the reflow does not also jump the pane to the top.
*/
func (p *Pane) reflow() {
	if p.box == nil || p.list == nil {
		return
	}
	at := p.list.GetScrollOffset()
	p.box.Objects[0] = p.buildList()
	p.box.Refresh()
	p.list.ScrollToOffset(at)
}

// charWidth is one monospace character at the pane's text size, measured once.
func (p *Pane) charWidth() float32 {
	if p.charW > 0 {
		return p.charW
	}
	p.charW = fyne.MeasureText("0", p.textSize, fyne.TextStyle{Monospace: true}).Width
	return p.charW
}

// Touch records that content arrived, for the "Last updated" stamp.
func (p *Pane) Touch() { p.updated = time.Now() }

/*
Pump redraws the pane on a timer until stop is called, then once more, so the
last lines are on screen even if they arrived between ticks. The quit channel
is what makes the wait terminate: stopping a ticker does not close its
channel.
*/
func (p *Pane) Pump() (stop func()) {
	ticker := time.NewTicker(PumpInterval)
	quit := make(chan struct{})
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		defer ticker.Stop()
		for {
			select {
			case <-quit:
				return
			case <-ticker.C:
				if !p.log.TakeDirty() {
					continue
				}
				fyne.Do(func() { p.Touch(); p.Draw() })
			}
		}
	}()
	var once sync.Once
	return func() {
		once.Do(func() {
			close(quit)
			<-stopped
			fyne.Do(func() { p.Touch(); p.Draw() })
		})
	}
}
