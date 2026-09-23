# Spec 034: a theme without that face

**Issue**: [#56](https://github.com/ushineko/fynedesygn/issues/56)

## Status: COMPLETE

## Context

Rendering this crashes the pane:

```
**bold with `code` inside**
```

Not an odd thing to write. It is how anybody emphasises an identifier in
prose, and spec 033's tutorial did it in its first draft -- which is how this
was found, by the test that renders the README.

### What happens

`Pane` hands inline Markdown to `widget.NewRichTextFromMarkdown`, which is the
right division of labour: Fyne parses the inline grammar and this package
decides what a *block* is. Bold around a code span gives a segment whose style
is bold **and** monospace, and the painter then asks the theme for that face:

```
Fyne error: font for style fyne.TextStyle{Bold:true, Monospace:true, ...}
            not defined in theme Default Test Theme
painter.loadMeasureFont({0x0, 0x0})
panic: runtime error: invalid memory address or nil pointer dereference
```

The theme returns nothing and Fyne dereferences it rather than falling back.

### Who it hits

Measured rather than assumed:

| Theme | Bold monospace | Monospace |
|---|---|---|
| Fyne's test theme | **absent** | present |
| fynedesygn's own | present | present |

So a program using this library's theme renders it. A program under the **Fyne
test driver** crashes -- which is every headless test, which is the thing this
library sells and the way every consumer tests their own sections. A document
with an emphasised identifier takes down a suite with a stack trace pointing
into Fyne's painter and nothing naming the document.

Not in production with our theme, then, and shipping it is still shipping a
crash: the consumer who hits it is the consumer following this project's own
advice about headless tests.

### The fix

**Do not ask a theme for a face it does not have.** Before handing segments to
the painter, look each style up; where the theme returns nothing, drop `Bold`
and keep `Monospace`.

Keep monospace because it carries more: it says the run *is* code, which is
why the author reached for backticks. Bold is emphasis, and emphasis that
cannot be drawn is a smaller loss than a code span that stops looking like
one.

**Conditional on the theme rather than always**, because a theme that has the
face should draw it. fynedesygn's own does. Stripping unconditionally would
degrade every real program to work around a gap in a test fixture.

This is a Fyne quirk and gets a row in `docs/fyne-quirks.md` with a canary
named after it, like the other thirty-four -- so that the day Fyne falls back
instead of dereferencing, one test fails and says which workaround to delete.

## Requirements

**R1. A document with bold around a code span renders**, under a theme with no
bold monospace face.

**R2. A theme that has the face still gets bold monospace.**

**R3. Every place this package builds a `RichText` from Markdown is covered**,
which is the document pane and the table cells.

**R4. The quirk is in `docs/fyne-quirks.md` with its canary.**

## Acceptance Criteria

- [x] AC1. Rendering bold around a code span under the test driver does not
      panic.
- [x] AC2. The segment comes out monospace and not bold under a theme with no
      such face.
- [x] AC3. Under a theme that has the face, the segment keeps both.
- [x] AC4. A pipe table whose cell holds bold-around-code renders too.
- [x] AC5. The README renders, which is the test that found this.
- [x] AC6. `docs/fyne-quirks.md` has the row, and the canary is named in it
      and exists.

## Alternatives Considered

- **Strip bold unconditionally.** Simpler, and it makes every real program pay
  for a gap in the test theme.
- **Give the theme a bold monospace face.** Ours already has one; this is
  about the themes we do not control, including Fyne's own.
- **Render inline Markdown ourselves.** A large amount of work to own a
  grammar Fyne already parses, to fix one missing fallback.
- **Leave it and document it.** Rejected: a crash a consumer hits by writing
  ordinary prose is not a documentation problem.

## Risks & Assumptions

- **The lookup is per segment, per render.** A document is rendered per block
  and near the viewport, so this is a handful of map lookups on work that
  already measures text; it should not be visible, and it is not the sort of
  thing to measure on a profile before it exists.

- **Only bold and monospace are known to collide.** Italic monospace, and
  bold italic, may have the same gap. The fix looks up whatever style a
  segment has rather than special-casing this pair, so they are covered by
  construction -- but only the reported pair is tested.

- **A theme can change while a document is on screen.** The appearance
  section changes it, and the shell rebuilds sections when it does, so blocks
  are rendered again under the new theme.

- **Rollback** is a revert. Nothing outside `markdown` changes.
