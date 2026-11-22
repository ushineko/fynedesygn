package main

import (
	"fmt"
	"math"
	"time"

	"fyne.io/fyne/v2"
	fynetheme "fyne.io/fyne/v2/theme"

	fd "github.com/ushineko/fynedesygn"
	"github.com/ushineko/fynedesygn/glance"
	"github.com/ushineko/fynedesygn/widgets"
)

/*
The two shapes spec 086 added, each behind a flag so the archetype in
docs/img/glance-monitor.png stays the four cards it shows.

-clock puts a card at the top whose heading is a clock: the time at the left
and the date at the right, each after its icon (SetTitle, SetTitleEnd).

-tunnels puts five tunnels under the bandwidth card's interface, in a run held
to three rows (Scroller), over a plot that does not scroll with them.
*/

// tunnelCount is how many tunnels -tunnels draws, and tunnelsShown how many
// rows of them show at once.
const (
	tunnelCount  = 5
	tunnelsShown = 3
)

// extras are the cards and rows the flags add. A nil field is a flag not
// given.
type extras struct {
	clock *glance.Card

	tunnels    *glance.Scroller
	tunnelRows []*glance.Row
	tunnelPlot *glance.Sparkline
}

// buildClock makes the clock card.
func (e *extras) buildClock() *glance.Card {
	e.clock = glance.NewCard("")
	e.clock.SetIcon(titleGrey(fynetheme.HistoryIcon()))
	e.clock.SetTitleSize(fynetheme.SizeNameSubHeadingText)
	return e.clock
}

// buildTunnels adds the tunnels and their plot to the bandwidth card, under
// the rows it has.
func (e *extras) buildTunnels(c *glance.Card) {
	e.tunnels = glance.NewScroller(tunnelsShown)
	for i := range tunnelCount {
		r := glance.NewRow(fmt.Sprintf("wg%d", i), glance.NoRate())
		r.SetID(fmt.Sprintf("wg%d", i))
		e.tunnelRows = append(e.tunnelRows, r)
		e.tunnels.AddRow(r)
	}
	c.Add(e.tunnels)
	e.tunnelPlot = glance.NewSparkline(60)
	e.tunnelPlot.AddSeries("wg0", widgets.StatusColor(fd.StatusGood), 5)
	c.AddObject(e.tunnelPlot)
}

// titleGrey colours an icon as a card's title is, so the clock's icons sit
// with its time and date rather than standing out from them.
func titleGrey(r fyne.Resource) fyne.Resource {
	return fynetheme.NewColoredResource(r, fynetheme.ColorNamePlaceHolder)
}

// render draws the extras for a tick.
func (e *extras) render(now time.Time, t float64) {
	if e.clock != nil {
		e.clock.SetAvailable(true)
		e.clock.SetTitle(now.Format("15:04:05"))
		e.clock.SetTitleEnd(now.Format("Mon 2006/01/02"), titleGrey(fynetheme.CalendarIcon()))
	}
	if e.tunnels != nil {
		for i, r := range e.tunnelRows {
			r.Set(glance.Measured(glance.Rate(math.Abs(math.Sin(t/7+float64(i))) * 2 * 1024 * 1024)))
		}
		e.tunnelPlot.Add("wg0", math.Abs(math.Sin(t/7)))
	}
}
