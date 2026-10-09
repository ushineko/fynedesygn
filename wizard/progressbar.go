package wizard

import (
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/ushineko/fynedesygn/widgets"
)

// ProgressInterval is how often a progress page with a window draws the
// latest values a job reported (R3).
const ProgressInterval = 100 * time.Millisecond

// ItemRunes is the longest item line, in runes, before it is shortened in
// the middle (R4).
const ItemRunes = 72

// counter is the bar region of a progress page: the latest values, stored
// by the job and drawn by the UI.
type counter struct {
	bar    *widget.ProgressBar
	status *widget.Label
	item   *widget.Label

	mu       sync.Mutex
	fraction float64
	text     [2]string
	changed  bool
	// draws counts the times the values reached the widgets, for a test.
	draws int
}

func newCounter() *counter {
	c := &counter{bar: widget.NewProgressBar(), status: widget.NewLabel(""), item: widget.NewLabel("")}
	c.bar.TextFormatter = func() string { return "" }
	c.item.Importance = widget.LowImportance
	c.item.Truncation = fyne.TextTruncateEllipsis
	c.status.Truncation = fyne.TextTruncateEllipsis
	return c
}

// widget is the region. Its labels hold a space until the first report, so
// the region has the height it will keep.
func (c *counter) widget() fyne.CanvasObject {
	c.status.SetText(" ")
	c.item.SetText(" ")
	box := container.NewVBox(c.bar, c.status, c.item)
	return widgets.FixedHeight(box, box.MinSize().Height)
}

func (c *counter) set(fraction float64, status, item string) {
	c.mu.Lock()
	c.fraction = min(max(fraction, 0), 1)
	c.text = [2]string{status, middle(item, ItemRunes)}
	c.changed = true
	c.mu.Unlock()
}

// draw puts the latest values on the widgets, when they changed. It runs
// on the UI thread.
func (c *counter) draw() {
	c.mu.Lock()
	if !c.changed {
		c.mu.Unlock()
		return
	}
	f, t := c.fraction, c.text
	c.changed = false
	c.draws++
	c.mu.Unlock()
	c.bar.SetValue(f)
	c.status.SetText(orSpace(t[0]))
	c.item.SetText(orSpace(t[1]))
}

// pump draws on a timer until stop is called, and draws once more then.
func (c *counter) pump(do func(func())) (stop func()) {
	done := make(chan struct{})
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		t := time.NewTicker(ProgressInterval)
		defer t.Stop()
		for {
			select {
			case <-done:
				return
			case <-t.C:
				do(c.draw)
			}
		}
	}()
	return func() {
		close(done)
		<-finished
		do(c.draw)
	}
}

func orSpace(s string) string {
	if s == "" {
		return " "
	}
	return s
}

// middle shortens s to at most n runes by replacing its middle with an
// ellipsis, so a path keeps its start and its file name.
func middle(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	keep := n - 1
	head := keep / 2
	tail := keep - head
	return string(r[:head]) + "…" + string(r[len(r)-tail:])
}
