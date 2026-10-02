//go:build !windows

package glance

// dragsToMove is false where the desktop moves a frameless window itself: on
// Linux, with Alt-drag or a window rule.
const dragsToMove = false
