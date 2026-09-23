/*
Package settings is a program's own settings, in one file the user can read.

Fyne has a preference store and the shell used to keep the appearance in it. It
is flat, untyped and lives in a directory named for the toolkit: nothing in it
is meant to be opened, edited or copied between machines by a person. A
program's settings are, so they go somewhere a person would look, in a format a
person can read.

One file holds one object whose top-level keys are sections, each decoded into
whatever the caller hands it:

	{
	  "fynedesygn.appearance": { "scheme": "Breeze Dark" },
	  "jobs":                  { "parallel": 4 }
	}

Keys beginning "fynedesygn." are the library's. Everything else is the
program's, and the library never looks inside it.

A section this build never asks for is kept, not dropped. Sections are held as
raw bytes until something asks for one, so an older build that reads and saves a
file written by a newer one preserves what it could not decode -- the opposite,
a save that quietly deletes a setting the user set, is the failure this rule
exists for.

See spec 011.
*/
package settings

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Prefix reserves the library's own sections. A program's key must not start
// with it.
const Prefix = "fynedesygn."

// DefaultDelay is how long a store waits after the last change before writing.
const DefaultDelay = time.Second

// File and directory modes. A settings file is the user's alone.
const (
	fileMode = 0o600
	dirMode  = 0o750
)

/*
Store is one settings file, loaded.

Safe from any goroutine: the document is held under a mutex, and the write that
follows a change happens on a timer rather than on the caller's thread.
*/
type Store struct {
	path  string
	codec Codec
	delay time.Duration

	mu    sync.Mutex
	doc   map[string]json.RawMessage
	dirty bool
	// unreadable is set when the file on disk did not parse. The store then
	// serves reads from defaults and refuses to write, because the one thing
	// worse than leaving a file nobody could read is overwriting it.
	unreadable *ParseError
	// seq lets a later change take ownership of the timer, so a run of changes
	// is one write a second after the last of them rather than one write each.
	seq   int
	onErr func(error)
}

/*
Open reads the settings at path, choosing a codec from its extension.

A missing file is not an error: it is a program that has never been configured,
and the store it returns writes one when something is set.

A file that does not parse is reported and left exactly as it is. Nothing on
disk is renamed, rewritten or removed. The store is still returned and still
serves reads, because a window that will not open because its settings file is
malformed is worse than one that opens on defaults -- but it refuses to save
until the caller calls Replace, since overwriting a file nobody could read
destroys the very thing its owner needs to fix.
*/
func Open(path string) (*Store, error) {
	c, err := CodecFor(path)
	if err != nil {
		return nil, err
	}
	return OpenWith(path, c)
}

/*
Memory is a store that holds settings for this run and writes nothing.

What a program gets when its settings file cannot be opened at all: the window
opens, every Get and Set works, and nothing is written anywhere. A caller never
has to check for a nil store.
*/
func Memory() *Store {
	return &Store{codec: JSON, delay: DefaultDelay, doc: map[string]json.RawMessage{}}
}

// OpenWith is Open with the codec named rather than inferred, for a path whose
// extension says nothing.
func OpenWith(path string, c Codec) (*Store, error) {
	if c == nil {
		return nil, errors.New("settings: no codec")
	}
	s := &Store{
		path:  path,
		codec: c,
		delay: DefaultDelay,
		doc:   map[string]json.RawMessage{},
	}
	return s, s.load()
}

/*
load reads the file into the document.

A file that does not parse is **left exactly where it is**. Nothing is renamed,
written or removed: a caller that asked to read settings has not asked this
package to rearrange the user's configuration directory, and the moment a file
fails to parse is the moment its owner most needs to find it where they left it.

The error carries the codec's own, which knows the line and column, so a program
can tell someone where the mistake is. The store then refuses to save until the
caller decides what to do about it -- see ParseError and Replace.
*/
func (s *Store) load() error {
	b, err := os.ReadFile(s.path) //nolint:gosec // the path the caller chose
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read %s: %w", s.path, err)
	}
	if len(b) == 0 {
		return nil
	}

	var doc map[string]json.RawMessage
	if err := s.codec.Unmarshal(b, &doc); err != nil {
		s.unreadable = &ParseError{Path: s.path, Err: err}
		return s.unreadable
	}
	if doc != nil {
		s.doc = doc
	}
	return nil
}

/*
ParseError is a settings file that could not be decoded.

It carries the codec's error, which for JSON and YAML alike names the line and
column, so a caller can show the user where to look rather than only that
something was wrong.
*/
type ParseError struct {
	Path string
	Err  error
}

// Error names the section that could not be decoded, so a program reporting
// this tells its user which part of their file to look at.
func (e *ParseError) Error() string {
	return fmt.Sprintf("%s does not parse: %v (settings will not be saved until it is fixed or replaced)",
		e.Path, e.Err)
}

// Unwrap is the codec's own error, for a caller that wants to match on it
// rather than on the section it came from.
func (e *ParseError) Unwrap() error { return e.Err }

/*
Unreadable is the error from a file that did not parse, or nil.

A caller can ask before writing rather than learning it from a failed Set.
*/
func (s *Store) Unreadable() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.unreadable == nil {
		return nil
	}
	return s.unreadable
}

/*
Replace abandons an unreadable file's contents, so the store saves again and the
next write overwrites it.

The caller decides this and the library never does. Refusing to save is the only
honest thing a package can do with a file it could not read: renaming it hides
the user's work somewhere they did not choose, and overwriting it destroys it.
Replace is how a program says the user has been told and has chosen to start
again.
*/
func (s *Store) Replace() {
	s.mu.Lock()
	s.unreadable = nil
	s.doc = map[string]json.RawMessage{}
	s.mu.Unlock()
}

// Path is the file this store reads and writes.
func (s *Store) Path() string { return s.path }

// OnError registers what to do with a write that fails. A save happens on a
// timer, long after the call that caused it, so there is nobody left to return
// an error to. The shell reports it.
func (s *Store) OnError(fn func(error)) {
	s.mu.Lock()
	s.onErr = fn
	s.mu.Unlock()
}

// SetDelay changes the quiet period before a write. For tests, and for a
// program that wants its settings on disk sooner.
func (s *Store) SetDelay(d time.Duration) {
	s.mu.Lock()
	if d > 0 {
		s.delay = d
	}
	s.mu.Unlock()
}

/*
Get decodes one section into v and reports whether there was one.

A section that will not decode into v reads as absent: a caller that has changed
the shape of its own settings gets its defaults rather than an error it has no
way to act on, and the next Set replaces what was there.
*/
func (s *Store) Get(key string, v any) bool {
	s.mu.Lock()
	raw, ok := s.doc[key]
	s.mu.Unlock()
	if !ok || len(raw) == 0 {
		return false
	}
	return json.Unmarshal(raw, v) == nil
}

/*
Set replaces one section and schedules a write.

The value is encoded here rather than at save time, so a value that cannot be
encoded is an error at the call that made it instead of a silent failure a
second later with nobody to tell.
*/
func (s *Store) Set(key string, v any) error {
	raw, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("encode settings section %q: %w", key, err)
	}
	s.mu.Lock()
	if s.unreadable != nil {
		err := s.unreadable
		s.mu.Unlock()
		return err
	}
	same := string(s.doc[key]) == string(raw)
	if !same {
		s.doc[key] = raw
		s.dirty = true
		s.seq++
		seq := s.seq
		delay := s.delay
		s.mu.Unlock()
		go s.writeAfter(delay, seq)
		return nil
	}
	s.mu.Unlock()
	return nil
}

// Has reports whether a section is present, without decoding it.
func (s *Store) Has(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.doc[key]
	return ok
}

// Pending reports whether a change is waiting for the quiet period.
func (s *Store) Pending() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.dirty
}

// writeAfter is one scheduled save. A later change bumps the sequence and this
// one stands down.
func (s *Store) writeAfter(delay time.Duration, seq int) {
	time.Sleep(delay)
	s.mu.Lock()
	if s.seq != seq || !s.dirty {
		s.mu.Unlock()
		return
	}
	s.mu.Unlock()
	if err := s.Flush(); err != nil {
		s.report(err)
	}
}

// report hands a failed write to whoever asked to hear about them.
func (s *Store) report(err error) {
	s.mu.Lock()
	fn := s.onErr
	s.mu.Unlock()
	if fn != nil {
		fn(err)
	}
}

/*
Flush writes a pending change now.

Called before the process quits or restarts, and by the timer. Nothing to write
is not an error and not a write: a program that flushes on exit does not rewrite
a file it never changed.
*/
func (s *Store) Flush() error {
	s.mu.Lock()
	if s.unreadable != nil {
		err := s.unreadable
		s.mu.Unlock()
		return err
	}
	if !s.dirty {
		s.mu.Unlock()
		return nil
	}
	s.dirty = false
	s.seq++ // a timer in flight finds a different sequence and stands down
	if s.path == "" {
		s.mu.Unlock()
		return nil // a store with nowhere to write: see Memory
	}
	doc := make(map[string]json.RawMessage, len(s.doc))
	for k, v := range s.doc {
		doc[k] = v
	}
	codec, path := s.codec, s.path
	s.mu.Unlock()

	b, err := codec.Marshal(doc)
	if err != nil {
		return err
	}
	return writeAtomic(path, b)
}

/*
writeAtomic writes through a temporary file in the same directory and renames.

A settings file rewritten in place is a file that a crash, a full disk or a
killed process can leave truncated -- and the first thing anyone would do with a
truncated settings file is lose whatever was in it. A rename within one
directory is atomic, so the file is either the old one or the new one.
*/
func writeAtomic(path string, b []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, dirMode); err != nil {
		return fmt.Errorf("create %s: %w", dir, err)
	}
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".*")
	if err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	name := tmp.Name()
	defer func() { _ = os.Remove(name) }() // a no-op once the rename has happened

	if err := tmp.Chmod(fileMode); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write %s: %w", path, err)
	}
	if _, err := tmp.Write(b); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write %s: %w", path, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	if err := os.Rename(name, path); err != nil {
		return fmt.Errorf("replace %s: %w", path, err)
	}
	return nil
}

// DefaultPath is where a program's settings live when it does not say: the
// user's configuration directory, a directory named for the program, and a
// JSON file in it.
func DefaultPath(appID string) (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("find the configuration directory: %w", err)
	}
	return filepath.Join(dir, appID, "settings.json"), nil
}
