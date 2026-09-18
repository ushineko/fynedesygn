package main

import (
	"testing"

	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/require"

	"github.com/ushineko/fynedesygn/fynetest"
	fdtheme "github.com/ushineko/fynedesygn/theme"
)

// testGallery is a gallery with no window: content is left nil so refresh is
// a no-op and every builder runs inline.
func testGallery(t *testing.T) *gallery {
	t.Helper()
	fdtheme.RescanFonts(t.TempDir())
	t.Cleanup(func() { fdtheme.RescanFonts() })
	app := fynetest.App(t)
	return &gallery{app: app, appearance: fdtheme.LoadAppearance(app.Preferences())}
}

func TestSectionNamesNeedNoApp(t *testing.T) {
	// Canary #8 (docs/fyne-quirks.md): the names are read before any Fyne app
	// exists, as --help and --version do. A theme icon constructed here would
	// log "Attempt to access current Fyne app when none is started".
	require.Equal(t, []string{"Appearance", "Widgets", "Table", "Fonts"}, sectionNames())
	require.Equal(t, 2, sectionIndex("table"))
	require.Equal(t, -1, sectionIndex("nope"))
}

func TestEverySectionRendersHeadlesslyInEveryScheme(t *testing.T) {
	g := testGallery(t)
	for _, scheme := range fdtheme.SchemeNames() {
		g.appearance.Scheme = scheme
		g.app.Settings().SetTheme(g.appearance.Theme())
		for _, s := range sections() {
			var o = s.build(g)
			require.NotPanics(t, func() {
				w := test.NewWindow(o)
				defer w.Close()
				w.Resize(o.MinSize())
			}, "%s in %s", s.title, scheme)
		}
	}
}

func TestAppearanceSectionShowsTheChosenSchemeAndItsRoles(t *testing.T) {
	g := testGallery(t)
	g.appearance.Scheme = "Windows Light"
	o := buildAppearance(g)
	text := fynetest.Text(o)
	require.Contains(t, text, "Windows Light: dark=false, corner radius 4, padding 4.")
	sel := fynetest.FindSelect(o)
	require.NotNil(t, sel)
	require.Equal(t, "Windows Light", sel.Selected)
}

func TestChangingTheSchemeInTheSectionSavesItUnlessItIsAOneRunOverride(t *testing.T) {
	g := testGallery(t)
	o := buildAppearance(g)
	fynetest.FindSelect(o).SetSelected("macOS Dark")
	require.Equal(t, "macOS Dark", g.app.Preferences().String(fdtheme.PrefScheme))

	g.oneRunScheme = true
	o = buildAppearance(g)
	fynetest.FindSelect(o).SetSelected("Oxygen Dark")
	require.Equal(t, "macOS Dark", g.app.Preferences().String(fdtheme.PrefScheme), "the override is applied, never saved")
	th, ok := g.app.Settings().Theme().(fdtheme.Theme)
	require.True(t, ok)
	require.Equal(t, "Oxygen Dark", th.Palette().Name)
}

func TestTheWidgetsSectionNamesEveryPrimitive(t *testing.T) {
	g := testGallery(t)
	text := fynetest.Text(buildWidgets(g))
	for _, name := range []string{"widgets.Heading", "widgets.Card", "widgets.Action", "widgets.AboutNote", "widgets.FixedHeight"} {
		require.Contains(t, text, name)
	}
}

func TestShowSwapsTheContentPane(t *testing.T) {
	g := testGallery(t)
	g.content = container.NewScroll(widget.NewLabel(""))
	g.show(2)
	require.Equal(t, 2, g.current)
	require.Contains(t, fynetest.Text(g.content.Content), "table.Detail")
}
