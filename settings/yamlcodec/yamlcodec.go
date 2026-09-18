/*
Package yamlcodec writes a settings file as YAML.

Imported for its effect: it registers itself for ".yaml" and ".yml", so a
settings path with either extension is YAML from then on.

	import _ "github.com/ushineko/fynedesygn/settings/yamlcodec"

A subpackage rather than part of settings, so a program that writes JSON does
not carry a YAML parser in its binary. Nothing in the core names this package.

It encodes through JSON. The json struct tags a caller already has decide the
field names in both formats, so the same settings written either way hold the
same keys -- where a second set of yaml tags would be two sets of names to keep
in step and one silent difference the first time they drift.

One thing YAML buys that this cannot keep: a save rewrites the file, so comments
written into it by hand do not survive the next change made in the window.

See spec 011.
*/
package yamlcodec

import (
	"bytes"
	"encoding/json"
	"fmt"

	yaml "go.yaml.in/yaml/v3"

	"github.com/ushineko/fynedesygn/settings"
)

// Codec is the YAML codec, for a caller that wants it by name rather than by
// file extension (settings.OpenWith).
var Codec settings.Codec = codec{}

// yamlIndent is two spaces, which is what a person writing YAML by hand uses.
const yamlIndent = 2

func init() {
	settings.Register(".yaml", Codec)
	settings.Register(".yml", Codec)
}

type codec struct{}

// Marshal encodes v as JSON first, so json tags name the fields, then writes
// that shape as YAML.
func (codec) Marshal(v any) ([]byte, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("encode settings: %w", err)
	}
	generic, err := decodeJSON(b)
	if err != nil {
		return nil, err
	}

	var out bytes.Buffer
	enc := yaml.NewEncoder(&out)
	enc.SetIndent(yamlIndent)
	if err := enc.Encode(generic); err != nil {
		return nil, fmt.Errorf("encode settings as YAML: %w", err)
	}
	if err := enc.Close(); err != nil {
		return nil, fmt.Errorf("encode settings as YAML: %w", err)
	}
	return out.Bytes(), nil
}

// Unmarshal reads YAML into a generic value and hands it to encoding/json, so
// the same json tags decode it.
func (codec) Unmarshal(b []byte, v any) error {
	var generic any
	if err := yaml.Unmarshal(b, &generic); err != nil {
		return fmt.Errorf("decode settings as YAML: %w", err)
	}
	j, err := json.Marshal(generic)
	if err != nil {
		return fmt.Errorf("decode settings as YAML: %w", err)
	}
	if err := json.Unmarshal(j, v); err != nil {
		return fmt.Errorf("decode settings: %w", err)
	}
	return nil
}

/*
decodeJSON reads JSON into a generic value, keeping whole numbers whole.

encoding/json decodes every number as a float64 unless asked otherwise, and a
float64 4 written as YAML is "4" only by luck of formatting -- 216000 becomes
2.16e+05. UseNumber keeps the text, and plain turns it back into an integer
where it is one, so a settings file holds the numbers the program set.
*/
func decodeJSON(b []byte) (any, error) {
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, fmt.Errorf("encode settings: %w", err)
	}
	return plain(v), nil
}

// plain replaces json.Number with the narrowest of int64 and float64 that holds
// it, through maps and slices.
func plain(v any) any {
	switch t := v.(type) {
	case json.Number:
		if n, err := t.Int64(); err == nil {
			return n
		}
		if f, err := t.Float64(); err == nil {
			return f
		}
		return t.String()
	case map[string]any:
		for k, val := range t {
			t[k] = plain(val)
		}
		return t
	case []any:
		for i, val := range t {
			t[i] = plain(val)
		}
		return t
	}
	return v
}
