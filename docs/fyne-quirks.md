# Fyne quirks this module works around

Each entry names the behaviour, the Fyne version it was observed in, what the
module does about it, and the canary test that will fail when Fyne changes. A
failing canary means the workaround can be reconsidered, not that the module is
broken.

Observed against `fyne.io/fyne/v2 v2.8.1` unless noted. Canary test names are
the target names for the implementation specs; a row without a test yet says
"planned".

| # | Quirk | Workaround | Canary |
|---|---|---|---|
| 1 | File and folder dialogs crash if `Resize` is called before `Show`: the dialog has no window yet and `Resize` asks it for its minimum size through a nil pointer. | `dialogs` shows first, then resizes to 760 x 520. | `TestChooserResizeAfterShowDoesNotPanic` (planned) |
| 2 | `widget.Check.SetChecked` (and other programmatic setters) fire `OnChanged`. | Form models carry a `setting` re-entry guard; list rows clear `OnChanged` before `SetChecked`. | `TestSetCheckedFiresOnChanged` (planned) |
| 3 | On a recycled table cell, `SetText` refreshes the label and the refresh is where `Importance` becomes a colour. Setting importance after the text paints the previous row's colour. | `table` sets `Importance` before `SetText`. | `table.TestImportanceAfterSetTextKeepsOldColour` |
| 4 | `widget.Table.UpdateHeader` is called with `Col == -1` for the corner cell. | Guarded in `table`. | `table.TestUpdateHeaderVisitsCornerCell` |
| 5 | `container.AppTabs` sizes to its tallest item, not the selected one. | Only the selected tab holds content; the rest hold an empty container. | `TestAppTabsSizeToTallestItem` (planned) |
| 6 | `widget.NewRichTextFromMarkdown` wraps code blocks in a horizontal scroller (`richCodeBlock`). The innermost scrollable under the pointer takes the wheel, and the scroller swaps axes when the block is wider than the pane but not taller. | `markdown.CodePanel` renders code blocks itself and wraps instead of scrolling. | `fynetest.TestFyneStillDrawsMarkdownCodeInsideAScroll` |
| 7 | `widget.RichText` lays out and repaints every segment on each refresh, and a scroller refreshes its content as it moves. A 250-segment document costs ~51 ms per frame in the software painter. | `markdown.Pane` renders per block and only near the viewport, reserving measured heights. | `markdown.TestDocumentRendersOnlyNearTheViewport` |
| 8 | Constructing a theme icon (`theme.XxxIcon()`) before `app.New` logs "Attempt to access current Fyne app when none is started". | Section icons are `func() fyne.Resource`, resolved after the app exists. | `shell.TestNamesNeedNoApp` |
| 9 | Fyne animates properties, not opacity; a widget has no alpha. | Banner fade animates the background rectangle's colour with `canvas.NewColorRGBAAnimation`. | none (design, not a bug) |
| 10 | No scroll callback on `widget.List`; no way to know the user scrolled. | Follow-tail inference compares the current offset with the one the last auto-scroll left, tolerance 4. | `TestFollowTailToleratesFractionalOffsets` (planned) |
| 11 | `fyne.Do` runs inline under the test driver, so a worker touching widgets races the test. | Shell runs work inline when not on screen. | `shell.TestPerformRunsInlineWhenHeadless` |
| 12 | Fyne ignores fontconfig and has no font chooser. | `theme` scans font directories and loads faces itself. | none |
| 13 | No per-app interface scale API; only `FYNE_SCALE`. | Scale preference is written into the environment after `app.New` and before the window; a restart applies it. | `TestScalePreferenceYieldsToEnvironment` (planned) |
| 14 | No resize callback on `fyne.Window`. A `CanvasObject` is told its own new size, so a widget around the thing that cares is where to learn it. | Programs that persist geometry poll every 500 ms; `logpane` wraps a widget around its list and reflows in `Resize`. | `logpane.TestThePaneReflowsWhenTheWindowIsResized` |
| 28 | `widget.List` reuses the widget it made for a row index and does not re-run `UpdateItem` for it, so content that changes without the row count changing is not redrawn. `Refresh` brings in rows that are new and leaves the ones whose index already existed; `RefreshItem` only reaches a row the layout currently lists as visible, which during a resize it may not. | `logpane` builds a new list when the wrapping changes: a list with no rows yet has nothing to reuse, and it is virtualised, so it costs the visible rows. | `logpane.TestThePaneReflowsWhenTheWindowIsResized` |
| 15 | GLFW's Wayland backend lacks `cursor-shape-v1`; native Wayland windows show the default cursor theme. | `theme.ApplyCursorTheme` exports `XCURSOR_THEME`/`XCURSOR_SIZE` from desktop config before the toolkit starts (Linux only). | none (needs a compositor) |
| 16 | `Border` centre is sized from what is left over, so a long label in the centre overlaps its neighbours. | Status bar is a sequential `HBox` with a spacer. | none (layout rule) |
| 17 | An `HBox` hands a truncating label its minimum size, which for a truncating label is nothing. | Pin the label's width with `FixedWidth`. | none (layout rule) |
| 18 | No multi-select file dialog. | "Add..." reopens a single-file chooser. | none |
| 19 | Glyphs outside the bundled font (the right arrow `→` among them) draw from a fallback face and mark the run boundary as a missing glyph. | No such symbols in labels; use words or theme icons. The up and down arrows `↑` `↓` are covered and are what sort headers use. | none (copy rule) |
| 20 | `widget.Table` has no double-tap. | Row actions live in buttons enabled by selection. | none |
| 21 | Relative image paths in Markdown cannot be resolved by Fyne at runtime: the image segment takes a URI. | `markdown.Pane` reads whole-block images from `Options.FS`. | `markdown.TestRelativeImagesResolveThroughTheFS` |
| 22 | `widget.NewRichTextFromMarkdown` renders a pipe table as a `TableSegment` inside a scroller, which takes the wheel like a code block does. | `markdown.Pane` draws pipe tables as a grid of inline-rendered cells. | `markdown.TestFyneStillDrawsMarkdownTablesInsideAScroll` |
| 23 | No tooltip. `desktop.Hoverable` exists and nothing is built on it, so guidance a control cannot fit in its label has nowhere to go but under it, which turns a row into a paragraph. | `widgets.WithTip` stacks a transparent hover catcher over the control. | `widgets.TestATipWaitsAndThenAppears` |
| 24 | A pointer event goes to the **last** match in the visible-tree walk, not the first: `FindObjectAtPositionMatching` assigns on every match and does not stop. So an object stacked over another takes its hover, and a widget merely wrapping a control never sees one. | `widgets.WithTip` relies on it deliberately: the catcher is stacked over, so it wins the hover, and is not `Tappable`, so taps walk past it to the control. | `widgets.TestFyneStillGivesAPointerEventToTheLastMatch` |
| 25 | A `canvas.Rectangle` with a nil fill colour is invisible under the GL painter and a nil dereference under the software one, so a window built with one works and cannot be rendered to an image. | Spacers and hover catchers fill with `color.Transparent`. | `widgets.TestTheSpacersRenderToAnImage`, `widgets.TestATipRendersToAnImage` |
| 26 | **An overlay takes every pointer event in the window.** `FindObjectAtPositionMatching` walks the top overlay *instead of* the content when there is one, so nothing below an overlay can be clicked wherever it is on screen. A non-modal `widget.PopUp` wraps itself in an `OverlayContainer`, which is itself `Tappable` and dismisses on a tap, so the click is consumed rather than ignored: a control whose own tip was showing needed two clicks, and the first one only took the tip down. | Tips and result banners are drawn into layers in the window's content (`widgets.NewTipLayer` and the shell's `flashLayout`) rather than in overlays. The things they are made of are not tappable, so the same walk finds the control underneath; a banner's own dismiss is tappable and still takes the clicks aimed at it. Only the busy popup stays an overlay, because blocking input is what it is for. | `widgets.TestFyneStillSendsEveryClickToTheTopOverlay`, `widgets.TestAControlStillTakesItsClickWhileItsTipIsShowing`, `shell.TestABannerDoesNotTakeTheWindowsClicks` |
| 27 | Nothing tells a control that it was clicked rather than hovered, so a hover tip stays up when the click opens a menu or a dialog: the pointer never leaves, and while the new overlay is up the tip's catcher gets no further mouse events either (quirk 26). | `widgets.HideTips` takes down every tip, shown or waiting; the shell's shape menu and every dialog in `dialogs` call it before they open. | `widgets.TestATipComesDownWhenSomethingElseOpens`, `widgets.TestAWaitingTipIsCancelledWhenSomethingElseOpens` |

## Adding an entry

Add the row, name the canary, and write the canary in the same commit as the
workaround. Bump Fyne in its own commit and run `make test`; a canary that
fails after the bump is the signal to remove its row and the code it guarded.
