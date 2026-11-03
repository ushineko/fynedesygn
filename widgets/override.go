package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

/*
ThemeOverride draws its content in a theme of its own. It is Fyne's
container.ThemeOverride, with the theme and the content settable and the
theme readable (OwnTheme).

**It applies its theme again at every Refresh**, as Fyne's does, and each
application is a new theme scope whose fonts Fyne keeps for the life of the
program (quirk 51). 0.1.111 applied it only when told to, which stopped that
leak and lost the theme instead: Fyne's cache destroys the renderers of
widgets that have not been drawn for a minute and takes them out of their
override as it does, and a widget that came back drew in the application's
theme. So an override belongs where it is refreshed rarely — a window, a
section, a dialog — never around a table, a list or anything a timer redraws.
Widgets that draw text in a face of their own (canvas.Text.FontSource) need no
override at all.
*/
type ThemeOverride struct {
	widget.BaseWidget

	inner *container.ThemeOverride
}

// NewThemeOverride draws content in th.
func NewThemeOverride(content fyne.CanvasObject, th fyne.Theme) *ThemeOverride {
	o := &ThemeOverride{inner: container.NewThemeOverride(content, th)}
	o.ExtendBaseWidget(o)
	return o
}

// OwnTheme is the theme the content is drawn in. Theme, from
// widget.BaseWidget, is the one the override itself sits in.
func (o *ThemeOverride) OwnTheme() fyne.Theme { return o.inner.Theme }

// Content is what the override draws.
func (o *ThemeOverride) Content() fyne.CanvasObject { return o.inner.Content }

// SetContent replaces what the override draws, and applies the theme to it.
func (o *ThemeOverride) SetContent(content fyne.CanvasObject) {
	o.inner.Content = content
	o.inner.Refresh()
	o.BaseWidget.Refresh()
}

// SetTheme replaces the theme and applies it to the content.
func (o *ThemeOverride) SetTheme(th fyne.Theme) {
	o.inner.Theme = th
	o.inner.Refresh()
}

// Reapply applies the theme again, to cover objects added to the content since
// it was last applied. Refresh does the same.
func (o *ThemeOverride) Reapply() {
	o.inner.Refresh()
}

// Refresh applies the theme again and redraws the content.
func (o *ThemeOverride) Refresh() {
	o.inner.Refresh()
	o.BaseWidget.Refresh()
}

// CreateRenderer implements fyne.Widget.
func (o *ThemeOverride) CreateRenderer() fyne.WidgetRenderer {
	return &themeOverrideRenderer{o: o, objects: []fyne.CanvasObject{o.inner}}
}

// themeOverrideRenderer lays out the Fyne override it holds. Refresh reaches
// that override through ThemeOverride.Refresh.
type themeOverrideRenderer struct {
	o       *ThemeOverride
	objects []fyne.CanvasObject
}

func (r *themeOverrideRenderer) Destroy() {}

func (r *themeOverrideRenderer) Layout(size fyne.Size) { r.o.inner.Resize(size) }

func (r *themeOverrideRenderer) MinSize() fyne.Size { return r.o.inner.MinSize() }

func (r *themeOverrideRenderer) Objects() []fyne.CanvasObject { return r.objects }

func (r *themeOverrideRenderer) Refresh() {}
