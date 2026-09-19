/*
Command glance-monitor is a fynedesygn example: an always-on-top status panel
in the shape of a peripheral and sensor monitor, with a bandwidth card, a
thermal card carrying a two-trace sparkline, and a context menu that is the
only interface it has.

The readings are generated, not measured. The point of the example is the
shape: how cards appear and disappear, how a source that stops answering dims
rather than blanks, how values are formatted so the window never resizes
because a number changed magnitude, and how the KDE window rule is offered
rather than installed.

Run it with -kwin to print the rule it would install, and -kwin-install to
write it.
*/
package main

import (
	"flag"
	"fmt"
	"image/color"
	"math"
	"math/rand/v2"
	"os"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	fynetheme "fyne.io/fyne/v2/theme"

	fd "github.com/ushineko/fynedesygn"
	"github.com/ushineko/fynedesygn/glance"
	"github.com/ushineko/fynedesygn/glance/kwin"
	fdtheme "github.com/ushineko/fynedesygn/theme"
	"github.com/ushineko/fynedesygn/widgets"
)

// appID is the window's identity. The same string is the Wayland app_id Fyne
// derives from it, the desktop file's basename, and what the KWin rule matches
// on, so all three stay in step.
const appID = "io.ushineko.fynedesygn.glance-monitor"

// pollInterval is the cadence of the generated readings. The real thing would
// give each card its own: a byte counter at 2 s and a thermal probe at 5 s
// share nothing but a window.
const pollInterval = 500 * time.Millisecond

// The coolant bands. They are the hardware's numbers rather than round ones:
// on the machine these came from, the pump-head over-temperature alarm tripped
// at 57.1 °C and cleared near 50.
const (
	coolantWarm = 50.0
	coolantHot  = 55.0
)

func main() {
	show := flag.Bool("kwin", false, "print the KDE window rule this program would install")
	install := flag.Bool("kwin-install", false, "install the KDE window rule")
	remove := flag.Bool("kwin-remove", false, "remove the KDE window rule")
	flag.Parse()

	if *show || *install || *remove {
		if err := manageRule(*install, *remove); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	a := app.NewWithID(appID)
	a.Settings().SetTheme(fdtheme.New(fdtheme.BreezeDark, fdtheme.Options{}))

	m := newMonitor()
	w := m.build(a)
	go m.poll()
	w.ShowAndRun()
}

// rule is the window rule this program offers. Opacity is here and not in the
// window because Fyne cannot draw a translucent window at all: the compositor
// is the only thing that can (quirk 32).
func rule() kwin.Rule {
	return kwin.Rule{
		AppID:       appID,
		Description: "fynedesygn glance monitor",
		AlwaysOnTop: true,
		NoBorder:    true,
		Opacity:     95,
	}
}

// manageRule is the command-line half: the rule is written when the user asks
// for it and never on a plain run.
func manageRule(install, remove bool) error {
	path, err := kwin.Path()
	if err != nil {
		return err
	}

	switch {
	case remove:
		removed, err := kwin.Remove(appID)
		if err != nil {
			return err
		}
		if !removed {
			fmt.Println("No rule for " + appID + " in " + path)
			return nil
		}
		fmt.Println("Removed the rule from " + path)
	case install:
		if err := kwin.Install(rule()); err != nil {
			return err
		}
		fmt.Println("Wrote the rule to " + path)
	default:
		r := rule()
		fmt.Printf("Would write to %s:\n\n", path)
		fmt.Printf("  wmclass=%s (exact)\n  above=true (forced)\n  noborder=true (forced)\n"+
			"  opacityactive=%d, opacityinactive=%d (applied initially)\n\n",
			r.AppID, r.Opacity, r.Opacity)
		fmt.Println("Run with -kwin-install to write it.")
		return nil
	}

	c := kwin.ReconfigureCall()
	fmt.Printf("KWin re-reads its rules when told: %s %s %s.%s\n"+
		"Logging out and back in has the same effect.\n",
		c.Destination, c.Path, c.Interface, c.Method)
	return nil
}

// monitor is the program's state: the cards and the rows they own.
type monitor struct {
	win *glance.Window

	// Bandwidth.
	bandwidth *glance.Card
	down, up  *glance.Row

	// Thermals.
	thermal      *glance.Card
	cpu, coolant *glance.Row
	fans         *glance.Row
	plot         *glance.Sparkline
	cpuHistory   []float64

	// Battery, which starts absent to show a card arriving.
	battery *glance.Card
	mouse   *glance.Row

	ticks int
}

// unitColumn is the width reserved for a temperature's unit, so the CPU and
// coolant rows line up with each other.
const unitColumn = 2

func newMonitor() *monitor { return &monitor{} }

// build assembles the window. Cards are added before it is shown, so the first
// thing drawn is the real size rather than an empty frame that grows.
func (m *monitor) build(a fyne.App) *glance.Window {
	m.win = glance.NewWindow(a, glance.Options{
		Title:    "Monitor",
		MinWidth: 300,
		OnTop:    true,
		Menu:     m.menu,
	})

	m.bandwidth = glance.NewCard("Bandwidth")
	m.down = glance.NewRow("eno2 down", glance.NoRate())
	m.up = glance.NewRow("eno2 up", glance.NoRate())
	m.bandwidth.AddRow(m.down, m.up)

	m.thermal = glance.NewCard("AIO")
	m.cpu = glance.NewRow("CPU", glance.NoQuantity("°C", unitColumn))
	m.coolant = glance.NewRow("Coolant", glance.NoQuantity("°C", unitColumn))
	m.fans = glance.NewRow("Fans", glance.NoCount(glance.NumberWidth, "rpm"))
	m.thermal.AddRow(m.cpu, m.coolant, m.fans)

	// 60 samples at the poll interval. Each trace is drawn in the colour of
	// its own row's value, which is the plot's only legend.
	m.plot = glance.NewSparkline(60)
	m.plot.AddSeries("coolant", widgets.StatusColor(fd.StatusGood), 5)
	m.plot.AddSeries("cpu", cpuTraceColour(), 5)
	m.thermal.AddObject(m.plot)

	m.battery = glance.NewCard("Peripherals")
	m.mouse = glance.NewRow("G502 X PLUS", glance.NoPercent())
	m.battery.AddRow(m.mouse)

	m.win.Panel().Add(m.battery, m.bandwidth, m.thermal)
	return m.win
}

// menu is the whole interface. It is rebuilt on every secondary tap so it can
// tick the current value of what it shows.
func (m *monitor) menu() *fyne.Menu {
	cards := []struct {
		name string
		card *glance.Card
	}{
		{"Peripherals", m.battery},
		{"Bandwidth", m.bandwidth},
		{"AIO", m.thermal},
	}

	var items []*fyne.MenuItem
	for _, c := range cards {
		item := fyne.NewMenuItem(c.name, nil)
		item.Checked = c.card.Allowed()
		item.Action = func() { c.card.SetAllowed(!c.card.Allowed()) }
		items = append(items, item)
	}

	show := fyne.NewMenuItem("Show", nil)
	show.ChildMenu = fyne.NewMenu("", items...)

	return fyne.NewMenu("",
		show,
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Quit", func() { m.win.Window().Close() }),
	)
}

// poll is the fake reader. It runs off the UI thread and hops back with
// fyne.Do, never fyne.DoAndWait, as everything in this module does.
func (m *monitor) poll() {
	for range time.Tick(pollInterval) {
		s := m.sample()
		fyne.Do(func() { m.render(s) })
	}
}

// sample is one reading of everything, as a plain value. The real thing reads
// each source in its own card; generating them together keeps the example
// short without changing the shape.
type sample struct {
	down, up   float64
	cpu        float64
	coolant    float64
	fans       int64
	mouse      float64
	aioPresent bool
	aioStale   bool
	batteryYet bool
}

func (m *monitor) sample() sample {
	m.ticks++
	t := float64(m.ticks)

	s := sample{
		// A range wide enough to cross magnitudes, which is what the fixed
		// width formatting is for.
		down: math.Abs(math.Sin(t/9)) * 8 * 1024 * 1024,
		up:   math.Abs(math.Cos(t/13)) * 900 * 1024,

		// CPU spikes the way a real one does on any compile.
		cpu:     62 + 30*math.Abs(math.Sin(t/5)) + rand.Float64()*6, // nolint:gosec // not cryptographic
		coolant: 46 + 6*math.Sin(t/40),
		fans:    1200 + int64(200*math.Sin(t/30)),
		mouse:   87,

		// The AIO source appears after a moment and drops out for a stretch,
		// so the example shows a card arriving and a card going stale.
		aioPresent: m.ticks > 4,
		aioStale:   m.ticks%80 > 60,

		// The battery card arrives later still.
		batteryYet: m.ticks > 12,
	}
	return s
}

// render draws a sample. It runs on the UI thread.
func (m *monitor) render(s sample) {
	m.bandwidth.SetAvailable(true)
	m.down.Set(glance.Measured(glance.Rate(s.down)))
	m.up.Set(glance.Measured(glance.Rate(s.up)))

	m.battery.SetAvailable(s.batteryYet)
	if s.batteryYet {
		m.mouse.Set(glance.Known(glance.Percent(s.mouse), batteryStatus(s.mouse)))
	}

	m.thermal.SetAvailable(s.aioPresent)
	if !s.aioPresent {
		return
	}
	// A source that was answering and has stopped keeps its last values,
	// dimmed. Nothing is blanked and nothing moves.
	m.thermal.SetStale(s.aioStale)
	if s.aioStale {
		return
	}

	st := coolantStatus(s.coolant)
	m.coolant.Set(glance.Known(glance.Quantity(s.coolant, "°C", unitColumn), st))
	m.fans.Set(glance.Measured(glance.Count(s.fans, glance.NumberWidth, "rpm")))

	// CPU is plotted as a trailing mean and is deliberately not colour graded:
	// a high boost temperature is normal, and grading it would paint every
	// compile red. Its colour identifies its trace, not a severity.
	mean := m.cpuMean(s.cpu)
	m.cpu.Set(glance.Measured(glance.Quantity(s.cpu, "°C", unitColumn)).Tinted(cpuTraceColour()))

	m.plot.SetColour("coolant", widgets.StatusColor(st))
	m.plot.Add("coolant", s.coolant)
	m.plot.Add("cpu", mean)
}

// cpuMeanWindow is the trailing mean the CPU trace is plotted as. Raw CPU is
// unreadable at 26 px: it spikes to 100 °C on any compile. Averaged, it is a
// trend line, and the interesting part is that it leads the coolant.
const cpuMeanWindow = 12

func (m *monitor) cpuMean(v float64) float64 {
	m.cpuHistory = append(m.cpuHistory, v)
	if len(m.cpuHistory) > cpuMeanWindow {
		m.cpuHistory = m.cpuHistory[len(m.cpuHistory)-cpuMeanWindow:]
	}
	sum := 0.0
	for _, x := range m.cpuHistory {
		sum += x
	}
	return sum / float64(len(m.cpuHistory))
}

// coolantStatus grades a coolant temperature. Only values with a threshold are
// graded; see the note on CPU in render.
func coolantStatus(c float64) fd.Status {
	switch {
	case c >= coolantHot:
		return fd.StatusBad
	case c >= coolantWarm:
		return fd.StatusWarn
	default:
		return fd.StatusGood
	}
}

// batteryStatus grades a battery percentage.
func batteryStatus(pct float64) fd.Status {
	switch {
	case pct <= 10:
		return fd.StatusBad
	case pct <= 25:
		return fd.StatusWarn
	default:
		return fd.StatusGood
	}
}

// cpuTraceColour is the colour the CPU trace and its row share. It is not a
// status colour, because the value it draws has no status: the plot carries no
// legend, so the row's colour is what says which trace is which.
//
// It comes from the scheme rather than a literal, so it stays legible when the
// scheme changes — the same rule every colour in this module follows.
func cpuTraceColour() color.Color {
	return fynetheme.Color(fynetheme.ColorNamePrimary)
}
