# 044 — The hint does not outlive the window

**Issue**: #114

## Context

GLFW window hints are sticky global state: a hint set once applies to every
window created afterwards, until something changes it back. Fyne never calls
`glfw.DefaultWindowHints`, and `grantTranslucent` relies on exactly that — it
sets `TransparentFramebuffer` and leaves it set so that the window Fyne is
about to create, the glance panel, inherits it. The handover is the whole
mechanism and the file says so.

What the file does not say is when the handover ends. The hint is cleared only
when the probe is *refused* (`glance/translucent_glfw.go:29,36`). Where it is
granted — which is the case on any desktop this feature is for — nothing clears
it, so every window the process creates from then on is born with a transparent
framebuffer, including the ones the library does not own.

hayami reported it. It has a second window, and opening the preferences window
from the panel drew that window and its dialogs see-through: the desktop
legible through a font chooser. It was first taken for an interaction with a
compositor blur effect, and disabling that effect entirely left the preferences
window just as transparent, which is what isolated it to this.

The sibling of this bug is already fixed and its comment is still in
`glance/window.go`: "A theme is app-wide, so a program with a second window had
that window turned transparent too." The theme leak was closed in spec 042. The
framebuffer hint beside it was not, and it is the same fault one layer down —
the library reaching outside the window it was asked for.

## Requirements

### The hint is cleared once it has been taken up

`Window.ShowAndRun` clears `TransparentFramebuffer` immediately after
`w.win.Show()` returns, through a `clearTranslucent` beside `grantTranslucent`
and built under the same tags — the real one calling `glfw.WindowHint`, the
stub on a build without GLFW doing nothing, as `grantTranslucent` already does.

Unconditionally, not only when the grant succeeded. A refusal already clears
it and clearing twice costs nothing; a cleared hint that is cleared again is
the state we want either way, and a condition here is a second thing that can
be wrong.

**After `Show` and not before.** The hint has to survive until Fyne actually
creates the GLFW window, and that happens inside `Show`: `ShowAndRun` runs its
body through `fyne.Do`, so the body is on the main goroutine, so Fyne's
`async.EnsureMain` runs inline rather than queueing, so `window.create` and its
`glfw.CreateWindow` are called before `Show` returns. Clearing any earlier
costs the panel its own translucency — silently, and only on the desktops that
would have granted it, which is the hardest kind of regression to notice.

This is a fact about Fyne's internals and it is load-bearing, so it is written
down here rather than trusted to stay true: Fyne 2.8.1,
`internal/async/goroutine.go:52` and `internal/driver/glfw/window_desktop.go:838`.

### Nothing else changes

The panel is still translucent where the desktop grants it, and still falls
back to opaque where it does not. This spec adds no setting, no option and no
new exported name; the only observable change is in windows that were never
supposed to be transparent.

## Acceptance criteria

- [x] `clearTranslucent` exists under both build tags, is called from
      `Window.ShowAndRun` immediately after `w.win.Show()`, and is called
      whether or not the grant succeeded.
- [x] A second window created after the panel is opaque.
- [x] The panel itself is still translucent, verified in a photograph rather
      than by reading the code — the failure mode is invisible to a headless
      test and shows only on a desktop that grants the hint.
- [x] `go test ./...` passes, and the pinned linter reports exactly what it
      reports on `main` — seven issues, all in files this spec does not touch
      (`fd-wrap/*`, `glance/meter.go`). `make lint` is red on `main` already;
      that is its own problem and not this one's to fix or to hide.
- [x] hayami, which reported it, draws a solid preferences window and a
      translucent panel against this change.

## Risks & Assumptions

- **The timing is the whole risk.** Clearing before Fyne creates the window
  takes the panel's translucency away, and neither a test nor a build catches
  it: the panel simply clears to black or to the theme on the desktops that
  would have shown through. The photograph in the acceptance criteria is the
  check, and it has to be taken on a desktop that grants the hint.
- **It depends on Fyne calling `create` inside `Show`.** True at 2.8.1 and
  cited above. A Fyne release that deferred window creation would break the
  handover, and the symptom would be the panel losing translucency rather than
  anything crashing.
- **Rollback**: revert the commit. Nothing is persisted, no exported shape
  changes, and no consumer has to do anything to take or leave it.

## Status: COMPLETE
