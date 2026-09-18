package theme

import (
	"image/color"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	fynetheme "fyne.io/fyne/v2/theme"
	"github.com/stretchr/testify/require"
)

func TestSchemeNamesAreOfferedInTheDocumentedOrder(t *testing.T) {
	require.Equal(t, []string{
		"Breeze Dark", "Breeze Light", "Oxygen Dark",
		"Adwaita Dark", "Adwaita Light",
		"Windows Dark", "Windows Light",
		"macOS Dark", "macOS Light",
	}, SchemeNames())
}

func TestAnUnknownSchemeNameFallsBackToTheDefaultRatherThanFailing(t *testing.T) {
	require.Equal(t, DefaultScheme().Name, SchemeByName("nonsense").Name)
	require.Equal(t, "Oxygen Dark", SchemeByName("Oxygen Dark").Name)
}

func TestEveryPaletteAnswersEveryMappedColourRole(t *testing.T) {
	for _, p := range Schemes() {
		th := New(p, Options{})
		for _, n := range MappedColors() {
			c := th.Color(n, fynetheme.VariantLight)
			require.NotNil(t, c, "%s: %s", p.Name, n)
		}
		// Every colour field of the palette itself is set; a nil one would
		// only show up as a panic in the painter.
		for name, c := range map[string]any{
			"WindowBG": p.WindowBG, "WindowFG": p.WindowFG, "ViewBG": p.ViewBG,
			"ViewAltBG": p.ViewAltBG, "ViewFG": p.ViewFG, "ButtonBG": p.ButtonBG,
			"ButtonFG": p.ButtonFG, "SelectionBG": p.SelectionBG, "SelectionFG": p.SelectionFG,
			"TooltipBG": p.TooltipBG, "InactiveFG": p.InactiveFG, "Negative": p.Negative,
			"Positive": p.Positive, "Neutral": p.Neutral, "Link": p.Link, "Focus": p.Focus,
			"Hover": p.Hover, "Separator": p.Separator,
		} {
			require.NotNil(t, c, "%s: %s", p.Name, name)
		}
		require.Positive(t, p.Radius, p.Name)
		require.Positive(t, p.Padding, p.Name)
	}
}

func TestThePaletteDecidesDarknessNotTheVariantFyneIsSetTo(t *testing.T) {
	th := New(BreezeDark, Options{})
	light := th.Color(fynetheme.ColorNameBackground, fynetheme.VariantLight)
	dark := th.Color(fynetheme.ColorNameBackground, fynetheme.VariantDark)
	require.Equal(t, light, dark)
	require.Equal(t, fynetheme.VariantDark, th.Variant())
	require.Equal(t, fynetheme.VariantLight, New(BreezeLight, Options{}).Variant())
}

func TestMonospaceIsNeverTakenFromTheInterfaceFamily(t *testing.T) {
	iface := &Font{regular: fyne.NewStaticResource("iface.ttf", []byte("x"))}
	th := New(BreezeDark, Options{Font: iface})
	require.Equal(t, iface.regular, th.Font(fyne.TextStyle{}))
	require.Equal(t, fynetheme.DefaultTheme().Font(fyne.TextStyle{Monospace: true}),
		th.Font(fyne.TextStyle{Monospace: true}))

	mono := &Font{regular: fyne.NewStaticResource("mono.ttf", []byte("y"))}
	th = New(BreezeDark, Options{Font: iface, Mono: mono})
	require.Equal(t, mono.regular, th.Font(fyne.TextStyle{Monospace: true}))
}

func TestSizesScaleFromTheChosenBaseAndThePaletteSetsTheCorners(t *testing.T) {
	th := New(BreezeDark, Options{})
	require.InDelta(t, DefaultTextSize, th.Size(fynetheme.SizeNameText), 0.001)
	require.InDelta(t, DefaultTextSize*1.3, th.Size(fynetheme.SizeNameHeadingText), 0.001)
	require.InDelta(t, 2, th.Size(fynetheme.SizeNameInputRadius), 0.001)
	require.InDelta(t, 3, th.Size(fynetheme.SizeNamePadding), 0.001)

	th = New(MacOSDark, Options{TextSize: 16})
	require.InDelta(t, 16, th.Size(fynetheme.SizeNameText), 0.001)
	require.InDelta(t, 16*0.85, th.Size(fynetheme.SizeNameCaptionText), 0.001)
	require.InDelta(t, 8, th.Size(fynetheme.SizeNameSelectionRadius), 0.001)

	// A palette without tokens gets Fyne's own.
	bare := New(Palette{}, Options{})
	require.Equal(t, fynetheme.DefaultTheme().Size(fynetheme.SizeNamePadding), bare.Size(fynetheme.SizeNamePadding))
}

func TestTheThemeCanBeSetOnAFyneApp(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()
	for _, p := range Schemes() {
		app.Settings().SetTheme(New(p, Options{}))
		require.Equal(t, p.WindowBG, app.Settings().Theme().Color(fynetheme.ColorNameBackground, app.Settings().ThemeVariant()))
	}
}

func TestAlphaKeepsTheHueAndSetsOnlyTheOpacity(t *testing.T) {
	c, ok := Alpha(RGB(10, 20, 30), 0x40).(color.NRGBA)
	require.True(t, ok)
	require.Equal(t, color.NRGBA{R: 10, G: 20, B: 30, A: 0x40}, c)
}
