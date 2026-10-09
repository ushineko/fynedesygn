# Spec 064: links that do what they say

**Issue**: [#187](https://github.com/ushineko/fynedesygn/issues/187)

## Status: COMPLETE

## Executive Summary

A link in a `markdown.Pane` either works or is not a link. A `#fragment`
scrolls the pane to the heading with that GitHub slug, an absolute http(s)
address opens in the browser, and anything else is drawn as plain text.
A link also takes a click anywhere over its text: Fyne's link boxes overlapped
the line below, and in a list of links a click on the lower half of an entry
did nothing. Reviewers should look first at `markdown/links.go`
(`linkFixer.link`, `linkRenderer.Refresh`, `Slug`, `Anchors`) and
`Pane.AnchorY` and `Pane.top` in `markdown/pane.go`.

## Context

What happened before this spec, read from Fyne 2.8.1's source: the Markdown
widget makes every link a `HyperlinkSegment` holding `url.Parse` of the
destination. A tap with no `OnTapped` calls `fyne.CurrentApp().OpenURL`, which
runs `rundll32 url.dll,FileProtocolHandler <url>` on Windows and `xdg-open
<url>` on Linux. For `#installing-it` or `docs/architecture.md` that is a
relative string with nothing to resolve against: nothing useful opens, and at
most an error is logged. hayami's About page is an embedded README whose
contents list is twelve such anchors, and it links two repository documents by
relative path. Its screenshots showed alt text because it passed no FS.

The issue asked for relative links resolved against a base URL. The owner
decided against it: links to other documents must be absolute URLs, so the
README reads the same on GitHub and in the window and there is no base to
keep in step with a branch.

## Requirements

- R1. A link whose destination is only a fragment scrolls the pane's followed
  scroller so the heading with that slug is at the top, as far as the
  document's length allows. An unknown fragment, or a pane with no scroller,
  does nothing.
- R2. Slugs are GitHub's: lower case; letters, digits, spaces, hyphens and
  underscores kept; spaces made hyphens; a repeated slug numbered -1, -2.
  Headings in code fences are not headings.
- R3. An absolute http or https link calls `Options.OpenURL`, or the app's
  `OpenURL` when that is nil.
- R4. Every other link (relative path, other scheme, and an anchor rendered by
  `RenderBlock` with no pane) is drawn as plain text, in prose, lists and
  table cells.
- R5. Relative images keep resolving through `Options.FS` and `Dir`;
  docs/markdown.md says how to embed a README's images.
- R6. The pane's position in the scroller's content is measured from the
  canvas, so a pane nested under a header scrolls to, and renders, the right
  blocks. A heading's offset is its cell's laid-out position when the column
  has been laid out, since a block drawn while a re-measure waits out
  SettleResize can be taller than its spacer.
- R7. A click anywhere over a link's text reaches that link, including in a
  list of links one under another.

## Acceptance Criteria

- [x] Tapping a contents-list link over its text puts the heading's top at the
  viewport's top (within half a pixel), for a near and a far heading, with the
  pane nested two containers deep under a header.
- [x] An unknown fragment leaves the offset alone.
- [x] An absolute link's tap reaches `Options.OpenURL` with the URL unchanged.
- [x] No hyperlink is drawn for a relative link in prose, a list or a table,
  and its text is still drawn.
- [x] `RenderBlock` draws an anchor as text and an absolute link as a link.
- [x] Slug and anchor tables pass, including duplicates and fences.
- [x] A click through the window's canvas at 20%, 50% and 80% of the height of
  each entry in a five-link list reaches that entry.
- [x] `go test -race ./...`, golangci-lint (windows and linux) and gofmt are
  clean.

## The overlapping link boxes

Found by hayami's real-window test, not by any headless one. Fyne draws a
link in a RichText as a Hyperlink widget the theme's inner padding (8) larger
than its text on every side, pulled back over the text by
`unpadTextWidgetLayout`. A Contents list's lines are about 16 points apart,
so each link's box covered the lower half of the line above, and Fyne gives a
click to the topmost box: the next entry's. That link either took the click
(headless, the test driver) or, its own check finding the point above its
text, dropped it (on the window). Only the top of an entry worked, and the
last entry of a list worked everywhere.

`linkText` draws a block that holds a live link. Its renderer applies a theme
override with an inner padding of zero to each link widget the RichText made,
then lays the widget out again: the box becomes the text's box and the text
does not move. It has to happen in the renderer's Refresh because a RichText
recreates the link widgets inside a list or paragraph on every refresh (its
visual cache keeps only top-level segments), and an override applies only to
objects that exist when it is applied.

## Risks & Assumptions

- `linkText` depends on how Fyne 2.8 draws a hyperlink segment (a container
  holding one Hyperlink). If that changes, the loop finds no links and the
  overlap returns; `TestEveryPartOfAListedLinkTakesTheClick` says so.

- GitHub's slug drops emoji and most symbols; this one drops anything that is
  not a Unicode letter or number. A heading made only of symbols gets an empty
  slug on both, which is not a useful anchor on either.
- `Pane.top` asks the driver for absolute positions. Outside a canvas it falls
  back to `Position`, which is right when the pane is the scroller's content
  or its column's first item.

## E2E Test Plan

hayami's About page is the consumer and carries the real-window check:
clicking contents links in the running window and reading the scroll from
screenshots (hayami spec for its About links).

### Results (2026-10-08)

Headless suite as in the acceptance criteria. Falsified: with the link fixer
switched off, the anchor, opener, relative-link and RenderBlock tests fail;
with `top` reduced to `Position`, the tap test fails by the 4-pixel padding
the nesting adds; with `linkRenderer` not unpadding, the list-click test fails
at 80% of every entry but the last.

hayami's real-window test (its spec 055) clicks three Contents entries by
posted mouse messages and reads the page from screenshots. Passing with this
change; with the unpadding off, the clicks on "Platform notes" and
"Installing" did nothing and the one on "Changelog", the last entry, worked.
