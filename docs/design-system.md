# fynedesygn design system

This document is the rulebook the components in this module follow. It is
transcribed from the rationale comments in the `internal/gui` packages of
angou, nmsbonker and clockwork-orange, which carried these rules as three
hand-synced copies. Where the three apps disagreed, the choice is recorded
here with its reason.

The rules are binding for code in this module and are the recommended
defaults for programs that use it. A program that departs from one should say
why in a comment at the point of departure.

## Contents

- [Vocabulary](#vocabulary)
- [The two binding rules](#the-two-binding-rules)
- [Window skeleton](#window-skeleton)
- [Sections](#sections)
- [Colour, type and spacing](#colour-type-and-spacing)
- [Fonts](#fonts)
- [Progress and results](#progress-and-results)
- [State and freshness](#state-and-freshness)
- [Tables and lists](#tables-and-lists)
- [Forms](#forms)
- [Dialogs](#dialogs)
- [Documents](#documents)
- [Copy voice](#copy-voice)
- [Threading](#threading)
- [Testing](#testing)
- [Platform](#platform)
- [Numbers](#numbers)

## Vocabulary

Sections are assembled from a fixed vocabulary of components rather than from
raw Fyne widgets, so that every section states the same kind of thing the same
way. Reach for the component before writing a new arrangement of labels. A
shape that appears in a second section belongs in the library.

`Status` (`info`, `good`, `warn`, `bad`) is the one presentation type the whole
module shares. Colour comes from the active scheme through `Status`, so every
component stays legible in every scheme. Applications map their own domain
levels onto `Status` on their side of the boundary; the library never learns a
domain level.

## The two binding rules

**Nothing transient may reflow the interface.** Result banners and the
progress indicator float over the content as popups and never insert
themselves into a section's layout. Fixed-height panes stay fixed. A document
reserves each block's measured height whether or not the block is currently
rendered. An indicator inserted above the content pushes everything below it
down: the row the pointer is over moves out from under the pointer, mid-click,
which is jarring in a window and dangerous in one whose buttons include
Remove or Deploy. The same rule covers toolbars: a button that cannot be used
right now is disabled, never hidden, because a toolbar that changes width
mid-operation is a toolbar whose buttons move under the pointer.

**One scroll per section.** The shell owns one scroller for the content pane.
A widget that scrolls inside a scrolling section stops the page wherever the
pointer happens to rest, so a component either fills the space it is given
(document pane, code panel) or is a fixed height with its own scrollbar the
reader can aim at (log pane, a details pane). Fyne hands a wheel event to the
innermost scrollable under the pointer and does not pass it on; Fyne 2.8 also
wraps Markdown code blocks in a horizontal scroller, which is why the module
renders code blocks itself.

## Window skeleton

```
Border{
  top:    header      = VBox( Padded(HBox(bold app name, Spacer, header actions...)), Separator )
  bottom: status bar  = VBox( Separator, Padded(HBox(segments..., Spacer)) )
  center: HSplit( nav *widget.List, content *container.Scroll ).SetOffset(0.16)
}
```

- Default window 1180 x 760, `SetMaster()`, no menu bar, no top-level tabs.
- The nav is a list of section titles with icons, in an order the program
  chooses. Selecting a section replaces the content pane and scrolls to top.
- The status bar is a sequential `HBox`, never a `Border` centre: a centre
  region is sized from what is left over rather than from its own width, so a
  long path pushed two segments into each other. Sequential layout cannot
  overlap.
- The header carries the app name and at most a few program-level actions
  (Refresh, an "Open..." button). Appearance is a section, not a header
  control.
- F5 and Ctrl+R invalidate and reload. Programs may add shortcuts.
- Window geometry persistence is the program's business (Fyne has no resize
  callback; clockwork-orange polls the size every 500 ms and saves it through
  a coalesced writer).

## Sections

- A section is a stateless build function (`shell.Section`: `Title`, `Icon`,
  `Build`): given the shell, return the content. Any state change rebuilds
  the section from state rather than patching widgets. Only the builder knows
  every reason a button is disabled.
- Titles are computable before a Fyne app exists (for `--section` flag help
  and `--version`), so icons are deferred (`func() fyne.Resource`). Asking
  for a theme icon before `app.New` logs "Attempt to access current Fyne app
  when none is started".
- A section that holds live widgets or a scroll callback (a log list, a
  document pane following a scroller) implements `shell.Detacher`, and the
  shell calls `Detach` before building the replacement. Built first and dropped afterwards,
  a section registers its brand-new list and then has it thrown away by the
  very call that put it on screen, so a running job streams into nothing.
- Four redraw levels, cheapest first: `RedrawStatus` (the status bar);
  `Refresh` (the current section, keeping its scroll offset); `Rebuild`
  (both); `Invalidate` (the program's `OnInvalidate` hook drops loaded data
  and reloads, then `Rebuild`). Navigation starts at the top;
  a rebuild in place keeps the offset.
- **A control that starts work is affixed.** Every control that starts, cancels
  or commits work occupies the same place in its section however much of the
  section is scrolled. What scrolls is the material the control acts on -- the
  form, the table, the statistics, the prose -- never the control itself. So a
  section holding one is `Border(nil, actions, nil, nil, VScroll(body))`, with
  the controls in a fixed edge and a scroller in the centre, and not a column
  that happens to fit the window it was built on. Sections built around one big
  table or log use `Border(top, bottom, nil, nil, FixedHeight(table, N))` and no
  scroller of their own.
- The rule is stronger than "keep the action strip visible", and the reason is
  two rules up: **Fyne hands a wheel event to the innermost scrollable under the
  pointer and does not pass it on.** A control the user must scroll to is not
  merely inconvenient in a section that also holds a log or a table -- the
  wheel goes to whichever of them the pointer is over, so at an ordinary window
  height the control can be unreachable. The user does not have a hard-to-find
  button; they have no button. This was found twice in one week in
  clockwork-orange: a plugin form eleven fields long pushed its Download button
  off the bottom, and the Service section put Start, Stop, Restart, Install and
  Uninstall in the scrolling half of a split whose other half was a log pane.
  The gallery's own Log section had it too.
- The exception is a control that *is* the exhibit: a gallery card whose prose
  describes the button beside it, a row's own Remove button in a list of rows.
  There the control is the material, and it travels with the text or the row
  that explains it. The test is whether the control acts on the section or
  belongs to what scrolls.
- A program with more than a couple of these keeps the list of what must stay
  affixed beside its sections and walks it in a test, the way it keeps the list
  of what must stay reachable. `fynetest.ScrolledButtons` is the check:
  a named control must not appear in it.
- Use `Border`, not `VBox`, when a child must fill the section: the content
  pane is a scroller that sizes its content to at least the viewport.
- `AppTabs` sizes to its tallest item, so only the selected tab holds real
  content; the others hold an empty container until selected.
- Order a section by consequence: routine actions first, then cached or
  reversible state, then irreversible actions last with danger buttons. Put
  the report the reader came for directly under the heading; a form above it
  can push the first warning off a default-sized window.

## Colour, type and spacing

- Nine schemes ship: Breeze Dark, Breeze Light, Oxygen Dark, Adwaita Dark,
  Adwaita Light, Windows Dark, Windows Light, macOS Dark, macOS Light. The
  default is the platform's native dark scheme (Breeze Dark on Linux, Windows
  Dark on Windows, macOS Dark on macOS). The palette struct uses KDE's
  `.colors` vocabulary so a transcription can be diffed against its source;
  the theme translates to Fyne roles. Oxygen Dark's negative/positive/neutral
  are lightened from the upstream file, the one non-literal transcription,
  because upstream authored them for a light window. The Windows pair is
  transcribed from Windows 11 Fluent tokens, the macOS pair from Apple's
  system colours; translucent tokens are composited over their base because
  Fyne wants opaque fills.
- Corner radius and padding are palette tokens, because they are what differ
  between desktop styles: Breeze and Adwaita 2 and 3, Fluent 4 and 4, macOS 26
  Liquid Glass 8 and 4. Fyne cannot draw translucency or refraction, so on
  macOS roundness and the system colours carry the style.
- The palette declares `dark` itself and the theme ignores Fyne's
  `ThemeVariant` argument on purpose: honouring both could paint dark text on
  a dark window. Following the operating system's light/dark switch is not
  offered; schemes are compiled in.
- Icons always come from Fyne's default theme. A colour scheme says nothing
  about icons, and following the desktop icon theme would mean reading it at
  runtime.
- Translucent Fyne roles (focus, hover, pressed, scrollbar, disabled button)
  are derived from scheme colours with an alpha, so they stay in the scheme's
  hue.
- Base text size 12 (Fyne's own is 14; 14 in a dense window of tables and
  reports reads like a phone application). Offered sizes 10, 11, 12, 13, 14,
  16, 18. Heading 1.3x, subheading 1.15x, caption 0.85x.
- Breeze is a tighter, squarer style than Fyne's default; pulling the radius
  and padding in is what stops the window reading as "Fyne wearing KDE
  colours".
- Interface scale is a preference written into `FYNE_SCALE` after the app
  exists and before the window does; an explicit environment value wins.
  Changing it needs a restart of the window.
- Prose longer than a short label wraps (`widgets.Wrapped`, `DimWrapped`,
  `Note`). `Dim` is for short labels only: a long note in an unwrapped label
  sets the section's minimum width to the whole line, and the shell's
  two-axis scroller then scrolls sideways instead of wrapping.
- No arrow glyphs or other symbols outside the bundled font's coverage in
  labels: they draw from a fallback face and the shaper marks the run boundary
  as a missing glyph. Use words or theme icons.

## Fonts

- Fyne draws its own text and does not consult fontconfig, so the module scans
  font directories itself and hands bytes to the theme. Only `.ttf` and
  `.otf`; variable fonts are skipped (they cannot be rendered at a fixed
  style). A family with no regular face is not offered; missing bold or italic
  falls back to the family's regular face rather than an unrelated font. An
  unreadable font falls back to the bundled face: a missing font is not a
  reason to refuse to draw a window.
- Monospace is never taken from the interface family. A proportional family
  has no mono face, and substituting one silently would misalign the places
  that asked for monospace precisely because alignment mattered. A program
  may offer a separate console font for its log panes.
- Fonts are discovered, never embedded, so the module carries no font
  licences.

## Sections

- A section is built for two reasons and the shell tells them apart: the
  navigation arriving at it, and a rebuild of the section already on screen
  (an operation starting or finishing, a load landing, F5). `Build` runs for
  both. `shell.Arriver` — `OnArrive` on a `FuncSection` — runs for the first
  only.
- **A section that refetches on arrival must use the hook, not the builder.**
  Refetching in `Build` is a loop: the fetch finishing rebuilds the section
  that started it, which refetches. This is what a program wants when its data
  lives somewhere other programs write to, and it should be current when you
  navigate to it without being fetched on every redraw.
- `shell.Detacher` is the other end: the shell telling a section it is about
  to be replaced, so it can release live widgets a worker writes into.

## Guidance

- A control says what it is in its label. What it *does*, what it depends on and
  why a value matters go in a hover tip: `widgets.WithTip(control, text)`.
- **Tips are not a place to put a paragraph.** A sentence or two. They wrap at
  420 and appear after a second, which is longer than it takes to move onto a
  control and click it -- so a tip does not arrive at the moment of a click on a
  control nobody was asking about.
- Printing the same text under every control is the alternative and it is worse:
  it triples the height of a form and competes with the controls for attention.
  A note that is always visible should be a `DimWrapped` under a card, once, not
  under each row.
- **A tip is drawn in the window's tip layer, never in an overlay.** Fyne routes
  pointer events to the top overlay only, so a tip in one takes every click in
  the window — a control whose own tip was showing needed two clicks, and the
  first only took the tip down (quirk 26). The shell puts a
  `widgets.NewTipLayer` over every window it builds; a program that assembles
  its own window puts one last in its content, and gets a popup that eats the
  next click if it does not.
- **Anything that opens over the window calls `widgets.HideTips` first.**
  Clicking a control leaves the pointer where it was, so nothing tells its tip
  the control has been used and it sits behind the menu or dialog that just
  opened (quirk 27). The shell's own controls and every dialog in `dialogs` do
  this; a program that opens something of its own does the same.

## Progress and results

- Every long call goes through `shell.Perform` (an operation, one at a time)
  or `shell.Load` (a section's data load, allowed beside other work) and gets
  a busy indicator, not just the obviously slow ones. A
  window that sits still with no explanation reads as frozen, and the button
  that looks like it did nothing is the button that gets clicked twice.
- The busy indicator is a centred modal popup with an infinite progress bar,
  shown after 300 ms so quick calls never flash it. It is indeterminate on
  purpose: a bar that filled steadily would be inventing a number. Programs
  with real steps show a step list beside a log pane instead. The busy count
  is a counter, not a flag: loading a section can start more than one call.
- The `what` text is a present participle with an ellipsis ("Clearing the
  history..."), because it is the popup's caption.
- One operation at a time. A second request while one runs is refused, and
  explained **only when the refusal needs explaining** — `Shell.SayBusy`. The
  busy popup is modal, so while it is up a click cannot reach a control and
  there is nothing to tell anybody; a banner said anyway sits on screen after
  the work it described has finished. It is worth saying in the 300 ms before
  the popup appears, when a click does land, and for work reported through
  `Options.AlsoWorking`, which shows no popup at all.
- When it is said, it **names what is running** and says what can be done about
  it (`Shell.BusyReason`, which a program gating its own buttons should use so
  it says the same thing). "Something is already running" is a refusal with
  nothing in it. Cancel is offered only when the running operation has one, and
  the banner is an Info rather than a Warn: nothing is wrong, the answer is
  "not yet", and a warning stays twice as long.
- A cancellable job puts its Cancel **on the popup**, not in the toolbar
  behind it. The popup is modal, so a toolbar Cancel left enabled while
  everything else is disabled is on screen, enabled and unclickable from the
  moment the popup appears. `shell.PerformCancellable` when the runner owns
  the job; `shell.BusyCancellable` when the job holds the indicator itself
  (a log pump around the call, its own cancellation record, its own outcome
  banner). The button is dead once pressed: cancelling is rarely instant.
- Results are banners (`shell.Flash`, `Report`, `OK`): a non-modal popup
  centred near the bottom of the window, at most 720 wide, one at a time. Good holds 6 s, warn 12 s, bad
  stays until dismissed. Fyne animates properties, not opacity, so the fade
  runs on the banner's background rectangle; the text stays at full strength,
  which is also the accessible choice. A monotonic sequence number guards
  against a stale fade timer clearing a newer banner.
- Cancellation is not a failure: a cancelled context produces no error banner.
- Starting and finishing work rebuilds the section on screen (keeping its
  scroll offset) and redraws the status bar, so buttons disabled while
  something ran come back without anyone tracking a button list.

## State and freshness

- Loaded data carries an explicit loaded flag next to it. "Loaded and empty"
  and "not loaded" are different states: inferring "not loaded" from an empty
  slice makes an empty store load forever, raising a fresh dialog each time
  round. Set the flag before the goroutine starts so a rebuild mid-load does
  not start a second fetch.
- Program-wide state (a session, a service status) belongs to the shell's
  status bar (`Options.StatusBar`) and is loaded in `Options.OnStart` and
  `OnInvalidate`, not by whichever section happens to be open. A program
  that keeps the shell in a field stores it in `Options.OnCreate`, which runs
  before any builder; one whose theme depends on its own configuration (a
  console font) supplies `Options.Theme`; keyboard navigation beyond the
  shell's F5 goes through `Options.OnTypedKey`.
- Settings live in one file the user can read: `settings.Store`, reached
  through `Shell.Settings()`. One file per program, one section per top-level
  key, each decoded into the caller's own type. Keys beginning `fynedesygn.`
  are the library's; everything else is the program's and the library never
  looks inside it. Written a second after the last change and when the window
  closes.
- A section the running build never asks for is kept, not dropped, so an older
  binary cannot quietly delete what a newer one wrote.
- The file's extension chooses the format. JSON is the default and is in the
  core; `.yaml` and `.yml` need `settings/yamlcodec` imported for its effect,
  which keeps a YAML parser out of the binary of a program that writes JSON.
- Appearance (scheme, font, text size, scale) is one of the library's
  sections, `fynedesygn.appearance`. It used to live in `fyne.Preferences` and
  is read from there once, for an installation that predates the file. A stale
  value falls back silently; it is not an error worth a dialog.
- A one-run override of the scheme (for screenshots) is applied without
  persisting.

## The navigation's shape

- Two axes: `NavMode` (`NavLabels`, `NavIcons`, `NavHidden`) and
  `NavPlacement` (`NavLeft`, `NavTop`). Labels down the left is the default.
- **A program declares what it allows** through `Options.NavModes` and
  `NavPlacements`. Empty means today's window: one shape, no control in the
  header, no shortcut bound. A window four programs already ship does not
  change because the library learned a new trick.
- More than one allowed shape puts one stock control in the header — a menu of
  exactly what the program allows, with the current one ticked. One control
  whatever the number of choices: a control that changes shape according to how
  many options a program offers is one the user has to re-learn per program.
- `Ctrl+B` hides and restores, bound only where hiding is allowed.
- **The control is never hidden by the mode it sets.** It is in the header,
  which is always drawn, so `NavHidden` always has a way back.
- Icons on the left is a fixed strip, not a split: a divider on something sized
  to its icons has nothing to give. Along the top it is a row that scrolls when
  it overflows.
- In any icons-only shape each icon carries its section's title as a tip, and a
  section with no icon of its own gets a generic one. Without the first the
  window is a row of pictures to guess at; without the second it is a section
  nobody can reach.
- The shape is the window's, not the section's: changing it keeps the current
  section and does not rebuild it.

## Dividers

- A pane under a section — a log, an output, a preview — sits under a bar the
  user can drag: `Shell.VSplit(key, offset, top, bottom)`, `HSplit` for side by
  side. A fixed region is wrong for the pane that matters, because the section
  where the output is worth reading is whichever one is failing.
- **The position belongs to the shell, not to the section.** Sections rebuild on
  every refresh and every selection, and each rebuild makes a new split, so a
  position the section owned would be undone by the next status-driven redraw.
  Fyne's split reports no drag, so the shell reads each live divider just before
  the content holding it is replaced, and again when the window closes.
- Positions are keyed per program and kept in the settings file. Two splits with
  the same key share one position on purpose: a log pane under every section is
  one pane as far as the user is concerned.
- A stored position that would hide a pane (0 or 1) is ignored in favour of the
  caller's default. The pane that would be gone is often the one holding the
  control that would bring it back.
- The shell's own navigation divider is one of these, under `NavSplitKey`.
- `logpane.Options.Height` is a minimum, not a size: in a fixed region the
  minimum is the height; inside a divider it is how small the pane may be
  dragged.
- **A log pane wraps.** Every row is one line of monospace text in a
  `widget.List`, which is what lets a thousand lines scroll like a terminal --
  uniform heights, no layout pass per row -- and the cost was that a long line
  was drawn past the right edge and could not be read. The pane wraps the model
  to its own width instead, so a long line becomes several rows and the rows
  stay uniform. A continuation row is indented and marked with `logpane.Continued`,
  so where a message starts is visible without reading it. The width is read on the pump's tick, because Fyne has no resize
  callback (quirk 14). Copy renders the model, not the rows, so a line broken
  for the screen arrives whole on the clipboard.

## Tables and lists

- The detail table holds strings and one `Status` per row and hands back a
  `widget.Table`; every section that wanted one was writing the same three
  closures. Column widths are measured from content (7 px per rune plus 24,
  clamped to 70..460) or set explicitly.
- Set `Importance` before `SetText` on a recycled cell. `SetText` refreshes the
  label, and the refresh is where importance becomes a colour; the other way
  round a scrolled table paints each recycled cell in the colour of the row it
  last held.
- `UpdateHeader` is called with column -1 for the corner cell; guard it.
- Sortable headers are low-importance leading-aligned buttons, not labels: a
  label cannot receive the tap, and the arrow lives in the header text because
  a header cell holds one object. Re-sorting clears the selection and disables
  row actions; carrying it over would leave Remove armed against a different
  row than the one highlighted.
- Row-action buttons start disabled and are enabled by selection.
- List rows are rebuilt by index-captured closures. Clear `OnChanged` before
  `SetChecked` during recycling, because `Check.SetChecked` fires `OnChanged`.
- Tables and lists take all the space they are given; wrap them in
  `FixedHeight` to keep a section stable while it is being looked at.
- The label column in fact rows is 190 wide, and unranked rows reserve the
  26 px marker gutter so they line up with ranked ones.
- **A list that is edited rather than read uses `widgets.PickList`.** Fyne's
  list selects one row; taking six things off a list one at a time, with a
  reload between each, is six times the work for one decision. The tick is a
  checkbox in the row rather than a modifier held while clicking: a checkbox
  says what it does by being there, and Fyne does not pass the modifier to a
  list's selection callback anyway. `OnPicked` reports the count so the action
  button can say what it is about to do, and picks are row numbers, so a caller
  whose list has changed calls `ClearPicks`. It hands out the list through
  `Widget()` rather than being a widget wrapping one, as `logpane.Pane` does and
  for the same reason: a wrapper has to pass on everything its container does,
  and a list that is resized but never laid out builds no rows at all.

## Forms

- A form model (`forms.Form`) holds its widgets apart from the section so
  the document-to-widget round trip is testable headlessly and so Revert can
  revert: rebuilding instead would leave typed values on screen until the
  reload landed.
- A `setting` re-entry guard wraps programmatic writes (`Form.Set`), because
  `Check.SetChecked` and friends fire change handlers. A section build must
  never schedule a save.
- Writes are coalesced: every change lands in the document and schedules one
  save a second after the last change. The first window-size reading is a
  baseline so start-up never announces "Saved".
- A slider and its entry (`forms.SliderEntry`) stay in step but only one
  commits: the slider on `OnChangeEnded`, the entry on `OnSubmitted`. A settings file written per
  pixel is a settings file written four hundred times to move one number.
  Update the card in place so focus and scroll survive.
- Numeric entries validate a range and sit in a `FixedWidth` of 120.
- Fields that share a group pack onto one row.

## Dialogs

- A destructive confirmation (`dialogs.ConfirmDestructive`) names the action
  on a danger button, keeps Cancel as the safe default, and its detail says
  what is *not* touched as well as what is. Not saying so is how a confirmation dialog becomes the
  thing people click through without reading. The canonical shape is three
  parts: where the result goes, what happens to what is there, what is left
  alone.
- File and folder choosers (`dialogs.ChooseFile`, `ChooseFolder`,
  `WithBrowse`) are 760 x 520 and are resized *after* `Show`.
  Before `Show` the dialog has no window, and `Resize` asks it for its minimum
  size, which in Fyne 2.8.1 dereferences that nil window and takes the process
  with it. A regression test taps every chooser button.
- A chooser starts in the directory of the field's current path, then its
  parent, then home.
- Fyne 2.8.1 has no multi-select chooser; an "Add..." button reopens a
  single-file chooser.
- Blocking questions from a worker goroutine (`dialogs.Decide`) hop to the
  UI thread with `fyne.Do`, put a dialog up, and wait on a channel; a `sent`
  guard makes every close path answer exactly once. Focus the entry after
  showing.
- Buttons that open a dialog end in an ellipsis.
- A validation failure inside an editor dialog is shown in the dialog, not as
  a banner behind it: the refusal names the line, and the editor is where to
  fix it.

## Documents

- Markdown renders per block with `widget.RichText` (`markdown.Pane`, or
  `markdown.Section` for a page that is one document); code blocks (fenced
  or indented) render in the module's own code panel, which wraps rather
  than scrolling sideways, and pipe tables render as a grid. Rationale and
  the block-level virtualisation are in [markdown.md](markdown.md).
- Mermaid fences render to PNG at development time and are embedded; see
  [mermaid.md](mermaid.md).
- Long-form documentation that already lives in a README should be shown
  from that README (embedded), never restated: a shortened copy becomes a
  second, less careful copy to keep in sync.

## Copy voice

Sentence case, second person, plain words, consequences named. Trailing
ellipsis on buttons that open a dialog. Destructive dialogs state what is not
touched. Empty states explain themselves. Screenshot alt text describes the
picture; it is the only description a screen-reader user gets, and a stale one
is worse than none.

## Threading

- Every core call runs on a goroutine and every callback hops back to the UI
  thread with `fyne.Do`. Never `fyne.DoAndWait`. `ui` fields are written only
  on the UI thread.
- Streams (`logpane.Model` and `Pane`) are written from workers behind a
  mutex and drawn on a 100 ms timer, never per line: a hundred `fyne.Do` calls a second would make
  the window slower than the job. A dirty flag makes a quiet job cost one
  comparison per tick. Stopping a ticker does not close its channel; the pump
  has its own quit channel and draws once more after stopping.
- Auto-scroll without a scroll callback: compare the list's offset to the
  offset the last auto-scroll left, with a tolerance of 4 because
  `ScrollToBottom` lands on fractional values. Reset the remembered offset on
  detach.
- When the shell is not on screen (headless tests), work runs inline. The
  Fyne test driver runs `fyne.Do` inline on the calling goroutine rather than
  serialising onto a main loop, so a worker refreshing a widget genuinely
  races the test driving it.

## Testing

- Tests run headless with `fyne.io/fyne/v2/test`; `make test` must pass on a
  machine with no display server.
- Tests pin behaviour, not layout: what a cell says, when a button is enabled,
  what the log keeps. A test asserting that a section holds a Border holding a
  VBox tests the code against itself and breaks on every rearrangement.
- Test names are sentences naming the bug they prevent.
- Cell contents are asserted through extracted pure functions, because a
  table builds its cells lazily and there is nothing in the object tree until
  it is on a canvas.
- Tests sandbox `HOME` and the `XDG_*` directories into `t.TempDir()`; a
  result that depends on the developer's machine is not a test.
- Every Fyne quirk the module works around has a canary test named after the
  quirk. When a canary fails, Fyne has changed and the workaround can go.
- Screenshots come from `tools/screenshot.sh`, which drives a program through
  `--section` and `--scheme` under a throwaway HOME, confirms the window has
  focus and checks the captured aspect ratio before trusting a shot. Alt text
  describes what is in the image.
- Under the Fyne test driver `fyne.Do` runs inline on the calling goroutine.
  A test must not inspect widgets while a worker goroutine (a log pump, a
  blocking question) draws them; stop and join the worker first, or drive the
  UI-thread half directly.

## Platform

- Linux, macOS, Windows. Platform code is build-tagged.
- Linux: before the toolkit starts, read the cursor theme and size from
  `kcminputrc`, then GTK 4, GTK 3 and `.gtkrc-2.0` settings, and export
  `XCURSOR_THEME` and `XCURSOR_SIZE` unless already set. GLFW's Wayland
  backend lacks `cursor-shape-v1`, and Plasma exports these only to XWayland
  clients. Read the files directly; do not shell out.
- Wayland matches a window to its desktop entry by `app_id`, which Fyne
  derives from the app ID. The desktop file basename must equal the app ID or
  the taskbar icon is lost.
- Font directories per platform: the XDG and `/usr/share` set on Linux,
  `/System/Library/Fonts` and `/Library/Fonts` on macOS, `C:\Windows\Fonts`
  and the per-user fonts folder on Windows.
- The GUI needs CGO; cross-compiling it needs a C toolchain per target.
  Command-line front ends stay `CGO_ENABLED=0`.

## Numbers

| Thing | Value |
|---|---|
| Default window | 1180 x 760 |
| Nav split offset | 0.16 |
| Base text size | 12 (offered: 10, 11, 12, 13, 14, 16, 18) |
| Padding / corner radius | 3 / 2 (Breeze, Oxygen, Adwaita); 4 / 4 (Windows); 4 / 8 (macOS) |
| Busy popup delay | 300 ms |
| Busy bar width | 320 |
| Banner width | min(720, canvas width - 40) |
| Banner offset from bottom | 56 |
| Banner hold: good / warn / bad | 6 s / 12 s / until dismissed |
| Banner fade | 700 ms |
| Log redraw interval | 100 ms |
| Log follow tolerance | 4 |
| Log pane height | 360 (clockwork-orange) or 420 (nmsbonker); library default 360 |
| Label column / marker gutter | 190 / 26 |
| Numeric entry width | 120 |
| Table measure | 7 px per rune + 24, clamped 70..460 |
| Thumbnail cell | 64 |
| Save coalesce delay | 1 s |
| Size poll interval | 500 ms |
| Destructive confirm | 560 x 320 |
| Confirm with body | 600 x 360 |
| Path dialog | 620 x 340 |
| File / folder chooser | 760 x 520 |
| About logo | 72 x 72, fill contain |
| Document overscan | 0.5 viewport each side |

Fixed pixel values above assume the base text size. Heights derived from a
banner or strip's minimum size should be computed from the theme's text size
in this module, not copied as constants: the apps' `48` and `18` were tuned to
size 12 only.
