package logpane

import (
	"fmt"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

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

/*
A line longer than the pane is wrapped, not lost off the right edge.

Every row is one line of monospace text in a widget.List, which is what makes a
thousand-line log scroll like a terminal: uniform heights, no layout pass per
row. The cost was that a long line was drawn past the edge and the rest of it
could not be read at all -- an auto-restore line naming fifteen cheats stopped
at the twelfth.
*/
func TestALongLineBecomesSeveralRows(t *testing.T) {
	got := wrapLine("the quick brown fox jumps over the lazy dog", 20)
	require.Equal(t, []string{
		"the quick brown fox",
		Continued + "jumps over the",
		Continued + "lazy dog",
	}, got)

	for _, row := range got {
		require.LessOrEqual(t, utf8.RuneCountInString(row), 20)
	}

	// Every row after the first says so, and nothing is lost in the breaking.
	var back []string
	for i, row := range got {
		if i > 0 {
			require.True(t, strings.HasPrefix(row, Continued), "%q is not marked", row)
			row = strings.TrimPrefix(row, Continued)
		}
		back = append(back, row)
	}
	require.Equal(t, "the quick brown fox jumps over the lazy dog", strings.Join(back, " "))
}

// A line that fits is one row, unchanged. Wrapping must not touch the ordinary
// case, which is almost every line.
func TestALineThatFitsIsLeftAlone(t *testing.T) {
	require.Equal(t, []string{"short"}, wrapLine("short", 20))
	require.Equal(t, []string{"exactly twenty chars"}, wrapLine("exactly twenty chars", 20))
	require.Equal(t, []string{"anything"}, wrapLine("anything", 0),
		"a pane with no width yet leaves lines alone")
}

/*
A word with no spaces in it is broken mid-word.

A path, a JSON blob or a cheat list joined by underscores has nowhere to break,
and losing its tail is worse than breaking it.
*/
func TestSomethingWithNoSpacesIsBrokenAnyway(t *testing.T) {
	got := wrapLine("/home/someone/.cache/terrariabonker/sprites/1.4.5.8/Item_3507.png", 20)
	require.Greater(t, len(got), 2)
	var back []string
	for i, row := range got {
		require.LessOrEqual(t, utf8.RuneCountInString(row), 20)
		if i > 0 {
			row = strings.TrimPrefix(row, Continued)
		}
		back = append(back, row)
	}
	require.Equal(t, "/home/someone/.cache/terrariabonker/sprites/1.4.5.8/Item_3507.png",
		strings.Join(back, ""))
}

/*
A break near the start of a line is not taken.

Breaking at the first space of "a verylongunbreakabletoken" would leave a row
holding one character and push the rest down, which reads worse than a broken
word.
*/
func TestAWrapDoesNotLeaveAnAlmostEmptyRow(t *testing.T) {
	got := wrapLine("a verylongwordthatcannotbebroken", 20)
	require.Equal(t, "a verylongwordthatca", got[0])
}

// Wrapping keeps each row's level, so a broken error line stays red all the way
// down.
func TestAWrappedLineKeepsItsLevel(t *testing.T) {
	m := NewModel(10)
	m.Append(Error, "an error message far longer than the pane can show in one row")
	m.Append(Info, "short")

	rows := wrapRows(m, 20)
	require.Greater(t, len(rows), 2)
	for _, row := range rows[:len(rows)-1] {
		require.Equal(t, Error, row.Level)
	}
	require.Equal(t, Info, rows[len(rows)-1].Level)
}

// What is copied is the log, not the rows: a line broken for the screen must
// arrive in one piece on the clipboard.
func TestCopyingIsUnaffectedByWrapping(t *testing.T) {
	m := NewModel(10)
	long := "a line far longer than any pane is wide, with several words in it"
	m.Append(Info, long)
	require.Greater(t, len(wrapRows(m, 20)), 1)
	require.Contains(t, m.Text(), long)
}

// The column count is measured short by a little: wrapping a column early is
// invisible, and a column late clips a character off every long line.
func TestTheColumnCountIsConservative(t *testing.T) {
	require.Equal(t, 0, columnsFor(0, 8), "a pane with no width yet does not wrap")
	require.Equal(t, 0, columnsFor(80, 8), "nor one too narrow to be worth it")
	require.Equal(t, 98, columnsFor(800, 8))
	require.Equal(t, 0, columnsFor(800, 0), "and an unmeasurable font does not either")
}

/*
The pane wraps without anything driving it.

This is the regression 0.1.16 shipped. The wrapping hung off Draw, which runs on
the pump's tick -- and most panes are never pumped: a section that rebuilds on
its own redraws the log with it, so nothing ever calls Draw and nothing ever
wrapped. The test that passed called Draw itself, which is a path the caller
does not take, so it reported a fix that did nothing.

The wrapping is in the list's length function now, which is reached however the
pane is driven.
*/
func TestThePaneWrapsWithoutDrawOrPump(t *testing.T) {
	fynetest.Sandbox(t)
	app := test.NewApp()
	t.Cleanup(app.Quit)

	m := NewModel(50)
	p := New(m)
	w := p.Widget(Options{Title: "Output", Height: 200})
	win := test.NewWindow(w)
	t.Cleanup(win.Close)
	win.Resize(fyne.NewSize(900, 260))

	m.Append(Info, "[auto-restore] cheats=[reach mining fast_place max_minions pickup "+
		"spawn_rate loot teleport vanity_accs inventory_accs smart_cursor pylons tool_reach "+
		"ore_extract] items=[2, 1, 0, 19, 39, 7, 9] pending=[] skipped=[]")
	m.Append(Info, "[auto-restore] 15 cheats applied")

	// No Draw, no Pump. Only a refresh, which is what a rebuilt section does.
	w.Refresh()

	require.Equal(t, 2, m.Len(), "two lines were logged")
	require.Greater(t, len(p.rows), 2, "and they are drawn as more rows than that")
	require.NotPanics(t, func() { _ = win.Canvas().Capture() })

	marked := 0
	for _, row := range p.rows {
		require.LessOrEqualf(t, utf8.RuneCountInString(row.Text), p.wrapCols,
			"%q is wider than the pane", row.Text)
		if strings.HasPrefix(row.Text, Continued) {
			marked++
		}
	}
	require.Positive(t, marked, "a continued row says that it is one")
}

// A pane nobody has laid out yet does not wrap, and does not lose anything by
// it: the rows are the lines until there is a width to wrap to.
func TestAPaneWithNoWidthYetKeepsItsLines(t *testing.T) {
	m := NewModel(10)
	m.Append(Info, "a line far longer than any pane that has not been laid out")
	p := New(m)
	p.Widget(Options{Title: "Output"})

	p.rewrap() // no list size yet
	require.Len(t, p.rows, 1)
}
