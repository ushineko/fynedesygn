package main

import (
	"testing"

	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/require"

	"github.com/ushineko/fynedesygn/fynetest"
	"github.com/ushineko/fynedesygn/shell"
)

func TestTheListingLoadsInlineAndSelectionShowsTheDetail(t *testing.T) {
	st := newState()
	s := shell.Headless(fynetest.App(t), options(st, 0))
	o := s.Current().Build(s)
	require.True(t, st.itemsOK)
	require.Len(t, st.items, 4, "a headless Perform runs inline, so the load landed before Build returned")
	require.Contains(t, fynetest.Text(o), "Select a row.")

	tbl := fynetest.Find[*widget.Table](o)
	require.NotNil(t, tbl)
	tbl.OnSelected(widget.TableCellID{Row: 3})
	require.Equal(t, 3, st.selected)
	text := fynetest.Text(s.Current().Build(s))
	require.Contains(t, text, "delta.so")
	require.Contains(t, text, "failed to load")

	bar := ""
	for _, seg := range options(st, 0).StatusBar(s) {
		bar += fynetest.Text(seg) + " "
	}
	require.Contains(t, bar, "4")

	s.Invalidate()
	require.False(t, st.itemsOK)
	require.Equal(t, -1, st.selected)
}
