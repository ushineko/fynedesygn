# Spec 065: a pane that does not grow with its log

First numbered 064, which `064-links-that-do-what-they-say.md` had taken
while this was in flight.

**Issue**: [#190](https://github.com/ushineko/fynedesygn/issues/190)

## Status: COMPLETE

## Executive Summary

The log pane's line counter sat in its header's row of controls, so its
minimum width followed its text. By the end of a long job it read
"1000 line(s), 19000 older dropped", and the header, the pane and any
fixed-size window around it grew. The counter now takes the header's spare
width, aligned to the controls, and truncates when there is none. Reviewers
should look at `Pane.Widget` in `logpane/pane.go`.

## Context

fynstall installed 20,000 files through the wizard (spec 061), logging one
path per file, and the fixed-size window grew while the job ran, from 820 px
to 969 px and then 977 px, cutting off widgets. "Nothing transient may move
the interface" (docs/design-system.md) is the rule it broke.

A probe in a real window (headless tests lay out no list rows) sampled the
window and its content's minimum size every 50 ms. The minimum width rose
in steps during the job: once when the log reached its limit and the
counter gained "older dropped", and again as the numbers grew.

The first reading blamed the list's rows: a row wrapped to the pane's width
looked like it could make the pane ask for more than it had. A test under
the test driver showed the list's minimum width stays at its template's
(32 px) with 240 wrapped rows in view, so the rows were wide but did not
set the pane's width. The counter did.

## Requirements

- R1 The pane's minimum width does not depend on its log: not on the
  lines, and not on the counter's text.
- R2 The counter stays readable: it takes the header's width between the
  title and the controls, aligned to the controls, and truncates only when
  there is no room.

## Acceptance Criteria

- [x] R1 A pane beside a 220 px column, in a 600 px window from the test
      driver, keeps its minimum width after more lines than the model keeps,
      when the counter says "older dropped", and the window never has to
      grow (`TestALongLineNeverWidensThePane`). Against `main`'s
      `pane.go` the test fails: the minimum goes from 288 px to 442 px.
- [x] R1 In fynstall's wizard, in a real window, installing 20,000
      long-path files: the content's minimum width stays at 745 px for the
      whole job and the window stays at 820 px. Before the fix, the same
      probe showed 807 px, 961 px and 969 px, and a window forced to 977 px.
- [x] R2 The counter shows its full text when the header has room (the
      existing logpane tests read it).

## Risks & Assumptions

- **A narrower header.** Where the header is narrow, the counter now shows
  an ellipsis rather than pushing the pane wider. That is the trade the
  design rule asks for.
- **Rollback.** Revert the commit; the counter goes back to the controls.

## Alternatives Considered

- Considered a minimum width of zero on the list's wrapper, against the
  row-wrapping explanation. Rejected once a test showed the list does not
  take its width from its rows; the change would have been a fix for a
  fault that is not there.
- Considered capping the wizard's page area at the window's width.
  Rejected: with the cause fixed it is not needed, and it would turn "the
  window grows" into "the content is cut off without a sign".

## E2E Test Plan

- Environment: KDE Plasma 6 (Wayland), this machine.
- Steps: the real-window probe above; then fynstall's installer of 20,000
  files at the desk, in fynstall spec 001 phase 4.
- Expected result: the window keeps its size for the whole install.
- Coverage: R1, R2.
