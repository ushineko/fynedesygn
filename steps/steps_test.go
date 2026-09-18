package steps

import (
	"testing"

	"github.com/stretchr/testify/require"

	fd "github.com/ushineko/fynedesygn"
	"github.com/ushineko/fynedesygn/fynetest"
)

func TestAdvanceFinishesEarlierStepsAndStopMarksOnlyRunningOnes(t *testing.T) {
	fynetest.App(t)
	l := New("Fetch", "Build", "Deploy")
	o := l.Widget()
	l.Advance(1, "compiling")
	got := l.Steps()
	require.Equal(t, Done, got[0].State)
	require.Equal(t, Running, got[1].State)
	require.Equal(t, "compiling", got[1].Note)
	require.Equal(t, Pending, got[2].State)
	text := fynetest.Text(o)
	require.Contains(t, text, "Build")
	require.Contains(t, text, "compiling")

	l.Stop(Cancelled, "stopped by the user")
	got = l.Steps()
	require.Equal(t, Done, got[0].State, "finished steps stay finished")
	require.Equal(t, Cancelled, got[1].State)
	require.Equal(t, Pending, got[2].State, "never started, so not cancelled")
	require.Equal(t, fd.StatusWarn, Cancelled.Status())
	require.Equal(t, "cancelled", Cancelled.String())

	l.Finish(2, "skipped")
	require.Equal(t, Done, l.Steps()[2].State)
	l.Reset()
	for _, s := range l.Steps() {
		require.Equal(t, Pending, s.State)
		require.Empty(t, s.Note)
	}
	l.Set(99, Done, "ignored")
}

func TestSetAfterDetachKeepsStateForTheNextWidget(t *testing.T) {
	fynetest.App(t)
	l := New("One")
	_ = l.Widget()
	l.Detach()
	require.NotPanics(t, func() { l.Set(0, Failed, "boom") })
	text := fynetest.Text(l.Widget())
	require.Contains(t, text, "boom")
	require.Equal(t, fd.StatusBad, l.Steps()[0].State.Status())
	require.NotNil(t, Running.Icon())
}

func TestAdvanceNeverReopensAFinishedStepAndResetKeepsStandingNotes(t *testing.T) {
	fynetest.App(t)
	l := NewSteps(Step{Name: "Read", Note: "the game install and the mod library"}, Step{Name: "Merge"}, Step{Name: "Compile"})
	require.Equal(t, "the game install and the mod library", l.Steps()[0].Note)
	require.Equal(t, Pending, l.Steps()[0].State)
	l.Advance(1, "merging")
	require.Equal(t, Done, l.Steps()[0].State)
	l.Advance(0, "one more file, late") // a stale message for a finished step
	require.Equal(t, Done, l.Steps()[0].State, "never backwards")
	require.Equal(t, Running, l.Steps()[1].State)
	l.Stop(Failed, "boom")
	l.Advance(1, "again")
	require.Equal(t, Failed, l.Steps()[1].State, "a failed step stays failed")
	l.Reset()
	require.Equal(t, "the game install and the mod library", l.Steps()[0].Note, "standing notes survive Reset")
	require.Empty(t, l.Steps()[1].Note)
	for _, s := range l.Steps() {
		require.Equal(t, Pending, s.State)
	}
}
