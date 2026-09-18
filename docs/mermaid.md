# Mermaid diagrams

Mermaid is a build-time step, never a runtime dependency. A program that uses
this module never needs node, a browser, or network access to show a diagram.

## Pipeline

1. Diagram sources live next to the document that uses them, either as
   `.mmd` files or as ```` ```mermaid ```` fences inside a Markdown file.
2. `go generate` runs `fynedesygn-mermaid` (in `cmd/`), which finds every
   source under the package directory, hashes each diagram's text
   (`mermaid.Hash`: SHA-256 of the trimmed source, first 16 hex characters),
   and renders the missing ones with `mmdc` (mermaid-cli) to PNG twice: once
   with mermaid's `default` theme and once with `dark`, both on a transparent
   background at 2x scale. Output lands in `diagrams/<hash>-light.png` and
   `diagrams/<hash>-dark.png` beside the sources.
3. The PNGs are committed and embedded with `//go:embed diagrams/*`.
4. At runtime `mermaid.NewSet(fs, "diagrams").Lookup(source, dark)` hashes
   the source and returns the variant matching the active palette's `Dark`
   flag (falling back to the other when only one exists); `mermaid.NewDiagram`
   draws it contained, at natural size divided by the render scale, with a
   minimum height that follows its width. `markdown.Pane` does this for every
   mermaid fence.
5. A diagram whose PNG is missing (source edited without `make generate`)
   renders as a code panel with the source and a caption "Diagram not
   rendered; run make generate". A test in the consuming program can assert
   that every fence has its image, so the gap is caught before a release.

## Why PNG and not SVG

Fyne rasterises SVG with oksvg, which does not support the CSS and
`foreignObject` text that mermaid emits. PNG rendered at 2x scale reads well at
the sizes a document pane uses.

## Why hash-keyed

The document is the source of truth. Keying the image by a hash of the
diagram's text means an edited diagram produces a different file, a stale one
is detected rather than shown, and no author maintains a mapping by hand.

## Requirements for `make generate`

- `mmdc` on the PATH (mermaid-cli 11 or newer). Installation is the
  developer's choice: a global npm install, a distribution package, or a
  container. The module does not install it.
- CI does not run `make generate`; it runs `fynedesygn-mermaid -check`,
  which needs no `mmdc` and fails the build when a source has no image pair
  or an image has no source. Rendering happens on a developer machine.

## Guidance for diagrams in a window

- Keep a diagram to what fits in about 700 x 400 at 1x. A window is not a
  page; a diagram that needs zooming belongs in the README.
- Use flowchart and sequence diagrams; avoid diagram types whose text is
  small at that size (gantt, large class diagrams).
- Colours come from mermaid's `default` and `dark` themes so the diagram
  matches light and dark schemes. Custom `%%{init}%%` theming is allowed but
  must be legible in both variants.
