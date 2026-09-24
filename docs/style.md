# How this is written

The documents in `docs/`, and the README, are written in what this repository
calls **plain technical English**.

The rules come from ASD-STE100, the Simplified Technical English specification
used for aerospace maintenance manuals. They are not the specification.
STE controls its vocabulary with a licensed dictionary, caps procedural
sentences at twenty words, and allows one idea per sentence; this repository
does none of those three. **Nothing here should claim to be STE**, and a
reader who checks it against the standard will find it does not comply.

What was worth taking is the discipline about words. What was worth leaving is
the sentence length: prose cut into eight-word pieces is harder to read, not
easier, because the reader has to reassemble the argument that the full stops
took apart.

## The rules

**Use plain verbs, and use one verb for one meaning.** `receives`, `examines`,
`returns`, `draws`. Not `takes`, `walks`, `consumes`, `grabs`. A verb that is
doing a metaphor's work is a verb the reader has to translate.

**Write in the active voice.** "An overlay receives every pointer event", not
"every pointer event is received by an overlay".

**Write in the simple present.** The behaviour is a fact about the code, not a
story about what happened when somebody found it.

**No idiom, no metaphor, no understatement.** "A table with a bend in it" and
"the pointer never leaves" read well and cost a non-native reader a stop. Say
what happens.

**No asides in em-dashes, and no stacked subordinate clauses.** One sentence
may join two clauses with `and`, `so`, `because` or a semicolon. It should not
join four.

**Aim for twenty-five words and stop at thirty.** A ceiling, not a target. A
compound sentence that keeps cause and effect together is better than two
sentences that separate them.

**Keep the bold lead sentence.** It is what makes a long table skimmable, and
plain language does not mean flat formatting.

**Name the thing the same way every time.** A `Section` is a section
everywhere; it is not a "page" in one paragraph and a "tab" in the next.

**Define the words this project invented.** `canary`, `quirk`, `segment`,
`glance` and `shell` are this repository's terms. Use them, and make sure each
is defined where a reader first meets it.

## What is not converted

The specifications in `specs/` and the changelog in the README are a record of
what was decided and when. They are left as they were written.

Doc comments in Go source are not converted either. They carry argument rather
than instruction, which is the part these rules serve least well.

## What is checked

`docs_test.go` holds the mechanical rules: sentence length, and a short list
of words that add nothing. Everything else on this page is judgement, and a
test that tried to enforce it would fail more often than it helped.
