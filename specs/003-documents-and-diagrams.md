# Spec 003: Documents and diagrams

> **Note**: This work has no associated issue tracker ticket. The repository
> is a personal public project without an issue tracker.

## Status: COMPLETE

## Executive Summary

Adds the `markdown` package (the document pane ported from clockwork-orange,
extended with pipe tables drawn as a grid, whole-block images from an
`fs.FS`, mermaid fences and a `Section` helper), the `mermaid` package
(hash-keyed light and dark PNG lookup, the diagram widget, and the `mmdc`
renderer and checker behind `go generate`), the `fynedesygn-mermaid`
command, the embedded `docs` package with the first rendered diagram, and
the gallery's Documents and README-in-About pages. Reviewers should look
first at `markdown/pane.go` `RenderBlock` (the block dispatch) and
`mermaid/set.go` (the hash contract).

## Context

clockwork-orange's spec 011 built `markdownPane` after its About section
scrolled in fits and bursts: one `RichText` over a 44-block README repainted
250 segments per wheel notch, and Fyne 2.8 wrapped every indented command in
a horizontal scroller that took the wheel. The pane renders per block, only
near the viewport, reserving measured heights, and draws code blocks itself.
It is the only Markdown renderer among the three apps and ports as is.

Mermaid is new. The user asked for diagrams converted to images and embedded
in the program. `docs/mermaid.md` fixes the pipeline: `mmdc` at development
time, hash-keyed light and dark PNG pairs, committed and embedded, never a
runtime dependency. A trial render on this machine took 0.7 s per diagram
and produced a 1300 x 556 PNG at 2x scale.

Fyne's Markdown image segment takes a URI and cannot read from an `fs.FS`,
so relative images in an embedded document are drawn by the pane itself when
a block is a single image (`![alt](path)`), which is how READMEs use them.

## Requirements

### R1. `markdown` package

- R1.1 `Pane` (widget): `New(src string, o Options) *Pane`, `Follow(sc)`,
  `Detach()`, `Blocks() int`, `Live() int`; `Options{FS fs.FS; Diagrams
  *mermaid.Set}`. Behaviour as clockwork-orange: blocks split on blank lines
  with fences kept whole; measured spacers; half-viewport overscan; nil
  scroll renders everything; `Resize` re-measures on width change.
- R1.2 `Blocks(src) []string`, `CodeBlock(src) (lang, code string, ok bool)`
  recognising fenced (with info string) and indented code, `ImageBlock`, and
  `TableBlock` (added during implementation: Fyne renders a pipe table in a
  scroller, quirk 22, so the pane draws tables as a grid).
- R1.3 `CodePanel` widget wrapping code on an input-coloured panel;
  `Refresh` repaints from the active scheme.
- R1.4 Block rendering: a ```` ```mermaid ```` fence looks the diagram up in
  `Options.Diagrams` by hash and the active palette's darkness; missing
  diagram or nil set renders the source in a `CodePanel` under a dim caption
  "Diagram not rendered; run make generate". An image-only block with a
  relative path reads the bytes from `Options.FS` and draws a contained
  `canvas.Image` with the file's natural size at 1x capped to the pane width;
  a missing file renders the alt text dimmed. Other blocks render through
  `widget.NewRichTextFromMarkdown`.
- R1.5 `Section(title string, icon func() fyne.Resource, src string, o
  Options) shell.Section`: a section that is one document following the
  shell's content scroller (`Shell.Scroller()`), detaching on replacement.
  Add `Scroller()` to `shell.Shell`.

### R2. `mermaid` package

- R2.1 `Hash(src) string`: SHA-256 of the source with surrounding whitespace
  trimmed and line endings normalised, first 16 hex characters.
- R2.2 `Set`: `NewSet(fsys fs.FS, dir string)`, `Lookup(src string, dark
  bool) (fyne.Resource, bool)` reading `<dir>/<hash>-dark.png` or `-light.png`
  and falling back to the other variant when one is missing; `Has(src)`.
- R2.3 `Diagram` widget: `NewDiagram(res fyne.Resource, scale float32)`
  draws the PNG contained, with a minimum height derived from the image's
  natural size divided by `scale` and capped to the width it is given.
- R2.4 `Sources(root string) ([]Source, error)`: every ```` ```mermaid ````
  fence in `*.md` files and every `*.mmd` file under root, skipping
  `diagrams/` and hidden directories; `Source{File string; Line int; Text
  string}`.
- R2.5 `Render(ctx, mmdc, src, outDir string) error`: writes both variants
  with `-t dark`/`-t default`, `-b transparent`, `-s 2`, `-q`, through a
  temporary input file; argument slice, no shell. `Args(...)` is the pure
  function that builds the arguments, for tests.
- R2.6 `Check(root, outDir string) (missing []Source, stale []string, err
  error)`: sources without a PNG pair, and PNGs no source refers to.

### R3. `cmd/fynedesygn-mermaid`

- R3.1 Flags: `-root` (default `.`), `-out` (default `<root>/diagrams`),
  `-mmdc` (default `mmdc`), `-check` (report missing and stale, exit 1 on
  any, render nothing), `-prune` (delete stale PNGs after rendering).
- R3.2 Default run renders only the missing pairs and prints what it did.
- R3.3 `go generate` directive in the `docs` package runs it for `docs/`.

### R4. `docs` package and CI

- R4.1 Package `docs` (directory `docs/`) embeds `*.md` and `diagrams/*.png`
  and exports `FS`. `docs/markdown.md` gains a mermaid flowchart of the
  rendering pipeline; its PNG pair is committed.
- R4.2 Makefile `generate` runs `go generate ./...`; CI's lint job runs
  `go run ./cmd/fynedesygn-mermaid -check -root docs` so a stale or missing
  diagram fails the build without needing `mmdc` on the runner.

### R5. Gallery

- R5.1 A Documents section built with `markdown.Section` showing
  `docs/markdown.md` with its diagram rendered from `docs.FS`.
- R5.2 The About section's `Extra` shows the module README through a `Pane`
  following the shell's scroller.

### R6. Canaries and documentation

- R6.1 Quirk 7 canary `TestDocumentRendersOnlyNearTheViewport` (the port's
  first test); quirk 21 canary `TestRelativeImagesResolveThroughTheFS`.
  Quirk 6's canary already lives in `fynetest`.
- R6.2 `docs/markdown.md` and `docs/mermaid.md` updated to the final API
  names; README table gains `mermaid` details and the `docs` package.

## Acceptance Criteria

- [x] AC1 The ported pane tests pass against this module's README: distant
  blocks out of the tree, blocks rendered as scrolled to, constant height,
  re-measure on narrowing, detach releases the scroll, nil viewport renders
  all, fences kept whole, every line kept (R1.1, R1.2).
- [x] AC2 `CodeBlock` returns the fence language for ```` ```mermaid ````
  and ```` ```sh ````, empty for indented code, false for prose and lists
  (R1.2).
- [x] AC3 No block of the README or of `docs/markdown.md` renders anything
  scrollable (`fynetest.ScrollableIn`) (R1.3, R1.4).
- [x] AC4 A mermaid fence with a matching PNG in the set renders a `Diagram`
  whose resource is the dark variant under a dark palette and the light one
  under a light palette; with no PNG it renders a `CodePanel` and the caption
  (R1.4, R2.2).
- [x] AC5 An image-only block resolves through the FS and renders an image;
  a missing file renders the alt text; canary
  `TestRelativeImagesResolveThroughTheFS` (R1.4).
- [x] AC6 `Hash` is stable across trailing whitespace and CRLF (R2.1).
- [x] AC7 `Sources` finds fences in `.md` and `.mmd` files with correct line
  numbers and skips `diagrams/` (R2.4).
- [x] AC8 `Args` produces the documented `mmdc` argument slices; `Render`
  against a real `mmdc` (skipped when absent) writes both PNGs decodable by
  `image/png` (R2.5).
- [x] AC9 `Check` reports a source without PNGs as missing and a PNG without
  a source as stale; the command's `-check` exits 1 in that case and 0 when
  clean (R2.6, R3.1).
- [x] AC10 `go run ./cmd/fynedesygn-mermaid -check -root docs` passes on the
  committed tree; `docs.FS` contains the PNG pair for the flowchart (R4).
- [x] AC11 The gallery's Documents and About sections render headlessly;
  `fynetest.Text` of Documents contains a heading from `docs/markdown.md`
  (R5).
- [x] AC12 `make test`, `make lint`, `go vet` clean; CI green including the
  new check step (R4.2). _CI run for commit 1377636: all four jobs green._

## Risks & Assumptions

- **Assumption**: hash keys are stable enough that an edited diagram shows
  the "not rendered" caption until `make generate` runs, and the CI check
  catches a forgotten run. This is the designed failure mode.
- **Assumption**: mermaid's own `dark` and `default` themes are legible on
  the dark and light palettes; custom theming is left to authors.
- **Risk**: `mmdc` renders through a headless Chromium; the CI does not run
  it. Rendering is a developer-machine step, as `docs/mermaid.md` says.
- **Risk**: image-only blocks only. Inline images inside a paragraph still
  go through Fyne's URI segment and will not resolve relative paths; the
  limitation is documented in `docs/markdown.md`.
- **Rollback**: `git revert`; nothing consumes these packages yet.

## Alternatives Considered

- Considered rendering mermaid at runtime through a bundled JavaScript engine
  or a Go port; rejected, no mature pure-Go renderer exists and a browser
  dependency in a desktop program is out of the question.
- Considered SVG output; rejected because mermaid's SVG uses CSS and
  `foreignObject` text that Fyne's rasteriser does not support.
- Considered a custom `fyne.URI` scheme so Fyne's image segment could read
  from the FS; rejected because `canvas.NewImageFromURI` resolves through
  Fyne's storage repositories, which would mean registering a global
  repository per document.
