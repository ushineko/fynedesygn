package theme

import (
	"os"
	"path/filepath"
	"testing"
	"time"

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

/*
Loading every family is not free, which is why previewing does not.

The number in the comment on PreviewFont comes from here. It is a measurement
rather than an assertion about this machine: the check is only that a family
costs enough to matter, because the design of the font chooser rests on it.
*/
func TestLoadingEveryFamilyIsNotFree(t *testing.T) {
	names := FontNames()
	if len(names) < 2 {
		t.Skip("no system fonts on this machine")
	}

	start := time.Now()
	bytes, loaded := 0, 0
	for _, name := range names {
		f := PreviewFont(name)
		if f == nil || f.regular == nil {
			continue
		}
		loaded++
		bytes += len(f.regular.Content())
	}
	took := time.Since(start)
	if loaded == 0 {
		t.Skip("no readable families on this machine")
	}
	t.Logf("%d families, %.0f MB of regular faces, %v", loaded,
		float64(bytes)/(1<<20), took)

	// A face is a file, and files are not small. If this ever stops being
	// true the chooser could draw every row in its own font after all.
	if perFamily := bytes / loaded; perFamily < 50*1024 {
		t.Errorf("a family's regular face averages %d bytes; preview-one-at-a-time "+
			"may no longer be necessary", perFamily)
	}
}

/*
Previewing a family does not leave it resident.

The chooser reads a font per name somebody moves through, and the font cache
never releases: previewing through LoadFont would have grown the process by a
family's worth of memory for every name scrolled past.
*/
func TestPreviewingDoesNotFillTheCache(t *testing.T) {
	names := FontNames()
	var name string
	for _, candidate := range names {
		if candidate != DefaultFontName && PreviewFont(candidate) != nil {
			name = candidate
			break
		}
	}
	if name == "" {
		t.Skip("no readable system font on this machine")
	}

	fontsMu.Lock()
	_, cached := fontCache[name]
	fontsMu.Unlock()
	if cached {
		t.Fatalf("%s was already resident, so this proves nothing", name)
	}

	PreviewFont(name)

	fontsMu.Lock()
	_, cached = fontCache[name]
	fontsMu.Unlock()
	if cached {
		t.Errorf("previewing %s left it in the cache", name)
	}

	// And loading it for real still does.
	LoadFont(name)
	fontsMu.Lock()
	_, cached = fontCache[name]
	fontsMu.Unlock()
	if !cached {
		t.Errorf("loading %s did not keep it", name)
	}
}
