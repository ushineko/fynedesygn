# Spec 014: A control that starts work is affixed

## Status: COMPLETE

- **Priority**: Medium
- **Estimated Complexity**: Low
- **Branch**: `feat/affixed-controls`
- **Asked for by**: clockwork-orange, twice in one week (its specs 013 R11 and
  014), and found once in this module's own gallery

## Context

The design system already held both halves of this rule, in two places, and
never joined them.

Under **One scroll per section**, it records the cause: *"Fyne hands a wheel
event to the innermost scrollable under the pointer and does not pass it on."*
Under **Sections**, it records the shape: *"Sections whose bottom action strip
must stay visible use `Border(nil, actions, nil, nil, VScroll(body))`."*

What was missing is the sentence between them — that a control you must scroll
to is a control you can be scrolled away from, and in a section that also holds
a log or a table it may not be reachable at all. Stated as a shape, the rule
only reaches a section whose author has already decided they need it. Both
sections that broke it were written by someone who had read the document.

The evidence:

- clockwork-orange's plugin section put Download now and Reset & run last in a
  single column. On Wallhaven, eleven fields long, they sat below the fold.
- Its Service section put Start, Stop, Restart, Install and Uninstall in the
  scrolling half of a split whose other half is a log pane. Below the fold
  again — and because the log pane is a `widget.List`, scrollable in its own
  right and the larger target at any ordinary window height, the wheel went to
  the log. The window reported that the service was running and offered no way
  to stop it. That one shipped.
- **This module's own gallery has it.** The Log section's Start and Stop sit in
  the scrolling half of a `VSplit` above a `logpane.Pane`: the same shape, in
  the program that exists to demonstrate the design system.

Three occurrences, one of them in the reference program, all after the rule was
written in its narrow form.

## Requirements

### R1. The rule

- R1.1 `docs/design-system.md` states it as a rule rather than a shape:

  > Every control that starts, cancels or commits work occupies the same place
  > in its section however much of the section is scrolled. What scrolls is the
  > material the control acts on — the form, the table, the statistics, the
  > prose — never the control itself.

- R1.2 It is stated together with its cause, which the document already carries
  two rules earlier: the wheel goes to the innermost scrollable under the
  pointer, so the failure is unreachable rather than merely hidden.
- R1.3 The exception is stated, because without it every gallery card trips the
  rule: a control that **is** the exhibit — a card's buttons that its prose
  describes, a row's own Remove button — is the material and travels with it.
  The test is whether the control acts on the section or belongs to what
  scrolls.

### R2. The check

- R2.1 `fynetest.Scrolled[T]` is every object of type T that a scroller
  encloses.
- R2.2 `fynetest.ScrolledButtons` is the labels of the enclosed buttons, which
  is the form the check usually takes: a named control must not appear in it.
- R2.3 Structural, not rendered: this asks where an object sits in the tree the
  program built, which is a property of the layout rather than of what was
  drawn.
- R2.4 An adopter with more than a couple of these keeps the list of what must
  stay affixed beside its sections and walks it in a test. clockwork-orange's
  `AffixedActions` is the worked example; the library ships the primitive, not
  the policy, because only the program knows which of its buttons is an action
  and which belongs to a row.

### R3. Walk descends a split

- R3.1 `children` descends `container.Split` through `Leading` and `Trailing`.
- R3.2 Prerequisite for R2: a `Split` is a widget with two exported halves, so
  a structural walk stopped at it — and since `Shell.VSplit` (spec 012) every
  section with a pane under a divider is one. Found in clockwork-orange, where
  a section became a split and every test looking for a button in it began
  finding nil, while `Tips`, which descends through renderers, still saw them.
  The two walkers disagreed about the same tree.

### R4. The gallery obeys the rule

- R4.1 The Log section's Start and Stop are affixed above the divider; the
  prose scrolls behind them.
- R4.2 A test pins it, and fails on the shape the section had.
- R4.3 The other sections keep their buttons in their cards, under R1.3, and
  the reason is in the test rather than left to be re-derived.

### R5. Release

- R5.1 v0.1.26. Additive: two new helpers, one walker that reaches further, a
  document rule and a gallery layout. Nothing existing changes shape.

## Acceptance Criteria

- [x] AC1 `docs/design-system.md` states the rule, its cause and its exception (R1)
  - Verified: `docs/design-system.md` under Sections — the rule, the wheel-event cause it points back to, and the exhibit exception
- [x] AC2 `fynetest.Scrolled[T]` and `ScrolledButtons` exist and are documented (R2.1, R2.2)
  - Verified: `fynetest/walk.go` — both documented with the rule they serve
- [x] AC3 They find a control nested several containers deep, and do not report one in a sibling half of a split (R2.3)
  - Verified: `TestScrolledLooksThroughTheWholeTree` — a button three containers deep inside the scroller is found, one in the other half of the split is not
- [x] AC4 `Walk` and `All` reach both halves of a `container.Split` (R3.1)
  - Verified: `TestWalkDescendsASplit`
- [x] AC5 The gallery's Log section has its controls affixed (R4.1)
  - Verified: `cmd/fynedesygn-gallery/sections_forms.go` — `Border(nil, HBox(start, stop), nil, nil, VScroll(prose))` in the split's top half
- [x] AC6 A test fails on the shape that section had (R4.2)
  - Verified: `TestTheLogSectionsControlsAreAffixed`, checked against the bug: stashing the section fails it with "a control that starts work must not scroll away from the section it acts on"
- [x] AC7 `make test` passes
  - Verified: `make test` green
- [x] AC8 `make lint` is clean
  - Verified: `make lint` — 0 issues
- [x] AC9 `govulncheck ./...` is clean
  - Verified: `govulncheck ./...` — no vulnerabilities found
- [x] AC10 The README changelog names the release (R5.1)
  - Verified: README changelog, 0.1.26, which also names the `Walk` behaviour change

## Risks & Assumptions

- **Rollback**: revert the commit. The module is `v0.x` and adopters pin a
  version, so nothing moves under them until they choose to.
- **Additive API.** `Scrolled` and `ScrolledButtons` are new; no existing
  signature changes.
- **`Walk` reaching further is a behaviour change**, not an API one. A test
  that counted objects under a split got nothing there before and gets both
  halves now. That is the bug being fixed, but an adopter whose test asserted
  an exact count could see it fail — named in the changelog for that reason.
- **The check is structural and cannot see the screen.** It pins that a control
  is outside the scroller, which is the property that was violated in all three
  cases. It cannot tell whether a control is *visible* at a given window size;
  a window rendered to an image could, and `fynetest` can already render one,
  so it is reachable if the structural check proves insufficient.

## Alternatives Considered

- **Ship a layout helper** — `widgets.WithActions(body, controls...)` — instead
  of a rule and a check. Rejected for now: the three sections that needed it
  wanted the controls in different edges and one of them inside a split half,
  and `container.NewBorder` already says it plainly. A helper would be a second
  way to spell one Fyne call.
- **Have the check infer which buttons matter.** Rejected: a walker cannot tell
  a section's action from a row's own button, and flagging a list's per-row
  Remove buttons would make the check noise. The program names them.
- **Leave the rule in the narrow form and fix the two programs.** Rejected:
  three occurrences after the narrow form was written, one in the gallery, is
  the argument against it.
