# Validation report: spec 036 a window that looks like the monitor

Date: 2026-09-26. Milestone: the glance window's margins, translucency and
opacity. Spec: `specs/036-a-window-that-looks-like-the-monitor.md`. Issue #73.

## Scope

`glance.Card` margins and title colour, `glance.Panel` gap and edge,
`glance.Options.Translucent` with its probe, `glance.Window.SetOpacity`,
`theme.WithTransparentBackground`, `kwin.OpacityScript` with its three call
descriptors, the `opacityactiverule` fix in `kwin.Rule`,
`examples/glance-monitor -translucent`, `docs/glance.md`,
`docs/fyne-quirks.md` quirk 32, and two refreshed captures in `docs/img/`.

## Phase 3: tests

`go test -race -count=1 ./...` on Linux, no display: all packages pass.
`go vet`, `gofmt -l`: clean. `golangci-lint v2.12.2`: 0 issues.

Ten tests added. Each was falsified by breaking the code and watching it fail,
which is the standard this repository holds visual work to, and it paid twice:

- `TestACardIsInsetFromItsOwnEdge` and `TestCardsAreSeparatedByTheCardGap`
  first asserted only `inner + 2*CardPadH()` and `height + CardGap()`. Both
  passed with the constants reverted to the padding token, because the
  assertion used the same function it was testing. Each gained a second
  assertion that the value has not collapsed to the scheme's padding.
- `TestTheOpacityScriptClampsToAFraction` survived its first mutation
  (`> 1` changed to `> 2`), which still clamped. Removing the clamp outright
  failed it.

On-screen checks, because a claim about what a window looks like is made
against a screenshot: the gap between two cards returns byte-identical to the
desktop pixel behind it under `Translucent`; the context menu over a
translucent panel is solid; a window under a package-installed rule at 50 % is
no longer opaque; both opacity routes move the card's pixel.

## Phase 4: code quality

- The four geometry constants are functions of the text size, not pixel
  counts, which is the rule `docs/glance.md` already stated for margins.
- `grantTranslucent` and `setWindowOpacity` each have a no-op twin behind the
  exact complement of their build constraint, so `CGO_ENABLED=0 go build
  ./glance/... ./theme/` passes. The gallery does not build without cgo and
  does not on `main` either.
- `glance/kwin` keeps its rule that it takes no dependency beyond Fyne:
  `OpacityScript` returns text and `LoadScriptCall`, `RunCall` and
  `UnloadScriptCall` return data, the shape `ReconfigureCall` set.
- The misnamed `ruleApplyInitially` constant is gone rather than corrected in
  place; the name was the bug's cause and keeping it would preserve the trap.

## Phase 5: security review

Independent of this report, run against the working-tree diff.

- **Dependencies**: `govulncheck ./...` (via `make vuln`): no vulnerabilities
  found. `go.mod` and `go.sum` are unchanged — nothing was added.
- **Injection**: `kwin.OpacityScript` interpolates an app ID into JavaScript
  that KWin executes. It is quoted with `%q`, and
  `TestTheOpacityScriptCannotBeEscapedByAnAppID` asserts that an app ID
  carrying a quote cannot close the literal. The first implementation escaped
  by hand *and* with `%q`, which double-escaped and mangled the value; the
  test caught it.
- **cgo**: `opacity_x11.go` opens a display, writes one property on a handle
  Fyne supplies, flushes and closes. No allocation crosses the boundary, the
  opacity is clamped by the caller, and there is no input from outside the
  program.
- **Secrets**: none added; no credentials, no network access, no runtime file
  writes in any library package. The opacity script is written to a file by
  the calling program, not by the library.

## Phase 5.5: release safety

- **Rollback**: revert the commits. Every new identifier is additive and
  `Options.Translucent` defaults to off, so a consumer that asks for none of
  this sees only the margins change.
- **Behaviour change without opt-in**: the card margins, the card gap, the
  flush edge and the muted title apply to every existing glance window. That
  is the point of the spec, it is cosmetic, and `glance-monitor.png` in the
  README shows the result.
- **Behaviour change with opt-in**: a rule installed by `kwin.Install` with a
  non-zero `Opacity` now actually fades the window. A consumer that set an
  opacity and saw no effect will see one after upgrading. That is the fix, and
  the changelog says so.
- No migration, no data, no schema.
