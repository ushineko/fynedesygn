# Glance windows

A glance window is a small, frameless, always-on-top panel that sits on the
desktop and is read without being interacted with. It has no header, no
navigation, no status bar and no scroller. The window is the content: a stack
of cards, each one a metric that is either worth a glance or not there at all.

[design-system.md](design-system.md) is the rulebook for application windows —
the shell with its header, nav, content scroller and status bar. This document
is the rulebook for the other shape. The two share `Status`, the palette, the
fonts, the threading rules and the copy voice; they disagree about almost
everything structural, and the disagreements are listed below rather than left
to be discovered.

The archetype is transcribed from `ag-scripts/peripheral-battery-monitor`
(PyQt6, same author), which has carried it through 1.x for peripheral battery,
network bandwidth, liquid-cooler thermals and Claude Code usage. Where a rule
below has a reason attached, the reason is that program's, earned in use.

Package `glance` implements this: the window, the card stack, the rows, the
fixed-width formatters and the sparkline. Package `glance/kwin` installs the
KDE Plasma window rule the desktop half needs. `examples/glance-monitor` is a
working panel in this shape, and its tests are where most of the rules below
are pinned.

The package is deliberately smaller than the archetype. What is written down
here and not built is named as such at the point it comes up.

![A small frameless panel on a desktop, four cards stacked, each a rounded surface a step lighter than the panel behind it. "Peripherals" shows a mouse at 87% in green. "Bandwidth" shows two interface rows with rates right-aligned in a monospace column. "AIO" shows CPU 74.5 °C in blue, Coolant 50.1 °C in amber and Fans 1368 rpm, over a sparkline whose two traces are drawn in the colours of the rows they came from. "Usage" shows two meters, each a label, a monospace caption and a bar: a green one at 48% of a five-hour window, an amber one at 82% of a monthly budget. No titlebar, no buttons, no scrollbar.](img/glance-monitor.png)

`examples/glance-monitor`, with the KDE window rule installed. Without the rule
KWin draws a titlebar on it, which is what `glance/kwin` exists to remove: a
splash window asks GLFW not to be decorated and the compositor decorates it
anyway.

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

Everything not listed inverts nothing. `Status` still names the four levels,
colour still comes from the active scheme, `fyne.Do` is still the only way onto
the UI thread, and copy is still sentence case with consequences named.

## What Fyne 2.8.1 gives and withholds

Verified against `fyne.io/fyne/v2 v2.8.1`, the module's pin.

| Need | Available | Where |
|---|---|---|
| A frameless window | Yes: `desktop.Driver.CreateSplashWindow()` — undecorated, `SetPadded(false)`, centred on screen | `internal/driver/glfw/window.go` |
| Always on top | Yes: `desktop.Window.RequestAlwaysOnTop()`, **before `Show`**; sets the GLFW `Floating` hint | `driver/desktop/window.go` |
| Position the window | Partly: `desktop.Window.RequestPosition(x, y)`. Its own doc says the request "may be ignored (for example Linux Wayland)" | `driver/desktop/window.go` |
| Read the window's position back | **No.** GLFW's position callback is wired internally; nothing on `fyne.Window` exposes a position | quirk 33 |
| A translucent window background | **No.** The desktop backend never requests `glfw.TransparentFramebuffer`, and asks for no alpha bits. Only the wasm backend requests `AlphaBits, 8` | quirk 32 |
| Size to content | Yes, in one direction only: a `SetFixedSize(true)` window grows to its content's minimum size on its own, and shrinks only when the program calls `Resize` explicitly | quirk 34 |
| Know the window was resized | No — quirk 14, already recorded. It does not bite here: a glance window is fixed-size and the program is the only thing that resizes it |

Two of these are load-bearing, and neither has a workaround inside the
process. They are why this document exists before any code does.

## The window

- One splash window, `SetFixedSize(true)`, `RequestAlwaysOnTop()` called before
  `Show`. `SetMaster()` as usual: closing it ends the program.
- **Dragging is the desktop's job, not the window's.** Qt exposes
  `startSystemMove()` and the monitor uses it; Fyne exposes nothing
  equivalent, and a drag implemented by calling `RequestPosition` from a mouse
  handler is a request the compositor may ignore on every frame. On Wayland
  the window is placed by the compositor and a window rule is the supported
  way to pin it. Document the rule; do not fight for the pixel.
- **Position persistence is out of reach in-process.** Nothing can read the
  position back (quirk 33), so there is nothing to save. The monitor reached
  the same conclusion from the other side — `moveEvent` does not fire for
  compositor-driven moves on Wayland — and drives KWin's scripting D-Bus API
  from outside Qt entirely. A program that wants its glance window in one
  place ships a compositor rule, the same way it ships a desktop entry.
- The desktop file basename must equal the app ID, as for any window, or
  Wayland cannot match the window to its icon (design-system.md, Platform).
- No menu bar, no tabs, no `CenterOnScreen` after the first run.

## Desktop integration

Three things a glance window needs — staying on top, losing its border, and
being translucent — are the compositor's to grant, and Fyne reaches only the
first two. The rest is desktop-specific, so it lives behind a build tag like
every other platform difference in this module (design-system.md, Platform).

The precedent is already here: `theme.ApplyCursorTheme` reads Plasma's
`kcminputrc` directly rather than shelling out to `kreadconfig`, on the
grounds that a cursor theme is not a good enough reason for a library to start
a subprocess. Desktop integration follows the same line — read and write the
desktop's own configuration, and use its D-Bus interface where one exists.

**The library never writes the user's compositor configuration on its own.** It
is an explicit installation step the program offers, the same way it installs a
desktop entry, and it is reversible. A glance window that silently edited
`kwinrulesrc` on first run would be a program that changed the desktop without
being asked.

### KDE Plasma

Implemented in package `glance/kwin`, transcribed from the monitor's
`install_kwin_rule.py` and `kwin_window_position.py`, which are the tested
versions of all of this. The rule file is implemented; the Scripting D-Bus API
for position is not, and is described below for the spec that adds it.

**Matching.** KWin matches by window class. On Wayland that is the `app_id`,
which Fyne sets from `fyne.CurrentApp().UniqueID()` — the driver passes it as
GLFW's `WaylandAppID` hint. So the app's unique ID, the desktop file basename
and the rule's `wmclass` are the same string, and the icon rule from
design-system.md is the same rule as this one.

**The rule file** is `~/.config/kwinrulesrc`: INI-shaped, **keys are case
sensitive**, with a `[General]` section holding `rules=` (a comma-separated
list of section names) and `count=`. A rule is a numbered section. Installing
means finding the section whose `wmclass` is the app's, or appending a new
one — never rewriting the file, which holds every other rule the user has.

| Key | Value | Effect |
|---|---|---|
| `wmclass` / `wmclassmatch` | the app ID / `1` | exact match |
| `above` / `aboverule` | `true` / `2` | keep above, forced |
| `noborder` / `noborderrule` | `true` / `2` | no titlebar, forced |
| `opacityactive` / `opacityactiverule` | 0–100 / `4` | **this is the translucency**, applied initially |
| `opacityinactive` / `opacityinactiverule` | 0–100 / `4` | same when unfocused |

`2` is "Force" and `4` is "Apply Initially" in KWin's rule vocabulary. Opacity
is applied initially rather than forced so the user can still override it from
the window menu.

**`noborder` is required, not belt-and-braces.** A splash window sets GLFW's
`Decorated` hint to false and KWin draws a titlebar on it regardless; the
screenshot at the top of this document needed the rule installed to be
frameless at all. `above` is closer to a duplicate of `RequestAlwaysOnTop` and
is still worth setting: the rule survives a compositor restart, and
`RequestAlwaysOnTop`'s own doc warns the window manager may decide other
windows stay above it.

**Position is deliberately not a rule.** KWin's `position` rule selects a
screen and snaps to its origin on Wayland; it does not honour intra-screen
coordinates. A rule that half-works is worse than none, and the monitor's
installer explicitly deletes stale `position`, `size` and `screen` keys a
previous version wrote.

**Position is the Scripting D-Bus API.** `org.kde.KWin` at `/Scripting` loads a
small KWin JS script, runs it, and unloads it; the script finds the window by
`resourceClass` and sets `frameGeometry` to restore, or reads `frameGeometry`
and calls back over D-Bus to report. A loaded KWin script runs once, so each
operation is a fresh load/run/unload cycle. This is the piece that answers
quirk 33: the position Fyne cannot read is one the compositor will hand over if
asked in its own language.

**Applying a rule change** is `reconfigure` on `org.kde.KWin` at `/KWin`. The
monitor tries `qdbus6`, `qdbus` and `dbus-send` in turn and warns rather than
failing when none works; a Go implementation would make the call directly.
`github.com/godbus/dbus/v5` is already in the module graph as a Fyne
dependency, so the session bus is reachable without adding one — but promoting
it to a direct dependency of a library package is a decision for the spec that
does it, under the standing rule that Fyne is the only required runtime
dependency of the core packages.

**Everything here is optional at runtime.** A glance window on a machine with
no KWin is a window that is frameless, on top, opaque and wherever the
compositor put it. That is a working program, not a degraded one.

### Other desktops

Not yet written, and not guessed at here. Each needs the same three answers —
how to stay above, how to lose the border, how to be translucent — from
whatever the desktop actually offers:

- **GNOME / Mutter** has no user-facing window-rule equivalent; the honest
  answer may be that only the Fyne-native part is available.
- **Windows** has `SetWindowPos` with `HWND_TOPMOST` and layered windows with
  `SetLayeredWindowAttributes` for whole-window alpha, both per-HWND — but Fyne
  exposes no window handle, so reaching them is its own problem.
- **macOS** has `NSWindow.level` and `alphaValue`, with the same problem.

Until each is written and tested on the desktop in question, a glance window
there is the Fyne-native subset.

## Opacity without alpha

The monitor sets `WA_TranslucentBackground` and offers 100/95/90/80/70 %; the
desktop shows through the panel. **Fyne cannot draw that** (quirk 32), and a
port that waits for it waits indefinitely. What it can do is ask the
compositor, which applies opacity to a window whose toolkit knows nothing about
it — on Plasma that is two lines in a KWin rule, and the monitor already ships
them. See [Desktop integration](#desktop-integration).

So there are two different things, and the design depends on only the first:

- **Whole-window opacity is a desktop setting**, available where the desktop
  offers it, absent where it does not. Offer it, honour it by writing the
  desktop's own rule, and treat its absence as normal.
- **Per-pixel alpha is not available at all.** A card cannot be more opaque
  than its window, a value cannot fade in, and a background cannot be tinted
  over the wallpaper.

Which yields the binding rule:

- **Separation comes from contrast, not from alpha.** The monitor's cards are
  `rgba(43,43,43,α)` on the desktop with a `rgba(255,255,255,20)` hairline.
  Opaque, that is a card surface one step from the window's own with a hairline
  border, which the palette already carries: the window is `WindowBG`, the card
  is `ButtonBG` and the border is `Separator`, reached through Fyne's
  `Background`, `Button` and `Separator` roles. `ButtonBG` rather than
  `ViewAltBG`, which is the alternating-row token and sits *darker* than the
  window in the dark schemes — a card drawn in it disappears into the panel.
  `Card` draws this.
- **A card takes a surface radius, not the scheme's control radius.** The
  palette's radius token is for entries and buttons and is 2 in the Breeze,
  Oxygen and Adwaita schemes; a card drawn with it reads as a square, which is
  not what any of those desktops draws for a floating panel. `glance.CardRadius`
  is 8, between the monitor's 8 for a section and 12 for the window around them.
- **The panel behind the cards is square.** Rounding it would round the window,
  and the window cannot be translucent (quirk 32), so its corners would be cut
  out of an opaque rectangle rather than showing the desktop.
- **No rule may depend on seeing through the window.** Nothing is positioned,
  sized or coloured on the assumption that the wallpaper is legible behind it.
  A glance window at 100 % opacity is the design; anything less is the user's
  preference applied on top of it.

## The card stack

A card is the glance window's unit, and it is closer to a section than to a
widget: self-contained, owning its own poll timer and its own rendering, and
declaring a small public API rather than exposing its labels.

The monitor's three cards each declare four to six methods. The shape that
recurs is worth keeping:

- **construct** with the settings it needs and a callback for settings it
  changes, so the card never learns where the file is;
- **`SetVisible(bool)`** — the user's toggle, which is *not* the same as
  availability. A card is drawn when the user allows it **and** it has
  something to say;
- **render a snapshot** — one method taking a plain value read elsewhere. All
  parsing lives outside the card; the card is UI and state only. The monitor
  keeps this line hard enough that its readers run as CLIs for debugging
  (`python3 aio_reader.py --json`), which a port should keep: a data layer you
  can run without a display is a data layer you can test without one.
- **restyle** when the scheme or text size changes.

**A hidden card costs nothing.** Its timer stops. A glance window on a machine
without the sensor it watches should be indistinguishable from one built
without the card.

Cards expand to the width of the widest card and stack top to bottom in an
order the program fixes. The monitor makes each section expand horizontally
for a stated reason: without it the narrow battery card left a gap beside the
wider bandwidth card, which in a translucent window was a hole in the panel.
Opaque, it is a ragged right edge, which is the same defect with less drama.

## Sizing

The window measures its content and takes that size. Fyne's fixed-size window
grows on its own and shrinks only when asked (quirk 34), so a card being
hidden must be followed by an explicit `Resize(content.MinSize())` — the
`adjustSize()` the monitor calls after every visibility change, every font
scale change and every card's first data.

- **Resize once the cause has finished, not on the next frame.** The monitor's
  sections call `updateGeometry()` and then reach up to the top-level widget
  and `adjustSize()` in the same call, rather than waiting for a paint. A
  window that resizes a frame late resizes visibly.
- **A minimum width, set once.** 260 in the monitor, "keeps the two slot cells
  wide enough to avoid cutoff names". Without a floor, a window with one card
  showing one short value is a chip.
- **Nothing transient resizes the window.** The binding rule from
  design-system.md is stricter here because the unit that moves is the whole
  panel, not a row. A value arriving must not change the window's size — which
  is what [Numbers that do not jitter](#numbers-that-do-not-jitter) is for. Only
  a *card* appearing or disappearing may resize, and that is a real event, not
  a repaint.
- Padding and spacing scale with the text size rather than staying fixed
  pixels. At 0.8x, margins tuned for 1.0x are proportionally enormous; the
  monitor multiplies every margin by the scale with a small floor so nothing
  collapses to zero. design-system.md makes the same point about the apps' `48`
  and `18` having been tuned to size 12 only.

## Numbers that do not jitter

This is where the no-reflow rule is enforced, and it is enforced at the glyph.
`glance.Rate`, `Size`, `Percent`, `Quantity` and `Count` do it, each with a
matching blank of the same width (`NoRate`, `NoPercent`, ...) for a reading
that has not arrived.

- **Monospace for any value that changes.** design-system.md already forbids
  substituting a proportional family's non-existent mono face for exactly this
  reason: the places that asked for monospace asked because alignment
  mattered.
- **Fixed column widths, chosen for the widest plausible value, not the
  current one.** The monitor formats bandwidth as a 5-character right-aligned
  number and a left-aligned unit padded to the longest unit plus the suffix,
  so `999.9 KiB/s` becoming `1.0 MiB/s` moves nothing. Integers below 1 KiB,
  one decimal above.
- **Pick the unit from the magnitude and pad the result** rather than letting
  the unit string set the width. `B`, `KiB`, `MiB`, `GiB`, `TiB` differ in
  width; the column does not.
- The same applies to temperatures (`53.0 °C`, never `53 °C`), percentages and
  durations. A value that sometimes drops its decimal is a value that
  sometimes moves the label beside it.
- Labels are left, values are right, and the gap between them is where the
  width goes. See also design-system.md, "An `HBox` hands a truncating label
  its minimum size" (quirk 17): pin the width.

## Meters

A **meter** is the one shape here that is a bar: a proportion of something with
a limit. A quota used, a budget spent, a window of time elapsed. `glance.Meter`
draws a short label, a caption carrying the detail, and a bar graded by how
close to the limit it is.

- **A reading with no limit is a row, not a meter.** A temperature, a rate and
  a fan speed have nothing to fill up, and drawing one as a bar invents a
  maximum. This is also why the design system's busy indicator is indeterminate
  (design-system.md, Progress and results): a bar that filled steadily would be
  inventing a number. A quota's number is real, which is what earns it a bar.
- **A quota is one of the things that genuinely has a threshold**, so grading
  it is a signal rather than decoration — unlike a CPU temperature, which is
  high in normal use (see [Colour is the legend](#colour-is-the-legend)).
- **The caption is not decoration.** A bar says "most of it". The caption says
  which window, how much of it, and when it resets, and that is what someone
  glancing at the panel wants. A meter without one is a bar nobody can act on.
- **The caption is a changing value and obeys the width rule.** It sets the
  meter's minimum width, so a caption built with `fmt.Sprintf("%.0f%%", pct)`
  grows from "5%" to "100%" and takes the window's width with it. Format it
  through `Percent`, `Quantity` or `Pad`.
- **Padding alone is not enough: the caption is drawn in the monospace face.**
  A space is narrower than a digit in a proportional face, so a value padded to
  a fixed character count still changes width. This one cost a test failure
  after the padding was already right.
- The bar reserves height and no width, so a meter never decides how wide the
  window is — **but the caption does.** It is the longest thing in the card, so
  it is what the panel is sized to. Keep it to the shape the monitor uses,
  `4 % · resets in 4h 32m`, not a sentence: a caption that explains itself in
  prose makes every other card in the window wider to match.

## Colour is the legend

- `Status` still names the levels, and colour still comes from the scheme.
- **Colour-grade only what has a threshold.** The monitor grades coolant
  temperature — green below 50 °C, amber to 55, red above — and deliberately
  does *not* grade CPU temperature, because a high boost temperature is normal
  and grading it would paint every compile red. A colour that is always on is
  not a signal.
- Thresholds are the hardware's, not round numbers: the monitor's bands come
  from a specific cooler whose over-temperature alarm tripped at 57.1 °C and
  cleared near 50. Record where a threshold came from beside it.
- **Where a value appears twice, one colour serves both.** The monitor paints
  each sparkline trace and its row's value in the same colour and ships no
  legend. A legend in a 260 px window is a second copy of the row labels.

## Trend plots

A sparkline belongs in a glance window in a way it does not belong in an
application window: it is the one thing a reader takes from a panel they do not
click. `glance.Sparkline` is the implementation.

- **Fixed capacity, right-anchored, so "now" is the right edge.** The monitor
  holds 60 samples at 5 s, a 5-minute window.
- **Plot what is readable at 26 px, not what was measured.** Raw CPU
  temperature spikes to 100 °C on any compile and is noise at this height; the
  monitor plots a 60-second trailing mean and says so. The interesting content
  is the lag between the traces, which averaging preserves and noise hides.
- **Scale each trace to its own range, and say the heights are not
  comparable.** In one recorded session CPU ranged 65–98 °C while coolant moved
  45.8–46.6 °C, roughly 40:1; on a shared axis the coolant trace would have
  been 2 px tall. Read shape from the plot and values from the rows.
- **A minimum span per trace**, 5 °C in the monitor, so an idle flat line stays
  flat instead of amplifying sensor jitter into a mountain range.
- **Gaps are gaps.** A poll that read nothing records nothing and is not
  interpolated. History is in memory and starts empty after a restart; that is
  not a state worth persisting.

## The context menu is the whole interface

A glance window has no room for controls and no reason to show them: every
setting is changed rarely and read never.

- Secondary tap on the window opens one menu carrying everything the program
  configures — text size, which card is shown, what each card watches.
- **Grouped in submenus, current value ticked.** The reader is checking what it
  is set to at least as often as changing it.
- **A submenu that cannot act is absent, not disabled.** The monitor omits its
  effect menu entirely when the device has not reported its effect list, on the
  grounds that the names cannot be guessed and a greyed menu of guesses is
  worse than no menu. This is the one place the "disabled, never hidden" rule
  from design-system.md does not carry: that rule protects a toolbar whose
  buttons must not move under the pointer, and a menu that is not open has no
  pointer over it.
- Anything that opens over the window calls `widgets.HideTips` first
  (quirk 27), as everywhere else.

## Degradation

Each card owns its own degradation story and none of them is an error dialog.
A glance window never raises a dialog: it is not the window you were looking
at when it happened.

Five states, all of which the monitor distinguishes:

| State | Rendering |
|---|---|
| Source absent (never present) | The card is hidden. The window is smaller. Nothing says so |
| Source present, one metric missing | That row is hidden. The card stays for the rows that have data |
| Source was present and has gone | The card stays, **last values dimmed**, a short marker in its header. Recovery clears the marker |
| Source present, value genuinely unknown | The value renders as a placeholder (`--%`) in the dim colour, which is not the same as a zero |
| Nothing at all available | The card is hidden and the window is unchanged from a build without it |

- **A missing reading is never an alarm.** The monitor states it directly:
  absence of evidence is not a stopped pump. A rendering that cannot tell
  "0 RPM" from "no reading" is a rendering that will eventually report a
  failure that did not happen, or mask one that did — its 1.17.0 note records
  a PSU probe being accepted as a coolant sensor beside a pump reading of `0`,
  which was indistinguishable from a dead pump.
- **Keep polling after a source disappears, slowly.** A daemon started later
  should bring its card back without a restart. The monitor drops to a 30 s
  probe and returns to 5 s when the source answers.
- **Dimming is the vocabulary for stale.** Not a spinner, not a strikethrough,
  not a removed value. The reader's question is "is this still true", and a
  dim number answers it without moving anything.

## Alerting is not the window's job

**A glance window must not be the only place a failure is visible.** The
monitor's cooling alert exists because a pump died silently: the CPU throttled
to 0.20 GHz at 100 °C while the widget displayed a plausible number, and nobody
was looking at the widget. Notification deliberately does not depend on anyone
looking.

- Raise a desktop notification through the platform's own mechanism, outside
  the window.
- **Debounce.** A state must persist for several consecutive polls — three,
  about 15 s, in the monitor — so one partial read cannot fire one.
- **Re-notify slowly and replace rather than stack**: at most every 10 minutes
  while the condition holds.
- **Send one recovery notification.** A condition that cleared and never said
  so leaves the reader believing the last thing they were told.

## Cadence

- Every card owns its timer and its interval. There is no shared tick: a
  bandwidth counter at 2 s and a thermal probe at 5 s share nothing but a
  window.
- **A poll is skipped entirely while the previous one is outstanding.** A slow
  source must not queue behind itself.
- **Every out-of-process read has a deadline.** The monitor gives HTTP 2 s and
  a subprocess 5 s, both because a hung source would otherwise stall the panel;
  contention on a hidraw node produced multi-second stalls in practice.
- The threading rules from design-system.md apply unchanged: work on a
  goroutine, back to the UI thread with `fyne.Do`, never `fyne.DoAndWait`,
  inline when the window is not on screen.
- A hidden card's timer is stopped, not ignored.

## Copy

design-system.md's voice applies, with two additions the small window forces:

- **A label is a noun, not a sentence.** `Coolant`, `Fans`, `Discharging`. The
  window has room for the value or for prose, not both.
- **Say the state, not the mechanism.** `Disconnected`, not `no BlueZ battery
  property`. The mechanism belongs in the log.
- Units are part of the value and are always shown. A bare `53` in a panel
  carrying both temperatures and RPM is a number the reader has to place.

## Testing

Everything in design-system.md's Testing section holds. Three additions:

- **Assert the formatter, not the panel.** Fixed-width numeric formatting is a
  pure function and the jitter rule is a property of it: the width of the
  rendered value is the same across every magnitude it can take. That is the
  test, and it does not need a canvas.
- **Availability and visibility are two tests.** A card the user has hidden and
  a card with no data render the same and arrive by different paths; a single
  test covering both will pass while one of them is broken.
- **Test the shrink.** Hiding a card and asserting the window got smaller is
  the test for quirk 34, and it is the one that fails silently in use — a
  window that grew and never shrank looks like a layout bug, not a missing
  `Resize`.

## Numbers

Carried from `peripheral-battery-monitor` as starting values, not as
transcriptions of a tuned Fyne layout. Fixed pixel values assume the base text
size, and heights derived from a card's minimum size should be computed from
the theme's text size rather than copied (design-system.md, Numbers).

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
