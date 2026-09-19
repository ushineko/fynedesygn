package glance_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ushineko/fynedesygn/glance"
)

// The rule these tests pin: a value's width does not depend on the value. A
// glance window sizes itself to its content, so a number that widens as it
// crosses a magnitude widens the window under the eye of someone reading it.

func TestARateIsTheSameWidthAtEveryMagnitude(t *testing.T) {
	values := []float64{0, 1, 999, 1023, 1024, 3686, 999_900, 1 << 20, 1 << 30, 1 << 40, 1 << 50}

	for _, v := range values {
		assert.Equal(t, glance.RateWidth, glance.Width(glance.Rate(v)),
			"rate %v is %q", v, glance.Rate(v))
	}
}

func TestAnUnknownRateIsTheSameWidthAsAKnownOne(t *testing.T) {
	assert.Equal(t, glance.Width(glance.Rate(3686)), glance.Width(glance.NoRate()))
	assert.Contains(t, glance.NoRate(), glance.Blank)
}

func TestASizeIsTheSameWidthAtEveryMagnitude(t *testing.T) {
	values := []float64{0, 512, 1024, 999_900, 1 << 30, 1 << 40, 1 << 50}

	for _, v := range values {
		assert.Equal(t, glance.SizeWidth, glance.Width(glance.Size(v)),
			"size %v is %q", v, glance.Size(v))
	}
	assert.Equal(t, glance.SizeWidth, glance.Width(glance.NoSize()))
}

// The magnitude boundary is where a naive formatter jitters: 999.9 KiB/s is
// five characters of number and 1.0 MiB/s is three, and the unit grows by none
// while the suffix stays. Both sides must land in the same columns.
func TestTheMagnitudeBoundaryDoesNotMoveTheColumn(t *testing.T) {
	below := glance.Rate(1023.9 * 1024)
	above := glance.Rate(1024 * 1024)

	require.Equal(t, glance.Width(below), glance.Width(above))
	// The unit begins one past the numeric column in both, which is the
	// property that keeps the two rows aligned.
	assert.Equal(t, glance.NumberWidth, strings.Index(strings.TrimLeft(below, " "), " ")+
		glance.Width(below)-glance.Width(strings.TrimLeft(below, " ")),
		"number column ends elsewhere in %q", below)
	assert.Equal(t, glance.NumberWidth, strings.Index(strings.TrimLeft(above, " "), " ")+
		glance.Width(above)-glance.Width(strings.TrimLeft(above, " ")),
		"number column ends elsewhere in %q", above)
}

// The overflow this promotion prevents: 1023.9 KiB is below the 1024 step and
// still prints six digits into a five-character column.
func TestAValueJustBelowTheStepPromotesRatherThanOverflowing(t *testing.T) {
	r := glance.Rate(1023.9 * 1024)

	assert.Equal(t, glance.RateWidth, glance.Width(r))
	assert.Contains(t, r, "MiB/s", "expected promotion to the next unit, got %q", r)
}

func TestABelowKilobyteRateIsWholeAndAboveItHasOneDecimal(t *testing.T) {
	assert.Contains(t, glance.Rate(900), "900 B/s")
	assert.Contains(t, glance.Rate(3686), "3.6 KiB/s")
}

// A whole number of megabytes still prints its decimal place. A formatter that
// dropped it would move every digit left once an hour, at round numbers.
func TestARoundValueKeepsItsDecimalPlace(t *testing.T) {
	assert.Contains(t, glance.Rate(2*1024*1024), "2.0 MiB/s")
	assert.Equal(t, glance.RateWidth, glance.Width(glance.Rate(2*1024*1024)))
}

func TestAPercentIsTheSameWidthFromZeroToAHundred(t *testing.T) {
	for _, v := range []float64{0, 7, 87, 100} {
		assert.Equal(t, glance.PercentWidth, glance.Width(glance.Percent(v)),
			"percent %v is %q", v, glance.Percent(v))
	}
	assert.Equal(t, glance.PercentWidth, glance.Width(glance.NoPercent()))
}

// Clamping rather than widening: the column was sized for 0..100, and a caller
// passing 1000 gets a wrong number rather than a window that changed shape.
func TestAPercentOutsideTheRangeIsClampedNotWidened(t *testing.T) {
	assert.Equal(t, glance.Percent(100), glance.Percent(1000))
	assert.Equal(t, glance.Percent(0), glance.Percent(-5))
}

func TestAQuantityIsTheSameWidthAcrossItsRange(t *testing.T) {
	const unit = "°C"
	w := glance.Width(glance.Quantity(53, unit, 2))

	for _, v := range []float64{-5, 0, 9.9, 46.6, 100, 999.9} {
		assert.Equal(t, w, glance.Width(glance.Quantity(v, unit, 2)),
			"quantity %v is %q", v, glance.Quantity(v, unit, 2))
	}
	assert.Equal(t, w, glance.Width(glance.NoQuantity(unit, 2)))
}

// A unit column is measured in runes, not bytes: "°C" is three bytes and two
// characters, and padding by bytes would leave the column a character short.
func TestAUnitColumnIsMeasuredInCharactersNotBytes(t *testing.T) {
	degrees := glance.Quantity(46.6, "°C", 3)
	rpm := glance.Count(1298, glance.NumberWidth, "rpm")

	assert.Equal(t, glance.Width(degrees), glance.Width(rpm),
		"%q and %q should occupy the same column", degrees, rpm)
}

func TestACountIsTheSameWidthAcrossItsDigits(t *testing.T) {
	w := glance.Width(glance.Count(1298, 5, "rpm"))

	for _, v := range []int64{0, 7, 500, 1298, 65535} {
		assert.Equal(t, w, glance.Width(glance.Count(v, 5, "rpm")))
	}
	assert.Equal(t, w, glance.Width(glance.NoCount(5, "rpm")))
}

func TestFitPadsToACommonColumnAndNeverTruncates(t *testing.T) {
	values := []string{glance.Percent(87), glance.Rate(3686)}
	w := glance.Widest(values...)

	for _, v := range values {
		assert.Equal(t, w, glance.Width(glance.Fit(v, w)))
	}
	// Asking for less than the value needs returns the value, not a cut one.
	assert.Equal(t, glance.Rate(3686), glance.Fit(glance.Rate(3686), 2))
}
