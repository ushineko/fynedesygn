package dragout

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/require"
)

// fakeBackend records what a Source asks of the platform.
type fakeBackend struct {
	ok       bool
	started  [][]string
	released int
	prepared int
}

func (f *fakeBackend) supports(fyne.Window) bool { return f.ok }
func (f *fakeBackend) prepare(fyne.Window)       { f.prepared++ }
func (f *fakeBackend) start(_ fyne.Window, paths []string) error {
	f.started = append(f.started, paths)
	return nil
}
func (f *fakeBackend) release(fyne.Window) { f.released++ }

// withFake swaps the platform for a fake that supports every window.
func withFake(t *testing.T) *fakeBackend {
	t.Helper()
	f := &fakeBackend{ok: true}
	was := platform
	platform = f
	t.Cleanup(func() { platform = was })
	return f
}

func shown(t *testing.T, s *Source) fyne.Window {
	t.Helper()
	test.NewApp()
	w := test.NewWindow(s)
	w.Resize(fyne.NewSize(200, 200))
	t.Cleanup(w.Close)
	return w
}

func file(t *testing.T, dir, name string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(p, []byte("x"), 0o600))
	return p
}

// R2: every path comes back out of the list it went in as, whatever it
// holds, and each line ends in CRLF.
func TestAURIListRoundTripsAwkwardPaths(t *testing.T) {
	paths := []string{
		"/home/u/Pictures/a b.jpg",
		"/tmp/50% #1?.png",
		"/srv/日本語/ünïcödé.webp",
		"/tmp/line\nbreak.png",
	}
	list := uriList(paths)
	require.True(t, strings.HasSuffix(list, "\r\n"))
	lines := strings.Split(strings.TrimSuffix(list, "\r\n"), "\r\n")
	require.Len(t, lines, len(paths))
	for i, line := range lines {
		require.NotContains(t, line, " ")
		require.NotContains(t, line, "\n")
		u, err := url.Parse(line)
		require.NoError(t, err)
		require.Equal(t, "file", u.Scheme)
		require.Equal(t, paths[i], u.Path)
	}
}

// R3: only absolute paths to regular files that exist now are dragged, a
// symbolic link to a file counts and one to a directory does not.
func TestOnlyExistingRegularFilesAreDragged(t *testing.T) {
	dir := t.TempDir()
	good := file(t, dir, "good.jpg")
	linkToFile := filepath.Join(dir, "link.jpg")
	require.NoError(t, os.Symlink(good, linkToFile))
	linkToDir := filepath.Join(dir, "linkdir")
	require.NoError(t, os.Symlink(dir, linkToDir))

	got := usable([]string{
		"relative.jpg",
		filepath.Join(dir, "missing.jpg"),
		dir,
		linkToDir,
		good,
		linkToFile,
	})
	require.Equal(t, []string{good, linkToFile}, got)
}

// R3: a list with nothing usable in it starts no drag and is not a failure.
func TestNothingUsableStartsNoDrag(t *testing.T) {
	f := withFake(t)
	s := New(widget.NewLabel("img"), func() []string { return []string{"relative.jpg"} })
	s.OnFailed = func(err error) { t.Fatalf("unexpected failure %v", err) }
	w := shown(t, s)

	test.Drag(w.Canvas(), fyne.NewPos(50, 20), 20, 0)
	require.Empty(t, f.started)
	require.Zero(t, f.released)
}

// R1: a drag leaves the window once it has travelled the threshold, once
// per gesture, and Fyne is then told the button is up (quirk 45).
func TestADragLeavesTheWindowPastTheThreshold(t *testing.T) {
	f := withFake(t)
	path := file(t, t.TempDir(), "a.jpg")
	s := New(widget.NewLabel("img"), func() []string { return []string{path} })
	shown(t, s)

	step := &fyne.DragEvent{Dragged: fyne.NewDelta(Threshold/2-1, 0)}
	s.Dragged(step)
	s.Dragged(step)
	require.Empty(t, f.started, "under the threshold is still a click")

	s.Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(0, Threshold)})
	require.Equal(t, [][]string{{path}}, f.started)
	require.Equal(t, 1, f.released)

	s.Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(50, 50)})
	require.Len(t, f.started, 1, "one gesture is one drag")

	s.DragEnd()
	s.Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(Threshold, 0)})
	require.Len(t, f.started, 2, "the next gesture is a new drag")
}

// R1: the files are asked for when the drag starts, not when the Source is
// made, so a pane showing one file at a time drags the current one.
func TestTheFilesAreAskedForWhenTheDragStarts(t *testing.T) {
	f := withFake(t)
	dir := t.TempDir()
	first, second := file(t, dir, "1.jpg"), file(t, dir, "2.jpg")
	current := first
	s := New(widget.NewLabel("img"), func() []string { return []string{current} })
	w := shown(t, s)

	current = second
	test.Drag(w.Canvas(), fyne.NewPos(50, 20), 20, 0)
	require.Equal(t, [][]string{{second}}, f.started)
}

// R1/R4: under the test driver nothing is supported, a drag reports that
// once and starts nothing, and taps still reach the wrapped object.
func TestUnderTheTestDriverASourceIsAPlainContainer(t *testing.T) {
	tapped := 0
	button := widget.NewButton("img", func() { tapped++ })
	var failures []error
	s := New(button, func() []string { return []string{"/etc/hostname"} })
	s.OnFailed = func(err error) { failures = append(failures, err) }
	w := shown(t, s)

	require.False(t, Supported())
	test.Drag(w.Canvas(), fyne.NewPos(50, 20), 20, 0)
	test.Drag(w.Canvas(), fyne.NewPos(50, 20), 20, 0)
	require.Equal(t, []error{ErrUnsupported}, failures, "said once, not per gesture")

	test.TapCanvas(w.Canvas(), fyne.NewPos(50, 20))
	require.Equal(t, 1, tapped)
}

// Wayland only hears of a press through a pointer bound before it, so the
// platform is readied as soon as the Source is drawn in a window.
func TestTheSourceReadiesThePlatformWhenItIsDrawn(t *testing.T) {
	f := withFake(t)
	s := New(widget.NewLabel("img"), func() []string { return nil })
	shown(t, s)
	require.Positive(t, f.prepared)
}
