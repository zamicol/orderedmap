// Package orderedmap provides OrderedMap, a map whose keys keep the order in
// which they were added and which round-trips that order through JSON.
//
// encoding/json has no way to preserve the order of object keys. See
// https://github.com/golang/go/issues/27179.
//
// By default, MarshalJSON does not HTML-escape <, >, or &, unlike
// encoding/json. Use SetEscapeHTML(true) to enable that escaping.
// encoding/json.Marshal will still HTML-escape the result; call
// MarshalJSON directly, or use a json.Encoder with SetEscapeHTML(false),
// to keep the unescaped output.
package orderedmap

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
)

type Pair struct {
	key   string
	value any
}

func (kv *Pair) Key() string {
	return kv.key
}

func (kv *Pair) Value() any {
	return kv.value
}

type ByPair struct {
	Pairs    []*Pair
	LessFunc func(a *Pair, j *Pair) bool
}

func (a ByPair) Len() int           { return len(a.Pairs) }
func (a ByPair) Swap(i, j int)      { a.Pairs[i], a.Pairs[j] = a.Pairs[j], a.Pairs[i] }
func (a ByPair) Less(i, j int) bool { return a.LessFunc(a.Pairs[i], a.Pairs[j]) }

type OrderedMap struct {
	keys              []string
	values            map[string]any
	escapeHTML        bool
	jsonLastValueWins bool // Permits duplicates on JSON unmarshal.  Otherwise errors on duplicates on unmarshal.
}

// New returns an empty OrderedMap. HTML escaping is off by default.
func New() *OrderedMap {
	o := OrderedMap{}
	o.keys = []string{}
	o.values = map[string]any{}
	o.escapeHTML = false
	return &o
}

// SetEscapeHTML sets whether MarshalJSON escapes <, >, and & as \u003c,
// \u003e, and \u0026. HTML escaping is off by default, unlike encoding/json.
// Call SetEscapeHTML(true) to match encoding/json's default.
//
// encoding/json.Marshal always HTML-escapes MarshalJSON's output. To get
// unescaped output, call MarshalJSON directly or use json.NewEncoder and
// Encoder.SetEscapeHTML(false).
func (o *OrderedMap) SetEscapeHTML(on bool) {
	o.escapeHTML = on
}

func (o *OrderedMap) SetJSONLastValueWins(on bool) {
	o.jsonLastValueWins = on
}

func (o *OrderedMap) Get(key string) (any, bool) {
	val, exists := o.values[key]
	return val, exists
}

func (o *OrderedMap) Set(key string, value any) {
	_, exists := o.values[key]
	if !exists {
		o.keys = append(o.keys, key)
	}
	o.values[key] = value
}

func (o *OrderedMap) Delete(key string) {
	// check key is in use
	_, ok := o.values[key]
	if !ok {
		return
	}
	// remove from keys
	for i, k := range o.keys {
		if k == key {
			o.keys = append(o.keys[:i], o.keys[i+1:]...)
			break
		}
	}
	// remove from values
	delete(o.values, key)
}

func (o *OrderedMap) Keys() []string {
	return o.keys
}

// Values returns the values in key order.
func (o *OrderedMap) Values() []any {
	v := make([]any, len(o.keys))
	for i, k := range o.keys {
		v[i] = o.values[k]
	}
	return v
}

// SortKeys sorts the map keys using the provided sort func.
func (o *OrderedMap) SortKeys(sortFunc func(keys []string)) {
	sortFunc(o.keys)
}

// Sort sorts the map using the provided less func.
func (o *OrderedMap) Sort(lessFunc func(a *Pair, b *Pair) bool) {
	pairs := make([]*Pair, len(o.keys))
	for i, key := range o.keys {
		pairs[i] = &Pair{key, o.values[key]}
	}

	sort.Sort(ByPair{pairs, lessFunc})

	for i, pair := range pairs {
		o.keys[i] = pair.key
	}
}

// UnmarshalJSON errors on duplicate JSON keys unless SetJSONLastValueWins(true)
// was called. "Last value wins" is bad practice. See CheckDuplicate.
// A JSON null unmarshals to an empty OrderedMap.
//
// encoding/json sets a pointer to nil when unmarshaling JSON null into *T, and
// does not call UnmarshalJSON. To apply null to an existing OrderedMap, pass
// the *OrderedMap to json.Unmarshal (not a pointer to it) or call
// UnmarshalJSON directly.
func (o *OrderedMap) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if bytes.Equal(b, []byte("null")) {
		o.keys = []string{}
		o.values = map[string]any{}
		return nil
	}

	var err error
	if !o.jsonLastValueWins {
		_, err = CheckDuplicate(json.NewDecoder(bytes.NewReader(b)))
		if err != nil {
			return err
		}
	}
	if o.values == nil {
		o.values = map[string]any{}
	}
	err = json.Unmarshal(b, &o.values)
	if err != nil {
		return err
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	if _, err = dec.Token(); err != nil { // skip '{'
		return err
	}
	o.keys = make([]string, 0, len(o.values))
	return decodeOrderedMap(dec, o)
}

func decodeOrderedMap(dec *json.Decoder, o *OrderedMap) error {
	hasKey := make(map[string]bool, len(o.values))
	for {
		token, err := dec.Token()
		if err != nil {
			return err
		}
		if delim, ok := token.(json.Delim); ok && delim == '}' {
			return nil
		}
		key := token.(string)
		if hasKey[key] {
			// duplicate key
			for j, k := range o.keys {
				if k == key {
					copy(o.keys[j:], o.keys[j+1:])
					break
				}
			}
			o.keys[len(o.keys)-1] = key
		} else {
			hasKey[key] = true
			o.keys = append(o.keys, key)
		}

		token, err = dec.Token()
		if err != nil {
			return err
		}
		if delim, ok := token.(json.Delim); ok {
			switch delim {
			case '{':
				if values, ok := o.values[key].(map[string]any); ok {
					newMap := OrderedMap{
						keys:       make([]string, 0, len(values)),
						values:     values,
						escapeHTML: o.escapeHTML,
					}
					if err = decodeOrderedMap(dec, &newMap); err != nil {
						return err
					}
					o.values[key] = newMap
				} else if oldMap, ok := o.values[key].(OrderedMap); ok {
					newMap := OrderedMap{
						keys:       make([]string, 0, len(oldMap.values)),
						values:     oldMap.values,
						escapeHTML: o.escapeHTML,
					}
					if err = decodeOrderedMap(dec, &newMap); err != nil {
						return err
					}
					o.values[key] = newMap
				} else if err = decodeOrderedMap(dec, &OrderedMap{}); err != nil {
					return err
				}
			case '[':
				if values, ok := o.values[key].([]any); ok {
					if err = decodeSlice(dec, values, o.escapeHTML); err != nil {
						return err
					}
				} else if err = decodeSlice(dec, []any{}, o.escapeHTML); err != nil {
					return err
				}
			}
		}
	}
}

func decodeSlice(dec *json.Decoder, s []any, escapeHTML bool) error {
	for index := 0; ; index++ {
		token, err := dec.Token()
		if err != nil {
			return err
		}
		if delim, ok := token.(json.Delim); ok {
			switch delim {
			case '{':
				if index < len(s) {
					if values, ok := s[index].(map[string]any); ok {
						newMap := OrderedMap{
							keys:       make([]string, 0, len(values)),
							values:     values,
							escapeHTML: escapeHTML,
						}
						if err = decodeOrderedMap(dec, &newMap); err != nil {
							return err
						}
						s[index] = newMap
					} else if oldMap, ok := s[index].(OrderedMap); ok {
						newMap := OrderedMap{
							keys:       make([]string, 0, len(oldMap.values)),
							values:     oldMap.values,
							escapeHTML: escapeHTML,
						}
						if err = decodeOrderedMap(dec, &newMap); err != nil {
							return err
						}
						s[index] = newMap
					} else if err = decodeOrderedMap(dec, &OrderedMap{}); err != nil {
						return err
					}
				} else if err = decodeOrderedMap(dec, &OrderedMap{}); err != nil {
					return err
				}
			case '[':
				if index < len(s) {
					if values, ok := s[index].([]any); ok {
						if err = decodeSlice(dec, values, escapeHTML); err != nil {
							return err
						}
					} else if err = decodeSlice(dec, []any{}, escapeHTML); err != nil {
						return err
					}
				} else if err = decodeSlice(dec, []any{}, escapeHTML); err != nil {
					return err
				}
			case ']':
				return nil
			}
		}
	}
}

// MarshalJSON writes keys in order as compact JSON, with no trailing newlines
// from the encoder. Output has no duplicate keys since OrderedMap keys are
// unique.
func (o OrderedMap) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(o.escapeHTML)
	for i, k := range o.keys {
		if i > 0 {
			buf.WriteByte(',')
		}
		// add key
		if err := encoder.Encode(k); err != nil {
			return nil, err
		}
		buf.Truncate(buf.Len() - 1) // strip Encode's trailing newline
		buf.WriteByte(':')
		// add value
		if err := encoder.Encode(o.values[k]); err != nil {
			return nil, err
		}
		buf.Truncate(buf.Len() - 1) // strip Encode's trailing newline
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

// ErrJSONDuplicate is returned when CheckDuplicate or UnmarshalJSON encounters
// a duplicate JSON object key. Compare with errors.Is. To get the field name,
// use errors.As with *DuplicateFieldError.
var ErrJSONDuplicate = errors.New("orderedmap: JSON duplicate field")

// DuplicateFieldError names the duplicate JSON key. It unwraps to
// ErrJSONDuplicate.
//
//	var e *DuplicateFieldError
//	if errors.As(err, &e) {
//		_ = e.Field
//	}
type DuplicateFieldError struct {
	Field string
}

func (e *DuplicateFieldError) Error() string {
	return fmt.Sprintf("orderedmap: JSON duplicate field %q", e.Field)
}

func (e *DuplicateFieldError) Unwrap() error {
	return ErrJSONDuplicate
}

// CheckDuplicate checks for JSON duplicates on ingest (unmarshal). If there are
// no duplicates, it returns the top-level unique keys in order. Note that
// Go maps and structs and Javascript objects (ES6) already require unique
// JSON names. Duplicate keys return a *DuplicateFieldError, which unwraps to
// ErrJSONDuplicate.
//
// Duplicate JSON fields are a security issue that wasn't addressed by the
// original spec that results in surprising behavior and is a source of bugs.
// See the article, "[An Exploration of JSON Interoperability
// Vulnerabilities](https://bishopfox.com/blog/json-interoperability-vulnerabilities)"
// and control-f "duplicate".
//
// encoding/json/v2, stable in Go 1.27, rejects duplicate keys. This helper
// remains useful with encoding/json v1, which still uses last-value-wins, and
// because it returns the top-level keys when the input has no duplicates.
//
// After JSON was widely adopted, Douglas Crockford (JSON's inventor), tried to
// fix this by updating JSON to define "must error on duplicates" as the correct
// behavior, but it was decided it was too late
// (https://esdiscuss.org/topic/json-duplicate-keys).
//
// Although Douglas Crockford couldn't change the JSON spec to force
// implementations to error on duplicate, his Java JSON implementation errors on
// duplicates. Other implementations' behaviors are `last-value-wins`, support
// duplicate keys, or other non-standard behavior. The [JSON
// RFC](https://datatracker.ietf.org/doc/html/rfc8259#section-4) states that
// implementations should not allow duplicate keys.  It then notes the varying behavior
// of existing implementations.
//
// Disallowing duplicates conforms to the small I-JSON RFC. The author of
// I-JSON, Tim Bray, is also the author of current JSON specification (RFC
// 8259).  See also https://github.com/json5/json5-spec/issues/38.
func CheckDuplicate(d *json.Decoder) (topLevelObjectKeys []string, e error) {
	t, err := d.Token()
	if err != nil {
		return nil, err
	}

	delim, ok := t.(json.Delim) // Is it a delimiter?
	if !ok {
		return nil, nil // scalar type, nothing to do
	}

	switch delim {
	case '{':
		keysBool := make(map[string]bool)
		var keys []string // preserves order
		for d.More() {
			t, err := d.Token() // Get field key.
			if err != nil {
				return nil, err
			}

			key := t.(string)
			if keysBool[key] { // Check for duplicates.
				return nil, &DuplicateFieldError{Field: key}
			}
			keysBool[key] = true
			keys = append(keys, key)
			// Recursive, Check value in case value is object.
			if _, err = CheckDuplicate(d); err != nil {
				return nil, err
			}
		}
		if _, err := d.Token(); err != nil { // consume trailing }
			return nil, err
		}
		return keys, nil

	case '[':
		for d.More() {
			if _, err := CheckDuplicate(d); err != nil {
				return nil, err
			}
		}
		// consume trailing ]
		if _, err := d.Token(); err != nil {
			return nil, err
		}
	}
	return nil, nil
}
