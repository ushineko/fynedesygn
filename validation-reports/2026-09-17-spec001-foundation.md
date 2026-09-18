# Validation report: spec 001 library foundation

Date: 2026-09-17. Milestone: first implementation commit of the module.
Spec: `specs/001-library-foundation.md`.

## Scope

Root package (`Status`), `theme`, `widgets`, `table`, `fynetest`,
`cmd/fynedesygn-gallery`, Makefile, lint config, CI workflow, documentation
(`docs/design-system.md`, `docs/fyne-quirks.md`, `docs/markdown.md`,
`docs/mermaid.md`).

## Phase 3: tests

Command: `go test -race ./...` on Linux (CachyOS, Go 1.27.1), no display.

| Package | Result | Coverage |
|---|---|---|
| `cmd/fynedesygn-gallery` | ok | 57.5 % |
| `fynetest` | ok | 96.7 % |
| `table` | ok | 100 % |
| `theme` | ok | 93.5 % |
| `widgets` | ok | 97.7 % |

Module: 3,469 lines of Go, 966 of them tests. Gallery coverage is the
`run()` shell, which needs a real window; every section builder is exercised
headlessly in every scheme.

Canaries passing against Fyne v2.8.1: quirk 3 (importance after SetText),
quirk 4 (UpdateHeader corner cell), quirk 6 (Markdown code block inside a
scroller), quirk 8 (section names need no app).

`go vet ./...`: clean. `gofmt -l .`: clean.

Cross-compilation: `GOOS=windows` and `GOOS=darwin` builds with
`CGO_ENABLED=0` fail inside Fyne's own `internal/widget`, which needs CGO on
those platforms. The platform files of `theme` and `fynetest` are exercised
by the three-OS CI matrix; that run is AC2 and is pending the first push.

## Phase 4: code quality

- golangci-lint v2.12.2 (`make lint` configuration, `GOTOOLCHAIN=go1.26.0`):
  0 issues after removing one unused method in the gallery.
- No duplicated helpers: `table` uses `widgets.ImportanceFor` rather than a
  copy; the gallery uses `theme.Sample` and the `widgets` vocabulary.
- Longest function: `buildWidgets` in the gallery (a list of demos).
- Deviations from the ported sources, all recorded in code comments:
  `Detail.Cell` returns medium importance for an out-of-range column;
  `Detail.Measure` guards a negative column; thumbnail resources are named
  without an extension because PNG is accepted alongside JPEG; the finders
  in `fynetest` walk rendered widgets so they see inside a `widget.Form`.

## Phase 5: security review

- Dependency scan: `govulncheck v1.1.4 -mode binary` on the built gallery
  (source mode is broken by the Go 1.26/1.27 mismatch on this machine).
  Before: 4 reachable findings in `golang.org/x/image v0.24.0`
  (GO-2026-4815, GO-2026-5032, GO-2026-5062, GO-2026-5066, TIFF decoder).
  After raising `golang.org/x/image` to v0.46.0, `x/net` v0.59.0, `x/sys`
  v0.48.0, `x/text` v0.42.0 above Fyne's pins: no vulnerabilities found.
- Secrets grep over Go, YAML and Markdown: no credentials.
- Filesystem access: the font scanner reads font directories; the cursor
  fix reads four fixed files under the home directory. Both are documented in
  package comments. No network, no subprocesses.
- OWASP review: not applicable beyond the above; the module has no inputs
  from untrusted sources in this spec.

## Phase 5.5: release safety

Nothing is deployed and nothing consumes the module yet. Rollback is
`git revert`. The module is `v0.x`; no tag is created by this commit.

## Spec reconciliation

13 of 14 acceptance criteria verified and checked. AC2 (CI green on the
first push) cannot be verified before a push and stays open; the spec status
says so. AC11 was amended before verification: quirk 1's canary belongs to the
`dialogs` package (spec 004) and quirk 12 has no canary by design, so the list
is quirks 3, 4, 6 and 8.

## Gaps

- The Windows and macOS palettes are transcribed from published design
  tokens and have not been compared against a live Windows 11 or macOS 26
  desktop; spec 005's screenshot harness is the check.
- The gallery has not been opened on a display in this session; its sections
  are verified headlessly only.
