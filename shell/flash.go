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
	// enough to clear the status bar, and flashSideGap the margin it keeps
	// from the window's edges in a window narrower than FlashWidth.
	flashBottom  float32 = 56
	flashSideGap float32 = 40
	// FlashHoldGood and FlashHoldWarn are how long a banner stays before it
	// fades. A warning gets longer because it usually names a condition to
	// act on. A failure never fades.
	FlashHoldGood = 6 * time.Second
	FlashHoldWarn = 12 * time.Second
	// FlashFade is the fade itself: long enough to read as intentional, short
	// enough that the banner is not sitting there half-gone.
	FlashFade = 700 * time.Millisecond
	// flashRadius and flashStroke are the banner's own box, drawn now that it
	// is not a popup with a box of its own.
	flashRadius float32 = 4
	flashStroke float32 = 1
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

	// Two rectangles, not one. The tint is translucent so text stays readable
	// over it in every scheme, which was fine while a banner was a popup and
	// the popup drew an opaque background underneath. Drawn in a layer of the
	// content there is nothing underneath but the section, so the banner needs
	// to bring its own: without it the log behind a banner reads straight
	// through the message.
	tint := s.flashTint(st)
	back := canvas.NewRectangle(s.flashBacking())
	back.CornerRadius = flashRadius
	back.StrokeColor = s.flashEdge()
	back.StrokeWidth = flashStroke
	bg := canvas.NewRectangle(tint)
	bg.CornerRadius = flashRadius

	label := widget.NewLabel(text)
	label.Wrapping = fyne.TextWrapWord

	// Dismissable, because a banner that only leaves on a timer leaves either
	// too early to read or too late to be rid of. This is also the only way to
	// clear a failure, which does not go on its own.
	dismiss := widget.NewButtonWithIcon("", fynetheme.CancelIcon(), func() { s.clearFlash(seq) })
	dismiss.Importance = widget.LowImportance

	banner := container.NewStack(back, bg, container.NewPadded(
		container.NewBorder(nil, nil, widgets.Marker(st), dismiss, label)))
	s.flashes.Objects = []fyne.CanvasObject{banner}
	s.flashes.Refresh()

	hold, fades := flashHold(st)
	if !fades || !s.OnScreen() {
		// Nothing to fade with no window: the timer would come back seconds
		// later to animate a rectangle nobody is drawing, on a goroutine the
		// test that made it has long since finished with.
		return
	}

	backing, edge := s.flashBacking(), s.flashEdge()
	go func() {
		time.Sleep(hold)
		fyne.Do(func() {
			if s.flashSeq != seq {
				return // a newer banner owns the slot
			}
			// One animation driving every layer. Fading the tint alone would
			// leave the banner's own opaque backing sitting there, which is
			// the whole banner minus its colour.
			fade := fyne.NewAnimation(FlashFade, func(done float32) {
				left := 1 - done
				bg.FillColor = faded(tint, left)
				back.FillColor = faded(backing, left)
				back.StrokeColor = faded(edge, left)
				canvas.Refresh(back)
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
}

/*
flashLayout floats the banner over the bottom of the content, centred.

A layout rather than a popup, and that is the whole point: a popup is an
overlay, and Fyne routes every pointer event to the top overlay, so a banner
drawn as one took every click in the window for the six to twelve seconds it was
up -- and the first click dismissed it rather than doing what it was aimed at
(quirk 26). The comment above Flash has always said the section behind a banner
stays usable. This is what makes that true.

Drawn in a layer of the window's content instead, the banner's own dismiss
button still takes its clicks, because it is walked last and is tappable, and
everything else falls through to the section underneath, because nothing else in
the banner is.
*/
type flashLayout struct{}

// MinSize is nothing: the layer is a float over the content and must not make
// the window any bigger.
func (flashLayout) MinSize([]fyne.CanvasObject) fyne.Size { return fyne.Size{} }

/*
Layout centres one banner near the bottom, narrower than the window.

Measured twice on purpose: a wrapping label does not know its own height until
it has a width, so the width is chosen, the banner is resized to it, and only
then does MinSize report the height the wrapping produced.
*/
func (flashLayout) Layout(objs []fyne.CanvasObject, size fyne.Size) {
	width := min(FlashWidth, size.Width-flashSideGap)
	if width <= 0 {
		return
	}
	for _, o := range objs {
		o.Resize(fyne.NewSize(width, o.MinSize().Height))
		height := o.MinSize().Height
		o.Resize(fyne.NewSize(width, height))
		o.Move(fyne.NewPos((size.Width-width)/2, size.Height-height-flashBottom))
	}
}

/*
flashBacking is what the banner is drawn on: the window's own background,
opaque, so the section behind it does not read through the message.

flashEdge is a hairline around it, so a banner over content of a similar colour
still reads as a separate thing rather than as text that has appeared in the
middle of the section.
*/
func (s *Shell) flashBacking() color.NRGBA {
	if th, ok := s.App.Settings().Theme().(fdtheme.Theme); ok {
		return opaque(th.Palette().WindowBG)
	}
	return opaque(fynetheme.Color(fynetheme.ColorNameOverlayBackground))
}

func (s *Shell) flashEdge() color.NRGBA {
	if th, ok := s.App.Settings().Theme().(fdtheme.Theme); ok {
		return opaque(th.Palette().Separator)
	}
	return opaque(fynetheme.Color(fynetheme.ColorNameSeparator))
}

// opaque is c at full alpha, because a backing that is not opaque is not a
// backing.
func opaque(c color.Color) color.NRGBA {
	r, g, b, _ := c.RGBA()
	return color.NRGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: 0xff}
}

// faded is c at a fraction of its own alpha, for the animation that takes every
// layer of the banner down together.
func faded(c color.NRGBA, left float32) color.NRGBA {
	c.A = uint8(float32(c.A) * left)
	return c
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
