package mermaid_test

import (
	"bytes"
	"image"
	"image/png"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/require"

	"github.com/ushineko/fynedesygn/imagecache"
	"github.com/ushineko/fynedesygn/mermaid"
)

func TestTheSameDiagramIsDecodedOnce(t *testing.T) {
	/*
		The measurement that produced the cache: a document pane measures its
		blocks by rendering them, and a section is rebuilt when somebody
		navigates to it, so one 1246x2186 diagram was decoded at 10.4 MB a
		time over and over. Twelve copies were alive at once in hotaru.
	*/
	test.NewTempApp(t)
	imagecache.Shared.Clear()

	res := fyne.NewStaticResource("decode-once-test", png2x2(t))
	before, _ := imagecache.Shared.Counts()

	first := mermaid.NewDiagram(res, 1)
	second := mermaid.NewDiagram(res, 1)
	require.Equal(t, first.Natural(), second.Natural())

	hits, _ := imagecache.Shared.Counts()
	require.Equal(t, before+1, hits, "the second diagram decoded again")
}

// png2x2 is the smallest thing image.Decode will accept.
func png2x2(t *testing.T) []byte {
	t.Helper()
	var out bytes.Buffer
	require.NoError(t, png.Encode(&out, image.NewRGBA(image.Rect(0, 0, 2, 2))))
	return out.Bytes()
}
