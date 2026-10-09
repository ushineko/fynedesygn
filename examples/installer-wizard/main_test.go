package main

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ushineko/fynedesygn/fynetest"
	"github.com/ushineko/fynedesygn/wizard"
)

func TestTheWholeWayThrough(t *testing.T) {
	w := wizard.Headless(fynetest.App(t), options(0, false))
	w.Next() // welcome
	fynetest.FindCheck(w.Content()).SetChecked(true)
	w.Next() // licence
	w.Next() // directory
	w.Next() // options
	require.Equal(t, "Ready", w.Current().Title())
	w.Next() // summary: Install, and the job runs inline
	require.Equal(t, "Installing", w.Current().Title())
	require.Equal(t, "Example is installed.", w.Message())
	w.Next()
	fynetest.FindCheck(w.Content()).SetChecked(true)
	w.Next()
	require.Equal(t, wizard.Finished, w.Result().Outcome)
	require.True(t, w.Result().Checks[launchNow])
}

func TestAFailureIsReportedAndClosing(t *testing.T) {
	w := wizard.Headless(fynetest.App(t), options(0, true))
	w.Next()
	fynetest.FindCheck(w.Content()).SetChecked(true)
	for range 4 {
		w.Next()
	}
	require.Equal(t, "Failed: the launcher directory is read-only", w.Message())
	w.Cancel()
	require.Equal(t, wizard.Failed, w.Result().Outcome)
}

func TestARelativeDirectoryIsRefused(t *testing.T) {
	w := wizard.Headless(fynetest.App(t), options(0, false))
	w.Next()
	fynetest.FindCheck(w.Content()).SetChecked(true)
	w.Next()
	fynetest.FindEntry(w.Content()).SetText("relative/path")
	w.Next()
	require.Equal(t, "Location", w.Current().Title(), "Next is disabled")
}

func TestTheConfirmWindowOffersToRemoveTheLeftovers(t *testing.T) {
	o := confirmOptions(0, false)
	o.Then = leftoversStep(0)
	c := wizard.HeadlessConfirm(fynetest.App(t), o)
	c.Act()
	require.Equal(t, "Example left 3 files it made.", c.Question())
	c.Act()
	require.Equal(t, "Example 1.0 was removed, with the files it made.", c.Message())
	c.Act()
	require.Equal(t, wizard.Finished, c.Result().Outcome)
}

func TestTheConfirmWindowAsksThenRemoves(t *testing.T) {
	c := wizard.HeadlessConfirm(fynetest.App(t), confirmOptions(0, false))
	c.Act()
	require.Equal(t, "Example 1.0 was removed.", c.Message())
	c.Act()
	require.Equal(t, wizard.Finished, c.Result().Outcome)
}
