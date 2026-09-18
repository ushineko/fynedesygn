package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ushineko/fynedesygn/fynetest"
	"github.com/ushineko/fynedesygn/shell"
	"github.com/ushineko/fynedesygn/steps"
)

func TestTheJobRunsInlineHeadlesslyAndCompletesEveryStep(t *testing.T) {
	j := newJob(0)
	s := shell.Headless(fynetest.App(t), options(j, true))
	o := s.Current().Build(s)
	fynetest.FindButton(o, "Run").OnTapped()
	for _, st := range j.steps.Steps() {
		require.Equal(t, steps.Done, st.State, st.Name)
	}
	require.Equal(t, "finished", j.outcome)
	require.Equal(t, 1, j.runs)
	require.Positive(t, j.pane.Model().Len())
	require.Equal(t, "The job finished.", s.FlashText())

	fynetest.FindButton(o, "Run and fail").OnTapped()
	require.Equal(t, "failed", j.outcome)
	require.Equal(t, steps.Failed, j.steps.Steps()[2].State)
	require.Contains(t, s.FlashText(), "2 checks failed")
}

func TestACancelledContextMarksTheRunningStepCancelled(t *testing.T) {
	j := newJob(0)
	s := shell.Headless(fynetest.App(t), options(j, true))
	_ = s.Current().Build(s)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := j.run(ctx, s, false)
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, steps.Cancelled, j.steps.Steps()[0].State)
	require.Equal(t, "cancelled", j.outcome)
	s.Report("Running", err)
	require.Empty(t, s.FlashText(), "cancellation is not a failure")
}
