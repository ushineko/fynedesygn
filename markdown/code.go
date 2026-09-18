package markdown

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	fynetheme "fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

/*
CodePanel is monospace text on the same panel Fyne draws a code block on,
without the horizontal scroll that takes the wheel from the section.

Long lines wrap rather than run off the side. Fyne's own block scrolls them
sideways, which reads better but puts the rest of the line behind a gesture
that is exactly the one the page needs; here the whole command is on screen
and the wheel always belongs to the document.
*/
type CodePanel struct {
	widget.BaseWidget
	code string
	bg   *canvas.Rectangle
}

// NewCodePanel wraps code, which is shown as given.
func NewCodePanel(code string) *CodePanel {
	c := &CodePanel{code: code}
	c.ExtendBaseWidget(c)
	return c
}

// Code is the text the panel shows.
func (c *CodePanel) Code() string { return c.code }

// CreateRenderer implements fyne.Widget.
func (c *CodePanel) CreateRenderer() fyne.WidgetRenderer {
	c.bg = canvas.NewRectangle(fynetheme.Color(fynetheme.ColorNameInputBackground))
	c.bg.StrokeColor = fynetheme.Color(fynetheme.ColorNameInputBorder)
	c.bg.StrokeWidth = 1
	c.bg.CornerRadius = fynetheme.Size(fynetheme.SizeNameInputRadius)

	text := widget.NewLabelWithStyle(c.code, fyne.TextAlignLeading, fyne.TextStyle{Monospace: true})
	text.Wrapping = fyne.TextWrapWord
	return widget.NewSimpleRenderer(container.NewStack(c.bg, container.NewPadded(text)))
}

// Refresh repaints the panel from the active scheme, so that changing the
// colours in Appearance does not leave a block drawn in the old ones.
func (c *CodePanel) Refresh() {
	if c.bg != nil {
		c.bg.FillColor = fynetheme.Color(fynetheme.ColorNameInputBackground)
		c.bg.StrokeColor = fynetheme.Color(fynetheme.ColorNameInputBorder)
	}
	c.BaseWidget.Refresh()
}
