# Spec 030: the navigation takes the header

**Issue**: TBD — to be filed before the PR.

## Status: COMPLETE

## Executive Summary

A navigation along the top is drawn in the header, where the program's name
used to be, instead of in a strip of its own beneath it. The window is one row
shorter in that shape and the name is not said twice — the shell already puts
it in the window title. Down the left nothing changes. Reviewers should read
`shell/shell.go`: `headerLead`, and the one branch it removed from
`shell/nav.go`'s `body`.

## Context

Reported from jira-viewer, running with `NavIcons` along the top: the window
spent one row on `jira-viewer` and the row under it on four icon buttons, with
the title bar directly above both saying `jira-viewer 0.12.0-dev`. The
reporter's words: "the top of the app doesn't technically need the name there.
`jira-viewer` is in the title. layout could be more compact if the section
buttons moved there instead. less wasted space." And: "we have this pattern on
ALL the fynedesygn apps."

They do. The header has held the program's name since spec 002, when the
navigation was only ever a list down the left and the name sat above it as the
thing the list belonged to. Spec 013 added the top placement and gave it a
strip of its own, which left every program that chose it paying two rows of
chrome for one row of controls.

## Design

`Shell.headerLead` decides what the header opens with:

- **Along the top, and not hidden** — the navigation. The name goes; the title
  bar has it.
- **Anything else** — the name, exactly as before.

`body` then returns the content alone for `NavTop`, because the header has
already drawn the navigation. The holder is still built by `navHolderFor(true)`
and `redrawNav` still repaints it in place, so moving the selection does not
rebuild the window.

Hidden is not a special case in the header for its own sake: with no navigation
to draw there is nothing to put in the name's place, so the name comes back.

### The header is a Border, not an HBox and a spacer

The two lay the actions out identically — hard against the trailing edge — but
a Border hands what is left to what it opens with, and what it opens with may
be the navigation, which is an `HScroll`. A scroller in an `HBox` asks for
almost nothing and gets it, so the sections would have arrived squeezed into a
few pixels at the leading edge: the shape of bug that reads as broken buttons
rather than as a layout that gave them no room.

## Requirements

- R1. With the navigation along the top, the content takes the whole body.
- R2. The navigation is drawn in the header in that shape, and the program's
  name is not.
- R3. Every other shape draws the header exactly as it did.
- R4. Moving the selection repaints the navigation in place, as before.
- R5. The navigation gets the header's spare width rather than its scroller's
  minimum.

## Acceptance Criteria

- [x] AC1. `NavIcons`/`NavTop`: `body()` is the content itself, and the header
  contains the nav holder (`TestEachShapeLaysOutWhereItShould`).
- [x] AC2. `NavLabels`/`NavTop`: the header carries the section titles and not
  the program name (same test).
- [x] AC3. `NavHidden`/`NavTop`: the body is the content and the header carries
  the name again (same test).
- [x] AC4. `NavLeft` in both modes is unchanged — split for labels, strip for
  icons (same test, unchanged assertions).
- [x] AC5. The navigation is given the width the actions do not take, rather
  than collapsing to its scroller's minimum (`TestTheTopNavigationIsGivenRoom`,
  at least 300 px of a 1000 px window).
- [x] AC6. The whole suite passes and `make lint` is clean.

## Risks & Assumptions

- **A wide navigation and a full header can collide.** An `HBox` does not wrap,
  so a program with many sections in `NavLabels`, many header actions and a
  narrow window will clip at the trailing edge. That is the header's existing
  behaviour with many actions rather than a new failure, and the narrow answer
  is `NavIcons` or `NavHidden`, both of which are a keystroke away.
- No API changed. Consumers get this by upgrading; a program that never allowed
  `NavTop` sees nothing.
- Rollback is a revert: one function and one branch.

## Alternatives Considered

- **An `Options.ShowName` flag.** Rejected: it is a knob for a question the
  shape already answers, and every consumer would have to find and set it to
  get the row back.
- **Keeping the name and putting the sections beside it.** Rejected: it saves
  the row but keeps the duplication, and the name pushes the sections away from
  the leading edge where they are easiest to hit.
