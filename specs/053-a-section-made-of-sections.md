# Spec 053: a section made of sections

**Issue**: [#162](https://github.com/ushineko/fynedesygn/issues/162)

## Status: COMPLETE

## Context

A section that grows past one screen of one activity has two options in the
shell today. It can scroll further, or it can become several sections, and
several sections make the navigation longer for something that is one place
in the reader's mind.

hotaru already has a third option. Its `internal/gui/create.go` puts Scenes,
Pictures and Dashboards under one Create entry. A row of buttons picks which
part is drawn, and the section passes the shell's lifecycle (Detach, Arrive)
through to its parts. nmsbonker issue #7 needs the same shape for its Tweaks
section, which grows from twelve built-in mods to about twenty-five grouped
by subject. Two programs carrying one hand-rolled shape is the point at which
the shape belongs in the library.

`NavGroup` (spec 031) is not a substitute. A group adds rows to the
navigation, which is what this shape avoids. A group also disappears in the
icons-only and along-the-top shapes, where a tab strip stays.

## Requirements

- R1 `shell.NewTabs(title string, icon func() fyne.Resource, parts ...Section) *Tabs`
  returns a `Section` whose `Build` draws a strip of one button per part,
  in the order given, above the chosen part's own `Build`. The chosen
  part's button has high importance and the others have none.
- R2 The strip sits in the `Border` top, so a part that fills its space with
  a `Border` and its own scroller (affixed controls) still fills the space
  under the strip.
- R3 Lifecycle. `Tabs.Detach` detaches every part that is a `Detacher`, not
  only the part showing. `Tabs.Arrive` arrives at the part showing.
  `Tabs.Wide` reports the part showing.
- R4 Choosing a tab is arriving somewhere. The shell detaches every section,
  arrives at the new part and scrolls the content pane to the top. It does
  not call `OnInvalidate`. Choosing the tab already showing does nothing.
- R5 The chosen tab survives a rebuild, navigating away and back, and a
  restart. It is stored by part title under `settings.Prefix + "tabs"`,
  keyed by the section's title. A stored title that names no part is
  ignored, and the first part shows.
- R6 `Tabs.Show(title string) bool` chooses a part by title,
  case-insensitively. It is for deep links and tests, and it stores the
  choice. `Tabs.Showing() string` and `Tabs.Parts() []Section` read the
  state back.
- R7 A `Tabs` with no parts builds an empty container and does not panic.
- R8 Documentation. `docs/design-system.md` says when to use tabs and when
  to use a navigation group. The README changelog has an entry under
  `### Unreleased`. The gallery's Shell section shows a `Tabs` section.

## Acceptance Criteria

- [x] R1 A headless test builds a three-part `Tabs`. The strip has three
      buttons in order, and only the first has high importance.
- [x] R3 A test with logging parts shows that a shell swap detaches all
      three parts and that navigating to the `Tabs` arrives at the part
      showing and no other.
- [x] R4 A test presses the second tab and checks three things: the second
      part is built, it arrives, and `OnInvalidate` is not called. Pressing
      the same tab again builds nothing.
- [x] R5 A test chooses a tab, stops the shell and opens a new one on the
      same settings file. The chosen tab shows. A stored title that names
      no part falls back to the first part.
- [x] R6 `Show` is case-insensitive and returns false for an unknown title.
- [x] R7 A `Tabs` with no parts builds without panicking.
- [x] R8 The design-system doc, the changelog and the gallery are updated.
      The gallery is photographed.

## Risks & Assumptions

- Additive. No existing API changes, so a consumer that does not use `Tabs`
  sees no difference.
- The stored choice is one small JSON object in the existing settings store.
- Rollback: revert the commit. Consumers pin a tag.
- hotaru's `Create` keeps its own copy until it adopts this. Its parts also
  pass a hotaru-specific `Changed` and `Busy` through, which a consumer can
  still do by wrapping `Tabs`.

## E2E Test Plan

nmsbonker issue #7 is the first consumer. Run nmsbonker-gui on the tagged
version and photograph its Tweaks section. Then switch tabs, leave the
section, return, and restart the window. The chosen tab survives each step
(R4, R5).
