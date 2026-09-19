# Spec 018: A README that cannot go stale

## Status: COMPLETE

- **Issue**: #11
- **Priority**: Medium
- **Estimated Complexity**: Low
- **Branch**: `feat/readme-canaries-and-sparkline-pool`
- **Found by**: the maintainer, reading the front page

## Context

Two things the module was doing quietly. Neither would have been reported by a
build, a test or a linter, and one of them was noticed only because someone
read the page.

### The README had drifted twenty-four releases

The **Version** line said `0.1.3`; the latest tag is `v0.1.27`. The `settings`
and `settings/yamlcodec` packages had never reached the "What is in it" table
although they have shipped since 0.1.10. Four gallery screenshots — About,
Dialogs, Fonts and Log — sat in `docs/img/` displayed nowhere, which means
their alt text had never been checked against the image, and alt text is the
only description a screen-reader user gets. `docs/glance.md` was not in the
Documentation list. Four `make` targets were not in the Development block. Two
specs' worth of work had no changelog entry.

The cause is structural rather than careless: **nothing builds the README and
nothing read it.** Every other kind of drift in this module is caught by
something. A Fyne behaviour change fails a canary. A stale mermaid diagram
fails `make check-diagrams`. A section that breaks the affixing rule fails
`fynetest.ScrolledButtons`. The front page had no such thing, so it drifted
until a reader found it — the worst way to find out, because the reader is the
person it exists for.

### A sparkline allocated 118 objects per sample

`glance.Sparkline` rebuilt its `canvas.Line` segments on every refresh and
`Add` refreshes on every sample, so two traces over sixty samples allocated 118
objects per reading and made the cost of drawing a plot proportional to how
often it was fed. At the monitor's five-second cadence this is invisible; the
point is that it scales the wrong way, and a glance program polling faster
would pay for it in garbage rather than in drawing.

## Requirements

- R1 The README brought back into line with the repository.
- R2 A written rule that it is updated in the same commit as the change, and
  what tagging requires.
- R3 Canary tests that fail when the repository and its front page have
  drifted, rather than relying on the rule being remembered.
- R4 `glance.Sparkline` reuses its segments, with a benchmark and an allocation
  test pinning it.

## Acceptance Criteria

- [x] AC1 The **Version** line names the latest tag; the `settings` packages
      are in the package table; `docs/glance.md` is in the Documentation list;
      `make build-examples`, `screenshots`, `coverage` and `vuln` are in the
      Development block; the four orphaned screenshots are displayed with alt
      text; the glance work has a changelog entry. (R1)
- [x] AC2 `.claude/CLAUDE.md` carries "The README is not optional (mandatory)":
      what must be updated in the same commit, and the four steps tagging
      requires. (R2)
- [x] AC3 Six canaries in `readme_test.go`, each verified to fail on the drift
      it is for and to pass on the real file. (R3)
- [x] AC4 The canaries look in the right section of the README rather than
      anywhere in it. The first version of `TestEveryPackageIsInTheReadme` was
      **green with the `settings` row deleted**, because `settings` is also an
      example and the name turned up in the other table. A canary that cannot
      fail is worse than none, so each was checked by breaking the thing it
      guards. (R3)
- [x] AC5 `BenchmarkSparklineRefresh`: 2397 ns and 118 allocations before, 357
      ns and none after. `TestRefreshingAPlotAllocatesNothing` pins the
      allocation count and `TestAShrunkPlotDoesNotDrawItsOldSegments` pins that
      the pool is reused rather than merely cleared. (R4)
- [x] AC6 `go test -race ./...` passes headless and `make lint` reports 0
      issues.

## Risks & Assumptions

- **The canaries check presence, not accuracy.** They fail when a package is
  missing from the table; they cannot tell whether its description is still
  true, or whether a screenshot's alt text matches the image. Those stay a
  human's job, and the rule in `.claude/CLAUDE.md` is what covers them.
- **The version canary cannot check the tag.** A CI checkout has no tags, so
  the test compares the Version line with the newest changelog heading instead.
  The rule that both match the tag is applied when tagging.
- **`plumbingTargets` is an allowlist** of `make` targets the README
  deliberately omits (`help`, `install-lint`, `clean`). Adding to it is how a
  future target opts out, and doing so to silence a failure rather than because
  the target is plumbing is the way this test gets defeated.
- **The label vocabulary in `.claude/CLAUDE.md` does not match the
  repository.** It names `bug`, `enhancement`, `chore` and `docs`; GitHub has
  `bug`, `enhancement` and `documentation`, and no `chore`. Issue #11 was filed
  as `documentation`. Not fixed here because it is a choice between creating
  two labels and amending the convention, and that is the maintainer's.
- **Rollback** is a revert. The sparkline change is behavioural and covered by
  three tests; everything else is documentation and tests.

## Alternatives Considered

- *A rule in `.claude/CLAUDE.md` and nothing else.* Rejected as insufficient on
  its own: the README reached 0.1.3-against-v0.1.27 under a convention that
  already said releases get changelog entries. A rule that depends on being
  remembered is how this happened.
- *Generate the package table from `go list`.* Rejected: the table's value is
  the one-line purpose beside each name, which cannot be generated, and a
  half-generated table invites edits that the generator then discards.
- *Fail the build rather than a test.* The tests run in CI on three platforms
  already, so a failing test is a failing build; adding a separate check would
  be a second thing to keep current.

## Executive Summary

The README had drifted twenty-four releases without anything noticing, because
nothing builds it and no test read it. This brings it back into line, writes
the rule down in `.claude/CLAUDE.md`, and — because a rule that must be
remembered is what failed — adds six canaries that fail when the repository and
its front page disagree. Each was verified by breaking the thing it guards;
one of them was blind on the first attempt and was caught that way.

Also fixes the sparkline allocating 118 `canvas.Line` objects per sample: 2397
ns and 118 allocations per refresh before, 357 ns and none after.

Reviewers should look at `readme_test.go` first, and specifically at whether
`plumbingTargets` and the section-scoped lookups are the right shape — that is
where these tests will be defeated if they ever are.
