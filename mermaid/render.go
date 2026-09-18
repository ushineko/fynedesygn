package mermaid

import (
	"bufio"
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// Source is one diagram found in a tree.
type Source struct {
	// File is the path it came from; Line is the 1-based line of the fence
	// or 1 for a .mmd file.
	File string
	Line int
	// Text is the diagram source without its fence.
	Text string
}

// Sources finds every ```mermaid fence in *.md files and every *.mmd file
// under root, skipping the diagrams directory and hidden directories. Files
// are visited in lexical order so the result is stable.
func Sources(root string) ([]Source, error) {
	// Collect first, read after: a read inside the walk callback is the
	// pattern gosec flags for symlink races, and the order is stable either way.
	var mdFiles, mmdFiles []string
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			base := d.Name()
			if p != root && (base == DefaultDir || strings.HasPrefix(base, ".")) {
				return filepath.SkipDir
			}
			return nil
		}
		switch strings.ToLower(filepath.Ext(p)) {
		case ".md":
			mdFiles = append(mdFiles, p)
		case ".mmd":
			mmdFiles = append(mmdFiles, p)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan %s: %w", root, err)
	}

	var out []Source
	for _, p := range mdFiles {
		found, err := fencesIn(p)
		if err != nil {
			return nil, err
		}
		out = append(out, found...)
	}
	for _, p := range mmdFiles {
		b, err := os.ReadFile(p) //nolint:gosec // a path found under the root the caller named
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", p, err)
		}
		out = append(out, Source{File: p, Line: 1, Text: string(b)})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].File != out[j].File {
			return out[i].File < out[j].File
		}
		return out[i].Line < out[j].Line
	})
	return out, nil
}

// fencesIn extracts the mermaid fences of one Markdown file.
func fencesIn(p string) ([]Source, error) {
	f, err := os.Open(p) //nolint:gosec // a path found under the root the caller named
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", p, err)
	}
	defer func() { _ = f.Close() }()

	var (
		out   []Source
		body  []string
		start int
		in    bool
		line  int
	)
	scan := bufio.NewScanner(f)
	scan.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scan.Scan() {
		line++
		text := scan.Text()
		trimmed := strings.TrimSpace(text)
		switch {
		case !in && strings.HasPrefix(trimmed, "```") && strings.TrimSpace(strings.TrimPrefix(trimmed, "```")) == "mermaid":
			in, start, body = true, line, nil
		case in && strings.HasPrefix(trimmed, "```"):
			out = append(out, Source{File: p, Line: start, Text: strings.Join(body, "\n")})
			in = false
		case in:
			body = append(body, text)
		}
	}
	if err := scan.Err(); err != nil {
		return nil, fmt.Errorf("read %s: %w", p, err)
	}
	return out, nil
}

// Args builds the mmdc argument slice for one variant: input file, output
// file, the mermaid theme matching the variant, a transparent background so
// the PNG sits on any scheme, the render scale, and quiet output.
func Args(in, out string, dark bool) []string {
	theme := "default"
	if dark {
		theme = "dark"
	}
	return []string{"-i", in, "-o", out, "-t", theme, "-b", "transparent", "-s", fmt.Sprintf("%g", Scale), "-q"}
}

// Render writes the light and dark PNGs for src into outDir using the mmdc
// executable named (a bare name is looked up on PATH). The source goes
// through a temporary file because mmdc reads a file, not an argument.
func Render(ctx context.Context, mmdc, src, outDir string) error {
	if err := os.MkdirAll(outDir, 0o750); err != nil {
		return fmt.Errorf("create %s: %w", outDir, err)
	}
	tmp, err := os.CreateTemp("", "fynedesygn-*.mmd")
	if err != nil {
		return fmt.Errorf("temp file: %w", err)
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	if _, err := tmp.WriteString(strings.TrimSpace(strings.ReplaceAll(src, "\r\n", "\n")) + "\n"); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write %s: %w", tmp.Name(), err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close %s: %w", tmp.Name(), err)
	}

	for _, dark := range []bool{false, true} {
		out := filepath.Join(outDir, FileName(src, dark))
		cmd := exec.CommandContext(ctx, mmdc, Args(tmp.Name(), out, dark)...) //nolint:gosec // an argument slice, no shell; mmdc is the developer's tool
		if b, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("mmdc %s: %w\n%s", out, err, strings.TrimSpace(string(b)))
		}
	}
	return nil
}

// Check compares the sources under root with the PNGs in outDir: sources
// with no image pair are missing, image files no source refers to are stale.
func Check(root, outDir string) (missing []Source, stale []string, err error) {
	srcs, err := Sources(root)
	if err != nil {
		return nil, nil, err
	}
	wanted := map[string]bool{}
	for _, s := range srcs {
		light, dark := FileName(s.Text, false), FileName(s.Text, true)
		wanted[light], wanted[dark] = true, true
		if !exists(filepath.Join(outDir, light)) || !exists(filepath.Join(outDir, dark)) {
			missing = append(missing, s)
		}
	}
	entries, err := os.ReadDir(outDir)
	if err != nil && !os.IsNotExist(err) {
		return nil, nil, fmt.Errorf("read %s: %w", outDir, err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".png") {
			continue
		}
		if !wanted[e.Name()] {
			stale = append(stale, filepath.Join(outDir, e.Name()))
		}
	}
	return missing, stale, nil
}

func exists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
