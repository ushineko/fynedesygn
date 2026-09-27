# Spec 036: a window that looks like the monitor

**Issue**: [#73](https://github.com/ushineko/fynedesygn/issues/73)

## Status: COMPLETE

## Executive Summary

A glance window now carries a surface's margins rather than the scheme's
control padding, runs to the window's edge, and draws its card titles in the
inactive foreground. `glance.Options.Translucent` gives a window the desktop
shows through, which this page said Fyne could not draw; `Window.SetOpacity`
and `kwin.OpacityScript` fade a window that is already on screen, which a KWin
rule cannot. The rule's own opacity has never been applied at all and is
fixed. Reviewers should start with the measurement table in "Context", then
`glance/translucent_glfw.go`, which is the only new mechanism.

## Context

Every claim below came from screenshotting the two programs side by side and
reading the pixels. Three of the four corrections contradict something this
repository had written down after reading source instead of running it, which
is the finding worth keeping.

**Margins.** A card used `container.NewPadded`, which is the scheme's control
padding: 3 px in Breeze. The monitor uses 15 px at the sides, 8 above the
title, 10 under the last row, and 11 between one section and the next, at its
11 px face. Measured from a capture of the running monitor: card edge at x=1,
text at x=24 device pixels on a 1.5 scale display, and a 17-device-pixel gap
between sections. The card radius, the fill, the border and the text sizes
were already right; the margins were the whole visible difference.

**Translucency.** `docs/glance.md` said "Fyne cannot draw this (quirk 32), and
a program that waits for it will wait without end". The painter has always
cleared the frame with `theme.Color(ColorNameBackground)` *including its
alpha* (`internal/painter/gl/painter.go`). What is missing is the window: the
desktop backends never set `glfw.TransparentFramebuffer` and ask for no alpha
bits. Hints are sticky and Fyne never calls `glfw.DefaultWindowHints`, so
setting the hint on the main thread between GLFW's initialisation and the
window's creation is enough, with no fork. Measured on Plasma 6 on Wayland:

| | gap between two cards | desktop behind it |
|---|---|---|
| hint and an alpha-zero background | (34,28,28) | (34,28,28) |
| hint only | (32,35,38) | — |
| alpha-zero background only | **(0,0,0)** | (78,81,89) |

The third row is why the grant is probed for rather than assumed: a window
that asks and is refused clears to black, not to the desktop, and a black
rectangle is worse than an opaque panel. Refusal is reported on AMD drivers on
Windows (glfw#2731) and on dedicated laptop GPUs (glfw#1288). Upstream has the
same feature in draft (fyne#6338), stalled on whether it belongs on
`fyne.Window` or on the `desktop.Window` interface 2.8 introduced, so this is
not a fork of something about to land.

**The rule's opacity has never worked.** The package wrote `4` for
`opacityactiverule` under the name `ruleApplyInitially`. KWin's `SetRule` enum
is DontAffect 1, Force 2, Apply 3, Remember 4, so `4` asks KWin to remember
whatever opacity the window already has. Measured at 100, 95, 80 and 50 %: the
window was opaque at every percentage at `4` and at `3`, and tracked the blend
against the desktop exactly at `2`. `docs/glance.md` had reasoned its way to
the wrong value, preferring "apply initially" over "force" so the window menu
would stay free; `3` is the real "apply initially" and it does not apply an
opacity either.

**Opacity while the window is up.** A rule applies at the window's creation,
so it cannot carry the monitor's opacity menu. Two mechanisms can. On X11 a
client writes `_NET_WM_WINDOW_OPACITY` on its own window, which is the whole
of GLFW's X11 implementation. On Wayland it cannot — GLFW's Wayland backend
answers `GLFW_FEATURE_UNAVAILABLE` — and KWin's scripting API is how the
compositor is asked. Spec 035 deferred scripting helpers on the grounds that
the program makes the calls with the D-Bus client it already has; that holds,
and what is added here is the script text and the method names, as data.

## Requirements

- R1 A card is inset by a surface's margin and not the scheme's control
  padding; the stack runs to the window's edge; the gap between two cards is
  the monitor's. Each is a factor of the text size, not a pixel count.
- R2 A card's title is drawn in the inactive foreground, not the full one.
- R3 `glance.Options.Translucent` gives a window the desktop shows through
  where the driver allows it, and an opaque window where it does not. The
  grant is probed for before the theme commits to an alpha of zero.
- R4 Only the `Background` role goes transparent. `OverlayBackground` does
  not, or a glance window's context menu — its whole interface — disappears.
- R5 `kwin.Rule` writes an opacity that KWin applies.
- R6 `glance.Window.SetOpacity` fades a window that is already on screen where
  the program can do it itself, and says which mechanism is needed where it
  cannot. `kwin.OpacityScript` is that mechanism on Plasma.
- R7 `docs/glance.md` and `docs/fyne-quirks.md` record what was measured,
  including that they were wrong and how.
- R8 `glance` and `theme` still build with `CGO_ENABLED=0`.

## Acceptance Criteria

- [x] R1: `TestACardIsInsetFromItsOwnEdge`, `TestCardsAreSeparatedByTheCardGap`, `TestTheCardStackRunsToTheWindowsEdge`. Each asserts the applied geometry *and* that the value has not collapsed back to the padding token; the first two passed against a reverted value until that second assertion was added.
- [x] R2: `TestACardsTitleIsMutedAgainstItsRows`.
- [x] R3: `TestATranslucentWindowIsOpaqueUntilItIsGranted`, and on screen: the gap between two cards comes back byte-identical to the desktop pixel behind it, the cards stay opaque.
- [x] R4: `TestATransparentBackgroundLeavesAPopupOpaque`, and on screen: the context menu drawn over a translucent panel is solid.
- [x] R5: `TestInstallingIntoAFileThatDoesNotExistYetCreatesIt` asserts `opacityactiverule=2`; measured, a window under a package-installed rule at 50 % is no longer opaque.
- [x] R6: `TestAWindowThatCannotFadeItselfSaysSo`, the four `kwin.OpacityScript` tests, and on screen both routes: `_NET_WM_WINDOW_OPACITY` under XWayland (card (41,44,48) to (128,121,83) at 50 %) and the KWin script on Wayland (card (41,44,48) to (68,68,68)).
- [x] R7: the documents read; the style canaries pass.
- [x] R8: `CGO_ENABLED=0 go build ./glance/... ./theme/`. The gallery does not build without cgo and does not on `main` either.

## Risks & Assumptions

- **The probe is the safety.** `grantTranslucent` makes a one-pixel hidden
  window, reads the attribute back and destroys it. Fyne hands back no
  `glfw.Window` — `glfw.GetCurrentContext()` is nil on the main thread outside
  a paint, measured — so a window of our own is the only way to ask. If the
  probe ever answers yes where the real window would answer no, the failure is
  a black panel.
- **The hint is set between two events we do not control.** GLFW must be
  initialised and Fyne's window must not yet exist. `ShowAndRun` queues the
  show with `fyne.Do` from a goroutine for that reason: called on the main
  goroutine before the loop is running, `fyne.Do` executes inline, which is
  too early. A change to Fyne's startup order would break this quietly, which
  is what `Window.Translucent()` reporting false is for.
- **`driver.RunNative` passes its context by value.** An assertion on
  `*driver.X11WindowContext` compiles, never matches, and looks exactly like a
  Wayland session. This cost a cycle and is now a comment at the call site.
- **The KWin script interpolates an app ID into JavaScript.** It comes from
  the program rather than from a user, and it is quoted with `%q`, with a test
  that an app ID carrying a quote cannot close the literal.
- Opacity and translucency are independent, measured: opacity moves the card
  and leaves the gap, translucency moves the gap and leaves the card.
- Rollback: revert the commit. Every new identifier is additive and
  `Options.Translucent` defaults to off, so a consumer that does not ask for
  any of this sees only the margins change.

## Alternatives Considered

- A `TransparentPanel` widget. Rejected: a widget draws inside the window and
  cannot change how the window was created, and per-pixel alpha inside an
  opaque window already worked.
- Waiting for fyne#6338. Rejected: it is a draft that missed 2.8 with no
  milestone, and the API it proposes is under review. When it lands,
  `Options.Translucent` becomes a thin adapter over it.
- Making the palette's `WindowBG` transparent rather than the `Background`
  role. Rejected: `OverlayBackground` reads the same palette entry, and a
  transparent popup is an invisible context menu.
- A D-Bus client in `glance/kwin` for the opacity script. Rejected for the
  reason spec 035 gave and `ReconfigureCall` documents: the library keeps to
  Fyne as its only dependency, and the program owns the bus.
