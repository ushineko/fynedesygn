//go:build !windows

package glance

import "fyne.io/fyne/v2"

// movesItself is false where only the desktop moves a window and a program
// cannot read where its window is (quirk 33): Wayland above all. A drawer on
// the left and a drawer that chooses its side are then the compositor's to
// help with; see Drawer.SetEdgeHeld and Drawer.Choose.
const movesItself = false

// moveWindowBy does nothing here: the compositor holds the edge, or the
// drawer opens on the right.
func moveWindowBy(fyne.Window, float32) {}

// roomBeside cannot answer here. A program asks the compositor and tells the
// drawer with Drawer.Choose.
func roomBeside(fyne.Window) (left, right float32, ok bool) { return 0, 0, false }

// yieldFocus reports that this platform has no way of its own: a Wayland
// client cannot activate another program's window. Drawer.SetYield is the
// program's way.
func yieldFocus(fyne.Window) bool { return false }

// hookWindow has nothing to wrap here.
func hookWindow(fyne.Window) {}

// beginMove does nothing here: the desktop moves a frameless window, with
// Meta-drag or a window rule.
func beginMove(fyne.Window) {}
