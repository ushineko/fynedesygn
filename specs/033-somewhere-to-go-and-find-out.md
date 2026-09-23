# Spec 033: somewhere to go and find out

**Issue**: [#55](https://github.com/ushineko/fynedesygn/issues/55)

## Status: COMPLETE

## Context

The API documentation is written and unreachable.

Every package carries a package comment. Every exported identifier carries a
doc comment but two -- `settings.ParseError`'s `Error` and `Unwrap` -- and the
twenty-seven others a scan turns up are renderer methods on unexported types,
which godoc never shows. That part of the house is in order, and it is in
order because the library rule says so: "every exported identifier has a doc
comment and appears in the gallery."

What the README does not contain, anywhere, is the string `pkg.go.dev`.

So the front page describes fifteen packages -- what each is *for*, in a
sentence apiece -- and offers no way to find out what is *in* any of them. A
reader who wants the signature of `shell.Options` has to guess the URL or
clone the repository. The reference exists and nothing links to it.

### What bubbletea does

Its README is largely a **tutorial**: a walkthrough that builds a working
program, with the Go Docs link beside it. The reference is generated and
linked; the README's job is to get somebody to the point where the reference
is useful.

fynedesygn has the opposite shape. It has an excellent reference nobody can
find, five complete examples listed in a table, and nothing that shows what a
program built on this actually looks like. "Using it" is `go get` and a note
about CGO.

### Three things, none of them prose about the API

**A badge and a table of links.** The Go Reference badge at the top, and every
package in "What is in it" linked to its own reference page. That turns a
table which already exists into the API index -- the sentence says what the
package is for, the link says what is in it, and nothing new has to be
written or kept in step.

**A tutorial over code that already runs.**
`examples/document-viewer/main.go` is forty-four lines, is built by
`make build-examples`, and has a headless test. Walking through *that* rather
than a new snippet means the tutorial cannot quietly stop compiling: the
thing it describes is in CI.

A new "hello window" smaller than it was the alternative and was rejected.
Untested code in a README is how a README comes to lie, and this repository
has a suite of canaries that exists because the front page drifted
twenty-four releases once.

**A canary for the links.** `TestEveryPackageIsInTheReadme` already fails when
a package is missing from the table. The same test should fail when a package
is in the table without a reference link, for the same reason and in the same
place.

## Requirements

**R1. The README carries a Go Reference badge.**

**R2. Every package in the table links to its reference.**

**R3. A test fails when a package is in the table without one.**

**R4. A tutorial walks through a program that is built and tested**, and names
the file so a reader can open it.

**R5. The two undocumented exported identifiers get comments.**

**R6. The Contents list and the heading structure stay in step**, which the
existing canary already enforces.

## Acceptance Criteria

- [x] AC1. The README contains a Go Reference badge linking to the module's
      reference page.
- [x] AC2. Every package in "What is in it" carries a link to its own
      reference page, and the test fails when one does not.
- [x] AC3. The tutorial names `examples/document-viewer/main.go` and the code
      it shows is that file's, not a paraphrase.
- [x] AC4. `settings.ParseError`'s `Error` and `Unwrap` are documented, and a
      scan for undocumented exported identifiers on exported types comes back
      empty.
- [x] AC5. The Contents list matches the headings, asserted by the existing
      canary.
- [x] AC6. Full suite and lint clean.

## Alternatives Considered

- **A new, smaller "hello window" for the tutorial.** Rejected: new code in a
  README needs its own test and its own row in the examples table, or it goes
  stale. `document-viewer` is already both.
- **Per-package doc.go overviews with runnable Examples.** Worth having and
  not this: it is a lot of writing across fifteen packages, and none of it
  helps until somebody can find the reference at all. This spec is the
  signposting; that would be the next one.
- **A docs site.** No. The reference is generated from the source and hosted
  already; the gap was a link.

## Risks & Assumptions

- **A table of fifteen links is fifteen chances to rot.** Which is what AC2's
  test is for: the link is checked in the same place the package name already
  is, so a package added without one fails the same way a package added
  without a row does.

- **The tutorial quotes code.** A quotation can drift from its source even
  when the source still compiles. The test names the file rather than
  comparing the text, so the guarantee is "this file exists and is built",
  not "these lines match" -- which is honest, and the reason the tutorial
  walks through a *short* file.

- **The tutorial cannot use bold around a code span.** Writing
  `**`+"`Build`"+`**` in the README crashed the markdown pane: Fyne's own
  Markdown parser emits a segment that is bold *and* monospace, its painter
  finds no face for that combination in the test theme, and dereferences the
  nil. Found by this spec's own prose, reported separately (#56). The
  workaround here is to write one or the other, which reads better anyway.

- **Rollback** is a revert. Nothing outside the README, one test and two
  comments changes.
