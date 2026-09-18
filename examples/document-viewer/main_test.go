package main

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ushineko/fynedesygn/fynetest"
	"github.com/ushineko/fynedesygn/markdown"
	"github.com/ushineko/fynedesygn/shell"
)

func TestTheGuideShowsWithItsDiagramAndNothingScrollableInside(t *testing.T) {
	s := shell.Headless(fynetest.App(t), options())
	o := s.Current().Build(s)
	text := fynetest.Text(o)
	require.Contains(t, text, "How the pieces fit")
	require.NotContains(t, text, markdown.NotRenderedCaption, "run go generate")
	require.False(t, fynetest.ScrollableIn(o))
	for _, sec := range s.Sections() {
		require.NotNil(t, sec.Build(s))
	}
}
