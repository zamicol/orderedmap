package orderedmap

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestOrderedMap(t *testing.T) {
	o := New()
	// number
	o.Set("number", 3)
	v, _ := o.Get("number")
	if v.(int) != 3 {
		t.Error("Set number")
	}
	// string
	o.Set("string", "x")
	v, _ = o.Get("string")
	if v.(string) != "x" {
		t.Error("Set string")
	}
	// string slice
	o.Set("strings", []string{
		"t",
		"u",
	})
	v, _ = o.Get("strings")
	if v.([]string)[0] != "t" {
		t.Error("Set strings first index")
	}
	if v.([]string)[1] != "u" {
		t.Error("Set strings second index")
	}
	// mixed slice
	o.Set("mixed", []any{
		1,
		"1",
	})
	v, _ = o.Get("mixed")
	if v.([]any)[0].(int) != 1 {
		t.Error("Set mixed int")
	}
	if v.([]any)[1].(string) != "1" {
		t.Error("Set mixed string")
	}
	// overriding existing key
	o.Set("number", 4)
	v, _ = o.Get("number")
	if v.(int) != 4 {
		t.Error("Override existing key")
	}
	// Keys method
	keys := o.Keys()
	expectedKeys := []string{
		"number",
		"string",
		"strings",
		"mixed",
	}
	for i, key := range keys {
		if key != expectedKeys[i] {
			t.Error("Keys method", key, "!=", expectedKeys[i])
		}
	}
	// Values method
	values := o.Values()
	expectedValues := []any{
		4, // 3 is overwritten
		"x",
		[]string{"t", "u"},
		[]any{1, "1"},
	}
	for i, val := range values {
		if !reflect.DeepEqual(expectedValues[i], val) {
			t.Error("Values method", expectedValues[i], "!=", val)
		}
	}
	// delete
	o.Delete("strings")
	o.Delete("not a key being used")
	if len(o.Keys()) != 3 {
		t.Error("Delete method")
	}
	_, ok := o.Get("strings")
	if ok {
		t.Error("Delete did not remove 'strings' key")
	}
}

func TestValuesOrder(t *testing.T) {
	o := New()
	o.Set("c", 3)
	o.Set("a", 1)
	o.Set("b", 2)
	got := o.Values()
	want := []any{3, 1, 2}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Values() = %v, want %v (insertion order)", got, want)
	}
	o.SortKeys(sort.Strings)
	got = o.Values()
	want = []any{1, 2, 3}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Values() after SortKeys = %v, want %v", got, want)
	}
}

func TestBlankMarshalJSON(t *testing.T) {
	o := New()
	// blank map
	b, err := json.Marshal(o)
	if err != nil {
		t.Error("Marshalling blank map to json", err)
	}
	s := string(b)
	// check json is correctly ordered
	if s != `{}` {
		t.Error("JSON Marshaling blank map value is incorrect", s)
	}
	// convert to indented json
	bi, err := json.MarshalIndent(o, "", "  ")
	if err != nil {
		t.Error("Marshalling indented json for blank map", err)
	}
	si := string(bi)
	ei := `{}`
	if si != ei {
		fmt.Println(ei)
		fmt.Println(si)
		t.Error("JSON MarshalIndent blank map value is incorrect", si)
	}
}

func TestMarshalJSON(t *testing.T) {
	o := New()
	// number
	o.Set("number", 3)
	// string
	o.Set("string", "x")
	// string
	o.Set("specialstring", "\\.<>[]{}_-")
	// new value keeps key in old position
	o.Set("number", 4)
	// keys not sorted alphabetically
	o.Set("z", 1)
	o.Set("a", 2)
	o.Set("b", 3)
	// slice
	o.Set("slice", []any{
		"1",
		1,
	})
	// orderedmap
	v := New()
	v.Set("e", 1)
	v.Set("a", 2)
	o.Set("orderedmap", v)
	// escape key
	o.Set("test\n\r\t\\\"ing", 9)
	// convert to json
	b, err := json.Marshal(o)
	if err != nil {
		t.Error("Marshalling json", err)
	}
	s := string(b)
	// check json is correctly ordered
	if s != `{"number":4,"string":"x","specialstring":"\\.\u003c\u003e[]{}_-","z":1,"a":2,"b":3,"slice":["1",1],"orderedmap":{"e":1,"a":2},"test\n\r\t\\\"ing":9}` {
		t.Error("JSON Marshal value is incorrect", s)
	}
	// convert to indented json
	bi, err := json.MarshalIndent(o, "", "  ")
	if err != nil {
		t.Error("Marshalling indented json", err)
	}
	si := string(bi)
	ei := `{
  "number": 4,
  "string": "x",
  "specialstring": "\\.\u003c\u003e[]{}_-",
  "z": 1,
  "a": 2,
  "b": 3,
  "slice": [
    "1",
    1
  ],
  "orderedmap": {
    "e": 1,
    "a": 2
  },
  "test\n\r\t\\\"ing": 9
}`
	if si != ei {
		fmt.Println(ei)
		fmt.Println(si)
		t.Error("JSON MarshalIndent value is incorrect", si)
	}
}

func TestMarshalJSONNoEscapeHTML(t *testing.T) {
	o := New()
	// string special characters
	o.Set("specialstring", "\\.<>[]{}_-")
	// convert to json
	b, err := o.MarshalJSON()
	if err != nil {
		t.Error("Marshalling json", err)
	}
	s := string(b)
	// check json is correctly ordered
	if s != `{"specialstring":"\\.<>[]{}_-"}` {
		t.Error("JSON Marshal value is incorrect", s)
	}
}

func TestMarshalJSONNoEscapeHTMLRecursive(t *testing.T) {
	src := `{"x":"<>","y":[{"z":["<>"]}]}`
	o := New()
	o.SetEscapeHTML(false)
	err := json.Unmarshal([]byte(src), &o)
	if err != nil {
		t.Error("JSON Unmarshal error with special chars", err)
	}
	b, err := o.MarshalJSON()
	if err != nil {
		t.Error("Marshalling json", err)
	}
	s := string(b)
	if s != src {
		t.Error("JSON Marshal value is incorrect", s)
	}
}

func TestMarshalJSONEscapeHTML(t *testing.T) {
	o := New()
	o.Set("chars", "<>&")
	b, err := o.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if s != `{"chars":"<>&"}` {
		t.Errorf("default MarshalJSON HTML-escaped: %s", s)
	}

	o.SetEscapeHTML(true)
	b, err = o.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	s = string(b)
	if s != `{"chars":"\u003c\u003e\u0026"}` {
		t.Errorf("SetEscapeHTML(true) did not escape: %s", s)
	}
}

func TestMarshalJSONNoNewlines(t *testing.T) {
	o := New()
	o.Set("a", 1)
	o.Set("b", "x\ny")
	b, err := o.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "\n") {
		t.Errorf("MarshalJSON left newlines: %q", b)
	}
	if string(b) != `{"a":1,"b":"x\ny"}` {
		t.Errorf("MarshalJSON = %s", b)
	}
}

func TestUnmarshalJSON(t *testing.T) {
	s := `{
  "number": 4,
  "string": "x",
  "z": 1,
  "a": "should not break with unclosed { character in value",
  "b": 3,
  "slice": [
    "1",
    1
  ],
  "orderedmap": {
    "e": 1,
    "a { nested key with brace": "with a }}}} }} {{{ brace value",
	"after": {
		"link": "test {{{ with even deeper nested braces }"
	}
  },
  "test\"ing": 9,
  "after": 1,
  "multitype_array": [
    "test",
	1,
	{ "map": "obj", "it" : 5, ":colon in key": "colon: in value" },
	[{"inner": "map"}]
  ],
  "should not break with { character in key": 1
}`
	o := New()
	err := json.Unmarshal([]byte(s), &o)
	if err != nil {
		t.Error("JSON Unmarshal error", err)
	}
	// Check the root keys
	expectedKeys := []string{
		"number",
		"string",
		"z",
		"a",
		"b",
		"slice",
		"orderedmap",
		"test\"ing",
		"after",
		"multitype_array",
		"should not break with { character in key",
	}
	k := o.Keys()
	for i := range k {
		if k[i] != expectedKeys[i] {
			t.Error("Unmarshal root key order", i, k[i], "!=", expectedKeys[i])
		}
	}
	// Check nested maps are converted to orderedmaps
	// nested 1 level deep
	expectedKeys = []string{
		"e",
		"a { nested key with brace",
		"after",
	}
	vi, ok := o.Get("orderedmap")
	if !ok {
		t.Error("Missing key for nested map 1 deep")
	}
	v := vi.(OrderedMap)
	k = v.Keys()
	for i := range k {
		if k[i] != expectedKeys[i] {
			t.Error("Key order for nested map 1 deep ", i, k[i], "!=", expectedKeys[i])
		}
	}
	// nested 2 levels deep
	expectedKeys = []string{
		"link",
	}
	vi, ok = v.Get("after")
	if !ok {
		t.Error("Missing key for nested map 2 deep")
	}
	v = vi.(OrderedMap)
	k = v.Keys()
	for i := range k {
		if k[i] != expectedKeys[i] {
			t.Error("Key order for nested map 2 deep", i, k[i], "!=", expectedKeys[i])
		}
	}
	// multitype array
	expectedKeys = []string{
		"map",
		"it",
		":colon in key",
	}
	vislice, ok := o.Get("multitype_array")
	if !ok {
		t.Error("Missing key for multitype array")
	}
	vslice := vislice.([]any)
	vmap := vslice[2].(OrderedMap)
	k = vmap.Keys()
	for i := range k {
		if k[i] != expectedKeys[i] {
			t.Error("Key order for nested map 2 deep", i, k[i], "!=", expectedKeys[i])
		}
	}
	// nested map 3 deep
	vislice, _ = o.Get("multitype_array")
	vslice = vislice.([]any)
	expectedKeys = []string{"inner"}
	vinnerslice := vslice[3].([]any)
	vinnermap := vinnerslice[0].(OrderedMap)
	k = vinnermap.Keys()
	for i := range k {
		if k[i] != expectedKeys[i] {
			t.Error("Key order for nested map 3 deep", i, k[i], "!=", expectedKeys[i])
		}
	}
}

func TestUnmarshalJSONNull(t *testing.T) {
	// json.Unmarshal into a *OrderedMap (not **OrderedMap). Passing &o would
	// set o to nil without calling UnmarshalJSON.
	o := New()
	o.Set("a", 1)
	if err := json.Unmarshal([]byte("null"), o); err != nil {
		t.Fatal(err)
	}
	if len(o.Keys()) != 0 {
		t.Fatalf("keys after null: %v", o.Keys())
	}
	if vals := o.Values(); len(vals) != 0 {
		t.Fatalf("values after null: %v", vals)
	}

	o = New()
	o.Set("a", 1)
	if err := json.Unmarshal([]byte(" \nnull\t"), o); err != nil {
		t.Fatal(err)
	}
	if len(o.Keys()) != 0 {
		t.Fatalf("keys after padded null: %v", o.Keys())
	}

	// Options are preserved; only the map contents are cleared.
	o = New()
	o.SetEscapeHTML(true)
	o.SetJSONLastValueWins(true)
	o.Set("a", 1)
	if err := o.UnmarshalJSON([]byte(" null ")); err != nil {
		t.Fatal(err)
	}
	if len(o.Keys()) != 0 {
		t.Fatalf("keys after spaced null: %v", o.Keys())
	}
	b, err := o.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `{}` {
		t.Fatalf("marshal empty after null: %q", b)
	}
}

func TestUnmarshalJSONDuplicateKeys(t *testing.T) {
	s := `{
		"a": [{}, []],
		"b": {"x":[1]},
		"c": "x",
		"d": {"x":1},
		"b": [{"x":[]}],
		"c": 1,
		"d": {"y": 2},
		"e": [{"x":1}],
		"e": [[]],
		"e": [{"z":2}],
		"a": {},
		"b": [[1]]
	}`
	o := New()
	o.SetJSONLastValueWins(true)
	err := json.Unmarshal([]byte(s), &o)
	if err != nil {
		t.Error("JSON Unmarshal error with special chars", err)
	}
	expectedKeys := []string{
		"c",
		"d",
		"e",
		"a",
		"b",
	}
	keys := o.Keys()
	if len(keys) != len(expectedKeys) {
		t.Error("Unmarshal key count", len(keys), "!=", len(expectedKeys))
	}
	for i, key := range keys {
		if key != expectedKeys[i] {
			t.Errorf("Unmarshal root key order: %d, %q != %q", i, key, expectedKeys[i])
		}
	}
	vimap, _ := o.Get("a")
	_ = vimap.(OrderedMap)
	vislice, _ := o.Get("b")
	_ = vislice.([]any)
	vival, _ := o.Get("c")
	_ = vival.(float64)

	vimap, _ = o.Get("d")
	m := vimap.(OrderedMap)
	expectedKeys = []string{"y"}
	keys = m.Keys()
	if len(keys) != len(expectedKeys) {
		t.Error("Unmarshal key count", len(keys), "!=", len(expectedKeys))
	}
	for i, key := range keys {
		if key != expectedKeys[i] {
			t.Errorf("Unmarshal key order: %d, %q != %q", i, key, expectedKeys[i])
		}
	}

	vislice, _ = o.Get("e")
	m = vislice.([]any)[0].(OrderedMap)
	expectedKeys = []string{"z"}
	keys = m.Keys()
	if len(keys) != len(expectedKeys) {
		t.Error("Unmarshal key count", len(keys), "!=", len(expectedKeys))
	}
	for i, key := range keys {
		if key != expectedKeys[i] {
			t.Errorf("Unmarshal key order: %d, %q != %q", i, key, expectedKeys[i])
		}
	}
}

// TestOrderedMapUnmarshalJSONDuplicate tests that orderedMap errors on
// duplicate JSON fields.
func TestOrderedMapUnmarshalJSONDuplicate(t *testing.T) {
	s := `{
		"a": [{}, []],
		"b": {"x":[1]},
		"c": "x",
		"d": {"x":1},
		"b": [{"x":[]}]
	}`
	o := New()
	err := json.Unmarshal([]byte(s), &o)
	if err == nil {
		t.Errorf("orderedMap unmarshal did not error on duplicate key")
	} else if !errors.Is(err, ErrJSONDuplicate) {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestUnmarshalJSONSpecialChars(t *testing.T) {
	s := `{ " \u0041\n\r\t\\\\\\\\\\\\ "  : { "\\\\\\" : "\\\\\"\\" }, "\\":  " \\\\ test ", "\n": "\r" }`
	o := New()
	err := json.Unmarshal([]byte(s), &o)
	if err != nil {
		t.Error("JSON Unmarshal error with special chars", err)
	}
	expectedKeys := []string{
		" \u0041\n\r\t\\\\\\\\\\\\ ",
		"\\",
		"\n",
	}
	keys := o.Keys()
	if len(keys) != len(expectedKeys) {
		t.Error("Unmarshal key count", len(keys), "!=", len(expectedKeys))
	}
	for i, key := range keys {
		if key != expectedKeys[i] {
			t.Errorf("Unmarshal root key order: %d, %q != %q", i, key, expectedKeys[i])
		}
	}
}

func TestUnmarshalJSONArrayOfMaps(t *testing.T) {
	s := `
{
  "name": "test",
  "percent": 6,
  "breakdown": [
    {
      "name": "a",
      "percent": 0.9
    },
    {
      "name": "b",
      "percent": 0.9
    },
    {
      "name": "d",
      "percent": 0.4
    },
    {
      "name": "e",
      "percent": 2.7
    }
  ]
}
`
	o := New()
	err := json.Unmarshal([]byte(s), &o)
	if err != nil {
		t.Error("JSON Unmarshal error", err)
	}
	// Check the root keys
	expectedKeys := []string{
		"name",
		"percent",
		"breakdown",
	}
	k := o.Keys()
	for i := range k {
		if k[i] != expectedKeys[i] {
			t.Error("Unmarshal root key order", i, k[i], "!=", expectedKeys[i])
		}
	}
	// Check nested maps are converted to orderedmaps
	// nested 1 level deep
	expectedKeys = []string{
		"name",
		"percent",
	}
	vi, ok := o.Get("breakdown")
	if !ok {
		t.Error("Missing key for nested map 1 deep")
	}
	vs := vi.([]any)
	for _, vInterface := range vs {
		v := vInterface.(OrderedMap)
		k = v.Keys()
		for i := range k {
			if k[i] != expectedKeys[i] {
				t.Error("Key order for nested map 1 deep ", i, k[i], "!=", expectedKeys[i])
			}
		}
	}
}

func TestUnmarshalJSONStruct(t *testing.T) {
	var v struct {
		Data *OrderedMap `json:"data"`
	}

	err := json.Unmarshal([]byte(`{ "data": { "x": 1 } }`), &v)
	if err != nil {
		t.Fatalf("JSON unmarshal error: %v", err)
	}

	x, ok := v.Data.Get("x")
	if !ok {
		t.Errorf("missing expected key")
	} else if x != float64(1) {
		t.Errorf("unexpected value: %#v", x)
	}
}

func TestOrderedMap_SortKeys(t *testing.T) {
	s := `
{
  "b": 2,
  "a": 1,
  "c": 3
}
`
	o := New()
	json.Unmarshal([]byte(s), &o)

	o.SortKeys(sort.Strings)

	// Check the root keys
	expectedKeys := []string{
		"a",
		"b",
		"c",
	}
	k := o.Keys()
	for i := range k {
		if k[i] != expectedKeys[i] {
			t.Error("SortKeys root key order", i, k[i], "!=", expectedKeys[i])
		}
	}
}

func TestOrderedMap_Sort(t *testing.T) {
	s := `
{
  "b": 2,
  "a": 1,
  "c": 3
}
`
	o := New()
	json.Unmarshal([]byte(s), &o)
	o.Sort(func(a *Pair, b *Pair) bool {
		return a.value.(float64) > b.value.(float64)
	})

	// Check the root keys
	expectedKeys := []string{
		"c",
		"b",
		"a",
	}
	k := o.Keys()
	for i := range k {
		if k[i] != expectedKeys[i] {
			t.Error("Sort root key order", i, k[i], "!=", expectedKeys[i])
		}
	}
}

// https://github.com/iancoleman/orderedmap/issues/11
func TestOrderedMap_empty_array(t *testing.T) {
	srcStr := `{"x":[]}`
	src := []byte(srcStr)
	om := New()
	json.Unmarshal(src, om)
	bs, _ := json.Marshal(om)
	marshalledStr := string(bs)
	if marshalledStr != srcStr {
		t.Error("Empty array does not serialise to json correctly")
		t.Error("Expect", srcStr)
		t.Error("Got", marshalledStr)
	}
}

// Inspired by
// https://github.com/iancoleman/orderedmap/issues/11
// but using empty maps instead of empty slices
func TestOrderedMap_empty_map(t *testing.T) {
	srcStr := `{"x":{}}`
	src := []byte(srcStr)
	om := New()
	json.Unmarshal(src, om)
	bs, _ := json.Marshal(om)
	marshalledStr := string(bs)
	if marshalledStr != srcStr {
		t.Error("Empty map does not serialise to json correctly")
		t.Error("Expect", srcStr)
		t.Error("Got", marshalledStr)
	}
}

// TestUnmarshalJSONNestedDuplicate tests that duplicate keys in nested objects,
// including objects inside arrays, cause an error unless last-value-wins is on.
func TestUnmarshalJSONNestedDuplicate(t *testing.T) {
	cases := []string{
		`{"a":{"x":1,"x":2}}`,
		`{"a":[{"x":1},{"y":1,"y":2}]}`,
		`{"a":[[{"x":{"z":1,"z":2}}]]}`,
	}
	for _, s := range cases {
		o := New()
		err := json.Unmarshal([]byte(s), &o)
		if err == nil {
			t.Errorf("no error on nested duplicate key: %s", s)
		} else if !errors.Is(err, ErrJSONDuplicate) {
			t.Errorf("unexpected error for %s: %v", s, err)
		}

		o = New()
		o.SetJSONLastValueWins(true)
		if err := json.Unmarshal([]byte(s), &o); err != nil {
			t.Errorf("unexpected error with last value wins for %s: %v", s, err)
		}
	}
}

func TestCheckDuplicate(t *testing.T) {
	// Happy path; no duplicate. Returns top-level keys in order, not nested keys.
	data := `{"a": "x", "b":"y", "c": {"z": 1, "zz": 2}}`
	keys, err := CheckDuplicate(json.NewDecoder(strings.NewReader(data)))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(keys, []string{"a", "b", "c"}) {
		t.Fatalf("Top level keys returned incorrectly. Got: %v", keys)
	}

	// Scalar and array roots have no object keys.
	keys, err = CheckDuplicate(json.NewDecoder(strings.NewReader(`"x"`)))
	if err != nil || keys != nil {
		t.Fatalf("scalar: keys=%v err=%v", keys, err)
	}
	keys, err = CheckDuplicate(json.NewDecoder(strings.NewReader(`[{"a":1}]`)))
	if err != nil || keys != nil {
		t.Fatalf("array: keys=%v err=%v", keys, err)
	}

	// Duplicate, should error.
	data = `{"a": "aValue", "a":true,"c":["field_3 string 1","field3 string2"], "d": {"z": 1, "zz": 2}}`
	_, err = CheckDuplicate(json.NewDecoder(strings.NewReader(data)))
	if err == nil {
		t.Fatal("Should have found duplicate.")
	}

	// Recursive check with duplicate in inner struct. Should error and return no keys.
	data = `{"a": "aValue", "b":"bValue", "c": {"z": 1, "z": 2}}`
	keys, err = CheckDuplicate(json.NewDecoder(strings.NewReader(data)))
	if err == nil {
		t.Fatal("Recursive check should have found duplicate.")
	}
	if keys != nil {
		t.Fatalf("duplicate should return no keys, got %v", keys)
	}
}

func ExampleCheckDuplicate() {
	data := `{"a": "x", "b":"y", "c": {"z": 1, "zz": 2}}`
	keys, err := CheckDuplicate(json.NewDecoder(strings.NewReader(data)))
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(keys)

	// Output:
	// [a b c]
}

func TestErrJSONDuplicate(t *testing.T) {
	plain := fmt.Errorf("orderedmap: JSON duplicate field %q", "a")
	if errors.Is(plain, ErrJSONDuplicate) {
		t.Fatal("plain fmt.Errorf should not match ErrJSONDuplicate")
	}
	var de *DuplicateFieldError
	if errors.As(plain, &de) {
		t.Fatal("plain fmt.Errorf should not match DuplicateFieldError")
	}

	_, err := CheckDuplicate(json.NewDecoder(strings.NewReader(`{"a":1,"a":2}`)))
	if err == nil {
		t.Fatal("expected duplicate error")
	}
	if !errors.Is(err, ErrJSONDuplicate) {
		t.Fatalf("errors.Is failed: %v", err)
	}
	if !errors.As(err, &de) {
		t.Fatalf("errors.As failed: %v", err)
	}
	if de.Field != "a" {
		t.Fatalf("Field = %q, want a", de.Field)
	}
	if err.Error() != `orderedmap: JSON duplicate field "a"` {
		t.Fatalf("Error() = %q", err.Error())
	}
}
