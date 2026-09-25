# Glance windows

A glance window is a small panel on the desktop. It has no frame, it stays
above other windows, and a person reads it without touching it. It has no
header, no navigation, no status bar and no scroller. The window is its
content: a stack of cards. Each card is one measurement that is worth a look,
or the card is absent.

[design-system.md](design-system.md) holds the rules for an application
window, which is the shell with its header, navigation, content scroller and
status bar. This page holds the rules for the other shape.

The two shapes share `Status`, the palette, the fonts, the threading rules and
the voice of the copy. They differ in almost all of their structure, and this
page lists the differences below so that you do not have to find them.

The shape comes from `ag-scripts/peripheral-battery-monitor`, which the same
author wrote in PyQt6. That program has used the shape through its 1.x
releases, for peripheral battery, network bandwidth, liquid-cooler
temperatures and Claude Code usage. Where a rule below has a reason, the
reason comes from the use of that program.

Package `glance` implements this: the window, the card stack, the rows, the
fixed-width formatters and the sparkline. Package `glance/kwin` installs the
KDE Plasma window rule the desktop half needs. `examples/glance-monitor` is a
working panel in this shape, and its tests are where most of the rules below
are pinned.

The package is smaller than that program, by design. Where this page
describes something that the package does not contain, the text says so at
that point.

![A small frameless panel on a desktop, four cards stacked, each a rounded surface a step lighter than the panel behind it. "Peripherals" shows a mouse at 87% in green. "Bandwidth" shows two interface rows with rates right-aligned in a monospace column. "AIO" shows CPU 74.5 °C in blue, Coolant 50.1 °C in amber and Fans 1368 rpm, over a sparkline whose two traces are drawn in the colours of the rows they came from. "Usage" shows two meters, each a label, a monospace caption and a bar: a green one at 48% of a five-hour window, an amber one at 82% of a monthly budget. No titlebar, no buttons, no scrollbar.](img/glance-monitor.png)

This is `examples/glance-monitor` with the KDE window rule installed. Without
the rule, KWin draws a titlebar on the window. `glance/kwin` exists to remove
that titlebar: a splash window asks GLFW for no decoration, and the compositor
decorates it anyway.

## Contents

- [What inverts](#what-inverts)
- [What Fyne 2.8.1 gives and withholds](#what-fyne-281-gives-and-withholds)
- [The window](#the-window)
- [Desktop integration](#desktop-integration)
- [Opacity without alpha](#opacity-without-alpha)
- [The card stack](#the-card-stack)
- [Sizing](#sizing)
- [Numbers that do not jitter](#numbers-that-do-not-jitter)
- [Meters](#meters)
- [Colour is the legend](#colour-is-the-legend)
- [Trend plots](#trend-plots)
- [The context menu is the whole interface](#the-context-menu-is-the-whole-interface)
- [Degradation](#degradation)
- [Alerting is not the window's job](#alerting-is-not-the-windows-job)
- [Cadence](#cadence)
- [Copy](#copy)
- [Testing](#testing)
- [Numbers](#numbers)

## What inverts

| design-system.md says | A glance window says | Why |
|---|---|---|
| Default window 1180 x 760, `SetMaster()` | The window is whatever its content measures, and never larger | There is no content to page through. A glance window that needed sizing would be an application window |
| One scroll per section; the shell owns a scroller | **No scroller anywhere.** Content that does not fit is content that should have been hidden | A scrollbar is an instruction to interact. Nobody scrolls a widget they are reading past |
| Header carries the app name and a few actions | No header, no app name, no buttons | The name is knowledge the reader already has. Chrome is the majority of a 260 px window |
| Nav is a list of section titles | No nav. Every card is visible at once | Four cards do not need an index |
| Nothing transient may reflow the interface | Still binding, and harder: the window itself resizes, so a card appearing moves the whole panel | See [Sizing](#sizing) |
| A control says what it is in its label; guidance goes in a hover tip | Almost nothing is a control. Configuration is in the context menu | See [The context menu](#the-context-menu-is-the-whole-interface) |
| A button that cannot be used is disabled, never hidden | A card with no data is **hidden**, not disabled or blanked | A disabled row still occupies a row. An absent sensor should cost nothing |
| Every long call gets a busy indicator | Polls are silent. A stale value is dimmed, never replaced by a spinner | A busy popup over a 260 px window covers the window. See [Degradation](#degradation) |
| Loaded data carries an explicit loaded flag | Unchanged, and load-bearing: "not read yet", "read and absent" and "was read, source has gone" are three different renderings |

Everything that the table does not list is the same in both shapes. `Status`
names the four levels, colour comes from the active scheme, `fyne.Do` is the
only way onto the UI thread, and copy is sentence case that names the
consequence.

## What Fyne 2.8.1 gives and withholds

Verified against `fyne.io/fyne/v2 v2.8.1`, the module's pin.

| Need | Available | Where |
|---|---|---|
| A frameless window | Yes: `desktop.Driver.CreateSplashWindow()` — undecorated, `SetPadded(false)`, centred on screen | `internal/driver/glfw/window.go` |
| Always on top | Yes: `desktop.Window.RequestAlwaysOnTop()`, **before `Show`**; sets the GLFW `Floating` hint | `driver/desktop/window.go` |
| Position the window | Partly: `desktop.Window.RequestPosition(x, y)`. Its own doc says the request "may be ignored (for example Linux Wayland)". A KWin rule can place the window; see Desktop integration | `driver/desktop/window.go` |
| Read the window's position back | **No.** GLFW's position callback is wired internally; nothing on `fyne.Window` exposes a position | quirk 33 |
| A translucent window background | **No.** The desktop backend never requests `glfw.TransparentFramebuffer`, and asks for no alpha bits. Only the wasm backend requests `AlphaBits, 8` | quirk 32 |
| Size to content | Yes, in one direction only: a `SetFixedSize(true)` window grows to its content's minimum size on its own, and shrinks only when the program calls `Resize` explicitly | quirk 34 |
| Know the window was resized | No — quirk 14, already recorded. It does not bite here: a glance window is fixed-size and the program is the only thing that resizes it |
| Show the window without taking the focus | **No.** A splash window takes the focus when it is shown, on Plasma 6 (quirk 36). A KWin rule with `acceptfocus=false` stops it | `glance/kwin` |

Two of these decide the design, and you cannot correct either one inside the
process. They are why this page exists before the code does.

## The window

- One splash window, `SetFixedSize(true)`, `RequestAlwaysOnTop()` called before
  `Show`. `SetMaster()` as usual: closing it ends the program.
- **The desktop moves the window, not the window itself.** Qt has
  `startSystemMove()` and the monitor uses it. Fyne has no equivalent, and a
  drag that calls `RequestPosition` from a mouse handler is a request that the
  compositor can refuse at every frame. On Wayland the compositor places the
  window, and a window rule is the supported way to fix its position. Write
  the rule down, and do not compete for the pixel.
- **A program cannot keep its window position.** Nothing can read the position
  back (quirk 33), so there is nothing to save. The monitor found the same
  result from the other direction, because `moveEvent` does not fire for a move
  that the compositor made on Wayland. It therefore uses the KWin scripting
  D-Bus API from outside Qt. A program that wants its glance window in one
  place supplies a compositor rule, in the same way that it supplies a desktop
  entry.
- **A splash window takes the focus when it is shown** (quirk 36). A panel that
  a person reads and never touches does not want it, and an indicator that
  took the focus from a game at every key press would be a defect. The KWin
  rule below has `NoFocus` for this.
- The desktop file basename must equal the app ID, as for any window, or
  Wayland cannot match the window to its icon (design-system.md, Platform).
- No menu bar, no tabs, no `CenterOnScreen` after the first run.

## Desktop integration

A glance window needs three things: to stay above other windows, to lose its
border, and to be translucent. The compositor grants all three, and Fyne
reaches only the first two. The rest is specific to the desktop, so it is
behind a build tag, like every other platform difference in this module. See
design-system.md, Platform.

This module already does this. `theme.ApplyCursorTheme` reads the Plasma
`kcminputrc` file directly and does not call `kreadconfig`, because a cursor
theme is not a sufficient reason for a library to start a subprocess. Desktop
integration follows the same rule: read and write the configuration of the
desktop, and use its D-Bus interface where one exists.

**The library never writes the compositor configuration of the user by
itself.** The program offers an installation step, in the same way that it
installs a desktop entry, and the step is reversible. A glance window that
edited `kwinrulesrc` at its first run, with no message, would be a program
that changed the desktop without permission.

### KDE Plasma

Package `glance/kwin` contains this. It comes from `install_kwin_rule.py` and
`kwin_window_position.py` in the monitor, which are the tested versions. The
package writes the rule file. It does not use the Scripting D-Bus API for
position; the text below describes that for the specification that adds it.

**How KWin finds the window.** KWin matches the window class. On Wayland that
is the `app_id`, which Fyne sets from `fyne.CurrentApp().UniqueID()`; the
driver gives it to GLFW as the `WaylandAppID` hint. The unique ID of the
program, the base name of the desktop file and the `wmclass` of the rule are
therefore the same string. The icon rule in design-system.md is this same
rule.

**The rule file** is `~/.config/kwinrulesrc`. It has the shape of an INI file
and **its keys are case sensitive**. A `[General]` section holds `rules=`,
which is a list of section names separated by commas, and `count=`. Each rule
is a numbered section.

To install a rule, find the section whose `wmclass` is that of the program, or
add a new section at the end. Never write the file again from the start,
because it holds every other rule that the user has.

| Key | Value | Effect |
|---|---|---|
| `wmclass` / `wmclassmatch` | the app ID / `1` | exact match |
| `above` / `aboverule` | `true` / `2` | keep above, forced |
| `noborder` / `noborderrule` | `true` / `2` | no titlebar, forced |
| `opacityactive` / `opacityactiverule` | 0–100 / `4` | **this is the translucency**, applied initially |
| `opacityinactive` / `opacityinactiverule` | 0–100 / `4` | same when unfocused |
| `title` / `titlematch` | the window title / `1` | exact; for a second window of a program whose main window shares the app ID |
| `skiptaskbar`, `skipswitcher`, `skippager` (each with `…rule`) | `true` / `2` | out of the taskbar, the switcher and the pager, forced |
| `acceptfocus` / `acceptfocusrule` | `false` / `2` | never takes the focus, forced |

In the KWin rule vocabulary, `2` is "Force" and `4` is "Apply Initially". Use
"Apply Initially" for the opacity and not "Force", so that the user can still
change it from the window menu.

**`noborder` is necessary, not an additional precaution.** A splash window
sets the GLFW `Decorated` hint to false, and KWin draws a titlebar on it
anyway. The window in the screenshot at the top of this page has no frame only
because the rule is installed.

`above` is close to a duplicate of `RequestAlwaysOnTop`, and it is still worth
setting. The rule continues after a restart of the compositor, and the
documentation of `RequestAlwaysOnTop` warns that the window manager can decide
that other windows stay above yours.

**A `position` rule does place the window, and this package still does not
write one.** On Plasma 6 on Wayland, measured on 2026-09-24, a rule with
`position=500,1300` forced put a splash window at 500,1300. An earlier version
of this page said that the rule moved the window to the origin of a screen;
that was not measured, and it is wrong for this version of KWin. The package
does not write a position, because a position in a rule is fixed to one
screen. An indicator should appear on the screen the pointer is on, and that
is a move the program makes at each show, by the Scripting API below. The
package removes the old `position`, `size` and `screen` keys of a rule it
rewrites, as before.

**Set the position with the Scripting D-Bus API.** `org.kde.KWin` at
`/Scripting` loads a small KWin script in JavaScript, runs it, and unloads it.
The script finds the window by `resourceClass`. It then sets `frameGeometry`
to move the window, or reads `frameGeometry` and reports the value back over
D-Bus. A loaded KWin script runs one time, so each operation is a new cycle of
load, run and unload.

This answers quirk 33. Fyne cannot read the position, and the compositor gives
it to you if you ask in the language of the compositor.

**To apply a rule change**, call `reconfigure` on `org.kde.KWin` at `/KWin`.
The monitor tries `qdbus6`, then `qdbus`, then `dbus-send`, and writes a
warning rather than failing when none of them works. A Go program makes the
call directly.

`github.com/godbus/dbus/v5` is already in the module graph as a dependency of
Fyne, so you can reach the session bus without a new dependency. To make it a
direct dependency of a library package is a decision for the specification
that does it, because Fyne is the only runtime dependency that the core
packages require.

**All of this is optional at run time.** On a machine with no KWin, a glance
window has no frame, stays above other windows, is opaque, and is where the
compositor put it. That is a program that works, and not a program that has
lost a function.

### Other desktops

These are not written yet, and this page does not guess at them. Each desktop
needs the same three answers, from what that desktop offers: how to stay above
other windows, how to lose the border, and how to be translucent.

- **GNOME and Mutter** have no equivalent window rule for the user. The
  correct answer can be that only the part that Fyne supplies is available.
- **Windows** has `SetWindowPos` with `HWND_TOPMOST`, and layered windows with
  `SetLayeredWindowAttributes` for the alpha of a whole window. Both work on an
  HWND, and Fyne gives you no window handle, so to reach them is a separate
  problem.
- **macOS** has `NSWindow.level` and `alphaValue`, and has the same problem.

Until somebody writes and tests each one on that desktop, a glance window there
has only the functions that Fyne supplies.

## A panel that comes and goes

An indicator is a glance window shown for a moment: the volume after a key
press, a switch of output, a mode that changed. `glance.Transient` holds one.
`Show` shows the window and starts a hold, `DefaultHold` of 1.5 s; a `Show`
during the hold restarts it, so changes that arrive close together keep the
window up rather than making it blink. When the hold runs out the window
hides on its own. `Hide` takes it down now.

- **It is a second window.** The program has a main window, so the indicator
  is made with `Options.Secondary`, which leaves it out of the master role:
  closing or hiding it does not end the program.
- **Its rule matches the title.** The main window and the indicator share
  the app ID, so a rule on the app ID alone would strip the main window's
  titlebar too. Give the indicator a `Title` of its own, and give `kwin.Rule`
  the same `Title`. Set `NoBorder`, `AlwaysOnTop`, the three skips and
  `NoFocus`.
- **Move it before it shows.** `Transient.OnShow` runs before the window is
  shown, once for each run. That is the moment to place the window on the
  screen the pointer is on, which on Wayland is the compositor's to do.
- **Draw a snapshot, as a card does.** An indicator draws one value: a
  `Meter` for a level, a `Row` for a state. The program sets the value and
  calls `Show`; the window does not read anything itself.

## Opacity without alpha

The monitor sets `WA_TranslucentBackground` and offers 100, 95, 90, 80 and
70 %, and the desktop is visible through the panel. **Fyne cannot draw this**
(quirk 32), and a program that waits for it will wait without end.

Fyne can ask the compositor instead. The compositor applies opacity to a
window whose toolkit knows nothing about opacity. On Plasma this is two lines
in a KWin rule, and the monitor already supplies them. See [Desktop
integration](#desktop-integration).

There are therefore two different things, and the design depends only on the
first:

- **The opacity of a whole window is a desktop setting.** It is available
  where the desktop offers it and absent where the desktop does not. Offer it,
  apply it by writing the rule of the desktop, and treat its absence as
  normal.
- **Alpha for each pixel is not available.** A card cannot be more opaque than
  its window, a value cannot appear gradually, and a background cannot take a
  colour from the wallpaper behind it.

Which yields the binding rule:

- **Contrast separates the cards, not alpha.** The cards of the monitor are
  `rgba(43,43,43,α)` on the desktop, with a thin `rgba(255,255,255,20)` border.
  When the window is opaque, that is a card surface one step from the surface
  of the window, with a thin border. The palette already has these three
  colours: the window is `WindowBG`, the card is `ButtonBG` and the border is
  `Separator`. Reach them through the Fyne roles `Background`, `Button` and
  `Separator`.

  Use `ButtonBG` and not `ViewAltBG`. `ViewAltBG` is the token for alternating
  rows, and in the dark schemes it is *darker* than the window, so a card drawn
  in it disappears into the panel. `Card` draws all of this.
- **A card takes a surface radius and not the control radius of the scheme.**
  The radius token of the palette is for entries and buttons, and it is 2 in
  the Breeze, Oxygen and Adwaita schemes. A card with that radius looks square,
  and none of those desktops draws a square floating panel. `glance.CardRadius`
  is 8, between the 8 that the monitor uses for a section and the 12 it uses
  for the window around them.
- **The panel behind the cards is square.** A round corner on the panel would
  make a round corner on the window. The window cannot be translucent
  (quirk 32), so the corner would be cut out of an opaque rectangle and would
  not show the desktop.
- **No rule may depend on a view through the window.** Do not position, size
  or colour anything on the assumption that the wallpaper is legible behind
  it. The design is a glance window at 100 % opacity. A lower value is the
  preference of the user, applied above the design.

## The card stack

A card is the unit of a glance window, and it is closer to a section than to a
widget. It is complete in itself, it owns its own timer and its own drawing,
and it declares a small public API instead of exposing its labels.

Each of the three cards in the monitor declares four to six methods. Keep this
shape:

- **Construct** the card with the settings it needs, and with a callback for
  the settings it changes. The card then never learns where the file is.
- **`SetVisible(bool)`** is the switch that the user operates. It is *not* the
  same as availability. Draw a card when the user permits it **and** it has
  something to report.
- **Draw a snapshot** with one method that takes a plain value read somewhere
  else. All parsing is outside the card, and the card holds the interface and
  the state only. The monitor keeps this division so completely that its
  readers run as command-line programs for debugging, such as
  `python3 aio_reader.py --json`. Keep that: a data layer you can run without a
  display is a data layer you can test without one.
- **Apply the style again** when the scheme or the text size changes.

**A card that is hidden costs nothing.** Its timer stops. A glance window on a
machine without the sensor that a card watches must look the same as a window
built without that card.

Each card takes the width of the widest card, and the cards stack from top to
bottom in an order that the program sets. The monitor makes each section take
the full width for a reason it records: without that, the narrow battery card
left a space beside the wider bandwidth card. In a translucent window that
space was a hole in the panel. In an opaque window it is an uneven right edge,
which is the same fault and less visible.

## Sizing

The window measures its content and takes that size. A fixed-size window in
Fyne grows by itself and becomes smaller only when you ask (quirk 34). After
you hide a card you must therefore call `Resize(content.MinSize())`. This is
the `adjustSize()` that the monitor calls after each change of visibility,
each change of font scale, and the first data of each card.

- **Resize when the cause is complete, and not at the next frame.** The
  sections of the monitor call `updateGeometry()`, then call `adjustSize()` on
  the top-level widget in the same call. They do not wait for a paint. A window
  that resizes one frame late resizes where the reader can see it.
- **Set a minimum width one time.** The monitor uses 260, which "keeps the two
  slot cells wide enough to avoid cutoff names". Without a minimum, a window
  with one card and one short value is very small.
- **Nothing temporary resizes the window.** The rule in design-system.md is
  stricter here, because the thing that moves is the whole panel and not one
  row. A value that arrives must not change the size of the window, which is
  what [Numbers that do not jitter](#numbers-that-do-not-jitter) prevents. Only
  a *card* that appears or disappears may resize the window, and that is a real
  event and not a repaint.
- Padding and spacing follow the text size; do not keep them at a fixed number
  of pixels. At a scale of 0.8, a margin set for a scale of 1.0 is too large in
  proportion. The monitor multiplies each margin by the scale, with a small
  minimum so that nothing becomes zero. design-system.md makes the same point
  about the `48` and `18` of the applications, which were set for size 12
  only.

## Numbers that do not jitter

This is where the module applies the rule against movement, and it applies it
at the glyph. `glance.Rate`, `Size`, `Percent`, `Quantity` and `Count` do this.
Each one has an empty form of the same width, such as `NoRate` and
`NoPercent`, for a reading that has not arrived.

- **Use monospace for every value that changes.** design-system.md forbids the
  use of a proportional family in place of a monospace face for this reason: a
  place that asks for monospace asks because the alignment is important.
- **Fix the column width for the widest value that can occur, not the value
  you have now.** The monitor draws bandwidth as a number of 5 characters
  aligned right, and a unit aligned left and padded to the longest unit and
  suffix. `999.9 KiB/s` therefore becomes `1.0 MiB/s` and nothing moves. Below
  1 KiB the number is an integer, and above it the number has one decimal.
- **Choose the unit from the size of the value, then pad the result.** Do not
  let the unit set the width. `B`, `KiB`, `MiB`, `GiB` and `TiB` have different
  widths, and the column does not.
- Do the same for temperatures, percentages and durations. Write `53.0 °C` and
  never `53 °C`. A value that sometimes removes its decimal is a value that
  sometimes moves the label beside it.
- Put labels on the left and values on the right, and let the space between
  them take the change in width. See also design-system.md, "An `HBox` hands a
  truncating label its minimum size" (quirk 17), and fix the width.

## Meters

A **meter** is the one shape here that is a bar. It shows a proportion of
something that has a limit: a quota that is used, a budget that is spent, or a
period of time that has passed. `glance.Meter` draws a short label, a caption
with the detail, and a bar whose colour follows how near the value is to the
limit.

- **A reading with no limit is a row and not a meter.** A temperature, a rate
  and a fan speed have nothing to fill, and a bar for one of them invents a
  maximum. This is also why the busy indicator of the design system has no
  value; see design-system.md, Progress and results. A bar that filled at a
  constant rate would invent a number. The number of a quota is real, and that
  is what permits a bar.
- **A quota is one of the few things that has a true threshold**, so its
  colour is a signal and not decoration. A CPU temperature is different,
  because it is high in normal use; see [Colour is the
  legend](#colour-is-the-legend).
- **The caption is not decoration.** A bar reports "most of it". The caption
  reports which period, how much of it, and when it starts again, and that is
  what a person who looks at the panel needs. A meter with no caption is a bar
  that nobody can act on.
- **The caption is a value that changes, and it obeys the width rule.** It sets
  the minimum width of the meter. A caption made with
  `fmt.Sprintf("%.0f%%", pct)` grows from "5%" to "100%", and it takes the
  width of the window with it. Format it with `Percent`, `Quantity` or `Pad`.
- **Padding is not sufficient, because the caption draws in the monospace
  face.** In a proportional face a space is narrower than a digit, so a value
  padded to a fixed number of characters still changes width. This caused a
  test failure after the padding was already correct.
- The bar keeps height and no width, so a meter never decides the width of the
  window. **The caption does.** It is the longest object in the card, so the
  panel takes its width. Keep it to the shape that the monitor uses,
  `4 % · resets in 4h 32m`, and do not write a sentence. A caption that
  explains itself in prose makes every other card in the window wider.

## Colour is the legend

- `Status` still names the levels, and colour still comes from the scheme.
- **Give a colour only to a value that has a threshold.** The monitor colours
  the coolant temperature: green below 50 °C, amber to 55 °C, and red above
  that. It does *not* colour the CPU temperature, because a high boost
  temperature is normal, and a colour would make every compilation red. A
  colour that is always present is not a signal.
- Take the thresholds from the hardware and do not use round numbers. The bands
  of the monitor come from one cooler, whose over-temperature alarm operated at
  57.1 °C and cleared near 50 °C. Record beside each threshold where it came
  from.
- **Where a value appears two times, use one colour for both.** The monitor
  paints each sparkline trace and the value in its row in the same colour, and
  supplies no legend. A legend in a window of 260 px is a second copy of the
  row labels.

## Trend plots

A sparkline belongs in a glance window, and it does not belong in an
application window. It is the one thing that a reader takes from a panel they
never click. `glance.Sparkline` draws it.

- **Hold a fixed number of samples and put the newest at the right edge.** The
  monitor holds 60 samples at 5 s, which is a period of 5 minutes.
- **Plot what is legible at 26 px, and not what you measured.** A CPU
  temperature goes to 100 °C during any compilation, and at this height that is
  noise. The monitor plots a mean of the last 60 seconds and says so in its
  text. The useful content is the delay between the traces, which a mean keeps
  and noise hides.
- **Scale each trace to its own range, and tell the reader that the heights do
  not compare.** In one recorded session the CPU moved between 65 °C and 98 °C
  while the coolant moved between 45.8 °C and 46.6 °C, a ratio of about 40 to
  1. On one axis the coolant trace would be 2 px tall. Read the shape from the
  plot and the values from the rows.
- **Give each trace a minimum range**, which is 5 °C in the monitor. An idle
  line then stays flat, and does not make sensor noise into large movements.
- **A gap is a gap.** A poll that read nothing records nothing, and the plot
  does not fill the space. The history is in memory and is empty after a
  restart, which is not a state worth keeping on disk.

## The context menu is the whole interface

A glance window has no space for controls and no reason to show them. The user
changes each setting seldom and reads it never.

- A secondary tap on the window opens one menu. That menu holds everything the
  program configures: the text size, which card is shown, and what each card
  watches.
- **Put the items in submenus and mark the current value.** The reader examines
  the current setting as often as they change it.
- **A submenu that cannot operate is absent, not disabled.** The monitor
  removes its effect menu when the device has not reported its list of effects,
  because nobody can guess the names, and a grey menu of guesses is worse than
  no menu. This is the one place where the rule "disabled, never hidden" from
  design-system.md does not apply. That rule protects a toolbar whose buttons
  must not move below the pointer, and a menu that is not open has no pointer
  above it.
- Anything that opens over the window calls `widgets.HideTips` first
  (quirk 27), as everywhere else.

## Degradation

Each card decides what it shows when its source fails, and none of them shows
an error dialog. A glance window never opens a dialog, because it is not the
window that the person was looking at when the failure occurred.

There are five states, and the monitor draws each one differently:

| State | Rendering |
|---|---|
| Source absent (never present) | The card is hidden. The window is smaller. Nothing says so |
| Source present, one metric missing | That row is hidden. The card stays for the rows that have data |
| Source was present and has gone | The card stays, **last values dimmed**, a short marker in its header. Recovery clears the marker |
| Source present, value genuinely unknown | The value renders as a placeholder (`--%`) in the dim colour, which is not the same as a zero |
| Nothing at all available | The card is hidden and the window is unchanged from a build without it |

- **An absent reading is never an alarm.** The monitor says this directly: no
  evidence is not a stopped pump. A drawing that cannot separate "0 RPM" from
  "no reading" will report a failure that did not occur, or hide one that did.
  The note for version 1.17.0 records a power-supply probe accepted as a
  coolant sensor, beside a pump reading of `0` that looked the same as a dead
  pump.
- **Continue to poll after a source goes, but poll slowly.** A daemon that
  starts later must bring its card back without a restart of the program. The
  monitor changes to a probe every 30 s, and returns to 5 s when the source
  answers.
- **A dim value means that the value is old.** Do not use a spinner, a line
  through the text, or an empty space. The question of the reader is whether
  the value is still true, and a dim number answers that and moves nothing.

## Alerting is not the window's job

**A glance window must not be the only place where a failure is visible.** The
cooling alert of the monitor exists because a pump failed with no message. The
CPU dropped to 0.20 GHz at 100 °C while the widget showed a reasonable number,
and nobody was looking at the widget. A notification must not depend on a
person who looks.

- Send a desktop notification through the mechanism of the platform, outside
  the window.
- **Wait before you send.** A state must continue for several polls in
  sequence, which is three polls and approximately 15 s in the monitor. One
  incomplete read then cannot send a notification.
- **Send again slowly, and replace the previous notification instead of adding
  to it.** Send at most one notification every 10 minutes while the condition
  continues.
- **Send one notification when the condition clears.** A condition that cleared
  and did not say so leaves the reader with the last thing you told them.

## Cadence

- Each card owns its timer and its interval. There is no shared tick: a
  bandwidth counter at 2 s and a temperature probe at 5 s share only a window.
- **Do not start a poll while the previous poll is still running.** A slow
  source must not make a queue of its own polls.
- **Give every read outside the process a deadline.** The monitor gives HTTP
  2 s and a subprocess 5 s, because a source that stops would otherwise stop
  the panel. Two programs that used one hidraw node caused stops of several
  seconds.
- The threading rules in design-system.md apply without change. Do the work on
  a goroutine, return to the UI thread with `fyne.Do`, never use
  `fyne.DoAndWait`, and run inline when the window is not on screen.
- Stop the timer of a card that is hidden. Do not let it run and discard the
  result.

## Copy

The voice in design-system.md applies. The small window adds two rules:

- **A label is a noun and not a sentence.** Write `Coolant`, `Fans` and
  `Discharging`. The window has space for the value or for prose, and not for
  both.
- **Write the state and not the mechanism.** Write `Disconnected` and not
  `no BlueZ battery property`. The mechanism belongs in the log.
- A unit is part of the value, and you always show it. A `53` with no unit, in
  a panel that carries temperatures and revolutions for each minute, is a
  number that the reader must identify.

## Testing

Everything in the Testing section of design-system.md applies. There are three
additions:

- **Assert the formatter and not the panel.** A fixed-width numeric format is a
  pure function, and the rule against movement is a property of that function:
  the width of the value is the same for every size of number it can take. That
  is the test, and it does not need a canvas.
- **Write two tests: one for availability and one for visibility.** A card that
  the user hid and a card with no data draw the same way and arrive by
  different paths. One test for both will pass while one of the two paths is
  defective.
- **Test that the window becomes smaller.** Hide a card and assert that the
  window is smaller. That is the test for quirk 34, and it is the failure that
  nobody sees in use: a window that grew and never became smaller looks like a
  fault in the layout and not an absent `Resize`.

## Numbers

These come from `peripheral-battery-monitor` as values to start from. They are
not the values of a Fyne layout that somebody has adjusted. A fixed number of
pixels assumes the base text size. Calculate a height that comes from the
minimum size of a card from the text size of the theme, and do not copy the
number. See design-system.md, Numbers.

| Thing | Value |
|---|---|
| Minimum window width | 260 |
| Text scale steps | 0.8, 1.0, 1.3 (small, medium, large) |
| Card outer margin, horizontal / vertical | 15 / 12, scaled, floors 6 / 5 |
| Card inner spacing | 15, scaled, floor 5 |
| Value / label / status text size | 22 / 11 / 10, times the scale |
| Icon | 24, scaled, floor 14 |
| Sparkline samples / cadence / span | 60 / 5 s / 5 minutes |
| Meter bar height / radius | 8 / 4 |
| Sparkline height | 26 |
| Sparkline minimum span per trace | 5 units |
| Trend smoothing | 60 s trailing mean |
| Fast poll / idle poll | 2 s (counters), 5 s (sensors) / 30 s |
| HTTP read deadline / subprocess deadline | 2 s / 5 s |
| Alert confirm / alert repeat | 3 consecutive polls (~15 s) / 10 minutes |
| Bandwidth numeric column | 5 characters, right aligned, monospace |
| Indicator hold after the last change | 1.5 s (`glance.DefaultHold`) |
