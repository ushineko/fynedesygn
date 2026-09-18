# fynedesygn

**Version**: 0.1.3

A design system and wrapper library for building desktop user interfaces with
[Fyne](https://fyne.io) in Go. It is the maintained home of the Fyne design
system that [angou](https://github.com/ushineko/angou),
[nmsbonker](https://github.com/ushineko/nmsbonker) and
[clockwork-orange](https://github.com/ushineko/clockwork-orange) carried as
hand-synced copies.

## Contents

- [What is in it](#what-is-in-it)
- [Screenshots](#screenshots)
- [Examples](#examples)
- [Status](#status)
- [Using it](#using-it)
- [Documentation](#documentation)
- [Development](#development)
- [Licence](#licence)
- [Changelog](#changelog)

## What is in it

| Package | Purpose |
|---|---|
| `fynedesygn` | Shared vocabulary: `Status`; the embedded README. |
| `theme` | Nine colour schemes (Breeze Dark and Light, Oxygen Dark, Adwaita Dark and Light, Windows Dark and Light, macOS Dark and Light), a `fyne.Theme` that applies them, system font discovery, appearance preferences, interface scale, the Linux cursor-theme fix. |
| `widgets` | The small shared primitives: headings, cards, fact rows, status text and markers, fixed-size wrappers, human-readable sizes and ages. |
| `table` | A read-only detail table with one status per row and measured column widths. |
| `logpane` | A mutex-guarded log model and a fixed-height monospace pane drawn on a timer with follow-tail, Copy and Clear. |
| `forms` | A form whose widgets are held apart from the section (Set never fires the change hook, so Revert reverts), the slider-and-entry pair that commits once per gesture, validated numeric entries. |
| `markdown` | A Markdown document pane that renders per block near the viewport, draws code blocks and pipe tables itself, resolves images from an `fs.FS`, and shows pre-rendered mermaid diagrams; `Section` for a document page. |
| `mermaid` | Hash-keyed lookup of pre-rendered diagram PNGs (light and dark), the widget that draws them, and the `mmdc` renderer and checker behind `go generate`. |
| `dialogs` | Destructive confirmation, prompt and detail dialogs, file and folder choosers that do not crash, the desktop opener, and a yes-or-no question a worker goroutine can ask. |
| `shell` | The window skeleton: header, section nav, content pane, status bar, busy indicator, banners, `Perform`, section lifecycle, plus the standard Appearance and About sections. |
| `steps` | The step list a job shows beside its log, updated in place. |
| `fynetest` | Headless test helpers: tree walking, finders, text extraction, scrollable detection. |
| `cmd/fynedesygn-gallery` | The reference program. Every component in every scheme, with `--section` and `--scheme` for screenshots. |
| `cmd/fynedesygn-mermaid` | The `go generate` helper that renders missing diagrams; `-check` in CI fails on stale ones. |
| `docs` | The documents in `docs/` and their diagrams, embedded for the gallery. |
| `examples/` | Small programs, one per UI pattern; see [Examples](#examples). |
| `tools/` | The screenshot harness for KDE/Wayland. |

The rules the components follow are in [docs/design-system.md](docs/design-system.md).

## Screenshots

Captured from the gallery in Breeze Dark by `make screenshots`; refresh them
with the same command and check the alt text still matches.

![The Shell section of the gallery: a card with three job buttons, a card with four banner buttons, and a grid of six colour swatches for the active scheme's roles, with the section list on the left and a status bar showing the job state at the bottom.](docs/img/gallery-shell.png)

![The Widgets section: the shared vocabulary drawn one primitive at a time, each captioned with its Go name, including a card of fact rows with status markers and a danger action row.](docs/img/gallery-widgets.png)

![The Table section: a detail table with rows coloured by status, a second table with a 64 px thumbnail column, and three sort header buttons.](docs/img/gallery-table.png)

![The Documents section: docs/markdown.md rendered in the window, with a flowchart of the rendering pipeline drawn from a pre-rendered PNG and a code panel that wraps.](docs/img/gallery-documents.png)

![The Forms section: a form with Save and Revert, a slider with its entry beside it, and a validated numeric field.](docs/img/gallery-forms.png)

![The Appearance section: pickers for the colour scheme, fonts, text size and interface scale over a live sample of regular, bold, monospace and status-coloured text.](docs/img/gallery-appearance.png)

## Examples

Each example is a complete program on `shell.Run`, with a headless test, in
`examples/`. Build them all with `make build-examples`.

| Example | Pattern |
|---|---|
| `master-detail` | A table with one status per row, a detail card for the selected row, Refresh through `Perform`, a count in the status bar, and the loaded-flag pattern. |
| `settings` | A `forms.Form` over a JSON file, saved a second after the last change through `forms.Saver`, with Revert and the standard Appearance section. |
| `job-runner` | A `steps.List` beside a `logpane.Pane`, a job run through `PerformCancellable` that advances steps and logs, and one banner for the result. |
| `document-viewer` | `markdown.Section` over an embedded guide with a mermaid diagram rendered by `go generate`. |

## Status

Pre-release. The module is `v0.x` and the API changes as the three consuming
programs adopt it. Nothing here is stable until `v1.0.0`.

## Using it

```
go get github.com/ushineko/fynedesygn@latest
```

The library pins `fyne.io/fyne/v2 v2.8.1`. Building a Fyne program needs CGO
and, on Linux, the OpenGL and X11 or Wayland development headers. Running this
module's tests does not: they use the Fyne test driver.

## Documentation

- [Design system](docs/design-system.md): the layout rules and policies.
- [Rendering Markdown](docs/markdown.md).
- [Mermaid diagrams](docs/mermaid.md).
- [Fyne quirks](docs/fyne-quirks.md): what the module works around and the
  canary test for each.

## Development

```
make setup     # install the pinned linter
make test      # headless, race detector
make lint
make gallery   # build the reference program for this machine
make generate  # render missing mermaid diagrams (needs mmdc)
make check-diagrams
```

Work is specified in `specs/` and follows the conventions in
`.claude/CLAUDE.md`.

## Licence

MIT. See [LICENSE](LICENSE).

## Changelog

### 0.1.18 (2026-09-18)

A wrapped log line's continuation rows are indented and marked, so where a
message starts is visible without reading it.

### 0.1.17 (2026-09-18)

Fix: the wrapping added in 0.1.16 never ran. It hung off `Draw`, which runs on
the pump's tick, and most panes are never pumped — a section that rebuilds on
its own redraws the log with it. The test that passed called `Draw` itself,
which is a path the caller does not take. The wrapping is in the list's length
function now, which is reached however the pane is driven, and the test builds
the widget and refreshes it the way a section does.

### 0.1.16 (2026-09-18)

`logpane` wraps. A line longer than the pane was drawn past the right edge and
the rest of it could not be read at all. Every row is still one line of
monospace text in a `widget.List` -- that is what makes a long log scroll like a
terminal -- so the wrapping is done in the pane's own terms: a long line becomes
several rows of the same height. Copy still renders the model, so a line broken
for the screen arrives whole on the clipboard.

### 0.1.15 (2026-09-18)

Fix: a result banner was unreadable over a busy section. The status tint is
translucent so text stays legible over it, which was fine while a banner was a
popup drawing an opaque background of its own; moved into a layer of the content
in 0.1.13 it had nothing underneath but the section, and a log behind it read
straight through the message. Banners now bring their own background and a
hairline edge, and every layer fades together.

### 0.1.14 (2026-09-18)

`widgets.TipDelay` is a second, up from half of one. Half a second is about how
long it takes to move the pointer onto a button and press it, so a tip arrived
at the moment of the click on controls nobody was asking about.

### 0.1.13 (2026-09-18)

Fix: a result banner took every click in the window while it was up. Same cause
as the tip in 0.1.12 — a banner was a popup, and a popup is an overlay — but for
six to twelve seconds after every operation, and the first click dismissed the
banner instead of doing what it was aimed at. Banners are drawn in a layer of
the content now; their own dismiss still works. The busy popup stays an overlay,
because blocking input is what it is for.

Fix: a tip stayed up when its control opened a menu or a dialog, because the
pointer never left it. `widgets.HideTips` takes down every tip, shown or
waiting, and the shell's controls and every dialog call it before they open.
Quirk 27.

### 0.1.12 (2026-09-18)

Fix: a hover tip took every click in the window while it was showing. A tip was
a popup, a popup is an overlay, and Fyne routes pointer events to the top
overlay instead of to the content — so a control whose own tip was up needed two
clicks, and the first only took the tip down. Tips are now drawn in a layer at
the top of the window's content (`widgets.NewTipLayer`, which the shell provides
for every window it builds), where the walk that finds what was clicked skips
them. Quirk 26.

### 0.1.11 (2026-09-18)

Fix: the navigation's shape menu opened in the corner of the window rather than
under the button that raises it. A widget's `Position` is measured from its
parent, so the button in the header's row reported a position near the origin of
that row.

### 0.1.10 (2026-09-18)

`settings.Store` and `Shell.Settings()`: a program's settings in one file the
user can read, one section per key, decoded into the caller's own type
(spec 011). The extension chooses the format — JSON in the core,
`settings/yamlcodec` for `.yaml` and `.yml`. Sections a build does not know are
kept rather than dropped. The appearance moves into it, read once from
`fyne.Preferences` for an installation that predates the file.

`Shell.Stop` runs `Options.OnStop` and writes pending settings when the window
closes, not only before a restart, and runs once however many times it is
called.

The navigation's shape (spec 013): icons and labels, icons alone or hidden, down
the left or along the top, with one stock control in the header and `Ctrl+B` to
hide. A program declares which shapes it allows through `Options.NavModes` and
`NavPlacements`; one that declares nothing keeps exactly the window it has, with
no control and no shortcut.

`Shell.VSplit` and `HSplit`: a divider the user can drag whose position outlives
the section that drew it and the run that set it (spec 012). Fyne's split
reports no drag and a section is rebuilt several times a minute, so the shell
reads each live divider before the content holding it is replaced. The
navigation's own divider goes through the same call, so the width of the section
list is now something the user can keep. `logpane.Options.Height` is documented
as a minimum rather than a size.

### 0.1.9 (2026-09-18)

Fix: `widgets.FixedWidth` and `FixedHeight` padded with a nil-coloured
rectangle, which is invisible under the GL painter and a nil dereference under
the software one. Every window built with them worked and could not be rendered
to an image, which is what a headless layout check needs. Quirk 25.

### 0.1.8 (2026-09-18)

Fix: the hover tip's catcher had the same nil fill, with the same effect.

### 0.1.7 (2026-09-18)

Fix: a long hover tip was drawn one line tall with its text outside the box. A
wrapping label reports a minimum width of about one character and does not know
its height until it has a width, so a tip is measured after being resized.

### 0.1.6 (2026-09-18)

`widgets.WithTip` (spec 010): a hover note for the guidance a control cannot
fit in its label. Fyne has no tooltip; this stacks a transparent catcher over
the control, which takes the hover while taps walk past it to the control
underneath. Asked for by terrariabonker's port, whose Qt panel keeps twenty-odd
of these. Additive.

### 0.1.5 (2026-09-17)

`shell.Arriver` (spec 009): the shell now tells a section whether it is being
built because the navigation arrived at it or because it is being rebuilt
where it stands. A section that refetches on arrival needs the difference —
refetching in the builder loops, because the fetch finishing rebuilds the
section that started it. Found in angou's adoption, which had worked around
it by remembering the last section title. Additive; `Section` is unchanged
and `Arriver` is optional.

### 0.1.4 (2026-09-17)

`shell.BusyCancellable` (spec 008): a job that holds the busy indicator itself
can put its Cancel on the popup. The popup is modal, so a Cancel left enabled
in the toolbar behind it could not be clicked — found in nmsbonker's build
section, where it had been unreachable since before the adoption. Additive;
`PerformCancellable` now runs through the same path.

### 0.1.3 (2026-09-18)

Second adopter feedback (spec 007): `steps.Advance` never reopens a finished
step, `steps.NewSteps` with standing notes that `Reset` keeps,
`logpane.Pane.SetFollowing`, `shell.Load`, `shell.About.URLText`. Behaviour
change: `forms.SliderEntry` passes a typed value to `Commit` as typed (the
controls still show it clamped) so a program can clamp and report itself.

### 0.1.2 (2026-09-18)

Fix: the Appearance section's notes were unwrapped labels, so the section's
minimum width was the whole line and the window scrolled sideways (seen on
Windows in clockwork-orange 4.1.0). New `widgets.DimWrapped` for secondary
paragraphs; every long note uses it; a test holds the section to a 500 px
viewport.

### 0.1.1 (2026-09-18)

Hooks the first adopter needed: `shell.Options.OnCreate`, `Theme` and
`OnTypedKey` (spec 006). Additive.

### 0.1.0 (2026-09-18)

First tagged release, for the first adopter (clockwork-orange). Packages
`theme` (nine schemes), `shell`, `widgets`, `table`, `steps`, `markdown`,
`mermaid`, `docs`, `dialogs`, `logpane`, `forms`, `fynetest`; the gallery,
the mermaid command, four examples and the screenshot harness. The API is
`v0` and changes as the three programs adopt it.
