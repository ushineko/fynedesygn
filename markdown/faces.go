package markdown

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

/*
drawable is a RichText whose segments ask only for faces the theme has.

**Fyne dereferences a missing font rather than falling back.** Bold around a
code span -- an emphasised identifier, in ordinary prose -- parses to a
segment that is bold *and* monospace, and a theme without that face returns
nothing:

	Fyne error: font for style {Bold:true, Monospace:true} not defined in theme
	painter.loadMeasureFont({0x0, 0x0})
	panic: invalid memory address or nil pointer dereference

fynedesygn's own theme has the face and renders it. Fyne's *test* theme does
not, so the crash lands in headless tests -- which is what this library sells,
and how every consumer tests their own sections. A document with an emphasised
identifier takes down a suite with a stack trace pointing into the painter and
nothing naming the document.

So each segment's style is looked up before the painter sees it, and where the
theme has nothing, bold is dropped and monospace kept: monospace says the run
*is* code, which is why the author reached for backticks, and emphasis that
cannot be drawn is the smaller loss.

Conditional on the theme rather than always, because a theme that has the face
should draw it. See docs/fyne-quirks.md.
*/
func drawable(rt *widget.RichText) *widget.RichText {
	for i, seg := range rt.Segments {
		text, ok := seg.(*widget.TextSegment)
		if !ok || hasFace(text.Style.TextStyle) {
			continue
		}

		/*
			Bold first, because it is the one that can be spared. If the
			theme has nothing for what is left either, it is left alone: a
			style nothing can draw is Fyne's to answer for, and guessing
			further would be this package inventing a fallback nobody asked
			for.
		*/
		plainer := text.Style.TextStyle
		plainer.Bold = false
		if !hasFace(plainer) {
			continue
		}
		text.Style.TextStyle = plainer
		rt.Segments[i] = text
	}
	return rt
}

// hasFace reports whether the running theme can draw a style. No app, no
// theme, no opinion: the caller is left with what it had.
func hasFace(style fyne.TextStyle) bool {
	app := fyne.CurrentApp()
	if app == nil {
		return true
	}
	return app.Settings().Theme().Font(style) != nil
}
