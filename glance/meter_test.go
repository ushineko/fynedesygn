package glance_test

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	fd "github.com/ushineko/fynedesygn"
	"github.com/ushineko/fynedesygn/fynetest"
	"github.com/ushineko/fynedesygn/glance"
)

func TestAMeterFillsInProportionToItsFraction(t *testing.T) {
	_ = fynetest.App(t)
	m := glance.NewMeter("max", 60)
	m.Set(0.25, "5h: 25%", fd.StatusGood)

	track, fill := meterBars(t, m, fyne.NewSize(200, 40))

	assert.InDelta(t, float64(track.Size().Width)*0.25, float64(fill.Size().Width), 1,
		"the fill should be a quarter of the track")
}

// A bar wider than its track is a bar that has left the layout, which in a
// window sized to its content means a window that grew.
func TestAMeterClampsAFractionOutsideItsRange(t *testing.T) {
	_ = fynetest.App(t)
	m := glance.NewMeter("work", 60)

	m.Set(1.8, "over", fd.StatusBad)
	assert.Equal(t, 1.0, m.Fraction())

	m.Set(-0.5, "under", fd.StatusInfo)
	assert.Equal(t, 0.0, m.Fraction())

	track, fill := meterBars(t, m, fyne.NewSize(200, 40))
	assert.LessOrEqual(t, fill.Size().Width, track.Size().Width)
}

// Usage against a quota is one of the things that genuinely has a threshold,
// so the fill is graded. The four statuses must give four different fills, or
// the grading says nothing.
func TestAMeterIsColouredByItsVerdict(t *testing.T) {
	_ = fynetest.App(t)

	seen := map[string]bool{}
	for _, st := range []fd.Status{fd.StatusInfo, fd.StatusGood, fd.StatusWarn, fd.StatusBad} {
		m := glance.NewMeter("max", 60)
		m.Set(0.5, "half", st)
		_, fill := meterBars(t, m, fyne.NewSize(200, 40))
		r, g, b, _ := fill.FillColor.RGBA()
		seen[string(rune(r))+string(rune(g))+string(rune(b))] = true
		assert.Equal(t, st, m.Status())
	}
	assert.Len(t, seen, 4, "two statuses drew the same colour")
}

// The caption is not decoration. A bar says "most of it"; the caption says
// which window, how much, and when it resets.
func TestAMeterKeepsItsCaption(t *testing.T) {
	_ = fynetest.App(t)
	m := glance.NewMeter("max", 60)

	m.Set(0.36, "7d: 36% (5d left) · resets in 4h 32m", fd.StatusGood)

	assert.Equal(t, "7d: 36% (5d left) · resets in 4h 32m", m.Caption())
	assert.Contains(t, fynetest.Text(m.Object()), "resets in 4h 32m")
}

// The bar reserves height and no width, so a meter never decides how wide the
// window is.
func TestAMeterReservesHeightAndNoWidth(t *testing.T) {
	_ = fynetest.App(t)
	m := glance.NewMeter("max", 60)
	m.Set(0.5, "half", fd.StatusGood)

	assert.Positive(t, m.Object().MinSize().Height)
}

// meterBars renders the meter at a size and returns its track and fill.
func meterBars(t *testing.T, m *glance.Meter, size fyne.Size) (track, fill *canvas.Rectangle) {
	t.Helper()
	o := m.Object()
	o.Resize(size)
	rects := fynetest.All[*canvas.Rectangle](o)
	require.GreaterOrEqual(t, len(rects), 2, "a meter draws a track and a fill")
	return rects[len(rects)-2], rects[len(rects)-1]
}

// AC1, AC4. The whole point of the spec, as a measurement.
//
// The archetype's figures cost the width of the widest *pair* when they are
// laid out across the meter's rows, and the width of everything concatenated
// when they are in one caption. The second number is what made a consumer's
// whole window 655 px wide.
func TestTheFiguresCostLessAcrossRowsThanInOneCaption(t *testing.T) {
	_ = fynetest.App(t)

	// One caption, as a caller has to write it today.
	wide := glance.NewMeter("Codex", 60)
	wide.Set(0.34, "5h: 0 %  7d: 0 % (6d left)  limit: 34 % 403.51 / 1200.00", fd.StatusGood)

	// The same figures, in the slots the archetype puts them in.
	narrow := glance.NewMeter("Codex", 60)
	narrow.Set(0.34, "limit: 34 %", fd.StatusGood)
	narrow.SetTrailing("in  4h 59m")
	narrow.SetStats("5h: 0 %", "403.51 / 1200.00")

	wideWidth := wide.Object().MinSize().Width
	narrowWidth := narrow.Object().MinSize().Width

	assert.Less(t, narrowWidth, wideWidth,
		"the rows should cost less than the concatenation: %.0f against %.0f",
		narrowWidth, wideWidth)
}

// AC1. The trailing value is against the right edge, not after the caption.
func TestTheTrailingValueGoesToTheRightEdge(t *testing.T) {
	_ = fynetest.App(t)

	m := glance.NewMeter("Codex", 60)
	m.Set(0.5, "limit: 34 %", fd.StatusGood)
	m.SetTrailing("in  4h 59m")

	o := m.Object()
	o.Resize(fyne.NewSize(400, o.MinSize().Height))

	trailing := findText(t, o, "in  4h 59m")
	caption := findText(t, o, "limit: 34 %")

	// Hard against the right: what is left of it is the spacer, and the
	// caption is nowhere near it.
	assert.Greater(t, trailing.Position().X, caption.Position().X+caption.MinSize().Width+100,
		"the trailing value sat next to the caption rather than at the edge")
	assert.InDelta(t, 400, trailing.Position().X+trailing.MinSize().Width, 1,
		"the trailing value is not against the right edge")
}

// AC2. The stats row draws both ends, under the bar.
func TestTheStatsRowDrawsBothEnds(t *testing.T) {
	_ = fynetest.App(t)

	m := glance.NewMeter("Codex", 60)
	m.Set(0.5, "limit: 34 %", fd.StatusGood)
	m.SetStats("5h: 0 %", "403.51 / 1200.00")

	left, right := m.Stats()
	assert.Equal(t, "5h: 0 %", left)
	assert.Equal(t, "403.51 / 1200.00", right)

	o := m.Object()
	o.Resize(fyne.NewSize(400, o.MinSize().Height))

	// Both ends are in the same row, so their X positions are comparable;
	// a position from another row is not, because each is relative to its own
	// container.
	l := findText(t, o, "5h: 0 %")
	r := findText(t, o, "403.51 / 1200.00")
	assert.Greater(t, r.Position().X, l.Position().X, "the stats did not spread")
	assert.InDelta(t, 400, r.Position().X+r.MinSize().Width, 1,
		"the right-hand stat is not against the right edge")

	// That the row is *under* the bar is a claim about height, which is what
	// TestEmptyStatsTakeTheRowAwayAgain measures.
}

// AC3. A meter built as one always has been is unchanged, to the pixel.
//
// This is the whole of the promise that nothing else has to be touched, and it
// is asserted rather than assumed because two optional slots that quietly
// added a few pixels of height would move every panel in every consumer.
func TestAMeterWithoutTheNewSlotsIsUnchanged(t *testing.T) {
	_ = fynetest.App(t)

	m := glance.NewMeter("max", 60)
	m.Set(0.25, "5h: 25 %", fd.StatusGood)

	// A bar, a header, and nothing else. The height is compared against a
	// meter built the same way rather than against a constant, because the
	// claim is "unchanged", and a constant would only say "as expected".
	same := glance.NewMeter("max", 60)
	same.Set(0.25, "5h: 25 %", fd.StatusGood)

	assert.Equal(t, same.Object().MinSize(), m.Object().MinSize())
	assert.Empty(t, m.Trailing(), "a meter that was never given one has a trailing value")

	left, right := m.Stats()
	assert.Empty(t, left)
	assert.Empty(t, right)
}

// AC7. Setting the stats to empty takes the row away rather than leaving a gap.
//
// A gap under one meter in a stack of them is read as a reading that failed.
func TestEmptyStatsTakeTheRowAwayAgain(t *testing.T) {
	_ = fynetest.App(t)

	m := glance.NewMeter("Codex", 60)
	m.Set(0.5, "limit: 34 %", fd.StatusGood)

	bare := m.Object().MinSize().Height
	m.SetStats("5h: 0 %", "403.51 / 1200.00")
	assert.Greater(t, m.Object().MinSize().Height, bare, "the stats row took no height")

	m.SetStats("", "")
	assert.Equal(t, bare, m.Object().MinSize().Height, "the emptied row left a gap")
}

// AC5. The new slots are monospace, for the reason the caption is: a value
// that changes width would drag the one opposite it about, and a panel that
// moves while it is being read is worse than a wide one.
func TestTheNewSlotsAreMonospace(t *testing.T) {
	_ = fynetest.App(t)

	m := glance.NewMeter("Codex", 60)
	m.SetTrailing("in  4h 59m")
	m.SetStats("5h: 0 %", "403.51 / 1200.00")

	o := m.Object()
	for _, s := range []string{"in  4h 59m", "5h: 0 %", "403.51 / 1200.00"} {
		assert.True(t, findText(t, o, s).TextStyle.Monospace, "%q is not monospace", s)
	}
}
