# Spec 028: a name that is not there

**Issue**: [#44](https://github.com/ushineko/fynedesygn/issues/44)

## Status: COMPLETE

## Context

`Shell.Select` has always been documented as "Unknown titles are ignored".
It navigated to the first section instead, because the lookup behind it
answered `0` for *not found* as well as for *the first one*:

	func (s *Shell) index(title string) int {
		if title == "" {
			return 0
		}
		for i, sec := range s.opts.Sections {
			if strings.EqualFold(sec.Title(), title) {
				return i
			}
		}
		return 0        // and here
	}

So a program that rearranged its sections -- folded one into a group, reworded
a title -- kept a `Select` call that still compiled, still ran, and quietly
sent the window to the front page.

### How it was found

hotaru's window registers a drop handler that takes a picture dragged onto it:

	s.Window.SetOnDropped(func(_ fyne.Position, uris []fyne.URI) {
		s.Select("Pictures")
		a.pictures.Dropped(s, uris)
	})

Pictures stopped being a top-level section when that program grouped it with
two others under Create. The drop still worked -- the picture was converted
and kept -- and the window jumped to the service page every time, which is
what got reported: "when you add a picture the entire gui jumps back to the
first section".

Nothing in either program was wrong except this.

### Two questions, two answers

`Options.Section` and `Select` both take a title, and they are not the same
question.

- **`Options.Section`** is a program saying where to open. A name that is not
  there should still open *somewhere*, so it falls back to the first section,
  as its own documentation says.
- **`Select`** is navigation, during a run, usually in response to something
  the user did. A name that is not there is a bug in the caller, and the least
  harmful thing to do about it is nothing: the window stays where the user
  left it.

The lookup returns -1 now and each caller decides, which is the distinction
that was missing rather than a new rule.

## Requirements

**R1. `Select` with a title no section has does nothing** -- no navigation, no
rebuild.

**R2. `Options.Section` with a title no section has opens the first.**

**R3. Both hold headless**, where there is no list to hold a selection and the
wrong answer is silent.

## Acceptance Criteria

- [x] AC1. Selecting a name that is not there leaves `Current` where it was,
      on screen and headless.
- [x] AC2. It rebuilds nothing: no detach, no build, no arrive.
- [x] AC3. `Options.Section` naming nothing opens the first section.
- [x] AC4. Verified in hotaru: a picture dropped on the window is kept and
      the navigation stays where it was.

## Risks & Assumptions

- **A caller relying on the old behaviour** -- `Select("")` to go home -- is
  unaffected: the empty title still means the first section, which is the one
  case where 0 was the right answer.
- **Silence is the point and the cost.** A `Select` for a section that has
  been renamed now does nothing rather than doing something wrong, and neither
  says anything. The library is not the place to decide that a program's
  navigation call is a mistake worth a banner.
- **Rollback** is a revert; nothing is stored and no signature changed.
