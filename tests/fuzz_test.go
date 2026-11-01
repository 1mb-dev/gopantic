package tests

import (
	"bytes"
	"encoding/json"
	"math"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/1mb-dev/gopantic/pkg/model"
)

// FuzzStruct is a struct used for fuzzing ParseInto
type FuzzStruct struct {
	ID        int       `json:"id" yaml:"id"`
	Name      string    `json:"name" yaml:"name"`
	Active    bool      `json:"active" yaml:"active"`
	Score     float64   `json:"score" yaml:"score"`
	Tags      []string  `json:"tags,omitempty" yaml:"tags,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty" yaml:"created_at,omitempty"`
}

// FuzzNestedStruct includes nested structures for comprehensive testing
type FuzzNestedStruct struct {
	User   FuzzStruct            `json:"user" yaml:"user"`
	Data   map[string]any        `json:"data,omitempty" yaml:"data,omitempty"`
	Values []int                 `json:"values,omitempty" yaml:"values,omitempty"`
	Meta   map[string]FuzzStruct `json:"meta,omitempty" yaml:"meta,omitempty"`
}

// FuzzParseInto tests the main ParseInto entry point
// This exercises the full parsing pipeline including:
// - Format detection (JSON vs YAML)
// - Parsing and unmarshaling
// - Type coercion
// - Validation
func FuzzParseInto(f *testing.F) {
	// Seed with valid JSON examples
	f.Add([]byte(`{"id": 1, "name": "test"}`))
	f.Add([]byte(`{"id": 42, "name": "user", "active": true, "score": 98.5}`))
	f.Add([]byte(`{"id": 100, "name": "with tags", "tags": ["a", "b", "c"]}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`{"id": "123", "name": 456}`)) // Type coercion case

	// Seed with valid YAML examples
	f.Add([]byte("id: 1\nname: test"))
	f.Add([]byte("id: 42\nname: user\nactive: true\nscore: 98.5"))
	f.Add([]byte("id: 100\nname: with tags\ntags:\n  - a\n  - b"))
	f.Add([]byte("---\nid: 1"))

	// Edge cases
	f.Add([]byte(`null`))
	f.Add([]byte(`[]`))
	f.Add([]byte(``))
	f.Add([]byte(`   `))
	f.Add([]byte(`{"id": 9223372036854775807}`))        // Max int64
	f.Add([]byte(`{"id": -9223372036854775808}`))       // Min int64
	f.Add([]byte(`{"score": 1.7976931348623157e+308}`)) // Max float64

	f.Fuzz(func(t *testing.T, data []byte) {
		// Must not panic - any panic is a bug
		_, _ = model.ParseInto[FuzzStruct](data)
	})
}

// FuzzParseIntoNested tests parsing with nested structures
func FuzzParseIntoNested(f *testing.F) {
	// Seed with nested JSON
	f.Add([]byte(`{"user": {"id": 1, "name": "test"}}`))
	f.Add([]byte(`{"user": {"id": 1, "name": "test"}, "data": {"key": "value"}}`))
	f.Add([]byte(`{"user": {"id": 1, "name": "test"}, "values": [1, 2, 3]}`))

	// Seed with nested YAML
	f.Add([]byte("user:\n  id: 1\n  name: test"))
	f.Add([]byte("user:\n  id: 1\n  name: test\ndata:\n  key: value"))

	f.Fuzz(func(t *testing.T, data []byte) {
		// Must not panic
		_, _ = model.ParseInto[FuzzNestedStruct](data)
	})
}

// FuzzParseIntoMap tests parsing into map types
func FuzzParseIntoMap(f *testing.F) {
	f.Add([]byte(`{"key": "value"}`))
	f.Add([]byte(`{"a": 1, "b": 2, "c": 3}`))
	f.Add([]byte(`{"nested": {"deep": {"value": 123}}}`))
	f.Add([]byte("key: value"))
	f.Add([]byte("a: 1\nb: 2"))

	f.Fuzz(func(t *testing.T, data []byte) {
		// Must not panic
		_, _ = model.ParseInto[map[string]any](data)
	})
}

// FuzzParseIntoSlice tests parsing into slice types
func FuzzParseIntoSlice(f *testing.F) {
	f.Add([]byte(`[1, 2, 3]`))
	f.Add([]byte(`["a", "b", "c"]`))
	f.Add([]byte(`[{"id": 1}, {"id": 2}]`))
	f.Add([]byte(`[]`))
	f.Add([]byte("- 1\n- 2\n- 3"))
	f.Add([]byte("- id: 1\n- id: 2"))

	f.Fuzz(func(t *testing.T, data []byte) {
		// Must not panic
		_, _ = model.ParseInto[[]FuzzStruct](data)
	})
}

// FuzzCoerceValue tests the type coercion logic
// This is where complex type conversion happens
func FuzzCoerceValue(f *testing.F) {
	// Seed with various input types that might be coerced

	// Strings that should coerce to numbers
	f.Add([]byte("123"))
	f.Add([]byte("-456"))
	f.Add([]byte("3.14159"))
	f.Add([]byte("1e10"))
	f.Add([]byte("0"))

	// Strings that should coerce to booleans
	f.Add([]byte("true"))
	f.Add([]byte("false"))
	f.Add([]byte("1"))
	f.Add([]byte("0"))
	f.Add([]byte("yes"))
	f.Add([]byte("no"))

	// Edge case strings
	f.Add([]byte(""))
	f.Add([]byte("   "))
	f.Add([]byte("null"))
	f.Add([]byte("NaN"))
	f.Add([]byte("Infinity"))
	f.Add([]byte("-Infinity"))

	// Unicode
	f.Add([]byte("你好"))
	f.Add([]byte("🎉"))
	f.Add([]byte("\u0000"))

	// Large numbers
	f.Add([]byte("999999999999999999999999999999"))
	f.Add([]byte("-999999999999999999999999999999"))

	f.Fuzz(func(t *testing.T, data []byte) {
		value := string(data)

		// Test coercion to various types - must not panic
		targetTypes := []reflect.Type{
			reflect.TypeOf(""),
			reflect.TypeOf(0),
			reflect.TypeOf(int64(0)),
			reflect.TypeOf(uint(0)),
			reflect.TypeOf(uint64(0)),
			reflect.TypeOf(float64(0)),
			reflect.TypeOf(false),
			reflect.TypeOf(time.Time{}),
		}

		for _, targetType := range targetTypes {
			// Must not panic - errors are acceptable
			_, _ = model.CoerceValue(value, targetType, "fuzz_field")
		}
	})
}

// FuzzCoerceNumeric specifically tests numeric coercion edge cases
func FuzzCoerceNumeric(f *testing.F) {
	// Add numeric seeds
	f.Add(float64(0))
	f.Add(float64(1))
	f.Add(float64(-1))
	f.Add(float64(3.14159))
	f.Add(float64(1e10))
	f.Add(float64(1e-10))
	f.Add(float64(1.7976931348623157e+308))  // Max float64
	f.Add(float64(-1.7976931348623157e+308)) // Min float64
	f.Add(float64(5e-324))                   // Smallest positive float64

	f.Fuzz(func(t *testing.T, value float64) {
		targetTypes := []reflect.Type{
			reflect.TypeOf(int(0)),
			reflect.TypeOf(int8(0)),
			reflect.TypeOf(int16(0)),
			reflect.TypeOf(int32(0)),
			reflect.TypeOf(int64(0)),
			reflect.TypeOf(uint(0)),
			reflect.TypeOf(uint8(0)),
			reflect.TypeOf(uint16(0)),
			reflect.TypeOf(uint32(0)),
			reflect.TypeOf(uint64(0)),
			reflect.TypeOf(float32(0)),
			reflect.TypeOf(float64(0)),
			reflect.TypeOf(""),
		}

		for _, targetType := range targetTypes {
			// Must not panic - errors are acceptable
			_, _ = model.CoerceValue(value, targetType, "fuzz_field")
		}
	})
}

// FuzzCoerceSlice tests slice coercion
func FuzzCoerceSlice(f *testing.F) {
	f.Add([]byte(`[1, 2, 3]`))
	f.Add([]byte(`["a", "b"]`))
	f.Add([]byte(`[true, false]`))
	f.Add([]byte(`[]`))

	f.Fuzz(func(t *testing.T, data []byte) {
		// Parse the data first to get a slice
		parsed, err := model.ParseInto[[]any](data)
		if err != nil {
			return // Can't proceed without valid slice
		}

		// Test coercing the slice to various slice types
		sliceTypes := []reflect.Type{
			reflect.TypeOf([]int{}),
			reflect.TypeOf([]string{}),
			reflect.TypeOf([]float64{}),
			reflect.TypeOf([]bool{}),
		}

		for _, sliceType := range sliceTypes {
			// Must not panic
			_, _ = model.CoerceValue(parsed, sliceType, "fuzz_slice")
		}
	})
}

// FuzzNumericRoundTrip holds the numeric contract: a parse that returns no error returned the
// number it was given.
func FuzzNumericRoundTrip(f *testing.F) {
	for _, seed := range []float64{0, 1, -1, 4.7, 127, 128, -129, 255, 256, 300, 1e3, 1e16, 1e30, 1.79e308, 2147483648, 9007199254740992} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, n float64) {
		if math.IsNaN(n) || math.IsInf(n, 0) {
			t.Skip("not representable in JSON")
		}
		literal := strconv.FormatFloat(n, 'g', -1, 64)
		for _, input := range [][]byte{
			[]byte(`{"v":` + literal + `}`),
			[]byte(`{"v":"` + literal + `"}`),
		} {
			checkRoundTrip[ctI8](t, input, n, func(v ctI8) float64 { return float64(v.V) })
			checkRoundTrip[ctU8](t, input, n, func(v ctU8) float64 { return float64(v.V) })
			checkRoundTrip[ctInt](t, input, n, func(v ctInt) float64 { return float64(v.V) })
			checkRoundTrip[ctUint](t, input, n, func(v ctUint) float64 { return float64(v.V) })
		}
		checkRoundTrip[ctI8Slice](t, []byte(`{"v":[`+literal+`]}`), n, func(v ctI8Slice) float64 {
			if len(v.V) != 1 {
				return math.NaN()
			}
			return float64(v.V[0])
		})
	})
}

func checkRoundTrip[T any](t *testing.T, input []byte, want float64, value func(T) float64) {
	t.Helper()
	got, err := model.ParseInto[T](input)
	if err != nil {
		return
	}
	if value(got) != want {
		t.Errorf("%s into %T: got %v, want %v", input, got, value(got), want)
	}
}

// fuzzZoo carries every shape the walker treats specially.
type fuzzZoo struct {
	Trigger int `json:"trigger"`
	CtPromoted
	Name  string            `json:"name"`
	Num   int64             `json:"num"`
	Flt   float64           `json:"flt"`
	Ok    bool              `json:"ok"`
	Tags  []string          `json:"tags"`
	Pair  [2]int            `json:"pair"`
	Attrs map[string]int    `json:"attrs"`
	ByID  map[int]string    `json:"by_id"`
	Ptr   *int              `json:"ptr"`
	Any   any               `json:"any"`
	Raw   json.RawMessage   `json:"raw"`
	Data  []byte            `json:"data"`
	Kids  []fuzzZoo         `json:"kids"`
	Next  *fuzzZoo          `json:"next"`
	Named map[string]CtItem `json:"named"`
}

// CtItem is a nested type with no rules, so a differential compares decoding alone.
type CtItem struct {
	Name string `json:"name"`
}

// FuzzWalkerDifferential holds the coercion contract: when encoding/json decodes an object,
// the same object with one field needing coercion decodes to the same value.
func FuzzWalkerDifferential(f *testing.F) {
	for _, seed := range []string{
		`"name":"a"`, `"num":9007199254740993`, `"flt":1.5`, `"ok":true`, `"tags":["a"]`,
		`"pair":[1,2]`, `"attrs":{"a":1}`, `"by_id":{"1":"a"}`, `"ptr":3`, `"any":{"x":[1,null]}`,
		`"raw":{"b": 1, "a":[ 2 ]}`, `"data":"aGk="`, `"kids":[{"name":"k"}]`, `"next":{"num":1}`,
		`"promoted":"p"`, `"NAME":"n"`, `"named":{"a":{"name":"x"}}`, `"name":null,"tags":null`,
		`"name":"a","name":"b"`, `"next":{"name":"a"},"next":{"num":1}`,
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, fields string) {
		var want fuzzZoo
		typed := []byte(`{"trigger":1,` + fields + `}`)
		if err := json.Unmarshal(typed, &want); err != nil {
			t.Skip()
		}

		coerced := bytes.Replace(typed, []byte(`"trigger":1`), []byte(`"trigger":"1"`), 1)
		got, err := model.ParseInto[fuzzZoo](coerced)
		if err != nil {
			t.Fatalf("encoding/json accepted %s, walker rejected it: %v", typed, err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("%s\ngot  %+v\nwant %+v", typed, got, want)
		}
	})
}

// FuzzNoPanicAnyTarget checks that no input panics for any kind of target type.
func FuzzNoPanicAnyTarget(f *testing.F) {
	for _, seed := range []string{
		`{"a":"x"}`, `{"a":"1"}`, `[null]`, `[{"a":null}]`, `{"v":"1"}`, `null`, `1`, `"s"`,
		"a: 1\nb: 2\n", "- 1\n- x\n", "v: \"1\"\nw: [1]\n", "{1: 2, x: y}\n",
	} {
		f.Add([]byte(seed))
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		for _, format := range []model.Format{model.FormatJSON, model.FormatYAML} {
			_, _ = model.ParseIntoWithFormat[map[string]int](data, format)
			_, _ = model.ParseIntoWithFormat[map[int]string](data, format)
			_, _ = model.ParseIntoWithFormat[*ctInt](data, format)
			_, _ = model.ParseIntoWithFormat[[]map[string]string](data, format)
			_, _ = model.ParseIntoWithFormat[[2]any](data, format)
			_, _ = model.ParseIntoWithFormat[any](data, format)
			_, _ = model.ParseIntoWithFormat[int](data, format)
			_, _ = model.ParseIntoWithFormat[fuzzZoo](data, format)
		}
	})
}
