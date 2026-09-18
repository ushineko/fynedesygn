# Spec 002: Shell

> **Note**: This work has no associated issue tracker ticket. The repository
> is a personal public project without an issue tracker.

## Status: COMPLETE

## Executive Summary

Adds the `shell` package: the window skeleton every consuming program will
run inside, with `Section`/`Detacher` as the lifecycle seam, `Options` hooks
for header actions, status-bar segments and program-wide loads, the
refcounted busy popup, one-at-a-time banners, `Perform` with a cancellable
form and an inline headless path, the redraw ladder, and the standard
Appearance and About sections. The gallery is rewritten onto it and gains a
Shell section that exercises the runner. Reviewers should look first at
`shell/section.go` and the `Options` struct in `shell/shell.go`: that is the
API the three programs will adopt.

## Context

The window skeleton is the largest shared chunk of the three apps and the one
where they disagree most. All three have: a header strip with the program name
and a Refresh button, a section list on the left at a 0.16 split, one content
scroller, a status bar of segments, a refcounted busy indicator, one result
banner at a time, `perform` running core calls off the UI thread, and the
redraw ladder (status bar, section, both, invalidate). angou still draws the
banner in a fixed 48 px slot and the busy indicator in an 18 px strip;
nmsbonker and clockwork-orange moved both into popups, which is the current
rule ("nothing transient may reflow"). clockwork-orange added a cancellable
busy popup and a `detach()` call before every swap; nmsbonker's `working()`
also counts a build run that lives outside `perform`.

Every one of these is coupled to the app's `ui` god-struct today. This spec
cuts the seam: the shell owns the window, the appearance and the transient
state, and a program plugs in sections, header actions, status-bar segments
and a loader for program-wide state.

The seam decides whether this is a library or a fourth copy. The four tests
from the extraction plan: section lifecycle as an interface with `Build` and
an optional `Detach`; the busy/flash/perform runner decoupled from any
`core.Deps`; the shell owning the section scroll so a nested scroller is
impossible to create by accident; and one place to absorb Fyne churn.

## Requirements

### R1. Section interface

```go
type Section interface {
    Title() string                       // computable before an app exists
    Icon() fyne.Resource                 // called only after the app exists
    Build(s *Shell) fyne.CanvasObject    // stateless: rebuilt on every change
}
type Detacher interface { Detach() }     // optional: forget live widgets and scroll callbacks
```

- R1.1 `NewSection(title, icon func() fyne.Resource, build)` returns a
  `*FuncSection` for programs that build from closures; `OnDetach(fn)` adds a
  detach hook. Programs may also implement `Section` on their own types.
- R1.2 `Names(sections []Section) []string` reads titles only, so `--section`
  flag help and `--version` never construct a theme icon (quirk 8).
- R1.3 The shell calls `Detach` on the outgoing section before building the
  incoming one, never after (clockwork-orange's ordering rule).

### R2. Options and construction

```go
type Options struct {
    AppID     string          // preference store; Wayland app_id; required
    Name      string          // header label and default window title
    Version   string          // appended to the window title
    Icon      fyne.Resource   // window and app icon; optional
    Sections  []Section
    Section   string          // open on this title; empty = first
    Scheme    string          // one-run scheme override, never saved
    Size      fyne.Size       // zero = 1180 x 760
    Header    func(s *Shell) []fyne.CanvasObject  // trailing header actions; nil = Refresh only
    StatusBar func(s *Shell) []fyne.CanvasObject  // segments in order; nil = none
    OnStart   func(s *Shell)  // after the window exists, before it shows: program-wide loads
    OnInvalidate func(s *Shell) // called by Invalidate before the rebuild: drop loaded flags, reload
    OnStop    func(s *Shell)  // before Restart replaces the process: flush a pending save
    AlsoWorking func() bool   // extra "something is running" predicate (a job outside Perform)
}
func New(o Options) *Shell   // ApplyCursorTheme, app.NewWithID, appearance, scale, window content
func Run(o Options)          // New(o).ShowAndRun()
func Headless(app fyne.App, o Options) *Shell // no window; OnScreen() false; for tests
```

- R2.1 `New` runs `theme.ApplyCursorTheme()` before `app.NewWithID`, loads
  `theme.Appearance`, applies the scheme override for the run without saving,
  calls `theme.ApplyScale`, sets the theme, sets the icons, builds the header,
  nav, content scroller, status bar, F5 and Ctrl+R shortcuts, resizes,
  `SetMaster`, selects the start section, calls `OnStart`.
- R2.2 `Headless` gives a shell whose content pane is nil; `OnScreen()` is
  false; `Build` can be called on any section; `Perform` runs inline.
- R2.3 The window skeleton is exactly `docs/design-system.md` "Window
  skeleton". The status bar is a sequential `HBox`. An unknown `Section`
  name opens the first section (a wrong screenshot is obvious, a dead window
  is not).

### R3. Appearance

- R3.1 `Appearance() theme.Appearance`, `SetAppearance(a)` applies and saves,
  except the scheme when the run has a one-run override (the gallery's rule).
- R3.2 `AppearanceSection(monoSample string) Section`: the standard picker
  (scheme, font, monospace font, text size, scale, reset, `theme.Sample`, the
  explanatory notes) so every program's Appearance section is the same one.
  A scale change shows a "restart the window" hint; `Restart()` is `R3.3`.
- R3.3 `Restart()` re-executes the process with the same arguments on Unix
  (`syscall.Exec`) after `OnStop`; on Windows it starts the new process and
  quits. Programs with a pending save hook `Options.OnStop func(s *Shell)`.

### R4. Redraw ladder and navigation

- R4.1 `Select(title)`, `Current() Section`, `RedrawStatus()`, `Refresh()`
  (current section, keeping scroll offset), `Rebuild()` (status bar plus
  section), `Invalidate()` (calls `OnInvalidate`, then `Rebuild`; a no-op past
  the hook when not on screen).
- R4.2 Navigation starts at the top of the section; a rebuild in place keeps
  the offset.
- R4.3 `OnScreen() bool`.

### R5. Busy, gate, perform

- R5.1 `Busy(what string) func()`: refcounted, safe from any goroutine,
  release is idempotent; shows the modal popup after 300 ms with an infinite
  bar 320 wide and the `what` text; hides at zero; calls `Regate` on the 0 to
  1 and 1 to 0 transitions.
- R5.2 `Working() bool` is `busy > 0 || AlsoWorking()`.
- R5.3 `Gate(buttons ...*widget.Button)` disables them when `Working()`.
- R5.4 `Perform(what, fn func(ctx) error)` and
  `PerformCancellable(what, fn)`: refuse with a warning banner when working;
  run inline when not on screen; otherwise a goroutine with `Busy`; errors go
  to `Report`. The cancellable form adds a Cancel button to the busy popup
  that cancels the context.
- R5.5 `Report(what string, err error)`: bad banner `"<what> failed: <err>"`;
  silent for `context.Canceled`.
- R5.6 `OK(msg string)`: good banner then `Invalidate`.

### R6. Banners

- R6.1 `Flash(text string, st fd.Status)`: one banner at a time in a non-modal
  popup centred 56 px above the bottom, width `min(720, canvas-40)`; marker,
  wrapped text, dismiss button; holds good 6 s, warn 12 s, bad until dismissed;
  700 ms fade on the background rectangle; sequence-guarded; no timers when
  not on screen.
- R6.2 The tint is the active palette's role at alpha 0x4d when the theme is a
  `theme.Theme`, else Fyne's Success/Warning/Error colour.
- R6.3 `ClearFlash()` dismisses the current banner.

### R7. About section

- R7.1 `AboutSection(About) Section` where
  `About{Icon fyne.Resource; Name, Version, Blurb, URL string; Notes []Note{Title, Detail}; Facts []Fact{Label, Value}; Extra func(s *Shell) fyne.CanvasObject}`:
  the shape shared by the three apps (72 px logo, name, dim version, wrapped
  blurb, hyperlink, "What it does" notes, "Facts" form). `Extra` is where
  spec 003 hangs the embedded README.

### R8. Gallery on the shell

- R8.1 `cmd/fynedesygn-gallery` is rewritten onto `shell.Run`: its own shell
  code is deleted; it gains a Shell section demonstrating `Perform` (a fake
  three-second job), `PerformCancellable`, all four `Flash` statuses,
  `Gate`, and a status-bar segment that changes while the job runs; and an
  About section built with `AboutSection`.
- R8.2 `--section`, `--scheme`, `--version` behave as before through
  `Options`.

### R9. Canaries and documentation

- R9.1 Quirk 11 canary `TestPerformRunsInlineWhenHeadless`; quirk 13 canary
  `TestScalePreferenceYieldsToTheEnvironment` already exists in `theme`.
- R9.2 `docs/design-system.md` "Sections" and "Progress and results" gain
  the API names; README package table gains `shell`.

## Acceptance Criteria

- [x] AC1 `shell.Names` works with no app; a test constructs sections and
  reads names before `test.NewApp` (R1.2, quirk 8).
- [x] AC2 A section implementing `Detacher` has `Detach` called before the
  next section's `Build`, verified by an ordering log in a headless shell
  with a window from `test.NewWindow` (R1.3).
- [x] AC3 `Headless` builds every section of the gallery without a window;
  `Perform` runs the function inline and returns after it (R2.2, R5.4).
  Canary `TestPerformRunsInlineWhenHeadless`.
- [x] AC4 `Perform` while `Working()` flashes a warning and does not run;
  `AlsoWorking` returning true has the same effect (R5.2, R5.4).
- [x] AC5 `Busy` from two goroutines releases only when both are done; a
  double release is harmless; `Working()` reflects the count (R5.1).
- [x] AC6 `Flash` shows one banner at a time; a bad banner does not
  self-clear; a newer banner supersedes an older one's timer; `ClearFlash`
  empties it (R6.1). Tint for each status is the palette role (R6.2).
- [x] AC7 `Report` with `context.Canceled` shows nothing; with another error
  shows a bad banner containing the error text (R5.5).
- [x] AC8 `SetAppearance` saves the scheme normally and does not save it under
  a one-run override; `AppearanceSection` renders headlessly and its scheme
  select is populated (R3.1, R3.2).
- [x] AC9 `AboutSection` renders headlessly and `fynetest.Text` contains
  the name, version, every note title and every fact (R7.1).
- [x] AC10 The gallery builds, its `--version` prints without an app, every
  section renders headlessly in every scheme, and its own shell code is gone
  (R8).
- [x] AC11 `Invalidate` calls `OnInvalidate` in headless mode without
  building; on screen it rebuilds (R4.1).
- [x] AC12 `make test`, `make lint`, `go vet` clean; README and design-system
  doc updated (R9.2).

## Risks & Assumptions

- **Assumption**: sections receive `*Shell` (a concrete type) rather than an
  interface. A program's sections are the program's code, and a concrete
  shell is simpler to use and to extend; tests get `Headless`. If a consumer
  needs to mock the shell, an interface can be extracted later without
  breaking callers that use the concrete type.
- **Assumption**: program state stays in the program. The shell offers hooks
  (`OnStart`, `OnInvalidate`, `StatusBar`) rather than a data store; the
  loaded-flag pattern is documented, not enforced.
- **Risk**: `Restart` on Windows is untested until the CI matrix or a Windows
  machine runs the gallery; the Unix path is the ported one.
- **Risk**: the banner popup and busy popup need a real canvas; headless tests
  verify state (sequence, count, text) and skip the popup.
- **Rollback**: `git revert`; nothing consumes the shell yet.

## Alternatives Considered

- Considered a `Section` interface with `OnShow`/`OnHide` lifecycle instead of
  stateless rebuild; rejected because all three apps rebuild from state and
  `regate` depends on it ("only the builder knows every reason a button is
  disabled").
- Considered a banner queue; rejected, the apps are deliberately one banner
  at a time so the most recent result is always the one on screen.
- Considered a generic `Loaded[T]` helper for the loaded-flag pattern;
  deferred until an adopter asks for it.
- Considered `fyne.io/fyne/v2/data/binding`; rejected as in spec 001.
