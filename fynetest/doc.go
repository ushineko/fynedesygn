/*
Package fynetest holds headless test helpers for programs built on this
library: walkers that reach every object in a Fyne tree, finders for the
widget a test wants to poke, a Text function that returns everything a tree
says, and a Sandbox that keeps a test off the developer's home directory.

The rule the helpers serve: tests pin behaviour, not layout. A test asserting
that a section holds a Border holding a VBox tests the code against itself and
breaks on every rearrangement. A test that asks Text for the facts on screen,
or FindButton for the one button it needs, survives a redesign and still fails
when the behaviour changes.

Two walkers exist because Fyne hides content in two ways. Walk follows the
object tree a program builds (containers, scrolls, tabs). WalkRendered also
opens every widget through test.WidgetRenderer to see what it draws, which is
how a test learns that a RichText paints a code block inside a scroller.
WalkRendered, and everything built on it, needs a running test app, because
creating a renderer asks the current app for its theme.

Ported from the test helpers of nmsbonker and clockwork-orange (same author,
MIT).
*/
package fynetest
