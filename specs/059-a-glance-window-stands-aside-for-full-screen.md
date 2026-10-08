# Spec 059: a glance window stands aside for a full-screen window

**Issue**: [#178](https://github.com/ushineko/fynedesygn/issues/178)

## Status: COMPLETE

## Context

A glance window is always on top. On Windows that keeps it over a borderless
full-screen window, which is what a browser in full screen (F11, or a web
player's full-screen button) and most current games in "windowed" or
borderless full screen are: a window sized to the monitor. hayami's
maintainer asked for the panel to stand aside while such a window is in front
and come back on returning to the desktop. Exclusive Direct3D full screen
covers it already.

On KWin an active full-screen window is in a layer above keep-above windows,
so the window manager does this already. Linux is a no-op, to be confirmed on
a KWin desk.

## Requirements

- R1 `Window.SetHideForFullscreen(on bool)`, off until asked for, can be
  turned on and off at any time; off, a window it hid is shown.
- R2 On Windows, every 500 ms: the window in front is full screen when its
  window rectangle (`GetWindowRect`) or the frame DWM draws
  (`DWMWA_EXTENDED_FRAME_BOUNDS`) **contains** the full rectangle of its
  monitor (`rcMonitor`, taskbar included): left <=, top <=, right >=,
  bottom >=. Containment, not equality, because a full-screen window is
  often a few pixels larger than its monitor; either rectangle, because
  `GetWindowRect` includes a resizable window's invisible borders and DWM
  gives no frame for some windows.
- R3 Not full screen: this program's own windows, the desktop and the
  taskbar (`GetShellWindow`; `Progman`, `WorkerW`, `Shell_TrayWnd`,
  `Shell_SecondaryTrayWnd`), a window that is invisible, minimised or
  cloaked, a window on another monitor than the glance window's, and a
  maximised window with a titlebar. That last is deliberate: a maximised
  window leaves the taskbar showing, and even where a hidden taskbar lets it
  cover the monitor it is not full screen. Hiding for maximised windows could
  be a later option.
- R4 Geometry alone decides. `SHQueryUserNotificationState` reports a
  full-screen app for exclusive Direct3D and presentation mode, and misses
  windowed and borderless full screen, the cases that matter; it is not
  consulted.
- R5 Two looks in a row decide a hide or a show, so a window passing through
  full screen (a game resizing, a browser on its way in) does not flicker the
  glance window.
- R6 Hidden with `ShowWindowAsync(SW_HIDE)` and shown with
  `ShowWindowAsync(SW_SHOWNOACTIVATE)`: posted to the window's thread, which
  is Fyne's main loop, so the watch never waits on it; the show never takes
  the focus; neither moves the window or changes its always-on-top.
- R7 Linux and macOS: no-op, documented.

## Acceptance Criteria

- [x] Table tests: exactly the monitor; 8 px larger on every side; 1 px
  short on the right and at the bottom (not full screen); a maximised window
  with the taskbar showing (not); a maximised window with a titlebar and the
  taskbar hidden (not); a maximised window without a titlebar, a browser
  after F11 (full screen); a borderless game window exactly the monitor's
  size and not maximised, as measured on Terraria in windowed full screen
  (full screen); either rectangle containing the monitor; own window, shell,
  hidden and another monitor (not); the two-look rule.
- [x] A real-window test (`FYNEDESYGN_WINDOW_TEST=1`): the glance-monitor
  example with `-hide-fullscreen`, a borderless popup exactly the monitor and
  one 8 px past every side each hide it, and closing the popup brings it back
  in the same place without the focus.
- [x] Without the option, the same popup leaves it shown.
- [x] Falsified: containment made always false fails three table tests and
  the real-window test ("did not hide").
- [ ] Confirmed on a KWin desk that an active full-screen window covers a
  glance window with nothing from this spec.

## Risks & Assumptions

- **Fyne still counts the window as shown.** A program that hides the
  window through Fyne while the option is on should turn it off first, or
  the watch may show it again. Documented on `SetHideForFullscreen`.
- **The position code and snapping are unaffected**: `GetWindowRect` answers
  the same for a hidden window, and hiding sends no `WM_MOVING`.
- **A window of this program in front** (a preferences window) never counts
  as full screen.
- **The real-window test takes the foreground for a few seconds.** It skips
  when a full-screen window is already in front, so it never takes over a
  game or a film.
- **Rollback**: revert; the option is off unless a program asks for it.

## Gaps found

- Not checked against a real browser here: the maintainer was playing a game
  on the desk, and a full-screen Edge window would have taken it over. The
  popup is the same shape (a borderless window sized to the monitor).

## Verification

2026-10-08, Windows 11 Pro 26200, a 3840x2160 monitor at 150 %, Go 1.26.0:

- Table tests pass.
- Real window, the glance-monitor example with `-hide-fullscreen`:
  exactly the monitor, hidden after 980 ms and back after 980 ms; 8 px past
  every side, hidden after 1.03 s and back after 980 ms; back in the same
  place each time, and the foreground was not the glance window.
- Without the option: still shown 2.5 s after the popup came up.
- Falsified as above, and restored.
