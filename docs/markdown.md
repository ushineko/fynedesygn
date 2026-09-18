# Rendering Markdown

This is the standard guidance for showing Markdown in a program built with this
module. The component is `markdown.Pane`; it descends from clockwork-orange's
`markdownPane` (spec 011 there), which was written after its About section
scrolled "in fits and bursts" on KDE Plasma 6.

## What to use it for

- An embedded README or user guide shown in an About or Help section.
- A generated report that is already Markdown.
- Any prose longer than a few paragraphs.

For a sentence or two, use a wrapped label. For structured data, use the
detail table, not a Markdown table.

## How it renders

1. The source is split into blocks on blank lines, tracking code fences so a
   blank line inside a fence does not split it.
2. Each block is classified: fenced or indented code, a `mermaid` fence, or
   prose.
3. Prose renders with `widget.NewRichTextFromMarkdown` with word wrapping.
   Code renders in `markdown.CodePanel`, a monospace label on an
   input-coloured rectangle that wraps long lines. Mermaid renders as an
   embedded PNG looked up by content hash (see [mermaid.md](mermaid.md)); a
   diagram with no pre-rendered image falls back to a code panel showing the
   source, with a caption saying so.
4. The pane measures every block once at the current width and reserves that
   height with a spacer. Only blocks within half a viewport of the visible
   area are rendered; the rest are spacers. Heights are measured, never
   estimated, and re-measured when the width changes, so the document's total
   height and every block's position are identical whether or not a block is
   currently rendered. Nothing reflows as the reader scrolls.
5. The pane is not scrollable itself. The section puts it inside its own
   scroller and hands that scroller to `Follow`; `Detach` gives the scroll
   callback back when the section is replaced.

## Why not one RichText

Fyne's `RichText` lays out and repaints every segment it holds on each
refresh, and a scroller refreshes its content as it moves. clockwork-orange's
README is 221 lines and became about 250 segments; a frame during a scroll cost
about 51 ms in the software painter with one `RichText` against about 33 ms
with the pane rendering 21 of 44 blocks.

## Why code blocks are the module's own

Fyne 2.8 wraps each Markdown code block in a horizontal scroller. Fyne gives a
wheel event to the innermost scrollable under the pointer and does not pass it
on, and the scroller swaps the axes when the code is wider than the pane but
not taller, so a vertical notch over a code block became a horizontal one and
the page stopped. The code panel wraps long lines instead. Sideways scrolling
would read better, but it hides the rest of the line behind the gesture the
page itself needs. A canary test asserts Fyne still does this; when it stops,
the panel can go.

## Rules for authors

- Keep the documentation in one place. If the program has a README, embed it
  and show it; do not restate it in the window.
- Relative image paths are resolved through the `fs.FS` the pane is given.
  Use paths relative to the document.
- Prefer fenced code with a language to indented code; both render the same,
  but the fence names the language for readers of the source.
- Mermaid fences must have a pre-rendered image in the program's diagram set
  or they show as source. `make generate` renders them.
- Headings, lists, emphasis, links and inline code are supported by Fyne's
  RichText. Tables render poorly in RichText; use the detail table component
  for tabular data in the window and keep Markdown tables for the file.

## Testing

`fynetest.ScrollableIn(obj)` walks a rendered object and reports whether any
scrollable lives inside it. A document's test asserts `!ScrollableIn` for every
block. `fynetest.TextIn(obj)` extracts the text so a test can assert content
without knowing the layout.
