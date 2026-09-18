package mermaid

import (
	"bytes"
	"context"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"testing/fstest"

	"fyne.io/fyne/v2"
	"github.com/stretchr/testify/require"
)

const flow = "flowchart LR\n  A --> B\n"

func TestHashIgnoresSurroundingWhitespaceAndLineEndings(t *testing.T) {
	h := Hash(flow)
	require.Len(t, h, 16)
	require.Equal(t, h, Hash("\n\n"+flow+"\n\n"))
	require.Equal(t, h, Hash("flowchart LR\r\n  A --> B\r\n"))
	require.NotEqual(t, h, Hash("flowchart LR\n  A --> C\n"))
}

// tinyPNG is a 2x4 image so the two variants can be told apart by size.
func tinyPNG(t *testing.T, w, h int) []byte {
	t.Helper()
	var b bytes.Buffer
	require.NoError(t, png.Encode(&b, blank(w, h)))
	return b.Bytes()
}

func TestLookupPicksTheVariantAndFallsBackToTheOther(t *testing.T) {
	fsys := fstest.MapFS{
		"diagrams/" + FileName(flow, true):          {Data: tinyPNG(t, 2, 4)},
		"diagrams/" + FileName(flow, false):         {Data: tinyPNG(t, 4, 2)},
		"diagrams/" + FileName("only light", false): {Data: tinyPNG(t, 6, 6)},
	}
	set := NewSet(fsys, "")
	dark, ok := set.Lookup(flow, true)
	require.True(t, ok)
	require.Equal(t, FileName(flow, true), dark.Name())
	light, ok := set.Lookup(flow, false)
	require.True(t, ok)
	require.Equal(t, FileName(flow, false), light.Name())

	fallback, ok := set.Lookup("only light", true)
	require.True(t, ok)
	require.Equal(t, FileName("only light", false), fallback.Name(), "the other variant beats nothing")

	_, ok = set.Lookup("never rendered", true)
	require.False(t, ok)
	require.False(t, set.Has("never rendered"))
	var none *Set
	_, ok = none.Lookup(flow, true)
	require.False(t, ok, "a nil set is a set with nothing in it")
}

func TestDiagramMinHeightFollowsItsWidth(t *testing.T) {
	d := NewDiagram(fyne.NewStaticResource("d.png", tinyPNG(t, 400, 200)), 2)
	require.Equal(t, fyne.NewSize(200, 100), d.Natural())
	require.InDelta(t, 100, d.MinSize().Height, 0.01, "natural height at 1x when there is room")
	d.Resize(fyne.NewSize(100, 0))
	require.InDelta(t, 50, d.MinSize().Height, 0.01, "half the width, half the height")
	d.Resize(fyne.NewSize(1000, 0))
	require.InDelta(t, 100, d.MinSize().Height, 0.01, "never taller than natural")

	broken := NewDiagram(fyne.NewStaticResource("x.png", []byte("not a png")), 2)
	require.Equal(t, fyne.NewSize(0, 0), broken.MinSize())
}

func TestSourcesFindsFencesAndFilesAndSkipsTheDiagramsDir(t *testing.T) {
	root := t.TempDir()
	write := func(rel, content string) {
		p := filepath.Join(root, rel)
		require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o755))
		require.NoError(t, os.WriteFile(p, []byte(content), 0o600))
	}
	write("guide.md", "# Guide\n\ntext\n\n```mermaid\ngraph TD\n  X --> Y\n```\n\n```sh\nnot a diagram\n```\n\n   ```mermaid\nsequenceDiagram\n  A->>B: hi\n```\n")
	write("arch.mmd", "classDiagram\n  Foo\n")
	write("diagrams/stray.md", "```mermaid\nignored\n```\n")
	write(".hidden/h.mmd", "ignored")

	srcs, err := Sources(root)
	require.NoError(t, err)
	require.Len(t, srcs, 3)
	require.Equal(t, filepath.Join(root, "arch.mmd"), srcs[0].File)
	require.Equal(t, 1, srcs[0].Line)
	require.Equal(t, "classDiagram\n  Foo\n", srcs[0].Text)
	require.Equal(t, 5, srcs[1].Line)
	require.Equal(t, "graph TD\n  X --> Y", srcs[1].Text)
	require.Equal(t, 14, srcs[2].Line)
	require.Equal(t, "sequenceDiagram\n  A->>B: hi", srcs[2].Text)
}

func TestArgsAreTheDocumentedMmdcInvocation(t *testing.T) {
	require.Equal(t, []string{"-i", "in.mmd", "-o", "out.png", "-t", "dark", "-b", "transparent", "-s", "2", "-q"},
		Args("in.mmd", "out.png", true))
	require.Equal(t, "default", Args("a", "b", false)[5])
}

func TestCheckReportsMissingAndStale(t *testing.T) {
	root := t.TempDir()
	out := filepath.Join(root, DefaultDir)
	require.NoError(t, os.MkdirAll(out, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "a.mmd"), []byte(flow), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(out, "deadbeefdeadbeef-dark.png"), tinyPNG(t, 1, 1), 0o600))

	missing, stale, err := Check(root, out)
	require.NoError(t, err)
	require.Len(t, missing, 1)
	require.Equal(t, []string{filepath.Join(out, "deadbeefdeadbeef-dark.png")}, stale)

	// Half a pair is still missing.
	require.NoError(t, os.WriteFile(filepath.Join(out, FileName(flow, true)), tinyPNG(t, 1, 1), 0o600))
	missing, _, err = Check(root, out)
	require.NoError(t, err)
	require.Len(t, missing, 1)
	require.NoError(t, os.WriteFile(filepath.Join(out, FileName(flow, false)), tinyPNG(t, 1, 1), 0o600))
	missing, _, err = Check(root, out)
	require.NoError(t, err)
	require.Empty(t, missing)
}

// The real renderer, when the developer's machine has it. CI does not, and
// skips; the argument slice is covered above.
func TestRenderWritesBothVariantsWithRealMmdc(t *testing.T) {
	if _, err := exec.LookPath("mmdc"); err != nil {
		t.Skip("mmdc not installed")
	}
	out := t.TempDir()
	require.NoError(t, Render(context.Background(), "mmdc", flow, out))
	for _, dark := range []bool{false, true} {
		b, err := os.ReadFile(filepath.Join(out, FileName(flow, dark)))
		require.NoError(t, err)
		cfg, err := png.DecodeConfig(bytes.NewReader(b))
		require.NoError(t, err)
		require.Positive(t, cfg.Width)
	}
}
