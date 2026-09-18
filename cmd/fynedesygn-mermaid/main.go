/*
Command fynedesygn-mermaid renders the Mermaid diagrams under a directory to
the light and dark PNG pairs the mermaid package looks up at runtime.

It is a go generate step, run on a developer's machine where mmdc is
installed. In CI it runs with -check, which needs no mmdc and fails when a
diagram's source has changed since its image was rendered or an image no
longer has a source.

	fynedesygn-mermaid -root docs            render what is missing
	fynedesygn-mermaid -root docs -check     exit 1 on missing or stale images
	fynedesygn-mermaid -root docs -prune     also delete stale images
*/
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/ushineko/fynedesygn/mermaid"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// say prints to a stream; a failed write to stdout or stderr is nothing the
// command can act on.
func say(w io.Writer, format string, args ...any) {
	_, _ = fmt.Fprintf(w, format, args...)
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("fynedesygn-mermaid", flag.ContinueOnError)
	fs.SetOutput(stderr)
	root := fs.String("root", ".", "directory to scan for *.md fences and *.mmd files")
	out := fs.String("out", "", "directory for the PNGs; default <root>/"+mermaid.DefaultDir)
	mmdc := fs.String("mmdc", "mmdc", "mermaid-cli executable")
	check := fs.Bool("check", false, "report missing and stale images and exit 1 on any; render nothing")
	prune := fs.Bool("prune", false, "delete images no source refers to")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *out == "" {
		*out = filepath.Join(*root, mermaid.DefaultDir)
	}

	missing, stale, err := mermaid.Check(*root, *out)
	if err != nil {
		say(stderr, "%v\n", err)
		return 1
	}
	if *check {
		for _, m := range missing {
			say(stdout, "missing: %s:%d (%s)\n", m.File, m.Line, mermaid.Hash(m.Text))
		}
		for _, s := range stale {
			say(stdout, "stale: %s\n", s)
		}
		if len(missing) > 0 || len(stale) > 0 {
			say(stderr, "%d missing, %d stale; run make generate\n", len(missing), len(stale))
			return 1
		}
		say(stdout, "%s\n", "diagrams up to date")
		return 0
	}

	ctx := context.Background()
	for _, m := range missing {
		if err := mermaid.Render(ctx, *mmdc, m.Text, *out); err != nil {
			say(stderr, "%s:%d: %v\n", m.File, m.Line, err)
			return 1
		}
		say(stdout, "rendered %s:%d -> %s\n", m.File, m.Line, mermaid.Hash(m.Text))
	}
	if *prune {
		for _, s := range stale {
			if err := os.Remove(s); err != nil {
				say(stderr, "remove %s: %v\n", s, err)
				return 1
			}
			say(stdout, "removed %s\n", s)
		}
	} else if len(stale) > 0 {
		say(stdout, "%d stale image(s); run with -prune to delete\n", len(stale))
	}
	if len(missing) == 0 {
		say(stdout, "%s\n", "nothing to render")
	}
	return 0
}
