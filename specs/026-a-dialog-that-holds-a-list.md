# Spec 026: a dialog that holds a list

**Issue**: [#32](https://github.com/ushineko/fynedesygn/issues/32)

## Status: COMPLETE

## Context

Fyne sizes a dialog to its content's minimum size. That suits a confirmation,
which is two lines and two buttons, and suits nothing with a list in it --
because **a scroller's minimum size is almost nothing**, so a dialog built
around one opens at the size of its buttons.

Found twice in a week, both times by somebody using the program rather than by
anybody reading the code:

- a file chooser showing about four names, which is a directory read through a
  slot. Worked around inside `dialogs` with a fixed 760x520.
- hotaru's shortcut chooser offering eighteen keys through a slot showing one
  and a half of them, clipped at both ends. Worked around in hotaru with a
  helper of its own, `Roomy`, which nobody here knew about.

Two programs, two copies, one behaviour -- which is the copy-carrying problem
this module exists to end.

### Sized from the window, not to a constant

760x520 is a reasonable file browser on a laptop and a postage stamp on a
large desk. Most of the window is right on both, and leaving a margin keeps
the window behind visible -- a dialog that covers everything reads as a new
window rather than as a question about this one.

The constant stays as a floor: most of a very small window is still too small
to read a directory in.

### Show, then resize

Carried from the existing helper and worth repeating in the new one's comment,
because it is a crash rather than a small dialog. Before `Show` a dialog has
no window, and `Resize` asks it for its minimum size, which in Fyne 2.8.1
dereferences that nil window and takes the process with it (quirk 1). It
reached somebody.

## Requirements

**R1. One helper**, in this module, that shows a dialog at most of the window.

**R2. A floor**, so a small window still gets a usable dialog.

**R3. Show before resize**, and say why where the next person will read it.

**R4. The file choosers use it**, rather than keeping their own sizing.

**R5. The rule is in `docs/design-system.md`**, next to the other dialog
rules, because the helper only reaches somebody who already suspects they need
it.

## Acceptance Criteria

- [x] AC1. `Roomy` shows and then resizes a dialog.
- [x] AC2. `RoomySize` is most of the window where that is larger than the
      floor, and the floor where it is not.
- [x] AC3. A nil window is the floor rather than a dereference.
- [x] AC4. `ChooseFile` and `ChooseFolder` go through it, and the existing
      test that taps every chooser button still passes.
- [x] AC5. The design system says a dialog holding a list is shown roomy, and
      why.

## Risks & Assumptions

- **85% is a judgement**, made against two cases. A dialog with three lines in
  it should not use this; the rule says "holds a list" for that reason.
- **A test cannot see a dialog's size on screen**, so the tests are on
  `RoomySize` and the existing tap-every-button regression test covers the
  crash. The size itself was checked by looking at it.
- **Rollback** is one function nothing is obliged to call.

## Alternatives Considered

Considered a size per dialog kind, as the package already has for the
destructive, prompt and chooser shapes; rejected because the thing that varies
is the window rather than the dialog, and four constants would all be wrong on
the same screen.
