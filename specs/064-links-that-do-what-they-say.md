# Spec 064: links that do what they say

**Issue**: [#187](https://github.com/ushineko/fynedesygn/issues/187)

## Status: COMPLETE

## Executive Summary

A link in a `markdown.Pane` either works or is not a link. A `#fragment`
scrolls the pane to the heading with that GitHub slug, an absolute http(s)
address opens in the browser, and anything else is drawn as plain text.
Reviewers should look first at `markdown/links.go` (`linkFixer.link`, `Slug`,
`Anchors`) and `Pane.AnchorY` and `Pane.top` in `markdown/pane.go`.

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
  blocks.

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
- [x] `go test -race ./...`, golangci-lint (windows and linux) and gofmt are
  clean.

## Risks & Assumptions

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
the nesting adds.
