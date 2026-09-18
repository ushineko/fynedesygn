package shell

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	fynetheme "fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/require"

	fd "github.com/ushineko/fynedesygn"
	"github.com/ushineko/fynedesygn/fynetest"
	fdtheme "github.com/ushineko/fynedesygn/theme"
)

// logSection records the order the shell calls it in.
type logSection struct {
	title string
	log   *[]string
}

func (l *logSection) Title() string       { return l.title }
func (l *logSection) Icon() fyne.Resource { return fynetheme.HomeIcon() }
func (l *logSection) Build(*Shell) fyne.CanvasObject {
	*l.log = append(*l.log, "build "+l.title)
	return widget.NewLabel(l.title)
}
func (l *logSection) Detach() { *l.log = append(*l.log, "detach "+l.title) }

func testOptions(sections ...Section) Options {
	return Options{AppID: "io.ushineko.fynedesygn.test", Name: "test", Sections: sections}
}

// headless is a shell with no window; onScreen adds a test window so the
// swap and popup paths run without a display.
func headless(t *testing.T, o Options) *Shell {
	t.Helper()
	return Headless(fynetest.App(t), o)
}

func onScreen(t *testing.T, o Options) *Shell {
	t.Helper()
	s := headless(t, o)
	s.Window = test.NewWindow(nil)
	t.Cleanup(s.Window.Close)
	s.buildWindow()
	return s
}

func TestNamesNeedNoApp(t *testing.T) {
	// Canary #8: read before any Fyne app exists, as flag help does.
	secs := []Section{NewSection("One", fynetheme.HomeIcon, nil), NewSection("Two", nil, nil)}
	require.Equal(t, []string{"One", "Two"}, Names(secs))
	require.Nil(t, secs[1].Icon())
}

func TestTheOutgoingSectionIsDetachedBeforeTheIncomingOneIsBuilt(t *testing.T) {
	var log []string
	a := &logSection{"A", &log}
	b := &logSection{"B", &log}
	s := onScreen(t, testOptions(a, b))
	log = nil
	s.Select("B")
	require.Equal(t, []string{"detach A", "detach B", "build B"}, log)
	require.Equal(t, b, s.Current())
}

func TestPerformRunsInlineWhenHeadless(t *testing.T) {
	// Canary #11: Fyne's test driver runs fyne.Do inline, so the shell runs
	// the work inline too and a test sees a finished operation on return.
	s := headless(t, testOptions(NewSection("A", nil, func(*Shell) fyne.CanvasObject { return widget.NewLabel("a") })))
	require.False(t, s.OnScreen())
	ran := false
	s.Perform("Working...", func(context.Context) error { ran = true; return nil })
	require.True(t, ran)
	require.NotNil(t, s.Current().Build(s))
}

func TestPerformRefusesWhileSomethingIsRunning(t *testing.T) {
	s := headless(t, testOptions(NewSection("A", nil, nil)))
	release := s.Busy("Loading...")
	ran := false
	s.Perform("Second...", func(context.Context) error { ran = true; return nil })
	require.False(t, ran)
	require.Contains(t, s.FlashText(), "already running")
	release()
	release() // idempotent
	require.False(t, s.Working())

	external := true
	s = headless(t, Options{AppID: "x", Name: "x", Sections: []Section{NewSection("A", nil, nil)}, AlsoWorking: func() bool { return external }})
	s.Perform("Third...", func(context.Context) error { ran = true; return nil })
	require.False(t, ran)
	external = false
	s.Perform("Third...", func(context.Context) error { ran = true; return nil })
	require.True(t, ran)
}

func TestBusyIsACountNotAFlag(t *testing.T) {
	s := headless(t, testOptions(NewSection("A", nil, nil)))
	var wg sync.WaitGroup
	releases := make(chan func(), 2)
	for range 2 {
		wg.Go(func() { releases <- s.Busy("Loading...") })
	}
	wg.Wait()
	require.True(t, s.Working())
	r1, r2 := <-releases, <-releases
	r1()
	require.True(t, s.Working(), "one of two still running")
	r2()
	require.False(t, s.Working())
}

func TestOneBannerAtATimeAndFailuresStayUntilDismissed(t *testing.T) {
	s := headless(t, testOptions(NewSection("A", nil, nil)))
	s.Flash("first", fd.StatusGood)
	s.Flash("second", fd.StatusBad)
	require.Equal(t, "second", s.FlashText())
	require.Len(t, s.flashes.Objects, 1)

	hold, fades := flashHold(fd.StatusBad)
	require.False(t, fades)
	require.Zero(t, hold)
	hold, fades = flashHold(fd.StatusWarn)
	require.True(t, fades)
	require.Equal(t, FlashHoldWarn, hold)

	// A stale timer for the first banner must not clear the second.
	s.clearFlash(s.flashSeq - 1)
	require.Equal(t, "second", s.FlashText())
	s.ClearFlash()
	require.Empty(t, s.FlashText())
}

func TestBannerTintIsThePaletteRoleAtLowAlpha(t *testing.T) {
	s := headless(t, testOptions(NewSection("A", nil, nil)))
	s.SetAppearance(fdtheme.Appearance{Scheme: "Breeze Dark", Font: fdtheme.DefaultFontName, Mono: fdtheme.DefaultFontName})
	p := fdtheme.BreezeDark
	for st, want := range map[fd.Status]any{
		fd.StatusGood: p.Positive, fd.StatusWarn: p.Neutral, fd.StatusBad: p.Negative, fd.StatusInfo: p.SelectionBG,
	} {
		got := s.flashTint(st)
		require.Equal(t, fdtheme.Alpha(want.(interface {
			RGBA() (uint32, uint32, uint32, uint32)
		}), 0x4d), got, st.String())
	}
}

func TestReportIsSilentOnCancellation(t *testing.T) {
	s := headless(t, testOptions(NewSection("A", nil, nil)))
	s.Report("Fetching", context.Canceled)
	require.Empty(t, s.FlashText())
	s.Report("Fetching", nil)
	require.Empty(t, s.FlashText())
	s.Report("Fetching", errors.New("network down"))
	require.Equal(t, "Fetching failed: network down", s.FlashText())
}

func TestPerformCancellableCancelsThroughTheContext(t *testing.T) {
	s := onScreen(t, testOptions(NewSection("A", nil, func(*Shell) fyne.CanvasObject { return widget.NewLabel("a") })))
	started := make(chan struct{})
	finished := make(chan error, 1)
	s.PerformCancellable("Long job...", func(ctx context.Context) error {
		close(started)
		<-ctx.Done()
		finished <- ctx.Err()
		return ctx.Err()
	})
	<-started
	require.Eventually(t, func() bool { return s.busyCancel != nil }, time.Second, 10*time.Millisecond)
	s.busyCancel()
	select {
	case err := <-finished:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(2 * time.Second):
		t.Fatal("the job did not see the cancellation")
	}
	require.Eventually(t, func() bool { return !s.Working() }, 2*time.Second, 10*time.Millisecond)
	require.Empty(t, s.FlashText(), "cancellation is not a failure")
}

func TestSetAppearanceSavesUnlessTheSchemeIsAOneRunOverride(t *testing.T) {
	s := headless(t, testOptions(NewSection("A", nil, nil)))
	a := s.Appearance()
	a.Scheme = "macOS Dark"
	s.SetAppearance(a)
	require.Equal(t, "macOS Dark", s.App.Preferences().String(fdtheme.PrefScheme))

	o := testOptions(NewSection("A", nil, nil))
	o.Scheme = "Oxygen Dark"
	s2 := Headless(s.App, o)
	require.Equal(t, "Oxygen Dark", s2.Appearance().Scheme)
	a = s2.Appearance()
	a.Scheme = "Windows Light"
	a.TextSize = 14
	s2.SetAppearance(a)
	require.Equal(t, "macOS Dark", s.App.Preferences().String(fdtheme.PrefScheme), "the override never reaches the store")
	require.InDelta(t, 14, s.App.Preferences().Float(fdtheme.PrefTextSize), 0.001, "the other choices do")
	th, ok := s.App.Settings().Theme().(fdtheme.Theme)
	require.True(t, ok)
	require.Equal(t, "Windows Light", th.Palette().Name)
}

func TestAppearanceSectionRendersWithItsPickerPopulated(t *testing.T) {
	fdtheme.RescanFonts(t.TempDir())
	t.Cleanup(func() { fdtheme.RescanFonts() })
	s := headless(t, testOptions(AppearanceSection("$ make test")))
	o := s.Current().Build(s)
	sel := fynetest.FindSelect(o)
	require.NotNil(t, sel)
	require.Equal(t, fdtheme.SchemeNames(), sel.Options)
	require.Contains(t, fynetest.Text(o), "$ make test")
	sel.SetSelected("Adwaita Light")
	require.Equal(t, "Adwaita Light", s.Appearance().Scheme)
}

func TestAboutSectionShowsEverythingItIsGiven(t *testing.T) {
	s := headless(t, testOptions(AboutSection(About{
		Icon: fynetheme.HomeIcon(), Name: "demo", Version: "1.2.3", Blurb: "Does a thing.",
		URL:   "https://example.invalid/demo",
		Notes: []Note{{"First", "detail one"}, {"Second", "detail two"}},
		Facts: []Fact{{"Licence", "MIT"}, {"Config", "~/.config/demo"}},
		Extra: func(*Shell) fyne.CanvasObject { return widget.NewLabel("extra content") },
	})))
	text := fynetest.Text(s.Current().Build(s))
	for _, want := range []string{"demo", "1.2.3", "Does a thing.", "https://example.invalid/demo", "First", "detail one", "Second", "Licence", "MIT", "~/.config/demo", "extra content"} {
		require.Contains(t, text, want)
	}
}

func TestInvalidateCallsTheHookAndBuildsOnlyOnScreen(t *testing.T) {
	builds, hooks := 0, 0
	sec := NewSection("A", nil, func(*Shell) fyne.CanvasObject { builds++; return widget.NewLabel("a") })
	o := testOptions(sec)
	o.OnInvalidate = func(*Shell) { hooks++ }
	s := headless(t, o)
	s.Invalidate()
	require.Equal(t, 1, hooks)
	require.Zero(t, builds)

	s = onScreen(t, o)
	builds = 0
	s.Invalidate()
	require.Equal(t, 2, hooks)
	require.Equal(t, 1, builds)
}

func TestUnknownStartSectionOpensTheFirst(t *testing.T) {
	o := testOptions(NewSection("A", nil, nil), NewSection("B", nil, nil))
	o.Section = "nope"
	require.Equal(t, "A", headless(t, o).Current().Title())
	o.Section = "b"
	require.Equal(t, "B", headless(t, o).Current().Title())
}

func TestGateDisablesOnlyWhileWorking(t *testing.T) {
	s := headless(t, testOptions(NewSection("A", nil, nil)))
	b := widget.NewButton("go", nil)
	s.Gate(b)
	require.False(t, b.Disabled())
	release := s.Busy("x")
	s.Gate(b)
	require.True(t, b.Disabled())
	release()
}

func TestOnCreateRunsBeforeAnyBuilderAndThemeHookShapesTheTheme(t *testing.T) {
	var got *Shell
	order := []string{}
	mono := &fdtheme.Font{}
	o := testOptions(NewSection("A", nil, func(s *Shell) fyne.CanvasObject {
		order = append(order, "build")
		require.Same(t, got, s, "the builder sees the shell OnCreate stored")
		return widget.NewLabel("a")
	}))
	o.OnCreate = func(s *Shell) { got = s; order = append(order, "create") }
	o.Theme = func(a fdtheme.Appearance) fyne.Theme {
		return fdtheme.New(fdtheme.SchemeByName(a.Scheme), fdtheme.Options{Mono: mono, TextSize: 17})
	}
	s := onScreen(t, o)
	require.Same(t, got, s)
	require.Equal(t, []string{"create", "build"}, order)
	th, ok := s.App.Settings().Theme().(fdtheme.Theme)
	require.True(t, ok)
	require.InDelta(t, 17, th.TextSize(), 0.001, "the hook's theme is applied at start")

	a := s.Appearance()
	a.Scheme = "macOS Light"
	s.SetAppearance(a)
	th, _ = s.App.Settings().Theme().(fdtheme.Theme)
	require.Equal(t, "macOS Light", th.Palette().Name)
	require.InDelta(t, 17, th.TextSize(), 0.001, "and again on SetAppearance")
}

func TestTypedKeysNotTakenByTheShellReachTheProgram(t *testing.T) {
	var keys []fyne.KeyName
	invalidated := 0
	o := testOptions(NewSection("A", nil, func(*Shell) fyne.CanvasObject { return widget.NewLabel("a") }))
	o.OnTypedKey = func(e *fyne.KeyEvent) { keys = append(keys, e.Name) }
	o.OnInvalidate = func(*Shell) { invalidated++ }
	s := onScreen(t, o)
	handler := s.Window.Canvas().OnTypedKey()
	require.NotNil(t, handler)
	handler(&fyne.KeyEvent{Name: fyne.KeyF5})
	handler(&fyne.KeyEvent{Name: fyne.KeyRight})
	handler(&fyne.KeyEvent{Name: fyne.KeySpace})
	require.Equal(t, 1, invalidated, "F5 is the shell's")
	require.Equal(t, []fyne.KeyName{fyne.KeyRight, fyne.KeySpace}, keys, "the rest are the program's")
}

// The section must fit the width it is given. A long note in an unwrapped
// label sets the section's minimum width to the whole line, and the shell's
// two-axis scroller then scrolls sideways instead of wrapping: the Appearance
// section shipped that way once and showed a horizontal scrollbar on Windows.
func TestTheAppearanceSectionFitsANarrowViewportWithoutSidewaysScroll(t *testing.T) {
	fdtheme.RescanFonts(t.TempDir())
	t.Cleanup(func() { fdtheme.RescanFonts() })
	s := headless(t, testOptions(AppearanceSection("$ make test")))
	for _, width := range []float32{700, 500} {
		o := s.Current().Build(s)
		w := test.NewWindow(o)
		w.Resize(fyne.NewSize(width, 900))
		require.LessOrEqual(t, o.MinSize().Width, width, "at %g wide", width)
		w.Close()
	}
}
