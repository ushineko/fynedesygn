# 041 — A section keeps the window's theme

**Issue**: #108

## Context

`shell.Options.OwnAppearance` draws a window in its own appearance by wrapping
its content in a `container.ThemeOverride`. It works for the section the window
opens on and fails for every section reached afterwards.

The cause is Fyne's, and this repository already documents it for the same
mechanism: `glance.Panel.SetTheme` says that items added to the content after
the override was built keep the default theme until the override is refreshed.
A shell builds its sections lazily. `swap` assigns `s.content.Content =
sec.Build(s)`, which installs a subtree the override has never walked, so it
draws in the application's theme.

That is not a neutral fallback, and it is why this is worth a spec rather than
a line. `OwnAppearance` exists for a program with two archetypes in it — a
glance panel read from across a desk and a settings window read at arm's length
— where the panel must own the application's theme because its cards are canvas
objects that cannot be overridden. So the application's theme *is* the other
window's, and a section that escapes the override does not lose its styling: it
puts on the panel's face, at the panel's size, with whatever the panel has
wrapped around its theme.

Reported against hayami, where opening the preferences window looked correct
and clicking any section in it switched that section to the panel's font and
the panel's card transparency. The window a user configures the panel from was
the one that stopped looking like itself.

## Requirements

### The shell holds its override and refreshes it

`layout` builds the override and throws the handle away. It keeps it, and a
`reapply` method refreshes it. `swap` calls `reapply` after installing a
section.

A no-op for a window that does not own its appearance, which is most of them:
there is no override to refresh.

### A canary

This is a Fyne behaviour the library works around rather than a rule of its
own, so it gets a row in `docs/fyne-quirks.md` and a test named after it, like
the other thirty-six. The test asserts on the size text is actually **drawn**
at, recovered from the rendered tree — the question is what reached the
widgets, and the theme is only what was asked for.

It asserts the opening section too, so that a failure says *when* a section was
built rather than merely that the override is broken.

## Acceptance criteria

- [x] `Shell` holds its `*container.ThemeOverride`, and `layout` clears it for
      a window that does not own its appearance.
- [x] `swap` refreshes the override after installing a section.
- [x] A section reached after the window was built draws at the window's own
      text size, not the application's.
- [x] The section the window opened on still does, so the test distinguishes
      the two cases.
- [x] `docs/fyne-quirks.md` has the quirk, numbered and in order, naming the
      canary.
- [x] `README.md` carries the changelog entry under `### Unreleased`.
- [x] The canary was falsified: removing the `reapply` call fails it.
- [x] `make test` passes.

## Risks & Assumptions

- **Refreshing an override walks its content.** It happens once per section
  change, which is a user gesture and not a poll, so the cost is not on any hot
  path. A shell that changed sections on a timer would be doing something else
  wrong.
- **Not a breaking change.** Nothing exported changes; a window that does not
  own its appearance is unaffected.
- **Rollback**: revert the commit.

## Status: COMPLETE
