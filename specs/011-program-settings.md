# Spec 011: Program settings on disk

> **Note**: This work has no associated issue tracker ticket. The repository
> is a personal public project without an issue tracker.

## Status: PROPOSED — awaiting review

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
- **After a quiet period**, through `forms.Saver`, which already exists for
  exactly this: at most one write per second of typing, and `Flush` before the
  process quits or restarts. `Shell` calls `Flush` in its stop path.
- **0600 for the file, 0750 for the directory**, which is what the example and
  the repository's lint settings already use.

### Reading a file that does not parse

Renamed aside once, to `<name>.bad`, and the store starts from defaults. Not
deleted, because it is the user's, and not left in place, because leaving it
would mean every save from then on refuses or overwrites it unannounced. The
store reports what it did; `Shell` surfaces that through `Report`.

### Format

The encoding sits behind a two-method interface:

	type Codec interface {
	    Marshal(v any) ([]byte, error)
	    Unmarshal(b []byte, v any) error
	}

`settings.JSON` is the default and lives in the core, because `encoding/json` is
in the standard library and the library's rule is that Fyne is the only required
runtime dependency of the core packages.

A YAML codec would be `settings/yamlcodec`, a subpackage, added without changing
this API. The cost is stated plainly rather than assumed away: a dependency in
any package of a module is a line in the module's `go.mod` and in every
consumer's `go.sum`, even for a consumer that never imports that subpackage. It
does not reach a binary that does not import it.

**Open question for review**: JSON only, or JSON now with the YAML subpackage in
the same change? The interface makes the second cheap and reversible either way.

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
- R5 Saves are debounced through `forms.Saver`; `Flush` writes immediately and
  is called by the shell before the process quits or restarts.
- R6 A file that does not parse is renamed to `<name>.bad` once, the store
  starts from defaults, and the caller is told.
- R7 Keys beginning `fynedesygn.` are reserved for the library and documented as
  such. The library reads and writes nothing else.
- R8 `Options.SettingsPath` overrides the default location; the default is
  `os.UserConfigDir()/<AppID>/settings.json`.
- R9 The encoding is behind `Codec`; `settings.JSON` is the default and the core
  gains no new dependency.
- R10 `theme.Appearance` is read from and written to the store, migrating once
  from `fyne.Preferences`.
- R11 The store is safe to use from the UI thread and the saver's goroutine at
  once.
- R12 The gallery shows it, `docs/design-system.md` records it, and the
  changelog names it.

## Acceptance Criteria

- [ ] AC1 Opening a path with no file gives a store whose `Get` reports absent
  and whose `Set` then `Flush` creates the file and its directory (R1, R2).
- [ ] AC2 A file holding a section this binary never names is still there,
  byte-identical in content, after a `Set` of a different section and a `Flush`
  (R3).
- [ ] AC3 A save interrupted after the temporary file is written leaves the
  original readable (R4).
- [ ] AC4 Two `Set` calls within the quiet period produce one write; `Flush`
  writes a pending change immediately and `Pending` then reports false (R5).
- [ ] AC5 A file of invalid JSON is renamed to `<name>.bad`, the store loads
  defaults, and the error is returned from `Open` rather than swallowed (R6).
- [ ] AC6 A program's `Set("jobs", …)` and the library's appearance survive each
  other; neither writes the other's key (R7).
- [ ] AC7 With no `SettingsPath`, the shell uses
  `os.UserConfigDir()/<AppID>/settings.json`; with one, it uses that (R8).
- [ ] AC8 A store built with a second `Codec` round-trips the same values (R9).
- [ ] AC9 A program whose appearance is only in `fyne.Preferences` opens with
  that appearance and has it in the settings file afterwards (R10).
- [ ] AC10 `go test -race` passes with a test that reads and writes the store
  from two goroutines (R11).
- [ ] AC11 Gallery card, design-system entry, changelog; tests, lint and vet
  clean (R12).

## Risks & Assumptions

- **Assumption**: one file per program is enough. Nothing here needs a setting
  shared between programs, and a shared one would raise questions of ownership
  and locking that no consumer is asking.
- **Risk**: the appearance migration is the one part that can lose something a
  user set. It is one-way and one-time, and AC9 is the test that pins it. The
  old keys are read and not deleted, so a rollback finds them where they were.
- **Risk**: `Set` takes `any` and encodes it, so a value that cannot be encoded
  fails at save rather than at the call. The store reports it through the same
  path as a failed write.
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
- **YAML in the core.** Deferred rather than rejected; see "Format" and the open
  question. It buys comments in a hand-edited file at the price of the core's
  only-Fyne dependency rule.
