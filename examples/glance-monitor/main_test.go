package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	fd "github.com/ushineko/fynedesygn"
	"github.com/ushineko/fynedesygn/fynetest"
	"github.com/ushineko/fynedesygn/glance"
)

// build assembles the monitor on the test driver. glance.NewWindow falls back
// to an ordinary window when the driver is not a desktop one, so the whole
// panel is exercised with no display.
func build(t *testing.T) *monitor {
	t.Helper()
	m := newMonitor()
	m.build(fynetest.App(t))
	return m
}

// render drives n polls through the render path, as the poll goroutine would.
func (m *monitor) renderTicks(n int) {
	for range n {
		m.render(m.sample())
	}
}

// The window starts with nothing on it. A card is drawn once its source
// answers, so the window does not flash an empty panel on the way up.
func TestTheWindowStartsWithNoCardsDrawn(t *testing.T) {
	m := build(t)

	assert.Equal(t, 0, m.win.Panel().Drawn())
}

func TestEachCardArrivesWhenItsSourceAnswers(t *testing.T) {
	m := build(t)

	m.renderTicks(1)
	assert.True(t, m.bandwidth.Drawn(), "the counter answers on the first poll")
	assert.False(t, m.thermal.Drawn(), "the thermal source has not answered yet")
	assert.False(t, m.battery.Drawn())

	m.renderTicks(6)
	assert.True(t, m.thermal.Drawn())

	m.renderTicks(10)
	assert.True(t, m.battery.Drawn())
	assert.True(t, m.usage.Drawn(), "the usage card arrives with the battery card")
	assert.Equal(t, 4, m.win.Panel().Drawn())
}

// The window grows as cards arrive and shrinks when one is hidden. Shrinking
// is the half Fyne does not do on its own (quirk 34).
func TestTheWindowGrowsWithItsCardsAndShrinksAgain(t *testing.T) {
	m := build(t)

	m.renderTicks(1)
	one := m.win.Panel().Size()

	m.renderTicks(15)
	all := m.win.Panel().Size()
	require.Greater(t, all.Height, one.Height, "cards arriving should have grown the window")

	m.thermal.SetAllowed(false)
	fewer := m.win.Panel().Size()
	assert.Less(t, fewer.Height, all.Height,
		"the window kept the height of a card it is no longer drawing")
}

// The rule the whole package exists for, asserted end to end: 500 polls of
// rates that cross several magnitudes, and the window never changes size.
func TestTheWindowNeverResizesBecauseAValueChanged(t *testing.T) {
	m := build(t)
	m.renderTicks(20) // let every card arrive

	settled := m.win.Panel().Size()
	for range 500 {
		m.render(m.sample())
		require.Equal(t, settled, m.win.Panel().Size(),
			"the panel resized on a reading; at tick %d", m.ticks)
	}
}

// A source that stops answering dims its card and keeps the last values. The
// reader's question is whether they are still true.
func TestASourceThatStopsAnsweringDimsRatherThanBlanks(t *testing.T) {
	m := build(t)
	m.renderTicks(20)

	// Render until the generated source drops out, keeping the last value it
	// reported while it was still answering.
	var lastLive string
	for !m.thermal.Stale() {
		lastLive = m.coolant.Reading().Text
		m.render(m.sample())
	}
	require.Contains(t, lastLive, "°C")
	before := lastLive

	require.True(t, m.thermal.Stale(), "the card should be showing last-known values")
	assert.True(t, m.thermal.Drawn(), "a source that has gone leaves the card up")
	assert.Equal(t, before, m.coolant.Reading().Text, "a stale card must not blank its reading")
	assert.True(t, m.coolant.Reading().Stale)
}

func TestRecoveryClearsTheStaleMarker(t *testing.T) {
	m := build(t)
	m.renderTicks(20)
	for m.ticks%80 <= 60 {
		m.render(m.sample())
	}
	require.True(t, m.thermal.Stale())

	for m.thermal.Stale() {
		m.render(m.sample())
	}

	assert.False(t, m.thermal.Stale())
}

// Only values with a threshold are graded. A high boost temperature is normal
// and grading it would paint every compile red.
func TestTheCPUReadingIsNotColourGraded(t *testing.T) {
	m := build(t)
	m.renderTicks(30)

	assert.Equal(t, fd.StatusInfo, m.cpu.Reading().Status,
		"CPU temperature has no threshold, so it has no verdict")
	assert.NotNil(t, m.cpu.Reading().Colour,
		"its colour identifies its sparkline trace, which is the plot's only legend")
}

func TestTheCoolantReadingIsGradedAgainstItsBands(t *testing.T) {
	assert.Equal(t, fd.StatusGood, coolantStatus(46.6))
	assert.Equal(t, fd.StatusWarn, coolantStatus(coolantWarm))
	assert.Equal(t, fd.StatusBad, coolantStatus(coolantHot))
}

// The plot leaves the row values' colours alone but takes its own from them,
// which is the mapping that replaces a legend.
func TestTheCoolantTraceTracksTheCoolantBand(t *testing.T) {
	m := build(t)
	m.renderTicks(40)

	assert.NotEmpty(t, m.plot.Samples("coolant"))
	assert.NotEmpty(t, m.plot.Samples("cpu"))
	assert.True(t, m.plot.HasData())
}

// The CPU trace is a trailing mean. Raw CPU spikes to 100 °C on any compile
// and is unreadable at 26 px; averaged it is a trend line.
func TestTheCPUTraceIsSmootherThanTheCPUReading(t *testing.T) {
	m := build(t)
	m.renderTicks(120)

	assert.Less(t, spread(m.plot.Samples("cpu")), spread(rawCPU(m)),
		"the plotted trace should move less than the raw reading it came from")
}

// rawCPU replays the raw CPU readings the samples would have produced.
func rawCPU(m *monitor) []float64 {
	probe := newMonitor()
	var out []float64
	for range m.ticks {
		out = append(out, probe.sample().cpu)
	}
	return out
}

func spread(vs []float64) float64 {
	if len(vs) == 0 {
		return 0
	}
	lo, hi := vs[0], vs[0]
	for _, v := range vs {
		if v < lo {
			lo = v
		}
		if v > hi {
			hi = v
		}
	}
	return hi - lo
}

// The menu is the whole interface, and it is rebuilt on every tap so it can
// tick what is currently shown.
func TestTheMenuTicksTheCardsThatAreAllowed(t *testing.T) {
	m := build(t)
	m.renderTicks(20)

	show := m.menu().Items[0]
	require.Equal(t, "Show", show.Label)
	require.Len(t, show.ChildMenu.Items, 4)
	for _, item := range show.ChildMenu.Items {
		assert.True(t, item.Checked, "%s should start ticked", item.Label)
	}

	// Untick the AIO card through the menu, as a user would.
	for _, item := range show.ChildMenu.Items {
		if item.Label == "AIO" {
			item.Action()
		}
	}

	assert.False(t, m.thermal.Drawn())
	rebuilt := m.menu().Items[0].ChildMenu.Items
	for _, item := range rebuilt {
		if item.Label == "AIO" {
			assert.False(t, item.Checked, "the menu should show the current state")
		}
	}
}

// Every value on screen is fixed width. This is the assertion that would catch
// a formatter being replaced with fmt.Sprintf.
func TestEveryValueOnScreenIsAFixedWidthOne(t *testing.T) {
	m := build(t)
	m.renderTicks(20)

	widths := map[*glance.Row]int{}
	for _, r := range []*glance.Row{m.down, m.up, m.cpu, m.coolant, m.fans, m.mouse} {
		widths[r] = glance.Width(r.Reading().Text)
	}

	for range 300 {
		m.render(m.sample())
		for r, w := range widths {
			assert.Equal(t, w, glance.Width(r.Reading().Text),
				"a value changed width: %q", r.Reading().Text)
		}
	}
}

// The KWin rule is printed on request and written only when asked for. A run
// with no flags must not touch the user's compositor configuration.
func TestTheWindowRuleIsOfferedAndNotInstalled(t *testing.T) {
	r := rule()

	assert.Equal(t, appID, r.AppID, "the rule matches the app ID Fyne gives the window")
	assert.True(t, r.AlwaysOnTop)
	assert.True(t, r.NoBorder)
	assert.Positive(t, r.Opacity, "opacity is the compositor's to grant; Fyne cannot draw it")
	assert.LessOrEqual(t, r.Opacity, 100)
	assert.NotEmpty(t, r.Description, "a rule the user cannot identify is one they cannot remove")
}

func TestTheAppIDIsUsableAsADesktopFileBasename(t *testing.T) {
	// Wayland matches a window to its desktop entry by app_id, so the two must
	// be the same string (docs/design-system.md, Platform).
	assert.NotContains(t, appID, " ")
	assert.NotContains(t, appID, "/")
	assert.True(t, strings.HasPrefix(appID, "io.ushineko."))
}

// A quota has a limit, so it is drawn as a meter rather than a row. The bands
// are a real threshold: the closer to the limit, the more it matters.
func TestTheQuotaMetersAreGradedByHowCloseToTheLimitTheyAre(t *testing.T) {
	assert.Equal(t, fd.StatusGood, quotaStatus(10))
	assert.Equal(t, fd.StatusWarn, quotaStatus(75))
	assert.Equal(t, fd.StatusBad, quotaStatus(90))
}

func TestTheUsageMetersCarryACaptionAndAFractionInRange(t *testing.T) {
	m := build(t)
	m.renderTicks(20)

	for _, meter := range []*glance.Meter{m.session, m.spend} {
		assert.NotEmpty(t, meter.Caption(), "a bar without a caption says only 'most of it'")
		assert.GreaterOrEqual(t, meter.Fraction(), 0.0)
		assert.LessOrEqual(t, meter.Fraction(), 1.0)
	}
	assert.Contains(t, m.spend.Caption(), "823.52")
}

// The meters are in a card, so their arrival and departure resize the window
// the same way any other card's does, and neither changes size as it fills.
func TestFillingAMeterDoesNotResizeTheWindow(t *testing.T) {
	m := build(t)
	m.renderTicks(20)
	settled := m.win.Panel().Size()

	for range 200 {
		m.render(m.sample())
		require.Equal(t, settled, m.win.Panel().Size(),
			"the panel resized while a meter filled; at tick %d", m.ticks)
	}
}
