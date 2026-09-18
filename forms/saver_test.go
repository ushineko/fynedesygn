package forms

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ushineko/fynedesygn/fynetest"
)

func TestSaverCoalescesChangesIntoOneSave(t *testing.T) {
	fynetest.App(t)
	var saves atomic.Int32
	s := NewSaver(40*time.Millisecond, func() { saves.Add(1) })
	s.Schedule()
	s.Schedule()
	s.Schedule()
	require.True(t, s.Pending())
	require.Eventually(t, func() bool { return saves.Load() == 1 }, time.Second, 5*time.Millisecond)
	time.Sleep(80 * time.Millisecond)
	require.Equal(t, int32(1), saves.Load(), "three changes, one write")
	require.False(t, s.Pending())
}

func TestFlushSavesAPendingChangeAtOnceAndNothingOtherwise(t *testing.T) {
	fynetest.App(t)
	var saves atomic.Int32
	s := NewSaver(time.Hour, func() { saves.Add(1) })
	s.Flush()
	require.Zero(t, saves.Load())
	s.Schedule()
	s.Flush()
	require.Equal(t, int32(1), saves.Load())
	require.False(t, s.Pending())
	require.Equal(t, DefaultSaveDelay, NewSaver(0, nil).delay)
}
