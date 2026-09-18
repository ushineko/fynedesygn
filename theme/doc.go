/*
Package theme holds the colour schemes, the fyne.Theme that applies them, the
system font scanner, the appearance preferences and the platform fixes that
make a Fyne window sit well next to the rest of a desktop.

Ported from angou's theme.go, fonts.go and cursor_linux.go (same author, MIT),
with clockwork-orange's separate monospace face and interface scale.

Nine schemes ship: Breeze Dark and Light and Oxygen Dark (transcribed from
KDE's .colors files), Adwaita Dark and Light (libadwaita's named colours),
Windows Dark and Light (Windows 11 Fluent tokens) and macOS Dark and Light
(Apple's system colours, with the corner radius of macOS 26 Tahoe). They are
compiled in: a window built on this package does not follow the desktop's
current scheme and does not need any desktop installed.

The package deliberately does not follow the operating system's light or dark
switch. A Palette declares whether it is dark, and Theme ignores the variant
Fyne passes, because honouring both could paint dark text on a dark window.

Importing this package alongside fyne.io/fyne/v2/theme needs an alias on one
of them; the examples use fdtheme for this one.
*/
package theme
