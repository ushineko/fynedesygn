package wizard

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/require"

	"github.com/ushineko/fynedesygn/forms"
	"github.com/ushineko/fynedesygn/fynetest"
	"github.com/ushineko/fynedesygn/logpane"
	"github.com/ushineko/fynedesygn/steps"
)

// page is a test page with switchable optional behaviour.
type page struct {
	title    string
	valid    bool
	skip     bool
	leaveErr error
	entered  int
}

func (p *page) Title() string                   { return p.title }
func (p *page) Build(*Wizard) fyne.CanvasObject { return widget.NewLabel(p.title) }
func (p *page) Valid() bool                     { return p.valid }
func (p *page) Skip() bool                      { return p.skip }
func (p *page) Leave(*Wizard, bool) error       { return p.leaveErr }
func (p *page) Enter(*Wizard)                   { p.entered++ }

func headless(t *testing.T, pages ...Page) *Wizard {
	t.Helper()
	return Headless(fynetest.App(t), Options{Name: "Test", Pages: pages})
}

func TestNextWaitsForAValidPage(t *testing.T) {
	a, b := &page{title: "A"}, &page{title: "B", valid: true}
	w := headless(t, a, b)
	require.True(t, w.next.Disabled())
	w.Next()
	require.Equal(t, a, w.Current(), "a disabled Next does nothing")

	a.valid = true
	w.Revalidate()
	require.False(t, w.next.Disabled())
	w.Next()
	require.Equal(t, b, w.Current())
}

func TestTheLicenceMustBeAccepted(t *testing.T) {
	l := Licence("Some terms.")
	w := headless(t, l, &page{title: "After", valid: true})
	require.True(t, w.next.Disabled())
	fynetest.FindCheck(w.Content()).SetChecked(true)
	require.False(t, w.next.Disabled())
}

func TestALeaverErrorKeepsThePageAndShowsOnTheMessageLine(t *testing.T) {
	a := &page{title: "A", valid: true, leaveErr: errors.New("the disk is full")}
	w := headless(t, a, &page{title: "B", valid: true})
	w.Next()
	require.Equal(t, a, w.Current())
	require.Equal(t, "the disk is full", w.Message())
}

func TestSkippedPagesArePassedInBothDirections(t *testing.T) {
	a, b, c := &page{title: "A", valid: true}, &page{title: "B", valid: true, skip: true}, &page{title: "C", valid: true}
	w := headless(t, a, b, c)
	w.Next()
	require.Equal(t, c, w.Current())
	require.Equal(t, 0, b.entered)
	require.Equal(t, "skipped", w.list.Steps()[1].Note)
	w.Back()
	require.Equal(t, a, w.Current())
}

func TestTheButtonRowNeverChangesWidth(t *testing.T) {
	w := headless(t,
		Welcome("Welcome", "Hello."),
		Summary("Ready", "Install", func() []Fact { return []Fact{{"Into", "/opt/x"}} }),
		Progress("Installing", []string{"Copy"}, "Installed.", func(context.Context, *Reporter) error { return nil }),
		Finish("Done", "All done.", widget.NewCheck("Launch now", nil)),
	)
	width := func() float32 { return w.buttons.MinSize().Width }
	want := width()
	for range 3 {
		w.Next()
		require.Equal(t, want, width(), "on %s, Next reads %q", w.Current().Title(), w.next.Text)
	}
	require.Equal(t, LabelFinish, w.next.Text)
}

func TestASuccessfulJobFinishesItsStepsAndEnablesNext(t *testing.T) {
	p := Progress("Installing", []string{"Copy", "Link"}, "Installed.", func(_ context.Context, r *Reporter) error {
		for i := range 2 {
			r.Advance(i, "working")
			r.Log(logpane.Info, fmt.Sprintf("step %d", i))
			r.Finish(i, "ok")
		}
		return nil
	})
	w := headless(t, p, Finish("Done", ""))
	for _, s := range p.Steps().Steps() {
		require.Equal(t, steps.Done, s.State, s.Name)
	}
	require.False(t, w.next.Disabled())
	require.True(t, w.back.Disabled(), "the point of no return")
	require.Equal(t, "Installed.", w.Message())
	require.Equal(t, 2, p.Log().Len())
}

func TestAFailedJobMarksTheStepAndCancelBecomesClose(t *testing.T) {
	p := Progress("Installing", []string{"Copy", "Link"}, "Installed.", func(_ context.Context, r *Reporter) error {
		r.Advance(0, "")
		r.Finish(0, "ok")
		r.Advance(1, "")
		return errors.New("permission denied")
	})
	w := headless(t, &page{title: "A", valid: true}, p, Finish("Done", ""))
	w.Next()
	require.Equal(t, steps.Failed, p.Steps().Steps()[1].State)
	require.Equal(t, "Failed: permission denied", w.Message())
	require.True(t, w.next.Disabled())
	require.True(t, w.back.Disabled())
	require.Equal(t, LabelClose, w.cancel.Text)
	require.False(t, w.cancel.Disabled())

	w.Cancel()
	require.Equal(t, Failed, w.Result().Outcome)
	require.EqualError(t, w.Result().Err, "permission denied")
}

func TestCancelDuringAJobAsksStopsItAndClosesOnlyWhenItReturns(t *testing.T) {
	var w *Wizard
	asked := 0
	sawDone := false
	p := Progress("Installing", []string{"Copy"}, "", func(ctx context.Context, r *Reporter) error {
		r.Advance(0, "")
		w.Cancel() // the user presses Cancel, or the title bar's close, mid-job
		require.Equal(t, 1, asked)
		require.False(t, w.closed, "the window waits for the job")
		<-ctx.Done()
		sawDone = true
		return fmt.Errorf("copying: %w", ctx.Err())
	})
	w = Headless(fynetest.App(t), Options{Name: "Test", Pages: []Page{&page{title: "A", valid: true}, p}})
	w.confirm = func(_, _, _ string, do func()) { asked++; do() }
	w.Next()
	require.True(t, sawDone)
	require.True(t, w.closed)
	require.Equal(t, Cancelled, w.Result().Outcome)
	require.Equal(t, steps.Cancelled, p.Steps().Steps()[0].State)
}

func TestAJobThatFinishesAsCancelArrivesIsFinished(t *testing.T) {
	var w *Wizard
	p := Progress("Installing", []string{"Copy"}, "Installed.", func(context.Context, *Reporter) error {
		w.Cancel()
		return nil // the work was already done
	})
	w = headless(t, &page{title: "A", valid: true}, p, Finish("Done", ""))
	w.Next()
	require.Equal(t, Finished, w.Result().Outcome, "an install that happened is not reported as cancelled")
}

func TestResults(t *testing.T) {
	t.Run("finished, with the finish page's checks", func(t *testing.T) {
		launch := widget.NewCheck("Launch now", nil)
		w := headless(t, &page{title: "A", valid: true}, Finish("Done", "", launch))
		w.Next()
		launch.SetChecked(true)
		w.Next()
		require.Equal(t, Result{Outcome: Finished, Checks: map[string]bool{"Launch now": true}}, w.Result())
	})
	t.Run("cancelled before anything ran", func(t *testing.T) {
		w := headless(t, &page{title: "A", valid: true}, Finish("Done", ""))
		w.Cancel()
		require.Equal(t, Cancelled, w.Result().Outcome)
	})
}

func TestCancelAsksOnlyWhenAPageHoldsAChange(t *testing.T) {
	d := Directory("Where", "Install into", "/opt/x", nil)
	w := Headless(fynetest.App(t), Options{Name: "Test", Pages: []Page{d, Finish("Done", "")}})
	asked := 0
	w.confirm = func(_, _, _ string, do func()) { asked++; do() }
	fynetest.FindEntry(w.Content()).SetText("/opt/y")
	w.Cancel()
	require.Equal(t, 1, asked)
	require.Equal(t, Cancelled, w.Result().Outcome)
}

func TestADirectoryPageShowsWhyItIsInvalid(t *testing.T) {
	d := Directory("Where", "Install into", "", func(s string) error {
		if s == "" {
			return errors.New("choose a directory")
		}
		return nil
	})
	w := headless(t, d, Finish("Done", ""))
	require.True(t, w.next.Disabled())
	require.NotNil(t, fynetest.FindLabel(w.Content(), "choose a directory"))
	fynetest.FindEntry(w.Content()).SetText("/opt/x")
	require.False(t, w.next.Disabled())
	require.Equal(t, "/opt/x", d.Value())
}

func TestAFormsInitialValuesAreSetWhenThePageIsBuilt(t *testing.T) {
	f := forms.New(forms.Check("menu", "Add a launcher entry"))
	p := Form("Options", f, map[string]string{"menu": "true"}, nil)
	w := headless(t, &page{title: "A", valid: true}, p)
	require.Equal(t, "false", f.Values()["menu"], "nothing touches the widgets before Build")
	w.Next()
	require.Equal(t, "true", f.Values()["menu"])
	require.False(t, p.Changed(), "the initial values are the starting point")
}

// NewIn is the path a real program takes, through a window; Headless never
// sets window content, so this is the test that would see a construction
// order that only a window exposes (found at the desk: a nil scroller
// measured by SetContent).
func TestNewInBuildsAWindowTheTestDriverCanLayOut(t *testing.T) {
	a := fynetest.App(t)
	w := NewIn(a, Options{Name: "Test", Pages: []Page{Welcome("Welcome", "Hello."), Finish("Done", "")}})
	t.Cleanup(w.Window.Close)
	require.Equal(t, w.Content(), w.Window.Content())
	require.Positive(t, w.Window.Content().MinSize().Width)
	w.Next()
	require.Equal(t, "Done", w.Current().Title())
}
