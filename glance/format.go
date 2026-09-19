package glance

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// The width of a formatted value must not depend on the value. A glance window
// sizes itself to its content, so a number that grows a character wide when it
// crosses a magnitude grows the window, and the panel moves under the eye of
// someone who was only reading it. Every formatter here pads to the width of
// the widest value it can produce, and every one of them has a test asserting
// that width is constant across the range.
//
// The padding is only half of it: the value must also be drawn in the monospace
// face, or the digits are not the same width as each other. Use Row, which
// does that.

// NumberWidth is the width of the numeric column, in characters. Five holds
// "999.9", which is the widest a scaled number gets before it becomes "1.0" of
// the next unit up.
const NumberWidth = 5

// Blank is what a value that is not known reads as. It is not "0", which is a
// reading, and not "", which is a row that lost its value.
const Blank = "--"

// byteUnits are the binary magnitudes, ascending. Capacities and rates are
// reported in powers of 1024 by /proc, sysfs and every tool that reads them,
// so converting to powers of 1000 for display would put a different number on
// screen from the one the source reported.
var byteUnits = []string{"B", "KiB", "MiB", "GiB", "TiB", "PiB"}

// unitWidth is the width of the unit column for a set of units plus a suffix:
// the longest unit, so "B/s" and "KiB/s" end in the same column.
func unitWidth(units []string, suffix string) int {
	w := 0
	for _, u := range units {
		if n := utf8.RuneCountInString(u); n > w {
			w = n
		}
	}
	return w + utf8.RuneCountInString(suffix)
}

// Pad lays a number and its unit into fixed columns: the number right-aligned
// so the digits line up, the unit left-aligned so it does not move when the
// number shortens. The two are separated by one space.
//
// It is the primitive the other formatters are built from, and is exported for
// a program formatting a quantity this package does not know about.
func Pad(number string, numberWidth int, unit string, unitWidth int) string {
	return fmt.Sprintf("%*s %-*s", numberWidth, number, unitWidth, unit)
}

// scale reduces n to the largest unit it exceeds, returning the scaled value
// and the unit. Values below the first step keep their unit and are reported
// whole: a byte count has no fractional part worth a decimal place.
//
// It promotes one unit further when the formatted number would not fit the
// numeric column. Dividing only while n >= step is not enough: 1023.9 KiB is
// below the step and still prints "1023.9", six characters in a five-character
// column, which is the jitter this package exists to prevent. Rounding is what
// decides, not the raw value, so 1023.96 KiB promotes rather than printing
// "1024.0".
func scale(n float64, units []string, step float64, numberWidth int) (value float64, unit string, whole bool) {
	i := 0
	for i < len(units)-1 {
		if n >= step || Width(number(n, i == 0)) > numberWidth {
			n /= step
			i++
			continue
		}
		break
	}
	return n, units[i], i == 0
}

// number formats a scaled value: whole below the first step, one decimal place
// above it. One decimal always, never "conditionally", because a value that
// drops its decimal at a round number is a value that moves the column.
func number(v float64, whole bool) string {
	if whole {
		return fmt.Sprintf("%.0f", v)
	}
	return fmt.Sprintf("%.1f", v)
}

// Rate formats a byte rate: "  3.6 KiB/s". Always RateWidth characters wide.
func Rate(bytesPerSecond float64) string {
	v, u, whole := scale(bytesPerSecond, byteUnits, 1024, NumberWidth)
	return Pad(number(v, whole), NumberWidth, u+"/s", unitWidth(byteUnits, "/s"))
}

// RateWidth is the width of every string Rate returns, including an unknown one.
var RateWidth = NumberWidth + 1 + unitWidth(byteUnits, "/s")

// NoRate is an unknown rate, padded to the width of a known one so a source
// that has not answered yet does not resize the window when it does.
func NoRate() string { return Pad(Blank, NumberWidth, "", unitWidth(byteUnits, "/s")) }

// Size formats a byte count: "  2.7 GiB". Always SizeWidth characters wide.
func Size(bytes float64) string {
	v, u, whole := scale(bytes, byteUnits, 1024, NumberWidth)
	return Pad(number(v, whole), NumberWidth, u, unitWidth(byteUnits, ""))
}

// SizeWidth is the width of every string Size returns, including an unknown one.
var SizeWidth = NumberWidth + 1 + unitWidth(byteUnits, "")

// NoSize is an unknown byte count, padded to the width of a known one.
func NoSize() string { return Pad(Blank, NumberWidth, "", unitWidth(byteUnits, "")) }

// percentWidth holds "100".
const percentWidth = 3

// Percent formats a percentage: " 87%". Values are clamped to 0..100, because
// the width is chosen for that range and a caller passing 1000 would widen the
// column rather than be told it was wrong.
func Percent(v float64) string {
	if v < 0 {
		v = 0
	}
	if v > 100 {
		v = 100
	}
	return Pad(fmt.Sprintf("%.0f", v), percentWidth, "%", 1)
}

// PercentWidth is the width of every string Percent returns.
const PercentWidth = percentWidth + 1 + 1

// NoPercent is an unknown percentage, padded to the width of a known one.
func NoPercent() string { return Pad(Blank, percentWidth, "%", 1) }

// Quantity formats a value with one decimal place and a unit of the caller's
// choosing: temperatures, voltages, anything measured rather than counted.
// unitColumn is the width to reserve for the unit, which a caller with more
// than one unit in a card sets to the longest of them so the rows line up.
//
// The unit is the caller's string and not this package's, because the glyphs a
// unit needs are the caller's problem: the bundled font does not cover every
// symbol, and one it does not cover marks the run boundary as a missing glyph
// (quirk 19). Degrees and percent are covered; check anything rarer.
func Quantity(v float64, unit string, unitColumn int) string {
	return Pad(fmt.Sprintf("%.1f", v), NumberWidth, unit, unitColumn)
}

// NoQuantity is an unknown Quantity, padded to the same width.
func NoQuantity(unit string, unitColumn int) string {
	return Pad(Blank, NumberWidth, unit, unitColumn)
}

// Count formats a whole number with a unit: " 1298 rpm". digits is the width to
// reserve, chosen for the largest value the source can report rather than the
// current one.
func Count(v int64, digits int, unit string) string {
	return Pad(fmt.Sprintf("%d", v), digits, unit, utf8.RuneCountInString(unit))
}

// NoCount is an unknown Count, padded to the same width.
func NoCount(digits int, unit string) string {
	return Pad(Blank, digits, unit, utf8.RuneCountInString(unit))
}

// Width is the rune count of a formatted value, which is what the tests assert
// is constant and what a caller sizing a column measures. It is rune count and
// not byte length because a unit may carry a multi-byte glyph.
func Width(s string) int { return utf8.RuneCountInString(s) }

// Widest returns the width of the longest of several formatted values, for a
// caller laying out a column of quantities in different units.
func Widest(values ...string) int {
	w := 0
	for _, v := range values {
		if n := Width(v); n > w {
			w = n
		}
	}
	return w
}

// Fit pads s on the right to width, so several formatters with different
// natural widths can share one column. It never truncates: a value cut to fit
// is a value read wrong, and the window is allowed to be wider than planned.
func Fit(s string, width int) string {
	if n := Width(s); n < width {
		return s + strings.Repeat(" ", width-n)
	}
	return s
}
