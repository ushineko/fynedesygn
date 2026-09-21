# Spec 021: a height is not a property of a block

**Issue**: [#21](https://github.com/ushineko/fynedesygn/issues/21)

## Status: COMPLETE

## Context

`Pane.measure` is the one place a document's shape is decided. It read:

	h := o.MinSize().Height
	o.Resize(fyne.NewSize(w, h))
	p.spacers[i].SetMinSize(fyne.NewSize(0, h))

with a comment saying the second `MinSize` the code used to make was "asking
the same question again after a Resize that cannot change the answer".

It can change the answer, and for two of the three kinds of block in this
module it does. Fyne's `RichText` reports the size of unwrapped text until it
has been resized -- the height of a paragraph is a property of the paragraph
*and the width it is given* -- and `mermaid.Diagram.MinSize` derives its height
from the width recorded by its last `Resize`. Asked before that width arrives,
a paragraph measures as one line and a diagram as its natural height.

On this module's own README the difference is 7141px measured against 11186px
drawn: the document reserves about two thirds of the space it needs, and every
block below a mis-measured one sits above where it will be drawn.

### Why it looked fine

Because the pane re-measures on every width change, and any width change after
the first gets the right answer -- the blocks have been resized by then. A
window that is opened and then touched is correct. The wrong measurement is
only ever the *first* one, and the first one is invisible unless something
compares it against a later one.

### What made it visible

hotaru's About section: a 448-line README embedded and rendered, in a window
that polls the machine every two seconds and rebuilds the section on screen
when anything it watches moves. Each rebuild is a new pane, each new pane
measures short, and the document under the reader's scroll position moves by
several screens. Reported as "jumping to top again when hitting the mermaid",
which is what it looks like: the reader is somewhere past the middle, the
document shrinks under them, and the scroller clamps.

The diagram is where it shows because a diagram is the tallest block in the
document and the one whose measured and drawn heights differ most.

### The rider

This came in as one sentence at the end of spec 015, a performance change:
"measure also asks each block its MinSize once rather than twice." The profile
that motivated 015 was real and its main fix -- coalescing the measure during
a drag -- is what makes the second call affordable: it now happens once per
settled width rather than hundreds of times per drag.

An optimisation that changes an answer is not an optimisation. The measured
heights are the contract the pane's whole virtualisation rests on, and this
spec is the note that says so next to the code.

## Requirements

**R1. A block is resized before it is measured.** Width first, then height,
for every block, because the height depends on the width for prose and for
diagrams.

**R2. A document is the same height however it got there.** A pane built and
measured once matches a pane that has been resized through other widths, on
the same source at the same width.

**R3. The coalescing stays.** Spec 015's `SettleResize` is what keeps the
second measurement cheap; nothing here reverts it.

## Acceptance Criteria

- [x] AC1. A freshly built pane's height equals that of a pane that has been
      laid out and scrolled, on the same document at the same width, and the
      test fails on the previous behaviour.
- [x] AC2. The existing pane tests pass unchanged -- virtualisation, stable
      height while scrolling, re-measure on a narrower window, the drag
      coalescing.
- [x] AC3. The reason lives beside the code, naming spec 015, so the next
      person to see two `MinSize` calls does not remove one again.

## Risks & Assumptions

- **One more MinSize per block per settled width.** It is the expensive call
  in this path. With `SettleResize` set it happens once per width the user
  stops at; without it, once per width change, which is what the module did
  before spec 015 and what every consumer ran on until then.
- **Rollback** is a revert; the previous behaviour is a document that measures
  short, so rolling back reinstates a wrong number rather than a slow one.

## Alternatives Considered

Considered measuring twice only for blocks whose height can depend on width --
prose and diagrams, not code panels; rejected because that is a list of block
kinds to keep in step with `RenderBlock`, and getting it wrong produces exactly
this bug in a narrower case.
