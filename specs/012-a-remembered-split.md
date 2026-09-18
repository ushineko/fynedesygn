# Spec 012: A split that remembers where it was dragged

> **Note**: This work has no associated issue tracker ticket. The repository
> is a personal public project without an issue tracker.

## Status: PROPOSED — awaiting review

**Depends on spec 011** for somewhere to keep the position between runs.

## Context

Every consumer has a pane under a section — a log, an output, a preview — and
the library's shape for one is a fixed height. `logpane.Options.Height` is
documented as "the pane's fixed height", and `widgets.FixedHeight` is how each
program pins it.

Fixed is wrong for the pane that matters. The section where the output is worth
reading is whichever one is failing, and a fixed pane gives it no more room; the
rest of the time it takes space the section could use. terrariabonker's port
raised this as the first thing to change after the port was usable.

Two facts make this less trivial than wrapping `container.NewVSplit`.

**Fyne's split does not report a drag.** In 2.8.1 `container.Split` is `Offset`,
`Horizontal`, `Leading`, `Trailing` and an internal flag; the divider writes
`Offset` directly and there is no callback. Nothing is notified when the user
moves it.

**A section is rebuilt often.** `Refresh`, `Invalidate`, selecting a section and
every status-driven redraw rebuild the section's content, and a split built by
the section is thrown away with it. terrariabonker demonstrated the result: with
the split owned by the section, a drag was undone by the next two-second status
poll. Its window works around this by keeping the offset on the window struct
and reading it off the live split just before that split is discarded — a
workaround that belongs here instead, once.

The shell has the same problem in its own skeleton and does not solve it either:
`NavOffset` is applied when the window is built and never read back, so the
width of the section list is not the user's to keep.

## Design

### The shell owns the position, because it outlives the section

	func (s *Shell) VSplit(key string, offset float64, top, bottom fyne.CanvasObject) fyne.CanvasObject
	func (s *Shell) HSplit(key string, offset float64, leading, trailing fyne.CanvasObject) fyne.CanvasObject

`key` names the divider within the program: `"output"`, `"preview"`. `offset` is
where it sits before anyone has moved it.

The shell keeps a map of key to position, and a reference to every live split it
has handed out. Two moments read the live splits back into the map:

- **before the content is replaced**, in `swap`, which is the last moment the
  old split exists; and
- **in the stop path**, so closing the window keeps the last drag.

The map is one section of the settings store, `fynedesygn.splits`, written
through the same debounced saver as everything else.

Positions are keyed per program, not per section. The same key used in two
sections is one divider in two places on purpose: a log pane under every section
should not jump when the section changes.

### Rules

- **A stored position outside the bar's range is ignored.** Zero or one opens
  the window with one of the two panes invisible and no obvious way back. Out of
  range means the default, not the stored value.
- **The pane's height becomes a minimum, not a size.** `logpane.Options.Height`
  is restated: it is how small the pane may be dragged. The default drops
  accordingly, because a floor that is most of the window is not a floor.
- **Nothing floats over it.** A split is structural, so the standing rule that
  nothing transient may reflow the interface is not in play here; banners and
  the progress popup keep floating over whatever the split contains.

### The navigation uses it

`buildWindow`'s `container.NewHSplit(s.nav, s.content)` becomes
`s.HSplit(navSplitKey, NavOffset, s.nav, s.content)`, under a reserved key. The
width of the section list becomes something the user can set and keep, which it
is not today, and the shell gets no second mechanism for the same job.

## Requirements

- R1 `Shell.VSplit` and `Shell.HSplit` return a `container.Split` positioned
  from the stored value for `key`, or from `offset` when there is none.
- R2 A drag survives a section rebuild: the position is read off the live split
  before the shell replaces the content.
- R3 A drag survives the process: the positions are saved through the settings
  store and read at start.
- R4 A stored position of zero or less, or one or more, is ignored in favour of
  the caller's default.
- R5 Two splits sharing a key share one position.
- R6 The navigation's own divider goes through the same call, under a reserved
  key, and is remembered.
- R7 `logpane.Options.Height` is documented and behaves as a minimum, with a
  default small enough to be one.
- R8 A shell built headless supports all of it, so the behaviour is testable
  without a display.
- R9 The gallery shows it, `docs/design-system.md` records it, and the changelog
  names it.

## Acceptance Criteria

- [ ] AC1 `VSplit` with no stored position uses the caller's default; with one,
  it uses the stored one (R1).
- [ ] AC2 Setting a split's `Offset`, then selecting another section and
  returning, gives a split at the moved position (R2).
- [ ] AC3 Setting a split's `Offset`, then running the shell's stop path, writes
  that position to the settings store; a new shell over the same store opens at
  it (R3).
- [ ] AC4 Stored positions of 0, 1, -0.5 and 4 all yield the caller's default
  (R4).
- [ ] AC5 Two sections calling `VSplit("output", …)` both open at the position
  either of them was last dragged to (R5).
- [ ] AC6 The section list's width survives a rebuild and a restart (R6).
- [ ] AC7 A `logpane` inside a split grows when the divider is dragged towards
  the section and stops at its minimum when dragged away (R7).
- [ ] AC8 Every criterion above is asserted against a headless shell (R8).
- [ ] AC9 Gallery card, design-system entry, changelog; tests, lint and vet
  clean (R9).

## Risks & Assumptions

- **Assumption**: `container.Split.Offset` is readable after a drag. It is a
  public field the divider writes, so reading it is the documented shape of the
  type rather than a trick. A canary test reads it back after a simulated drag,
  so an upstream change shows up as one failing test.
- **Risk**: the shell holds references to splits a section has discarded. They
  are dropped when the content is replaced, in the same step that reads their
  position, so a section that is rebuilt a thousand times does not accumulate a
  thousand splits. A test asserts the registry's size after repeated rebuilds.
- **Risk**: a program that passes a per-section key for what is conceptually one
  pane gets a divider that jumps. Documented; the key is the caller's choice and
  the sharing rule is stated in both directions.
- **Rollback**: additive. `git revert`; sections that do not call it keep their
  fixed panes.

## Alternatives Considered

- **A widget that wraps `container.Split` and reports drags.** Rejected: it
  still does not know when the section that owns it is discarded, so it solves
  the smaller half of the problem and adds a type.
- **Poll the offset on a timer.** Rejected: a timer to notice a number that
  changes when the user drags something, in a library whose rule is that work
  hops to the UI thread, is cost with no ceiling and no better result.
- **Keep the position in `fyne.Preferences`.** Rejected for the reasons spec 011
  gives; it is also what terrariabonker did, and the position it stores is one a
  user might reasonably want to find and reset.
