# Spec 004: Dialogs, log pane, forms

> **Note**: This work has no associated issue tracker ticket. The repository
> is a personal public project without an issue tracker.

## Status: COMPLETE

## Executive Summary

Adds `dialogs` (destructive confirmation, confirm with a body, prompt,
detail, choosers resized after show, desktop opener, and `Decide` for a
worker goroutine), `logpane` (the mutex-guarded model, follow-tail, and the
fixed-height pane with Copy and Clear, drawn by a pump) and `forms` (a form
with a re-entry guard so Revert reverts, validated numeric entries, and the
slider-and-entry pair that commits once per gesture), plus gallery sections
for each. Reviewers should look first at `forms/forms.go` (`Field` and
`Form.Set`) and `dialogs/decide.go`.

## Context

Three more shapes are shared by the apps and coupled to their `ui` structs
today. The dialogs (angou's `confirmDestructive`, `pathDialog`, the file and
folder choosers with the resize-after-show workaround, nmsbonker's
`confirmWithBody`, `showDetail`, `openPath`) need only a `fyne.Window`. The
log pane (nmsbonker's `buildlog.go` generalised by clockwork-orange into
`logpane.go`: a mutex-guarded model, a fixed-height list of `canvas.Text`
rows drawn on a 100 ms timer, follow-tail inference) needs a clipboard and a
banner callback. The form shapes (nmsbonker's `settingsForm` with widgets
held apart from the section, `editField` rows, the `paramRow` slider and
entry pair; clockwork-orange's `numericEntry` and `intRange`) share one
rule, the `setting` re-entry guard, and one purpose, a Revert that reverts.

Also ported here from angou: the blocking question from a worker goroutine
(`fyne.Do` to put a dialog up, wait on a channel, answer exactly once).

## Requirements

### R1. `dialogs` package

All functions take the `fyne.Window` they belong to; none takes a shell.

- R1.1 `ConfirmDestructive(win, title, detail, confirm string, do func())`:
  560 x 320, wrapped detail, Cancel as the safe default, danger button. The
  doc comment carries the rule that the detail says what is not touched.
- R1.2 `ConfirmWithBody(win, title, body, confirm, do) *dialog.CustomDialog`
  (600 x 360, caller shows it so it can read the body's widgets in `do`).
- R1.3 `Prompt(win, title, confirm string, body, do)`: fields plus a confirm
  (angou's `pathDialog`), 620 x 340.
- R1.4 `ShowDetail(win, title, body, w, h)`: read-only with a Close button.
- R1.5 `PickerStart(text string) fyne.ListableURI`: the path if it is a
  directory, else its parent, else home, expanding a leading `~`.
- R1.6 `BrowseButton(win, field, dir) *widget.Button`, `WithBrowse(win,
  field, dir)`, `ChooseFile(win, start, filter, then)`, `ChooseFolder(win,
  start, then)`: choosers 760 x 520, resized after `Show` (quirk 1). The
  file chooser closes the handle it is given and passes the path.
- R1.7 `OpenPath(path string) error` hands a path to the desktop opener
  (`xdg-open`, `open`, `explorer`) as a single argument, detached from any
  context; `Opener() string` names it. The caller reports the error.
- R1.8 `Decide(win, title, detail, yes, no string) bool`: for a worker
  goroutine; hops to the UI thread, shows the question, waits on a channel;
  every close path answers exactly once. Documented as never to be called
  on the UI thread.

### R2. `logpane` package

- R2.1 `Level` (`Debug`, `Info`, `Warn`, `Error`) with `String()` and
  `Status() fd.Status`; `Line{Level, Text}`.
- R2.2 `Model`: `NewModel(maxLines int)` (0 means `DefaultMaxLines` = 1000),
  `Append(level, text)` one row per line dropping `DropChunk` = 128 at the
  cap, `Replace(text)`, `Reset()`, `Len()`, `Dropped()`, `At(i)`,
  `TakeDirty()`, `Text()` with the dropped-lines preamble. Safe from any
  goroutine.
- R2.3 `FollowTail(following bool, current, wanted float32) bool` with
  tolerance 4.
- R2.4 `Pane`: `New(model *Model)` (nil makes one), `Widget(o Options)
  fyne.CanvasObject` with `Options{Title string; Height float32 (0 =
  DefaultHeight 360); TextSize float32 (0 = theme); OnClear func();
  Clipboard fyne.Clipboard; Flash func(string, fd.Status)}`; rows are
  `canvas.Text` coloured by level; header with counter, Follow check, Copy,
  Clear when `OnClear` is set; `Draw()`, `Touch()`, `Pump() (stop func())`,
  `Detach()`, `Log(level, msg)` (timestamped `15:04:05 LEVEL msg`),
  `Writer(level) io.Writer` splitting on newlines.
- R2.5 `Pane` implements `shell.Detacher` by having `Detach()`; it does not
  import `shell`.

### R3. `forms` package

- R3.1 `Field{Key, Label string; Widget fyne.CanvasObject; Get func()
  string; Set func(string)}` with constructors `Entry(key, label,
  placeholder)`, `Select(key, label, options)`, `Check(key, label)` (values
  `true`/`false`), `Path(win, key, label string, dir bool)` (entry with
  Browse), `Numeric(key, label string, lo, hi int)` (validated, 120 wide),
  `Custom(key, label, widget, get, set, bind)`.
- R3.2 `Form`: `New(fields ...*Field)`, `Set(map[string]string)` under a
  re-entry guard so `OnChange` does not fire, `Values()`, `Field(key)`,
  `Widget()` (a `widget.Form`), `OnChange func(key, value string)`.
- R3.3 `IntRange(lo, hi) fyne.StringValidator`, `NumericEntry(lo, hi,
  onChange func(int)) *widget.Entry`.
- R3.4 `SliderEntry`: `NewSliderEntry(SliderOptions{Min, Max, Step, Value
  float64; Format func(float64) string; Commit func(float64); EntryWidth
  float32})`, `Widget()`, `Set(v)`; the slider updates the entry while
  dragging and commits on release, the entry commits on Enter, a non-number
  restores the current value and reports through `OnInvalid`.
- R3.5 `Group(title, blurb string, rows ...fyne.CanvasObject)`: a titled
  block with a dim blurb over its rows.

### R4. Gallery and canaries

- R4.1 Gallery sections Dialogs (every dialog behind a button, a path field
  with Browse, Open), Log (a pane fed by a fake job at 20 lines a second
  with Follow, Copy, Clear) and Forms (a form with Save and Revert, a
  slider-entry pair, a numeric entry with validation).
- R4.2 Canaries: quirk 1 `TestChooserResizeAfterShowDoesNotPanic`, quirk 2
  `TestSetCheckedFiresOnChanged`, quirk 10
  `TestFollowTailToleratesFractionalOffsets`.
- R4.3 README package table and `docs/design-system.md` ("Dialogs",
  "Forms", "Threading") gain the API names.

## Acceptance Criteria

- [x] AC1 Every dialog constructor shows on a test window without panic; the
  destructive dialog's proceed button has danger importance and `do` runs
  only on proceed (R1.1 to R1.4).
- [x] AC2 `PickerStart` resolves a directory, a file's parent, `~`, and
  falls back to home for nonsense (R1.5).
- [x] AC3 Canary `TestChooserResizeAfterShowDoesNotPanic`: tapping Browse
  for a file and for a folder on a test window does not panic (R1.6).
- [x] AC4 `Opener()` names the platform tool; `OpenPath("")` returns an
  error without starting anything (R1.7).
- [x] AC5 The question `Decide` puts up answers true on yes, false on no and
  false on dismissal, exactly once each; the dialog is gone afterwards
  (R1.8). _Amended: driving `Decide` itself from a goroutine under the test
  driver races by construction (quirk 11), so the test drives `ask`; `Decide`
  is the hop plus the wait and is exercised by the gallery._
- [x] AC6 The model drops the oldest chunk at the cap and reports the count;
  `Text()` carries the preamble; `At` out of range is the zero line;
  `TakeDirty` clears (R2.2).
- [x] AC7 Canary `TestFollowTailToleratesFractionalOffsets` and the rule
  that scrolling up stops following (R2.3).
- [x] AC8 `Widget` renders headlessly; `Log` produces timestamped rows;
  `Writer` splits multi-line writes into rows; `Pump` redraws at least once
  after `Append` and once more after stop; Copy puts `Text()` on the
  clipboard and flashes; Clear resets and calls `OnClear` (R2.4).
- [x] AC9 `Form.Set` does not fire `OnChange`; typing into an entry does;
  `Values` round-trips every field kind including `Check` (R3.1, R3.2).
- [x] AC10 Canary `TestSetCheckedFiresOnChanged` (R3.1, quirk 2).
- [x] AC11 `IntRange` rejects non-numbers and out-of-range; `NumericEntry`
  calls `onChange` only for valid values (R3.3).
- [x] AC12 `SliderEntry`: dragging updates the entry without committing;
  release commits once; Enter in the entry commits; a non-number restores
  the text and calls `OnInvalid` (R3.4).
- [x] AC13 Gallery sections render headlessly in every scheme; README and
  design doc updated; `make test`, `make lint`, `go vet` clean (R4).

## Risks & Assumptions

- **Assumption**: `Decide` is only called off the UI thread. Under the test
  driver `fyne.Do` runs inline, so a test drives it from a goroutine and taps
  the overlay button from the test goroutine.
- **Risk**: `OpenPath` starts a desktop process; it is the one subprocess in
  the library, detached on purpose so closing the window does not close the
  user's editor. Not exercised in tests beyond the empty-path error.
- **Risk**: `canvas.Text` row height under the test driver differs from a
  real canvas; the pane's tests assert model and state, not pixel heights.
- **Rollback**: `git revert`; nothing consumes these packages yet.

## Alternatives Considered

- Considered binding forms to `fyne.io/fyne/v2/data/binding`; rejected as in
  spec 001, and because the apps' `Set`/`Values` map shape is what makes
  Save and Revert testable headlessly.
- Considered a `Field` per Go type (`IntField`, `BoolField`); rejected, the
  apps compare string maps against a settings file and `Numeric`/`Check`
  keep that while validating.
