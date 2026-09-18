# fynedesygn

**Version**: 0.0.0 (not yet released)

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

Nothing released yet.
