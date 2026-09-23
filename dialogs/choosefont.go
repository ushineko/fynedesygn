package dialogs

import (
	"slices"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	fynetheme "fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	fdtheme "github.com/ushineko/fynedesygn/theme"
)

/*
ChooseFont offers the installed families and shows what the highlighted one
looks like, so a font can be judged before it is chosen.

A dropdown cannot do this. Fyne draws every widget in the app's own theme, so
the only way to render a name in its own face is to put it under a theme of
its own — and a list that did that per row would read every font file it drew.
This machine carries 311 families at 2.5 MB each: 778 MB to show a menu, and a
font cache that never gives it back (theme.TestLoadingEveryFamilyIsNotFree).

So the list is plain and the sample is not. Moving through the names redraws
one sample in one family, read and released as the highlight moves, and the
cost of looking through all of them is the cost of looking at one.

The choice is only made on Choose: browsing changes nothing, which is the
point — the alternative was committing a font to find out what it looked like
and committing another to get back.
*/
func ChooseFont(win fyne.Window, title, current string, base fdtheme.Appearance,
	mono bool, then func(name string)) {
	names := fdtheme.FontNames()
	if mono {
		/*
			Only the families that are actually monospace.

			A proportional family chosen as the monospace face is not a
			matter of taste but a mistake: the places that asked for
			monospace did so because alignment was the point. Offering 310
			families when 79 of them can do the job is offering 231 chances
			to make it.

			Measured, never read off the name — "Meslo LGLDZ Nerd Font
			Propo" is the proportional one — and what is already chosen stays
			on the list whatever it is, because a picker that cannot show the
			current setting is a picker that has lost it.
		*/
		names = withCurrent(fdtheme.MonospaceNames(), current)
	}
	ChooseFontFrom(win, title, names, current, base, mono, then)
}

// withCurrent makes sure a list offers what is already set, so the chooser
// opens on it rather than on nothing.
func withCurrent(names []string, current string) []string {
	if current == "" || slices.Contains(names, current) {
		return names
	}
	return append(names, current)
}

/*
ChooseFontFrom is ChooseFont over a list the caller assembles.

For a program that offers something besides the installed families — most
often an entry meaning "whatever the interface is using", for a surface whose
font is layered over the window's. A name the machine cannot read previews as
the appearance's own font, which is what such an entry would draw as anyway
(Appearance.PreviewTheme).
*/
func ChooseFontFrom(win fyne.Window, title string, names []string, current string,
	base fdtheme.Appearance, mono bool, then func(name string)) {
	shown := append([]string{}, names...)
	picked := current

	sample := container.NewStack()
	drawSample := func(name string) {
		if name == "" {
			return
		}
		sample.Objects = []fyne.CanvasObject{sampleBlock(name, mono, base)}
		sample.Refresh()
	}

	list := widget.NewList(
		func() int { return len(shown) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(i widget.ListItemID, o fyne.CanvasObject) {
			if label, ok := o.(*widget.Label); ok && i < len(shown) {
				label.SetText(shown[i])
			}
		},
	)
	/*
		One row means one thing: the family the sample is showing, and the one
		Choose will take.

		widget.List offers two callbacks and neither means what it sounds
		like. OnHighlighted fires when the *pointer passes over* a row as
		well as when the arrow keys move — so a preview driven by it followed
		the mouse on its way to the Choose button. And because the preview
		also selected, and Select returns early for the row already selected,
		the click that followed did nothing at all: the family under the
		pointer had chosen itself on the way past.

		So the pointer merely draws Fyne's own hover, the cursor is moved by
		the arrow keys and by clicks, and those two are the only things that
		change what is chosen. The guard is for the round trip, because Select
		calls back into this.
	*/
	syncing := false
	cursor := -1
	choose := func(i widget.ListItemID) {
		if syncing || i < 0 || i >= len(shown) {
			return
		}
		cursor, picked = i, shown[i]
		drawSample(picked)

		syncing = true
		list.Select(i)
		syncing = false
	}
	list.OnSelected = choose
	// Hover draws, and decides nothing.
	list.OnHighlighted = nil

	move := func(delta int) {
		next := cursor + delta
		if next < 0 || next >= len(shown) {
			return
		}
		list.Highlight(next)
		choose(next)
	}

	/*
		A filter, because 311 families is not a list anybody reads.

		It narrows on what is typed and keeps the selection when the chosen
		family survives the narrowing, so typing a few letters does not
		quietly change what Choose would take.
	*/
	filter := newFontFilter()
	filter.SetPlaceHolder("Filter…")
	filter.OnChanged = func(text string) {
		text = strings.ToLower(strings.TrimSpace(text))
		shown = shown[:0]
		for _, name := range names {
			if text == "" || strings.Contains(strings.ToLower(name), text) {
				shown = append(shown, name)
			}
		}
		list.Refresh()
		selectFamily(list, shown, picked)
	}

	/*
		The keyboard lives on the filter, and stays there.

		Somewhere has to hold it: widget.List spends the arrow keys itself
		and ignores Enter, and it takes the focus back on every click, so a
		key handler anywhere else is one click from silence. The filter is
		where a person's hands already are — and from here the arrows move
		the cursor and Enter takes what the sample is showing, which is what
		a list of 311 names needs to be usable at all.
	*/
	filter.up = func() { move(-1) }
	filter.down = func() { move(1) }

	drawSample(picked)
	selectFamily(list, shown, picked)
	if i := indexOf(shown, picked); i >= 0 {
		cursor = i
	}

	head := fyne.CanvasObject(filter)
	if mono {
		// Said plainly, because a list of 79 where the interface font's had
		// 310 is otherwise a list that looks broken.
		head = container.NewVBox(filter,
			caption("Monospace families only — measured, not chosen by name."))
	}

	body := container.NewBorder(head, sample, nil, nil, list)
	d := ConfirmWithBody(win, title, body, "Choose", func() { then(picked) })
	filter.accept = func() {
		d.Hide()
		then(picked)
	}
	d.Resize(RoomySize(win, FontChooserFloor))
	raise(win, d)
	// Focused, so the arrows and Enter work without a click first.
	if canvas := fyne.CurrentApp().Driver().CanvasForObject(list); canvas != nil {
		canvas.Focus(filter)
	}
}

// FontChooserFloor is the smallest the chooser will be: a list that shows
// enough names to scroll through, and a sample big enough to judge.
var FontChooserFloor = fyne.NewSize(720, 560) //nolint:gochecknoglobals // a size, not state

/*
fontFilter is the filter box, which is also where the keyboard lives.

widget.List spends the arrow keys itself, ignores Enter, and takes the focus
on every click, so a picker that wants "arrow to it and press Enter" has to
hold the keys somewhere the list cannot take them back from.
*/
type fontFilter struct {
	widget.Entry

	up, down, accept func()
}

func newFontFilter() *fontFilter {
	f := &fontFilter{}
	f.ExtendBaseWidget(f)
	return f
}

func (f *fontFilter) TypedKey(e *fyne.KeyEvent) {
	switch e.Name {
	case fyne.KeyUp:
		call(f.up)
	case fyne.KeyDown:
		call(f.down)
	case fyne.KeyReturn, fyne.KeyEnter:
		call(f.accept)
	default:
		f.Entry.TypedKey(e)
	}
}

func call(fn func()) {
	if fn != nil {
		fn()
	}
}

// indexOf is where a family sits in a list, or -1.
func indexOf(names []string, name string) int {
	for i, candidate := range names {
		if candidate == name {
			return i
		}
	}
	return -1
}

// selectFamily highlights a family if it is in the list, and highlights
// nothing when the filter has excluded it.
func selectFamily(list *widget.List, shown []string, name string) {
	for i, candidate := range shown {
		if candidate == name {
			list.Select(i)
			list.ScrollTo(i)
			return
		}
	}
	list.UnselectAll()
}

// The sample's lines. Prose for the shape of the letters, and the characters
// that tell families apart in a window full of ticket numbers and log lines.
const (
	samplePangram = "The quick brown fox jumps over the lazy dog."
	sampleShapes  = "0O1lI!|  AP-20106  #ff00ff  {}[]()<>"
)

/*
sampleBlock is the preview: a caption, the sample itself, and a line saying
what was drawn from the family and what was not.

The sample is built from canvas.Text objects carrying the family's file in
FontSource, which is Fyne's own "draw this string from this font" and the only
way to preview one that does not depend on a theme being consulted.

It was a container.ThemeOverride before, and it did not work. The theme was
right — CurrentForWidget on the painted objects resolved to the highlighted
family, under the real driver as well as the test one — but that is not what
the painter reads. Text is measured through a path that is handed no object at
all (canvas.Text.MinSize -> RenderedTextSize), so it cannot see a scope; the
face is cached against a scope id the text objects do not always carry; and
monospace text is resolved through a list of faces rather than the one the
theme named. Three ways to lose the family between deciding it and drawing it.
A resource on the object survives all of them.

Only the sample is in the family's own face. The caption and the note are the
program talking, and a block where every line changed face left no way to tell
which part was the sample.
*/
func sampleBlock(name string, mono bool, base fdtheme.Appearance) fyne.CanvasObject {
	family := fdtheme.PreviewFont(name)
	face := family.Face(fyne.TextStyle{})
	size := base.TextSize
	if size <= 0 {
		size = fdtheme.DefaultTextSize
	}

	lines := []fyne.CanvasObject{
		sampleLine(samplePangram, size, face),
		sampleLine(sampleShapes, size, face),
	}
	/*
		What this family can draw of its own, when it cannot draw the rest.

		A script font — and most of the families on a Linux machine are one,
		Noto ships one per writing system — carries punctuation and digits
		and not a single Latin letter. Its own digits in its own shapes are
		worth more than two lines of somebody else's glyphs.
	*/
	if own := family.Covered(sampleShapes); family.Missing(samplePangram) > 0 && own != "" {
		lines = append(lines, sampleLine(own, size, face))
	}

	return container.NewVBox(
		caption("Sample — "+name),
		container.NewVBox(lines...),
		caption(drawnIn(name, mono, base, family)),
	)
}

/*
sampleLine is one line of the sample, drawn from the family's own file.

A canvas.Text rather than a widget.Label: a Label takes its font from the
theme, and the theme is exactly what cannot be relied on to carry a family
this program has not adopted. A nil face is the theme's font, which is the
right answer for a family that cannot be read.
*/
func sampleLine(text string, size float32, face fyne.Resource) *canvas.Text {
	line := canvas.NewText(text, fynetheme.Color(fynetheme.ColorNameForeground))
	line.TextSize = size
	line.FontSource = face
	return line
}

/*
caption is a line of the program's own voice: dim, in the interface font, and
cut with an ellipsis rather than wrapped.

Never wrapped. A wrapping label's height depends on its width, and these sit
in a Border's bottom, which takes its height from a measurement made before
the preview theme widened the glyphs — so the second line fell outside the
reserved height and was clipped mid-word.
*/
func caption(text string) fyne.CanvasObject {
	label := widget.NewLabel(text)
	label.Importance = widget.LowImportance
	label.Wrapping = fyne.TextWrapOff
	label.Truncation = fyne.TextTruncateEllipsis
	return label
}

/*
drawnIn says what the sample is really made of.

"Is this a different font?" is a question the eye often cannot answer:
families differ by a hair at a sentence's size, and a proportional variant of
a monospace family reads like an ordinary sans. Worse, most of the families
on a Linux machine cannot draw a Latin sentence at all — Noto ships a font per
writing system — and what appears is another font's glyphs standing in for the
missing ones, identical for every such family and indistinguishable from a
preview that is not working.

Naming the file was not enough on its own: the theme happily reports a family
that is drawing none of what is on screen. So this counts what the family
cannot draw and says so.
*/
func drawnIn(name string, mono bool, base fdtheme.Appearance, family *fdtheme.Font) string {
	style := fyne.TextStyle{Monospace: mono}
	face := base.PreviewTheme(name, mono).Font(style)
	fallback := "the built-in face"
	if face != nil {
		fallback = face.Name()
	}
	if family == nil {
		return "no readable face — drawn in " + fallback
	}

	missing := family.Missing(samplePangram)
	proportional := ""
	if mono && !family.IsMonospace() {
		// The thing worth knowing when choosing a monospace face, and the
		// one a sentence of prose will not reveal.
		proportional = " — not a monospace family; columns will not line up"
	}
	switch {
	case missing == 0:
		return "drawn in " + face.Name() + proportional
	case missing >= len([]rune(samplePangram)):
		return face.Name() + " has no Latin letters — the sentence above is another font's"
	default:
		return face.Name() + " is missing " + strconv.Itoa(missing) +
			" of those characters — they are another font's"
	}
}
