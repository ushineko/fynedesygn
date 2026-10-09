package wizard

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"

	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/require"

	"github.com/ushineko/fynedesygn/fynetest"
)

func TestTheBarIsOptInAndKeepsItsHeight(t *testing.T) {
	plain := Progress("Installing", []string{"Copy"}, "Done.", func(context.Context, *Reporter) error { return nil })
	w := headless(t, plain, Finish("Done", ""))
	require.Empty(t, fynetest.All[*widget.ProgressBar](w.Content()), "no bar without WithBar")

	var before float32
	var p *ProgressPage
	p = Progress("Installing", []string{"Copy"}, "Done.", func(_ context.Context, r *Reporter) error {
		before = p.count.widget().MinSize().Height
		r.Progress(1.5, "3 of 3 files · 12 MB of 12 MB", "bin/hello")
		return nil
	}).WithBar()
	w = headless(t, p, Finish("Done", ""))
	require.Len(t, fynetest.All[*widget.ProgressBar](w.Content()), 1)
	require.Equal(t, 1.0, p.count.bar.Value, "the fraction is clamped")
	require.Equal(t, "3 of 3 files · 12 MB of 12 MB", p.count.status.Text)
	require.Equal(t, "bin/hello", p.count.item.Text)
	require.Equal(t, before, p.count.widget().MinSize().Height, "counting moves nothing")
}

func TestALongItemKeepsItsStartAndItsName(t *testing.T) {
	long := "lib/python3.12/site-packages/" + strings.Repeat("deeply/nested/", 10) + "module_with_a_long_name.py"
	got := middle(long, ItemRunes)
	require.Equal(t, ItemRunes, utf8.RuneCountInString(got))
	require.True(t, strings.HasPrefix(got, "lib/python3.12/"), got)
	require.True(t, strings.HasSuffix(got, "module_with_a_long_name.py"), got)
	require.Contains(t, got, "…")
	require.Equal(t, "short.py", middle("short.py", ItemRunes))
}

// The job reports once per file, as an installer of thousands of files
// does; with a window the page draws on its timer, not per report.
//
// The test driver's fyne.Do runs a function on the calling goroutine, where
// the real driver runs it on its one UI thread. The test stands a mutex in
// for that thread, so the timer and the job's end do not draw at once.
func TestThousandsOfReportsCostAFewDraws(t *testing.T) {
	const files = 10000
	var ui sync.Mutex
	done := make(chan struct{})
	p := Progress("Installing", []string{"Copy"}, "Done.", func(_ context.Context, r *Reporter) error {
		defer close(done)
		for i := 1; i <= files; i++ {
			r.Progress(float64(i)/files, fmt.Sprintf("%d of %d files", i, files), fmt.Sprintf("lib/file%05d.py", i))
		}
		return nil
	}).WithBar()
	w := NewIn(fynetest.App(t), Options{Name: "Test", Pages: []Page{&page{title: "A", valid: true}, p, Finish("Done", "")}})
	t.Cleanup(w.Window.Close)
	w.do = func(fn func()) { ui.Lock(); defer ui.Unlock(); fn() }
	ui.Lock()
	w.Next() // starts the job on its goroutine
	ui.Unlock()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("the job did not finish")
	}
	require.Eventually(t, func() bool {
		ui.Lock()
		defer ui.Unlock()
		return p.count.status.Text == fmt.Sprintf("%d of %d files", files, files) && w.succeeded
	}, 5*time.Second, 20*time.Millisecond, "the last value is drawn when the job ends")
	p.count.mu.Lock()
	draws := p.count.draws
	p.count.mu.Unlock()
	require.Less(t, draws, 100, "10,000 reports, %d draws", draws)
}
