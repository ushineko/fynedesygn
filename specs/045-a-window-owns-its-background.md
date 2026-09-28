# 045 — A window owns its background

**Issue**: #118

## Context

A glance panel is supposed to be invisible between its cards: not blended, not
faded, *absent* — alpha 0, with the desktop unobstructed. It is drawing filled
gaps instead.

The alpha channel is not the problem. `grantTranslucent` asks for a transparent
framebuffer and gets one. What fills the gaps is the clear:

```go
// fyne internal/painter/gl/painter.go:77
func (p *painter) Clear() {
	r, g, b, a := theme.Color(theme.ColorNameBackground).RGBA()
```

Fyne clears every window's framebuffer from the **global application theme**.
`Panel.bg` is a rectangle inside the window's content and is therefore painted
*after* that clear, so once the clear is opaque nothing inside the content can
restore alpha — it can only paint over it.

`6922478` is where this started. It replaced the app-wide transparent theme
with the panel-owned rectangle:

```diff
-	p.bg.FillColor = theme.Color(theme.ColorNameBackground)
+	p.bg.FillColor = p.BackgroundColour()   // color.Transparent when translucent
```

The fault it fixed was real: a Fyne theme is app-wide, so a program with a
second window had that window turned transparent too, and a window whose
framebuffer has no alpha clears to **black**. But the replacement cannot reach
the clear, so it traded a visible fault for a silent one.

`theme.WithTransparentBackground` has been orphaned since: nothing calls it,
and its own doc still says "`glance.Options.Translucent` asks for both, in the
right order", which stopped being true at that commit.

## Requirements

### A window declares its background; it does not inherit one

This is the continuation of spec 042, which gave the panel its own face while
the application's stayed with the window that has overlays. Background is the
same kind of property and it is the one that was left behind.

The arrangement, which is the archetype's: **nothing paints globally, and each
window paints itself.** `peripheral-battery-monitor` sets
`WA_TranslucentBackground` on the panel widget alone, paints its cards with an
explicit `rgba(43, 43, 43, alpha)`, and forces everything else to
`background: transparent`. Qt scopes that per widget. Fyne has exactly one
global clear, so the same arrangement is reached from the other side: the clear
is set to nothing, and every window that wants a background paints one.

### The app theme carries the transparent background, behind the grant

`Window.ShowAndRun` wraps the **application** theme in
`theme.WithTransparentBackground` when — and only when — `grantTranslucent`
returned true. That is the only lever that reaches `painter.Clear`.

**Only behind the grant.** A window whose framebuffer has no alpha clears a
transparent background to black, which is worse than an opaque panel. The
existing probe already gates the rest of the translucent path and gates this
too.

Only `ColorNameBackground` changes. `OverlayBackground` deliberately does not,
so dialogs, popup menus and `Select` dropdowns stay opaque — a transparent
popup menu would make the only interface a glance window has invisible.

### The clear survives everything that sets the app's theme

Wrapping once in `ShowAndRun` is not enough, and the gap was found by opening
the preferences window: the panel went solid again. `shell` sets the
*application's* theme as it is constructed, and a fresh theme has an opaque
background, so building a second window put the clear back.

Every place in this library that sets the app theme goes through
`theme.KeepTransparentBackground`, which re-wraps when the current background
is already clear. Four sites: two in `shell`'s construction, one in
`SetAppearance`, one in `Appearance.Apply`.

Detected rather than remembered. The window that asked for the clear is a
different one and may not exist yet, so an alpha of zero on the current
background is the whole of the signal — and a program that never had a
translucent window is untouched, because its background was never clear.

### A shell window paints its own background

`shell` puts a rectangle at the bottom of its content stack, filled from its
own theme's `ColorNameBackground`, and repaints it when that theme changes.
`glance.Panel` has done this with `bg` since `6922478`; the shell needs it for
the same reason and now needs it for real, because the clear underneath it is
no longer opaque.

Inside the subtree override where there is one, so a shell with
`OwnAppearance` paints its own scheme's background rather than the
application's.

### What a consumer has to do

Only a program with a translucent glance window whose grant succeeded is
affected at all. For it, two things, and neither applies if its other windows
come from `shell`:

- **A window built by hand** — `App.NewWindow` and `SetContent` — clears
  transparent too and draws see-through. It paints its own background: a
  `canvas.Rectangle` filled from its theme's `ColorNameBackground`, under its
  content, as `Panel` does with `bg`.
- **Setting the app's theme** afterwards refills the panel's gaps. Wrap with
  `theme.KeepTransparentBackground`, or go through `shell`.

Unaffected: a program with no glance window, one with `Translucent: false`,
one whose grant was refused, every `shell` window, and every dialog and popup
menu — only `ColorNameBackground` is wrapped.

In this repository nothing needs changing: only `shell` and `glance` create
windows, and the example goes through `glance.NewWindow`.

## Acceptance criteria

- [x] A translucent glance panel's gaps are alpha 0: the desktop is
      unobstructed between the cards, verified in a photograph.
- [x] A `shell` window in the same process is solid, verified in the same
      photograph.
- [x] The app theme is wrapped only when `grantTranslucent` returned true.
- [x] `WithTransparentBackground` is called again, and its doc's claim about
      `Options.Translucent` is true once more.
- [x] A shell window with `OwnAppearance` paints its own theme's background,
      not the application's.
- [x] Opening a second window does not refill the panel's gaps, verified in a
      photograph with the preferences window open.
- [x] `Options.Translucent` documents both consumer obligations.
- [x] Dialogs and popup menus are unchanged.
- [x] `go test ./...` passes and `make lint` is clean.

## Risks & Assumptions

- **Every `shell` consumer inherits "no background unless painted".** The
  library paints it, so no consumer has to act — but a consumer that builds a
  window some other way, outside `shell`, now gets a transparent one in a
  process that also runs a translucent panel. That is the cost of the clear
  being global and it is worth saying out loud.
- **The refused-grant path cannot be tested on the machine this was written
  on**, which grants it. The failure mode there is a black window rather than
  a transparent one, and it is guarded by the same probe that already gates
  the framebuffer hint — but it is guarded, not demonstrated.
- **This is a workaround for a Fyne limitation**, recorded as #119: the
  painter already holds its canvas and could resolve the theme from it. If
  that lands upstream, the app-theme wrap and the shell's rectangle both stop
  being necessary and the library goes back to per-window themes meaning what
  they say.
- **Rollback**: revert the commit. No setting changes meaning and nothing is
  persisted.

## Status: COMPLETE
