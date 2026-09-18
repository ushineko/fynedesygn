/*
Package docs embeds this module's documentation and its rendered diagrams so
a program can show them in a window: the gallery's Documents section reads
docs/markdown.md from here, diagram images included.

The diagrams are rendered by go generate; CI checks they are current.
*/
package docs

import "embed"

//go:generate go run ../cmd/fynedesygn-mermaid -root .

// FS holds every Markdown document in this directory and the PNGs under
// diagrams/.
//
//go:embed *.md diagrams/*.png
var FS embed.FS
