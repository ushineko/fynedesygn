package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ushineko/fynedesygn/fynetest"
	"github.com/ushineko/fynedesygn/shell"
)

func TestTheTransferRunsInlineHeadlesslyToTheEnd(t *testing.T) {
	s := shell.Headless(fynetest.App(t), options(0))
	o := s.Current().Build(s)
	fynetest.FindButton(o, "Copy 64 MiB").OnTapped()
	require.False(t, s.Working())
	require.Equal(t, "The transfer finished.", s.FlashText())
}

func TestACancelledContextStopsTheCopyWithTheErrorWrapped(t *testing.T) {
	s := shell.Headless(fynetest.App(t), options(0))
	_ = s.Current().Build(s)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := copyBytes(ctx, s, 0)
	require.ErrorIs(t, err, context.Canceled)
	require.Contains(t, err.Error(), "stopped at 0 of 64 MiB")
	require.Empty(t, s.FlashText(), "a stopped copy is not announced as finished")
}
