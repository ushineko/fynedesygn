# Mermaid diagrams

Mermaid is a step in the build and never a dependency at run time. A program
that uses this module does not need node, a browser or network access to show
a diagram.

## Pipeline

1. Diagram sources are beside the document that uses them. A source is either
   a `.mmd` file or a ```` ```mermaid ```` fence in a Markdown file.
2. `go generate` runs `fynedesygn-mermaid`, in `cmd/`. The program finds every
   source below the package directory and makes a hash of the text of each
   diagram. The hash is `mermaid.Hash`: the SHA-256 of the trimmed source,
   first 16 hexadecimal characters.
3. `fynedesygn-mermaid` then renders each diagram that has no image. It uses
   `mmdc`, which is mermaid-cli, and makes two PNG files: one with the
   mermaid `default` theme and one with `dark`. Both have a transparent
   background and a scale of 2. The files are `diagrams/<hash>-light.png` and
   `diagrams/<hash>-dark.png`, beside the sources.
4. Commit the PNG files and embed them with `//go:embed diagrams/*`.
5. At run time, `mermaid.NewSet(fs, "diagrams").Lookup(source, dark)` makes
   the hash of the source and returns the file that agrees with the `Dark`
   flag of the active palette. If only one file exists, it returns that one.
   `mermaid.NewDiagram` draws the image at its natural size divided by the
   render scale, and gives it a minimum height that follows its width.
   `markdown.Pane` does this for every mermaid fence.
6. A diagram whose PNG is absent, because somebody edited the source without
   `make generate`, draws as a code panel. The panel shows the source and the
   caption "Diagram not rendered; run make generate". A test in your own
   program can assert that every fence has its image, which finds the problem
   before a release.

## Why PNG and not SVG

Fyne draws SVG with oksvg, which does not support the CSS and `foreignObject`
text that mermaid writes. A PNG rendered at a scale of 2 is legible at the
sizes that a document pane uses.

## Why the hash

The document is the source of truth. The image is keyed by a hash of the text
of the diagram, so an edited diagram makes a different file. The module finds
an image that is out of date instead of showing it, and no author keeps a list
of names.

## What `make generate` needs

- `mmdc` on the PATH, from mermaid-cli 11 or later. Install it how you prefer:
  a global npm install, a package from your distribution, or a container. This
  module does not install it.
- CI does not run `make generate`. It runs `fynedesygn-mermaid -check`, which
  does not need `mmdc`. The check fails the build when a source has no pair of
  images, or an image has no source. Rendering is done on a developer machine.

## How to make a diagram for a window

- Keep a diagram to approximately 700 x 400 at a scale of 1. A window is not a
  page, and a diagram that you must magnify belongs in the README.
- Use flowchart and sequence diagrams. Do not use the types whose text is
  small at that size, such as gantt and large class diagrams.
- Colours come from the mermaid `default` and `dark` themes, so a diagram
  agrees with the light and dark schemes. You can set your own theme with
  `%%{init}%%`, but the diagram must be legible in both variants.
