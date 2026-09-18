/*
Package markdown renders a Markdown document in a Fyne window a screenful at
a time, with code blocks and diagrams drawn by this module.

Ported from clockwork-orange's markdownPane (same author, MIT), written after
its About section scrolled in fits and bursts: Fyne's RichText lays out and
repaints every segment it holds on each refresh, a scroll refreshes its
content as it moves, and a README of 250 segments cost the whole document on
every wheel notch. Fyne 2.8 also draws each code block inside a horizontal
scroller that takes the wheel from the page.

Pane keeps only the blocks near the viewport in the widget tree. Each block
has a transparent spacer sized to the height it needs at the current width;
the rendered block is laid on the spacer when it comes near the viewport and
taken away when it goes far. The spacer stays either way, so the document's
height and every block's position are the same whether or not a block is
rendered: scrolling never reflows the interface and the scrollbar never
jumps. Heights are measured, never estimated.

The standard guidance for authors is in docs/markdown.md.
*/
package markdown
