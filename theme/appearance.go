package theme

import (
	"fmt"
	"os"
	"strconv"

	"fyne.io/fyne/v2"

	"github.com/ushineko/fynedesygn/settings"
)

// Preference keys. Namespaced so a later setting cannot collide with one of
// these by accident. They are the only keys this module stores: every setting
// a program's command-line front end can also see belongs in the program's
// own configuration, so the two front ends agree.
const (
	PrefScheme   = "appearance.scheme"
	PrefFont     = "appearance.font"
	PrefMono     = "appearance.mono"
	PrefTextSize = "appearance.textSize"
	// PrefScale is the interface scale multiplier, applied through FYNE_SCALE
	// at window creation (see ApplyScale). 0 means the system's.
	PrefScale = "appearance.scale"
)

// Appearance is the user's look-and-feel choices: the whole of what makes a
// Fyne window look like it belongs on the user's desktop, since Fyne draws its
// own widgets.
type Appearance struct {
	// Scheme is a Palette name; unknown names fall back to DefaultScheme.
	Scheme string `json:"scheme"`
	// Font is an interface family name or DefaultFontName.
	Font string `json:"font"`
	// Mono is the monospace family name or DefaultFontName.
	Mono string `json:"mono"`
	// TextSize in points; 0 means DefaultTextSize.
	TextSize float32 `json:"textSize"`
	// Scale is the interface scale; 0 means the system's.
	Scale float32 `json:"scale"`
}

// SettingsKey is the appearance's section in a settings store.
const SettingsKey = settings.Prefix + "appearance"

// DefaultAppearance is what a fresh installation gets.
func DefaultAppearance() Appearance {
	return Appearance{
		Scheme:   DefaultScheme().Name,
		Font:     DefaultFontName,
		Mono:     DefaultFontName,
		TextSize: DefaultTextSize,
	}
}

// LoadAppearance reads the saved appearance, falling back to the defaults. A
// stale value (a scheme since renamed, a font since uninstalled) falls back
// rather than failing: SchemeByName and LoadFont both tolerate an unknown
// name.
func LoadAppearance(p fyne.Preferences) Appearance {
	d := DefaultAppearance()
	return Appearance{
		Scheme:   p.StringWithFallback(PrefScheme, d.Scheme),
		Font:     p.StringWithFallback(PrefFont, d.Font),
		Mono:     p.StringWithFallback(PrefMono, d.Mono),
		TextSize: float32(p.FloatWithFallback(PrefTextSize, float64(d.TextSize))),
		Scale:    float32(p.FloatWithFallback(PrefScale, 0)),
	}
}

/*
LoadAppearanceFrom reads the appearance from a settings store, migrating once
from the preference store it used to live in.

A program updated to a version that keeps its settings in a file must not open
in a theme its user never chose, so an installation with nothing in the file
takes what is in Fyne's preferences and writes it through. The old keys are read
and not deleted: a rollback finds them where they were.
*/
func LoadAppearanceFrom(st *settings.Store, p fyne.Preferences) Appearance {
	a := DefaultAppearance()
	if st != nil && st.Get(SettingsKey, &a) {
		return a
	}
	// Only when the preference store actually holds something. A program nobody
	// has configured gets the defaults without a settings file being written
	// for it, so the file appears the first time a choice is made.
	if p == nil || p.String(PrefScheme) == "" {
		return a
	}
	a = LoadAppearance(p)
	if st != nil {
		_ = st.Set(SettingsKey, a)
	}
	return a
}

// SaveTo writes the appearance to a settings store.
func (a Appearance) SaveTo(st *settings.Store) {
	if st == nil {
		return
	}
	_ = st.Set(SettingsKey, a)
}

// Save writes the appearance to the preference store.
func (a Appearance) Save(p fyne.Preferences) {
	p.SetString(PrefScheme, a.Scheme)
	p.SetString(PrefFont, a.Font)
	p.SetString(PrefMono, a.Mono)
	p.SetFloat(PrefTextSize, float64(a.TextSize))
	p.SetFloat(PrefScale, float64(a.Scale))
}

// Theme builds the fyne.Theme these choices describe.
func (a Appearance) Theme() Theme {
	return New(SchemeByName(a.Scheme), Options{
		Font:     LoadFont(a.Font),
		Mono:     LoadFont(a.Mono),
		TextSize: a.TextSize,
	})
}

// Apply saves the appearance and sets the app's theme from it. The scale is
// saved but not applied: Fyne fixes a window's scale when the window is
// created, so a changed scale takes effect when the window next opens.
func (a Appearance) Apply(app fyne.App) {
	a.Save(app.Preferences())
	app.Settings().SetTheme(a.Theme())
}

// ScaleEnv is Fyne's scale override, read when a window is created.
const ScaleEnv = "FYNE_SCALE"

// ScaleChoices are the interface scales the Appearance picker offers; 0 is
// "System".
func ScaleChoices() []float32 { return []float32{0, 1.1, 1.2, 1.3, 1.5, 1.75, 2} }

/*
ApplyScale hands an interface scale to Fyne and reports whether it did.

Fyne has no per-application scale setting: FYNE_SCALE in the environment or a
scale in Fyne's own settings file, which every Fyne program on the machine
reads. But the environment variable is read when a window is created, not when
the process starts, so setting it here, after the app exists and before the
window does, scopes it to this program. An explicit FYNE_SCALE from the shell
wins, so a capture script can still force one.

Why it exists: Fyne's text has no hinting, and on a fractional-scale Wayland
desktop it reads soft; drawn a fifth larger it reads well. A user who wants
that should not have to know an environment variable.
*/
func ApplyScale(scale float32) bool {
	if os.Getenv(ScaleEnv) != "" || scale <= 0 {
		return false
	}
	return os.Setenv(ScaleEnv, strconv.FormatFloat(float64(scale), 'f', 2, 32)) == nil
}

// ScaleLabel names one scale choice; anything not in the table is shown as
// its number so a value set by hand is not silently replaced.
func ScaleLabel(s float32) string {
	if s <= 0 {
		return "System"
	}
	return fmt.Sprintf("%gx", s)
}

// ScaleValue is the inverse of ScaleLabel.
func ScaleValue(label string) float32 {
	if label == "System" {
		return 0
	}
	var f float32
	if _, err := fmt.Sscanf(label, "%g", &f); err != nil || f <= 0 {
		return 0
	}
	return f
}
