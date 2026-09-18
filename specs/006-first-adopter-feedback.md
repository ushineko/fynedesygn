# Spec 006: First adopter feedback

> **Note**: This work has no associated issue tracker ticket. The repository
> is a personal public project without an issue tracker.

## Status: COMPLETE

## Executive Summary

Three hooks on `shell.Options` that clockwork-orange's adoption (its spec
012) had to work around: `OnCreate` (hold the shell before any builder
runs), `Theme` (build the theme from the appearance plus program state such
as a console font), `OnTypedKey` (keyboard handling beside the shell's F5).
Released as v0.1.1.

## Context

The adoption in clockwork-orange recorded four gaps. Three are the shell
assuming it is the only party with an opinion: it built the first section
and ran `OnStart` before `New` returned, so a program could not hold the
shell in a field first; it applied `Appearance.Theme()` itself, so a program
with a console font in its own configuration had to apply a second theme
over it; and it owned the canvas's typed-key handler for F5. The fourth
(`Appearance.Save` writing `appearance.mono` as the default) is harmless and
left as is: a saved default is a stable file, and a program that does not
use the key ignores it.

## Requirements

- R1 `Options.OnCreate func(*Shell)`: called by `New` and `Headless` after
  the shell exists and before any section is built, the status bar composed
  or `OnStart` called.
- R2 `Options.Theme func(fdtheme.Appearance) fyne.Theme`: when set, used
  wherever the shell applied `Appearance.Theme()` (`New`, `Headless`,
  `SetAppearance`). `SetAppearance` with an unchanged appearance re-applies
  it, which is how a program reacts to a change in its own state.
- R3 `Options.OnTypedKey func(*fyne.KeyEvent)`: called for every typed key
  the shell did not take (it takes F5).

## Acceptance Criteria

- [x] AC1 A builder sees the shell stored by `OnCreate`; `OnCreate` runs
  before the first `Build` (R1).
- [x] AC2 The theme on the app after `New`/`onScreen` and after
  `SetAppearance` is the hook's (R2).
- [x] AC3 F5 invalidates and is not passed on; other keys reach the hook
  (R3).
- [x] AC4 Tests, lint and vet clean; `docs/design-system.md` names the hooks.

## Risks & Assumptions

- Additive change; no existing caller changes. Rollback: `git revert`.
