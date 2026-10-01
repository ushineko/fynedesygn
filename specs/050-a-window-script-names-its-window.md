# Spec 050: a window script names its window

**Issue**: [#154](https://github.com/ushineko/fynedesygn/issues/154)

## Status: COMPLETE

## Context

The four window scripts in `glance/kwin` matched `w.resourceClass` and
nothing else. On Wayland every window of an app carries the app's ID, so
hayami's preferences window matched too: a drag of it was reported by the
watch and saved as the panel's position (ushineko/hayami#107), and
`PlaceScript`'s "first window of the class" (spec 049) is the wrong window
when the program is started onto a preferences page. Measured on the desk
on 2026-09-30 with `hayami --preferences=about`: the compositor showed
"hayami preferences 0.8.2" 140 ms before "hayami".

A program knows its windows' titles; it set them. The caption is the
identity the class lacks.

## Requirements

- R1 `kwin.Target{Class, Caption}` names a window. A script matches a
  window whose resource class is `Class` and, when `Caption` is not empty,
  whose caption is `Caption`. An empty caption is the old matching.
- R2 `PositionScript`, `PlaceScript`, `ReportGeometryScript` and
  `WatchGeometryScript` take a `Target` in place of the app ID, and share
  one `matches(w)` so they agree on what a window is. Breaking; hayami is
  the only consumer and moves with it.
- R3 `PlaceScript` still places once; its guard is now against a later
  match or a second copy of the program.
- R4 A caption, like every other string, cannot escape its literal.
- R5 `docs/glance.md` says a window script takes a target and why.

## Acceptance Criteria

- [x] R1, R2 (`TestATargetWithACaptionIsOneWindowOfItsClass`,
      `TestTheWatchScriptMatchesTheTarget`, and the four scripts' existing
      tests updated to the target)
- [x] R3 (`TestThePlaceScriptPlacesOnlyTheFirstWindow`)
- [x] R4 (`TestAPositionScriptCannotBeEscapedByItsStrings`, with a caption
      in the watch and position scripts)
- [x] R5 (`docs/glance.md`, Desktop integration)
- [ ] On the desk, in hayami: started onto a preferences page, the panel is
      placed and the preferences window is not; a drag of the preferences
      window changes nothing in the settings (hayami spec 030)

## Risks & Assumptions

- A caption is an exact match. KWin appends " <2>" to a duplicate caption
  on X11 only; on Wayland the caption is the toplevel's title as set, which
  is what the program passed.
- The caption must be set before the window is mapped, or the first
  `windowAdded` would not match. A toolkit sets the title at creation;
  checked on the desk by `kdotool getwindowname` reporting the title the
  first time the window is listed.
- Breaking for the four signatures, in a v0.x module with one consumer.
