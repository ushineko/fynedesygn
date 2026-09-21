package markdown

import (
	"image/color"
	"io/fs"
	"path"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	fynetheme "fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ushineko/fynedesygn/mermaid"
	fdtheme "github.com/ushineko/fynedesygn/theme"
)

/*
Overscan is how much of a viewport's worth of document is kept rendered above
and below what is on screen, so that a fast scroll never reaches a block
before the block is rendered.

Half a screen of runway either side is several wheel notches, and it is what
the cost is paid for: with a 44-block README in a 900 x 700 pane, a frame
costs about a third less than the same document as one RichText. A larger
margin renders more of the document than anyone is looking at; a smaller one
saves little more and starts to matter when the scrollbar is thrown.
*/
const Overscan = 0.5

// NotRenderedCaption is shown over a mermaid fence that has no image yet.
const NotRenderedCaption = "Diagram not rendered; run make generate"

// Options tell a Pane where a document's images and diagrams live.
type Options struct {
	// FS resolves relative image paths; nil leaves them unresolved.
	FS fs.FS
	// Dir is the directory within FS the document was read from, so its
	// relative paths resolve beside it. Empty means the FS root.
	Dir string
	// Diagrams are the pre-rendered mermaid images; nil means mermaid fences
	// show their source.
	Diagrams *mermaid.Set
	/*
		SettleResize coalesces the re-measure that a width change forces, and
		zero -- the default -- measures on every change as this pane always has.

		Fyne hands a widget a Resize for every step of a drag, from inside the
		event poll, and measuring this document means rendering every block to
		ask its height. In clockwork-orange's About section that was 32% of all
		CPU during a drag, and the event queue cannot drain while it runs, so
		the window moved in bursts.

		Set it to something like 120ms in a program with a real window. It is
		opt-in because it needs a driver to hop back to -- the work runs on a
		timer and returns to the UI thread with fyne.Do, which is a real thread
		hop in a real program and an inline call under the test driver. A
		headless caller leaves it at zero and keeps today's behaviour, the way
		logpane's redraw timer is started by the program rather than by itself.
	*/
	SettleResize time.Duration
}

// Pane is a Markdown document rendered per block, only near the viewport.
type Pane struct {
	widget.BaseWidget

	opts Options
	// blocks is the document split into paragraph-sized pieces, in order.
	blocks []string
	// vis[i] is block i rendered; built by the measuring pass and kept, so a
	// block that scrolls back into view is not parsed again.
	vis []fyne.CanvasObject
	// spacers[i] holds block i's height whether or not its text is in the
	// tree; cells[i] is what the document's column actually contains.
	spacers []*canvas.Rectangle
	cells   []*fyne.Container
	// live[i] says block i's text is currently in the tree.
	live []bool

	// body is the column of cells, and the widget's only child.
	body *fyne.Container
	// view is the scroll this pane is inside, watched for movement. Nil means
	// nobody is scrolling it, and then every block is rendered: a pane with no
	// viewport to be outside of is a plain document.
	view *container.Scroll
	// width is the width the blocks were last measured at.
	width float32
	// pending is the width a settled resize will measure at, and settle is the
	// timer that will do it. Guarded by mu, which is held only around these
	// two: everything else here is the UI thread's.
	mu      sync.Mutex
	pending float32
	settle  *time.Timer
}

/*
DragFraction is how much the width may change and still be treated as a drag.

Below it the change is waited out; at or above it the document is measured at
once. A quarter is comfortably more than a drag's step and comfortably less
than the jump from one section's width to another's. Only consulted when
Options.SettleResize is set.
*/
var DragFraction float32 = 0.25

/*
New renders src, which is Markdown.

The pane is not scrollable itself. It is meant to go inside a section's
scroller and be handed that scroller with Follow, so that one scrollbar
governs the document and whatever the section puts above it.
*/
func New(src string, o Options) *Pane {
	p := &Pane{opts: o, blocks: Blocks(src)}
	n := len(p.blocks)
	p.vis = make([]fyne.CanvasObject, n)
	p.spacers = make([]*canvas.Rectangle, n)
	p.cells = make([]*fyne.Container, n)
	p.live = make([]bool, n)

	cells := make([]fyne.CanvasObject, n)
	for i := range p.blocks {
		p.spacers[i] = canvas.NewRectangle(color.Transparent)
		p.cells[i] = container.NewStack(p.spacers[i])
		cells[i] = p.cells[i]
	}
	p.body = container.NewVBox(cells...)
	p.ExtendBaseWidget(p)
	return p
}

/*
Follow watches sc for movement, so the pane can render the blocks the user is
looking at. Call Detach before the section holding the pane is replaced: the
scroll outlives it.

A nil scroll is not an error: a section built with no window (which is how
headless tests build every section) has no viewport, and a pane with no
viewport renders the whole document.
*/
func (p *Pane) Follow(sc *container.Scroll) {
	if sc == nil {
		p.sync()
		return
	}
	p.view = sc
	sc.OnScrolled = func(fyne.Position) { p.sync() }
	p.sync()
}

// Detach gives the scroll back: its callback would otherwise keep calling
// into a pane that is no longer on screen.
func (p *Pane) Detach() {
	if p.view != nil {
		p.view.OnScrolled = nil
		p.view = nil
	}
	// A pending measure would otherwise fire into a pane the section has
	// already replaced, and hop to the UI thread to do it.
	p.cancelPending()
}

// cancelPending drops a scheduled measure, whether because it has been
// answered or because there is no longer a pane to answer for.
func (p *Pane) cancelPending() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.settle != nil {
		p.settle.Stop()
		p.settle = nil
	}
	p.pending = 0
}

// Blocks is how many blocks the document split into.
func (p *Pane) Blocks() int { return len(p.blocks) }

// Live is how many blocks currently have their rendering in the tree.
func (p *Pane) Live() int {
	n := 0
	for _, l := range p.live {
		if l {
			n++
		}
	}
	return n
}

// IsLive reports whether block i is currently rendered.
func (p *Pane) IsLive(i int) bool { return i >= 0 && i < len(p.live) && p.live[i] }

// Visual is block i rendered, built on first use. Tests use it to inspect
// what a block became.
func (p *Pane) Visual(i int) fyne.CanvasObject {
	if p.vis[i] == nil {
		p.vis[i] = RenderBlock(p.blocks[i], p.opts)
	}
	return p.vis[i]
}

// CreateRenderer implements fyne.Widget.
func (p *Pane) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(p.body)
}

/*
Resize measures the blocks again when the width changes (a narrower window
wraps the same paragraph into more lines) and then renders what is in view.

With Options.SettleResize set, a drag is waited out rather than measured at
every step; a jump is always measured at once. See Options.SettleResize.
*/
func (p *Pane) Resize(s fyne.Size) {
	p.BaseWidget.Resize(s)
	if p.opts.SettleResize > 0 && p.isDrag(s.Width) {
		p.remeasureWhenSettled(s.Width)
	} else {
		// Measuring now answers whatever was waiting, so nothing should fire
		// afterwards and re-measure at a width that has been overtaken.
		p.cancelPending()
		p.measure(s.Width)
	}
	p.sync()
}

/*
isDrag distinguishes a width worth waiting on from one worth measuring now.

A drag moves the edge a few pixels at a time, hundreds of times. A section
being shown, a window being maximised or a pane being laid out for the first
time arrives as one large jump, and delaying that would show the document with
the wrong heights for as long as it took to settle -- visible, for no gain,
because it happens once.
*/
func (p *Pane) isDrag(w float32) bool {
	if p.width <= 0 {
		return false // never measured: there is nothing to show yet
	}
	d := w - p.width
	if d < 0 {
		d = -d
	}
	return d < p.width*DragFraction
}

/*
remeasureWhenSettled arranges for a measure once w has stopped changing.

The timer is restarted on every call, so a drag schedules the work repeatedly
and performs it once. The callback hops to the UI thread: it runs on the
timer's goroutine, and everything it touches belongs to the interface.
*/
func (p *Pane) remeasureWhenSettled(w float32) {
	if w <= 0 || w == p.width {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.pending = w
	if p.settle != nil {
		p.settle.Stop()
	}
	p.settle = time.AfterFunc(p.opts.SettleResize, func() {
		p.mu.Lock()
		w := p.pending
		p.mu.Unlock()
		fyne.Do(func() {
			p.measure(w)
			p.sync()
		})
	})
}

/*
Settle measures now rather than waiting, and reports whether there was
anything to measure.

For a test, and for a caller that knows the resizing has stopped.
*/
func (p *Pane) Settle() bool {
	p.mu.Lock()
	w := p.pending
	if p.settle != nil {
		p.settle.Stop()
		p.settle = nil
	}
	p.mu.Unlock()
	if w <= 0 || w == p.width {
		return false
	}
	p.measure(w)
	p.sync()
	return true
}

/*
measure sizes every block's spacer to the height that block needs at width w.

It is the one place the document's shape is decided. Blocks are measured, not
estimated: an estimate that came out short would clip the last line of a
paragraph, and one that came out long would leave a gap, and both would be
visible only on the machines whose font differs from the one that was guessed
against.
*/
func (p *Pane) measure(w float32) {
	if w <= 0 || w == p.width {
		return
	}
	p.width = w
	for i := range p.blocks {
		o := p.Visual(i)
		/*
			Width first, then height. A block's height is not a property of
			the block: Fyne's RichText reports unwrapped text until it has
			been resized, and a mermaid diagram derives its height from the
			width it was given.

			Spec 015 dropped the second MinSize as "asking the same question
			again after a Resize that cannot change the answer". It can, and
			the answer was a document measured several screens short of what
			it draws as -- see spec 021.
		*/
		o.Resize(fyne.NewSize(w, o.MinSize().Height))
		h := o.MinSize().Height
		p.spacers[i].SetMinSize(fyne.NewSize(0, h))
	}
	p.body.Refresh()
}

/*
sync puts the blocks near the viewport in the tree and takes the rest out.

"Near" is Overscan viewports above and below what is on screen, so a fling
does not outrun the rendering: by the time a block is on screen it has been in
the tree since it was half a screen away.
*/
func (p *Pane) sync() {
	if p.view == nil {
		for i := range p.blocks {
			p.render(i, true)
		}
		return
	}
	// The pane's own coordinates: where the viewport is over this widget, not
	// over the scroll's content, which also holds whatever is above the pane.
	viewH := p.view.Size().Height
	over := viewH * Overscan
	top := p.view.Offset.Y - p.Position().Y - over
	bottom := p.view.Offset.Y - p.Position().Y + viewH + over

	var y float32
	for i := range p.blocks {
		h := p.spacers[i].MinSize().Height
		p.render(i, y+h >= top && y <= bottom)
		y += h + fynetheme.Padding()
	}
}

// render puts block i's rendering in the tree, or takes it out. The spacer
// stays in both cases: the block keeps its height and the document its shape.
func (p *Pane) render(i int, want bool) {
	if p.live[i] == want {
		return
	}
	p.live[i] = want
	if want {
		p.cells[i].Objects = []fyne.CanvasObject{p.spacers[i], p.Visual(i)}
	} else {
		p.cells[i].Objects = []fyne.CanvasObject{p.spacers[i]}
	}
	p.cells[i].Refresh()
}

/*
RenderBlock renders one block: a mermaid fence as its pre-rendered diagram,
other code through CodePanel, a whole-block image from the document's
filesystem, and prose through Fyne's Markdown widget.

Code is not left to the Markdown widget because since Fyne 2.8 it draws a code
block as a label inside a horizontal scroll. A scroll takes the wheel from
whatever is under the pointer and does not pass it on, and when the code is
wider than the pane and the block is not taller than it, the scroller turns a
vertical wheel notch into a horizontal one. A README is mostly commands, so
the section stopped dead wherever the pointer happened to rest on one.
*/
func RenderBlock(src string, o Options) fyne.CanvasObject {
	if lang, code, ok := CodeBlock(src); ok {
		if lang == "mermaid" {
			return renderDiagram(code, o)
		}
		return NewCodePanel(code)
	}
	if alt, p, ok := ImageBlock(src); ok && isRelative(p) {
		return renderImage(alt, p, o)
	}
	if rows, ok := TableBlock(src); ok {
		return renderTable(rows)
	}
	rt := widget.NewRichTextFromMarkdown(src)
	rt.Wrapping = fyne.TextWrapWord
	return rt
}

// renderTable draws a pipe table as a grid of cells with a bold header row,
// each cell rendered as inline Markdown so code spans and emphasis survive.
// Fyne's own table segment lives in a scroller, which would take the wheel.
// renderDiagram looks the diagram up for the active palette's darkness, and
// falls back to the source under a caption when there is no image.
func renderDiagram(code string, o Options) fyne.CanvasObject {
	if res, ok := o.Diagrams.Lookup(code, activeDark()); ok {
		return mermaid.NewDiagram(res, mermaid.Scale)
	}
	caption := widget.NewLabel(NotRenderedCaption)
	caption.Importance = widget.LowImportance
	return container.NewVBox(caption, NewCodePanel(code))
}

// renderImage reads a relative image from the document's filesystem. A
// missing file shows the alt text, dimmed, so the reader knows what was meant.
func renderImage(alt, rel string, o Options) fyne.CanvasObject {
	if o.FS != nil {
		if b, err := fs.ReadFile(o.FS, path.Join(o.Dir, rel)); err == nil {
			return mermaid.NewDiagram(fyne.NewStaticResource(path.Base(rel), b), 1)
		}
	}
	l := widget.NewLabel("[" + alt + "]")
	l.Importance = widget.LowImportance
	l.Wrapping = fyne.TextWrapWord
	return l
}

// activeDark reports whether the running app's theme is a dark palette. A
// theme that is not this module's answers through Fyne's variant.
func activeDark() bool {
	a := fyne.CurrentApp()
	if a == nil {
		return true
	}
	if th, ok := a.Settings().Theme().(fdtheme.Theme); ok {
		return th.Palette().Dark
	}
	return a.Settings().ThemeVariant() == fynetheme.VariantDark
}

// MeasuredWidth is the width the blocks' heights were last measured at, which
// is not always the width the pane has: see SettleDelay.
func (p *Pane) MeasuredWidth() float32 { return p.width }
