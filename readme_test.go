package fynedesygn_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

/*
The README goes stale silently: nothing builds it and, until these tests, no
test read it. It drifted twenty-four releases once — the Version line said
0.1.3 while the latest tag was v0.1.27, the settings packages had never reached
the package table, four gallery screenshots existed that nothing displayed, and
two specs' worth of work had no changelog entry.

These are canaries for that, in the same spirit as the Fyne quirk canaries:
they fail when the repository and its front page have drifted apart, and the
fix is always to write the missing line rather than to relax the test.
*/

// readme is the front page, read once per test.
func readme(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile("README.md")
	require.NoError(t, err)
	return string(b)
}

// plumbingTargets are the make targets the README deliberately does not list:
// they are either self-describing or only run by another target.
var plumbingTargets = map[string]bool{
	"help":         true, // prints the list this test reads
	"install-lint": true, // a dependency of lint and setup
	"clean":        true, // does what it says
}

// TestEveryPackageIsInTheReadme fails when a package has been added and the
// "What is in it" table has not. A reader's only index of this module is that
// table.
//
// Each name is looked for in its own section rather than anywhere in the file.
// Searching the whole README passes on a package the table never had, because
// the name turns up in the Examples table or in a changelog entry: `settings`
// is both a package and an example, and the first version of this test was
// green with the package row deleted.
func TestEveryPackageIsInTheReadme(t *testing.T) {
	doc := readme(t)
	packages := section(t, doc, "## What is in it", "## Screenshots")
	examples := section(t, doc, "## Examples", "## Status")

	for _, dir := range goPackageDirs(t) {
		// Examples have their own table, keyed by their directory name.
		if strings.HasPrefix(dir, "examples/") {
			name := strings.TrimPrefix(dir, "examples/")
			require.Contains(t, examples, "`"+name+"`",
				"example %s is not in the Examples table", dir)
			continue
		}
		require.Contains(t, packages, "`"+dir+"`",
			"package %s is not in the What is in it table", dir)
	}
}

// section is the part of the README between two headings, so a name is looked
// for where it belongs rather than anywhere in the file.
func section(t *testing.T, doc, from, to string) string {
	t.Helper()

	i := strings.Index(doc, from)
	require.Positive(t, i, "the README has no %q heading", from)
	rest := doc[i+len(from):]

	j := strings.Index(rest, to)
	require.Positive(t, j, "the README has no %q heading after %q", to, from)
	return rest[:j]
}

// TestEveryDocumentIsLinkedFromTheReadme fails when a document has been added
// to docs/ and the Documentation list has not learned about it.
func TestEveryDocumentIsLinkedFromTheReadme(t *testing.T) {
	doc := readme(t)

	links := section(t, doc, "## Documentation", "## Development")

	files, err := filepath.Glob(filepath.Join("docs", "*.md"))
	require.NoError(t, err)
	require.NotEmpty(t, files)

	for _, f := range files {
		require.Contains(t, links, filepath.Base(f),
			"%s is not linked from the Documentation list", f)
	}
}

// TestEveryScreenshotIsShownInTheReadme fails when make screenshots produces
// an image the README does not display. An image nobody displays is an image
// nobody checks, and its alt text is the only description a screen-reader user
// gets.
//
// An image used by a document rather than the README is excused, because the
// document is where its alt text lives.
func TestEveryScreenshotIsShownInTheReadme(t *testing.T) {
	doc := readme(t)

	files, err := filepath.Glob(filepath.Join("docs", "img", "*.png"))
	require.NoError(t, err)
	require.NotEmpty(t, files)

	docs := allDocs(t)
	for _, f := range files {
		base := filepath.Base(f)
		if strings.Contains(doc, base) || strings.Contains(docs, base) {
			continue
		}
		t.Errorf("%s is displayed nowhere: add it to the README with alt text, "+
			"or delete it", f)
	}
}

// TestEveryMakeTargetIsInTheReadme fails when a documented make target is
// missing from the Development block, which is where anyone looks first.
func TestEveryMakeTargetIsInTheReadme(t *testing.T) {
	doc := readme(t)

	b, err := os.ReadFile("Makefile")
	require.NoError(t, err)

	// Targets that carry a `## ` description are the public ones.
	re := regexp.MustCompile(`(?m)^([a-z][a-z-]*):[^\n]*## `)
	matches := re.FindAllStringSubmatch(string(b), -1)
	require.NotEmpty(t, matches)

	for _, m := range matches {
		target := m[1]
		if plumbingTargets[target] {
			continue
		}
		require.Contains(t, section(t, doc, "## Development", "## Licence"), "make "+target,
			"make %s is documented in the Makefile but not in the Development block", target)
	}
}

// TestTheReadmeNamesItsVersion fails when the Version line is missing or is not
// a version. It cannot check the line against the latest tag, because a
// checkout has no tags in CI — the rule that they match is in .claude/CLAUDE.md
// and is applied when tagging.
func TestTheReadmeNamesItsVersion(t *testing.T) {
	doc := readme(t)

	m := regexp.MustCompile(`\*\*Version\*\*: (\d+\.\d+\.\d+)`).FindStringSubmatch(doc)
	require.Len(t, m, 2, "the README does not name a version")

	// The changelog's newest released heading should be that version: an
	// entry newer than the Version line means a release went out without the
	// line being moved, which is exactly how it reached 0.1.3 against v0.1.27.
	headings := regexp.MustCompile(`(?m)^### (\d+\.\d+\.\d+) `).FindAllStringSubmatch(doc, -1)
	require.NotEmpty(t, headings, "the changelog has no released entries")
	require.Equal(t, headings[0][1], m[1],
		"the Version line says %s but the newest changelog entry is %s", m[1], headings[0][1])
}

// TestUnreleasedWorkIsInTheChangelog fails when the changelog's first heading
// is neither Unreleased nor a version, which would mean the section has been
// reshaped and these canaries are reading the wrong thing.
func TestUnreleasedWorkIsInTheChangelog(t *testing.T) {
	doc := readme(t)

	i := strings.Index(doc, "## Changelog")
	require.Positive(t, i, "the README has no changelog")

	first := regexp.MustCompile(`(?m)^### (.+)$`).FindStringSubmatch(doc[i:])
	require.Len(t, first, 2)
	require.Regexp(t, `^(Unreleased|\d+\.\d+\.\d+ \(\d{4}-\d{2}-\d{2}\))$`, first[1],
		"the first changelog heading is %q, which is neither Unreleased nor a dated version", first[1])
}

// goPackageDirs lists every directory holding Go source, relative to the
// module root, skipping test fixtures and anything without .go files.
func goPackageDirs(t *testing.T) []string {
	t.Helper()

	var dirs []string
	err := filepath.WalkDir(".", func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		base := filepath.Base(path)
		if base == "testdata" || base == "bin" || strings.HasPrefix(base, ".") && path != "." {
			return filepath.SkipDir
		}
		if path == "." {
			return nil
		}
		entries, err := os.ReadDir(path)
		if err != nil {
			return err
		}
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".go") {
				dirs = append(dirs, filepath.ToSlash(path))
				break
			}
		}
		return nil
	})
	require.NoError(t, err)
	require.NotEmpty(t, dirs)
	return dirs
}

// allDocs is every document under docs/, concatenated, so a screenshot used by
// one of them counts as displayed.
func allDocs(t *testing.T) string {
	t.Helper()

	files, err := filepath.Glob(filepath.Join("docs", "*.md"))
	require.NoError(t, err)

	var b strings.Builder
	for _, f := range files {
		c, err := os.ReadFile(f) // nolint:gosec // a path from this repository
		require.NoError(t, err)
		b.Write(c)
	}
	return b.String()
}

// TestTheContentsListMatchesTheHeadings fails when a section has been added
// and the Contents list has not, or the other way round. Adding a section and
// forgetting its entry is the easiest of these to do — it happened while
// writing the "Used by" section that this test now guards.
func TestTheContentsListMatchesTheHeadings(t *testing.T) {
	doc := readme(t)

	contents := section(t, doc, "## Contents", "## ")
	listed := map[string]bool{}
	for _, m := range regexp.MustCompile(`\[[^\]]+\]\(#([a-z0-9-]+)\)`).FindAllStringSubmatch(contents, -1) {
		listed[m[1]] = true
	}
	require.NotEmpty(t, listed, "the Contents list has no anchors")

	for _, m := range regexp.MustCompile(`(?m)^## (.+)$`).FindAllStringSubmatch(doc, -1) {
		title := m[1]
		if title == "Contents" {
			continue
		}
		require.True(t, listed[anchorFor(title)],
			"the section %q is not in the Contents list", title)
	}
}

// anchorFor is GitHub's heading anchor: lower case, spaces to hyphens, and
// anything else dropped.
func anchorFor(title string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(title) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == ' ', r == '-':
			b.WriteRune('-')
		}
	}
	return b.String()
}

/*
TestEverySpecIsInTheChangelog fails when work has a spec and no entry.

The README rule says every change worth a spec gets one, and the existing
canary only checks that the first changelog heading has the right shape. Three
entries were lost in one afternoon because the edit that added them looked for
`### Unreleased`, which stopped existing the moment a release was tagged, and
a string replace that matches nothing says nothing.

A spec number is the thing both halves have: the file is `specs/NNN-...` and
the entry ends `(spec NNN, #issue)`.
*/
func TestEverySpecIsInTheChangelog(t *testing.T) {
	doc := readme(t)
	at := strings.Index(doc, "## Changelog")
	require.Positive(t, at, "the README has no changelog")
	changelog := doc[at:]

	specs, err := filepath.Glob(filepath.Join("specs", "*.md"))
	require.NoError(t, err)
	require.NotEmpty(t, specs)

	for _, path := range specs {
		found := regexp.MustCompile(`^(\d+)-`).FindStringSubmatch(filepath.Base(path))
		if found == nil {
			continue
		}
		number, err := strconv.Atoi(found[1])
		require.NoError(t, err)
		if number < firstCitedSpec {
			continue
		}
		body, err := os.ReadFile(path)
		require.NoError(t, err)
		if !strings.Contains(string(body), "## Status: COMPLETE") {
			continue // not finished, so not in a changelog yet
		}

		require.Regexp(t, regexp.MustCompile(`(?i)specs? 0*`+found[1]), changelog,
			"%s is complete and the changelog does not mention it", path)
	}
}

/*
firstCitedSpec is where the convention starts.

Spec 018 is the one that wrote the README rule this canary enforces, so 019 is
the first entry that could have been written knowing it. The fourteen before
it are described in the changelog in prose -- 001 to 005 as "First tagged
release", 014 inside the entry for 0.1.26 -- without the word "spec", and
rewriting released entries to satisfy a test written afterwards would be
changing the record to fit the guard.
*/
const firstCitedSpec = 19
