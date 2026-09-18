package forms

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/require"

	"github.com/ushineko/fynedesygn/fynetest"
)

func testForm(t *testing.T) (*Form, *[]string) {
	t.Helper()
	fynetest.App(t)
	w := test.NewWindow(nil)
	t.Cleanup(w.Close)
	f := New(
		Entry("name", "Name", "a name"),
		Select("mode", "Mode", []string{"fast", "safe"}),
		Check("enabled", "Enabled"),
		Path(w, "dir", "Directory", true),
		Numeric("jobs", "Jobs", 1, 16),
	)
	var changes []string
	f.OnChange = func(k, v string) { changes = append(changes, k+"="+v) }
	return f, &changes
}

func TestSetDoesNotFireOnChangeButTypingDoes(t *testing.T) {
	f, changes := testForm(t)
	f.Set(map[string]string{"name": "alpha", "mode": "safe", "enabled": "true", "dir": "/tmp", "jobs": "4", "unknown": "x"})
	require.Empty(t, *changes, "a Set is not a user change")
	require.Equal(t, map[string]string{"name": "alpha", "mode": "safe", "enabled": "true", "dir": "/tmp", "jobs": "4"}, f.Values())

	fynetest.FindEntry(f.Field("name").Widget).SetText("beta")
	f.Field("mode").Widget.(*widget.Select).SetSelected("fast")
	f.Field("enabled").Widget.(*widget.Check).SetChecked(false)
	fynetest.FindEntry(f.Field("jobs").Widget).SetText("99")
	fynetest.FindEntry(f.Field("jobs").Widget).SetText("8")
	require.Equal(t, []string{"name=beta", "mode=fast", "enabled=false", "jobs=8"}, *changes, "out-of-range 99 is not reported")
	require.Nil(t, f.Field("nope"))
	require.Len(t, f.Fields(), 5)

	// A Revert is a Set: the typed values are gone at once.
	f.Set(map[string]string{"name": "alpha", "jobs": "4"})
	require.Equal(t, "alpha", f.Values()["name"])
	require.Len(t, *changes, 4)
	require.NotNil(t, f.Widget())
}

// Canary #2 (docs/fyne-quirks.md): Fyne's setters fire the change handlers,
// which is why Form.Set runs under a guard.
func TestSetCheckedFiresOnChanged(t *testing.T) {
	fynetest.App(t)
	fired := 0
	c := widget.NewCheck("", func(bool) { fired++ })
	c.SetChecked(true)
	require.Equal(t, 1, fired, "when this fails, Fyne has changed and the guard can be reconsidered")
	s := widget.NewSelect([]string{"a", "b"}, func(string) { fired++ })
	s.SetSelected("b")
	require.Equal(t, 2, fired)
}

func TestIntRangeAndNumericEntry(t *testing.T) {
	v := IntRange(1, 10)
	require.NoError(t, v("5"))
	require.NoError(t, v(" 10 "))
	require.Error(t, v("x"))
	require.Error(t, v("0"))
	require.Error(t, v("11"))

	fynetest.App(t)
	var got []int
	e := NumericEntry(1, 10, func(n int) { got = append(got, n) })
	e.SetText("3")
	e.SetText("30")
	e.SetText("abc")
	e.SetText("7")
	require.Equal(t, []int{3, 7}, got)
}

func TestSliderEntryCommitsOncePerGesture(t *testing.T) {
	fynetest.App(t)
	var commits []float64
	var invalid []string
	s := NewSliderEntry(SliderOptions{
		Min: 0, Max: 100, Step: 1, Value: 25,
		Commit:    func(v float64) { commits = append(commits, v) },
		OnInvalid: func(text string) { invalid = append(invalid, text) },
	})
	w := test.NewWindow(s.Widget())
	defer w.Close()
	w.Resize(fyne.NewSize(600, 60))
	require.Equal(t, "25", s.Entry().Text)

	// Dragging updates the text and commits nothing.
	s.Slider().OnChanged(60)
	require.Equal(t, "60", s.Entry().Text)
	require.Empty(t, commits)
	s.Slider().OnChangeEnded(60)
	require.Equal(t, []float64{60}, commits)
	require.InDelta(t, 60, s.Value(), 0.001)

	// Enter in the entry commits; a non-number restores and reports.
	s.Entry().OnSubmitted("80")
	require.Equal(t, []float64{60, 80}, commits)
	require.InDelta(t, 80, s.Slider().Value, 0.001)
	s.Entry().OnSubmitted("lots")
	require.Equal(t, []string{"lots"}, invalid)
	require.Equal(t, "80", s.Entry().Text)
	require.Len(t, commits, 2)

	// Out of range clamps; Set moves both without committing.
	s.Entry().OnSubmitted("500")
	require.InDelta(t, 100, commits[len(commits)-1], 0.001)
	s.Set(10)
	require.Equal(t, "10", s.Entry().Text)
	require.InDelta(t, 10, s.Slider().Value, 0.001)
	require.Len(t, commits, 3)
}

func TestGroupAndCustomField(t *testing.T) {
	fynetest.App(t)
	label := widget.NewLabel("custom")
	value := "one"
	c := Custom("c", "Custom", label, func() string { return value }, func(v string) { value = v })
	f := New(c)
	var got string
	f.OnChange = func(_, v string) { got = v }
	c.Notify("two")
	require.Equal(t, "two", got)
	f.Set(map[string]string{"c": "three"})
	require.Equal(t, "three", value)
	require.Equal(t, "two", got, "Set did not notify")

	g := Group("Limits", "The blurb.", widget.NewLabel("row"))
	text := fynetest.Text(g)
	require.Contains(t, text, "Limits")
	require.Contains(t, text, "The blurb.")
}
