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
The quirks table is a table, and its rows are in order.

It stopped being one for eighteen releases. Two blank lines were left in the
middle of it, a blank line ends a Markdown table, and rows 29 to 35 rendered as
paragraphs with literal pipes in them -- on GitHub, and in the gallery, which
embeds docs/. Nobody noticed because nothing read the document.

The same reasoning as the README canaries next door: a document nothing builds
and no test reads is a document that drifts. This one has structure worth
checking, so it is checked.
*/
func TestTheQuirksTableIsOneTable(t *testing.T) {
	lines := strings.Split(quirks(t), "\n")

	first, last := -1, -1
	for i, line := range lines {
		if strings.HasPrefix(line, "| ") && !strings.HasPrefix(line, "| #") {
			if first < 0 {
				first = i
			}
			last = i
		}
	}
	require.Positive(t, first, "no table in docs/fyne-quirks.md")

	for i := first; i <= last; i++ {
		require.NotEmpty(t, strings.TrimSpace(lines[i]),
			"line %d is blank inside the table, which ends it: "+
				"every row after this one renders as a paragraph", i+1)
	}
}

func TestEveryQuirkIsNumberedInOrder(t *testing.T) {
	/*
		Rows are referred to by number -- "quirk 26" appears in quirk 27, and
		the canaries name them -- so a row inserted in the wrong place makes a
		reference point at the wrong thing. Row 28 sat between 14 and 15 for
		eighteen releases.
	*/
	numbered := regexp.MustCompile(`(?m)^\| (\d+) \|`)

	var got []int
	for _, m := range numbered.FindAllStringSubmatch(quirks(t), -1) {
		n, err := strconv.Atoi(m[1])
		require.NoError(t, err)
		got = append(got, n)
	}
	require.NotEmpty(t, got, "no numbered rows")

	for i, n := range got {
		require.Equal(t, i+1, n,
			"quirk %d is in position %d; the rows are numbered out of order", n, i+1)
	}
}

func TestEveryQuirkNamesItsCanary(t *testing.T) {
	/*
		"Add the row, name the canary, and write the canary in the same commit
		as the workaround" -- the document's own last paragraph. A row whose
		last cell is empty is a workaround nothing will tell us to delete when
		Fyne fixes the thing underneath it.
	*/
	for _, row := range strings.Split(quirks(t), "\n") {
		if !strings.HasPrefix(row, "| ") || strings.HasPrefix(row, "| #") {
			continue
		}
		cells := strings.Split(strings.Trim(row, "|"), " | ")
		require.Len(t, cells, 4, "row %.20s has %d cells, not four", row, len(cells))
		require.NotEmpty(t, strings.TrimSpace(cells[3]),
			"quirk %s names no canary", strings.TrimSpace(cells[0]))
	}
}

// quirks is the document, read once per test.
func quirks(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("docs", "fyne-quirks.md"))
	require.NoError(t, err)
	return string(b)
}
