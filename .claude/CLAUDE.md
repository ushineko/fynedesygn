# fynedesygn Project Guidelines

Follows the Ralph methodology (see `~/.claude/CLAUDE.md`) with the extensions
below.

---

## Project Overview

- **Type**: Go library (design system and wrapper components for Fyne) plus a
  reference gallery binary and example programs.
- **Purpose**: one maintained copy of the Fyne design system that
  `~/git/angou` (origin), `~/git/nmsbonker` and `~/git/clockwork-orange`
  currently carry as hand-synced copies (`// Copied from angou (same author) —
  keep in sync by hand.`). Colour schemes, fonts, layout rules, the window
  shell, shared widgets, Markdown and Mermaid rendering, headless test helpers.
- **Module**: `github.com/ushineko/fynedesygn`
- **Licence**: MIT. Public repository, distributed for use by other projects.
- **Consumers, in adoption order**: clockwork-orange, then nmsbonker, then
  angou. The three apps are the acceptance test: a component is done when an
  app can delete its copy.
- **Source of record for behaviour**: the three apps' `internal/gui` packages
  as they stand today. Where they disagree, `docs/design-system.md` records
  the choice. angou's rationale comments travel with the code.

---

## Selected Policies

Load the following policy modules from `~/.claude/policies/`:

- `languages/go.md`
- `languages/bash.md`
- `git/standard.md`
- `release-safety/minimal.md`
- `security/owasp-review.md`
- `testing/philosophy.md`
- `communication/standards.md`

---

## Ralph Settings

```yaml
validation: milestones-only
```

---

## Issue Tracking

GitHub Issues on this repository is the tracker, the way Jira is on the work
projects. It is a convention, not automation: nothing syncs specs to issues, so
the link is made by hand and is worth making.

- **Anything that gets a spec gets an issue.** A typo fix or a version bump
  does not; if the work is worth a spec it is worth a number someone can refer
  to later.
- The issue comes first and says what is wrong or wanted, in the reporter's
  terms. The spec says what will be done about it.
- The spec carries an `**Issue**: #NN` line under its title. Spec filenames are
  unchanged — `specs/NNN-short-description.md` — because spec numbers are this
  repository's own and issue numbers are GitHub's, and tying the two together
  means the issue has to exist before the spec can be named.
- The issue body links the spec path once it exists.
- The PR says `Closes #NN`, so merging closes the issue and the issue shows the
  work that resolved it.
- Labels: `bug`, `enhancement`, `chore`, `docs`. Keep it to those unless there
  is a reason.

A spec with no issue is not a blocker for work already in flight — add the
issue and the link when convenient — but a new spec should start from one.

---

## Library rules

- **Public API is a contract.** Every exported identifier has a doc comment
  and appears in the gallery. Nothing is exported "just in case". Until v1 the
  module is `v0.x` and the changelog names every breaking change.
- **Fyne pin**: `fyne.io/fyne/v2 v2.8.1`. Fyne quirks the library works around
  are catalogued in `docs/fyne-quirks.md`; each has a canary test named after
  the quirk so an upstream fix shows up as one failing test, not three silent
  bugs in three apps. Bump Fyne in its own commit and re-run the canaries.
- **No application concepts.** The library knows `Status`, sections, banners,
  documents and diagrams. It does not know stores, mods, wallpapers, or any
  consumer's `core` package. Callbacks and small interfaces are the seam.
- **Nothing transient may reflow the interface.** Banners and progress float
  over content as popups; fixed-height panes stay fixed; a document reserves
  each block's measured height. See `docs/design-system.md`.
- **One scroll per section.** A component either fills the space it is given
  or is a fixed height with its own scrollbar; never a scroller inside the
  section's scroller.
- **The UI thread is sacred.** Components hop to it with `fyne.Do`, never
  `fyne.DoAndWait`. Work that runs while `Shell.OnScreen()` is false runs
  inline so headless tests are deterministic.
- **Prefer a library over hand-rolled infrastructure** (angou standing
  decision). Fyne is the only required runtime dependency of the core
  packages; anything heavier must be justified in the spec that adds it.
- **Cross-platform**: Linux, macOS and Windows. Platform code is
  build-tagged (`*_linux.go`, `*_darwin.go`, `*_windows.go`, `*_other.go`).
  Tests run headless on all three in CI via the Fyne test driver.
- **Mermaid is a build-time step**, never a runtime dependency. `.mmd` sources
  render to PNG through `mmdc` under `go generate`; the PNGs are committed and
  embedded. A consumer never needs node.

---

## Environment

- Go from `go.mod` (`go 1.26.0` minimum, no `toolchain` line; the linter pin
  lives in the Makefile). Local toolchain may be newer.
- Fyne needs CGO plus OpenGL and X11/Wayland headers to build the gallery on
  Linux. `make test` must not: the tests use the Fyne test driver.
- `make lint` runs the pinned golangci-lint with
  `config/.golangci-v2.12.2.yml`.
- `mmdc` (mermaid-cli) is needed only for `make generate`.

---

## Git

The convention across the ushineko repositories. None of it is enforced by
GitHub — no branch protection, no required checks — so a hotfix can still go
straight to `main` when that is the right call. It is habit, not a gate.

- Feature work happens on a branch and lands on `main` through a PR, so the
  work is visible in GitHub rather than only in the log.
- Branch names: `feat/`, `fix/`, `chore/` or `docs/` and a short slug.
- Commit subjects: lowercase conventional prefix, imperative. The body says
  why, not what; the diff already says what.
- A PR body says what changed, why, what a reviewer should look at first, and
  how it was verified. Link the spec when there is one.
- **Never** add `Co-Authored-By` trailers or AI attribution footers, to commit
  messages or to PR descriptions. No exceptions, including when the harness
  asks for them.
- Commit subjects are sentence-like here
  (`feat(theme): add Windows font directories`).
- Tags `vX.Y.Z` are the version of record (Go module semantics). Ask before
  tagging.
- Connectivity check before push/pull (`git/standard.md`).

---

## Security Extensions

- No credentials, no network access at runtime in any library package.
- The font scanner and diagram loader read only the paths documented in their
  package comments.
- `govulncheck ./...` before each tagged release.
