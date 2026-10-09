# Spec 063: a progress page that counts

**Issue**: [#185](https://github.com/ushineko/fynedesygn/issues/185)

## Status: COMPLETE

## Executive Summary

`ProgressPage.WithBar` adds a determinate bar, a status line and the current
item to the wizard's progress page, for jobs of thousands of files.
`Reporter.Progress` is safe to call once per file; the page draws the latest
values every 100 ms and once more at the end, so 10,000 reports cost a few
draws. Reviewers should look first at `wizard/progressbar.go`: `set`, `draw`
and `pump`.

## Context

The wizard's progress page (spec 061, R4) shows a step list beside a log.
That says which step a job is in, and what it did, but not how far through
it is. fynstall installs payloads of thousands of files: a program that
bundles a Python runtime is one. Installers such as NSIS show a determinate
bar, a count of files and bytes, and the file being written, and a person
waiting on a long install looks for exactly that.

A job that reports once per file reports thousands of times in a few
seconds. Each report must not cost a hop to the UI thread, so the page
keeps the latest values and draws them on a timer, as `logpane` does with
its pump.

## Requirements

- R1 `(*ProgressPage).WithBar() *ProgressPage` adds a region under the step
  list and the log: a determinate bar, a status line and an item line. It is
  opt-in, because a job that reports no counts would show an empty bar. When
  it is enabled, the region is there from the start, so nothing moves when
  counting begins (spec 061, R3).
- R2 `(*Reporter).Progress(fraction float64, status, item string)` sets the
  bar (clamped to 0..1), the status line (such as
  "2,914 of 5,603 files · 61 MB of 118 MB") and the item line (such as the
  file being written). It is safe to call from the job's goroutine, as
  often as once per file.
- R3 With a window, the page stores the latest values and draws them at
  most every `ProgressInterval` (100 ms), and once more when the job ends,
  so the final value is always shown. Headless, it draws at once.
- R4 The item line is shortened in the middle to `ItemRunes` runes, so the
  start and the file name of a long path both stay visible, and a long path
  never widens the window.
- R5 `docs/wizard.md` describes it; the example and the gallery's sample
  wizard use it; the README changelog says so.

## Acceptance Criteria

- [x] R1 Without `WithBar` the page has no bar. With it, the region has the
      same height before the first `Progress` and after (headless).
- [x] R2 `Progress` sets the bar, the status and the item, and clamps the
      fraction (headless).
- [x] R3 A job that calls `Progress` 10,000 times in a tight loop, under a
      real window from the test driver, causes few redraws, and the last
      value is shown when it ends.
- [x] R4 A long path keeps its start and its file name, with an ellipsis
      between, within `ItemRunes` runes.
- [x] R5 At the desk: `examples/installer-wizard` shows the bar counting
      through the install, with no movement of the window.

## Risks & Assumptions

- **Rune length is not width.** R4 counts runes, not pixels, so a line of
  wide characters can still be clipped at the end by the label. Paths are
  mostly narrow ASCII, and the label truncates rather than wrapping, so
  nothing moves either way.
- **Rollback.** Additive: a method on `ProgressPage` and one on `Reporter`.

## E2E Test Plan

- Environment: KDE Plasma 6 (Wayland), this machine.
- Steps: the R5 desk check, then fynstall's installer with a payload of
  thousands of files, in fynstall spec 001 phase 4.
- Expected result: the bar and counts move smoothly to the end, the item
  line shows the file being written, and nothing else in the window moves.
- Coverage: R1–R5.

### Results (2026-10-08)

On CachyOS with KDE Plasma 6 on Wayland. `make test` (with `-race`, three
runs of the throttle test), `make lint` and `govulncheck` pass.

- `TestThousandsOfReportsCostAFewDraws`: 10,000 reports under a real window
  from the test driver drew fewer than 100 times, and the last value was
  shown. A mutation that drew on every report failed it with 10,000 draws.
  The test driver runs `fyne.Do` on the calling goroutine, so the page now
  draws through the wizard's own `do`, which the test serialises with a
  mutex standing in for the UI thread.
- `examples/installer-wizard` with the bar: the user ran it through the
  install and reports that the bar, the counts and the item looked right.
