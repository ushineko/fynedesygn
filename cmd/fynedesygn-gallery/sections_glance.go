package main

import (
	"math"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	fynetheme "fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	fd "github.com/ushineko/fynedesygn"
	"github.com/ushineko/fynedesygn/glance"
	"github.com/ushineko/fynedesygn/shell"
	"github.com/ushineko/fynedesygn/widgets"
)

/*
The Glance section.

A glance window cannot be shown inside the gallery: it is frameless, always on
top, and sized to its own content, and a window like that drawn inside another
window is a picture of neither. What the gallery can show is the vocabulary it
is assembled from — the cards, the rows, the value formatting and the plot —
drawn at the size and in the scheme a program would get.

So this section is the widget styles, each captioned with its Go name, in the
same shape as the Widgets section. The archetype itself is in
docs/img/glance-monitor.png, captured from examples/glance-monitor as it
actually sits on a desktop.
*/

// glanceUnit is the width reserved for a temperature's unit, so the rows in
// the demonstration cards line up with each other the way a real card's do.
const glanceUnit = 2

// panelWidth is the width every card here is drawn at: a little over
// glance.MinWidth, which is what a real glance window settles at with two
// device cells. Drawing them at the gallery's full width would show a shape no
// program produces.
const panelWidth float32 = 320

// buildGlance draws the glance vocabulary. Every card here is forced available,
// because in a program availability is what a poll reports and there is no poll
// in a gallery.
func buildGlance(_ *shell.Shell) fyne.CanvasObject {
	// Everything is drawn at panelWidth rather than filling the content pane.
	// A glance card lives in a window a few hundred pixels wide; stretched
	// across a gallery pane it is a shape no program will ever see, and the
	// value column ends up under the scrollbar. Left-aligned in an HBox so the
	// card keeps its own width instead of centring in a pane it does not use.
	panel := func(o fyne.CanvasObject) fyne.CanvasObject {
		return container.NewHBox(widgets.FixedWidth(o, panelWidth), layout.NewSpacer())
	}
	demo := func(name string, o fyne.CanvasObject) fyne.CanvasObject {
		return container.NewVBox(widgets.Dim("glance."+name), panel(o), widget.NewSeparator())
	}
	// The formatters are a table of values rather than a card, so they are not
	// held to the panel width.
	wide := func(name string, o fyne.CanvasObject) fyne.CanvasObject {
		return container.NewVBox(widgets.Dim("glance."+name), o, widget.NewSeparator())
	}

	return container.NewVScroll(container.NewVBox(
		widgets.Heading("Glance",
			"The second window archetype: a frameless, always-on-top panel sized to its content and read "+
				"without being interacted with. The window cannot be drawn inside this one, so these are the "+
				"pieces it is built from. The rules are in docs/glance.md."),

		demo("Card, live", liveCard().Object()),
		// The cell card sits beside the row card it is the alternative to, and
		// inside the section's first screen, which is what the README shows.
		demo("Card + CellGrid", cellCard().Object()),
		demo("Card, source gone", staleCard().Object()),
		demo("Card + Sparkline", plotCard().Object()),
		demo("Card, InsertRow over a plot", insertedCard().Object()),
		demo("Sparkline, ScaleShared, SeriesColour + Faded", sharedPlot(90)),

		demo("Card + Meter", meterCard().Object()),

		demo("Row", container.NewVBox(
			glanceRow("Coolant", glance.Known(glance.Quantity(46.6, "°C", glanceUnit), fd.StatusGood)),
			glanceRow("CPU", glance.Measured(glance.Quantity(78.4, "°C", glanceUnit))),
			glanceRow("Pump", glance.Known(glance.Count(0, glance.NumberWidth, "rpm"), fd.StatusBad)),
			glanceRow("Sensor", glance.Missing(glance.NoQuantity("°C", glanceUnit))),
		)),

		demo("Row, Parted + NewRowWidth", partsCard().Object()),

		// The formatters are the opinionated half of the package and the place
		// the no-reflow rule is enforced, so they are shown as a column of
		// values rather than described. Every line is the same width.
		wide("Rate / Size / Percent / Quantity / Count", container.NewVBox(
			glanceValue("Rate", glance.Rate(900), glance.Rate(3686), glance.Rate(3.9*(1<<30))),
			glanceValue("Size", glance.Size(1536), glance.Size(43.5*(1<<30)), glance.NoSize()),
			glanceValue("Percent", glance.Percent(7), glance.Percent(87), glance.NoPercent()),
			glanceValue("Quantity", glance.Quantity(9.9, "°C", glanceUnit),
				glance.Quantity(46.6, "°C", glanceUnit), glance.NoQuantity("°C", glanceUnit)),
			glanceValue("Count", glance.Count(0, glance.NumberWidth, "rpm"),
				glance.Count(1298, glance.NumberWidth, "rpm"), glance.NoCount(glance.NumberWidth, "rpm")),
			widgets.DimWrapped("Each row is one formatter at three magnitudes, including its unknown value. "+
				"Every value a formatter can produce is the same width, because the window is sized to its "+
				"content and a number that widens as it crosses a magnitude widens the window."),
		)),

		demo("Sparkline", glancePlot(90)),
	))
}

// glanceRow is one row, drawn at its natural width.
func glanceRow(label string, r glance.Reading) fyne.CanvasObject {
	row := glance.NewRow(label, r.Text)
	row.Set(r)
	return row.Object()
}

// glanceValue is one formatter's name and three of its outputs, in the
// monospace face so the columns are visible as columns.
func glanceValue(name string, values ...string) fyne.CanvasObject {
	row := container.NewHBox(widgets.FixedWidth(widgets.Dim(name), 90))
	for _, v := range values {
		l := widget.NewLabelWithStyle(v, fyne.TextAlignLeading, fyne.TextStyle{Monospace: true})
		row.Add(l)
	}
	return row
}

// liveCard is a card whose source is answering.
func liveCard() *glance.Card {
	c := glance.NewCard("Peripherals")
	mouse := glance.NewRow("G502 X PLUS", glance.NoPercent())
	phones := glance.NewRow("Headphones", glance.NoPercent())
	c.AddRow(mouse, phones)
	c.SetAvailable(true)
	mouse.Set(glance.Known(glance.Percent(87), fd.StatusGood))
	phones.Set(glance.Missing(glance.NoPercent()))
	return c
}

/*
cellCard is the other shape a reading takes: a block rather than a line.

A battery is the case. In a row the percentage is a small number at the right
margin in the same weight as the device name, so the eye reads the name and
then hunts for the figure; in a cell the figure is what it lands on and the
name is only which device it belongs to. The card above this one is the same
peripherals section drawn as rows, which is the comparison worth making.

Three of the cells carry a bar under their state, a length the eye compares
across the card. The keyboard has no reading and so no bar, and its cell is the
same height as the others: the bar's row is reserved whether or not one is
drawn in it.
*/
func cellCard() *glance.Card {
	c := glance.NewCard("Peripherals")
	grid := glance.NewCellGrid()

	mouse := glance.NewCell("G502 X PLUS", glance.NoPercent())
	mouse.Set(glance.Known(glance.Percent(67), fd.StatusGood))
	mouse.SetNote("Discharging")
	mouse.SetBar(0.67, fd.StatusGood)

	board := glance.NewCell("Keychron K4 HE", glance.NoPercent())
	board.Set(glance.Missing(glance.NoPercent()))
	board.SetNote("No reading")

	buds := glance.NewCell("AirPods Pro", glance.NoPercent())
	buds.Set(glance.Known(glance.Percent(80), fd.StatusGood))
	buds.SetNote("L 80  R 90")
	buds.SetBar(0.80, fd.StatusGood)

	phones := glance.NewCell("Arctis Nova Pro", glance.NoPercent())
	phones.Set(glance.Known(glance.Percent(15), fd.StatusBad))
	phones.SetNote("Discharging")
	phones.SetBar(0.15, fd.StatusBad)

	grid.Add(mouse, board, buds, phones)
	c.Add(grid)
	c.SetAvailable(true)
	return c
}

// staleCard is the same card after its source stopped answering: the last
// values are kept and dimmed, and the header is marked. Nothing is blanked,
// because the reader's question is whether the values are still true.
func staleCard() *glance.Card {
	c := glance.NewCard("Bandwidth")
	down := glance.NewRow("eno2 down", glance.NoRate())
	up := glance.NewRow("eno2 up", glance.NoRate())
	c.AddRow(down, up)
	c.SetAvailable(true)
	down.Set(glance.Measured(glance.Rate(65.3 * 1024)))
	up.Set(glance.Measured(glance.Rate(65.1 * 1024)))
	c.SetStale(true)
	return c
}

// plotCard is the shape the archetype is really for: graded rows over a plot
// whose traces are drawn in the colours of the rows they came from.
func plotCard() *glance.Card {
	c := glance.NewCard("AIO")
	cpu := glance.NewRow("CPU", glance.NoQuantity("°C", glanceUnit))
	coolant := glance.NewRow("Coolant", glance.NoQuantity("°C", glanceUnit))
	fans := glance.NewRow("Fans", glance.NoCount(glance.NumberWidth, "rpm"))
	c.AddRow(cpu, coolant, fans)
	c.AddObject(glancePlot(60))
	c.SetAvailable(true)

	// CPU is not colour graded: a high boost temperature is normal, and its
	// colour identifies its trace rather than ranking the value.
	cpu.Set(glance.Measured(glance.Quantity(53.0, "°C", glanceUnit)).
		Tinted(fynetheme.Color(fynetheme.ColorNamePrimary)))
	coolant.Set(glance.Known(glance.Quantity(38.8, "°C", glanceUnit), fd.StatusGood))
	fans.Set(glance.Measured(glance.Count(1298, glance.NumberWidth, "rpm")))
	return c
}

/*
partsCard is a value in coloured parts under labels in a pinned column.

Each rate is graded by its own size, so the quiet download stays the
foreground while the busy upload beside it is amber and bold; coloured as one
value, the download would be amber too. The labels are pinned to one width,
so the long hardware name is clipped (truncated with an ellipsis here, as a
caller would) rather than widening the card.
*/
func partsCard() *glance.Card {
	c := glance.NewCard("Bandwidth")
	blank := glance.NoRate() + " " + glance.NoRate()
	eth := glance.NewRowWidth("eno2", blank, glancePartsLabel)
	vpn := glance.NewRowWidth("veth4a1b2c9…", blank, glancePartsLabel)
	c.AddRow(eth, vpn)
	c.SetAvailable(true)
	eth.Set(glance.Parted(
		glance.Part{Text: glance.Rate(2048)},
		glance.Part{Text: " "},
		glance.Part{Text: glance.Rate(40 * (1 << 20)), Status: fd.StatusWarn, Bold: true},
	))
	vpn.Set(glance.Parted(
		glance.Part{Text: glance.Rate(900)},
		glance.Part{Text: " "},
		glance.Part{Text: glance.Rate(3686)},
	))
	return c
}

// glancePartsLabel pins the label column of the parted rows.
const glancePartsLabel float32 = 96

// meterCard is the quota shape: a proportion of something with a limit, drawn
// as a label, a caption carrying the detail, and a bar graded by how close to
// the limit it is. A reading with no limit is a Row — drawing a temperature as
// a bar would invent a maximum.
func meterCard() *glance.Card {
	c := glance.NewCard("Usage")
	for _, m := range []struct {
		label   string
		frac    float64
		caption string
		st      fd.Status
	}{
		{"5h", 0.04, glance.Percent(4) + " · resets in 4h 32m", fd.StatusGood},
		{"7d", 0.36, glance.Percent(36) + " · 5 days left", fd.StatusGood},
		{"month", 0.82, "$ 823.52 / $1000 · 1 Oct", fd.StatusWarn},
		{"burst", 0.96, glance.Percent(96) + " · burst allowance", fd.StatusBad},
	} {
		meter := glance.NewMeter(m.label, glanceMeterLabel)
		meter.Set(m.frac, m.caption, m.st)
		c.Add(meter)
	}
	c.SetAvailable(true)
	return c
}

// glanceMeterLabel pins the label column so stacked meters line their captions
// up with each other.
const glanceMeterLabel float32 = 52

// glancePlot is a two-trace plot with enough samples to show a shape. The
// traces are deliberately on ranges an order of magnitude apart, which is what
// the independent scaling is for: on a shared axis the coolant trace would be
// a flat line two pixels tall.
func glancePlot(n int) *glance.Sparkline {
	p := glance.NewSparkline(n)
	p.AddSeries("coolant", widgets.StatusColor(fd.StatusGood), 5)
	p.AddSeries("cpu", fynetheme.Color(fynetheme.ColorNamePrimary), 5)

	for i := range n {
		t := float64(i)
		p.Add("coolant", 38.5+1.2*math.Sin(t/11))
		p.Add("cpu", 62+14*math.Sin(t/7)+4*math.Sin(t/3))
	}
	return p
}

// sharedPlot is a bandwidth plot: two interfaces, each a pair of traces for
// down and up, drawn under ScaleShared so their heights compare. An interface's
// down is its SeriesColour (the scheme's link colour for the first, the
// categorical violet for the second) and its up the Faded version of it. The wired link
// moves tens of MiB/s and the wireless one about one, so the wireless pair is
// a low line along the floor, which is what the shared scale is for.
func sharedPlot(n int) *glance.Sparkline {
	th := fynetheme.Current()
	p := glance.NewSparkline(n)
	p.SetScale(glance.ScaleShared)
	for i, iface := range []string{"eno1", "wlan0"} {
		c := glance.SeriesColour(th, i)
		p.AddSeries(iface+" down", c, 64*1024)
		p.AddSeries(iface+" up", glance.Faded(c), 64*1024)
	}

	const mib = 1 << 20
	for i := range n {
		t := float64(i)
		p.Add("eno1 down", 18*mib*(0.55+0.45*math.Sin(t/9)*math.Sin(t/4)))
		p.Add("eno1 up", 4*mib*(0.5+0.4*math.Sin(t/6)))
		p.Add("wlan0 down", 1.2*mib*(0.6+0.35*math.Sin(t/5)))
		p.Add("wlan0 up", 0.3*mib*(0.6+0.3*math.Sin(t/7)))
	}
	return p
}

// insertedCard is a card whose rows follow what is read: built with one row and
// a plot, then given a second graphics card and a reason by InsertRow. Both land
// among the rows and above the plot, where an appended row would have gone under
// it (spec 056).
func insertedCard() *glance.Card {
	c := glance.NewCard("Processors")
	cpu := glance.NewRow("CPU", glance.NoQuantity("°C", glanceUnit))
	cpu.SetID("cpu")
	c.AddRow(cpu)
	c.AddObject(glancePlot(60))
	c.SetAvailable(true)

	gpu := glance.NewRow("GPU 1", glance.NoQuantity("°C", glanceUnit))
	gpu.SetID("gpu:1")
	reason := glance.NewRow("Coolant", "no cooler")
	reason.SetID("reason:coolant")
	// The gallery builds its own cards, so a refusal here is a fault in the
	// gallery and not something a reader could cause.
	if err := c.InsertRow(1, gpu); err != nil {
		panic(err)
	}
	if err := c.InsertRow(len(c.Rows()), reason); err != nil {
		panic(err)
	}

	cpu.Set(glance.Measured(glance.Quantity(53.0, "°C", glanceUnit)))
	gpu.Set(glance.Measured(glance.Quantity(41.5, "°C", glanceUnit)))
	return c
}
