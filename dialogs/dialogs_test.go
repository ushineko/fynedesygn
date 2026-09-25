package dialogs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/require"

	"github.com/ushineko/fynedesygn/fynetest"
	fdtheme "github.com/ushineko/fynedesygn/theme"
)

func testWindow(t *testing.T) fyne.Window {
	t.Helper()
	fynetest.App(t)
	w := test.NewWindow(widget.NewLabel("root"))
	w.Resize(fyne.NewSize(900, 700))
	t.Cleanup(w.Close)
	return w
}

// overlay is the dialog on top of the window, for tapping its buttons.
func overlay(t *testing.T, w fyne.Window) fyne.CanvasObject {
	t.Helper()
	top := w.Canvas().Overlays().Top()
	require.NotNil(t, top, "a dialog should be showing")
	return top
}

func TestConfirmDestructiveRunsDoOnlyOnTheDangerButton(t *testing.T) {
	w := testWindow(t)
	ran := 0
	ConfirmDestructive(w, "Remove the thing?", "The thing goes. Its backups stay.", "Remove", func() { ran++ })
	dlg := overlay(t, w)
	proceed := fynetest.FindButton(dlg, "Remove")
	require.NotNil(t, proceed)
	require.Equal(t, widget.DangerImportance, proceed.Importance)
	test.Tap(fynetest.FindButton(dlg, "Cancel"))
	require.Zero(t, ran)
	require.Nil(t, w.Canvas().Overlays().Top())

	ConfirmDestructive(w, "Again?", "detail", "Remove", func() { ran++ })
	test.Tap(fynetest.FindButton(overlay(t, w), "Remove"))
	require.Equal(t, 1, ran)
}

func TestTheOtherDialogsShowWithoutPanic(t *testing.T) {
	w := testWindow(t)
	require.NotPanics(t, func() {
		d := ConfirmWithBody(w, "Choice", widget.NewCheck("also this", nil), "Do it", func() {})
		d.Show()
		d.Hide()
		Prompt(w, "Name it", "Save", widget.NewEntry(), func() {})
		require.NotNil(t, w.Canvas().Overlays().Top())
		test.Tap(fynetest.FindButton(overlay(t, w), "Cancel"))
		ShowDetail(w, "Details", widget.NewLabel("long answer"), 600, 400)
		test.Tap(fynetest.FindButton(overlay(t, w), "Close"))
	})
}

func TestPickerStartPrefersTheFieldThenItsParentThenHome(t *testing.T) {
	home := fynetest.Sandbox(t)
	dir := filepath.Join(home, "work")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	file := filepath.Join(dir, "a.txt")
	require.NoError(t, os.WriteFile(file, []byte("x"), 0o600))

	// Fyne's URI paths use forward slashes on every platform; compare in
	// that form so the test holds on Windows.
	at := func(lu fyne.ListableURI) string { return filepath.ToSlash(lu.Path()) }
	require.Equal(t, filepath.ToSlash(dir), at(PickerStart(dir)))
	require.Equal(t, filepath.ToSlash(dir), at(PickerStart(file)), "a file opens in its directory")
	require.Equal(t, filepath.ToSlash(dir), at(PickerStart("~/work")), "a tilde is the home directory")
	require.Equal(t, filepath.ToSlash(home), at(PickerStart("/nonsense/that/does/not/exist")))
	require.Equal(t, filepath.ToSlash(home), at(PickerStart("")))
}

// Canary #1 (docs/fyne-quirks.md): a file dialog resized before Show
// dereferences a nil window in Fyne 2.8.1. The choosers resize after.
func TestChooserResizeAfterShowDoesNotPanic(t *testing.T) {
	w := testWindow(t)
	field := widget.NewEntry()
	require.NotPanics(t, func() { test.Tap(BrowseButton(w, field, false)) })
	require.NotNil(t, w.Canvas().Overlays().Top())
	w.Canvas().Overlays().Top().Hide()
	require.NotPanics(t, func() { test.Tap(BrowseButton(w, field, true)) })
	require.NotPanics(t, func() { ChooseFile(w, "", nil, func(string) {}) })
	require.NotPanics(t, func() { ChooseFolder(w, "", func(string) {}) })
	require.NotNil(t, WithBrowse(w, field, false))
}

func TestOpenPathRefusesAnEmptyPathAndNamesTheOpener(t *testing.T) {
	require.ErrorIs(t, OpenPath(""), ErrNoPath)
	require.NotEmpty(t, Opener())
}

// Decide itself is a goroutine hop plus a channel wait, and under the test
// driver fyne.Do runs inline on the worker's goroutine, so driving Decide from
// a goroutine while tapping from the test races by construction (quirk 11).
// The question it puts up is tested directly.
func TestTheQuestionAnswersExactlyOnceWhicheverWayItCloses(t *testing.T) {
	w := testWindow(t)
	for _, tc := range []struct {
		tap  string
		want bool
	}{{"Yes", true}, {"No", false}} {
		answer := make(chan bool, 2)
		ask(w, "Continue?", "It is safe.", "Yes", "No", answer)
		dlg := overlay(t, w)
		test.Tap(fynetest.FindButton(dlg, tc.tap))
		select {
		case got := <-answer:
			require.Equal(t, tc.want, got)
		case <-time.After(time.Second):
			t.Fatal("no answer")
		}
		require.Nil(t, w.Canvas().Overlays().Top(), "the dialog is gone afterwards")
		select {
		case extra := <-answer:
			t.Fatalf("answered twice: %v", extra)
		default:
		}
	}

	// Dismissed without a button (Escape, a tap outside): one false.
	answer := make(chan bool, 2)
	d := ask(w, "Continue?", "detail", "Yes", "No", answer)
	d.Hide()
	select {
	case got := <-answer:
		require.False(t, got)
	case <-time.After(time.Second):
		t.Fatal("no answer on dismissal")
	}
}

// closer is the dialog's corner X: the one button with an icon and no label.
func closer(t *testing.T, o fyne.CanvasObject) *widget.Button {
	t.Helper()
	for _, b := range fynetest.All[*widget.Button](o) {
		if b.Text == "" && b.Icon != nil {
			return b
		}
	}
	return nil
}

/*
Every dialog closes from its corner.

Fyne's dialog draws a title and the buttons it is given and nothing else, so
without this a dialog can only be left by finding the right button among the
others -- which is fine for a confirmation, where choosing is the point, and
wrong for a long read-only answer somebody opened to look at.
*/
func TestEveryDialogClosesFromItsCorner(t *testing.T) {
	t.Run("ShowDetail", func(t *testing.T) {
		w := testWindow(t)
		ShowDetail(w, "A long answer", widget.NewLabel("body"), 400, 300)
		x := closer(t, overlay(t, w))
		require.NotNil(t, x, "no corner close")
		test.Tap(x)
		require.Nil(t, w.Canvas().Overlays().Top())
	})

	// The X on a confirmation is a dismissal, never the destructive choice.
	t.Run("ConfirmDestructive", func(t *testing.T) {
		w := testWindow(t)
		ran := 0
		ConfirmDestructive(w, "Remove?", "detail", "Remove", func() { ran++ })
		test.Tap(closer(t, overlay(t, w)))
		require.Nil(t, w.Canvas().Overlays().Top())
		require.Zero(t, ran, "the corner X ran the destructive action")
	})

	t.Run("Prompt", func(t *testing.T) {
		w := testWindow(t)
		ran := 0
		Prompt(w, "Name it", "Save", widget.NewEntry(), func() { ran++ })
		test.Tap(closer(t, overlay(t, w)))
		require.Nil(t, w.Canvas().Overlays().Top())
		require.Zero(t, ran, "the corner X confirmed the prompt")
	})

	// A blocking question must answer exactly once however it is closed, or
	// the worker waiting on it never wakes up.
	t.Run("Decide", func(t *testing.T) {
		w := testWindow(t)
		answer := make(chan bool, 1)
		ask(w, "Go on?", "detail", "Yes", "No", answer)
		test.Tap(closer(t, overlay(t, w)))
		select {
		case got := <-answer:
			require.False(t, got, "the corner X answered yes")
		case <-time.After(time.Second):
			t.Fatal("the corner X left the caller waiting")
		}
	})
}

/*
The font chooser previews the highlighted family and commits nothing until
Choose.

The dropdown it replaces listed 311 names in a face that told you nothing
about any of them, so choosing meant applying one to the whole window to see
it and applying another to get back.
*/
func TestChooseFontPreviewsBeforeCommitting(t *testing.T) {
	w := testWindow(t)
	chosen := ""
	base := fdtheme.DefaultAppearance()

	ChooseFont(w, "Interface font", base.Font, base, false, func(name string) { chosen = name })
	dlg := overlay(t, w)

	/*
		The sample carries the family's own file.

		A theme override does not survive the trip to the painter: text is
		measured through a path that is handed no object, and the face is
		cached against a scope the text objects do not always carry. A
		resource on the object is what actually decides the glyphs.

		Highlighted first, because the chooser opens on whatever the window
		is using, and the bundled default has no file to name.
	*/
	list, found := fynetest.First[*widget.List](dlg)
	require.True(t, found, "the chooser has no list")
	for i, name := range fdtheme.FontNames() {
		if fdtheme.PreviewFont(name) != nil {
			list.Select(i)
			break
		}
	}
	var sourced bool
	for _, line := range fynetest.All[*canvas.Text](dlg) {
		if line.FontSource != nil {
			sourced = true
			break
		}
	}
	if !sourced {
		t.Error("no line of the sample names the font it is drawn from")
	}
	// And it names the family, which is what the eye checks against the list.
	if !strings.Contains(strings.Join(fynetest.Texts(dlg), "\n"), base.Font) {
		t.Errorf("the sample does not name the family: %v", fynetest.Texts(dlg))
	}

	// Browsing has answered nothing yet.
	if chosen != "" {
		t.Errorf("opening the chooser already chose %q", chosen)
	}

	// Choose takes the family the sample is showing, which is the one that
	// was highlighted.
	showing := ""
	for _, text := range fynetest.Texts(dlg) {
		if strings.HasPrefix(text, "Sample — ") {
			showing = strings.TrimPrefix(text, "Sample — ")
		}
	}
	test.Tap(fynetest.FindButton(dlg, "Choose"))
	if chosen != showing {
		t.Errorf("Choose answered %q while the sample showed %q", chosen, showing)
	}
}

// Cancel answers nothing at all: the window keeps the font it had.
func TestChooseFontCancelsWithoutChoosing(t *testing.T) {
	w := testWindow(t)
	chosen := ""
	base := fdtheme.DefaultAppearance()

	ChooseFont(w, "Interface font", base.Font, base, false, func(name string) { chosen = name })
	test.Tap(fynetest.FindButton(overlay(t, w), "Cancel"))

	if chosen != "" {
		t.Errorf("Cancel chose %q", chosen)
	}
	require.Nil(t, w.Canvas().Overlays().Top())
}

// TestChooseFontWithDrawsTheCallersSample: the sample is the caller's, drawn
// for the family the cursor is on, and it moves with the cursor.
func TestChooseFontWithDrawsTheCallersSample(t *testing.T) {
	a := fynetest.App(t)
	w := test.NewWindow(widget.NewLabel(""))
	t.Cleanup(w.Close)
	base := fdtheme.Appearance{TextSize: 12}
	names := []string{"Alpha Sans", "Beta Serif"}
	var drawn []string
	ChooseFontWith(w, "Indicator font", names, "Alpha Sans", base, false,
		func(name string, _ *fdtheme.Font) fyne.CanvasObject {
			drawn = append(drawn, name)
			return widget.NewLabel("sample in " + name)
		}, func(string) {})
	require.NotEmpty(t, drawn)
	require.Equal(t, "Alpha Sans", drawn[len(drawn)-1], "the sample was not drawn for the current family")
	overlay := w.Canvas().Overlays().Top()
	require.NotNil(t, overlay)
	require.NotNil(t, fynetest.FindLabel(overlay, "sample in Alpha Sans"))
	list := fynetest.Find[*widget.List](overlay)
	list.Select(1)
	require.Equal(t, "Beta Serif", drawn[len(drawn)-1], "the sample did not follow the cursor")
	require.NotNil(t, fynetest.FindLabel(overlay, "sample in Beta Serif"))
	_ = a
}
