/*
Package logpane is a fixed-height list of monospace lines that scrolls itself
to the end unless the reader has scrolled up to read something.

Ported from nmsbonker's buildlog.go through clockwork-orange's logpane.go
(same author, MIT). Two rules shape it. The log is written from worker
goroutines and read on the UI thread, so the model is behind a mutex and the
widget is redrawn on a timer rather than per line: a hundred fyne.Do calls a
second would make the window slower than the job. And the pane never grows:
it is a fixed-height list inside a section that keeps its shape, because
nothing transient may reflow the interface.

Rows are canvas.Text rather than labels so a console text size applies; a
Label draws at the theme's one size.
*/
package logpane

import (
	"strconv"
	"strings"
	"sync"

	fd "github.com/ushineko/fynedesygn"
)

// Level ranks a line. Programs map their own levels onto it.
type Level int

// The levels, least to most severe.
const (
	Debug Level = iota
	Info
	Warn
	Error
)

// String is the upper-case name shown in a timestamped line.
func (l Level) String() string {
	switch l {
	case Debug:
		return "DEBUG"
	case Info:
		return "INFO"
	case Warn:
		return "WARN"
	case Error:
		return "ERROR"
	}
	return "INFO"
}

// Status is how a line of this level is painted.
func (l Level) Status() fd.Status {
	switch l {
	case Warn:
		return fd.StatusWarn
	case Error:
		return fd.StatusBad
	case Debug, Info:
		return fd.StatusInfo
	}
	return fd.StatusInfo
}

// DefaultMaxLines is how much output a model retains unless told otherwise.
const DefaultMaxLines = 1000

// DropChunk is how many of the oldest lines go at once when the cap is hit,
// so the drop is not a per-line shuffle.
const DropChunk = 128

// Line is one row and how to paint it.
type Line struct {
	Level Level
	Text  string
}

// Model holds the lines. Safe to append to from any goroutine.
type Model struct {
	mu      sync.Mutex
	max     int
	lines   []Line
	dropped int
	dirty   bool
}

// NewModel makes a model keeping the last maxLines; 0 means DefaultMaxLines.
func NewModel(maxLines int) *Model {
	if maxLines <= 0 {
		maxLines = DefaultMaxLines
	}
	return &Model{max: maxLines}
}

// Append adds a message, one row per line of it, dropping the oldest chunk
// when the cap is reached.
func (m *Model) Append(level Level, text string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, line := range strings.Split(strings.TrimRight(text, "\n"), "\n") {
		if len(m.lines) >= m.max {
			drop := min(DropChunk, len(m.lines))
			m.lines = append(m.lines[:0], m.lines[drop:]...)
			m.dropped += drop
		}
		m.lines = append(m.lines, Line{Level: level, Text: strings.TrimRight(line, "\r")})
	}
	m.dirty = true
}

// Replace swaps the whole content, for a re-read of a file tail.
func (m *Model) Replace(text string) {
	m.mu.Lock()
	m.lines, m.dropped = nil, 0
	m.mu.Unlock()
	m.Append(Info, text)
}

// Reset empties the log.
func (m *Model) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lines, m.dropped, m.dirty = nil, 0, true
}

// Len is the number of lines held.
func (m *Model) Len() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.lines)
}

// Dropped is how many older lines have been discarded at the cap.
func (m *Model) Dropped() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.dropped
}

// At returns one line, or the zero line for an index that has since been
// dropped: a list's item count and its update callback are read at different
// moments, so an out-of-range index is expected rather than a bug.
func (m *Model) At(i int) Line {
	m.mu.Lock()
	defer m.mu.Unlock()
	if i < 0 || i >= len(m.lines) {
		return Line{}
	}
	return m.lines[i]
}

// TakeDirty reports whether anything changed since the last call, and clears
// the flag. A quiet job costs the pump one comparison per tick.
func (m *Model) TakeDirty() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	was := m.dirty
	m.dirty = false
	return was
}

// Text renders the whole log for the clipboard, with a line saying how many
// earlier lines were dropped when any were.
func (m *Model) Text() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	var b strings.Builder
	if m.dropped > 0 {
		b.WriteString("... ")
		b.WriteString(strconv.Itoa(m.dropped))
		b.WriteString(" earlier line(s) were dropped; the log keeps the last ")
		b.WriteString(strconv.Itoa(m.max))
		b.WriteString(".\n")
	}
	for _, l := range m.lines {
		b.WriteString(l.Text)
		b.WriteString("\n")
	}
	return b.String()
}

/*
FollowTail decides whether the pane should keep scrolling to the end.

Fyne's list has no "the user scrolled" callback, so this compares the offset
the last auto-scroll left behind with the offset the list is at now. Lower
means the user dragged the pane up to read something, and a pane that yanks
itself back to the bottom half a second later is unusable. Following resumes
when they scroll back down, or when they tick the box.

The tolerance is there because ScrollToBottom lands on a fractional offset and
the list re-measures as rows arrive; without it the pane stops following
itself.
*/
func FollowTail(following bool, current, wanted float32) bool {
	const tolerance = 4
	if !following {
		return false
	}
	return current >= wanted-tolerance
}
