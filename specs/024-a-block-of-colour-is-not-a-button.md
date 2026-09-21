# Spec 024: a block of colour is not a button

**Issue**: [#26](https://github.com/ushineko/fynedesygn/issues/26)

## Status: COMPLETE

## Context

A 25-second CPU profile of hotaru's scene editor being resized, taken with
this module's own `profiling` package:

	14.33s of 18.84s   76%   scrollContainerRenderer.Layout
	 5.36s             28%   runtime.mapaccess2
	 2.69s             14%   buttonRenderer.MinSize

Nothing in it is one slow function. It is breadth: a scroller lays out
everything it holds rather than what is visible, so a drag step measures every
widget on the page, and the page is a block per run of lights across six
devices.

Each block was a `canvas.Rectangle` with an invisible `widget.Button` stacked
over it to catch the tap, wrapped again for a tooltip -- five objects. And a
button measures itself with a theme lookup, a padding calculation and a
`RichText.MinSize` for a label that is the empty string.

That is the right amount of work for a button and the wrong amount for a
coloured square.

### Why it belongs here

Because the shape recurs: "an invisible control stacked over a drawing" is how
anybody builds a palette, a colour grid, a heat map or a seating plan out of
what Fyne provides. A component that would be rebuilt per app belongs in this
module, and the count -- five objects, and what each of them measures -- is
the kind of thing nobody looks at until a profile makes them.

### What it does not fix

The other 76%. A scroller laying out its whole content is Fyne's, and the
answer to it is virtualisation of the kind `markdown.Pane` already does for a
document. That is a larger piece of work and is not this spec's.

## Requirements

**R1. One widget.** A tappable block of colour, not a stack of a drawing and
a control.

**R2. Its minimum size is the number it was given.** No theme lookup, no
padding, no text to shape.

**R3. It recolours without being rebuilt.** Fields set directly and a
`Refresh`, because building a grid once and recolouring it is the shape a
palette wants -- and rebuilding a grid of widgets when one changes is what
makes an editor slow in the first place.

**R4. A swatch with nothing to do does nothing**, rather than calling a nil
function.

**R5. In the gallery**, like every other exported thing here.

## Acceptance Criteria

- [x] AC1. `MinSize` is the size the swatch was made with.
- [x] AC2. A benchmark in the test says a swatch measures faster than the
      button it replaces, and fails if that stops being true.
- [x] AC3. Tapping calls `OnTapped`; a swatch without one does not panic.
- [x] AC4. Colours can be changed and refreshed on a laid-out swatch.
- [x] AC5. The gallery's Widgets section shows a row of them.
- [x] AC6. `docs/performance.md` says what the finding was, next to the
      rule it produced.

## Risks & Assumptions

- **A swatch is not a button and does not look like one.** No hover state, no
  focus, no keyboard activation. That is the trade: a control somebody must be
  able to reach with a keyboard should stay a button.
- **The benchmark compares a warm measurement.** Fyne caches a button's
  minimum size, so the gap in the test is smaller than the gap in the profile,
  where a resize invalidates it. The test is a direction, not the
  measurement; the measurement is the before-and-after profile.
- **Rollback** is a widget nothing else has to use.

## Alternatives Considered

Considered making the rectangle itself tappable by giving `canvas.Rectangle` a
wrapper that implements `fyne.Tappable`; rejected because that is the same
object count with the indirection moved, and a widget is what Fyne's event
walk is looking for anyway.
