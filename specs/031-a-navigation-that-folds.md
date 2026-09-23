# Spec 031: a navigation that folds

**Issue**: [#51](https://github.com/ushineko/fynedesygn/issues/51)

## Status: COMPLETE

## Context

A program whose navigation has grown says so by listing more sections, and
past seven or eight the list stops reading as a set of places and starts
reading as an inventory. The sections that caused it are usually of a kind --
three sources of pictures, four reports, five importers -- and what the reader
wants from the list is the kind, until the moment they want one of them.

clockwork-orange is the case. Its window lists `Local`, `Wallhaven` and
`DuckDuckGo Images` between `Service` and `History`: three of nine entries,
all three of one kind, and a fourth plugin would make it ten. Along the top
they are nine buttons on one line.

## Requirements

- **R1** A program may declare groups. A group has a title, an optional icon
  and a list of member section titles, matched case-insensitively.
- **R2** A group is **not a section**. It has no page, `Select` does not reach
  it, `Current` never returns it, and `Ctrl+1..9` count sections in the order
  the program listed them -- so folding three sections under a heading does
  not renumber the shortcuts of the ones below.
- **R3** A member title that names no section is ignored, and a group whose
  members all do is not drawn. A program that builds sections conditionally
  lists the group either way.
- **R4** Opening and closing is remembered between runs, in the settings store
  the shell already owns, under `fynedesygn.nav.groups`. Closed is what is
  stored, so a group a program adds later starts open.
- **R5** Closing a group does not change the page on screen and does not
  rebuild it. The reader stays where they were reading.
- **R6** Navigating to a member of a closed group opens the group. `--section`,
  `Select` and `Ctrl+1..9` are all navigation, and a window that went somewhere
  without showing where is a window that lost the reader.
- **R7** Every shape draws groups:
  - **Labels down the left** -- the list. A heading row with a disclosure mark
    and its members indented beneath it.
  - **Icons down the left** -- buttons. A group button with its members
    indented beneath it.
  - **Along the top** -- buttons. A group button on the top row and its
    members on a second row beneath it, never in the middle of the row the
    reader picks a group from.
- **R8** The group button is drawn as current whenever the section being read
  is one of its members, open or closed. Open that is two lit buttons for one
  section, which is what it is: the branch and the leaf.
- **R9** The header's own actions keep their own height and sit against the
  top of the header row. A `Border` stretches what it is given to the height
  of the row, and the row is two buttons tall whenever a group is open along
  the top.

## Design

`Options.Groups []NavGroup`, an annotation over the flat `Sections` list
rather than a change to it. `s.current` stays an index into `Sections`, so
every existing path -- `index`, `Select`, `Current`, `selectIndex`, the
shortcuts -- is unchanged.

The list is drawn from a row model (`shell/group.go`): `buildRows` walks the
sections in order and inserts a heading before the first member of each group,
leaving out the members of a closed one. `rowFor` maps a section back to its
row, which is what `selectIndex` needs and what restores the highlight after
a group is opened or closed.

A heading is not a place, so `widget.List.OnSelected` on a heading row puts the
selection back where it was and toggles the group instead. Putting it back is
also what makes a second click on the heading work at all: `List.Select`
returns early when the row is already selected, so a heading that kept the
highlight would go deaf.

Row layout is `navRowLayout` rather than an `HBox`: the twisty and the indent
occupy one column, so a member's icon lines up under its heading's.

## Acceptance Criteria

- [x] A group draws a heading with its members indented beneath it, and the
      heading sits where the first member was (`TestAGroupDrawsItsMembersUnderAHeading`)
- [x] A group is not counted among the sections (same test)
- [x] Closing a group hides its members, keeps the page on screen and rebuilds
      nothing (`TestClosingAGroupHidesItsMembersAndKeepsThePage`)
- [x] The heading never keeps the highlight, so a second click closes what the
      first opened (`TestTheHeadingIsNeverWhatIsHighlighted`)
- [x] Selecting into a closed group opens it (`TestSelectingIntoAClosedGroupOpensIt`)
- [x] Ctrl+1..9 count sections, not rows (`TestNumberKeysCountSectionsNotRows`)
- [x] A closed group is remembered, and a stored title that names no group is
      dropped (`TestAClosedGroupIsRememberedAndAnUnknownOneIsDropped`)
- [x] A group naming no section draws nothing (`TestAGroupNamingNoSectionDrawsNothing`)
- [x] The button navigations draw the group and honour closed
      (`TestTheButtonNavigationsDrawTheGroupToo`)
- [x] Along the top, the members are on a second row (`TestTheTopRowKeepsTheGroupsMembersOnASecondRow`)
- [x] The group button opens and closes rather than navigating
      (`TestTheGroupButtonOpensAndClosesRatherThanNavigating`)
- [x] The group button is lit while the page is one of its members
      (`TestTheGroupButtonIsLitWhileThePageIsOneOfItsMembers`)
- [x] The header actions keep their height and stay on the first row
      (`TestTheHeaderActionsDoNotGrowWithASecondNavRow`)
- [x] The gallery demonstrates a group (`Components`, over Widgets, Table and Fonts)

## Risks & Assumptions

- **Members should be contiguous** in `Sections` order. The list is drawn in
  that order and the heading appears where the first member would have been,
  so a group whose members straddle a non-member would draw that non-member
  after the group's members. Documented on `NavGroup` rather than enforced:
  rejecting it would mean an error path for a program that can simply order
  its own sections.
- **Rollback**: the feature is additive. A program that lists no groups gets
  the navigation it had; `Options.Groups` defaults to nil and every row is a
  section. The one change that reaches a program without groups is R9, the
  header actions no longer stretching -- which was only ever visible on a
  header taller than one button, and nothing but a group makes one.
- No new dependency, no network, no file format change beyond one added key in
  a settings file the shell already writes.

## Gaps found

None. The row model, the settings store, the tip wrapper and the test helpers
were all already here.
