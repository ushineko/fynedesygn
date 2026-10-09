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

// withThen adds a step that asks to remove what the first job left; step
// is its job.
func withThen(o ConfirmOptions, ask bool, step func(context.Context) error) ConfirmOptions {
	o.Then = &ConfirmStep{
		Action: "Remove them too", Destructive: true, Decline: "Keep them",
		Ask: func() (string, string, bool) {
			return "Hello left 2 files it made.", "- `cache.db`\n- `state.json`", ask
		},
		Job:  step,
		Done: "Hello 0.1.0 was removed, with the files it made.",
	}
	return o
}

func TestTheStepIsAskedAfterTheJobWithWhatItFound(t *testing.T) {
	steps := 0
	c := HeadlessConfirm(fynetest.App(t), withThen(confirmOptions(func(context.Context) error { return nil }), true,
		func(context.Context) error { steps++; return nil }))
	width := c.buttons.MinSize().Width
	c.Act()
	require.Equal(t, "Hello left 2 files it made.", c.Question())
	require.Equal(t, "Hello 0.1.0 was removed.", c.Message(), "the first job's result stays on the message line")
	require.Equal(t, "Remove them too", c.action.Text)
	require.Equal(t, "Keep them", c.cancel.Text)
	require.False(t, c.cancel.Disabled())
	require.Equal(t, width, c.buttons.MinSize().Width, "the row keeps its width")
	require.Equal(t, 0, steps)

	c.Act()
	require.Equal(t, 1, steps)
	require.Equal(t, "Hello 0.1.0 was removed, with the files it made.", c.Message())
	require.Equal(t, LabelClose, c.action.Text)
	require.True(t, c.cancel.Disabled())
	require.Equal(t, width, c.buttons.MinSize().Width)
	c.Act()
	require.Equal(t, Finished, c.Result().Outcome)
}

func TestDecliningTheStepFinishesWithoutRunningIt(t *testing.T) {
	ran := false
	c := HeadlessConfirm(fynetest.App(t), withThen(confirmOptions(func(context.Context) error { return nil }), true,
		func(context.Context) error { ran = true; return nil }))
	c.Act()
	c.Cancel()
	require.Equal(t, Finished, c.Result().Outcome, "the first job's work is done")
	require.False(t, ran)
}

func TestAFailingStepClosesAsFailed(t *testing.T) {
	c := HeadlessConfirm(fynetest.App(t), withThen(confirmOptions(func(context.Context) error { return nil }), true,
		func(context.Context) error { return errors.New("read-only") }))
	c.Act()
	c.Act()
	require.Equal(t, "Failed: read-only", c.Message())
	c.Act()
	require.Equal(t, Failed, c.Result().Outcome)
	require.EqualError(t, c.Result().Err, "read-only")
}

func TestACancelledStepAsksWaitsAndFinishes(t *testing.T) {
	var c *ConfirmWindow
	asked := 0
	c = HeadlessConfirm(fynetest.App(t), withThen(confirmOptions(func(context.Context) error { return nil }), true,
		func(ctx context.Context) error {
			c.Cancel()
			require.False(t, c.closed, "the window waits for the step's job")
			<-ctx.Done()
			return fmt.Errorf("removing: %w", ctx.Err())
		}))
	c.confirm = func(_, _, _ string, do func()) { asked++; do() }
	c.Act()
	c.Act()
	require.Equal(t, 1, asked)
	require.Equal(t, Finished, c.Result().Outcome, "the first job's work stands")
}

func TestAStepThatAsksNothingShowsThePlainResult(t *testing.T) {
	ran := false
	c := HeadlessConfirm(fynetest.App(t), withThen(confirmOptions(func(context.Context) error { return nil }), false,
		func(context.Context) error { ran = true; return nil }))
	c.Act()
	require.Equal(t, "Uninstall Hello 0.1.0?", c.Question())
	require.Equal(t, LabelClose, c.action.Text)
	c.Act()
	require.Equal(t, Finished, c.Result().Outcome)
	require.False(t, ran)
}

func TestAFailingFirstJobNeverAsksTheStep(t *testing.T) {
	c := HeadlessConfirm(fynetest.App(t), withThen(confirmOptions(func(context.Context) error { return errors.New("busy") }), true,
		func(context.Context) error { return nil }))
	c.Act()
	require.Equal(t, "Uninstall Hello 0.1.0?", c.Question())
	require.Equal(t, "Failed: busy", c.Message())
}

func TestSkipQuestionStillAsksTheStep(t *testing.T) {
	o := withThen(confirmOptions(func(context.Context) error { return nil }), true, func(context.Context) error { return nil })
	o.SkipQuestion = true
	c := HeadlessConfirm(fynetest.App(t), o)
	require.Equal(t, "Hello left 2 files it made.", c.Question())
	require.Equal(t, "Remove them too", c.action.Text)
}
