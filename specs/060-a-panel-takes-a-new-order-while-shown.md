# Spec 060: a panel takes a new order while it is shown

**Issue**: [#173](https://github.com/ushineko/fynedesygn/issues/173)

## Status: COMPLETE

## Context

A glance `Panel` added cards and never moved one, so the order of a panel's
cards was fixed when the window was built. hayami lets the user reorder its
sections from the preferences; the window could follow only at the next start,
and the preferences did not say so, so a reorder looked broken (hayami #158).

## Requirements

- R1 `Panel.SetOrder(cards ...*Card) error` puts the cards in a new order while
  the panel is shown. The cards named come first, in the order given; cards
  not named follow in the order they already had, so a caller ordering only the
  cards it knows of never loses one.
- R2 A card the panel does not hold, or a card named twice, is `ErrCardOrder`,
  and the order is left as it was.
- R3 No card is rebuilt: each keeps its rows (the handles `Card.RowByID`
  returns), theme, allowed, available and stale state, and the arrangement,
  which carries on with the new order in Stack, Grid and Lines.
- R4 A new order resizes the window once, as a card appearing does; the order
  the panel already has does nothing.
- R5 It runs on the UI thread, as the panel's other setters do.

No `Panel.Remove`: nothing asked for one, and a reorder does not need it.

## Acceptance Criteria

- [x] A new order is drawn top to bottom in Stack and in Lines, and is the
  order Grid lays out (`TestANewOrderIsDrawnInStackAndLines`,
  `TestANewOrderIsTheGridsOrder`).
- [x] Rows, hidden and stale state and the arrangement survive
  (`TestACardKeepsItsStateAcrossANewOrder`).
- [x] Cards not named follow in their own order
  (`TestCardsNotNamedFollowInTheirOwnOrder`).
- [x] A stranger or a card named twice is refused and changes nothing
  (`TestAnOrderNamingAStrangerOrATwiceIsRefused`).
- [x] The window keeps its size across a reorder of the same cards
  (`TestANewOrderKeepsTheWindowsSize`).
- [x] Falsified: with the drawn objects left in the old order the Stack, Lines
  and Grid tests fail; with the check for strangers removed the refusal test
  fails.

## Risks & Assumptions

- The cards' objects are moved within the one container the layout already
  owns (spec 057's single layout for every arrangement), so nothing is
  re-parented and no size is lost.
- Rollback: revert; nothing else calls `SetOrder`.

## Gaps found

- No gallery entry: a reorder is a change over time, and the gallery shows
  still demos.

## Verification

2026-10-08, Windows 11, Go 1.26.0, `go test -race ./...`: all packages pass;
the six order tests above pass and fail as falsified.
