# Spec 087: a row's label can be bold and indented

**Issue**: [#258](https://github.com/ushineko/fynedesygn/issues/258)

## Status: COMPLETE

## Executive Summary

`Row.SetLabelBold` draws a row's label bold, where the theme has the face,
and `Row.SetIndent` sets it in by steps of `IndentStep` text sizes, with the
value still against the right edge. Both are additive and follow a change of
face. Reviewers should look at the label's holder in `NewRowWidth` and at
`Restyle`.

## Context

hayami's Development drawer (hayami spec 107) lists what was done lately
under its parts: a day, the repositories worked in that day, and the things
worked on in each, then the fortnight's busiest repositories. As rows they
are one flat column: a day's heading, a repository and a pull request's
title all start at one x in one weight, and a part's name reads like any line
under it. Asked for on the desk: indents, and bold headers to tell the
sections apart.

## Requirements

- R1 `Row.SetLabelBold(on)` draws the label bold, where the theme has the
  face (quirk 35); the value is untouched. Off, it is regular again.
- R2 `Row.SetIndent(levels)` sets the label in by `levels` steps of
  `IndentStep` (one text size); the value stays against the right edge.
  Zero or less is no indent.
- R3 Both follow a change of face: the bold face is the theme's, and a step
  is a text size.
- R4 Additive: a row that asks for neither is drawn as before.

## Acceptance Criteria

- [x] `make test`, `make lint`.
- [x] Tests: a label bold in a theme with the face, its value regular, and
      regular again when turned off (`TestALabelCanBeBold`); an indent of two
      moves the label two steps and leaves the value's right edge, and none
      puts it back (`TestAnIndentMovesTheLabelAndNotTheValue`); a face twice
      the size indents twice as far (`TestAnIndentFollowsTheFace`).
- [x] Falsified: bold ignored, the indent ignored, and the indent in pixels
      instead of text sizes each fail a test.
- [x] `docs/glance.md` says how to use them.

## Risks & Assumptions

- The label now stands in a holder with a left margin, zero unless an
  indent is asked for; a row's size and every existing test are unchanged.
- Rollback is reverting the release.

## E2E Test Plan

hayami spec 107's Development drawer, on the desk: part names and days in
bold, repositories and things indented under them, captured in the real
window.
