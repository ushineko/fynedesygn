# Spec 035: an indicator that comes and goes

**Issue**: [#63](https://github.com/ushineko/fynedesygn/issues/63)

## Status: COMPLETE

## Executive Summary

`glance.Transient` shows a glance window for a hold after each change and
hides it on its own; `Options.Secondary` makes a glance window that is not
the program's master. `kwin.Rule` gains `Title`, the three skip flags and
`NoFocus`, with `LookupTitled` and `RemoveTitled` keyed on the app ID and
title pair. Two measurements on Plasma 6 correct `docs/glance.md`: a splash
window takes the focus when shown (quirk 36), and a forced `position` rule
does place a window. Reviewers should start with `glance/transient.go` and
the ordering comment in `Show`, then the titled-rule test.

## Context

ototo, the port of audio-source-switcher, needs a volume indicator: a small
frameless panel that appears for 1.5 s when the volume changes, on the screen
the pointer is on, and never takes the focus. That is a glance window shown
for a moment, and three things were missing.

The rule package matched on the app ID only. An indicator is a second window
of a program whose main window has the same app ID, so a rule on the app ID
would strip the main window's titlebar. The indicator must also stay out of
the taskbar, the switcher and the pager, and must not accept the focus.

Nothing showed a glance window for a while and hid it again with the hold
restarted on each show.

`docs/glance.md` said a KWin `position` rule moves a window to a screen's
origin on Wayland. ototo's spec 001 R8.4 measured otherwise on 2026-09-24:
`position=500,1300` forced placed a Fyne splash window at 500,1300, and
`acceptfocus=false` forced stopped it taking the focus, which it does with no
rule.

## Requirements

- R1 `kwin.Rule` matches on `Title` as well as `AppID` when `Title` is set,
  exactly, and a titled rule is a different rule from the untitled one for
  the same app ID: installing one never replaces the other. `LookupTitled`
  and `RemoveTitled` read and remove by the pair; `Lookup` and `Remove` keep
  their meaning for the untitled rule.
- R2 `kwin.Rule` writes `skiptaskbar`, `skipswitcher` and `skippager`, forced,
  and `acceptfocus=false`, forced, from four booleans; off means the keys are
  absent, as the other flags.
- R3 `glance.Transient`: `Show` shows the window and starts the hold; a
  `Show` during the hold restarts it and does not run `OnShow` again; the
  timer hides the window on the UI thread; `Hide` hides now and stops the
  timer; a timer that lost a race to a later `Show` does nothing.
- R4 `glance.Options.Secondary` leaves the window out of the master role.
- R5 `docs/glance.md` records the two measurements and describes the
  indicator shape; `docs/fyne-quirks.md` gains quirk 36.
- R6 The package still writes no `position`: a position in a rule is fixed to
  one screen, and the program moves the window at each show.

## Acceptance Criteria

- [x] R1: `TestATitledRuleIsADifferentRuleFromTheUntitledOne`.
- [x] R2: the same test and `TestTurningOffNoFocusRemovesTheKeys`.
- [x] R3: `TestAnIndicatorGoesAwayOnItsOwn`, `TestChangesCloseTogetherKeepTheIndicatorUp`, `TestHideStopsTheTimer`, all under the race detector.
- [x] R4: `TestASecondaryWindowIsNotTheMaster`.
- [x] R5: the documents read; the style canaries pass.
- [x] R6: `staleKeys` unchanged; its comment says why.

## Risks & Assumptions

- Under the Fyne test driver `fyne.Do` runs inline on the timer's goroutine,
  so the timer's hide and the next show touch the window from two
  goroutines. `Transient` arms the timer after the window's own `Show` and
  serialises the window calls on a mutex; the tests run five times under the
  race detector.
- The focus measurement is one compositor and one version. A quirk with no
  canary is a quirk that is re-measured when someone asks.
- Rollback: revert the commit. No consumer uses the new identifiers yet.

## Alternatives Considered

- A `Position` field on the rule, since the rule works; rejected because a
  position in a rule is fixed to one screen, and the indicator wants the
  pointer's screen.
- Helpers for the KWin Scripting API in the library; deferred. The program
  that needs the move makes the calls with the D-Bus client it already has,
  and the library keeps to Fyne as its only dependency.
