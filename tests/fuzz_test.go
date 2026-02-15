package tests

import (
	"reflect"
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
