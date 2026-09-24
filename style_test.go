package fynedesygn_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

/*
The mechanical half of docs/style.md.

Only the rules a test can judge without being wrong more often than it is
right. Plain verbs, consistent naming and "no metaphor" are judgement, and a
test that guessed at them would be turned off within a month.

converted lists the documents the style applies to. The specs and the
changelog are a record of what was decided and when, and are left alone; this
list is how a document opts in, so adding one is a decision somebody makes.
*/
var converted = []string{
	filepath.Join("docs", "fyne-quirks.md"),
	filepath.Join("docs", "style.md"),
	filepath.Join("docs", "mermaid.md"),
	filepath.Join("docs", "markdown.md"),
	filepath.Join("docs", "performance.md"),
	filepath.Join("docs", "glance.md"),
	filepath.Join("docs", "design-system.md"),
}

/*
sentences splits prose into sentences.

Three things have to be taken out of the way first, or the count is of the
wrong thing. A full stop inside a code span is part of an identifier, not
punctuation. A link is one idea however long its URL. And emphasis marks sit
between a full stop and the space after it, so a bold lead sentence would
otherwise be joined to the sentence that follows it.
*/
func sentences(text string) []string {
	text = regexp.MustCompile("`[^`]*`").ReplaceAllString(text, "CODE")
	text = regexp.MustCompile(`\[[^\]]*\]\([^)]*\)`).ReplaceAllString(text, "LINK")
	text = strings.NewReplacer("**", "", "*", "", "_", "").Replace(text)

	var out []string
	for _, s := range regexp.MustCompile(`(?m)[.!?]\s+|[.!?]$`).Split(text, -1) {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	return out
}

// cells is a table row split into its cells, so a row is measured as the
// prose it holds rather than as one long line with pipes in it.
func cells(line string) []string {
	if !strings.HasPrefix(line, "| ") {
		return []string{line}
	}
	return strings.Split(strings.Trim(line, "| "), " | ")
}

func TestConvertedDocumentsKeepTheirSentencesShort(t *testing.T) {
	/*
		Thirty words is the ceiling in docs/style.md, not the target. A
		sentence past it is usually two sentences that have been joined by a
		comma, which is the habit these documents were converted away from.
	*/
	const most = 30

	for _, path := range converted {
		body, err := os.ReadFile(path)
		require.NoError(t, err)

		for _, line := range strings.Split(string(body), "\n") {
			if strings.HasPrefix(line, "|---") || strings.HasPrefix(line, "| #") {
				continue
			}
			for _, cell := range cells(line) {
				for _, sentence := range sentences(cell) {
					words := len(strings.Fields(sentence))
					require.LessOrEqual(t, words, most,
						"%s: a sentence of %d words, which is past the ceiling in docs/style.md:\n%s",
						path, words, sentence)
				}
			}
		}
	}
}

/*
empty are the words that take a line's length and give nothing back.

Short, and every entry earns its place: each is a word that tells the reader
how to feel about a fact instead of telling them the fact. "Simply" and
"just" say the thing is easy, which it was not for whoever hit it; "of
course" and "obviously" say they should have known.
*/
var empty = []string{"simply", "just ", "obviously", "of course", "needless to say", "easily"}

func TestConvertedDocumentsSayNothingEmpty(t *testing.T) {
	for _, path := range converted {
		body, err := os.ReadFile(path)
		require.NoError(t, err)

		lower := strings.ToLower(string(body))
		for _, word := range empty {
			// docs/style.md names them to ban them.
			if path == filepath.Join("docs", "style.md") {
				continue
			}
			require.NotContains(t, lower, word,
				"%s contains %q, which docs/style.md asks you to delete", path, word)
		}
	}
}

func TestTheStyleIsWrittenDown(t *testing.T) {
	// A convention nobody can read is a convention that lasts one author.
	body, err := os.ReadFile(filepath.Join("docs", "style.md"))
	require.NoError(t, err)

	text := string(body)
	require.Contains(t, text, "plain technical English")
	require.Contains(t, text, "ASD-STE100")
	require.Contains(t, text, "Nothing here should claim to be STE",
		"the style document must say it does not comply, so nobody checks it against the standard")
}
