# Spec 019: an unreadable settings file is reported, not renamed

**Issue**: [#15](https://github.com/ushineko/fynedesygn/issues/15)

## Status: COMPLETE

## Context

`settings.Store` renames a file it cannot parse to `<name>.bad` and starts from
defaults. Spec 011 introduced that deliberately (R6, AC5), reasoning:

> Not deleted, because it is the user's, and not left in place, because leaving
> it would mean every save from then on refuses or overwrites it unannounced.

The dilemma is real; the resolution was wrong. A library has no business moving
a user's files, and this one does it as a side effect of *reading* — a caller
that only wanted to load settings mutates the user's configuration directory.
The moment it happens is the worst one available: someone has hand-edited their
settings and mistyped, and the file they are looking for is now somewhere else
under a name they did not choose.

It also throws away the most useful thing available. A JSON or YAML parse error
carries a line and a column; the program can show the user where the mistake is.
Renaming replaces that with "your file is called `.bad` now".

The third option spec 011 did not take is to **refuse, and say why**: leave the
file exactly where it is, report the error with enough context for the caller to
present a cogent reason, and treat the store as read-only until the caller
decides what to do. Silently overwriting a file that could not be parsed is the
same mistake as renaming it, with the evidence destroyed rather than relocated.

Found while building [hotaru](https://github.com/ushineko/hotaru), whose
hand-edited rules file must never be written by the program. It currently reads
that file itself rather than use this package, which is the sort of thing a
design system should notice about itself.

## Requirements

1. A file that does not parse is left untouched. Nothing is renamed, written,
   created or removed in response to a failed load.
2. `Open` returns a typed error carrying the path and the underlying parse
   error, so a caller can show the user where the fault is.
3. A store whose file did not parse refuses to save. `Set` and `Flush` return
   an error naming the reason rather than overwriting a file nobody could read.
4. The caller — never the library — may abandon the unreadable contents
   explicitly, after which the store behaves normally and the next save
   overwrites the file.
5. A store in that state still serves reads: `Get` reports every section as
   absent, so a program runs on its defaults.
6. `Shell` continues to open, reports the error it already collects, and its own
   appearance and layout writes fail quietly rather than crashing a window.

## Acceptance Criteria

- [x] AC1 Opening an unparseable file leaves the directory byte-for-byte as it
      was: the file unchanged, and no `.bad` or other file created (R1).
- [x] AC2 `Open` returns a `*ParseError` whose message names the path and
      includes the codec's own error, and which unwraps to it (R2).
- [x] AC3 `Set` on such a store returns an error naming the file and does not
      schedule a write; `Flush` does the same and writes nothing (R3).
- [x] AC4 `Replace` clears the condition; a subsequent `Set` and `Flush` write
      the file, and reopening it reads what was written (R4).
- [x] AC5 `Get` on such a store reports sections absent rather than failing, so
      a caller runs on defaults (R5).
- [x] AC6 A `Shell` whose settings file does not parse opens, `Report`s the
      error, and does not panic when appearance or splits are written (R6).
- [x] AC7 `Unreadable` reports the condition, so a caller can ask before
      writing rather than discovering it from an error.
- [x] AC8 The spec 011 test asserting the rename is replaced rather than
      adapted, and spec 011's R6/AC5 are marked superseded by this spec.
- [x] AC9 README changelog entry; `make test` and `make lint` pass.

## Executive Summary

`settings` no longer renames a file it cannot parse. The error is reported
through a new `*ParseError` carrying the path and the codec's own complaint, the
file is left where its owner put it, and the store serves reads from defaults
while refusing to save until the caller calls `Replace`. Reviewers should look
at `load` in `settings/settings.go` for what was removed, and at
`TestAFileThatDoesNotParseIsReportedAndLeftWhereItIs` for the guarantee that a
failed read writes nothing at all.

## Testing

`make test` passes (full suite, headless); `make lint` reports 0 issues;
`govulncheck ./...` finds nothing. The spec 011 rename tests in `settings` and
`yamlcodec` were replaced rather than adapted, and a shell-level test covers a
program opening on a settings file it cannot read.

## Risks & Assumptions

- **Behaviour change in a `v0.x` public package.** No consumer outside this
  repository imports `settings` (checked across angou, nmsbonker,
  clockwork-orange and terrariabonker), so the blast radius is the shell, the
  examples and the tests. The changelog names it as a change either way.
- **A program that ignored the `Open` error now silently stops saving** where
  before it saved to a fresh file. That is the intended trade: the old
  behaviour's "success" was built on having moved the user's file aside. The
  error is returned at `Open`, at `Set` and at `Flush`, so there are three
  places to notice it and `Shell` already reports the first.
- **Rollback**: revert the commit. The renaming behaviour returns.

## Alternatives Considered

- Considered keeping the rename but making it opt-in; rejected as a flag that
  would be set once, in one program, and then be dead configuration everywhere
  else.
- Considered loading defaults and allowing saves to overwrite the unreadable
  file; rejected because destroying the file is worse than moving it, which is
  what this spec is about.
