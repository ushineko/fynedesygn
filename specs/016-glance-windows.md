# Spec 016: Glance windows

## Status: COMPLETE

- **Issue**: #7
- **Priority**: Medium
- **Estimated Complexity**: Medium
- **Branch**: `feat/glance-windows`
- **Transcribed from**: `ag-scripts/peripheral-battery-monitor` v1.16.0 (PyQt6,
  same author), which has carried this archetype since its 1.0

> **Note**: this spec was written after the work, not before it. The issue and
> the link were added when the gap was noticed. The convention is that a new
> spec starts from an issue; this one did not, and the record says so rather
> than pretending otherwise.

## Context

The module describes one kind of window. `shell` builds it and
`docs/design-system.md` is its rulebook: a header, a section navigation, one
content scroller, a status bar, sections that rebuild from state.

There is a second kind. A glance window is small, frameless, always on top, and
read without being interacted with — the peripheral and sensor panel that sits
in the corner of the screen. It has no header, no nav, no scroller and almost
no controls. The window is the content: a stack of cards, each a metric that is
either worth a glance or not drawn at all, and the window is exactly the size of
what it is currently showing.

`peripheral-battery-monitor` is three years of that archetype in PyQt6, and is
a candidate for a port to Go and Fyne. The rules it works by are worth more
than the program, and are worth writing down before a port rather than
rediscovering them during one — the same bet the design system made when it was
transcribed out of angou, nmsbonker and clockwork-orange.

### Half the design system inverts

Enough of it that a section inside `docs/design-system.md` would mean
qualifying its rules with "except in a status widget":

- One scroll per section becomes **no scroller anywhere**. Content that does
  not fit is content that should have been hidden.
- A default window of 1180 x 760 becomes **whatever the content measures**.
- "A button that cannot be used is disabled, never hidden" becomes **a card
  with no data is hidden**. A disabled row still occupies a row; an absent
  sensor should cost nothing.
- Every long call gets a busy indicator becomes **polls are silent**. A busy
  popup over a 260 px window covers the window.

What does not invert: `Status`, the palette, the fonts, the threading rules,
the copy voice, and "nothing transient may reflow the interface" — which gets
*harder*, because the unit that moves is the whole panel rather than a row.

### What the pinned Fyne cannot do

Read out of `fyne.io/fyne/v2 v2.8.1`, not assumed:

| Need | Available |
|---|---|
| Frameless | `desktop.Driver.CreateSplashWindow()` |
| Always on top | `desktop.Window.RequestAlwaysOnTop()`, before `Show` |
| Position | `RequestPosition`, documented as ignorable on Wayland |
| **Translucency** | **No.** The desktop backend never requests `glfw.TransparentFramebuffer` and asks for no alpha bits |
| **Read position back** | **No.** GLFW's position callback is internal; nothing on `fyne.Window` exposes a position |
| Shrink to content | Only on request: `fitContent` raises the requested size and never lowers it |

The last three are quirks 32, 33 and 34. None has a workaround inside the
process, which is why the constraints are recorded before the code rather than
discovered halfway in.

The monitor's translucency has an answer that is not Fyne's: a KWin rule sets
`opacityactive`, and the compositor applies it to a window whose toolkit knows
nothing about it. Whole-window only, never per-pixel.

## Requirements

- R1 A rulebook for the archetype as its own document, cross-referenced from
  `docs/design-system.md`, naming which of its rules invert and why.
- R2 The Fyne constraints recorded as quirks, each with the source location
  that proves it and a canary where one can be written.
- R3 Desktop-specific guidance for what the compositor grants, KDE Plasma
  first, other desktops named as unwritten rather than guessed at.
- R4 An implementation large enough to find out what the API wants to be: the
  window, the card stack, the rows, the value formatting and the plot.
- R5 The KDE window rule written by the library, changing only its own rule and
  keeping every other line of the user's file.
- R6 An example program in the shape of the monitor, with a headless test, as
  the acceptance test for R4 and R5.
- R7 Value formatting that cannot change width with the value, since the window
  is sized to its content.

## Acceptance Criteria

- [x] AC1 `docs/glance.md` exists, is linked from `docs/design-system.md`, the
      README and the root `doc.go`, and opens with the table of what inverts.
      (R1)
- [x] AC2 Quirks 32, 33 and 34 are in `docs/fyne-quirks.md`, each citing the
      Fyne source that shows it. Quirk 34 has a real canary,
      `glance.TestAHiddenCardShrinksTheWindow`; 32 and 33 are recorded as
      planned, because a canary for an absent API has nothing to call. (R2)
- [x] AC3 `docs/glance.md` carries a Desktop integration section with the KWin
      rule keys, the rule-type vocabulary, why position is not a rule, and the
      Scripting D-Bus API for the position work that is not built. GNOME,
      Windows and macOS are listed as unwritten. (R3)
- [x] AC4 Package `glance` builds a frameless, always-on-top, fixed-size window
      over `CreateSplashWindow`, falling back to an ordinary window on a driver
      without it so the panel is exercised headless. (R4)
- [x] AC5 A card is drawn only when the user allows it **and** its source has
      answered, the two are separately settable, and a card that stops being
      drawn is told so it can stop its poll.
      `TestACardIsDrawnOnlyWhenAllowedAndAvailable` and
      `TestHidingACardStopsItsPoll`. (R4)
- [x] AC6 Hiding a card shrinks the window.
      `glance.TestAHiddenCardShrinksTheWindow` and
      `TestTheWindowGrowsWithItsCardsAndShrinksAgain`. (R4, R2)
- [x] AC7 A source that stops answering keeps its last values, dimmed, and
      marks its header; recovery clears the marker.
      `TestAStaleCardKeepsItsValuesAndMarksItsHeader`,
      `TestASourceThatStopsAnsweringDimsRatherThanBlanks`,
      `TestRecoveryClearsTheStaleMarker`. (R4)
- [x] AC8 Every formatter is the same width at every magnitude, including its
      unknown value, and the magnitude boundary does not move the column.
      Thirteen tests in `glance/format_test.go`. (R7)
- [x] AC9 A reading changing magnitude does not resize the window, asserted
      over 500 polls of the example.
      `TestTheWindowNeverResizesBecauseAValueChanged` and
      `TestEveryValueOnScreenIsAFixedWidthOne`. (R7)
- [x] AC10 `glance.Sparkline` scales each trace to its own range with a minimum
      span and carries no legend. `TestEachTraceIsScaledToItsOwnRange`,
      `TestAMinimumSpanKeepsAnIdleTraceFlat`. (R4)
- [x] AC11 `glance/kwin` installs, looks up and removes the rule, keeps every
      other rule and KDE's own sections, updates in place rather than adding a
      second rule for the same window, clears a stale position rule, and
      refuses a value carrying a line break. Fifteen tests in
      `glance/kwin/rule_test.go`; the fourteen that touch the filesystem
      sandbox `XDG_CONFIG_HOME` into a temporary directory, and the fifteenth
      only names a D-Bus method. (R5)
- [x] AC12 `examples/glance-monitor` runs as a panel of three cards with a
      two-trace sparkline and a context menu as its only interface, and offers
      the KWin rule behind `-kwin` without ever writing it on a plain run.
      `TestTheWindowRuleIsOfferedAndNotInstalled`. (R6)
- [x] AC13 `go test -race ./...` passes headless and `make lint` reports 0
      issues.

## What this deliberately does not build

Named here so the next spec starts from a list rather than from a reading of
the code:

- **Position restore.** The KWin Scripting D-Bus route is described in
  `docs/glance.md` and not implemented. It needs a session bus client, which is
  the dependency decision below.
- **The D-Bus `reconfigure` call.** `kwin.ReconfigureCall` returns the method
  coordinates as data instead, so the library takes no bus dependency.
- **Desktops other than KDE.**
- **Scaling padding with the text size.** The monitor multiplies every margin
  by the font scale with a floor; `glance` uses the theme's padding as it
  stands. The rule is written down and not yet built.
- **Alerting.** `docs/glance.md` says a glance window must not be the only
  place a failure is visible, and nothing in the package sends a notification.
  It is the program's, until two programs want the same thing.
- **A gallery section.** Every exported identifier is meant to appear in the
  gallery; `glance` does not yet, because a frameless always-on-top window
  inside a gallery window is a different problem.

## Risks & Assumptions

- **The API is a first cut and will change.** The module is `v0.x` and this
  package has no consumer yet. The example is the only thing holding the shape,
  and a real port will move it.
- **Quirks 32 and 33 may be fixed upstream** rather than worked around. Both
  are absent APIs, so both would arrive as new ones and neither breaks
  anything here when they do; the opaque design stays correct either way.
- **`glance/kwin` writes to the user's configuration.** It is an explicit call
  a program makes, never an automatic one, it changes only the rule matching
  its own app ID, and it writes through a temporary file and a rename so an
  interrupted write cannot leave a half file. The values it interpolates are
  refused if they carry a line break, since one would inject keys into a file
  holding every window rule the user has.
- **A direct D-Bus dependency is deferred, not decided.** `godbus/dbus/v5` is
  already in the module graph through Fyne, so the bus is reachable without a
  new module — but promoting it to a direct dependency of a library package
  runs against "Fyne is the only required runtime dependency of the core
  packages". That is a call for the spec that needs the bus, which is the
  position-restore one.
- **The rules are one program's experience**, and a second consumer may
  disagree with some of them. Where the monitor's reason is recorded, the
  reason travels with the rule, so a disagreement can be argued rather than
  guessed at.
- **Rollback** is a revert of two commits. Nothing existing changed behaviour:
  `docs/design-system.md`, the README and `doc.go` gained cross-references, and
  every other file is new.

## Alternatives Considered

- *A section inside `docs/design-system.md`.* Rejected: too many of its rules
  invert, and qualifying each one with "except in a status widget" would make
  the application-window rulebook harder to read to save a file.
- *Wait for Fyne to support a transparent framebuffer.* Rejected: it would
  block the archetype on an upstream change nobody has asked for, and the
  compositor grants whole-window opacity today. The design is opaque and
  depends on nothing seeing through the window, so upstream support would be an
  improvement rather than a prerequisite.
- *Implement dragging by calling `RequestPosition` from a mouse handler.*
  Rejected: on Wayland it is a request the compositor may ignore on every
  frame, and there is no position to read back to know whether it worked. The
  desktop places the window; the rule is documented instead.
- *Use a configuration library for `kwinrulesrc`.* Rejected: the file holds
  every window rule the user has, and a parser that normalised it — reordering
  sections, dropping comments, folding key case — would hand back a file that
  was not theirs. KWin's keys are case sensitive, so folding case produces a
  rule it silently ignores.
- *Build cards as stateless build functions, as `shell` sections are.*
  Rejected: a card is redrawn several times a minute from a poll, and
  rebuilding the window at that rate would fight the no-reflow rule. `Row` is
  a live handle, and it is the one place this archetype departs from "rebuild
  the section from state".

## Executive Summary

A second window archetype: frameless, always-on-top status panels, sized to
their content, read without being interacted with. `docs/glance.md` is the
rulebook, transcribed from `peripheral-battery-monitor`; package `glance` is a
first implementation and `examples/glance-monitor` is what drove its API;
`glance/kwin` writes the KDE window rule for the parts Fyne cannot reach.

Reviewers should start with `docs/glance.md`'s "What inverts" table, which is
the argument for a separate document, and then `glance/format.go` — the
fixed-width value formatting is the most opinionated part of the API and the
place the no-reflow rule is actually enforced. `glance/kwin/rule.go` is the
only code that writes to the user's machine and is worth reading for that
alone.

Verified with `go test -race ./...` headless and `make lint` at 0 issues.
Three Fyne constraints are recorded as quirks 32, 33 and 34, each cited to the
source that shows it.
