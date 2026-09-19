/*
Package kwin installs the KDE Plasma window rule a glance window needs.

Three things an always-on-top status panel wants are the compositor's to grant,
and the pinned Fyne reaches only two of them:

  - staying above other windows — desktop.Window.RequestAlwaysOnTop, which the
    window manager may decline;
  - losing its titlebar — a splash window, which is undecorated;
  - being translucent — nothing at all. The GLFW backend never requests a
    transparent framebuffer, so no code in the process can draw through the
    window (quirk 32).

A KWin rule grants all three, to a window whose toolkit knows nothing about it.
This package writes that rule into ~/.config/kwinrulesrc:

	err := kwin.Install(kwin.Rule{
		AppID:       a.UniqueID(),
		Description: "Sensors panel",
		AlwaysOnTop: true,
		NoBorder:    true,
		Opacity:     95,
	})

Installing is an explicit step a program offers, the same way it installs a
desktop entry, and Remove undoes it. **Nothing here runs on its own.** A glance
window that silently edited the user's compositor configuration on first run
would be a program that changed the desktop without being asked.

The file holds every window rule the user has, written by System Settings, so
this package changes only the rule matching its own AppID and keeps every other
line — including comments and section order — as it found them. Keys are case
sensitive: KWin reads `wmclass`, not `WMClass`.

# Matching

KWin matches by window class. On Wayland that is the `app_id`, which Fyne sets
from fyne.App.UniqueID, so the app's unique ID, its desktop file basename and
Rule.AppID are the same string — the same identity the taskbar icon depends on
(docs/design-system.md, Platform).

# Position is not here

KWin's `position` rule selects a screen and snaps to its origin on Wayland; it
does not honour intra-screen coordinates. A rule that half-works is worse than
none, so this package does not write one and clears one an earlier version may
have left. Placing a window exactly means driving KWin's Scripting D-Bus API
(org.kde.KWin at /Scripting), which is not implemented here — see
docs/glance.md, Desktop integration, for what it involves.

# Telling KWin

KWin does not re-read its rules until it is asked, over the session bus:

	destination org.kde.KWin
	path        /KWin
	interface   org.kde.KWin
	method      reconfigure

ReconfigureCall returns those coordinates rather than making the call, because
this module's core packages take no dependency beyond Fyne and a D-Bus client
is not Fyne. A program that already has a session bus connection makes the call
itself; one that does not can tell the user to log out and back in, which has
the same effect.

# Other desktops

There is no equivalent package for GNOME, Windows or macOS yet, and none is
guessed at. A glance window on those desktops is the Fyne-native subset:
frameless, on top if the window manager agrees, opaque.
*/
package kwin
