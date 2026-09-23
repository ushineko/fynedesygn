package dialogs

import (
	"bytes"
	"image"
	"image/draw"
	"image/png"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"

	"github.com/ushineko/fynedesygn/fynetest"
	fdtheme "github.com/ushineko/fynedesygn/theme"
)

// TestThePreviewRedrawsInTheHighlightedFamily is the whole point of the
// chooser: moving the highlight must change what the sample looks like.
func TestThePreviewRedrawsInTheHighlightedFamily(t *testing.T) {
	w := testWindow(t)
	base := fdtheme.DefaultAppearance()
	ChooseFont(w, "Interface font", base.Font, base, false, func(string) {})
	w.Resize(fyne.NewSize(900, 700))

	/*
		Only the sample's own rectangle.

		A whole-canvas comparison would pass on the list's selection
		highlight alone, which moves whether or not the sample redraws — and
		the sample redrawing is the entire point of the chooser.
	*/
	/*
		The prose line, whose text is the same whatever family is
		highlighted.

		Cropping the whole sample would pass on the heading alone: the
		heading is the family's *name*, so its pixels change when the
		highlight moves even if not one glyph is drawn in a different font.
	*/
	var sample fyne.CanvasObject
	for _, line := range fynetest.All[*canvas.Text](overlay(t, w)) {
		if line.Text == samplePangram {
			sample = line
			break
		}
	}
	if sample == nil {
		t.Fatal("the chooser draws no sample line")
	}
	shot := func() []byte {
		full := w.Canvas().Capture()
		at := fyne.CurrentApp().Driver().AbsolutePositionForObject(sample)
		size := sample.Size()
		box := image.Rect(int(at.X), int(at.Y),
			int(at.X+size.Width), int(at.Y+size.Height))
		if box.Empty() {
			t.Fatal("the sample has no size")
		}
		cropped := image.NewRGBA(box)
		draw.Draw(cropped, box, full, box.Min, draw.Src)
		var buf bytes.Buffer
		if err := png.Encode(&buf, cropped); err != nil {
			t.Fatal(err)
		}
		return buf.Bytes()
	}

	list, ok := fynetest.First[*widget.List](overlay(t, w))
	if !ok {
		t.Fatal("the chooser has no list")
	}

	names := fdtheme.FontNames()
	other := -1
	for i, n := range names {
		if n != base.Font && fdtheme.PreviewFont(n) != nil {
			other = i
			break
		}
	}
	if other < 0 {
		t.Skip("no second readable family on this machine")
	}
	t.Logf("highlighting %q (index %d of %d)", names[other], other, len(names))

	before := shot()
	list.Select(other)
	w.Canvas().Capture()
	after := shot()

	if bytes.Equal(before, after) {
		t.Error("the sample looks the same after highlighting another family")
	}
}

/*
The sample names the file it is drawn from.

Because the eye cannot always answer "is this really a different font?".
Families differ by a hair at a sentence's size, a proportional variant of a
monospace family reads like an ordinary sans, and an unreadable family falls
back silently — three ways for a working preview to look broken.
*/
func TestTheSampleNamesTheFaceItIsDrawnIn(t *testing.T) {
	w := testWindow(t)
	base := fdtheme.DefaultAppearance()
	ChooseFont(w, "Interface font", base.Font, base, false, func(string) {})

	list, ok := fynetest.First[*widget.List](overlay(t, w))
	if !ok {
		t.Fatal("the chooser has no list")
	}

	names := fdtheme.FontNames()
	said := func() string {
		for _, text := range fynetest.Texts(overlay(t, w)) {
			if strings.HasPrefix(text, "drawn in ") || strings.Contains(text, "no readable face") {
				return text
			}
		}
		return ""
	}

	var readable int
	for i, n := range names {
		if fdtheme.PreviewFont(n) != nil {
			readable = i
			break
		}
	}
	if readable == 0 {
		t.Skip("no readable family on this machine")
	}

	list.Select(readable)
	got := said()
	if got == "" {
		t.Fatalf("the sample says nothing about what it is drawing: %v",
			fynetest.Texts(overlay(t, w)))
	}
	// It names a file, and the file is the family's rather than the
	// interface's — which is the whole claim the preview makes.
	if !strings.HasSuffix(got, ".ttf") && !strings.HasSuffix(got, ".otf") {
		t.Errorf("it named no face: %q", got)
	}
	face := fdtheme.PreviewFont(names[readable])
	if face == nil {
		t.Fatal("the family became unreadable mid-test")
	}
	want := face.Face(fyne.TextStyle{})
	if want == nil {
		t.Fatal("the family has no regular face")
	}
	if !strings.Contains(got, want.Name()) {
		t.Errorf("the sample says %q, but %s draws from %s",
			got, names[readable], want.Name())
	}
}

/*
The preview follows the keyboard, not only the click.

widget.List draws two highlights and reports them separately: the arrow keys
move currentHighlight and call OnHighlighted, while OnSelected fires on a
click or Space. Previewing on selection alone left the sample sitting on
whatever had last been clicked while a bright highlight moved down the list —
the list saying one family and the sample showing another.
*/
func TestTheKeyboardMovesThePreview(t *testing.T) {
	w := testWindow(t)
	base := fdtheme.DefaultAppearance()
	ChooseFont(w, "Interface font", base.Font, base, false, func(string) {})

	list, ok := fynetest.First[*widget.List](overlay(t, w))
	if !ok {
		t.Fatal("the chooser has no list")
	}
	if list.OnHighlighted == nil {
		t.Fatal("the chooser does not listen for the keyboard's highlight")
	}

	drawn := func() string {
		for _, text := range fynetest.Texts(overlay(t, w)) {
			if strings.HasPrefix(text, "Sample — ") {
				return strings.TrimPrefix(text, "Sample — ")
			}
		}
		return ""
	}

	names := fdtheme.FontNames()
	before := drawn()

	// What the arrow keys do: move the highlight without selecting.
	list.OnHighlighted(2)

	after := drawn()
	if after == before {
		t.Errorf("the sample stayed on %q when the highlight moved", before)
	}
	if after != names[2] {
		t.Errorf("the sample shows %q, the highlight is on %q", after, names[2])
	}
}

/*
A family that cannot draw the sample says so.

Most families on a Linux machine are script fonts — Noto ships one per
writing system — and most of those carry punctuation and digits and not one
Latin letter. Drawing a Latin sentence in one shows another font's glyphs
standing in: identical for every such family, and indistinguishable from a
preview that has stopped working. Which is exactly how it was read.
*/
func TestAFamilyThatCannotDrawTheSampleSaysSo(t *testing.T) {
	w := testWindow(t)
	base := fdtheme.DefaultAppearance()
	ChooseFont(w, "Interface font", base.Font, base, false, func(string) {})

	list, ok := fynetest.First[*widget.List](overlay(t, w))
	if !ok {
		t.Fatal("the chooser has no list")
	}
	names := fdtheme.FontNames()

	// A family with no Latin letters at all, if this machine has one.
	letterless := -1
	for i, n := range names {
		f := fdtheme.PreviewFont(n)
		if f != nil && f.Missing(samplePangram) >= len([]rune(samplePangram)) {
			letterless = i
			break
		}
	}
	if letterless < 0 {
		t.Skip("every family on this machine draws Latin")
	}

	list.OnHighlighted(letterless)
	said := strings.Join(fynetest.Texts(overlay(t, w)), "\n")
	if !strings.Contains(said, "no Latin letters") {
		t.Errorf("%s draws no Latin and the sample does not say so:\n%s",
			names[letterless], said)
	}
	// And it shows what the family *can* draw, in the family's own face.
	own := fdtheme.PreviewFont(names[letterless]).Covered(sampleShapes)
	if own != "" && !strings.Contains(said, own) {
		t.Errorf("the sample does not show the %q the family does have", own)
	}
}

// A family that draws the sample in full says that instead, and names it.
func TestAFamilyThatDrawsTheSampleNamesItsFace(t *testing.T) {
	w := testWindow(t)
	base := fdtheme.DefaultAppearance()
	ChooseFont(w, "Interface font", base.Font, base, false, func(string) {})

	list, _ := fynetest.First[*widget.List](overlay(t, w))
	names := fdtheme.FontNames()
	full := -1
	for i, n := range names {
		f := fdtheme.PreviewFont(n)
		if f != nil && f.Missing(samplePangram) == 0 {
			full = i
			break
		}
	}
	if full < 0 {
		t.Skip("no family on this machine draws the whole sample")
	}

	list.OnHighlighted(full)
	said := strings.Join(fynetest.Texts(overlay(t, w)), "\n")
	if !strings.Contains(said, "drawn in ") || strings.Contains(said, "missing") {
		t.Errorf("%s draws the whole sample, and the sample says:\n%s", names[full], said)
	}
}

/*
Two families previewed as the monospace face look different from each other.

They did not. Fyne resolves monospace text through a list of faces — the
theme's, its own bundled monospace, and a system font for the generic
"monospace" family — and what came out was not the face the theme had handed
it, so every family in the monospace chooser drew the same glyphs. The sample
takes the plain path now.
*/
func TestTheMonospaceChooserPreviewsTheFamily(t *testing.T) {
	base := fdtheme.DefaultAppearance()
	base.Font, base.Mono = "Noto Sans", fdtheme.DefaultFontName

	var families []string
	for _, n := range fdtheme.FontNames() {
		f := fdtheme.PreviewFont(n)
		if f != nil && f.Missing(samplePangram) == 0 {
			families = append(families, n)
		}
		if len(families) == 2 {
			break
		}
	}
	if len(families) < 2 {
		t.Skip("fewer than two families on this machine draw the sample")
	}

	shot := func(family string) []byte {
		w := testWindow(t)
		w.SetContent(sampleBlock(family, true, base))
		w.Resize(fyne.NewSize(700, 220))
		var buf bytes.Buffer
		if err := png.Encode(&buf, w.Canvas().Capture()); err != nil {
			t.Fatal(err)
		}
		return buf.Bytes()
	}

	if bytes.Equal(shot(families[0]), shot(families[1])) {
		t.Errorf("%q and %q draw the same sample in the monospace chooser",
			families[0], families[1])
	}
}

/*
And a proportional family says so when it is offered as the monospace face.

A sentence of prose will not reveal it — that is the whole trouble — and
choosing one is not a matter of taste but a mistake: the places that asked for
monospace did so because alignment was the point.
*/
func TestAProportionalFamilySaysSoInTheMonospaceChooser(t *testing.T) {
	base := fdtheme.DefaultAppearance()
	proportional, monospaced := "", ""
	for _, n := range fdtheme.FontNames() {
		f := fdtheme.PreviewFont(n)
		if f == nil || f.Missing(samplePangram) > 0 {
			continue
		}
		if f.IsMonospace() && monospaced == "" {
			monospaced = n
		}
		if !f.IsMonospace() && proportional == "" {
			proportional = n
		}
	}
	if proportional == "" || monospaced == "" {
		t.Skip("this machine has no pair to compare")
	}
	t.Logf("proportional %q, monospaced %q", proportional, monospaced)

	if got := drawnIn(proportional, true, base, fdtheme.PreviewFont(proportional)); !strings.Contains(got, "not a monospace family") {
		t.Errorf("%s is proportional and the sample says %q", proportional, got)
	}
	if got := drawnIn(monospaced, true, base, fdtheme.PreviewFont(monospaced)); strings.Contains(got, "not a monospace") {
		t.Errorf("%s is monospaced and the sample says %q", monospaced, got)
	}
}

/*
The monospace chooser offers only monospace families.

A proportional family chosen as the monospace face is not a matter of taste
but a mistake: the places that asked for monospace did so because alignment
was the point. And it cannot be filtered by name — "Meslo LGLDZ Nerd Font
Propo" is the proportional one.
*/
func TestTheMonospaceChooserOffersOnlyMonospaceFamilies(t *testing.T) {
	w := testWindow(t)
	base := fdtheme.DefaultAppearance()
	ChooseFont(w, "Monospace font", fdtheme.DefaultFontName, base, true, func(string) {})

	list, ok := fynetest.First[*widget.List](overlay(t, w))
	if !ok {
		t.Fatal("the chooser has no list")
	}
	offered := fdtheme.MonospaceNames()
	if list.Length() != len(offered) {
		t.Errorf("the chooser lists %d families, %d are monospace",
			list.Length(), len(offered))
	}
	if len(offered) >= len(fdtheme.FontNames()) {
		t.Skip("every family on this machine is monospace")
	}

	// Every one of them really is, and the interface chooser still offers
	// all of them.
	for _, name := range offered {
		f := fdtheme.PreviewFont(name)
		if f != nil && !f.IsMonospace() {
			t.Errorf("%q is offered as monospace and is not", name)
		}
	}
}

/*
And what is already chosen stays on the list, whatever it is.

A picker that cannot show the current setting is a picker that has lost it —
and the setting may predate the filter, or have been written by hand.
*/
func TestTheMonospaceChooserKeepsTheCurrentFamily(t *testing.T) {
	proportional := ""
	for _, name := range fdtheme.FontNames() {
		f := fdtheme.PreviewFont(name)
		if f != nil && !f.IsMonospace() {
			proportional = name
			break
		}
	}
	if proportional == "" {
		t.Skip("every family on this machine is monospace")
	}

	w := testWindow(t)
	base := fdtheme.DefaultAppearance()
	ChooseFont(w, "Monospace font", proportional, base, true, func(string) {})

	said := strings.Join(fynetest.Texts(overlay(t, w)), "\n")
	if !strings.Contains(said, proportional) {
		t.Errorf("the chooser dropped the family that is set, %q", proportional)
	}
	// And says why it should not have been chosen.
	if !strings.Contains(said, "not a monospace family") {
		t.Errorf("it does not say %q is proportional:\n%s", proportional, said)
	}
}
