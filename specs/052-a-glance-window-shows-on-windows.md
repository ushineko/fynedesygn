# Spec 052: a glance window shows on Windows

**Issue**: [#160](https://github.com/ushineko/fynedesygn/issues/160)

## Status: INCOMPLETE

## Context

hayami's desktop panel, a translucent `glance.Window`, was brought up on
Windows 11. Two things stood between it and a usable panel.

**It panicked before drawing anything:**

```
panic: runtime error: invalid memory address or nil pointer dereference
github.com/go-gl/glfw/v3.4/glfw.(*Window).GetWin32Window
fyne.io/fyne/v2/internal/driver/glfw.(*window).setDarkMode      window_windows.go:17
fyne.io/fyne/v2/internal/driver/glfw.(*gLDriver).applyThemeToWindow
fyne.io/fyne/v2/app.(*settings).SetTheme
github.com/ushineko/fynedesygn/glance.(*Window).ShowAndRun.func1   window.go:280
```

`ShowAndRun` sets the transparent theme after the grant and before `Show`.
On Windows every theme change calls `setDarkMode` on every window, which asks
the GLFW view for its `HWND`; the view does not exist until `Show` creates
it. Linux has no `setDarkMode`, so the order never mattered there.

**Once it drew, it could not be moved.** A frameless window on Windows has no
titlebar to drag and Windows has no Meta-drag, so the panel opened centred,
always on top, over whatever was there, and stayed. `docs/glance.md` records
that on Linux the compositor moves the window; Windows has no such fallback.

The translucency itself works on Windows once the window is up: the GLFW
probe is granted and the desktop shows between the cards.

## Requirements

- R1 `ShowAndRun` applies `theme.WithTransparentBackground` after
  `w.win.Show()`, in the same main-thread callback, and only when the grant
  succeeded. The panel's own translucency is still set before `Show`.
- R2 `docs/fyne-quirks.md` records the panic as quirk 43, with the advice for
  a program that changes the theme itself.
- R3 On Windows, a primary press anywhere on a frameless glance window moves
  it: the catcher is `desktop.Mouseable` there, releases GLFW's mouse capture,
  sends `WM_NCLBUTTONDOWN` with `HTCAPTION`, and posts the release the move
  loop swallowed so Fyne does not believe the button is still down.
- R4 A frameless glance window on Windows has the catcher whether or not it
  has a menu. On other platforms the catcher is unchanged and is not
  `Mouseable`.
- R5 `docs/glance.md` says how the window moves on Windows and what that
  costs: a primary click reaches nothing inside a glance window there.
- R6 README changelog under `### Unreleased`.

## Acceptance Criteria

- [ ] `make test` passes on Linux (CI).
- [x] `go test -tags migrated_fynedo ./...` passes on Windows 11 (25
  packages).
- [x] On Windows 11, hayami's panel starts, stays up and is translucent:
  screenshot looked at.
- [x] On Windows 11, a real pointer dragging the panel moves it by the
  distance dragged, and a right-click still opens the menu afterwards.
- [ ] On Linux (Plasma), the panel's first frame is still translucent, with
  no opaque flash: screenshot looked at.

## Risks & Assumptions

- **Assumption**: the first frame is drawn after the `fyne.Do` body
  returns, so setting the theme after `Show` in that body changes nothing
  visible. To be confirmed on Linux by the last criterion.
- **Not covered**: a program that creates a second window and leaves it
  unshown while changing the theme still panics on Windows. That is Fyne's
  and is documented in the quirk rather than worked around.
- **A primary click is taken on Windows.** Nothing in a glance window is
  meant to be clicked; a program that put a control in one would find it dead
  on Windows. Documented in `docs/glance.md`.
- **The move loop blocks the main thread** while the button is held, as a
  titlebar drag does for every Win32 window; the panel does not repaint
  mid-drag.
- **Rollback**: revert.

## Verification

2026-09-30, Windows 11 Pro 26200, through hayami (its spec 033):

- Before: the panic above on every start. After: the panel starts and
  stays up; a DPI-aware screen capture shows the cards with the desktop
  between them.
- Drag: pointer pressed at (1794, 821), moved by (-600, -300) in twenty
  steps and released; the window went from (1734, 781) to (1134, 481). The
  tip under the pointer then appeared on hover as normal.
- Right-click on the panel after the change opened the menu (Preferences,
  Opacity, Quit), as it does in a build without it.
