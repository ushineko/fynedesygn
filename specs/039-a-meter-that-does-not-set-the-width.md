# Spec 039: a meter that does not set the width

**Issue**: [#83](https://github.com/ushineko/fynedesygn/issues/83)

## Status: INCOMPLETE

## Executive Summary

Populated before the PR is opened.

## Context

`Meter` puts its label and its caption on one line and the bar underneath. The
caption therefore sets the meter's minimum width, the meter sets the panel's,
and the panel sets the window's. A caption with three figures in it makes the
whole window as wide as that sentence, and every other card is padded out to
match.

Measured in hayami, which is the first consumer to feel it. Same machine, same
settings, only the drawn sections differ:

| sections drawn | window width |
|---|---|
| cooler + peripherals | 268 px |
| cooler + peripherals + usage | **655 px** |

One caption does that:

```
5h: 0 %  7d: 0 % (6d left)  limit: 34 % 403.51 / 1200.00 (2d left) · in  4h 59m
```

The archetype this package comes from does not lay it out that way. The Codex
section of `peripheral-battery-monitor` is three stacked rows, each one a
widget, a stretch, and a widget:

```
[icon] Codex                    Resets in 4h 59m
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Limit: 34%           Individual: 403/1200 (34%)
```

**The stretch in the middle is the whole point.** A row laid out that way is as
wide as its two ends, not as wide as everything it contains laid end to end.
Three figures cost the width of the longest pair rather than the sum of all
three and their separators, and the reset moves out of the sentence entirely
and into a column of its own.

`docs/glance.md`'s argument for the caption is untouched by this — *"the
caption does the work the bar cannot; a bar says 'most of it', the caption says
which window, how much of it, and when it resets"*. Those words still earn
their place. This is about where they go.

The no-jitter rule is what makes this delicate rather than obvious. A meter's
caption is already held to it: *"it sets the section's width"*, so a caption
formatted with `%.0f%%` takes the window's width with it as the value changes.
Moving figures into a trailing slot does not exempt them — a right-aligned
value that changes width drags the left-hand one about, which is worse than a
wide panel because it moves while being read. Everything in the new slots is
held to the same rule as the caption, and the tests say so.

## Requirements

- R1 A meter can carry a **trailing value in its header**, right-aligned
  against the meter's own width, for the reading the archetype puts there.
- R2 A meter can carry a **stats row under the bar**, a value at each end.
- R3 Both are optional and empty by default. A meter built as one is today
  lays out exactly as it does today, to the pixel.
- R4 The meter's minimum width is set by the widest *pair* across its rows,
  not by the sum of what the rows contain.
- R5 The new slots are held to the no-jitter rule: they are drawn in the
  monospace face the caption already uses, so a value that changes does not
  move the one opposite it.
- R6 The label column still pins, so several meters stacked in a card line
  their captions up as they do now.

## Acceptance Criteria

- [ ] AC1 A meter with a trailing header value lays it out against the right edge, and its minimum width is the label plus the caption plus the trailing value, not their concatenation. (R1, R4)
- [ ] AC2 A meter with a stats row draws both ends of it under the bar. (R2)
- [ ] AC3 A meter built with neither has the same minimum size and the same object tree as one built before this spec. (R3)
- [ ] AC4 A meter carrying the archetype's figures in the new slots is narrower than the same figures in one caption. Asserted as a measurement, with both widths. (R4)
- [ ] AC5 The new slots are monospace, so a value that changes width does not move its neighbour. (R5)
- [ ] AC6 Several meters in a card still line their captions up. (R6)
- [ ] AC7 Setting the slots after the meter is built repaints them, and setting them to empty takes the row away rather than leaving a gap. (R1, R2)

## Risks & Assumptions

- **This is the archetype's layout, not an invention.** The three-row shape,
  the stretch in the middle and what goes in each slot are all read off
  `peripheral-battery-monitor`'s Codex section, which has run that way through
  its 1.x releases. Where this differs from it, it is because Fyne's layout
  differs from Qt's, and not because the shape was improved on.
- **A wider API is a wider thing to get wrong.** Two optional slots is two
  more ways for a caller to put a changing value somewhere it jitters. They
  are documented against the same rule as the caption and AC5 asserts it, but
  the library cannot stop a caller formatting badly — the same as today.
- Rollback: revert. A meter built without the new fields is unchanged, which
  AC3 asserts rather than assumes.
