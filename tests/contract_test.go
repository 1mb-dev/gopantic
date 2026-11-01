package tests

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net"
	"reflect"
	"strings"
	"sync"
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

type ctCoercedItem struct {
	ID   int    `json:"id"`
	Name string `json:"name" validate:"required"`
}

type ctDeepRule struct {
	Name string      `json:"name" validate:"required"`
	Next *ctDeepRule `json:"next"`
}

type ctDeepNoRules struct {
	ID   int            `json:"id"`
	Next *ctDeepNoRules `json:"next"`
}

type ctEmbeddedRule struct {
	ctItem
}

type ctPathSlice struct {
	Items []ctItem `json:"items"`
}

type ctTwoFields struct {
	Bad  int    `json:"bad"`
	Name string `json:"name" validate:"required"`
}

type ctYPoint struct {
	A int `yaml:"a"`
	B int `yaml:"b"`
}

type ctYMerge struct {
	Trigger int      `yaml:"trigger"`
	V       ctYPoint `yaml:"v"`
}

type ctYInlineBase struct {
	Host string `yaml:"host"`
}

type ctYInline struct {
	Trigger       int `yaml:"trigger"`
	ctYInlineBase `yaml:",inline"`
	Extra         map[string]string `yaml:",inline"`
}

type ctYRecursive struct {
	C []ctYRecursive `yaml:"c" json:"c"`
}

type ctDupKeys struct {
	Trigger int    `json:"trigger"`
	Name    string `json:"name"`
	Nested  struct {
		A int `json:"a"`
		B int `json:"b"`
	} `json:"nested"`
}

type ctBytes struct {
	Trigger int    `json:"trigger"`
	Data    []byte `json:"data"`
}

type ctIntKeys struct {
	Trigger int            `json:"trigger"`
	ByID    map[int]string `json:"by_id"`
}

type ctTagForms struct {
	Trigger int    `json:"trigger"`
	Ignored string `json:"-"`
	Dash    string `json:"-,"`
	Legacy  string
}

// CtShadow is embedded for its Name, which the outer struct's field must shadow.
type CtShadow struct {
	Name string `json:"name"`
	Only string `json:"only"`
}

// CtShadowWrap adds a level, since vet reports a repeated tag one level down as a mistake.
type CtShadowWrap struct {
	CtShadow
}

// CtAmbiguousA and CtAmbiguousB both claim Name at one depth, so neither decodes. The fields
// are untagged because vet reports a repeated tag on embedded fields as a mistake.
type CtAmbiguousA struct {
	Name string
}

// CtAmbiguousB is the other half of the ambiguity.
type CtAmbiguousB struct {
	Name string
}

type ctEmbedForms struct {
	*ctExportedPtr
	CtShadowWrap
	Trigger int    `json:"trigger"`
	Name    string `json:"name"`
}

type ctExportedPtr struct {
	Via string `json:"via"`
}

type ctAmbiguous struct {
	CtAmbiguousA
	CtAmbiguousB
	Trigger int `json:"trigger"`
}

type ctNulls struct {
	Trigger int      `json:"trigger"`
	I       int      `json:"i"`
	P       *int     `json:"p"`
	S       []string `json:"s"`
	A       any      `json:"a"`
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

type ctLateDefault struct {
	A string `json:"a" validate:"ct_late_default"`
}

type ctLateCross struct {
	A string `json:"a" validate:"ct_late_cross"`
}

type ctLateFactory struct {
	A string `json:"a" validate:"ct_late_factory"`
}

type ctLateRace struct {
	A string `json:"a" validate:"ct_late_race"`
}

type ctCoercedTypo struct {
	N int    `json:"n"`
	A string `json:"a" validate:"requird"`
}

type ctNestedTypo struct {
	Item ctTypoItem `json:"item"`
}

type ctTypoCycle struct {
	A    string       `json:"a" validate:"requird"`
	Next *ctTypoCycle `json:"next"`
}

type ctEmptyMinValue struct {
	A string `json:"a" validate:"min="`
}

type ctNaNMin struct {
	A string `json:"a" validate:"min=NaN"`
}

type ctStrayComma struct {
	A string `json:"a" validate:"required,"`
}

type ctFractionLength struct {
	A string `json:"a" validate:"length=2.5"`
}

type ctContainerCycle []ctContainerCycle

type ctMinOverride struct {
	A string `json:"a" validate:"ct_override=abc"`
}

type ctSensitive struct {
	Password int `json:"p"`
}

type ctSensitiveKey struct {
	P int `json:"api_key"`
}

type ctSensitiveTime struct {
	Password time.Time `json:"p"`
}

type ctSensitiveParent struct {
	Credentials struct {
		Value int `json:"value"`
	} `json:"c"`
}

type ctSensitiveSlice struct {
	Passwords []int `json:"p"`
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

// wantTagError requires the error to come from the tag rather than the value.
func wantTagError[T any](input string) error {
	_, err := model.ParseInto[T]([]byte(input))
	if err == nil || !strings.Contains(err.Error(), "invalid validate tag") {
		return fmt.Errorf("got %v", err)
	}
	return nil
}

// wantRedacted requires a ParseError that hides the value everywhere except its raw Value field.
func wantRedacted[T any](input string) error {
	_, err := model.ParseInto[T]([]byte(input))
	var list model.ErrorList
	if !errors.As(err, &list) || len(list) != 1 {
		return fmt.Errorf("unexpected error %v", err)
	}
	if strings.Contains(err.Error(), "4321") {
		return fmt.Errorf("value in message: %v", err)
	}
	var parseErr *model.ParseError
	if !errors.As(list[0], &parseErr) {
		return fmt.Errorf("not a ParseError: %v", list[0])
	}
	if got := parseErr.SanitizedValue(); got != model.RedactedValue {
		return fmt.Errorf("SanitizedValue returned %v", got)
	}
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
		{"divergence", "containers present", 0, func() error {
			const body = `"slice":["a"],"array":[1,2],"map":{"k":"v"},"ptr":3,"any":{"x":1.5},"nested":{"value":4}}`
			return wantSameAsJSON[ctContainers](`{"trigger":1,`+body, `{"trigger":"1",`+body)
		}},
		{"divergence", "containers absent", 0, func() error {
			return wantSameAsJSON[ctContainers](`{"trigger":1}`, `{"trigger":"1"}`)
		}},
		{"divergence", "map field present", 0, func() error {
			return wantSameAsJSON[ctDeepTrigger](`{"map":{"a":1}}`, `{"map":{"a":"1"}}`)
		}},
		{"divergence", "exported embedded fields", 0, func() error {
			return wantSameAsJSON[ctEmbeddedExported](
				`{"trigger":1,"base":7,"promoted":"p"}`, `{"trigger":"1","base":7,"promoted":"p"}`)
		}},
		{"divergence", "embedded field keeps its value", 0, func() error {
			return wantSameAsJSON[ctEmbedded](`{"trigger":1,"base":7}`, `{"trigger":"1","base":7}`)
		}},
		{"divergence", "tag with no name", 0, func() error {
			return wantSameAsJSON[ctEmptyName](`{"trigger":1,"Name":"n"}`, `{"trigger":"1","Name":"n"}`)
		}},
		{"divergence", "Unmarshaler field", 0, func() error {
			return wantSameAsJSON[ctUnmarshaler](`{"trigger":1,"ip":"1.2.3.4"}`, `{"trigger":"1","ip":"1.2.3.4"}`)
		}},
		{"divergence", "RawMessage keeps its bytes", 0, func() error {
			const meta = `{"b": 1,  "a": [ 2 ]}`
			return wantSameAsJSON[ctRaw](`{"trigger":1,"meta":`+meta+`}`, `{"trigger":"1","meta":`+meta+`}`)
		}},
		{"divergence", "case-insensitive key", 0, func() error {
			return wantSameAsJSON[ctCase](`{"trigger":1,"NAME":"n"}`, `{"trigger":"1","NAME":"n"}`)
		}},
		{"divergence", "Kelvin sign folds like encoding/json", 0, func() error {
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
		{"divergence", "int64 above 2^53 stays exact", 0, func() error {
			return wantSameAsJSON[ctBig](`{"trigger":1,"big":9007199254740993}`, `{"trigger":"1","big":9007199254740993}`)
		}},
		{"divergence", "trigger inside a nested object", 0, func() error {
			return wantSameAsJSON[ctDeepTrigger](
				`{"nested":{"trigger":1,"name":"n"},"map":{"a":1}}`,
				`{"nested":{"trigger":"1","name":"n"},"map":{"a":1}}`)
		}},
		{"divergence", "trigger inside a slice element", 0, func() error {
			return wantSameAsJSON[ctSliceTrigger](`{"items":[{"trigger":1}],"map":{"a":1}}`, `{"items":[{"trigger":"1"}],"map":{"a":1}}`)
		}},
		{"divergence", "nested cross-field validator accepts valid input", 0, func() error {
			return wantSameAsJSON[ctCrossOuter](
				`{"trigger":1,"cross":{"a":"x","b":"x"}}`, `{"trigger":"1","cross":{"a":"x","b":"x"}}`)
		}},

		// YAML.
		{"yaml", "json tag honoured without coercion", 0, func() error {
			got, err := model.ParseIntoWithFormat[ctYAMLJSONTag]([]byte("user_name: x\n"), model.FormatYAML)
			if err != nil || got.UserName != "x" {
				return fmt.Errorf("got %+v, err %v", got, err)
			}
			return nil
		}},
		{"yaml", "yaml tag honoured inside []struct", 0, func() error {
			return wantSameAsYAML[ctYAMLList](
				"trigger: 1\nitems:\n  - full_name: a\n", "trigger: \"1\"\nitems:\n  - full_name: a\n")
		}},
		{"yaml", "depth limit covers integer keys", 0, func() error {
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

		{"yaml", "explicit key beats a merged one", 0, func() error {
			const body = "base: &b {a: 1, b: 2}\nv:\n  <<: *b\n  b: 3\n"
			return wantSameAsYAML[ctYMerge]("trigger: 1\n"+body, "trigger: \"1\"\n"+body)
		}},
		{"yaml", "explicit key after the merge also wins", 0, func() error {
			const body = "base: &b {a: 1, b: 2}\nv:\n  b: 3\n  <<: *b\n"
			return wantSameAsYAML[ctYMerge]("trigger: 1\n"+body, "trigger: \"1\"\n"+body)
		}},
		{"yaml", "earlier source wins in a merge list", 0, func() error {
			const body = "x: &x {a: 1}\ny: &y {a: 9, b: 2}\nv:\n  <<: [*x, *y]\n"
			return wantSameAsYAML[ctYMerge]("trigger: 1\n"+body, "trigger: \"1\"\n"+body)
		}},
		{"yaml", "plain alias", 0, func() error {
			const body = "x: &x {a: 1, b: 2}\nv: *x\n"
			return wantSameAsYAML[ctYMerge]("trigger: 1\n"+body, "trigger: \"1\"\n"+body)
		}},
		{"yaml", "inline struct and map", 0, func() error {
			const body = "host: h\nother: o\n"
			return wantSameAsYAML[ctYInline]("trigger: 1\n"+body, "trigger: \"1\"\n"+body)
		}},
		{"yaml", "alias bomb errors with the depth limit off", 0, func() error {
			orig := model.GetMaxStructureDepth()
			model.SetMaxStructureDepth(0)
			defer model.SetMaxStructureDepth(orig)
			var b strings.Builder
			b.WriteString("a0: &a0 {c: [{c: []}, {c: []}, {c: []}, {c: []}, {c: []}, {c: []}, {c: []}, {c: []}, {c: []}]}\n")
			for i := 1; i < 9; i++ {
				fmt.Fprintf(&b, "a%d: &a%d {c: [*a%d, *a%d, *a%d, *a%d, *a%d, *a%d, *a%d, *a%d, *a%d]}\n", i, i, i-1, i-1, i-1, i-1, i-1, i-1, i-1, i-1, i-1)
			}
			b.WriteString("c: [*a8]\n")
			_, err := model.ParseIntoWithFormat[ctYRecursive]([]byte(b.String()), model.FormatYAML)
			if err == nil || !strings.Contains(err.Error(), "aliasing") {
				return fmt.Errorf("got %.120v", err)
			}
			return nil
		}},
		{"yaml", "self-reference errors with the depth limit off", 0, func() error {
			orig := model.GetMaxStructureDepth()
			model.SetMaxStructureDepth(0)
			defer model.SetMaxStructureDepth(orig)
			_, err := model.ParseIntoWithFormat[ctYRecursive]([]byte("c: &x\n  - c: *x\n"), model.FormatYAML)
			if err == nil {
				return errors.New("no error")
			}
			return nil
		}},

		// Decoder details the walker has to match.
		{"parity", "last duplicate key wins", 0, func() error {
			const body = `"name":"a","name":"b"}`
			return wantSameAsJSON[ctDupKeys](`{"trigger":1,`+body, `{"trigger":"1",`+body)
		}},
		{"parity", "duplicate object key merges", 0, func() error {
			const body = `"nested":{"a":1},"nested":{"b":2}}`
			return wantSameAsJSON[ctDupKeys](`{"trigger":1,`+body, `{"trigger":"1",`+body)
		}},
		{"parity", "escaped key", 0, func() error {
			const body = `"na\u006de":"x"}`
			return wantSameAsJSON[ctDupKeys](`{"trigger":1,`+body, `{"trigger":"1",`+body)
		}},
		{"parity", "[]byte from base64", 0, func() error {
			const body = `"data":"aGk="}`
			return wantSameAsJSON[ctBytes](`{"trigger":1,`+body, `{"trigger":"1",`+body)
		}},
		{"parity", "int map keys", 0, func() error {
			const body = `"by_id":{"1":"a","2":"b"}}`
			return wantSameAsJSON[ctIntKeys](`{"trigger":1,`+body, `{"trigger":"1",`+body)
		}},
		{"parity", "tag forms", 0, func() error {
			const body = `"Ignored":"no","-":"dash","Legacy":"l"}`
			return wantSameAsJSON[ctTagForms](`{"trigger":1,`+body, `{"trigger":"1",`+body)
		}},
		{"parity", "embedded precedence and pointers", 0, func() error {
			const body = `"name":"outer","only":"o"}`
			return wantSameAsJSON[ctEmbedForms](`{"trigger":1,`+body, `{"trigger":"1",`+body)
		}},
		{"parity", "ambiguous embedded fields stay zero", 0, func() error {
			const body = `"Name":"x"}`
			return wantSameAsJSON[ctAmbiguous](`{"trigger":1,`+body, `{"trigger":"1",`+body)
		}},
		{"parity", "nulls", 0, func() error {
			const body = `"i":null,"p":null,"s":null,"a":null}`
			return wantSameAsJSON[ctNulls](`{"trigger":1,`+body, `{"trigger":"1",`+body)
		}},
		{"parity", "depth scan ignores brackets in strings", 0, func() error {
			value := strings.Repeat("[", 200) + `\"` + strings.Repeat("{", 200)
			if _, err := model.ParseInto[ctStringField]([]byte(`{"v":"` + value + `"}`)); err != nil {
				return err
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
		{"validation", "[]struct elements", 0, func() error {
			return wantError[ctSliceOuter](`{"trigger":1,"items":[{"name":""}]}`)
		}},
		{"validation", "[N]struct elements", 0, func() error {
			return wantError[ctArrayOuter](`{"trigger":1,"items":[{"name":""}]}`)
		}},
		{"validation", "map[string]struct elements", 0, func() error {
			return wantError[ctMapOuter](`{"trigger":1,"items":{"a":{"name":""}}}`)
		}},
		{"validation", "top-level []struct", 0, func() error { return wantError[[]ctItem](`[{"name":""}]`) }},
		{"validation", "top-level *struct", 0, func() error { return wantError[*ctItem](`{"name":""}`) }},
		{"validation", "verdict matches across paths for []struct", 0, func() error {
			return wantSameVerdict[ctSliceOuter](`{"trigger":1,"items":[{"name":""}]}`, `{"trigger":"1","items":[{"name":""}]}`)
		}},
		{"validation", "failed coercion reports one fault", 0, func() error {
			if got := verdict[ctThreeFaults](`{"v":"x"}`); len(got) != 0 {
				return fmt.Errorf("validation errors on a field that failed coercion: %v", got)
			}
			return nil
		}},
		{"validation", "FieldPath is the full path", 0, func() error {
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
		{"validation", "NaN fails required, min and max", 0, func() error {
			if err := model.Validate(&ctBounded{V: math.NaN()}); err == nil {
				return errors.New("NaN passed required,min=0,max=100")
			}
			return nil
		}},
		{"validation", "rule on an embedded struct's field runs", 0, func() error {
			return wantError[ctEmbeddedRule](`{"name":""}`)
		}},
		{"validation", "error path through a slice", 0, func() error {
			_, err := model.ParseInto[ctPathSlice]([]byte(`{"items":[{"name":"a"},{"name":""}]}`))
			var list model.ErrorList
			if !errors.As(err, &list) || len(list.ValidationErrors()) != 1 {
				return fmt.Errorf("unexpected error %v", err)
			}
			if got := list.ValidationErrors()[0].FieldPath; got != "Items[1].Name" {
				return fmt.Errorf("FieldPath %q", got)
			}
			return nil
		}},
		{"validation", "coercion error on one field keeps validation on another", 0, func() error {
			got := verdict[ctTwoFields](`{"bad":"x","name":""}`)
			if !got["Name/required"] {
				return fmt.Errorf("rule on the other field did not run: %v", got)
			}
			return nil
		}},
		{"validation", "top-level []struct on the coercion path", 0, func() error {
			return wantError[[]ctCoercedItem](`[{"id":"1","name":""}]`)
		}},
		{"validation", "top-level [N]struct on the coercion path", 0, func() error {
			return wantError[[1]ctCoercedItem](`[{"id":"1","name":""}]`)
		}},
		{"validation", "long map key stays out of the path", 0, func() error {
			input := `{"` + strings.Repeat("k", 1<<20) + `":[{"name":""}]}`
			_, err := model.ParseInto[map[string][]ctItem]([]byte(input))
			var list model.ErrorList
			if !errors.As(err, &list) || len(list.ValidationErrors()) != 1 {
				return fmt.Errorf("unexpected error %.100v", err)
			}
			if got := len(list.ValidationErrors()[0].FieldPath); got > 128 {
				return fmt.Errorf("FieldPath is %d bytes", got)
			}
			return nil
		}},
		{"validation", "error count is capped on a map", 0, func() error {
			var input strings.Builder
			input.WriteString(`{"a":{"name":""}`)
			for i := 0; i < 5000; i++ {
				fmt.Fprintf(&input, `,"k%d":{"name":""}`, i)
			}
			input.WriteString(`}`)
			_, err := model.ParseInto[map[string]ctItem]([]byte(input.String()))
			var list model.ErrorList
			if !errors.As(err, &list) || len(list) != 101 {
				return fmt.Errorf("got %d errors", len(list))
			}
			return nil
		}},
		{"validation", "rule below the depth limit fails closed", 0, func() error {
			input := strings.Repeat(`{"next":`, 40) + `{"name":""}` + strings.Repeat(`}`, 40)
			return wantError[ctDeepRule](input)
		}},
		{"validation", "rule-free nesting past the depth limit decodes", 0, func() error {
			levels := model.GetMaxValidationDepth() + 8
			input := strings.Repeat(`{"id":"1","next":`, levels) + `null` + strings.Repeat(`}`, levels)
			if _, err := model.ParseInto[ctDeepNoRules]([]byte(input)); err != nil {
				return err
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
		{"tags", "misspelled validator", 0, func() error { return wantTagError[ctTypo](`{"a":"x"}`) }},
		{"tags", "error names struct, field, tag and the way out", 0, func() error {
			_, err := model.ParseInto[ctTypo]([]byte(`{"a":"x"}`))
			for _, part := range []string{"ctTypo.A", `"requird,emial"`, `"requird"`, "required", "register it first"} {
				if err == nil || !strings.Contains(err.Error(), part) {
					return fmt.Errorf("missing %q in %v", part, err)
				}
			}
			return nil
		}},
		{"tags", "errors on every call", 0, func() error {
			for i := 0; i < 3; i++ {
				if err := wantTagError[ctTypo](`{"a":"x"}`); err != nil {
					return fmt.Errorf("call %d: %w", i, err)
				}
			}
			return nil
		}},
		{"tags", "errors on the coercion path", 0, func() error { return wantTagError[ctCoercedTypo](`{"n":"1","a":"x"}`) }},
		{"tags", "errors on YAML input", 0, func() error {
			_, err := model.ParseIntoWithFormat[ctTypo]([]byte("a: x\n"), model.FormatYAML)
			if err == nil || !strings.Contains(err.Error(), "unknown validator") {
				return fmt.Errorf("got %v", err)
			}
			return nil
		}},
		{"tags", "errors from Validate", 0, func() error {
			err := model.Validate(&ctTypo{A: "x"})
			if err == nil || !strings.Contains(err.Error(), "unknown validator") {
				return fmt.Errorf("got %v", err)
			}
			return nil
		}},
		{"tags", "errors when nested", 0, func() error { return wantTagError[ctNestedTypo](`{"item":{"a":"x"}}`) }},
		{"tags", "errors in a recursive type", 0, func() error { return wantTagError[ctTypoCycle](`{"a":"x"}`) }},
		{"tags", "len enforces length", 0, func() error { return wantError[ctLen](`{"a":"abc"}`) }},
		{"tags", "len accepts the right length", 0, func() error { return wantValue(`{"a":"abcdefgh"}`, ctLen{A: "abcdefgh"}) }},
		{"tags", "min with a bad parameter", 0, func() error { return wantTagError[ctBadMin](`{"a":"x"}`) }},
		{"tags", "bare min", 0, func() error { return wantTagError[ctBareMin](`{"a":"x"}`) }},
		{"tags", "min with an empty parameter", 0, func() error { return wantTagError[ctEmptyMinValue](`{"a":"x"}`) }},
		{"tags", "min with NaN", 0, func() error { return wantTagError[ctNaNMin](`{"a":"x"}`) }},
		{"tags", "stray comma", 0, func() error { return wantTagError[ctStrayComma](`{"a":"x"}`) }},
		{"tags", "fractional length", 0, func() error { return wantTagError[ctFractionLength](`{"a":"xx"}`) }},
		{"tags", "unknown tag reported under an empty slice", 0, func() error {
			return wantTagError[ctTypoInSlice](`{"items":[]}`)
		}},
		{"tags", "unknown tag reported under an empty top-level slice", 0, func() error {
			return wantTagError[[]ctTypoItem](`[]`)
		}},
		{"tags", "unknown tag reported under an empty top-level map", 0, func() error {
			return wantTagError[map[string]*ctTypoItem](`{}`)
		}},
		{"tags", "recursive container type terminates", 0, func() error {
			_, err := model.ParseInto[ctContainerCycle]([]byte(`[[],[[]]]`))
			return err
		}},
		{"tags", "custom validator can take a built-in's parameter form", 0, func() error {
			model.RegisterGlobalFunc("ct_override", func(string, interface{}, map[string]interface{}) error { return nil })
			return wantValue(`{"a":"x"}`, ctMinOverride{A: "x"})
		}},
		{"tags", "RegisterGlobalFunc reaches a parsed type", 0, func() error {
			_, _ = model.ParseInto[ctLate]([]byte(`{"a":"x"}`))
			model.RegisterGlobalFunc("ct_late", func(string, interface{}, map[string]interface{}) error { return nil })
			return wantValue(`{"a":"x"}`, ctLate{A: "x"})
		}},
		{"tags", "default registry RegisterFunc reaches a parsed type", 0, func() error {
			_, _ = model.ParseInto[ctLateDefault]([]byte(`{"a":"x"}`))
			model.GetDefaultRegistry().RegisterFunc("ct_late_default", func(string, interface{}, map[string]interface{}) error { return nil })
			return wantValue(`{"a":"x"}`, ctLateDefault{A: "x"})
		}},
		{"tags", "cross-field registration reaches a parsed type", 0, func() error {
			_, _ = model.ParseInto[ctLateCross]([]byte(`{"a":"x"}`))
			model.RegisterGlobalCrossFieldFunc("ct_late_cross", func(string, interface{}, reflect.Value, map[string]interface{}) error { return nil })
			return wantValue(`{"a":"x"}`, ctLateCross{A: "x"})
		}},
		{"tags", "factory registration reaches a parsed type", 0, func() error {
			_, _ = model.ParseInto[ctLateFactory]([]byte(`{"a":"x"}`))
			model.GetDefaultRegistry().Register("ct_late_factory", func(map[string]interface{}) model.Validator {
				return &model.AlphaValidator{}
			})
			return wantValue(`{"a":"x"}`, ctLateFactory{A: "x"})
		}},
		{"tags", "registration racing parses leaves no stale rules", 0, func() error {
			var parses sync.WaitGroup
			done := make(chan struct{})
			for i := 0; i < 8; i++ {
				parses.Add(1)
				go func() {
					defer parses.Done()
					for {
						select {
						case <-done:
							return
						default:
							_, _ = model.ParseInto[ctLateRace]([]byte(`{"a":"x"}`))
						}
					}
				}()
			}
			model.RegisterGlobalFunc("ct_late_race", func(string, interface{}, map[string]interface{}) error { return nil })
			parsed := wantValue(`{"a":"x"}`, ctLateRace{A: "x"})
			close(done)
			parses.Wait()
			if parsed != nil {
				return parsed
			}
			return wantValue(`{"a":"x"}`, ctLateRace{A: "x"})
		}},

		// Parsing must not panic.
		{"panics", "map target with a bad value", 0, func() error {
			return wantNoPanicOrError[map[string]int](`{"a":"x"}`)
		}},
		{"panics", "map target with a coercible value", 0, func() error {
			return wantValue(`{"a":"1"}`, map[string]int{"a": 1})
		}},
		{"panics", "pointer target needing coercion", 0, func() error {
			return wantNoPanicOrError[*ctInt](`{"v":"1"}`)
		}},
		{"panics", "null element in []map", 0, func() error {
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
		{"errors", "sensitive Go name keeps the value out of the message", 0, func() error {
			return wantRedacted[ctSensitive](`{"p":"not-a-pin-4321"}`)
		}},
		{"errors", "sensitive input key keeps the value out of the message", 0, func() error {
			return wantRedacted[ctSensitiveKey](`{"api_key":"not-a-pin-4321"}`)
		}},
		{"errors", "sensitive time field", 0, func() error {
			return wantRedacted[ctSensitiveTime](`{"p":"not-a-pin-4321"}`)
		}},
		{"errors", "field under a sensitive parent", 0, func() error {
			return wantRedacted[ctSensitiveParent](`{"c":{"value":"not-a-pin-4321"}}`)
		}},
		{"errors", "slice element of a sensitive field", 0, func() error {
			return wantRedacted[ctSensitiveSlice](`{"p":["not-a-pin-4321"]}`)
		}},
		{"errors", "sensitive map key", 0, func() error {
			return wantRedacted[map[string]int](`{"password":"not-a-pin-4321"}`)
		}},
		{"errors", "CoerceValue on a sensitive field", 0, func() error {
			_, err := model.CoerceValue("not-a-pin-4321", reflect.TypeOf(0), "password")
			if err == nil || strings.Contains(err.Error(), "4321") {
				return fmt.Errorf("got %v", err)
			}
			return nil
		}},
		{"errors", "ordinary field keeps the value in the message", 0, func() error {
			_, err := model.ParseInto[ctInt]([]byte(`{"v":"not-a-number-4321"}`))
			if err == nil || !strings.Contains(err.Error(), "4321") {
				return fmt.Errorf("got %v", err)
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
