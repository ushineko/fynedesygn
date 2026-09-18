package markdown

import "strings"

/*
Blocks splits a Markdown document into the pieces that are rendered
independently.

A blank line is the split, which is Markdown's own paragraph rule: headings,
paragraphs, lists and indented code blocks all end at one. Fenced code is the
exception, a blank line inside a fence is part of the code, so fences are
tracked and not split on.
*/
func Blocks(src string) []string {
	var (
		blocks []string
		cur    []string
		fenced bool
	)
	flush := func() {
		if len(cur) > 0 {
			blocks = append(blocks, strings.Join(cur, "\n"))
			cur = nil
		}
	}
	for _, line := range strings.Split(strings.ReplaceAll(src, "\r\n", "\n"), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			fenced = !fenced
			cur = append(cur, line)
			if !fenced {
				flush()
			}
			continue
		}
		if !fenced && strings.TrimSpace(line) == "" {
			flush()
			continue
		}
		cur = append(cur, line)
	}
	flush()
	return blocks
}

/*
CodeBlock reports whether a block is code and gives back the code with its
fence or its indent taken off, and the fence's info string (its language)
when it had one.

Both of Markdown's spellings count: a fenced block, and a run of lines each
indented by four spaces or a tab, which is how many READMEs show commands.
*/
func CodeBlock(src string) (lang, code string, ok bool) {
	lines := strings.Split(src, "\n")
	first := strings.TrimSpace(lines[0])
	if strings.HasPrefix(first, "```") {
		lang = strings.TrimSpace(strings.TrimPrefix(first, "```"))
		body := lines[1:]
		if n := len(body); n > 0 && strings.HasPrefix(strings.TrimSpace(body[n-1]), "```") {
			body = body[:n-1]
		}
		return lang, strings.Join(body, "\n"), true
	}
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		switch {
		case strings.TrimSpace(line) == "":
			out = append(out, "")
		case strings.HasPrefix(line, "    "):
			out = append(out, line[4:])
		case strings.HasPrefix(line, "\t"):
			out = append(out, line[1:])
		default:
			return "", "", false
		}
	}
	return "", strings.Join(out, "\n"), true
}

// ImageBlock reports whether a block is a single Markdown image and returns
// its alt text and path. Only whole-block images are handled by the pane;
// an image inside a paragraph stays with Fyne's own rendering.
func ImageBlock(src string) (alt, path string, ok bool) {
	s := strings.TrimSpace(src)
	if !strings.HasPrefix(s, "![") {
		return "", "", false
	}
	end := strings.Index(s, "](")
	if end < 0 || !strings.HasSuffix(s, ")") {
		return "", "", false
	}
	alt = s[2:end]
	path = strings.TrimSpace(s[end+2 : len(s)-1])
	// A title after the path ("path \"title\"") is dropped.
	if i := strings.IndexAny(path, " \t"); i > 0 {
		path = path[:i]
	}
	if path == "" || strings.Contains(alt, "\n") {
		return "", "", false
	}
	return alt, path, true
}

// isRelative reports whether an image path is relative to the document rather
// than a URL or an absolute path.
func isRelative(p string) bool {
	return !strings.Contains(p, "://") && !strings.HasPrefix(p, "/") && !strings.HasPrefix(p, "\\")
}

/*
TableBlock reports whether a block is a pipe table (a header row, a separator
row of dashes, and body rows) and returns its cells. Fyne renders a Markdown
table inside a scroller, which takes the wheel from the page, so the pane
draws tables itself as a grid.
*/
func TableBlock(src string) (rows [][]string, ok bool) {
	lines := strings.Split(strings.TrimSpace(src), "\n")
	if len(lines) < 2 || !strings.Contains(lines[0], "|") || !isTableSeparator(lines[1]) {
		return nil, false
	}
	head := splitRow(lines[0])
	rows = append(rows, head)
	for _, line := range lines[2:] {
		if !strings.Contains(line, "|") {
			return nil, false
		}
		cells := splitRow(line)
		for len(cells) < len(head) {
			cells = append(cells, "")
		}
		rows = append(rows, cells[:len(head)])
	}
	return rows, true
}

// isTableSeparator recognises "|---|:--:|" and friends.
func isTableSeparator(line string) bool {
	s := strings.TrimSpace(line)
	if !strings.Contains(s, "-") {
		return false
	}
	for _, r := range s {
		if r != '|' && r != '-' && r != ':' && r != ' ' {
			return false
		}
	}
	return true
}

// splitRow splits a table row on unescaped pipes, dropping the outer ones.
func splitRow(line string) []string {
	s := strings.TrimSpace(line)
	s = strings.TrimPrefix(s, "|")
	s = strings.TrimSuffix(s, "|")
	var cells []string
	var cur strings.Builder
	escaped := false
	for _, r := range s {
		switch {
		case escaped:
			cur.WriteRune(r)
			escaped = false
		case r == '\\':
			escaped = true
		case r == '|':
			cells = append(cells, strings.TrimSpace(cur.String()))
			cur.Reset()
		default:
			cur.WriteRune(r)
		}
	}
	cells = append(cells, strings.TrimSpace(cur.String()))
	return cells
}
