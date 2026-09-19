/*
Package glance builds always-on-top status panels: small frameless windows
that sit on the desktop and are read without being interacted with.

It is the second of this module's two window archetypes. Package shell is the
first — a header, a section navigation, one content scroller and a status bar,
for a window someone works in. A glance window has none of those. The window is
the content: a stack of cards, each a metric that is either worth a glance or
not drawn at all, sized to exactly what it is showing.

The rules are in docs/glance.md, which also records what the pinned Fyne cannot
do here. Two constraints shape the whole package:

  - A window cannot be translucent (quirk 32). Cards are opaque and separate
    themselves by contrast. Whole-window opacity is the compositor's to grant,
    and the program asks for it through the desktop's own configuration —
    package glance/kwin does this for KDE Plasma.
  - A fixed-size window grows to fit its content and never shrinks back on its
    own (quirk 34). Panel.Resize is what takes it down again, and it is called
    whenever a card appears or disappears.

A program builds a window, adds cards, and polls:

	w := glance.NewWindow(a, glance.Options{Title: "Sensors", OnTop: true})

	temp := glance.NewRow("Coolant", glance.NoQuantity("°C", 2))
	card := glance.NewCard("AIO")
	card.AddRow(temp)
	w.Panel().Add(card)

	// from a poll, on the UI thread:
	temp.Set(glance.Known(glance.Quantity(46.6, "°C", 2), fd.StatusGood))
	card.SetAvailable(true)

	w.ShowAndRun()

Values are formatted through this package rather than with fmt, because the
width of a formatted value must not depend on the value: the window is sized to
its content, so a number that widens as it crosses a magnitude widens the
window under the eye of someone who was only reading it. Rate, Size, Percent,
Quantity and Count pad to a fixed width, and each has a matching blank of the
same width for a reading that has not arrived.

Cards are drawn only when the user allows them and their source has something
to say, and those are separate questions: a card the user hid and a card with
no data look the same and arrive by different paths. A card that is not drawn
stops its poll, so a glance window on a machine without the sensor it watches
costs what a window built without the card would.

Transcribed from ag-scripts/peripheral-battery-monitor (PyQt6, same author),
which carried this archetype through its 1.x series.
*/
package glance
