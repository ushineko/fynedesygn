# Spec 013: The navigation's shape

> **Note**: This work has no associated issue tracker ticket. The repository
> is a personal public project without an issue tracker.

## Status: PROPOSED — awaiting review

**Depends on spec 011** for somewhere to keep the choice, and on **spec 012**
for the divider beside it.

## Context

The shell's navigation is one shape and has been since spec 002: a
`widget.List` of icon-and-label rows, in the leading half of an `HSplit` at a
fixed `NavOffset` of 0.16, on the left, always visible.

That is a reasonable default and the only option. Nothing in the library can
hide it, narrow it to its icons, or put it along the top. The three consumers
each have a section that wants the horizontal room — an inventory grid, a
document, a diagram — and a screenshot of any of them spends a sixth of its
width on a list of seven words.

## Design

### Two axes, named

	type NavMode uint8      // NavLabels, NavIcons, NavHidden
	type NavPlacement uint8 // NavLeft, NavTop

`NavLabels` on `NavLeft` is what the window is today and stays the default.

### A program declares what it allows

	Options.NavModes      []NavMode      // empty: NavLabels only
	Options.NavPlacements []NavPlacement // empty: NavLeft only
	Options.Nav           NavMode        // where a new installation starts
	Options.NavPlace      NavPlacement

Empty means today's window exactly: no control appears, nothing moves, and a
program that says nothing gets no new behaviour. The public API is a contract
and this is a change to the window every consumer already ships; opting in is
one line per program.

A stored choice wins over `Options.Nav` once the user has made one, and a stored
choice the program no longer allows falls back to the first allowed mode rather
than to a shape the program has withdrawn.

### The control

One stock control, in the header, shown when more than one mode or placement is
allowed: a button that opens a menu of the allowed modes and placements with the
current one ticked. One rule rather than a toggle for two options and a menu for
three — a control that changes shape according to how many choices exist is a
control the user has to re-learn per program.

`Ctrl+B` toggles between hidden and the last non-hidden mode, bound only when
`NavHidden` is allowed. It is the binding people already try, and it is the same
class of shortcut as the `F5` and `Ctrl+R` the shell already binds.

**The control is never hidden by the mode it sets.** It lives in the header,
which is always present, so `NavHidden` always has a way back. A navigation that
can be hidden with no way to bring it back is a broken window, not a compact one.

### What each shape is

| Mode | Placement | Layout |
| --- | --- | --- |
| Labels | Left | `HSplit`, position remembered (spec 012) — today's window |
| Icons | Left | A fixed-width strip of icon buttons, no divider |
| Labels | Top | A row under the header, fixed height, scrolls when it overflows |
| Icons | Top | The same row, icons only |
| Hidden | either | No navigation; the content takes the whole window |

Icons-only on the left is a strip rather than a split because a divider on
something sized to its icons has nothing to give.

**An icon carries its section's title as a hover tip** (spec 010), in every
icons-only shape. Without it, icons-only is a guessing game, and the library
already has the piece that fixes it.

**A section with no icon gets a generic one.** `Section.Icon()` may return nil
and does in places; a blank cell in an icons-only navigation is a section the
user cannot reach. The tip still names it.

### What does not change

- The selected section survives a change of shape and of placement.
- `Shell.Select`, `Options.Section`, `Current` and `Sections` are untouched.
- The header and the status bar keep their places. Only the region between them
  changes, which is the region the split already owns.
- Result banners and the progress popup keep floating over the content.

### Where the choice is kept

`fynedesygn.nav` in the settings store (spec 011), as `{mode, placement}`.

## Requirements

- R1 `NavMode` and `NavPlacement` exist with the values above and are documented.
- R2 A program that sets neither `NavModes` nor `NavPlacements` gets today's
  window: same layout, no control, no shortcut.
- R3 A program that allows more than one option gets one stock control in the
  header offering exactly the options it allows.
- R4 `Ctrl+B` toggles hidden and back, and is bound only when `NavHidden` is
  allowed.
- R5 The control is reachable in every mode, including `NavHidden`.
- R6 Each of the five shapes lays out as the table says, and the content region
  takes the space the navigation is not using.
- R7 In any icons-only shape, each icon carries its section's title as a tip,
  and a section with no icon of its own gets a generic one.
- R8 The selected section is the same section before and after a change of mode
  or placement.
- R9 The choice is stored under `fynedesygn.nav` and read at start; a stored
  choice the program no longer allows falls back to its first allowed one.
- R10 The left-and-labels shape keeps the remembered divider from spec 012.
- R11 Every shape renders headlessly, in every colour scheme, in the existing
  section-render test.
- R12 The gallery allows every option and demonstrates the control;
  `docs/design-system.md` records the shapes and the changelog names them.

## Acceptance Criteria

- [ ] AC1 A shell with no nav options has the same content tree as before this
  spec: a two-child `HSplit` with the list leading, and no control in the header
  (R2).
- [ ] AC2 A shell allowing two modes has one header control listing exactly
  those two; allowing all three and both placements lists five entries with the
  current one ticked (R3).
- [ ] AC3 `Ctrl+B` hides and restores the navigation when `NavHidden` is
  allowed, and does nothing when it is not (R4).
- [ ] AC4 With the navigation hidden, the header control is still in the tree
  and still opens (R5).
- [ ] AC5 Each of the five shapes produces the layout the table describes, with
  the content region filling the remainder (R6).
- [ ] AC6 In an icons-only shape, every section's title is reachable as a tip,
  including one whose `Icon()` returns nil (R7).
- [ ] AC7 Selecting the third section, then changing mode and placement, leaves
  the third section current and built once (R8).
- [ ] AC8 A choice made in one run is the shape at the start of the next; a
  stored mode a program has withdrawn opens at its first allowed mode (R9).
- [ ] AC9 The divider position in left-and-labels survives a mode change away
  and back (R10).
- [ ] AC10 `TestEverySectionRendersHeadlesslyInEveryScheme`, or its successor,
  covers all five shapes (R11).
- [ ] AC11 Gallery, design-system entry, changelog; tests, lint and vet clean
  (R12).

## Risks & Assumptions

- **Risk**: the horizontal placement is a layout the library has never had. A
  row that overflows must scroll rather than squeeze, and the status bar and
  header must not move when it does. AC5 covers the layout; the render test
  covers the rest.
- **Risk**: five shapes times the existing schemes is a wider render matrix than
  the suite has today. It is the cheapest place to catch a shape that draws
  wrong, and the existing test already iterates schemes.
- **Assumption**: one stock control is enough, and no consumer needs the choice
  in its own menu. `Options.Header` is still there for a program that wants a
  second way in.
- **Risk**: documentation screenshots show the left-and-labels window and would
  become one shape among five. They stay as they are; the gallery is where the
  others are demonstrated.
- **Rollback**: additive by construction — R2 is the guarantee that a consumer
  that does nothing sees nothing. `git revert` returns the fixed layout.

## Alternatives Considered

- **Make every shape available to every program with no declaration.**
  Rejected: it changes the window of four shipped programs without their
  authors choosing it, and one of them captures documentation screenshots from
  that window.
- **A collapse button on the navigation itself.** Rejected: it disappears with
  the thing it collapses, which is the one state that needs it.
- **`container.AppTabs` for the horizontal placement.** Rejected: `AppTabs` owns
  the content it switches between, and the shell's content region is built by
  `swap` from a section. Two owners of one region is the bug that would follow.
