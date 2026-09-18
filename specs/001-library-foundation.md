# Spec 001: Library foundation

> **Note**: This work has no associated issue tracker ticket. The repository
> is a personal public project without an issue tracker.

## Status: COMPLETE

## Executive Summary

Creates the module and extracts the application-free leaves of the Fyne
design system shared by angou, nmsbonker and clockwork-orange: the `theme`
package (nine schemes including new Windows and macOS pairs, the
`fyne.Theme` adapter, font discovery, appearance preferences, scale, cursor
fix), the `widgets` primitives, the `table` detail table, and the `fynetest`
helpers, with a gallery binary and three-OS CI. Reviewers should look first
at `theme/palette.go` and `theme/theme.go` (the API every consumer touches),
then `docs/design-system.md` (the rules the code is held to).

## Context

angou (2026-08-31), nmsbonker (2026-09-11) and clockwork-orange v4
(2026-09-15) each carry a Fyne GUI built on the same design system. angou is
the origin; the other two copied `theme.go`, `fonts.go`, `cursor_*.go`,
`icon.go`, the shell in `app.go`, the dialogs and the small widgets with a
header `// Copied from angou (same author) — keep in sync by hand.`
clockwork-orange then added `markdownpane.go` (its spec 011), `logpane.go`
and a thumbnail column on the detail table; nmsbonker added the build-log
model, step list and cancellable run. Roughly 3,300 lines exist in three
copies and have already diverged: nmsbonker's log pane is 420 high and
clockwork-orange's 360; angou still has a fixed flash slot while the other
two moved banners into popups; clockwork-orange's theme carries a separate
mono face that the others lack.

nmsbonker's spec 003 rejected a shared module in 2026-09 ("two consumers,
same author; hand-copying with a keep-in-sync note is less ceremony"). With a
third consumer and a fourth on the way, that calculus has flipped.

This spec creates the module and extracts the leaves: the parts with no
application coupling. It also sets up the gallery, the test kit, CI, and the
documentation that later specs build on. The shell, documents, diagrams and
forms follow in their own specs (see Roadmap) because that is where the three
apps disagree and where the API needs a human decision.

Surveys of the three codebases that informed this spec are summarised in
`docs/design-system.md` and `docs/fyne-quirks.md`.

## Requirements

### R1. Module and tooling

- R1.1 Module `github.com/ushineko/fynedesygn`, `go 1.26.0`, no `toolchain`
  line. `fyne.io/fyne/v2 v2.8.1` and `github.com/stretchr/testify` are the
  only direct dependencies of this spec.
- R1.2 Makefile targets `setup`, `lint`, `test`, `coverage`, `vuln`,
  `generate`, `gallery`, `clean`. Linter pinned to golangci-lint v2.12.2 with
  `GOTOOLCHAIN=go1.26.0`, config `config/.golangci-v2.12.2.yml`.
- R1.3 `make test` passes with no display server, with the race detector.
- R1.4 GitHub Actions workflow `build.yml`: a `test` job on `ubuntu-latest`,
  `windows-latest` and `macos-latest` that runs `make test`; `make lint` and
  `make gallery` on Ubuntu; `govulncheck` on Ubuntu. No release job yet.
- R1.5 MIT licence, `Copyright (c) 2026 ushineko`.

### R2. Root package

- R2.1 `fynedesygn.Status` with `StatusInfo`, `StatusGood`, `StatusWarn`,
  `StatusBad` and `String()`.

### R3. `theme` package

Ported from angou's `theme.go` and `fonts.go` and clockwork-orange's mono-face
extension. Rationale comments travel with the code.

- R3.1 `Palette` with KDE `.colors` field names plus `Radius` and `Padding`
  tokens, exported; nine schemes as values: Breeze Dark, Breeze Light,
  Oxygen Dark, Adwaita Dark, Adwaita Light (as in the apps), and new Windows
  Dark, Windows Light (Windows 11 Fluent tokens, 4 px corners), macOS Dark,
  macOS Light (Apple system colours, 8 px corners for the macOS 26 Tahoe
  Liquid Glass style; Fyne cannot draw translucency, so roundness and colour
  carry it). `Schemes() []Palette`, `SchemeNames() []string`,
  `SchemeByName(string) Palette` falling back to `DefaultScheme()`, which is
  the platform's native dark scheme (build-tagged: Breeze Dark on Linux,
  Windows Dark on Windows, macOS Dark on macOS).
- R3.2 `Theme` implementing `fyne.Theme`, constructed with
  `New(Palette, Options)` where options carry an interface font, a monospace
  font and a text size. Icons from the default theme; monospace never taken
  from the interface family; the `ThemeVariant` argument ignored; corner
  radius and padding from the palette, text sizes as in
  `docs/design-system.md`.
- R3.3 `DefaultTextSize = 12`, `TextSizes()`.
- R3.4 Font discovery: `FontNames()`, `LoadFont(name) *Font`, `DefaultFontName`,
  `RescanFonts(dirs...)` for tests and for programs that install a font.
  Font directories per platform via build-tagged files: Linux and other Unix,
  macOS, Windows (`%WINDIR%\Fonts`, `%LOCALAPPDATA%\Microsoft\Windows\Fonts`).
  Variable fonts skipped; families without a regular face dropped; unreadable
  fonts fall back to nil.
- R3.5 Appearance preferences: `Appearance{Scheme, Font, Mono, TextSize,
  Scale}`, `LoadAppearance(fyne.Preferences) Appearance`,
  `(Appearance) Save(fyne.Preferences)`, `(Appearance) Theme() fyne.Theme`,
  keys `appearance.scheme`, `appearance.font`, `appearance.mono`,
  `appearance.textSize`, `appearance.scale`. Stale values fall back silently.
- R3.6 `ApplyScale(scale float32) bool` writes `FYNE_SCALE` unless the
  environment already sets it; `ScaleChoices()`, `ScaleLabel`, `ScaleValue`.
- R3.7 `ApplyCursorTheme()` on Linux reads `kcminputrc`, GTK 4, GTK 3 and
  `.gtkrc-2.0` and exports `XCURSOR_THEME` and `XCURSOR_SIZE` unless set;
  no-op elsewhere. No subprocesses.
- R3.8 A `Sample(monoLine)` widget (the Appearance section's live sample:
  regular, bold, a caller-supplied monospace line, and the three status
  colours) so programs and the gallery show the same preview.

### R4. `widgets` package

Ported from the small shared widgets in angou's and nmsbonker's `app.go` and
`dialogs.go`. Every function takes plain values or a `Status`; none takes a
shell.

- R4.1 `Dim`, `Sep`, `Wrapped`, `StatusText`, `Marker`, `Heading`, `Card`,
  `Note`, `FactRow`, `PlainRow`, `RowWithAction`, `Action` (returns the
  container and the button), `AboutNote`.
- R4.2 `FixedHeight`, `FixedWidth`.
- R4.3 `HumanSize` (GiB/MiB/KiB/B, one decimal), `HumanAgo`, `OrNone`.
- R4.4 `ImportanceFor(Status) widget.Importance`, `StatusColor(Status)
  color.Color` reading the active theme.
- R4.5 Label column width and marker gutter are exported constants
  (`LabelWidth = 190`, `MarkerGutter = 26`).

### R5. `table` package

Ported from clockwork-orange's `views_table.go` (nmsbonker's plus the
optional thumbnail column).

- R5.1 `Detail` with `Header`, `Row(Status, cols...)`, `SetWidths`,
  `Measure`, `Widget() *widget.Table`, optional thumbnail column with a decode
  cache. Importance set before text; corner-cell guard.
- R5.2 `SortHeader` helper: a low-importance leading button with the arrow in
  its text, and a documented contract that re-sorting clears selection.

### R6. `fynetest` package

- R6.1 `Walk(obj, visit)` descending containers, scrollers and tabs.
- R6.2 `FindButton(obj, text)`, `FindCheck`, `FindEntry`, `FindSlider`.
- R6.3 `Text(obj) string` joining every label, button and check text.
- R6.4 `ScrollableIn(obj) bool`.
- R6.5 `Sandbox(t)` setting `HOME`, `XDG_CONFIG_HOME`, `XDG_DATA_HOME`,
  `XDG_CACHE_HOME` to `t.TempDir()`.

### R7. Gallery

- R7.1 `cmd/fynedesygn-gallery` with flags `--section`, `--scheme`,
  `--version`. Until spec 002 lands it uses a minimal local shell; it is
  rewritten onto `shell` in spec 002.
- R7.2 Sections in this spec: Appearance (the real picker plus `Sample`),
  Widgets (every `widgets` function with its name as caption), Table (a
  detail table with all four statuses and a thumbnail column), Fonts (the
  discovered families).
- R7.3 Builds with `make gallery` on Linux, macOS and Windows.

### R8. Documentation

- R8.1 `docs/design-system.md`, `docs/fyne-quirks.md`, `docs/markdown.md`,
  `docs/mermaid.md` exist (written with this spec) and are linked from the
  README.
- R8.2 Every exported identifier has a doc comment. `go vet` and the linter
  are clean.
- R8.3 Ported rationale comments keep their content; the "Copied from"
  headers are replaced by a package comment naming the origin once.

### R9. Canary tests

Canaries for quirks 3, 4, 6 and 8 from `docs/fyne-quirks.md` land with the
packages in this spec (6 lands in `fynetest`, whose `ScrollableIn` exists for
it). Quirk 12 has no canary by design. Quirk 1 belongs to `dialogs` and lands
with spec 004, as do 2, 5, 7, 10, 11, 13 and 21 with specs 002 to 004.

## Acceptance Criteria

- [x] AC1 `go build ./...`, `make test`, `make lint` and `go vet ./...` pass
  on Linux with no display, with only `fyne.io/fyne/v2` and `testify` as direct
  dependencies (R1.1 to R1.3).
- [x] AC2 The CI workflow runs `make test` on Ubuntu, Windows and macOS and
  `make gallery` on Ubuntu, and is green on the first push (R1.4).
  _First push: all three test jobs green; the scan job failed on the Go
  1.26.0 standard library itself, fixed by scanning with the latest stable
  Go. Second run (spec 002 commit) fully green._
- [x] AC3 `theme.SchemeNames()` returns the nine names in the documented
  order, and `theme.SchemeByName("nonsense")` returns `DefaultScheme()`,
  which is Breeze Dark on Linux (R3.1).
- [x] AC4 A test constructs `theme.New` for every scheme and asserts every
  `fyne.ThemeColorName` the theme maps returns a non-nil colour, and that
  `Font(Monospace)` returns the default theme's face regardless of the
  interface font (R3.2).
- [x] AC5 A test with a temporary font directory containing a regular, bold
  and variable `.ttf` shows the variable face skipped and bold resolved; an
  unreadable file yields the default (R3.4). Font directory lists exist for
  linux, darwin and windows build tags and compile on all three in CI.
- [x] AC6 `LoadAppearance` on empty preferences returns the defaults;
  `Save` then `LoadAppearance` round-trips; a stale scheme name loads as the
  default (R3.5).
- [x] AC7 `ApplyScale` leaves an existing `FYNE_SCALE` untouched and sets it
  otherwise (R3.6).
- [x] AC8 `ApplyCursorTheme` reads a temporary `kcminputrc` and sets both
  variables; does not override a preset `XCURSOR_THEME`; the `!linux` build
  compiles a no-op (R3.7).
- [x] AC9 Every `widgets` function renders headlessly without panic and
  `fynetest.Text` of the result contains the given text (R4.1 to R4.3).
- [x] AC10 A `table.Detail` with rows of all four statuses renders headlessly;
  a test through an extracted cell function shows the cell importance for
  each row; the corner-cell guard is exercised (R5.1, quirks 3 and 4).
- [x] AC11 The canary tests for quirks 3, 4, 6 and 8 exist, are named after
  the quirk, and pass against Fyne v2.8.1 (R9):
  `table.TestImportanceAfterSetTextKeepsOldColour`,
  `table.TestUpdateHeaderVisitsCornerCell`,
  `fynetest.TestFyneStillDrawsMarkdownCodeInsideAScroll`,
  gallery `TestSectionNamesNeedNoApp`.
- [x] AC12 `fynedesygn-gallery --section Widgets --scheme "Adwaita Light"`
  opens on that section in that scheme without persisting the scheme;
  `--version` prints and exits without a Fyne app being created (R7).
- [x] AC13 The README lists the packages and links the four documents; every
  exported identifier has a doc comment (`revive` exported rule on) (R8).
- [x] AC14 A validation report for this spec is committed (milestone commit):
  `validation-reports/2026-09-17-spec001-foundation.md`.

## Risks & Assumptions

- **Assumption**: the Windows and macOS palettes are transcribed from
  published design tokens (WinUI 3 theme resources, Apple HIG system colours)
  and composited by hand; they have not been checked pixel-for-pixel against
  a live Windows 11 or macOS 26 desktop. The gallery screenshots in spec 005
  are the check.
- **Assumption**: Fyne v2.8.1 for all consumers. clockwork-orange, nmsbonker
  and angou are all on it today.
- **Assumption**: the gallery is the only binary in this spec; no packaging,
  no release job. Rollback for anything in this spec is `git revert`; nothing
  is deployed.
- **Risk**: the three apps' constants differ in places (log pane height,
  flash slot vs popup). This spec extracts only leaves where all three agree;
  disagreements are left to specs 002 to 004 and recorded in
  `docs/design-system.md`.
- **Risk**: CGO on Windows and macOS runners. The Fyne test driver does not
  need a display, but `go test` still compiles GLFW through CGO; the runners
  have a C toolchain. If a runner fails to link, the fallback is to run
  `make test` on Ubuntu only and build the gallery on the others, and to
  record the gap.
- **Risk**: golangci-lint v2.12.2 under Go 1.27 on the runners; the Makefile
  pins `GOTOOLCHAIN=go1.26.0` for the lint target, which downloads that
  toolchain in CI.
- **Query budget / DB / migrations**: not applicable, no persistence beyond
  `fyne.Preferences`.
- **Shared-code safety**: nothing consumes the module yet; the adopting
  programs are unaffected until their own adoption specs.

## Alternatives Considered

- Considered one `components` package instead of `widgets`, `table`,
  `logpane`, `markdown`, `dialogs`; rejected because the heavier components
  (document pane, log pane) carry state and lifecycle that the primitives do
  not, and a consumer that wants only the theme should not compile the rest.
  Kept small enough that a stutter-free name exists for each.
- Considered keeping `Status` in `widgets`; rejected because `table`,
  `logpane` and `shell` all need it and a root-package vocabulary avoids an
  import cycle.
- Considered embedding a font so all three platforms look identical; rejected
  because the apps deliberately discover system fonts and the module would
  inherit a font licence.
- Considered `fyne.io/fyne/v2/data/binding` for state; rejected, matching
  nmsbonker spec 003: the apps rebuild views from plain state and consistency
  wins.

## Roadmap (later specs, not part of this one)

- **002 Shell**: `shell.Shell`, `shell.Section` interface (`Title`, `Icon`,
  `Build`, optional `Detach`), header actions, status-bar segments, busy
  popup, banners, `Perform` with context and inline headless path, the redraw
  ladder, `Run(Options)`. Gallery rewritten onto it. The API decisions here
  are the ones that need the human: how a section reaches program state, and
  whether a queue of banners is ever wanted.
- **003 Documents and diagrams**: `markdown.Pane`, `markdown.CodePanel`,
  `mermaid` package and `cmd/fynedesygn-mermaid`, `fs.FS` for images, the
  canaries for quirks 6, 7 and 21. An About section pattern in the gallery
  showing this module's own README with a diagram.
- **004 Dialogs, log pane, forms**: `dialogs`, `logpane` (model, pane, pump,
  follow-tail), form model (`settings` form, slider-entry pair, numeric
  entry), canaries for quirks 2, 5, 10, 11, 13.
- **005 Examples and screenshots**: `examples/` programs per pattern
  (master-detail with a table, settings form with auto-save, job runner with
  log and steps, document viewer), a screenshot harness ported from angou's
  `tools/screenshot.sh`, README screenshots.
- **006 Adopt in clockwork-orange**, then nmsbonker (007), then angou (008).
  Each deletes its copy and the module gains whatever the adopter needed.
  v0 stays unstable until the third adopter; `v1.0.0` after angou.
