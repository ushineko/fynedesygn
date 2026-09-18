package theme

import (
	"testing"

	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/require"
)

func TestAppearanceOnEmptyPreferencesIsTheDefault(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()
	require.Equal(t, DefaultAppearance(), LoadAppearance(app.Preferences()))
	require.Equal(t, DefaultScheme().Name, DefaultAppearance().Scheme)
}

func TestAppearanceRoundTripsThroughThePreferenceStore(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()
	want := Appearance{Scheme: "Windows Light", Font: "Some Font", Mono: "Some Mono", TextSize: 14, Scale: 1.2}
	want.Save(app.Preferences())
	require.Equal(t, want, LoadAppearance(app.Preferences()))
}

func TestAStaleSchemeNameLoadsAsTheDefaultScheme(t *testing.T) {
	a := Appearance{Scheme: "Renamed Since", Font: "Uninstalled Since"}
	th := a.Theme()
	require.Equal(t, DefaultScheme().Name, th.Palette().Name)
	require.Nil(t, th.font)
}

func TestApplySavesAndSetsTheTheme(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()
	a := DefaultAppearance()
	a.Scheme = "macOS Light"
	a.Apply(app)
	require.Equal(t, "macOS Light", app.Preferences().String(PrefScheme))
	th, ok := app.Settings().Theme().(Theme)
	require.True(t, ok)
	require.Equal(t, "macOS Light", th.Palette().Name)
}

func TestScalePreferenceYieldsToTheEnvironment(t *testing.T) {
	t.Setenv(ScaleEnv, "")
	require.False(t, ApplyScale(0), "system scale sets nothing")
	require.True(t, ApplyScale(1.2))
	t.Setenv(ScaleEnv, "1.50")
	require.False(t, ApplyScale(2), "an explicit FYNE_SCALE wins")
}

func TestScaleLabelsRoundTrip(t *testing.T) {
	for _, s := range ScaleChoices() {
		require.InDelta(t, s, ScaleValue(ScaleLabel(s)), 0.001)
	}
	require.Equal(t, "System", ScaleLabel(0))
	require.Equal(t, "1.2x", ScaleLabel(1.2))
	require.InDelta(t, 0, ScaleValue("garbage"), 0.001)
}
