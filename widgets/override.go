package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

/*
ThemeOverride draws its content in a theme of its own, like Fyne's
container.ThemeOverride, and applies that theme only when it is built and
when it is told to: by SetContent, SetTheme or Reapply. A Refresh refreshes
the content and nothing else.

Fyne's override applies its theme again on every Refresh, and each application
makes a new theme scope (quirk 51 in docs/fyne-quirks.md). Fyne keeps the fonts
it builds for a scope for as long as the program runs, so an override that is
refreshed leaks memory at each refresh. A refresh also reaches the override
whenever any container around it is refreshed, because fyne.Container.Refresh
refreshes every child. A program that refreshes its table every few seconds
grows without bound.

**Reapply after adding to the content.** An override covers the objects that
existed when its theme was last applied (quirk 37). Fyne's own override made
up for that by applying the theme at every refresh. This one leaves it to the
caller, who knows when the tree has changed. Each Reapply makes one scope, so
call it after content is added, not on a timer. Lists, trees and text grids
give the rows they create their parent's scope themselves and need no
Reapply.
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
// it was last applied.
func (o *ThemeOverride) Reapply() {
	o.inner.Refresh()
}

// Refresh redraws the content without applying the theme again.
func (o *ThemeOverride) Refresh() {
	o.inner.Content.Refresh()
	o.BaseWidget.Refresh()
}

// CreateRenderer implements fyne.Widget.
func (o *ThemeOverride) CreateRenderer() fyne.WidgetRenderer {
	return &themeOverrideRenderer{o: o, objects: []fyne.CanvasObject{o.inner}}
}

// themeOverrideRenderer lays out the Fyne override it holds and never
// refreshes it, which is the point of the type.
type themeOverrideRenderer struct {
	o       *ThemeOverride
	objects []fyne.CanvasObject
}

func (r *themeOverrideRenderer) Destroy() {}

func (r *themeOverrideRenderer) Layout(size fyne.Size) { r.o.inner.Resize(size) }

func (r *themeOverrideRenderer) MinSize() fyne.Size { return r.o.inner.MinSize() }

func (r *themeOverrideRenderer) Objects() []fyne.CanvasObject { return r.objects }

func (r *themeOverrideRenderer) Refresh() {}
