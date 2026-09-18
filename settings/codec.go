package settings

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
)

/*
Codec is how a store's document becomes bytes and back.

Two ship. JSON is here, because encoding/json is in the standard library and
this package's rule is the library's: nothing outside it is required to read a
program's settings. YAML is settings/yamlcodec, a subpackage, so a program that
writes JSON does not carry a YAML parser in its binary.
*/
type Codec interface {
	Marshal(v any) ([]byte, error)
	Unmarshal(b []byte, v any) error
}

// JSON is the default codec, and the one the core registers for ".json".
var JSON Codec = jsonCodec{}

// jsonCodec writes the document indented, because a settings file is one a
// person may open.
type jsonCodec struct{}

func (jsonCodec) Marshal(v any) ([]byte, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode settings as JSON: %w", err)
	}
	return append(b, '\n'), nil
}

func (jsonCodec) Unmarshal(b []byte, v any) error {
	if err := json.Unmarshal(b, v); err != nil {
		return fmt.Errorf("decode settings as JSON: %w", err)
	}
	return nil
}

// codecs is the extension registry. A codec registers itself, the way
// image/png registers a decoder, so that nothing in this package names the
// packages that implement the other formats.
var (
	codecMu sync.RWMutex
	codecs  = map[string]Codec{".json": JSON}
)

/*
Register makes c the codec for files with extension ext (".yaml", with the dot,
case-insensitive).

Called from a codec package's init. Registering twice for one extension replaces
the earlier codec rather than failing: a program that wants its own encoding for
a well-known extension is entitled to it, and a panic at init would be a poor
way to learn about the collision.
*/
func Register(ext string, c Codec) {
	if c == nil || ext == "" {
		return
	}
	codecMu.Lock()
	defer codecMu.Unlock()
	codecs[strings.ToLower(ext)] = c
}

// Registered lists the extensions a codec has been registered for, in no
// particular order. For an error message, and for a test that wants to know
// what this binary can read.
func Registered() []string {
	codecMu.RLock()
	defer codecMu.RUnlock()
	out := make([]string, 0, len(codecs))
	for ext := range codecs {
		out = append(out, ext)
	}
	return out
}

/*
CodecFor is the codec for a path's extension.

An unregistered extension is an error naming what would make it work, rather
than a silent fallback to JSON: a JSON document written to a file called
settings.yaml is worse than a refusal, because nothing downstream would notice.
*/
func CodecFor(path string) (Codec, error) {
	ext := strings.ToLower(filepath.Ext(path))
	codecMu.RLock()
	c, ok := codecs[ext]
	codecMu.RUnlock()
	if ok {
		return c, nil
	}
	return nil, fmt.Errorf("settings: no codec for %q%s", ext, hint(ext))
}

// hint names the import that would register a format this build knows about but
// has not linked. The path is a string here; naming the package in an import
// would put its dependency in every binary.
func hint(ext string) string {
	if ext == ".yaml" || ext == ".yml" {
		return ` (import _ "github.com/ushineko/fynedesygn/settings/yamlcodec" for YAML)`
	}
	return ""
}
