/*
Package mermaid embeds Mermaid diagrams in a Fyne program as images rendered
ahead of time.

Mermaid is a build-time step, never a runtime dependency. Diagram sources
(```mermaid fences in Markdown files, or .mmd files) are rendered by mmdc
(mermaid-cli) under go generate to a light and a dark PNG named by a hash of
the diagram text; the PNGs are committed and embedded, and a program looks
them up by hashing the source again at runtime. A program that uses this
package never needs node, a browser or network access to show a diagram.

Keying by content hash means an edited diagram produces a different file, a
stale image is detected rather than shown, and nobody maintains a mapping by
hand. Check reports the sources without images and the images without
sources, and the fynedesygn-mermaid command runs it in CI.

The rules for diagrams in a window are in docs/mermaid.md.
*/
package mermaid
