# Spec 009: Telling a section it was arrived at

> **Note**: This work has no associated issue tracker ticket. The repository
> is a personal public project without an issue tracker.

## Status: COMPLETE

## Executive Summary

`shell.Arriver` — `Arrive()`, registered on a `FuncSection` with `OnArrive` —
is the shell telling a section that the navigation reached it, as opposed to
rebuilding it where it stands. `swap` already knew: it navigates with
`keepScroll` false and rebuilds with it true. A section that refetches on
arrival needs the difference, because refetching in the builder loops — the
fetch finishing rebuilds the section that started it. Found by angou's
adoption, which had worked around it by remembering the last section title.
Additive, and the mirror of `Detacher`. Released as v0.1.5.

## Context

The shell builds a section for two different reasons and tells it neither.
Navigating to a section builds it; so does every rebuild the shell does while
it is on screen — an operation starting or finishing, a load landing, F5,
`Refresh`, `Invalidate`. `Section.Build` is called the same way in both cases.

That is enough for a section whose builder is a pure function of state, which
is what the first two adopters had. The third does not. angou reloads what it
has loaded when the navigation *arrives* at a section and reopening the store
would go through silently, because the store is a plain directory that the CLI
writes to and that a sync service writes to — a file encrypted from a terminal
should not stay invisible until the window is restarted. Doing that on every
build instead is a loop: dropping the loaded flags starts the loads, and a load
finishing rebuilds the section, which drops the flags again.

angou's adoption (its spec 003) worked around it by remembering the last
section title and comparing, which is four lines and works, but every program
that reloads on arrival would write those four lines, and the shell already
knows the answer exactly: it navigates in `swap(false)` and rebuilds in
`swap(true)`.

`Detacher` is the same shape of thing from the other end — the shell telling a
section it is about to be replaced — so this is the hook that was missing
beside it.

## Requirements

- R1 `shell.Arriver` is an optional interface, `Arrive()`, implemented by a
  section that wants to know when the navigation reaches it.
- R2 The shell calls `Arrive` when it switches the content pane to a section:
  after the outgoing sections are detached, before the incoming one is built.
  That is navigation — the list, `Select`, and the section the window opens on.
- R3 The shell does not call it when the section already on screen is rebuilt:
  `Refresh`, `Rebuild`, `Invalidate`, and the rebuilds an operation's start and
  end cause. This is the whole distinction the hook exists to draw.
- R4 `FuncSection.OnArrive(fn)` registers it and returns the section, as
  `OnDetach` does.
- R5 Off screen there is no content pane and nothing is built, so nothing
  arrives either: a headless test that calls `Build` directly gets the state it
  set up.
- R6 The gallery demonstrates it, `docs/design-system.md` records it, and the
  changelog names the addition.

## Acceptance Criteria

- [x] AC1 Navigating to a section calls `Arrive` once, before `Build` and after
  the outgoing section's `Detach` (R1, R2).
- [x] AC2 `Refresh`, `Rebuild` and `Invalidate` build the current section
  without calling `Arrive` (R3).
- [x] AC3 The section the window opens on arrives once (R2).
- [x] AC4 Navigating away and back arrives again; `Select` to the section
  already current does not build it twice or arrive twice (R2).
- [x] AC5 A headless shell calls neither (R5).
- [x] AC6 Gallery card, design-system entry, changelog; tests, lint and vet
  clean (R6).

Verified in `shell/shell_test.go` by
`TestArriveIsNavigationOnlyAndRebuildsDoNotCount` (AC1–AC4, asserting the
exact order of detach, arrive and build, and that three rebuilds in a row
arrive at nothing), `TestArriveDoesNotFireWithoutAWindow` (AC5) and
`TestASectionWithoutTheHookIsUnaffected` (the interface is optional). AC6:
the "shell.Arriver" card in `cmd/fynedesygn-gallery/sections.go` counts
arrivals against builds so the two can be seen diverging, "Sections" in
`docs/design-system.md`, 0.1.5 in the README changelog; `make test` (race),
`make lint` (0 issues) and `go vet ./...` clean, and `govulncheck -mode
binary` on an unstripped gallery reports nothing.

## Risks & Assumptions

- **Assumption**: `swap(false)` is navigation and `swap(true)` is a rebuild in
  place, with no third caller. There are exactly two call sites.
- **Risk**: a section that reloads on arrival and whose reload rebuilds the
  section will still loop if it implements `Arrive` wrongly — the hook removes
  the need to distinguish, not the ability to get it wrong. The doc comment
  says what it is for.
- **Rollback**: additive. `git revert`; `Section` is unchanged and `Arriver` is
  a new optional interface, so every existing section keeps compiling.
