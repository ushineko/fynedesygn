//go:build !linux && !darwin && !windows

package theme

// DefaultScheme is the scheme used when no preference is stored and the one a
// stale preference falls back to. Platforms without a native scheme here get
// Breeze Dark.
func DefaultScheme() Palette { return BreezeDark }
