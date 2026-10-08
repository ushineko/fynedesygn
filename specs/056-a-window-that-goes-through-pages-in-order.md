# Spec 056: a window that goes through pages in order

**Issue**: [#172](https://github.com/ushineko/fynedesygn/issues/172)

## Status: INCOMPLETE

## Executive Summary

Populated before the PR is opened.

## Context

fynstall ([ushineko/fynstall#1](https://github.com/ushineko/fynstall/issues/1))
builds installers for the programs in the "Used by" table. Its GUI mode is a
wizard: a fixed sequence of pages with Back, Next and Cancel, where the user
cannot continue until the current page is complete, and where the last pages
run a job and report its result.

Neither archetype in this module has that shape. `shell` lets the user go to
any section in any order, and `glance` is a panel that is read and not
operated. A wizard is a third archetype. It has an order, a current page and
a point after which Back no longer has a meaning.

The parts it needs already exist. A form is `forms.Form`, a job is the
`steps.List` beside a `logpane.Pane` that `examples/job-runner` shows, a
document is `markdown`, a folder choice is `dialogs.WithBrowse` and a
confirmation is `dialogs.ConfirmDestructive`. This spec adds the window that
holds them in order, and a small set of standard pages built from them.

fynstall is the first consumer. The API is written so that a non-installer
flow (a first-run setup, an import) can use it too, so nothing in the package
names files, directories or installs except the standard `Directory` page.

## Requirements

- R1 Package `wizard`. `wizard.Page` is an interface with `Title() string`
  and `Build(w *Wizard) fyne.CanvasObject`. Optional behaviour uses separate
  interfaces, in the way `shell.Arriver` does:
  - `Validator`: `Valid() bool`. When the page state changes, the page calls
    `w.Revalidate()`, and Next is enabled only while `Valid()` is true.
  - `Enterer`: `Enter(w *Wizard)`, called each time the page becomes current.
  - `Leaver`: `Leave(w *Wizard, forward bool) error`. A non-nil error keeps the
    current page and shows the error as a banner.
  - `Skipper`: `Skip() bool`. Back and Next pass over a page that returns
    true, and the page list draws it as skipped.
- R2 Window skeleton. A page list is on the left, using `steps`
  vocabulary: done, current, pending and skipped. The page title and
  content are in the centre, inside the one scroller. A button row is at the
  bottom: Back, then Next, then Cancel, aligned to the trailing edge. The
  window has a fixed default size (proposed 820 x 560) and the program can
  change it.
- R3 Nothing temporary moves the interface. The Next button can show "Next",
  "Install", "Finish" or a label the page gives. It has the width of the
  widest label it can show, so the row never changes width. A button that the
  user cannot use is disabled and never hidden.
- R4 A `Progress` page runs `func(ctx context.Context, r Reporter) error`
  through the same cancellable path that `shell.PerformCancellable` uses.
  `Reporter` advances the `steps.List` and writes to the `logpane.Pane`.
  While the job runs, Back is disabled and Next is disabled. Cancel asks for
  confirmation, then cancels the context and waits for the job to return.
  After the job, Next goes to the following page. On failure the page
  shows a `bad` banner, the log stays visible, and Cancel changes to
  Close.
- R5 After the user starts a page marked `PointOfNoReturn`, the user cannot
  go back to an earlier page with Back. This is normally the progress page.
- R6 Cancel before the point of no return closes the window, after
  confirmation if any page reports itself as changed. Cancel after the point
  of no return is available only while a job runs, as R4 describes. Closing
  the window from its title bar has the same effect as Cancel.
- R7 Standard pages, each built only from components that exist in the
  module:
  - `Welcome(title, body string)`: Markdown body.
  - `Licence(body string)`: Markdown body and an "I accept" check. The page
    is valid only while the check is set.
  - `Directory(label, initial string, check func(string) error)`: an entry
    with Browse (`dialogs.WithBrowse`). The page is valid while `check`
    returns nil, and the error shows below the entry as status text.
  - `Form(f *forms.Form)`: options. The page is valid while every field is
    valid.
  - `Summary(facts func() []Fact)`: a read-only fact table that is
    rebuilt each time the page becomes current.
  - `Progress(...)`: as R4.
  - `Finish(body string, checks ...*widget.Check)`: Markdown and optional
    checks such as "Launch now". `w.Result()` returns their state after the
    window closes.
- R8 `wizard.Run(o Options) Result` blocks until the window closes, and
  `wizard.Headless(a fyne.App, o Options) *Wizard` is for tests. `Result`
  reports Finished, Cancelled or Failed, and the state of the checks on the
  finish page.
- R9 The package follows the threading rules in `docs/design-system.md`.
  Page hooks run on the UI thread. `Reporter` methods are safe to call from
  the job goroutine.
- R10 Documentation, gallery and example. `docs/wizard.md` holds the rules
  for this archetype, and `docs/design-system.md` links to it from the window
  skeleton. The gallery gets a Wizard section that opens a sample wizard. The
  example is `examples/installer-wizard`, with a fake job and a headless test.
  The README package table, examples table, "Used by" row for fynstall and
  changelog are updated in the same commit.

## Acceptance Criteria

- [ ] R1 With the headless driver: Next is disabled while a `Validator` page
      is invalid, and enabled after `Revalidate()` with valid state.
      (`fynetest`)
- [ ] R1 `Leaver` returning an error keeps the current page and shows a
      banner. `Skipper` pages are passed over in both directions. (headless)
- [ ] R3 The button row has the same width on every page of the example,
      measured in a headless test across all pages.
- [ ] R4 A job that succeeds advances every step to done and enables Next.
      A job that fails marks the current step bad, shows the banner and
      changes Cancel to Close. A cancelled job receives `ctx.Done()` and the
      wizard waits for it to return. (headless, fake job)
- [ ] R5 After the progress page starts, Back is disabled on every later
      page. (headless)
- [ ] R6 Closing the window during a job asks for confirmation and does not
      close until the job returns. (headless, with a close intercept)
- [ ] R8 `Result` reports Finished, Cancelled and Failed in the three
      matching paths. (headless)
- [ ] R10 At the desk: `go run ./examples/installer-wizard` in Breeze Dark
      and Breeze Light on KDE Plasma 6. Go forward, go back, refuse the
      licence, choose a folder, run, cancel during the run, and finish. The
      results are recorded in the PR with screenshots.
- [ ] R10 The gallery Wizard section, `docs/wizard.md`, README rows and
      changelog are present. `readme_test.go` passes with fynstall in the
      "Used by" table.

## Risks & Assumptions

- **API shape before a second consumer.** fynstall is the only consumer when
  this lands. The interfaces are kept small (R1) so that a second consumer can
  add an optional interface without a change to existing pages.
- **Fixed window size.** Fyne's `SetFixedSize` is a hint that some window
  managers ignore. A resized window must still lay out correctly. The content
  is in a scroller for that reason.
- **Close intercept.** R6 depends on `Window.SetCloseIntercept`, which works
  on GLFW backends. The test driver is checked for the same behaviour.
- **Rollback.** The package is new and additive. Reverting the commit
  removes it, and no existing package changes behaviour.

## E2E Test Plan

- Environment: KDE Plasma 6 (Wayland), this machine.
- Steps: the R10 desk check above, then the fynstall phase that consumes
  `wizard` (fynstall `specs/001-installer-prototype.md`, phase 4) runs a real install through it.
- Expected result: every R10 step behaves as written, with no change in the
  button row width and no layout movement during a job.
- Coverage: R1, R3, R4, R5, R6 and R10.
