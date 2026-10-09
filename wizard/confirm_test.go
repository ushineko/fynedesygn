package wizard

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ushineko/fynedesygn/fynetest"
)

func confirmOptions(job func(context.Context) error) ConfirmOptions {
	return ConfirmOptions{
		Name: "Uninstall Hello", Question: "Uninstall Hello 0.1.0?", Detail: "It removes the program.",
		Action: "Uninstall", Destructive: true, Job: job, Done: "Hello 0.1.0 was removed.",
	}
}

func TestAskThenRunThenReport(t *testing.T) {
	ran := 0
	c := HeadlessConfirm(fynetest.App(t), confirmOptions(func(context.Context) error { ran++; return nil }))
	width := c.buttons.MinSize().Width
	require.Equal(t, "Uninstall", c.action.Text)
	require.Equal(t, 0, ran, "nothing runs before the question is answered")

	c.Act()
	require.Equal(t, 1, ran)
	require.Equal(t, "Hello 0.1.0 was removed.", c.Message())
	require.Equal(t, LabelClose, c.action.Text)
	require.True(t, c.cancel.Disabled())
	require.Equal(t, width, c.buttons.MinSize().Width, "the row keeps its width")

	c.Act()
	require.Equal(t, Finished, c.Result().Outcome)
}

func TestCancelWhileAskingRunsNothing(t *testing.T) {
	ran := false
	c := HeadlessConfirm(fynetest.App(t), confirmOptions(func(context.Context) error { ran = true; return nil }))
	c.Cancel()
	require.Equal(t, Cancelled, c.Result().Outcome)
	require.False(t, ran)
}

func TestAFailingJobShowsItsErrorAndClosesAsFailed(t *testing.T) {
	c := HeadlessConfirm(fynetest.App(t), confirmOptions(func(context.Context) error { return errors.New("permission denied") }))
	c.Act()
	require.Equal(t, "Failed: permission denied", c.Message())
	c.Cancel() // the title bar's close, on the result, is Close
	require.Equal(t, Failed, c.Result().Outcome)
	require.EqualError(t, c.Result().Err, "permission denied")
}

func TestCancelWhileRunningAsksAndWaitsForTheJob(t *testing.T) {
	var c *ConfirmWindow
	asked, sawDone := 0, false
	c = HeadlessConfirm(fynetest.App(t), confirmOptions(func(ctx context.Context) error {
		c.Cancel()
		require.Equal(t, 1, asked)
		require.False(t, c.closed, "the window waits for the job")
		<-ctx.Done()
		sawDone = true
		return fmt.Errorf("removing: %w", ctx.Err())
	}))
	c.confirm = func(_, _, _ string, do func()) { asked++; do() }
	c.Act()
	require.True(t, sawDone)
	require.Equal(t, Cancelled, c.Result().Outcome)
}

func TestAJobThatFinishesAsCancelArrivesStillFinishes(t *testing.T) {
	var c *ConfirmWindow
	c = HeadlessConfirm(fynetest.App(t), confirmOptions(func(context.Context) error {
		c.Cancel()
		return nil // the work was already done
	}))
	c.Act()
	require.Equal(t, Finished, c.Result().Outcome, "work that happened is not reported as cancelled")
}

func TestSkipQuestionShowsOnlyTheResult(t *testing.T) {
	o := confirmOptions(func(context.Context) error { return nil })
	o.SkipQuestion = true
	c := HeadlessConfirm(fynetest.App(t), o)
	require.Equal(t, "Hello 0.1.0 was removed.", c.Message(), "the job ran without the question")
	require.Equal(t, LabelClose, c.action.Text)
}

func TestNewConfirmInLaysOutARealWindow(t *testing.T) {
	c := NewConfirmIn(fynetest.App(t), confirmOptions(func(context.Context) error { return nil }))
	t.Cleanup(c.Window.Close)
	require.Equal(t, c.Content(), c.Window.Content())
	require.Positive(t, c.Window.Content().MinSize().Width)
}
