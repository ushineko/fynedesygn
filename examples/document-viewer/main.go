/*
Command document-viewer is a fynedesygn example: a section that is one
Markdown document, embedded with its pre-rendered diagram, following the
shell's scroller so one scrollbar governs the page.
*/
package main

import (
	"embed"

	"fyne.io/fyne/v2"
	fynetheme "fyne.io/fyne/v2/theme"

	"github.com/ushineko/fynedesygn/markdown"
	"github.com/ushineko/fynedesygn/mermaid"
	"github.com/ushineko/fynedesygn/shell"
	"github.com/ushineko/fynedesygn/widgets"
)

//go:generate go run ../../cmd/fynedesygn-mermaid -root .

//go:embed guide.md diagrams/*.png
var content embed.FS

func main() { shell.Run(options()) }

func options() shell.Options {
	guide, _ := content.ReadFile("guide.md")
	opts := markdown.Options{FS: content, Diagrams: mermaid.NewSet(content, "diagrams")}
	return shell.Options{
		AppID: "io.ushineko.fynedesygn.example.docviewer",
		Name:  "document-viewer",
		Sections: []shell.Section{
			markdown.Section("Guide", fynetheme.DocumentIcon, string(guide), opts, func(*shell.Shell) fyne.CanvasObject {
				return widgets.Heading("Guide", "An embedded document with a diagram rendered at development time.")
			}),
			shell.AboutSection(shell.About{
				Name: "document-viewer", Version: "example",
				Blurb: "One of the fynedesygn examples.",
				Facts: []shell.Fact{{Label: "Document", Value: "guide.md, embedded"}},
			}),
		},
	}
}
