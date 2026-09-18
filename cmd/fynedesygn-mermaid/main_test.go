package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ushineko/fynedesygn/mermaid"
)

func TestCheckExitsOneWhenAnImageIsMissingAndZeroWhenClean(t *testing.T) {
	root := t.TempDir()
	out := filepath.Join(root, mermaid.DefaultDir)
	require.NoError(t, os.MkdirAll(out, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "a.mmd"), []byte("flowchart LR\n A-->B\n"), 0o600))
	devnull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	require.NoError(t, err)
	defer func() { _ = devnull.Close() }()

	require.Equal(t, 1, run([]string{"-check", "-root", root}, devnull, devnull), "missing pair")

	for _, dark := range []bool{false, true} {
		require.NoError(t, os.WriteFile(filepath.Join(out, mermaid.FileName("flowchart LR\n A-->B\n", dark)), []byte("png"), 0o600))
	}
	require.Equal(t, 0, run([]string{"-check", "-root", root}, devnull, devnull), "clean")

	require.NoError(t, os.WriteFile(filepath.Join(out, "0123456789abcdef-dark.png"), []byte("png"), 0o600))
	require.Equal(t, 1, run([]string{"-check", "-root", root}, devnull, devnull), "stale image")
	require.Equal(t, 0, run([]string{"-prune", "-root", root}, devnull, devnull), "prune deletes the stale image without rendering")
	_, err = os.Stat(filepath.Join(out, "0123456789abcdef-dark.png"))
	require.True(t, os.IsNotExist(err))
	require.Equal(t, 2, run([]string{"-bogus"}, devnull, devnull), "bad flag")
}

func TestTheCommittedDocsAreCurrent(t *testing.T) {
	devnull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	require.NoError(t, err)
	defer func() { _ = devnull.Close() }()
	require.Equal(t, 0, run([]string{"-check", "-root", "../../docs"}, devnull, devnull))
}
