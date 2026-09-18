# Spec 011: Program settings on disk

> **Note**: This work has no associated issue tracker ticket. The repository
> is a personal public project without an issue tracker.

## Status: COMPLETE

## Context

The library has no way for a program to keep a setting, and it has two halves of
one hiding in plain sight.

**The first half is the preference store.** `theme.LoadAppearance` and
`Appearance.Save` read and write five keys through `fyne.Preferences`, which
Fyne keeps in its own directory, keyed by `AppID`, in a flat file of untyped
keys. It works. It is also invisible: nothing in it is meant to be read, edited
or copied between machines by a person, and it holds one kind of setting — the
library's own appearance — with no room for a program's.

**The second half is `examples/settings`.** That example hand-rolls what is
missing: a JSON document under `os.UserConfigDir()`, read at start, written a
second after the last change through `forms.Saver`, with unknown keys preserved
across a save. Ninety lines of it, in an example, because the library it
demonstrates cannot do it.

Four programs would each write those ninety lines again. Two of them already
have: terrariabonker keeps its cheat profile in `~/.config/terrariabonker`, and
its Go window put a dragged divider position into `fyne.Preferences` because
that was the only store within reach — a number the user chose, kept somewhere
the user cannot see.

Specs 012 and 013 each need somewhere to keep a choice about the window: where a
divider was dragged, and which shape the navigation is in. Those are the
library's settings rather than any program's, and they must not become a fifth
hand-rolled file per consumer.

## Design

### One file, one API, two namespaces

A `settings` package holding a `Store` over a single file.

	st, err := settings.Open(path)          // missing file is not an error
	st.Get("jobs", &cfg)                    // decode one section into a struct
	st.Set("jobs", cfg)                     // replace it, schedule a save
	st.Flush()                              // write now; called before quitting

The file is one object whose top-level keys are sections, each decoded into
whatever the caller hands it:

	{
	  "fynedesygn.appearance": { "scheme": "Breeze Dark", "textSize": 13 },
	  "fynedesygn.nav":        { "mode": "icons", "placement": "left" },
	  "fynedesygn.splits":     { "nav": 0.16, "output": 0.78 },
	  "jobs":                  { "parallel": 4, "keepLogs": true }
	}

Keys beginning `fynedesygn.` are the library's. Everything else is the
program's, and the library never looks inside it.

**A section the running binary does not know is kept, not dropped.** Sections
are held as raw bytes until something asks for one, so an older build that reads
and saves a file written by a newer one preserves what it could not decode. The
opposite — a save that silently deletes a setting the user set — is the failure
this rule exists for.

### Where the file lives

`settings.Open` takes a path, and the shell supplies one so the common case
needs no decision:

	Options.SettingsPath  // explicit, wins when set
	                      // otherwise: os.UserConfigDir()/<AppID>/settings.json

`AppID` rather than `Name` because it is already the unique one — it names the
Fyne preference store and, on Wayland, the desktop entry. A program that already
owns a configuration directory (terrariabonker owns `~/.config/terrariabonker`)
points `SettingsPath` at it and keeps one directory rather than two.

### Writes

- **Atomically.** Written to a temporary file beside the target and renamed, so
  an interrupted save cannot truncate what was there. The example does not do
  this; a library that holds the user's settings should.
- **After a quiet period**: at most one write per second of typing, and `Flush`
  before the process quits or restarts. `Shell.Stop` calls it.

  Not through `forms.Saver`, though that is the same idea. `forms.Saver` runs
  its callback on the UI thread through `fyne.Do`, which is right for a
  program's document and wrong for a file write, and reusing it would pull
  `widgets` and `dialogs` into a package that is otherwise standard library
  alone.
- **0600 for the file, 0750 for the directory**, which is what the example and
  the repository's lint settings already use.

### Reading a file that does not parse

Renamed aside once, to `<name>.bad`, and the store starts from defaults. Not
deleted, because it is the user's, and not left in place, because leaving it
would mean every save from then on refuses or overwrites it unannounced. The
store reports what it did; `Shell` surfaces that through `Report`.

### Format: both, chosen by the file's extension

The encoding sits behind a two-method interface:

	type Codec interface {
	    Marshal(v any) ([]byte, error)
	    Unmarshal(b []byte, v any) error
	}

Two ship. `settings.JSON` is in the core, because `encoding/json` is in the
standard library and the core's rule is that Fyne is its only required runtime
dependency. `settings/yamlcodec` is a subpackage over `go.yaml.in/yaml/v3`.

**The cost of the second one is already paid.** All four consumers carry
`go.yaml.in/yaml/v3 v3.0.5` in their module graphs today, and clockwork-orange
depends on `gopkg.in/yaml.v3` directly. The subpackage adds a line to the
module's `go.mod` and to a `go.sum` that already has it, and reaches a binary
only when that binary imports it.

**A codec registers itself for its extensions**, the way `image/png` registers a
decoder:

	import _ "github.com/ushineko/fynedesygn/settings/yamlcodec"

	settings.Open(".../settings.yaml")   // YAML, because of the extension

`settings.Register(ext string, c Codec)` is the seam; `yamlcodec`'s `init` calls
it for `.yaml` and `.yml`, and the core registers `.json` for itself. This is
what keeps the core free of the dependency: nothing in it names the YAML package,
so a program that does not want a YAML parser in its binary does not get one.
`settings.OpenWith(path, codec)` sets a codec explicitly for a path whose
extension says nothing.

A path whose extension has no registered codec is an error naming the blank
import, not a JSON file written under a `.yaml` name.

**One set of struct tags for both formats.** The YAML codec marshals through
JSON — encode with `encoding/json`, decode into a generic value, emit YAML, and
the reverse — so the `json:` tags on a caller's struct decide the field names in
both files and the two formats hold the same keys. Numbers are decoded with
`UseNumber` so a whole number does not come back out as `4e+00`. The alternative,
`yaml:` tags beside the `json:` ones, is two sets of names to keep in step and
one silent difference the first time they drift.

### What moves onto it

`theme.Appearance` moves to the new store, read once from `fyne.Preferences`
when the store has no appearance section and written through, so no one loses
their theme. `Appearance.Save(fyne.Preferences)` stays as it is — it is public
API — and gains a sibling that takes a `*settings.Store`.

Two stores for one program is the thing this spec exists to end, so the library
stops writing to `fyne.Preferences` once the move is done.

## Requirements

- R1 `settings.Open(path)` returns a usable store whether or not the file
  exists. A missing file is the defaults, not an error.
- R2 `Get(key, &v)` decodes one section and reports whether there was one.
  `Set(key, v)` replaces a section and schedules a save.
- R3 A section the running binary never asks for survives a load and a save
  unchanged.
- R4 Saves are atomic: an interrupted write leaves the previous file intact.
- R5 Saves are debounced; `Flush` writes immediately and is called by the shell
  when the window closes and before the process restarts.
- R6 A file that does not parse is renamed to `<name>.bad` once, the store
  starts from defaults, and the caller is told.
- R7 Keys beginning `fynedesygn.` are reserved for the library and documented as
  such. The library reads and writes nothing else.
- R8 `Options.SettingsPath` overrides the default location; the default is
  `os.UserConfigDir()/<AppID>/settings.json`.
- R9 The encoding is behind `Codec`. `settings.JSON` is in the core and the core
  gains no new dependency; `settings/yamlcodec` is a subpackage over
  `go.yaml.in/yaml/v3`.
- R9a A codec registers itself for its extensions through `settings.Register`,
  and `Open` picks one from the path. Nothing in the core names the YAML
  package. `OpenWith` sets one explicitly.
- R9b An extension with no registered codec is an error that names the blank
  import needed for it.
- R9c A value written as JSON and read as YAML, or the reverse, has the same
  keys: the YAML codec goes through `encoding/json`, so `json:` tags decide both
  and whole numbers stay whole.
- R10 `theme.Appearance` is read from and written to the store, migrating once
  from `fyne.Preferences`.
- R11 The store is safe to use from the UI thread and the saver's goroutine at
  once.
- R12 The gallery shows it, `docs/design-system.md` records it, and the
  changelog names it.

## Acceptance Criteria

- [x] AC1 Opening a path with no file gives a store whose `Get` reports absent
  and whose `Set` then `Flush` creates the file and its directory (R1, R2).
- [x] AC2 A file holding a section this binary never names is still there,
  byte-identical in content, after a `Set` of a different section and a `Flush`
  (R3).
- [x] AC3 A save interrupted after the temporary file is written leaves the
  original readable (R4).
- [x] AC4 Two `Set` calls within the quiet period produce one write; `Flush`
  writes a pending change immediately and `Pending` then reports false (R5).
- [x] AC5 A file of invalid JSON is renamed to `<name>.bad`, the store loads
  defaults, and the error is returned from `Open` rather than swallowed (R6).
- [x] AC6 A program's `Set("jobs", …)` and the library's appearance survive each
  other; neither writes the other's key (R7).
- [x] AC7 With no `SettingsPath`, the shell uses
  `os.UserConfigDir()/<AppID>/settings.json`; with one, it uses that (R8).
- [x] AC8 A store over `settings.yaml` with `yamlcodec` imported writes YAML and
  reads it back; the same values written as JSON and renamed produce the same
  `Get` results, key for key (R9, R9a, R9c).
- [x] AC8a `Open` on a `.yaml` path without the codec imported returns an error
  naming the blank import, and writes nothing (R9b).
- [x] AC8b An integer set through the store is an integer in the YAML file, not
  a float (R9c).
- [x] AC8c The core package's import graph does not reach the YAML package —
  asserted by a test, not by reading (R9, R9a).
- [x] AC9 A program whose appearance is only in `fyne.Preferences` opens with
  that appearance and has it in the settings file afterwards (R10).
- [x] AC10 `go test -race` passes with a test that reads and writes the store
  from two goroutines (R11).
- [x] AC11 Gallery card, design-system entry, changelog; tests, lint and vet
  clean (R12).

## Risks & Assumptions

- **Assumption**: one file per program is enough. Nothing here needs a setting
  shared between programs, and a shared one would raise questions of ownership
  and locking that no consumer is asking.
- **Risk**: the appearance migration is the one part that can lose something a
  user set. It is one-way and one-time, and AC9 is the test that pins it. The
  old keys are read and not deleted, so a rollback finds them where they were.
- **Risk**: a save rewrites the file, so comments a user wrote into a YAML
  settings file do not survive the next change made in the window. No Go YAML
  encoder round-trips comments through a marshal of arbitrary values. Recorded
  rather than worked around: it is the one thing hand-editing YAML buys that
  this store cannot keep.
- **Resolved during implementation**: `Set` encodes the value at the call, so a
  value that cannot be encoded is an error where it was set rather than a silent
  failure a second later. A failed *write* still has nobody to return to, and
  goes to `OnError`, which the shell reports.
- **Rollback**: additive until R10. `git revert` of the migration commit alone
  returns the appearance to `fyne.Preferences` with the old keys still present.

## Alternatives Considered

- **Keep using `fyne.Preferences` and add keys.** Rejected: flat, untyped, and
  in a location chosen for the toolkit rather than for the user. A dragged
  divider is tolerable there; a program's own configuration is not.
- **A settings API with typed accessors (`GetString`, `GetInt`).** Rejected:
  every consumer would rebuild its own struct from loose keys, which is the work
  `encoding/json` already does. One section decoded into the caller's own type
  is less API and less code on both sides.
- **JSON alone.** Rejected at review: both formats ship. The dependency is
  already in every consumer's module graph, so the price the core's only-Fyne
  rule was protecting against is not being paid twice.
- **YAML in the core rather than a subpackage.** Rejected: it would put a YAML
  parser in the binary of every program that links the library, including the
  ones that write JSON. A blank import is one line for the programs that want it.
- **`yaml:` tags beside `json:` tags.** Rejected: two sets of names for one
  struct, kept in step by hand, and a file whose keys change when the format
  does. Marshalling through JSON gives one set of names for both.

Verified in `settings/settings_test.go`: `TestAMissingFileIsTheDefaults` (AC1),
`TestASectionThisBuildDoesNotKnowIsKept` (AC2),
`TestAFailedWriteLeavesTheOldSettingsIntact` (AC3),
`TestChangesInOneMomentAreOneWrite` and `TestSettingTheSameValueIsNotAChange`
(AC4), `TestAFileThatDoesNotParseIsMovedAside` (AC5),
`TestTheLibraryAndTheProgramShareOneFile` (AC6),
`TestTheStoreIsSafeFromSeveralGoroutines` (AC10), plus
`TestAValueThatCannotBeEncodedFailsAtTheCall`,
`TestASectionThatNoLongerFitsReadsAsAbsent`,
`TestAnUnknownExtensionIsRefusedAndSaysWhatIsMissing` and
`TestAFailedWriteIsReported`. In `settings/yamlcodec/yamlcodec_test.go`:
`TestAYamlPathIsYamlOnceTheCodecIsImported` and
`TestTheSameSettingsHaveTheSameKeysInEitherFormat` (AC8),
`TestAFileThatIsNotYamlIsMovedAside` (AC8a's sibling),
`TestAWholeNumberIsWrittenWhole` (AC8b) and
`TestTheCoreDoesNotLinkTheYamlPackage` (AC8c). In `shell/shell_test.go`:
`TestTheShellOpensASettingsFileForTheProgram` and
`TestABrokenSettingsFileStillLeavesAUsableStore` (AC7's sibling),
`TestTheDefaultSettingsPathIsUnderTheConfigDirectory` (AC7),
`TestSetAppearanceSavesUnlessTheSchemeIsAOneRunOverride` (AC9) and
`TestStopRunsOnceAndWritesWhatIsWaiting`. AC11: the "shell.Settings" card in
the gallery's Shell section, "State and freshness" in `docs/design-system.md`,
0.1.10 in the README changelog.

AC8a is covered by `TestAnUnknownExtensionIsRefusedAndSaysWhatIsMissing` for an
unregistered extension generally; the YAML-specific hint is in `CodecFor`'s
error text and is exercised there.
