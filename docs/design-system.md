# fynedesygn design system

This page holds the rules that the components in this module follow. The rules
come from the comments in the `internal/gui` packages of angou, nmsbonker and
clockwork-orange, which held these rules as three copies that people kept in
agreement by hand. Where the three programs disagreed, this page records the
choice and the reason for it.

The rules are mandatory for code in this module, and they are the recommended
defaults for a program that uses the module. If your program does not follow a
rule, write the reason in a comment at that point.

This page is written in plain technical English; see [How this is
written](style.md).

## Contents

- [Vocabulary](#vocabulary)
- [The two binding rules](#the-two-binding-rules)
- [Window skeleton](#window-skeleton)
- [Glance windows](glance.md) (separate document)
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

Build a section from a fixed set of components and not from Fyne widgets, so
that every section reports the same kind of thing in the same way. Use a
component before you write a new arrangement of labels. A shape that appears in
a second section belongs in the library.

`Status` is the one presentation type that the whole module shares. Its values
are `info`, `good`, `warn` and `bad`. Colour comes from the active scheme
through `Status`, so every component is legible in every scheme. Your program
converts its own levels to `Status` on its side of the boundary, and the
library never learns a level of your domain.

## The two binding rules

**Nothing temporary may move the interface.** A result banner and the progress
indicator are popups above the content, and they never enter the layout of a
section. A pane of fixed height keeps its height. A document keeps the measured
height of each block, whether or not it draws that block now.

An indicator above the content moves everything below it down. The row below
the pointer then moves away from the pointer during a click. That is
unpleasant in any window, and it is dangerous in a window whose buttons
include Remove or Deploy.

The same rule covers a toolbar. Disable a button that the user cannot use now,
and never hide it. A toolbar that changes width during an operation is a
toolbar whose buttons move below the pointer.

**One scroller for each section.** The shell owns one scroller for the content
pane. A widget that scrolls inside a section that scrolls stops the page
wherever the pointer is.

A component therefore does one of two things. It fills the space it receives,
which is what a document pane and a code panel do. Or it has a fixed height
and its own scrollbar that the reader can point at, which is what a log pane
and a details pane do.

Fyne gives a wheel event to the innermost scroller below the pointer and does
not send it on. Fyne 2.8 also puts a Markdown code block in a horizontal
scroller, which is why this module draws a code block itself.

## Window skeleton

```
Border{
  top:    header      = VBox( Padded(Border(trailing: VBox(HBox(header actions...)),
                                           centre:   bold app name, or the
                                                     navigation when it is
                                                     along the top)), Separator )
  bottom: status bar  = VBox( Separator, Padded(HBox(segments..., Spacer)) )
  center: HSplit( nav *widget.List, content *container.Scroll ).SetOffset(0.16)
          nav rows = sections, plus a heading per group, minus a closed
                     group's members
}
```

- The default window is 1180 x 760, with `SetMaster()`, no menu bar and no
  tabs at the top level.
- The navigation is a list of section titles with icons, in an order that the
  program chooses. When the user selects a section, the shell replaces the
  content pane and scrolls to the top.
- The status bar is an `HBox` in sequence and never the centre of a `Border`.
  A centre region takes the space that is left rather than its own width, so a
  long path moved two segments into each other. A layout in sequence cannot
  overlap.
- The header holds the name of the program and a small number of actions for
  the program, such as Refresh or an "Open..." button. Appearance is a section
  and not a control in the header.
- F5 and Ctrl+R invalidate the data and load it again. Your program can add
  more shortcuts.
- Your program keeps the geometry of its window. Fyne has no resize callback;
  clockwork-orange reads the size every 500 ms and saves it through a writer
  that collects the changes.
- This skeleton is the application window. A small panel that stays above
  other windows, and that a person reads without touching, is a different
  shape with its own rules: it has no header, no navigation and no scroller.
  See [glance.md](glance.md).

## Sections

- A section is a build function that holds no state. `shell.Section` is
  `Title`, `Icon` and `Build`: it receives the shell and returns the content.
  Every change of state rebuilds the section from the state, and does not
  change widgets in position. Only the builder knows every reason that a
  button is disabled.
- A program can calculate a title before a Fyne app exists, for the help of
  the `--section` flag and for `--version`. An icon is therefore a
  `func() fyne.Resource`. A request for a theme icon before `app.New` writes
  "Attempt to access current Fyne app when none is started".
- A section that holds live widgets or a scroll callback implements
  `shell.Detacher`. A log list and a document pane that follows a scroller are
  two examples. The shell calls `Detach` before it builds the replacement. If
  the shell built first and detached afterwards, a section would register its
  new list and the same call would then discard it, and a job that was running
  would write into nothing.
- There are four levels of redraw. `RedrawStatus` draws the status bar.
  `Refresh` draws the current section and keeps its scroll offset. `Rebuild`
  does both. `Invalidate` calls the `OnInvalidate` hook of the program, which
  discards the loaded data and loads it again, then calls `Rebuild`.
  Navigation starts at the top, and a rebuild in position keeps the offset.
- **A control that starts work stays in position.** Every control that starts,
  cancels or commits work keeps the same place in its section, however far the
  user scrolls. What scrolls is the material that the control acts on: the
  form, the table, the statistics or the prose. The control itself never
  scrolls.

  A section that holds such a control is therefore
  `Border(nil, actions, nil, nil, VScroll(body))`. The controls are at a fixed
  edge and the scroller is in the centre. Do not use a column that fits the
  window you built it on. A section built around one large table or log uses
  `Border(top, bottom, nil, nil, FixedHeight(table, N))` and has no scroller of
  its own.
- This rule is stronger than "keep the action strip visible", and the reason is
  two rules above: **Fyne gives a wheel event to the innermost scroller below
  the pointer and does not send it on.**

  A control that the user must scroll to is more than inconvenient in a section
  that also holds a log or a table. The wheel goes to whichever of them is
  below the pointer, so at a normal window height the user cannot reach the
  control. The user does not have a button that is difficult to find. The user
  has no button.

  clockwork-orange found this two times in one week. A plugin form of eleven
  fields moved its Download button below the bottom of the window. The Service
  section put Start, Stop, Restart, Install and Uninstall in the half of a
  split that scrolled, and the other half was a log pane. The Log section of
  the gallery had the same fault.
- There is one exception: a control that *is* the material. A gallery card
  whose prose describes the button beside it, and the Remove button of one row
  in a list of rows, are both examples. There the control travels with the text
  or the row that explains it. Ask whether the control acts on the section or
  belongs to what scrolls.
- A program with several of these keeps a list of the controls that must stay
  in position, beside its sections, and reads that list in a test. It keeps a
  list of what must stay reachable in the same way. `fynetest.ScrolledButtons`
  is the check: a control on the list must not appear in its result.
- Use `Border` and not `VBox` when a child must fill the section. The content
  pane is a scroller that gives its content at least the size of the viewport.
- `AppTabs` takes the height of its tallest item, so put real content only in
  the selected tab. The other tabs hold an empty container until the user
  selects them.
- Order a section by consequence. Put the routine actions first, then the state
  that is cached or reversible, then the actions that you cannot reverse, with
  danger buttons. Put the report that the reader came for directly below the
  heading. A form above the report can move the first warning below the bottom
  of a default window.

## Colour, type and spacing

- The module has nine schemes: Breeze Dark, Breeze Light, Oxygen Dark, Adwaita
  Dark, Adwaita Light, Windows Dark, Windows Light, macOS Dark and macOS
  Light. The default is the native dark scheme of the platform: Breeze Dark on
  Linux, Windows Dark on Windows, and macOS Dark on macOS.
- The palette structure uses the `.colors` vocabulary of KDE, so that you can
  compare a copy against its source. The theme converts the palette to Fyne
  roles.
- The negative, positive and neutral colours of Oxygen Dark are lighter than
  those in the file upstream. This is the one colour set that is not a literal
  copy, because the authors upstream wrote it for a light window.
- The Windows pair comes from the Windows 11 Fluent tokens, and the macOS pair
  from the system colours of Apple. A translucent token is composited over its
  base colour, because Fyne needs an opaque fill.
- The corner radius and the padding are palette tokens, because they are what
  differs between desktop styles: 2 and 3 for Breeze and Adwaita, 4 and 4 for
  Fluent, and 8 and 4 for macOS 26 Liquid Glass. Fyne cannot draw translucency
  or refraction, so on macOS the roundness and the system colours carry the
  style.
- The palette declares `dark` itself, and the theme ignores the `ThemeVariant`
  argument of Fyne by design. To use both could draw dark text on a dark
  window. The module does not follow the light and dark switch of the
  operating system, and it compiles the schemes in.
- Icons always come from the default theme of Fyne. A colour scheme says
  nothing about icons, and to follow the icon theme of the desktop would mean
  to read it at run time.
- The translucent Fyne roles are focus, hover, pressed, scrollbar and disabled
  button. They come from the scheme colours with an alpha value, so they keep
  the hue of the scheme.
- The base text size is 12. Fyne uses 14, and 14 in a dense window of tables
  and reports looks like a telephone application. The sizes offered are 10, 11,
  12, 13, 14, 16 and 18. A heading is 1.3 times the base, a subheading is 1.15
  times, and a caption is 0.85 times.
- Breeze is a tighter and squarer style than the default of Fyne. The smaller
  radius and padding are what stop the window from looking like Fyne with KDE
  colours on it.
- The interface scale is a preference that the shell writes into `FYNE_SCALE`
  after the app exists and before the window exists. A value already in the
  environment wins. A change needs a restart of the window.
- Prose that is longer than a short label wraps. Use `widgets.Wrapped`,
  `DimWrapped` or `Note`. Use `Dim` for short labels only. A long note in a
  label that does not wrap sets the minimum width of the section to the length
  of the line, and the scroller of the shell then scrolls sideways instead.
- Do not put an arrow glyph, or another symbol that the bundled font does not
  have, in a label. It draws from a fallback face, and the shaper marks the end
  of the run as a missing glyph. Use words or theme icons.

## Fonts

- Fyne draws its own text and does not read fontconfig, so this module
  examines the font directories itself and gives the bytes to the theme. It
  accepts `.ttf` and `.otf` only, and it does not use a variable font, which it
  cannot draw at a fixed style.
- The module does not offer a family that has no regular face. Where a family
  has no bold or italic face, the module uses the regular face of that family
  and not an unrelated font. Where a font will not read, the module uses the
  bundled face. An absent font is not a reason to refuse to draw a window.
- Never take the monospace face from the interface family. A proportional
  family has no monospace face, and to substitute one without a message would
  misalign the places that asked for monospace because the alignment was
  important. Your program can offer a separate console font for its log
  panes.
- The module finds fonts and never embeds them, so it carries no font
  licences.
- **Choose a font in `dialogs.ChooseFont` and not in a dropdown.** The dialog
  shows a sample in the family that is highlighted: the name of the family, a
  line of prose, and the shapes that separate one family from another. It
  commits nothing until the user presses Choose. A dropdown of names tells the
  user nothing about any of them, so to choose a font meant to apply one to the
  whole window to see it, then apply another to return.
- **In a picker, the pointer decides nothing.** `widget.List.OnHighlighted`
  fires when the pointer passes over a row, and also when the arrow keys move.
  A preview that follows it therefore follows the mouse on its way to the
  confirm button. If the preview also selects the row, the list discards the
  click that follows, because `Select` returns early for a row that is already
  selected. The cursor moves on the arrow keys and on a click, and a hover only
  draws. Put Enter on a widget that the list cannot take the focus from, which
  in practice is the filter box.
- **A monospace picker offers monospace families only**
  (`theme.MonospaceNames`). Measure the advance width, and do not filter by
  name: "Meslo LGLDZ Nerd Font Propo" is the proportional one. Keep the family
  that is already set on the list, even when it fails the measurement, because
  a picker that cannot show the current setting has lost it.
- **A font preview draws from the file and never through a theme.**
  `canvas.Text.FontSource` names the face, and Fyne reads it before any theme
  or scope. A `container.ThemeOverride` appears to work and does not. Fyne
  measures text through a path that receives no object; it caches the face
  against a scope that the text objects do not always carry; and it resolves
  monospace text through a list of faces and not the face that the theme named.
  All three lose the family between the decision and the drawing. The result is
  a preview that reports the family it received and draws the glyphs of a
  different one.
- **A preview must fail where the user can see it.** It follows the highlight
  however the highlight moved. `widget.List` reports the keyboard through
  `OnHighlighted` and the mouse through `OnSelected`, and a preview that
  listens to one of them shows a family that nobody is pointing at. The preview
  also reports what it drew.

  Most families on a Linux machine are script fonts with no Latin letters, so a
  Latin sample in one of them is the glyphs of another font. That result is the
  same for every such family, and looks the same as a preview that has stopped
  working. `theme.Font.Missing` counts what a family cannot draw, and `Covered`
  returns what it can.
- **The list of names draws in the interface font, by design.** Fyne draws
  every widget in the theme of the app, so a row in its own face needs a theme
  of its own and a font read from the disk. This machine has 311 families at
  2.5 MB each: 778 MB and 731 ms to draw one menu, in a cache that never
  releases. `theme.PreviewFont` reads a family and does not keep it, so one
  font is in memory at a time, and to look at every family costs what one
  family costs.

## Sections

- The shell builds a section for two reasons, and it separates them. The first
  is the navigation that arrives at the section. The second is a rebuild of the
  section that is on screen, when an operation starts or ends, a load arrives,
  or the user presses F5. `Build` runs for both. `shell.Arriver`, which is
  `OnArrive` on a `FuncSection`, runs for the first reason only.
- **A section that reads its data again on arrival must use the hook and not
  the builder.** A read in `Build` is a loop: the end of the read rebuilds the
  section that started it, which reads again. Use the hook when your data is in
  a place that other programs write to, and it must be current when the user
  navigates to it, but not read at every redraw.
- `shell.Detacher` is the opposite end. The shell tells a section that it is
  about to replace it, so that the section can release the live widgets that a
  worker writes into.

## Guidance

- A control says what it is in its label. Put what it *does*, what it depends
  on, and why a value is important in a hover tip: `widgets.WithTip(control,
  text)`.
- **A tip is not a place for a paragraph.** Write one or two sentences. A tip
  wraps at 420 and appears after one second, which is longer than a person
  needs to move onto a control and click it. A tip therefore does not arrive at
  the moment of a click on a control that nobody asked about.
- The alternative is to print the same text below every control, and it is
  worse. It makes a form three times as tall, and it competes with the controls
  for the attention of the reader. A note that is always visible is one
  `DimWrapped` below a card, and not one below each row.
- **Draw a tip in the tip layer of the window and never in an overlay.** Fyne
  sends a pointer event to the top overlay only, so a tip in an overlay
  receives every click in the window. A control that showed its own tip needed
  two clicks, and the first click only removed the tip (quirk 26). The shell
  puts a `widgets.NewTipLayer` above every window it builds. A program that
  builds its own window puts one last in its content. Without it, the program
  has a popup that receives the next click.
- **Anything that opens above the window calls `widgets.HideTips` first.** A
  click on a control leaves the pointer where it was, so nothing tells the tip
  that the user has used the control, and the tip stays behind the menu or
  dialog that opened (quirk 27). The controls of the shell and every dialog in
  `dialogs` do this, and a program that opens something of its own does the
  same.

## Progress and results

- Send every long call through `shell.Perform`, which runs one operation at a
  time, or through `shell.Load`, which loads the data of a section and can run
  beside other work. Both show a busy indicator. Do this for every call and not
  only for the calls you expect to be slow. A window that does not move and
  gives no reason looks as if it has stopped, and a button that appears to have
  done nothing is a button that the user clicks two times.
- The busy indicator is a modal popup in the centre with a progress bar that
  has no end. It appears after 300 ms, so a quick call never shows it. The bar
  has no value by design, because a bar that filled at a constant rate would
  invent a number. A program with real steps shows a step list beside a log
  pane instead. The busy state is a count and not a flag, because the load of a
  section can start more than one call.
- The `what` text ends in an ellipsis, as in "Clearing the history...", because
  it is the caption of the popup.
- Run one operation at a time. Refuse a second request while one operation
  runs, and explain the refusal **only when the explanation is useful**. Use
  `Shell.SayBusy`.

  The busy popup is modal, so while it is present a click cannot reach a
  control and there is nobody to tell. A banner sent anyway stays on screen
  after the work it described has finished. Send the banner in the 300 ms
  before the popup appears, when a click does arrive, and for work reported
  through `Options.AlsoWorking`, which shows no popup.
- When you do send it, **name what is running** and say what the user can do.
  Use `Shell.BusyReason`, and use it also in a program that disables its own
  buttons, so that both say the same thing. "Something is already running" is a
  refusal with no content. Offer Cancel only when the operation that runs has a
  cancel. Send the banner as Info and not as Warn: nothing is wrong, the answer
  is "not yet", and a warning stays on screen two times as long.
- Put the Cancel of a job **on the popup** and not in the toolbar behind it.
  The popup is modal, so a Cancel in the toolbar that stays enabled while
  everything else is disabled is visible, enabled and impossible to click from
  the moment the popup appears. Use `shell.PerformCancellable` when the runner
  owns the job. Use `shell.BusyCancellable` when the job holds the indicator
  itself, with a log pump around the call, its own record of the cancellation,
  and its own banner for the result. The button stops after one press, because
  a cancellation is seldom immediate.
- A result is a banner: `shell.Flash`, `Report` or `OK`. It is a popup that is
  not modal, in the centre near the bottom of the window, at most 720 wide, and
  one at a time. A good banner stays 6 s, a warning stays 12 s, and a bad
  banner stays until the user removes it.
- Fyne animates a property and not the opacity, so the fade runs on the
  background rectangle of the banner. The text keeps its full strength, which
  is also the choice that is easier to read. A sequence number that always
  increases stops an old fade timer from removing a newer banner.
- A cancellation is not a failure, and a context that the program cancelled
  produces no error banner.
- The start and the end of work rebuild the section on screen, keep its scroll
  offset, and draw the status bar again. A button that the program disabled
  while the work ran therefore returns, and nobody keeps a list of buttons.

## State and freshness

- Keep a loaded flag beside your loaded data. "Loaded and empty" and "not
  loaded" are two different states. If you take "not loaded" from an empty
  slice, an empty store loads without end and opens a new dialog each time. Set
  the flag before the goroutine starts, so that a rebuild during the load does
  not start a second read.
- State that belongs to the whole program, such as a session or the status of
  a service, belongs in the status bar of the shell (`Options.StatusBar`). Load
  it in `Options.OnStart` and `OnInvalidate`, and not in whichever section is
  open. A program that keeps the shell in a field stores it in
  `Options.OnCreate`, which runs before any builder. A program whose theme
  depends on its own configuration, such as a console font, supplies
  `Options.Theme`. Keyboard navigation beyond the F5 of the shell goes through
  `Options.OnTypedKey`.
- Keep the settings in one file that the user can read. Use `settings.Store`,
  which you reach through `Shell.Settings()`. There is one file for each
  program and one section for each key at the top level, and the store decodes
  each section into the type of the caller. A key that starts with
  `fynedesygn.` belongs to the library. Every other key belongs to the program,
  and the library never reads inside it. The store writes the file one second
  after the last change, and when the window closes.
- The store keeps a section that the running program never asks for, and does
  not discard it. An older program therefore cannot delete what a newer one
  wrote.
- The extension of the file chooses the format. JSON is the default and is in
  the core package. For `.yaml` and `.yml`, import `settings/yamlcodec` for its
  effect. This keeps a YAML parser out of the binary of a program that writes
  JSON.
- Appearance is one of the sections of the library, `fynedesygn.appearance`. It
  holds the scheme, the font, the text size and the scale. It was in
  `fyne.Preferences` before, and the library reads it from there one time, for
  an installation that is older than the file. Where a value is out of date,
  the library uses the default and shows no message. That is not an error that
  needs a dialog.
- A one-run override of the scheme (for screenshots) is applied without
  persisting.

## The navigation's shape

- There are two axes. `NavMode` is `NavLabels`, `NavIcons` or `NavHidden`.
  `NavPlacement` is `NavLeft` or `NavTop`. The default is labels down the
  left.
- **Your program declares what it permits**, through `Options.NavModes` and
  `NavPlacements`. An empty value means the window that programs have today:
  one shape, no control in the header, and no shortcut. A window that four
  programs already supply does not change because the library learned a new
  function.
- Where a program permits more than one shape, the shell puts one control in
  the header. That control is a menu of what the program permits, and it marks
  the current shape. There is one control for any number of choices. A control
  that changes shape with the number of options is a control that the user
  must learn again for each program.
- `Ctrl+B` hides the navigation and shows it again. The shell binds the key
  only where the program permits hiding.
- **The mode that the control sets never hides the control.** It is in the
  header, which the shell always draws, so `NavHidden` always has a way
  back.
- Icons down the left are a fixed strip and not a split. A divider on something
  sized to its icons has nothing to give. Along the top, the navigation is a row
  that scrolls when it is too long.
- **Along the top means in the header**, where the name of the program would
  be, and not in a strip below it. The window is one row shorter, and the
  window does not say the name two times, because the shell already puts it in
  the window title. Down the left, the name stays, where it reads as the owner
  of the list. When the user hides the navigation, the name returns, because
  there is nothing else to put in that place.
- In a shape that shows icons only, each icon carries the title of its section
  as a tip, and a section with no icon of its own receives a general icon.
  Without the tip, the window is a row of pictures that the user must guess at.
  Without the general icon, the window has a section that nobody can reach.
- The shape belongs to the window and not to the section. A change of shape
  keeps the current section and does not rebuild it.
- The actions in the header keep their own height and stay against the top of
  the header row. A `Border` makes what it receives as tall as its row, and the
  row is two buttons tall whenever a group is open along the top. A Refresh
  button drawn two times as tall as every other button says that it is two
  times the control.

## Groups in the navigation

- **A navigation of more than seven or eight entries stops reading as a set of
  places and starts reading as a list of items.** The entries that caused this
  are usually of one kind: three sources of pictures, four reports, or five
  importers. What the reader wants from the list is the kind, until the moment
  they want one entry. `Options.Groups []NavGroup` puts them below a heading.
- **A group is not a section.** It has no page, `Select` does not reach it,
  `Current` never returns it, and `Ctrl+1..9` does not count it. Those keys stay
  attached to the sections in the order that the program listed them, so a
  group of three does not change the numbers of the sections below it.
- **The members of a group are section titles.** The shell matches them and
  ignores upper and lower case. Keep the members together in the order of
  `Sections`: the shell draws the list in that order, and it puts the heading
  where the first member would be. The shell ignores a title that names no
  section, and does not draw a group whose members all name no section. A
  program that builds its sections conditionally therefore lists the group in
  both cases.
- **To close a group does not move the reader.** The page on screen stays and
  the shell does not rebuild it. When the user closes the group that holds the
  page, nothing is marked, which is correct, and when they open it again the
  mark returns.
- **To navigate to a member opens its group.** `--section`, `Select` and
  `Ctrl+1..9` are all navigation, and a window that moved somewhere without
  showing where has lost the reader.
- **Every shape draws a group.** Down the left with labels, it is a row that
  opens, with its members indented. Down the left with icons only, it is a
  button with its members indented below it. Along the top, it is a button on
  the first row with its members on a second row, and never in the middle of
  the row that the reader chooses a group from.
- **The group button is marked while the page is one of its members**, whether
  the group is open or closed. When it is open, two buttons are marked for one
  section, which is correct: the branch and the leaf.
- The shell keeps the open and closed state under `fynedesygn.nav.groups`. It
  stores the closed groups, so a group that a program adds later starts
  open.

## Dividers

- A pane below a section, such as a log, an output or a preview, sits below a
  bar that the user can drag. Use `Shell.VSplit(key, offset, top, bottom)`, or
  `HSplit` for two panes beside each other. A region of fixed size is wrong for
  the pane that is important, because the section whose output is worth reading
  is whichever section is failing.
- **The position belongs to the shell and not to the section.** The shell
  rebuilds a section at every refresh and every selection, and each rebuild
  makes a new split. A position that the section owned would therefore be lost
  at the next redraw. The split of Fyne does not report a drag, so the shell
  reads each divider that is on screen before it replaces the content that
  holds it, and again when the window closes.
- The shell keys a position for each program and keeps it in the settings file.
  Two splits with the same key share one position, by design: a log pane below
  every section is one pane to the user.
- The shell ignores a stored position of 0 or 1, which would hide a pane, and
  uses the default of the caller instead. The pane that would be absent often
  holds the control that would bring it back.
- The navigation divider of the shell is one of these, under `NavSplitKey`.
- `logpane.Options.Height` is a minimum and not a size. In a region of fixed
  size, the minimum is the height. Inside a divider, it is the smallest size
  that the user can drag the pane to.
- **A log pane wraps its text.** Each row is one line of monospace text in a
  `widget.List`. That is what lets a thousand lines scroll like a terminal: the
  heights are the same and there is no layout pass for each row. The cost was
  that the pane drew a long line past the right edge, where nobody could read
  it.

  The pane now wraps the model to its own width, so a long line becomes several
  rows and the rows keep the same height. The pane indents a row that
  continues a line and marks it with `logpane.Continued`, so the reader sees
  where a message starts without reading it. The pane reads the width at the
  tick of the pump, because Fyne has no resize callback (quirk 14). Copy uses
  the model and not the rows, so a line that the pane divided for the screen
  arrives complete on the clipboard.

## Tables and lists

- The detail table holds strings and one `Status` for each row, and returns a
  `widget.Table`. Every section that wanted one was writing the same three
  closures. The table measures its column widths from the content, at 7 px for
  each rune plus 24, between 70 and 460, or you set them yourself.
- Set `Importance` before `SetText` on a cell that the table uses again.
  `SetText` refreshes the label, and the refresh is where the importance
  becomes a colour. In the other order, a table that the user scrolled paints
  each cell in the colour of the row that the cell held before.
- `UpdateHeader` receives column -1 for the corner cell. Test for that value.
- A header that sorts is a button of low importance, aligned to the leading
  edge, and not a label. A label cannot receive the tap, and the arrow is in the
  text of the header because a header cell holds one object.
- A new sort clears the selection and disables the row actions. To keep the
  selection would leave Remove ready to act on a row that is not the row the
  table marks.
- A row action button starts disabled, and the selection enables it.
- The table rebuilds a list row with a closure that captured the index. Clear
  `OnChanged` before `SetChecked` while the table uses a row again, because
  `Check.SetChecked` fires `OnChanged`.
- A table and a list take all the space they receive. Put them in
  `FixedHeight` to keep a section still while a person reads it.
- The label column in a fact row is 190 wide. A row with no rank keeps the
  marker space of 26 px, so that it aligns with a row that has one.
- **Use `widgets.PickList` for a list that the user edits and does not only
  read.** The list of Fyne selects one row, and to remove six things one at a
  time, with a load between each one, is six times the work for one decision.

  The mark is a checkbox in the row, and not a key held during a click. A
  checkbox says what it does because it is there, and Fyne does not give the
  key to the selection callback of a list.

  `OnPicked` reports the count, so the action button can say what it will do.
  A pick is a row number, so a caller whose list has changed calls
  `ClearPicks`. `PickList` returns its list through `Widget()`, and is not a
  widget that contains one. `logpane.Pane` does the same, for the same reason:
  a wrapper must pass on everything that its container does, and a list that
  something resizes but never lays out builds no rows.

## Forms

- A form model (`forms.Form`) holds its widgets apart from the section. You can
  then test the path from the document to the widgets and back without a
  display, and Revert can return the old values. A rebuild would leave the
  typed values on screen until the load arrived.
- A `setting` guard against re-entry contains the writes that the program makes
  (`Form.Set`), because `Check.SetChecked` and the other setters fire the
  change handlers. The build of a section must never schedule a save.
- The form collects its writes. Each change goes into the document and
  schedules one save, one second after the last change. The first reading of
  the window size is a baseline, so the start of the program never reports
  "Saved".
- A slider and its entry (`forms.SliderEntry`) hold the same value, and only
  one of them commits it: the slider on `OnChangeEnded` and the entry on
  `OnSubmitted`. A settings file written for each pixel is a settings file
  written four hundred times to move one number. Change the card in position,
  so that the focus and the scroll position survive.
- Numeric entries validate a range and sit in a `FixedWidth` of 120.
- Fields that share a group pack onto one row.

## Dialogs

- A confirmation for a destructive action (`dialogs.ConfirmDestructive`) names
  the action on a danger button and keeps Cancel as the safe default. Its
  detail says what the action does *not* change, as well as what it changes. A
  dialog that does not say so becomes a dialog that people accept without
  reading. The correct shape has three parts: where the result goes, what
  happens to what is there now, and what the action leaves alone.
- **Show a dialog that holds a list with `dialogs.Roomy`.** Fyne gives a dialog
  the minimum size of its content. That is correct for a confirmation and wrong
  for anything with a list in it, because the minimum size of a scroller is
  almost nothing. A dialog built around a scroller therefore opens at the size
  of its buttons.

  Two people found this in one week: a file browser that showed four names, and
  a chooser that offered eighteen keys through an opening that showed one and a
  half of them, cut at both ends. `Roomy` takes 85% of the window, with the
  chooser size as a minimum, so it is correct on a laptop and on a large
  screen.
- `Roomy` shows the dialog and *then* resizes it, and anything that sizes a
  dialog must do the same. Before `Show`, a dialog has no window, and `Resize`
  asks it for its minimum size. In Fyne 2.8.1 that reads through the nil window
  and stops the process. That is a failure of the program and not a small
  dialog, and it reached a user. A test taps every chooser button.
- A chooser starts in the directory of the field's current path, then its
  parent, then home.
- Fyne 2.8.1 has no multi-select chooser; an "Add..." button reopens a
  single-file chooser.
- A question from a worker goroutine that waits for an answer
  (`dialogs.Decide`) goes to the UI thread with `fyne.Do`, opens a dialog, and
  waits on a channel. A `sent` guard makes every path that closes the dialog
  answer one time. Give the entry the focus after you show the dialog.
- **Every dialog closes from its corner.** Put an X at the top right of the
  content, as well as the buttons that the dialog carries. The dialog of Fyne
  has no control to close it, and no way to put one in its title bar. Without
  the X, the only way to leave a dialog is to find the correct button among the
  others. That is correct for a confirmation, where the choice is the purpose,
  and wrong for a long answer that the user only reads. The X closes the dialog
  and is never an answer: it does not run a destructive action, confirm a
  prompt, or answer a waiting question with yes.
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
- Where long documentation is already in a README, embed that README and show
  it. Do not write the text again. A shorter copy becomes a second copy that is
  less careful, and that somebody must keep in agreement with the first.

## Copy voice

Use sentence case, the second person, and plain words, and name the
consequence. End a button that opens a dialog with an ellipsis. A destructive
dialog says what it does not change. A state with no content explains itself.

The alternative text of a screenshot describes the picture. It is the only
description that a user of a screen reader receives, and one that is out of
date is worse than none.

See also [How this is written](style.md), which holds the rules for this page
and for the other documents in `docs/`.

## Threading

- Run every core call on a goroutine, and return every callback to the UI
  thread with `fyne.Do`. Never use `fyne.DoAndWait`. Write a `ui` field on the
  UI thread only.
- A worker writes to a stream, which is `logpane.Model` and `Pane`, behind a
  mutex. The pane draws on a timer every 100 ms and never for each line. A
  hundred `fyne.Do` calls each second would make the window slower than the
  job. A flag that marks a change makes a quiet job cost one comparison for
  each tick. To stop a ticker does not close its channel, so the pump has its
  own quit channel and draws one more time after it stops.
- There is no scroll callback, so the pane follows the end of the log in this
  way: compare the offset of the list with the offset that the last automatic
  scroll left, to a tolerance of 4, because `ScrollToBottom` arrives at
  fractional values. Clear the remembered offset when the shell detaches the
  pane.
- When the shell is not on screen, which is the case in a headless test, work
  runs inline. The Fyne test driver runs `fyne.Do` inline on the goroutine that
  called it, and does not put the work on a main loop. A worker that refreshes
  a widget therefore races the test that drives it.

## Testing

- Run the tests without a display, with `fyne.io/fyne/v2/test`. `make test`
  must pass on a machine that has no display server.
- Test the behaviour and not the layout: what a cell says, when a button is
  enabled, and what the log keeps. A test that asserts that a section holds a
  `Border` that holds a `VBox` tests the code against itself, and it fails
  every time somebody moves something.
- Write a test name as a sentence that names the defect the test prevents.
- Assert the content of a cell through a pure function that you extracted,
  because a table builds its cells only when it needs them, and there is
  nothing in the object tree until the table is on a canvas.
- Put `HOME` and the `XDG_*` directories in `t.TempDir()` for each test. A
  result that depends on the machine of the developer is not a test.
- Every Fyne quirk that this module corrects has a canary test named after the
  quirk. When a canary fails, Fyne has changed and you can remove the
  correction.
- Take screenshots with `tools/screenshot.sh`. It drives a program through
  `--section` and `--scheme` under a temporary HOME, confirms that the window
  has the focus, and checks the aspect ratio of the image before it accepts
  the result. The alternative text describes what is in the image.
- Under the Fyne test driver, `fyne.Do` runs inline on the goroutine that
  called it. A test must not examine widgets while a worker goroutine draws
  them; a log pump and a question that waits are two examples. Stop the worker
  and wait for it first, or drive the UI-thread part of the code
  directly.

## Platform

- The module supports Linux, macOS and Windows. Put platform code behind a
  build tag.
- On Linux, before the toolkit starts, read the cursor theme and size from
  `kcminputrc`, then from the GTK 4, GTK 3 and `.gtkrc-2.0` settings. Then set
  `XCURSOR_THEME` and `XCURSOR_SIZE`, unless they are set already. The Wayland
  backend of GLFW does not have `cursor-shape-v1`, and Plasma gives these
  values to XWayland clients only. Read the files directly and do not start a
  subprocess.
- Wayland matches a window to its desktop entry by `app_id`, which Fyne takes
  from the app ID. The base name of the desktop file must be the same as the
  app ID, or the window loses its icon in the taskbar.
- The font directories differ by platform. On Linux they are the XDG
  directories and `/usr/share`. On macOS they are `/System/Library/Fonts` and
  `/Library/Fonts`. On Windows they are `C:\Windows\Fonts` and the fonts
  folder of the user.
- The interface needs CGO. To build it for a different platform, you need a C
  toolchain for that platform. A command-line program stays at
  `CGO_ENABLED=0`.

## Numbers

| Item | Value |
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
