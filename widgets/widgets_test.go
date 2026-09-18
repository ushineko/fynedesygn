package widgets

import (
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/require"

	fd "github.com/ushineko/fynedesygn"
)

// textIn walks a canvas object tree headlessly and collects the text of every
// label, button and canvas text it finds, so a test can assert what a
// constructor put on screen without a window.
func textIn(o fyne.CanvasObject) []string {
	var out []string
	var walk func(o fyne.CanvasObject)
	walk = func(o fyne.CanvasObject) {
		switch v := o.(type) {
		case *widget.Label:
			out = append(out, v.Text)
		case *widget.Button:
			out = append(out, v.Text)
		case *canvas.Text:
			out = append(out, v.Text)
		case *fyne.Container:
			for _, c := range v.Objects {
				walk(c)
			}
			return
		}
		if w, ok := o.(fyne.Widget); ok {
			for _, c := range test.WidgetRenderer(w).Objects() {
				walk(c)
			}
		}
	}
	walk(o)
	return out
}

func TestEveryConstructorRendersHeadlesslyWithItsText(t *testing.T) {
	test.NewTempApp(t)
	cases := []struct {
		name  string
		build func() fyne.CanvasObject
		want  []string
	}{
		{"Dim", func() fyne.CanvasObject { return Dim("dim text") }, []string{"dim text"}},
		{"Sep", Sep, []string{"·"}},
		{"Wrapped", func() fyne.CanvasObject { return Wrapped("a long sentence") }, []string{"a long sentence"}},
		{"StatusText", func() fyne.CanvasObject { return StatusText("ok", fd.StatusGood) }, []string{"ok"}},
		{"Marker", func() fyne.CanvasObject { return Marker(fd.StatusBad) }, nil},
		{"Heading", func() fyne.CanvasObject { return Heading("Title", "blurb here") }, []string{"Title", "blurb here"}},
		{"Card", func() fyne.CanvasObject { return Card("Card title", Dim("body")) }, []string{"Card title", "body"}},
		{"Note", func() fyne.CanvasObject { return Note("a note", fd.StatusWarn) }, []string{"a note"}},
		{"FactRow", func() fyne.CanvasObject { return FactRow("label", "value", fd.StatusGood) }, []string{"label", "value"}},
		{"PlainRow", func() fyne.CanvasObject { return PlainRow("plain", "no verdict") }, []string{"plain", "no verdict"}},
		{"RowWithAction", func() fyne.CanvasObject {
			return RowWithAction(PlainRow("path", "/tmp"), widget.NewButton("Open", nil))
		}, []string{"path", "/tmp", "Open"}},
		{"Action", func() fyne.CanvasObject {
			row, _ := Action("Do it", "consequences", "Go", false, nil)
			return row
		}, []string{"Do it", "consequences", "Go"}},
		{"AboutNote", func() fyne.CanvasObject { return AboutNote("About", "detail") }, []string{"About", "detail"}},
		{"FixedHeight", func() fyne.CanvasObject { return FixedHeight(Dim("tall"), 100) }, []string{"tall"}},
		{"FixedWidth", func() fyne.CanvasObject { return FixedWidth(Dim("wide"), 100) }, []string{"wide"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var o fyne.CanvasObject
			require.NotPanics(t, func() {
				o = tc.build()
				o.Resize(fyne.NewSize(400, 200))
				o.Refresh()
			})
			got := textIn(o)
			for _, w := range tc.want {
				require.Contains(t, got, w, "%s should show %q; saw %v", tc.name, w, got)
			}
		})
	}
}

func TestStatusTextAndNoteTakeTheStatusImportance(t *testing.T) {
	test.NewTempApp(t)
	require.Equal(t, widget.SuccessImportance, StatusText("x", fd.StatusGood).(*widget.Label).Importance)
	require.Equal(t, widget.WarningImportance, StatusText("x", fd.StatusWarn).(*widget.Label).Importance)
	require.Equal(t, widget.DangerImportance, StatusText("x", fd.StatusBad).(*widget.Label).Importance)
	require.Equal(t, widget.MediumImportance, StatusText("x", fd.StatusInfo).(*widget.Label).Importance)

	noteLabel := func(st fd.Status) *widget.Label {
		for _, o := range Note("n", st).(*fyne.Container).Objects {
			if l, ok := o.(*widget.Label); ok {
				return l
			}
		}
		t.Fatal("note has no label")
		return nil
	}
	require.Equal(t, widget.WarningImportance, noteLabel(fd.StatusWarn).Importance)
	require.Equal(t, widget.DangerImportance, noteLabel(fd.StatusBad).Importance)
	require.Equal(t, widget.LowImportance, noteLabel(fd.StatusGood).Importance)
	require.Equal(t, widget.LowImportance, noteLabel(fd.StatusInfo).Importance)
}

func TestDangerActionReturnsTheButtonInTheTree(t *testing.T) {
	test.NewTempApp(t)
	tapped := false
	row, btn := Action("Delete", "gone for good", "Delete", true, func() { tapped = true })
	require.Equal(t, widget.DangerImportance, btn.Importance)

	var found *widget.Button
	var walk func(o fyne.CanvasObject)
	walk = func(o fyne.CanvasObject) {
		if b, ok := o.(*widget.Button); ok {
			found = b
			return
		}
		if c, ok := o.(*fyne.Container); ok {
			for _, child := range c.Objects {
				walk(child)
			}
		}
	}
	walk(row)
	require.Same(t, btn, found, "the returned button is the one in the tree")
	test.Tap(found)
	require.True(t, tapped)

	_, plain := Action("Keep", "harmless", "Keep", false, nil)
	require.Equal(t, widget.MediumImportance, plain.Importance)
}

func TestFixedSizeWrappersReserveAtLeastTheRequestedDimension(t *testing.T) {
	test.NewTempApp(t)
	require.GreaterOrEqual(t, FixedHeight(widget.NewLabel("x"), 240).MinSize().Height, float32(240))
	require.GreaterOrEqual(t, FixedWidth(widget.NewLabel("x"), 190).MinSize().Width, float32(190))
	// A wrapper never shrinks the content below its own minimum.
	l := widget.NewLabel("a fairly long label that is wider than ten")
	require.GreaterOrEqual(t, FixedWidth(l, 10).MinSize().Width, l.MinSize().Width)
}

func TestHumanSizeUsesBinaryUnitsWithOneDecimal(t *testing.T) {
	cases := []struct {
		n    int64
		want string
	}{
		{0, "0 B"},
		{1023, "1023 B"},
		{1024, "1.0 KiB"},
		{3 << 19, "1.5 MiB"},
		{2 << 30, "2.0 GiB"},
	}
	for _, tc := range cases {
		require.Equal(t, tc.want, HumanSize(tc.n), "HumanSize(%d)", tc.n)
	}
}

func TestHumanAgoAtCoarsensToTheLargestUnit(t *testing.T) {
	now := time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		ago  time.Duration
		zero bool
		want string
	}{
		{zero: true, want: "never"},
		{ago: 30 * time.Second, want: "just now"},
		{ago: 5 * time.Minute, want: "5m ago"},
		{ago: 3 * time.Hour, want: "3h ago"},
		{ago: 3 * 24 * time.Hour, want: "3d ago"},
	}
	for _, tc := range cases {
		at := now.Add(-tc.ago)
		if tc.zero {
			at = time.Time{}
		}
		require.Equal(t, tc.want, HumanAgoAt(at, now))
	}
	require.Equal(t, "never", HumanAgo(time.Time{}))
	require.Equal(t, "just now", HumanAgo(time.Now()))
}

func TestOrNoneSubstitutesForBlankValues(t *testing.T) {
	require.Equal(t, "none", OrNone("", "none"))
	require.Equal(t, "none", OrNone("   ", "none"))
	require.Equal(t, "value", OrNone("value", "none"))
}

func TestImportanceForMapsEveryStatus(t *testing.T) {
	require.Equal(t, widget.MediumImportance, ImportanceFor(fd.StatusInfo))
	require.Equal(t, widget.SuccessImportance, ImportanceFor(fd.StatusGood))
	require.Equal(t, widget.WarningImportance, ImportanceFor(fd.StatusWarn))
	require.Equal(t, widget.DangerImportance, ImportanceFor(fd.StatusBad))
}

func TestStatusColorFollowsTheActiveTheme(t *testing.T) {
	test.NewTempApp(t)
	th := fyne.CurrentApp().Settings().Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()
	require.Equal(t, th.Color("success", v), StatusColor(fd.StatusGood))
	require.Equal(t, th.Color("warning", v), StatusColor(fd.StatusWarn))
	require.Equal(t, th.Color("error", v), StatusColor(fd.StatusBad))
	require.Equal(t, th.Color("primary", v), StatusColor(fd.StatusInfo))
}

func TestDimWrappedFitsTheWidthItIsGiven(t *testing.T) {
	test.NewTempApp(t)
	long := "A long explanatory note that would otherwise set the minimum width of its section to the whole unwrapped line and make the shell scroll sideways."
	o := DimWrapped(long)
	o.Resize(fyne.NewSize(300, 200))
	require.LessOrEqual(t, o.MinSize().Width, float32(300))
	require.Greater(t, Dim(long).MinSize().Width, float32(300), "Dim is the unwrapped label")
}
