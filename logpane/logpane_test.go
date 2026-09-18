package logpane

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/require"

	fd "github.com/ushineko/fynedesygn"
	"github.com/ushineko/fynedesygn/fynetest"
)

func TestTheModelDropsTheOldestChunkAtTheCapAndSaysSo(t *testing.T) {
	m := NewModel(200)
	for i := range 200 {
		m.Append(Info, fmt.Sprintf("line %d", i))
	}
	require.Equal(t, 200, m.Len())
	require.Zero(t, m.Dropped())
	m.Append(Warn, "one more\nand another")
	require.Equal(t, 200-DropChunk+2, m.Len())
	require.Equal(t, DropChunk, m.Dropped())
	require.Equal(t, Line{Info, "line 128"}, m.At(0), "the oldest surviving line")
	require.Equal(t, Line{}, m.At(-1))
	require.Equal(t, Line{}, m.At(10_000))
	require.True(t, strings.HasPrefix(m.Text(), "... 128 earlier line(s) were dropped; the log keeps the last 200.\n"))
	require.True(t, strings.HasSuffix(m.Text(), "one more\nand another\n"))

	require.True(t, m.TakeDirty())
	require.False(t, m.TakeDirty(), "cleared by the take")
	m.Replace("fresh\r\ncontent\n")
	require.Equal(t, 2, m.Len())
	require.Equal(t, "fresh", m.At(0).Text, "CR stripped")
	require.Zero(t, m.Dropped())
	m.Reset()
	require.Zero(t, m.Len())
	require.True(t, m.TakeDirty())
	require.Equal(t, DefaultMaxLines, NewModel(0).max)
}

// Canary #10 (docs/fyne-quirks.md): the list has no scroll callback, so the
// pane infers "the user scrolled up" from offsets, with a tolerance because
// ScrollToBottom lands on fractional values.
func TestFollowTailToleratesFractionalOffsets(t *testing.T) {
	require.True(t, FollowTail(true, 100, 100))
	require.True(t, FollowTail(true, 97.5, 100), "within tolerance still follows")
	require.False(t, FollowTail(true, 40, 100), "scrolled up: stop following")
	require.False(t, FollowTail(false, 100, 100), "off stays off until the box is ticked")
	require.True(t, FollowTail(true, 120, 100), "past the end still follows")
}

func TestLevelsNameAndRankThemselves(t *testing.T) {
	require.Equal(t, "WARN", Warn.String())
	require.Equal(t, fd.StatusBad, Error.Status())
	require.Equal(t, fd.StatusInfo, Debug.Status())
	require.Equal(t, "INFO", Level(42).String())
}

func TestWidgetRendersAndTheControlsWork(t *testing.T) {
	app := fynetest.App(t)
	p := New(nil)
	cleared, flashed := 0, ""
	o := p.Widget(Options{
		Title:     "Activity",
		Clipboard: app.Clipboard(),
		Flash:     func(text string, _ fd.Status) { flashed = text },
		OnClear:   func() { cleared++ },
	})
	w := test.NewWindow(o)
	defer w.Close()
	w.Resize(fyne.NewSize(800, 500))

	p.Log(Info, "started")
	p.Log(Error, "went wrong")
	p.Draw()
	require.Equal(t, 2, p.Model().Len())
	line := p.Model().At(0).Text
	require.Regexp(t, `^\d\d:\d\d:\d\d INFO started$`, line)
	require.Contains(t, fynetest.Text(o), "2 line(s)")

	test.Tap(fynetest.FindButton(o, "Copy"))
	require.Contains(t, app.Clipboard().Content(), "went wrong")
	require.Equal(t, "Copied 2 line(s) to the clipboard.", flashed)

	test.Tap(fynetest.FindButton(o, "Clear"))
	require.Equal(t, 1, cleared)
	require.Zero(t, p.Model().Len())

	require.True(t, p.Following())
	fynetest.FindCheck(o).SetChecked(false)
	require.False(t, p.Following())

	p.Detach()
	require.NotPanics(t, p.Draw, "drawing with no widgets is a no-op")
}

func TestWriterSplitsLinesAndHoldsPartialOnes(t *testing.T) {
	p := New(nil)
	w := p.Writer(Warn)
	_, err := fmt.Fprint(w, "one\ntwo\nthr")
	require.NoError(t, err)
	require.Equal(t, 2, p.Model().Len())
	_, err = fmt.Fprint(w, "ee\n")
	require.NoError(t, err)
	require.Equal(t, 3, p.Model().Len())
	require.Equal(t, Line{Warn, "three"}, p.Model().At(2))
}

func TestPumpDrawsOnATimerAndOnceMoreAfterStop(t *testing.T) {
	fynetest.App(t)
	p := New(nil)
	o := p.Widget(Options{Title: "Job"})
	w := test.NewWindow(o)
	defer w.Close()
	w.Resize(fyne.NewSize(800, 400))

	// Under the test driver fyne.Do runs inline on the pump's goroutine, so
	// the test must not touch the pane while the pump runs (quirk 11): it
	// waits, stops (which joins the goroutine and draws once more), then reads.
	stop := p.Pump()
	p.Model().Append(Info, "tick")
	time.Sleep(3 * PumpInterval)
	p.Model().Append(Info, "late line between ticks")
	stop()
	stop() // idempotent
	require.False(t, p.updated.IsZero(), "the pump touched the stamp")
	require.Contains(t, fynetest.Text(o), "2 line(s)", "the final draw after stop shows the late line")
}

func TestSetFollowingReArmsTheTailAndTheBoxSaysSo(t *testing.T) {
	fynetest.App(t)
	p := New(nil)
	o := p.Widget(Options{Title: "Job"})
	w := test.NewWindow(o)
	defer w.Close()
	w.Resize(fyne.NewSize(600, 300))
	fynetest.FindCheck(o).SetChecked(false)
	require.False(t, p.Following())
	p.SetFollowing(true)
	require.True(t, p.Following())
	require.True(t, fynetest.FindCheck(o).Checked)
	p.Detach()
	require.NotPanics(t, func() { p.SetFollowing(false) })
}
