package shell

import (
	"image/color"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	fynetheme "fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	fd "github.com/ushineko/fynedesygn"
	fdtheme "github.com/ushineko/fynedesygn/theme"
	"github.com/ushineko/fynedesygn/widgets"
)

// The banner's geometry and timings.
const (
	// FlashWidth is how wide a banner is drawn, so a long message wraps
	// rather than spanning the window.
	FlashWidth float32 = 720
	// flashBottom is the gap between the banner and the bottom of the canvas,
	// enough to clear the status bar.
	flashBottom float32 = 56
	// FlashHoldGood and FlashHoldWarn are how long a banner stays before it
	// fades. A warning gets longer because it usually names a condition to
	// act on. A failure never fades.
	FlashHoldGood = 6 * time.Second
	FlashHoldWarn = 12 * time.Second
	// FlashFade is the fade itself: long enough to read as intentional, short
	// enough that the banner is not sitting there half-gone.
	FlashFade = 700 * time.Millisecond
)

// flashHold says how long a banner of this status stays up, and whether it
// goes on its own at all. A failure does not: it waits to be dismissed, or
// until another operation replaces it. An error that removes itself on a
// timer is an error nobody read.
func flashHold(st fd.Status) (time.Duration, bool) {
	switch st {
	case fd.StatusBad:
		return 0, false
	case fd.StatusWarn:
		return FlashHoldWarn, true
	case fd.StatusGood, fd.StatusInfo:
		return FlashHoldGood, true
	}
	return FlashHoldGood, true
}

/*
Flash reports the result of an operation as a banner floated over the bottom
of the content, centred. One banner shows at a time: a newer result replaces
an older one rather than stacking, so the most recent thing that happened is
always the thing on screen, and nothing in the section behind it moves.

Fyne animates properties, not opacity: a widget has no alpha to fade. So the
fade is on the banner's own background rectangle, whose colour animates from
the status tint to fully transparent. The text is left at full strength for
the whole life of the banner, which is the accessible choice anyway.

Streams do not come through here. A job emits hundreds of lines and one banner
per line would be a slot flickering for minutes; the log pane is where those
go, and one banner summarises the result. Call on the UI thread.
*/
func (s *Shell) Flash(text string, st fd.Status) {
	s.flashSeq++
	seq := s.flashSeq

	tint := s.flashTint(st)
	bg := canvas.NewRectangle(tint)
	bg.CornerRadius = 2

	label := widget.NewLabel(text)
	label.Wrapping = fyne.TextWrapWord

	// Dismissable, because a banner that only leaves on a timer leaves either
	// too early to read or too late to be rid of. This is also the only way to
	// clear a failure, which does not go on its own.
	dismiss := widget.NewButtonWithIcon("", fynetheme.CancelIcon(), func() { s.clearFlash(seq) })
	dismiss.Importance = widget.LowImportance

	banner := container.NewStack(bg, container.NewPadded(
		container.NewBorder(nil, nil, widgets.Marker(st), dismiss, label)))
	s.flashes.Objects = []fyne.CanvasObject{banner}
	s.flashes.Refresh()
	s.showFlashPop()

	hold, fades := flashHold(st)
	if !fades || !s.OnScreen() {
		// Nothing to fade with no window: the timer would come back seconds
		// later to animate a rectangle nobody is drawing, on a goroutine the
		// test that made it has long since finished with.
		return
	}

	transparent := color.NRGBA{R: tint.R, G: tint.G, B: tint.B, A: 0}
	go func() {
		time.Sleep(hold)
		fyne.Do(func() {
			if s.flashSeq != seq {
				return // a newer banner owns the slot
			}
			fade := canvas.NewColorRGBAAnimation(tint, transparent, FlashFade, func(c color.Color) {
				bg.FillColor = c
				canvas.Refresh(bg)
			})
			fade.Curve = fyne.AnimationEaseIn
			fade.Start()
		})
		time.Sleep(FlashFade)
		fyne.Do(func() { s.clearFlash(seq) })
	}()
}

// ClearFlash dismisses the banner on screen, if any.
func (s *Shell) ClearFlash() { s.clearFlash(s.flashSeq) }

// FlashText is the text of the banner on screen, or "" when there is none.
// Tests read it; programs have no reason to.
func (s *Shell) FlashText() string {
	if len(s.flashes.Objects) == 0 {
		return ""
	}
	return widgetTextIn(s.flashes.Objects[0])
}

// widgetTextIn finds the first label's text in a small tree.
func widgetTextIn(o fyne.CanvasObject) string {
	switch v := o.(type) {
	case *widget.Label:
		return v.Text
	case *fyne.Container:
		for _, c := range v.Objects {
			if t := widgetTextIn(c); t != "" {
				return t
			}
		}
	}
	return ""
}

// clearFlash empties the slot, unless a newer banner has taken it. Called
// from the dismiss button and from the fade's own timer, which may arrive
// after the banner it belongs to has already been replaced.
func (s *Shell) clearFlash(seq int) {
	if s.flashSeq != seq {
		return
	}
	s.flashes.Objects = nil
	s.flashes.Refresh()
	if s.flashPop != nil {
		s.flashPop.Hide()
	}
}

// showFlashPop floats the banner over the content, centred, a little above
// the status bar. Not modal: a result is something to read, not something to
// answer, and the section behind it stays usable.
func (s *Shell) showFlashPop() {
	if !s.OnScreen() {
		return
	}
	c := s.Window.Canvas()
	if s.flashPop == nil {
		s.flashPop = widget.NewPopUp(widgets.FixedWidth(s.flashes, FlashWidth), c)
	}
	cs := c.Size()
	width := min(FlashWidth, cs.Width-40)
	s.flashPop.Content = widgets.FixedWidth(s.flashes, width)
	size := s.flashPop.Content.MinSize()
	pos := fyne.NewPos((cs.Width-size.Width)/2, cs.Height-size.Height-flashBottom)
	s.flashPop.ShowAtPosition(pos)
}

// flashTint is the banner's starting colour: the status role from the active
// palette at low alpha, so text stays readable over it in every scheme. A
// theme that is not this module's gets Fyne's own roles.
func (s *Shell) flashTint(st fd.Status) color.NRGBA {
	var c color.Color
	if th, ok := s.App.Settings().Theme().(fdtheme.Theme); ok {
		p := th.Palette()
		switch st {
		case fd.StatusGood:
			c = p.Positive
		case fd.StatusWarn:
			c = p.Neutral
		case fd.StatusBad:
			c = p.Negative
		case fd.StatusInfo:
			c = p.SelectionBG
		}
	} else {
		c = widgets.StatusColor(st)
	}
	tint, _ := fdtheme.Alpha(c, 0x4d).(color.NRGBA)
	return tint
}
