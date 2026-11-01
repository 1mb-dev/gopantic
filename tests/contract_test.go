package tests

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/1mb-dev/gopantic/pkg/model"
	"gopkg.in/yaml.v3"
)

// Contract types use unique names: validation metadata is cached per type for the life of
// the process, so a shared type would let one row poison another.

type ctItem struct {
	Name string `json:"name" yaml:"name" validate:"required"`
}

type ctPathOuter struct {
	Item ctItem `json:"item"`
}

type ctOuterSecond struct {
	Item ctItem `json:"item"`
}

type ctPtrOuter struct {
	Item *ctItem `json:"item"`
}

type ctSliceOuter struct {
	Trigger int      `json:"trigger"`
	Items   []ctItem `json:"items"`
}

type ctArrayOuter struct {
	Trigger int       `json:"trigger"`
	Items   [1]ctItem `json:"items"`
}

type ctMapOuter struct {
	Trigger int               `json:"trigger"`
	Items   map[string]ctItem `json:"items"`
}

type ctScalars struct {
	Trigger int     `json:"trigger"`
	B       bool    `json:"b"`
	I8      int8    `json:"i8"`
	I16     int16   `json:"i16"`
	I32     int32   `json:"i32"`
	I64     int64   `json:"i64"`
	U       uint    `json:"u"`
	U8      uint8   `json:"u8"`
	U16     uint16  `json:"u16"`
	U32     uint32  `json:"u32"`
	U64     uint64  `json:"u64"`
	F32     float32 `json:"f32"`
	F64     float64 `json:"f64"`
	S       string  `json:"s"`
}

type ctContainers struct {
	Trigger int               `json:"trigger"`
	Slice   []string          `json:"slice"`
	Array   [2]int            `json:"array"`
	Map     map[string]string `json:"map"`
	Ptr     *int              `json:"ptr"`
	Any     any               `json:"any"`
	Nested  struct {
		Value int `json:"value"`
	} `json:"nested"`
}

type ctBase struct {
	Base int `json:"base"`
}

type ctEmbedded struct {
	ctBase
	Trigger int `json:"trigger"`
}

type ctExportedBase struct {
	Base int `json:"base"`
}

type ctEmbeddedExported struct {
	ctExportedBase
	CtPromoted
	Trigger int `json:"trigger"`
}

// CtPromoted is exported so encoding/json promotes its fields through embedding.
type CtPromoted struct {
	Promoted string `json:"promoted"`
}

type ctEmptyName struct {
	Trigger int    `json:"trigger"`
	Name    string `json:",omitempty"`
}

type ctUnmarshaler struct {
	Trigger int    `json:"trigger"`
	IP      net.IP `json:"ip"`
}

type ctRaw struct {
	Trigger int             `json:"trigger"`
	Meta    json.RawMessage `json:"meta"`
}

type ctCase struct {
	Trigger int    `json:"trigger"`
	Name    string `json:"name"`
}

type ctKelvin struct {
	Trigger int    `json:"trigger"`
	Kind    string `json:"kind"`
}

type ctTime struct {
	Trigger int       `json:"trigger"`
	At      time.Time `json:"at"`
}

type ctDeepTrigger struct {
	Nested struct {
		Trigger int    `json:"trigger"`
		Name    string `json:"name"`
	} `json:"nested"`
	Map map[string]int `json:"map"`
}

type ctSliceTrigger struct {
	Items []struct {
		Trigger int `json:"trigger"`
	} `json:"items"`
	Map map[string]int `json:"map"`
}

type ctI8 struct {
	V int8 `json:"v" yaml:"v"`
}

type ctU8 struct {
	V uint8 `json:"v" yaml:"v"`
}

type ctInt struct {
	V int `json:"v" yaml:"v"`
}

type ctUint struct {
	V uint `json:"v" yaml:"v"`
}

type ctFloat32 struct {
	V float32 `json:"v" yaml:"v"`
}

type ctFloat64 struct {
	V float64 `json:"v" yaml:"v"`
}

type ctI8Slice struct {
	V []int8 `json:"v"`
}

type ctI8Array struct {
	V [1]int8 `json:"v"`
}

type ctI8Ptr struct {
	V *int8 `json:"v"`
}

type ctBounded struct {
	V float64 `json:"v" validate:"required,min=0,max=100"`
}

type ctStringField struct {
	V string `json:"v"`
}

type ctBig struct {
	Trigger int   `json:"trigger"`
	Big     int64 `json:"big"`
}

type ctNullElem struct {
	Trigger int                 `json:"trigger"`
	Items   []map[string]string `json:"items"`
}

type ctCross struct {
	A string `json:"a" validate:"ct_same"`
	B string `json:"b"`
}

type ctCrossOuter struct {
	Trigger int     `json:"trigger"`
	Cross   ctCross `json:"cross"`
}

type ctYAMLJSONTag struct {
	UserName string `json:"user_name"`
}

type ctYAMLItem struct {
	Full string `yaml:"full_name"`
}

type ctYAMLList struct {
	Trigger int          `yaml:"trigger"`
	Items   []ctYAMLItem `yaml:"items"`
}

type ctTypo struct {
	A string `json:"a" validate:"requird,emial"`
}

type ctLen struct {
	A string `json:"a" validate:"len=8"`
}

type ctBadMin struct {
	A string `json:"a" validate:"min=abc"`
}

type ctBareMin struct {
	A string `json:"a" validate:"min"`
}

type ctTypoItem struct {
	A string `json:"a" validate:"requird"`
}

type ctTypoInSlice struct {
	Items []ctTypoItem `json:"items"`
}

type ctLate struct {
	A string `json:"a" validate:"ct_late"`
}

type ctSensitive struct {
	Pin int `json:"pin"`
}

type ctTagged struct {
	Trigger int      `json:"trigger"`
	Tags    []string `json:"tags"`
}

type ctThreeFaults struct {
	V int `json:"v" validate:"required,min=1"`
}

// contractZoo is what TestContractKinds measures. Add a type here when adding a row for a
// new shape.
var contractZoo = []reflect.Type{
	reflect.TypeFor[ctScalars](),
	reflect.TypeFor[ctContainers](),
	reflect.TypeFor[ctEmbeddedExported](),
	reflect.TypeFor[ctUnmarshaler](),
	reflect.TypeFor[ctRaw](),
	reflect.TypeFor[ctTime](),
	reflect.TypeFor[ctDeepTrigger](),
	reflect.TypeFor[ctSliceTrigger](),
	reflect.TypeFor[ctMapOuter](),
	reflect.TypeFor[ctArrayOuter](),
	reflect.TypeFor[ctPtrOuter](),
}

func TestContractKinds(t *testing.T) {
	supported := []reflect.Kind{
		reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64,
		reflect.String, reflect.Slice, reflect.Array, reflect.Map, reflect.Pointer,
		reflect.Struct, reflect.Interface,
	}

	seen := map[reflect.Kind]bool{}
	visited := map[reflect.Type]bool{}
	var walk func(reflect.Type)
	walk = func(typ reflect.Type) {
		if visited[typ] {
			return
		}
		visited[typ] = true
		seen[typ.Kind()] = true
		switch typ.Kind() {
		case reflect.Struct:
			for i := 0; i < typ.NumField(); i++ {
				walk(typ.Field(i).Type)
			}
		case reflect.Slice, reflect.Array, reflect.Pointer, reflect.Map:
			walk(typ.Elem())
		}
	}
	for _, typ := range contractZoo {
		walk(typ)
	}

	for _, kind := range supported {
		if !seen[kind] {
			t.Errorf("no contract zoo type covers reflect.%s", kind)
		}
	}
}

// wantError holds when parsing fails.
func wantError[T any](input string) error {
	got, err := model.ParseInto[T]([]byte(input))
	if err == nil {
		return fmt.Errorf("no error, got %+v", got)
	}
	return nil
}

// wantValue holds when parsing succeeds with exactly want.
func wantValue[T any](input string, want T) error {
	got, err := model.ParseInto[T]([]byte(input))
	if err != nil {
		return fmt.Errorf("unexpected error: %w", err)
	}
	if !reflect.DeepEqual(got, want) {
		return fmt.Errorf("got %+v, want %+v", got, want)
	}
	return nil
}

// wantSameAsJSON is the value differential: triggered needs coercion, typed does not, and
// encoding/json on typed is the oracle.
func wantSameAsJSON[T any](typed, triggered string) error {
	var want T
	if err := json.Unmarshal([]byte(typed), &want); err != nil {
		return fmt.Errorf("oracle rejected typed input: %w", err)
	}
	return wantValue(triggered, want)
}

// wantSameAsYAML is wantSameAsJSON for YAML; the oracle only reads yaml tags.
func wantSameAsYAML[T any](typed, triggered string) error {
	var want T
	if err := yaml.Unmarshal([]byte(typed), &want); err != nil {
		return fmt.Errorf("oracle rejected typed input: %w", err)
	}
	got, err := model.ParseIntoWithFormat[T]([]byte(triggered), model.FormatYAML)
	if err != nil {
		return fmt.Errorf("unexpected error: %w", err)
	}
	if !reflect.DeepEqual(got, want) {
		return fmt.Errorf("got %+v, want %+v", got, want)
	}
	return nil
}

// verdict returns the set of (leaf field, rule) pairs a parse reports.
func verdict[T any](input string) map[string]bool {
	set := map[string]bool{}
	_, err := model.ParseInto[T]([]byte(input))
	var list model.ErrorList
	if errors.As(err, &list) {
		for _, ve := range list.ValidationErrors() {
			field := ve.Field
			if dot := strings.LastIndex(field, "."); dot >= 0 {
				field = field[dot+1:]
			}
			set[field+"/"+ve.Rule] = true
		}
		return set
	}
	if err != nil {
		set["error"] = true
	}
	return set
}

// wantSameVerdict is the verdict differential across the two decode paths.
func wantSameVerdict[T any](typed, triggered string) error {
	a, b := verdict[T](typed), verdict[T](triggered)
	if !reflect.DeepEqual(a, b) {
		return fmt.Errorf("typed input gives %v, coerced input gives %v", a, b)
	}
	return nil
}

func wantNoPanicOrError[T any](input string) error {
	_, _ = model.ParseInto[T]([]byte(input))
	return nil
}

type contractRow struct {
	group string
	name  string
	// fixedIn is the phase expected to make the row hold. Zero means it must hold now. A row
	// with fixedIn set that holds fails the suite, so each fix has to retire its row.
	fixedIn int
	check   func() error
}

func contractRows() []contractRow {
	return []contractRow{
		// Numeric values that do not fit the target.
		{"numeric", "int8 rejects 300", 0, func() error { return wantError[ctI8](`{"v":300}`) }},
		{"numeric", "int8 rejects string 300", 0, func() error { return wantError[ctI8](`{"v":"300"}`) }},
		{"numeric", "uint8 rejects 300", 0, func() error { return wantError[ctU8](`{"v":300}`) }},
		{"numeric", "int rejects 20 digits", 0, func() error { return wantError[ctInt](`{"v":99999999999999999999}`) }},
		{"numeric", "int rejects 1e30", 0, func() error { return wantError[ctInt](`{"v":1e30}`) }},
		{"numeric", "int rejects 4.7", 0, func() error { return wantError[ctInt](`{"v":4.7}`) }},
		{"numeric", "uint rejects 3.7", 0, func() error { return wantError[ctUint](`{"v":3.7}`) }},
		{"numeric", "uint rejects 1e30", 0, func() error { return wantError[ctUint](`{"v":1e30}`) }},
		{"numeric", "float32 rejects 1e60", 0, func() error { return wantError[ctFloat32](`{"v":1e60}`) }},
		{"numeric", "[]int8 rejects 300", 0, func() error { return wantError[ctI8Slice](`{"v":[300]}`) }},
		{"numeric", "[1]int8 rejects 300", 0, func() error { return wantError[ctI8Array](`{"v":[300]}`) }},
		{"numeric", "*int8 rejects 300", 0, func() error { return wantError[ctI8Ptr](`{"v":300}`) }},
		{"numeric", "string NaN rejected", 0, func() error { return wantError[ctBounded](`{"v":"NaN"}`) }},
		{"numeric", "string Inf rejected", 0, func() error { return wantError[ctFloat64](`{"v":"Inf"}`) }},
		{"numeric", "number keeps its text in a string", 0, func() error {
			return wantValue(`{"v":12345678}`, ctStringField{V: "12345678"})
		}},
		{"numeric", "object is not a string", 0, func() error { return wantError[ctStringField](`{"v":{"a":1}}`) }},
		{"numeric", "int64 zero coerces to false", 0, func() error {
			got, err := model.CoerceValue(int64(0), reflect.TypeOf(true), "f")
			if err != nil || got != false {
				return fmt.Errorf("got %v, err %v", got, err)
			}
			return nil
		}},
		{"numeric", "integral float still coerces", 0, func() error { return wantValue(`{"v":4.0}`, ctInt{V: 4}) }},
		{"numeric", "exponent still coerces", 0, func() error { return wantValue(`{"v":1e3}`, ctInt{V: 1000}) }},
		{"numeric", "exponent integer past 2^53 rejected", 0, func() error { return wantError[ctInt](`{"v":1e16}`) }},
		{"numeric", "uint rejects a negative", 0, func() error { return wantError[ctUint](`{"v":-1}`) }},
		{"numeric", "minimum int8 still fits", 0, func() error { return wantValue(`{"v":-128}`, ctI8{V: -128}) }},
		{"numeric", "maximum uint8 still fits", 0, func() error { return wantValue(`{"v":"255"}`, ctU8{V: 255}) }},
		{"numeric", "maximum int8 still fits", 0, func() error { return wantValue(`{"v":127}`, ctI8{V: 127}) }},
		{"numeric", "string float32 overflow rejected", 0, func() error { return wantError[ctFloat32](`{"v":"1e60"}`) }},

		// One field needing coercion must not change how the others decode.
		{"divergence", "scalars", 0, func() error {
			const body = `"b":true,"i8":1,"i16":2,"i32":3,"i64":4,"u":5,"u8":6,"u16":7,"u32":8,"u64":9,"f32":1.5,"f64":2.5,"s":"x"}`
			return wantSameAsJSON[ctScalars](`{"trigger":1,`+body, `{"trigger":"1",`+body)
		}},
		{"divergence", "containers present", 5, func() error {
			const body = `"slice":["a"],"array":[1,2],"map":{"k":"v"},"ptr":3,"any":{"x":1.5},"nested":{"value":4}}`
			return wantSameAsJSON[ctContainers](`{"trigger":1,`+body, `{"trigger":"1",`+body)
		}},
		{"divergence", "containers absent", 5, func() error {
			return wantSameAsJSON[ctContainers](`{"trigger":1}`, `{"trigger":"1"}`)
		}},
		{"divergence", "map field present", 5, func() error {
			return wantSameAsJSON[ctDeepTrigger](`{"map":{"a":1}}`, `{"map":{"a":"1"}}`)
		}},
		{"divergence", "exported embedded fields", 5, func() error {
			return wantSameAsJSON[ctEmbeddedExported](
				`{"trigger":1,"base":7,"promoted":"p"}`, `{"trigger":"1","base":7,"promoted":"p"}`)
		}},
		{"divergence", "embedded field keeps its value", 5, func() error {
			return wantSameAsJSON[ctEmbedded](`{"trigger":1,"base":7}`, `{"trigger":"1","base":7}`)
		}},
		{"divergence", "tag with no name", 5, func() error {
			return wantSameAsJSON[ctEmptyName](`{"trigger":1,"Name":"n"}`, `{"trigger":"1","Name":"n"}`)
		}},
		{"divergence", "Unmarshaler field", 5, func() error {
			return wantSameAsJSON[ctUnmarshaler](`{"trigger":1,"ip":"1.2.3.4"}`, `{"trigger":"1","ip":"1.2.3.4"}`)
		}},
		{"divergence", "RawMessage keeps its bytes", 5, func() error {
			const meta = `{"b": 1,  "a": [ 2 ]}`
			return wantSameAsJSON[ctRaw](`{"trigger":1,"meta":`+meta+`}`, `{"trigger":"1","meta":`+meta+`}`)
		}},
		{"divergence", "case-insensitive key", 5, func() error {
			return wantSameAsJSON[ctCase](`{"trigger":1,"NAME":"n"}`, `{"trigger":"1","NAME":"n"}`)
		}},
		{"divergence", "Kelvin sign folds like encoding/json", 5, func() error {
			return wantSameAsJSON[ctKelvin](`{"trigger":1,"\u212Aind":"n"}`, `{"trigger":"1","\u212Aind":"n"}`)
		}},
		{"divergence", "time field", 0, func() error {
			const at = `"at":"2026-01-02T03:04:05Z"}`
			var want ctTime
			if err := json.Unmarshal([]byte(`{"trigger":1,`+at), &want); err != nil {
				return err
			}
			got, err := model.ParseInto[ctTime]([]byte(`{"trigger":"1",` + at))
			if err != nil {
				return err
			}
			if !got.At.Equal(want.At) || got.Trigger != want.Trigger {
				return fmt.Errorf("got %+v, want %+v", got, want)
			}
			return nil
		}},
		{"divergence", "int64 above 2^53 stays exact", 5, func() error {
			return wantSameAsJSON[ctBig](`{"trigger":1,"big":9007199254740993}`, `{"trigger":"1","big":9007199254740993}`)
		}},
		{"divergence", "trigger inside a nested object", 5, func() error {
			return wantSameAsJSON[ctDeepTrigger](
				`{"nested":{"trigger":1,"name":"n"},"map":{"a":1}}`,
				`{"nested":{"trigger":"1","name":"n"},"map":{"a":1}}`)
		}},
		{"divergence", "trigger inside a slice element", 5, func() error {
			return wantSameAsJSON[ctSliceTrigger](`{"items":[{"trigger":1}],"map":{"a":1}}`, `{"items":[{"trigger":"1"}],"map":{"a":1}}`)
		}},
		{"divergence", "nested cross-field validator accepts valid input", 5, func() error {
			return wantSameAsJSON[ctCrossOuter](
				`{"trigger":1,"cross":{"a":"x","b":"x"}}`, `{"trigger":"1","cross":{"a":"x","b":"x"}}`)
		}},

		// YAML.
		{"yaml", "json tag honoured without coercion", 5, func() error {
			got, err := model.ParseIntoWithFormat[ctYAMLJSONTag]([]byte("user_name: x\n"), model.FormatYAML)
			if err != nil || got.UserName != "x" {
				return fmt.Errorf("got %+v, err %v", got, err)
			}
			return nil
		}},
		{"yaml", "yaml tag honoured inside []struct", 5, func() error {
			return wantSameAsYAML[ctYAMLList](
				"trigger: 1\nitems:\n  - full_name: a\n", "trigger: \"1\"\nitems:\n  - full_name: a\n")
		}},
		{"yaml", "depth limit covers integer keys", 5, func() error {
			deep := strings.Repeat("{1: ", 200) + "x" + strings.Repeat("}", 200)
			if _, err := model.GetParser(model.FormatYAML).Parse([]byte(deep)); err == nil {
				return errors.New("depth 200 accepted with limit 64")
			}
			return nil
		}},
		{"yaml", "depth limit covers string keys", 0, func() error {
			deep := strings.Repeat("{a: ", 200) + "x" + strings.Repeat("}", 200)
			if _, err := model.GetParser(model.FormatYAML).Parse([]byte(deep)); err == nil {
				return errors.New("depth 200 accepted with limit 64")
			}
			return nil
		}},

		// Validation must run on every path, every time.
		{"validation", "nested rule, second parse", 0, func() error {
			if err := wantError[ctOuterSecond](`{"item":{"name":""}}`); err != nil {
				return fmt.Errorf("first parse: %w", err)
			}
			if err := wantError[ctOuterSecond](`{"item":{"name":""}}`); err != nil {
				return fmt.Errorf("second parse: %w", err)
			}
			return nil
		}},
		{"validation", "pointer rule, second parse", 0, func() error {
			if err := wantError[ctPtrOuter](`{"item":{"name":""}}`); err != nil {
				return fmt.Errorf("first parse: %w", err)
			}
			if err := wantError[ctPtrOuter](`{"item":{"name":""}}`); err != nil {
				return fmt.Errorf("second parse: %w", err)
			}
			return nil
		}},
		{"validation", "[]struct elements", 6, func() error {
			return wantError[ctSliceOuter](`{"trigger":1,"items":[{"name":""}]}`)
		}},
		{"validation", "[N]struct elements", 6, func() error {
			return wantError[ctArrayOuter](`{"trigger":1,"items":[{"name":""}]}`)
		}},
		{"validation", "map[string]struct elements", 6, func() error {
			return wantError[ctMapOuter](`{"trigger":1,"items":{"a":{"name":""}}}`)
		}},
		{"validation", "top-level []struct", 6, func() error { return wantError[[]ctItem](`[{"name":""}]`) }},
		{"validation", "top-level *struct", 6, func() error { return wantError[*ctItem](`{"name":""}`) }},
		{"validation", "verdict matches across paths for []struct", 6, func() error {
			return wantSameVerdict[ctSliceOuter](`{"trigger":1,"items":[{"name":""}]}`, `{"trigger":"1","items":[{"name":""}]}`)
		}},
		{"validation", "failed coercion reports one fault", 6, func() error {
			if got := verdict[ctThreeFaults](`{"v":"x"}`); len(got) != 0 {
				return fmt.Errorf("validation errors on a field that failed coercion: %v", got)
			}
			return nil
		}},
		{"validation", "FieldPath is the full path", 6, func() error {
			_, err := model.ParseInto[ctPathOuter]([]byte(`{"item":{"name":""}}`))
			var list model.ErrorList
			if !errors.As(err, &list) || len(list.ValidationErrors()) != 1 {
				return fmt.Errorf("unexpected error %v", err)
			}
			if ve := list.ValidationErrors()[0]; ve.Field != "Name" || ve.FieldPath != "Item.Name" {
				return fmt.Errorf("Field %q, FieldPath %q", ve.Field, ve.FieldPath)
			}
			return nil
		}},
		{"validation", "NaN fails required, min and max", 6, func() error {
			if err := model.Validate(&ctBounded{V: math.NaN()}); err == nil {
				return errors.New("NaN passed required,min=0,max=100")
			}
			return nil
		}},
		{"validation", "error count is capped on []struct", 0, func() error {
			input := `{"trigger":1,"items":[` + strings.Repeat(`{},`, 5000) + `{}]}`
			_, err := model.ParseInto[ctSliceOuter]([]byte(input))
			var list model.ErrorList
			if errors.As(err, &list) && len(list) > 101 {
				return fmt.Errorf("%d errors", len(list))
			}
			return nil
		}},
		{"validation", "error count is capped on coercion", 0, func() error {
			input := `{"v":[` + strings.Repeat(`"x",`, 5000) + `"x"]}`
			_, err := model.ParseInto[ctI8Slice]([]byte(input))
			var list model.ErrorList
			if errors.As(err, &list) && len(list) > 101 {
				return fmt.Errorf("%d errors", len(list))
			}
			return nil
		}},

		// Tags that cannot run must not pass silently.
		{"tags", "misspelled validator", 7, func() error { return wantError[ctTypo](`{"a":""}`) }},
		{"tags", "len enforces length", 7, func() error { return wantError[ctLen](`{"a":"abc"}`) }},
		{"tags", "min with a bad parameter", 7, func() error { return wantError[ctBadMin](`{"a":"x"}`) }},
		{"tags", "bare min", 7, func() error { return wantError[ctBareMin](`{"a":"x"}`) }},
		{"tags", "unknown tag reported under an empty slice", 7, func() error {
			return wantError[ctTypoInSlice](`{"items":[]}`)
		}},
		{"tags", "validator registered after first parse runs", 7, func() error {
			if _, err := model.ParseInto[ctLate]([]byte(`{"a":"x"}`)); err == nil {
				// Before Phase 7 the unknown rule is dropped and cached, so the new
				// validator never runs.
				model.RegisterGlobalFunc("ct_late", func(field string, value interface{}, _ map[string]interface{}) error {
					return model.NewValidationError(field, value, "ct_late", "always fails")
				})
				return wantError[ctLate](`{"a":"x"}`)
			}
			return errors.New("first parse errored")
		}},

		// Parsing must not panic.
		{"panics", "map target with a bad value", 5, func() error {
			return wantNoPanicOrError[map[string]int](`{"a":"x"}`)
		}},
		{"panics", "map target with a coercible value", 5, func() error {
			return wantValue(`{"a":"1"}`, map[string]int{"a": 1})
		}},
		{"panics", "pointer target needing coercion", 5, func() error {
			return wantNoPanicOrError[*ctInt](`{"v":"1"}`)
		}},
		{"panics", "null element in []map", 5, func() error {
			return wantNoPanicOrError[ctNullElem](`{"trigger":"1","items":[null]}`)
		}},
		{"panics", "CachedParser of an interface type", 8, func() error {
			cached := model.NewCachedParser[any](nil)
			cached.Close()
			return nil
		}},
		{"panics", "CachedParser closed twice", 8, func() error {
			cached := model.NewCachedParser[ctItem](nil)
			cached.Close()
			cached.Close()
			return nil
		}},

		// Cache isolation.
		{"cache", "mutating a hit does not reach the next caller", 8, func() error {
			cached := model.NewCachedParser[ctTagged](nil)
			defer cached.Close()
			input := []byte(`{"trigger":1,"tags":["a"]}`)
			if _, err := cached.Parse(input); err != nil {
				return err
			}
			first, err := cached.Parse(input)
			if err != nil {
				return err
			}
			first.Tags[0] = "mutated"
			second, err := cached.Parse(input)
			if err != nil {
				return err
			}
			if second.Tags[0] != "a" {
				return fmt.Errorf("cached value changed to %q", second.Tags[0])
			}
			return nil
		}},
		{"cache", "mutating a miss does not reach the next caller", 8, func() error {
			cached := model.NewCachedParser[ctTagged](nil)
			defer cached.Close()
			input := []byte(`{"trigger":2,"tags":["a"]}`)
			first, err := cached.Parse(input)
			if err != nil {
				return err
			}
			first.Tags[0] = "mutated"
			second, err := cached.Parse(input)
			if err != nil {
				return err
			}
			if second.Tags[0] != "a" {
				return fmt.Errorf("cached value changed to %q", second.Tags[0])
			}
			return nil
		}},

		// Error messages.
		{"errors", "sensitive value stays out of ParseError", 7, func() error {
			_, err := model.ParseInto[ctSensitive]([]byte(`{"pin":"not-a-pin-4321"}`))
			if err == nil {
				return errors.New("no error")
			}
			if strings.Contains(err.Error(), "4321") {
				return fmt.Errorf("value in message: %v", err)
			}
			return nil
		}},
	}
}

func runContractRow(row contractRow) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("panic: %v", recovered)
		}
	}()
	return row.check()
}

// TestContract fails when a row marked fixedIn starts to hold, as well as when a row that
// must hold breaks, so CI stays green and every fix has to retire the row it fixes.
func TestContract(t *testing.T) {
	model.RegisterGlobalCrossFieldFunc("ct_same", func(field string, value interface{}, parent reflect.Value, _ map[string]interface{}) error {
		if parent.FieldByName("B").String() != value.(string) {
			return model.NewValidationError(field, value, "ct_same", "fields differ")
		}
		return nil
	})

	for _, row := range contractRows() {
		t.Run(row.group+"/"+row.name, func(t *testing.T) {
			err := runContractRow(row)
			switch {
			case row.fixedIn == 0 && err != nil:
				t.Fatal(err)
			case row.fixedIn != 0 && err == nil:
				t.Fatalf("holds, but is marked as fixed in phase %d: clear fixedIn", row.fixedIn)
			case row.fixedIn != 0:
				t.Logf("expected failure until phase %d: %v", row.fixedIn, err)
			}
		})
	}
}
