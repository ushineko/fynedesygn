package theme

import (
	"os"
	"path/filepath"
	"testing"

	"fyne.io/fyne/v2"
	"github.com/stretchr/testify/require"
)

// fontDir builds a temporary font directory. The files need only exist with
// the right names: the scanner classifies by filename and LoadFont reads
// bytes without parsing them.
func fontDir(t *testing.T, names ...string) string {
	t.Helper()
	dir := t.TempDir()
	for _, n := range names {
		require.NoError(t, os.WriteFile(filepath.Join(dir, n), []byte("font "+n), 0o600))
	}
	return dir
}

func TestVariableFontsAreSkippedAndBoldResolvesToItsOwnFace(t *testing.T) {
	dir := fontDir(t,
		"DejaVuSans.ttf", "DejaVuSans-Bold.ttf", "DejaVuSans-Oblique.ttf",
		"Inter[wght].ttf",
		"OnlyBold-Bold.otf", // no regular face: not offered
		"notes.txt",
	)
	RescanFonts(dir)
	t.Cleanup(func() { RescanFonts() })

	require.Equal(t, []string{DefaultFontName, "Deja Vu Sans"}, FontNames())

	f := LoadFont("Deja Vu Sans")
	require.NotNil(t, f)
	require.Equal(t, "DejaVuSans-Bold.ttf", f.Face(fyne.TextStyle{Bold: true}).Name())
	require.Equal(t, "DejaVuSans-Oblique.ttf", f.Face(fyne.TextStyle{Italic: true}).Name())
	// No bold-italic face ships, so it falls back to regular rather than to an
	// unrelated font.
	require.Equal(t, "DejaVuSans.ttf", f.Face(fyne.TextStyle{Bold: true, Italic: true}).Name())
}

func TestTheDefaultAndUnknownAndUnreadableFontsAllMeanFynesOwn(t *testing.T) {
	dir := fontDir(t, "Gone.ttf")
	RescanFonts(dir)
	t.Cleanup(func() { RescanFonts() })
	require.Nil(t, LoadFont(DefaultFontName))
	require.Nil(t, LoadFont(""))
	require.Nil(t, LoadFont("Never Installed"))

	// Listed, then removed before it is read: falls back, never fails.
	require.NoError(t, os.Remove(filepath.Join(dir, "Gone.ttf")))
	require.Nil(t, LoadFont("Gone"))
	var nilFont *Font
	require.Nil(t, nilFont.Face(fyne.TextStyle{Bold: true}))
}

func TestSplitFaceReadsFamilyAndStyleFromTheFilename(t *testing.T) {
	for _, tc := range []struct{ in, name, style string }{
		{"DejaVuSans-BoldOblique.ttf", "Deja Vu Sans", "boldoblique"},
		{"arial.ttf", "arial", "regular"},
		{"NotoSansMono-Regular.otf", "Noto Sans Mono", "regular"},
		{"Inter[wght].ttf", "", ""},
	} {
		name, style := splitFace(tc.in)
		require.Equal(t, tc.name, name, tc.in)
		require.Equal(t, tc.style, style, tc.in)
	}
}

func TestPlatformFontDirectoriesAreNonEmpty(t *testing.T) {
	require.NotEmpty(t, platformFontDirs())
}
