package glance_test

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	fynetheme "fyne.io/fyne/v2/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ushineko/fynedesygn/fynetest"
	"github.com/ushineko/fynedesygn/glance"
	fdtheme "github.com/ushineko/fynedesygn/theme"
)

// labelOf is the row's label text as drawn.
func labelOf(t *testing.T, r *glance.Row, text string) *canvas.Text {
	t.Helper()
	for _, x := range fynetest.All[*canvas.Text](r.Object()) {
		if x.Text == text {
			return x
		}
	}
	require.Failf(t, "no label", "%q is not drawn", text)
	return nil
}

// Spec 087 R1. A bold label is bold where the theme has the face, and the
// row's value is not touched; turned off, it is regular again.
func TestALabelCanBeBold(t *testing.T) {
	_ = fynetest.App(t)
	r := glance.NewRow("Recent", "")
	r.Set(glance.Measured("12"))
	label := labelOf(t, r, "Recent")
	require.False(t, label.TextStyle.Bold)

	// The design system's theme has a bold face.
	r.SetTheme(fdtheme.New(fdtheme.BreezeDark, fdtheme.Options{}))
	r.SetLabelBold(true)
	assert.True(t, label.TextStyle.Bold, "the label is not bold in a theme with the face")
	assert.False(t, r.Reading().Stale)
	for _, x := range fynetest.All[*canvas.Text](r.Object()) {
		if x.Text == "12" {
			assert.False(t, x.TextStyle.Bold, "the value went bold with the label")
		}
	}

	r.SetLabelBold(false)
	assert.False(t, label.TextStyle.Bold)
}

// Spec 087 R2. An indent moves the label in by its steps and leaves the value
// against the right edge.
func TestAnIndentMovesTheLabelAndNotTheValue(t *testing.T) {
	_ = fynetest.App(t)
	flat, deep := glance.NewRow("hayami", ""), glance.NewRow("hayami", "")
	flat.Set(glance.Measured("3c"))
	deep.Set(glance.Measured("3c"))
	deep.SetIndent(2)
	w := test.NewWindow(container.NewVBox(flat.Object(), deep.Object()))
	t.Cleanup(w.Close)
	w.SetPadded(false)
	w.Resize(fyne.NewSize(300, 100))

	drv := fyne.CurrentApp().Driver()
	x := func(o fyne.CanvasObject) float32 { return drv.AbsolutePositionForObject(o).X }
	step := glance.IndentStep * fyne.CurrentApp().Settings().Theme().Size(fynetheme.SizeNameText)
	assert.InDelta(t, x(labelOf(t, flat, "hayami"))+2*step, x(labelOf(t, deep, "hayami")), 0.01)
	right := func(r *glance.Row) float32 {
		v := labelOf(t, r, "3c")
		return x(v) + v.Size().Width
	}
	assert.InDelta(t, right(flat), right(deep), 0.01, "the indent moved the value")

	deep.SetIndent(0)
	w.Content().Refresh()
	assert.InDelta(t, x(labelOf(t, flat, "hayami")), x(labelOf(t, deep, "hayami")), 0.01)
}

// Spec 087 R3. An indent is in text sizes, so a larger face indents further.
func TestAnIndentFollowsTheFace(t *testing.T) {
	_ = fynetest.App(t)
	r := glance.NewRow("hayami", "")
	r.SetIndent(1)
	before := r.Object().MinSize().Width
	plain := glance.NewRow("hayami", "")
	base := plain.Object().MinSize().Width

	big := bigFace{fyne.CurrentApp().Settings().Theme()}
	r.SetTheme(big)
	plain.SetTheme(big)
	assert.InDelta(t, 2*(before-base), r.Object().MinSize().Width-plain.Object().MinSize().Width, 0.5)
}

// bigFace is a theme with twice the text size.
type bigFace struct{ fyne.Theme }

func (t bigFace) Size(n fyne.ThemeSizeName) float32 {
	if n == fynetheme.SizeNameText {
		return 2 * t.Theme.Size(n)
	}
	return t.Theme.Size(n)
}
