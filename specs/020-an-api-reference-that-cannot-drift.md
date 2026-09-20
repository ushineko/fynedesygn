# Spec 020: an API reference that cannot drift

**Issue**: [#17](https://github.com/ushineko/fynedesygn/issues/17)

## Status: INCOMPLETE

## Context

The module documents everything except its public surface. `README.md` names
each package and says in one sentence what it is for; `docs/design-system.md`
and `docs/glance.md` say what the components must do; `docs/fyne-quirks.md`
says what upstream made them do. Nothing lists what a caller can actually
write. The answer lives in the source, or on pkg.go.dev, which needs a network
and a published tag and shows one package at a time.

The library's own rule is that the public API is a contract: every exported
identifier has a doc comment, appears in the gallery, and nothing is exported
"just in case" (`.claude/CLAUDE.md`, Library rules). The doc-comment half of
that is unchecked. A scan finds six exported identifiers without one today —
all interface-method implementations — so the gap is not the problem. The
seventh arriving silently is, and so is the fact that a PR which widens the
public surface of a `v0.x` module shows that nowhere a reviewer looks.

This repository has already paid for the other way of doing it. Spec 018 exists
because the README drifted twenty-four releases while nothing read it, and the
fix was not to write more carefully but to make a test read it. A hand-written
API guide would go stale the same way and for the same reason. So the reference
is generated from the doc comments — the source of truth — and a canary fails
when the committed file and the code disagree, in the same spirit as
`readme_test.go` and `cmd/fynedesygn-mermaid -check`.

What is generated is an index, not a transcript: the package synopsis and each
exported identifier with its signature and the first sentence of its
documentation. `go doc -all` across the module is about 3,500 lines of prose
that pkg.go.dev already renders better, and a diff that large per release is a
diff nobody reads. A signature index is small enough that a reviewer sees an
added export at a glance, which is the reason for having it in the repository
at all.

## Requirements

1. `docs/api.md` is a generated reference: one section per public package, in
   import-path order, carrying the package synopsis and the exported
   identifiers with their signatures and first documentation sentence.
   Constants, variables, types (with their methods) and functions are grouped,
   and a type's constructor is listed with the type rather than alphabetically
   apart from it.
2. Package `main` is excluded. The gallery, the mermaid helper and the examples
   export nothing a caller can call, and the README already describes them.
3. Generation is a `go generate` step that needs no CGO, no network and no
   `mmdc`, so it runs anywhere `make test` runs.
4. A test fails when `docs/api.md` and the code disagree, naming what changed,
   and the fix is to re-run the generator. The test regenerates in memory; it
   does not shell out to the generator binary.
5. A test fails when an exported identifier in a public package has no doc
   comment, naming the identifier and its package. The six that exist today are
   given doc comments rather than an exemption list.
6. The reference is a document like the others: embedded through `docs.FS`,
   linked from the README's Documentation list, and readable in the gallery's
   Documents section.
7. The generator and its shared code are repository plumbing, not public API:
   nothing new is exported from a package a consumer imports.

## Acceptance Criteria

- [ ] AC1 `docs/api.md` exists, is committed, and contains a section for every
      non-`main` package in `go list ./...` with that package's synopsis and
      its exported identifiers' signatures (R1, R2).
- [ ] AC2 Running the generator on a clean checkout rewrites `docs/api.md`
      byte-for-byte identically — it is deterministic, and `git status` is
      clean afterwards (R1).
- [ ] AC3 The generator runs under `go generate` with CGO disabled and no
      network, and `make generate` does not gain an `mmdc` dependency it did
      not have (R3).
- [ ] AC4 Adding an exported identifier and running `make test` fails the
      staleness canary with a message naming that identifier; regenerating
      makes it pass (R4).
- [ ] AC5 Removing a doc comment from an exported identifier fails the
      doc-comment canary with a message naming the identifier and its package;
      restoring it makes it pass (R5).
- [ ] AC6 The six exported identifiers that have no doc comment today have one,
      and the canary passes on `main` with no exemptions (R5).
- [ ] AC7 `docs.FS` serves `api.md`, the gallery's Documents section can show
      it, and it renders in the Markdown pane without a block the pane cannot
      draw (R6).
- [ ] AC8 `go doc` on every package a consumer imports lists no identifier that
      did not exist before this spec (R7).
- [ ] AC9 README: a row in "What is in it" for each new package directory, the
      `docs/api.md` link in the Documentation list, a line in the Development
      block for any new make target, and a changelog entry under
      `### Unreleased` (R6). The existing README canaries enforce the first
      three.
- [ ] AC10 `make test` and `make lint` pass.

## Risks & Assumptions

- **A generated file in the tree is a merge-conflict surface.** Two branches
  that both add an export both rewrite `docs/api.md`. The conflict is real but
  cheap: regenerating resolves it, and the canary fails if anyone resolves it
  by hand and gets it wrong. Accepted for the same reason the rendered mermaid
  PNGs are committed.
- **The canary must not be relaxable.** A test that can be silenced with an
  exemption list becomes an exemption list. If a future identifier genuinely
  cannot carry a doc comment, that is a spec, not a map entry.
- **Parsing, not building.** The generator reads the source with `go/parser`
  and `go/doc` rather than loading packages, so it needs no build and no CGO.
  The assumption is that doc-comment extraction from the AST matches what
  pkg.go.dev shows; the deterministic-output criterion (AC2) is what catches a
  mismatch in practice.
- **Rollback**: revert the commit. `docs/api.md` and both canaries go with it;
  nothing a consumer imports changes, so no consumer is affected.

## Alternatives Considered

- Considered a hand-written narrative API guide; rejected because it is the
  exact failure mode spec 018 was written about — nothing would build it and no
  test would read it.
- Considered committing `go doc -all` output verbatim; rejected as ~3,500 lines
  that duplicate pkg.go.dev and produce a per-release diff nobody reads. The
  index is the part a reviewer can use.
- Considered leaving the reference to pkg.go.dev and shipping only the
  doc-comment canary; rejected because pkg.go.dev needs a network and a
  published tag, and shows one package at a time — and because the API change
  would still not appear in the PR diff, which is half the point.
- Considered generating in CI only, like `cmd/fynedesygn-mermaid -check`;
  rejected because a failure that only CI can see is a failure found late. The
  canary is a test, so `make test` finds it before the push.
