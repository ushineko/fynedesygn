# 073 — a glyph the face can draw

**Issue**: #217

## Status: COMPLETE

Built and tested. Verified on Linux against Fyne's bundled face and Adwaita
Sans; the Windows reading that prompted it is hayami #213, and hayami's own
change is what will confirm the fix on that desk.

## Executive Summary

A program that wants an arrow in a label picks a character and hopes. The
hope is the bug, and it failed twice in the same place: U+2190 drew as a
missing-glyph box on one desk, the character chosen to fix it drew as a box
on another. `theme.FirstDrawable` asks the face instead -- the theme knows
which file it draws with, and the file says which characters it has.

## Context

hayami's drawer gained a back button and breadcrumb separators (spec 072,
hayami spec 065). U+2190 LEFTWARDS ARROW is not in Adwaita Sans and drew as
a box on the Linux desk it was written at, so it was replaced with U+25C2
BLACK LEFT-POINTING SMALL TRIANGLE, which looked right there. On Windows
both the triangle and the matching U+25B8 drew as boxes (hayami #213, spec
066 gap 3).

Checking the faces says why, and says the first fix was never right:
**neither Fyne's bundled face nor Adwaita Sans contains U+25C2**. The Linux
desk drew it because the platform substituted another font at draw time.
Windows did not substitute. So the character was wrong on both desks and
only one of them said so, which is the worst way for a fault to behave.

Neither desk can speak for the other, and no amount of looking at one
screen establishes what a second will draw. The question is not "does this
look right here" but "does the face have this character", and that is a
question with an answer.

## Requirements

- R1 `theme.FirstDrawable(th, style, candidates...)` is the first candidate
  whose every character the theme's face has a glyph for.
- R2 When none can be drawn, the answer is the last candidate: the caller's
  fallback, which should be one every face has and in practice means ASCII.
  A nil theme and a face that will not parse answer the same way, because
  nothing can be said about a face that cannot be read. No candidates at all
  answers with "".
- R3 `BackGlyphs` and `ForwardGlyphs` are ordered candidate lists, best
  first, each ending in ASCII. A program that wants to point back should not
  have to assemble its own list and get it wrong in its own way.
- R4 Parsed faces are kept by resource name. Parsing is reading and checking
  a file, and a label asks about its glyph every time it is drawn.
- R5 The answer is for the face the theme hands out now. A program whose
  font has changed asks again; the library does not cache the answer on the
  program's behalf.

## Acceptance Criteria

- [x] `FirstDrawable` returns a candidate the face has
      (`TestFirstDrawableAnswersWithAGlyphTheFaceHas`).
- [x] A character the face lacks is passed over
      (`TestFirstDrawablePassesOverWhatTheFaceLacks`), shown with a
      private-use code point no ordinary face has.
- [x] The last candidate comes back when none can be drawn, when there is no
      theme, and when the face will not parse
      (`TestFirstDrawableFallsBackToTheLast`).
- [x] Both standard lists resolve against Fyne's bundled face and against
      Adwaita Sans: each gives U+2039 and U+203A, because neither face has
      the triangles.
- [x] A Windows desk draws the chosen glyphs as glyphs: confirmed on
      gamerson-w11 (Windows 11) through hayami's change, by the desk that
      reported the boxes. No Linux desk can see that, so the tick is that
      desk's word and not this one's.

## Out of scope

- Choosing a glyph per platform. The face is the thing that decides, not the
  operating system, and a program that asked the operating system would be
  wrong on a machine whose font had been changed.
- Font fallback. Making a face draw what it does not have is the platform's
  business; this says what a face has.

## Risks & Assumptions

- `FirstDrawable` answers for the theme's own face. A platform that
  substitutes another font at draw time may well draw a character this
  reports as absent -- that is exactly what the Linux desk did -- so the
  answer is conservative: it picks what the face has rather than what the
  platform might find. A program wanting the prettier glyph on a desk that
  would substitute does not get it.
- The cache is by resource name and never emptied. Two different fonts
  served under one name would collide; nothing does that today.
- Only the regular style's face is asked for the style given. A face whose
  bold differs in coverage from its regular is not handled.

## Verification

Against Fyne's bundled face and Adwaita Sans on Linux: both lists resolve to
U+2039 and U+203A, and neither face has U+25C2 or U+25B8. The tests fail
when the fallback is removed.

Not verified here: how the chosen glyphs draw on Windows or macOS -- this
desk cannot see either screen. Windows is confirmed by gamerson-w11 through
hayami's change; macOS is still nobody's report.
