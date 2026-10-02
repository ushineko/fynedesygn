# Spec 054: a scroller that survives a rebuild

**Issue**: [#164](https://github.com/ushineko/fynedesygn/issues/164)

## Status: COMPLETE

## Context

A section with affixed controls draws
`Border(nil, actions, nil, nil, container.NewVScroll(body))`. The shell keeps
its content pane's offset on a rebuild in place, but the section's inner
scroller is a new object every build. Every operation rebuilds the section
(`OK` invalidates, and the busy tail regates). Ticking a checkbox halfway down
nmsbonker's Tweaks page therefore threw the page back to the top under the
pointer.

## Requirements

- R1 `(*Shell).VScroll(name string, content fyne.CanvasObject) *container.Scroll`
  returns a vertical scroller. When a live scroller of that name exists, its
  offset is carried over.
- R2 The offset is assigned, not scrolled to. `ScrollToOffset` before the
  first layout clamps against a zero size and loses the value.
- R3 Navigation (the list, `Select`, a `Tabs` strip) forgets every scroller,
  so a section arrived at starts at the top, as the content pane does.
- R4 The offset is not stored across runs.

## Acceptance Criteria

- [x] R1 A section rebuilt by `Refresh` or `Invalidate` keeps its scroller at
      the offset it was scrolled to (headless test, laid out in a test
      window).
- [x] R3 Leaving the section and returning starts at 0.
- [x] R1 Two names keep separate offsets.
- [x] Changelog and design-system doc updated.

## Risks & Assumptions

Additive. A section that keeps using `container.NewVScroll` behaves as before.
Rollback: revert.

## E2E Test Plan

nmsbonker's Tweaks pages use `VScroll`. Scroll down, tick a checkbox, and the
page stays where it was. Photograph it.
