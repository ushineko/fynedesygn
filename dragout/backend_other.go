//go:build !cgo || !linux || android || test_web_driver

package dragout

import "fyne.io/fyne/v2"

// noBackend is every build without a drag source: no cgo, the test driver,
// and the platforms not done yet (Windows and macOS, spec 055 phase 2).
type noBackend struct{}

func newBackend() backend { return noBackend{} }

func (noBackend) supports(fyne.Window) bool         { return false }
func (noBackend) prepare(fyne.Window)               {}
func (noBackend) start(fyne.Window, []string) error { return ErrUnsupported }
func (noBackend) release(fyne.Window)               {}
