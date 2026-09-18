# Validation report: spec 003 documents and diagrams

Date: 2026-09-18. Milestone: Markdown rendering and the Mermaid pipeline.
Spec: `specs/003-documents-and-diagrams.md`.

## Scope

`markdown`, `mermaid`, `cmd/fynedesygn-mermaid`, `docs` (embedded documents
and diagrams), the root package's embedded README, `Shell.Scroller()`, the
gallery's Documents section and README-in-About, CI diagram check.

## Phase 3: tests

`go test -race ./...` on Linux, no display: all packages pass.

| Package | Coverage |
|---|---|
| `markdown` | 91.7 % |
| `mermaid` | 87.4 % |
| `cmd/fynedesygn-mermaid` | 74.4 % |
| `cmd/fynedesygn-gallery` | 61.4 % |
| `shell` | 75.7 % |

The ported pane tests run against this module's README (distant blocks out
of the tree, rendered as scrolled to, constant height, re-measure on
narrowing, detach, nil viewport, fences whole, every line kept). The real
`mmdc` render test ran on this machine (mermaid-cli 11.17) and is skipped
where `mmdc` is absent. Canaries: quirk 7 (`TestDocumentRendersOnlyNearTheViewport`),
quirk 21 (`TestRelativeImagesResolveThroughTheFS`), and new quirk 22
(`TestFyneStillDrawsMarkdownTablesInsideAScroll`), found when the README's
package table failed the "nothing takes the wheel" test.

`go vet`, `gofmt`: clean. Module total 6,443 lines of Go.

## Phase 4: code quality

- golangci-lint v2.12.2: 0 issues after fixing eight findings in the new
  code (unchecked print returns in the command, a file read inside a
  directory walk, a 0755 directory).
- The README is embedded once, in the root package; a copy under `docs/`
  was made during implementation and removed before commit.
- `markdown.Section` and the gallery's `aboutDoc` are the two shapes for a
  document in a section; both detach through `shell.Detacher`.

## Phase 5: security review

- `govulncheck -mode binary` on both built commands: no vulnerabilities.
- `mermaid.Render` runs `mmdc` with an argument slice from `Args`, never a
  shell string; the source goes through a temporary file that is removed.
  It is a development-time tool and is not reachable from any library
  package's runtime path.
- `mermaid.Sources` reads only under the root the caller names; `Set`
  reads only from the `fs.FS` it is given.
- No credentials, no network. Secrets grep: clean.

## Phase 5.5: release safety

Nothing consumes these packages yet. Rollback is `git revert`. A stale
diagram cannot ship unnoticed: CI runs the checker.

## Spec reconciliation

Eleven of twelve acceptance criteria verified. AC12's CI part is verified by
the run this commit triggers; the spec says so.

## Gaps

- Inline images inside paragraphs still go through Fyne's URI segment and do
  not resolve relative paths; documented in `docs/markdown.md`.
- Mermaid's `dark` and `default` themes were judged against the palettes in
  the gallery headlessly, not on a display.
