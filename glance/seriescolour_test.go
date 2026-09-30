package glance_test

import (
	"image/color"
	"testing"

	"github.com/stretchr/testify/assert"

	fd "github.com/ushineko/fynedesygn"
	"github.com/ushineko/fynedesygn/glance"
	"github.com/ushineko/fynedesygn/theme"
	"github.com/ushineko/fynedesygn/widgets"
)

func nrgba(c color.Color) color.NRGBA {
	n, _ := color.NRGBAModel.Convert(c).(color.NRGBA)
	return n
}

// The cycle is the theme's link colour and then three fixed categorical
// colours, wrapping after the fourth in both directions.
func TestSeriesColourCyclesAndWraps(t *testing.T) {
	for _, p := range theme.Schemes() {
		th := theme.New(p, theme.Options{})
		assert.Equal(t, nrgba(p.Link), nrgba(glance.SeriesColour(th, 0)), "%s: series 0 is the link colour", p.Name)
		for i := range 4 {
			assert.Equal(t, nrgba(glance.SeriesColour(th, i)), nrgba(glance.SeriesColour(th, i+4)), "%s: series %d wraps", p.Name, i+4)
		}
		assert.Equal(t, nrgba(glance.SeriesColour(th, 3)), nrgba(glance.SeriesColour(th, -1)), "%s: a negative index wraps too", p.Name)
	}
}

// The four series are told apart from each other and from the two status
// colours that alarm, in every scheme. The palette alone could not do this:
// its selection accent is StatusInfo's colour, positive is StatusGood's, hover
// equals the accent in four schemes and link equals it in two.
func TestSeriesColoursAreDistinctAndNotWarnOrBad(t *testing.T) {
	for _, p := range theme.Schemes() {
		th := theme.New(p, theme.Options{})
		warn := nrgba(th.Color(widgets.StatusColorName(fd.StatusWarn), 0))
		bad := nrgba(th.Color(widgets.StatusColorName(fd.StatusBad), 0))

		seen := map[color.NRGBA]int{}
		for i := range 4 {
			c := nrgba(glance.SeriesColour(th, i))
			if j, dup := seen[c]; dup {
				t.Errorf("%s: series %d is the same colour as series %d", p.Name, i, j)
			}
			seen[c] = i
			assert.NotEqual(t, warn, c, "%s: series %d is the warn colour", p.Name, i)
			assert.NotEqual(t, bad, c, "%s: series %d is the bad colour", p.Name, i)
		}
	}
}

// The categorical three follow the background: a dark scheme gets the light
// weights and a light scheme the dark ones.
func TestSeriesColourFollowsTheBackground(t *testing.T) {
	dark := theme.New(theme.BreezeDark, theme.Options{})
	light := theme.New(theme.BreezeLight, theme.Options{})

	for i := 1; i < 4; i++ {
		assert.NotEqual(t, nrgba(glance.SeriesColour(dark, i)), nrgba(glance.SeriesColour(light, i)), "series %d", i)
	}
}

func TestFadedKeepsTheHueAndLowersTheAlpha(t *testing.T) {
	c := color.NRGBA{R: 0x3d, G: 0xae, B: 0xe9, A: 0xff}

	f := nrgba(glance.Faded(c))

	assert.Equal(t, [3]uint8{c.R, c.G, c.B}, [3]uint8{f.R, f.G, f.B}, "the hue is the primary's")
	assert.Less(t, f.A, c.A, "the alpha is lower")
	assert.Positive(t, f.A, "and not gone")
}
