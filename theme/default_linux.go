//go:build linux

package theme

// DefaultScheme is the scheme used when no preference is stored and the one a
// stale preference falls back to. It is the platform's native dark scheme:
// Breeze Dark on Linux.
func DefaultScheme() Palette { return BreezeDark }
