package kwin

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// A minimal, order-preserving reader and writer for KDE's INI-shaped
// configuration files.
//
// It exists rather than a library because of what it must not do.
// kwinrulesrc holds every window rule the user has, written by System
// Settings; a parser that normalised the file — reordered sections, dropped
// comments, lower-cased keys, rewrote quoting — would hand the user back a
// file that was not theirs. This one keeps every line it did not come to
// change, in the order it found them.
//
// **Keys are case sensitive.** KWin reads `wmclass`, not `WMClass`, and a
// reader that folded case would produce a rule KWin silently ignores.

// iniLine is one line: a key and value, or a raw line (a comment or a blank)
// kept verbatim.
type iniLine struct {
	key, value string
	raw        string
	isRaw      bool
}

// iniSection is a named section and its lines in file order.
type iniSection struct {
	name  string
	lines []iniLine
}

// get returns the value of key, and whether it was present.
func (s *iniSection) get(key string) (string, bool) {
	for _, l := range s.lines {
		if !l.isRaw && l.key == key {
			return l.value, true
		}
	}
	return "", false
}

// set writes key, in place if it is already there and appended if not, so a
// rule the user edited keeps the shape they left it in.
func (s *iniSection) set(key, value string) {
	for i, l := range s.lines {
		if !l.isRaw && l.key == key {
			s.lines[i].value = value
			return
		}
	}
	s.lines = append(s.lines, iniLine{key: key, value: value})
}

// unset removes key if present. Used to clear stale keys an earlier version of
// a rule wrote.
func (s *iniSection) unset(key string) {
	for i, l := range s.lines {
		if !l.isRaw && l.key == key {
			s.lines = append(s.lines[:i], s.lines[i+1:]...)
			return
		}
	}
}

// iniFile is a whole file: its sections in order, plus anything above the
// first section header.
type iniFile struct {
	preamble []iniLine
	sections []*iniSection
}

// section returns the named section, or nil.
func (f *iniFile) section(name string) *iniSection {
	for _, s := range f.sections {
		if s.name == name {
			return s
		}
	}
	return nil
}

// ensure returns the named section, appending an empty one if it is absent.
func (f *iniFile) ensure(name string) *iniSection {
	if s := f.section(name); s != nil {
		return s
	}
	s := &iniSection{name: name}
	f.sections = append(f.sections, s)
	return s
}

// remove drops the named section if present.
func (f *iniFile) remove(name string) {
	for i, s := range f.sections {
		if s.name == name {
			f.sections = append(f.sections[:i], f.sections[i+1:]...)
			return
		}
	}
}

// parseINI reads a file. A malformed line is kept raw rather than rejected:
// this package is a guest in the user's configuration and refusing to read a
// file it did not write would be the wrong failure.
func parseINI(r io.Reader) (*iniFile, error) {
	f := &iniFile{}
	var current *iniSection

	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		line := sc.Text()
		trimmed := strings.TrimSpace(line)

		switch {
		case strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]"):
			current = &iniSection{name: trimmed[1 : len(trimmed)-1]}
			f.sections = append(f.sections, current)
			continue
		case trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, ";"):
			// keep verbatim
		default:
			if k, v, ok := strings.Cut(line, "="); ok {
				l := iniLine{key: strings.TrimSpace(k), value: v}
				if current == nil {
					f.preamble = append(f.preamble, l)
				} else {
					current.lines = append(current.lines, l)
				}
				continue
			}
		}

		l := iniLine{raw: line, isRaw: true}
		if current == nil {
			f.preamble = append(f.preamble, l)
		} else {
			current.lines = append(current.lines, l)
		}
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("reading the configuration: %w", err)
	}
	return f, nil
}

// render writes the file back. No space around the delimiter, which is how
// KDE writes it.
//
// A section is separated from the one above by exactly one blank line, and the
// blank line a file already had counts. KDE's own writer ends each section with
// one, and the parser keeps it as a raw line inside that section; emitting a
// separator regardless would add a second, and every install would add one
// more. Read against a real kwinrulesrc of thirteen rules, that was thirteen
// new blank lines per run.
func (f *iniFile) render() string {
	var b strings.Builder
	writeLines := func(lines []iniLine) {
		for _, l := range lines {
			if l.isRaw {
				b.WriteString(l.raw)
			} else {
				b.WriteString(l.key)
				b.WriteString("=")
				b.WriteString(l.value)
			}
			b.WriteString("\n")
		}
	}

	// endsBlank reports whether what has been written already finishes with an
	// empty line, so the separator is not doubled.
	endsBlank := func() bool { return strings.HasSuffix(b.String(), "\n\n") }

	writeLines(f.preamble)
	for i, s := range f.sections {
		if (i > 0 || len(f.preamble) > 0) && !endsBlank() {
			b.WriteString("\n")
		}
		b.WriteString("[")
		b.WriteString(s.name)
		b.WriteString("]\n")
		writeLines(s.lines)
	}

	// Exactly one newline at the end, which is what KDE's own writer leaves.
	// Without this, installing a rule and removing it again does not restore
	// the file: the separator written before the appended section is re-read
	// as a trailing blank line of the section above it, and survives the
	// removal.
	return strings.TrimRight(b.String(), "\n") + "\n"
}
