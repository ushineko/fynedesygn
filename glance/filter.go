package glance

import (
	"regexp"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
)

// DefaultFilterKey starts a filter, as it does in btop.
const DefaultFilterKey = 'f'

/*
Filter is one line in a drawer that narrows a list from the keyboard, in the
manner of btop: its key starts it, what is typed narrows the list as it is
typed, Return keeps the filter and Escape clears it (spec 068).

**The line is always there**, so starting or clearing a filter moves no row.
It shows the key as a hint while there is no filter, the text and a cursor
while one is typed, and the filter in force afterwards.

**The text is a regular expression, matched without regard to case.** Text
that is not one, as most are for a moment while being typed (an open
bracket), is matched as it is written and drawn in the warning colour, so a
half-typed expression never empties the list.

The filter does not hold the list. The program asks Matches for each name and
redraws when OnChanged says the filter is different.

It is not an entry widget, and cannot be: the panel's catcher takes every
click, so an entry could not be clicked into, and a focused widget would take
the canvas's keys for itself. Drawer.SetFilter points the window's keys here
instead.
*/
type Filter struct {
	themed

	// Key starts the filter. DefaultFilterKey unless changed.
	Key rune

	// OnChanged is called, on the UI thread, when what the filter matches has
	// changed.
	OnChanged func()

	text    string
	editing bool
	expr    *regexp.Regexp

	// done is called when Return or Escape ends the filter. The drawer gives
	// the keyboard back there.
	done func()

	line *canvas.Text
	box  *fyne.Container
}

// NewFilter builds an empty filter.
func NewFilter() *Filter {
	f := &Filter{Key: DefaultFilterKey, line: canvas.NewText("", nil)}
	f.line.TextStyle = fyne.TextStyle{Monospace: true}
	f.box = container.NewStack(f.line)
	f.Restyle()
	return f
}

// Object is the filter's line, for the drawer's content.
func (f *Filter) Object() fyne.CanvasObject { return f.box }

// SetTheme gives the filter a face of its own and repaints in it.
func (f *Filter) SetTheme(th fyne.Theme) {
	f.themed.SetTheme(th)
	f.Restyle()
}

// Text is the filter as typed, empty when there is none.
func (f *Filter) Text() string { return f.text }

// Editing reports whether the filter is being typed now.
func (f *Filter) Editing() bool { return f.editing }

// Valid reports whether the text is a regular expression. An empty filter is
// valid; one that is not is matched as plain text.
func (f *Filter) Valid() bool { return f.text == "" || f.expr != nil }

// Matches reports whether a name passes the filter. Everything passes an
// empty one.
func (f *Filter) Matches(name string) bool {
	switch {
	case f.text == "":
		return true
	case f.expr != nil:
		return f.expr.MatchString(name)
	default:
		return strings.Contains(strings.ToLower(name), strings.ToLower(f.text))
	}
}

// SetText replaces the filter's text, as if it had been typed and kept.
func (f *Filter) SetText(text string) {
	if f.text == text {
		return
	}
	f.set(text)
	f.Restyle()
}

// Clear takes the filter away and ends its typing. It does not give the
// keyboard back: that is Escape's.
func (f *Filter) Clear() {
	f.editing = false
	f.SetText("")
	f.Restyle()
}

func (f *Filter) set(text string) {
	f.text = text
	f.expr = nil
	if text != "" {
		// Compile cannot be given a pattern that does harm: Go's expressions
		// run in time linear in the input, whatever is typed.
		if re, err := regexp.Compile("(?i)" + text); err == nil {
			f.expr = re
		}
	}
	if f.OnChanged != nil {
		f.OnChanged()
	}
}

// stop ends the typing and keeps the text, for a drawer that closes.
func (f *Filter) stop() {
	if f.editing {
		f.editing = false
		f.Restyle()
	}
}

// TypedRune takes a character: the filter's key starts it, and while it is
// being typed every character is added to it. It reports whether the
// character was the filter's.
func (f *Filter) TypedRune(r rune) bool {
	switch {
	case f.editing:
		f.set(f.text + string(r))
	case r == f.Key:
		f.editing = true
	default:
		return false
	}
	f.Restyle()
	return true
}

// TypedKey takes a key: Backspace removes a character, Return keeps the
// filter, Escape clears it. It reports whether the key was the filter's.
func (f *Filter) TypedKey(e *fyne.KeyEvent) bool {
	if e == nil || (!f.editing && f.text == "") {
		return false
	}
	ended := false
	switch e.Name {
	case fyne.KeyBackspace:
		if !f.editing {
			return false
		}
		if runes := []rune(f.text); len(runes) > 0 {
			f.set(string(runes[:len(runes)-1]))
		}
	case fyne.KeyReturn, fyne.KeyEnter:
		if !f.editing {
			return false
		}
		f.editing = false
		ended = true
	case fyne.KeyEscape:
		f.editing = false
		if f.text != "" {
			f.set("")
		}
		ended = true
	default:
		return false
	}
	f.Restyle()
	if ended && f.done != nil {
		f.done()
	}
	return true
}

// filterRunes is how much of a long filter is drawn: its end, which is where
// the typing is. A line that grew with its text would run out of the drawer,
// and on the left that is over the cards.
const filterRunes = 28

// shown is the text as drawn: the whole of it, or its end after an ellipsis.
func (f *Filter) shown() string {
	runes := []rune(f.text)
	if len(runes) <= filterRunes {
		return f.text
	}
	return Ellipsis + string(runes[len(runes)-filterRunes+1:])
}

// Restyle repaints the line in the current theme and state.
func (f *Filter) Restyle() {
	text, colour := string(f.Key)+"  filter", theme.ColorNameDisabled
	switch {
	case f.editing:
		// An underscore and not a block: every mono family has one.
		text, colour = "/"+f.shown()+"_", theme.ColorNameForeground
	case f.text != "":
		text, colour = "/"+f.shown(), theme.ColorNamePrimary
	}
	if !f.Valid() {
		colour = theme.ColorNameWarning
	}
	f.line.Text = text
	f.line.TextSize = f.textSize()
	f.line.Color = f.colour(colour)
	f.line.Refresh()
	f.refit(f.line)
}
