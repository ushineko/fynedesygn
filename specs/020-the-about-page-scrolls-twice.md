# Spec 020: the About page scrolls twice

**Issue**: [#19](https://github.com/ushineko/fynedesygn/issues/19)

## Status: COMPLETE

## Context

`AboutSection` ends with `container.NewVScroll(container.NewVBox(items...))`.
The shell already puts every section's `Build` output inside `Shell.Scroller()`,
so an About page has been a scroller inside a scroller since it was written --
the "one scroll per section" rule broken by the module that states it.

It looked harmless because it behaves almost correctly: Fyne gives the wheel to
the innermost scrollable under the pointer, so the inner scroller moves, the
outer one never does, and the page scrolls.

`About.Extra` is where it stops being harmless. Its own documentation says the
field "receives the shell so it can follow the content scroller" -- for the
embedded README that is the whole point of the field. A `markdown.Pane` that
calls `Follow(s.Scroller())` from inside `Extra` attaches to the outer
scroller, which never moves, while the pane itself lives in the inner one. The
pane renders the blocks near the top and nothing else, and no amount of
scrolling tells it otherwise: an About page that is one screen of README
followed by blank space the height of the rest of it.

Found in hotaru, embedding a 448-line README. The pane was right, the document
was right, and the composition was wrong in a way neither of them could see.

### Why the fix is removal and not a flag

A `Section` is only ever built by a shell, and a shell always scrolls what a
section builds. There is no caller for which the inner scroller is the only
one -- so it is not a choice with two answers, it is a line that should never
have been there.

## Requirements

**R1. `AboutSection` returns content, not a scroller.** The shell's content
scroller governs the page, as it does for every other section.

**R2. A canary says so.** `fynetest.ScrollableIn` over what `AboutSection`
builds, so the day somebody wraps it again the test names the rule.

**R3. Nothing else about the page changes.** Same order, same widgets, same
`Extra`.

## Acceptance Criteria

- [x] AC1. `AboutSection(...).Build(s)` contains no scrollable.
- [x] AC2. The existing About tests pass unchanged: the page still shows the
      name, version, blurb, link, notes and facts it is given.
- [x] AC3. What `Extra` builds is inside the scroller `Extra` is handed, which
      is what makes `Follow(s.Scroller())` mean anything; asserted on a shell
      with a window rather than by reading the code.

## Risks & Assumptions

- **Consumers see one behaviour change**: an About page now scrolls as the
  section scrolls. Since the inner scroller expanded to fill the outer one,
  the page looked and behaved the same; what changes is which scrollbar moves.
- **Rollback** is a revert of three lines.

## Alternatives Considered

Considered leaving the scroller and giving `About` a field to hand the inner
scroller back to `Extra`; rejected because it makes every consumer thread a
scroller through a callback to work around a wrapper none of them asked for.
