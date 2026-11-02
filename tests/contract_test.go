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

type ctDashRule struct {
	A string `json:"-" validate:"required"`
}

type ctDashCoerced struct {
	N int    `json:"n"`
	A string `json:"-" validate:"required"`
}

type ctDashYAMLRule struct {
	A string `json:"-" yaml:"a" validate:"required"`
}

type ctDashNoRule struct {
	A string `json:"-"`
	B string `json:"b" validate:"required"`
}

type ctDashKey struct {
	A string `json:"-," validate:"required"`
}

type ctDashYAMLOff struct {
	A string `json:"a" yaml:"-" validate:"required"`
}

type ctDashBoth struct {
	A string `json:"-" yaml:"-" validate:"required"`
}

type ctDashNested struct {
	Inner ctItem `json:"-" yaml:"inner"`
}

type ctDashList struct {
	Items []ctItem `json:"-" yaml:"items"`
}

type ctDashInline struct {
	ctItem `json:"-" yaml:",inline"`
}

type ctDashTypo struct {
	A string `json:"-" validate:"requird"`
}

type ctDashYAMLTypo struct {
	Inner ctTypoItem `json:"-" yaml:"inner"`
}

// ctForeign stands for a type from another library, tagged for another validator.
type ctForeign struct {
	Name string `validate:"required_with=Other"`
}

type ctDashForeign struct {
	Owner *ctForeign `json:"-"`
	Name  string     `json:"name"`
}

type ctDashTree struct {
	Name     string        `json:"name" validate:"required"`
	Children []*ctDashTree `json:"children"`
	Parent   *ctDashTree   `json:"-"`
}

type ctDashHiddenEmbed struct {
	ctItem `json:"-"`
	N      int `json:"n"`
}

type ctDashHash struct {
	Hash string `json:"-" validate:"min=5"`
}

type ctDashHashParent struct {
	Inner struct {
		Code string `yaml:"code" validate:"min=5"`
	} `json:"-" yaml:"inner"`
}

type ctDashYAMLInt struct {
	N int `json:"-" yaml:"n"`
}

type ctDashEmbedHash struct {
	Code string `yaml:"code" validate:"min=5"`
}

type ctDashHiddenEmbedHash struct {
	ctDashEmbedHash `json:"-" yaml:",inline"`
}

type ctDashInt struct {
	N int `yaml:"n"`
}

type ctDashInlineInt struct {
	Inner ctDashInt `json:"-" yaml:",inline"`
}

type ctDashInlineMap struct {
	Extra map[string]int `json:"-" yaml:",inline"`
}

type ctYInt struct {
	V int `yaml:"v"`
}

type ctYI8 struct {
	V int8 `yaml:"v"`
}

type ctYU64 struct {
	V uint64 `yaml:"v"`
}

type ctYF32 struct {
	V float32 `yaml:"v"`
}

type ctYInlineInner struct {
	Extra map[string]string `yaml:",inline"`
}

type ctYPtrInline struct {
	Name string          `yaml:"name"`
	In   *ctYInlineInner `yaml:",inline"`
}

type ctYHiddenInlineStruct struct {
	meta ctItem `yaml:",inline"`
}

type ctYHiddenInlineMap struct {
	A     int               `yaml:"a"`
	extra map[string]string `yaml:",inline"`
}

type ctLoopMap map[string]ctLoopMap

type ctLoopHolder struct {
	ID   int       `validate:"min=0"`
	Kids ctLoopMap `json:"kids"`
}

type ctLoopSlice []ctLoopSlice

type ctLoopSliceHolder struct {
	ID   int         `validate:"min=0"`
	Kids ctLoopSlice `json:"kids"`
}

type ctTimeField struct {
	At time.Time `json:"at" yaml:"at"`
}

type ctKeySensitive struct {
	Code string `json:"password" validate:"min=100"`
}

type ctKeySensitiveYAML struct {
	Code string `json:"c" yaml:"api_key" validate:"min=100"`
}

type ctKeySensitiveParent struct {
	Inner struct {
		Code string `json:"code" validate:"min=100"`
	} `json:"credentials"`
}

// ctOldYAML uses the func-signature form of UnmarshalYAML, which yaml.v3 still calls.
type ctOldYAML struct {
	Got string
}

func (o *ctOldYAML) UnmarshalYAML(unmarshal func(interface{}) error) error {
	return unmarshal(&o.Got)
}

type ctOldYAMLField struct {
	C ctOldYAML `yaml:"c"`
}

type ctYAMLNodeField struct {
	Cfg yaml.Node `yaml:"cfg"`
}

type ctRequiredFloat struct {
	V float64 `yaml:"v" validate:"required"`
}

// ctNullAware records whether its decoder ran.
type ctNullAware struct {
	Set bool
}

func (n *ctNullAware) UnmarshalJSON([]byte) error {
	n.Set = true
	return nil
}

type ctNullField struct {
	Trigger int         `json:"trigger"`
	O       ctNullAware `json:"o"`
}

type ctQuoted struct {
	Trigger int     `json:"trigger"`
	Name    string  `json:"name,string"`
	N       int     `json:"n,string"`
	B       bool    `json:"b,string"`
	F       float64 `json:"f,string"`
}

type ctCoerceTarget struct {
	Age int `json:"age" validate:"min=18"`
}

type ctYDuration struct {
	D time.Duration `yaml:"d"`
}

type ctYUintptr struct {
	P uintptr `yaml:"p"`
}

type ctYBool struct {
	V bool `yaml:"v"`
}

type ctQuotedInt struct {
	A int  `json:"a,string"`
	B bool `json:"b,string"`
}

type ctNumber struct {
	N json.Number `json:"n"`
}

type ctYPtrInlineMap struct {
	Name  string          `yaml:"name"`
	Extra *map[string]int `yaml:",inline"`
}

type ctYAnyKeys struct {
	Meta map[interface{}]interface{} `yaml:"meta"`
}

type ctYExtraItem struct {
	Age int `yaml:"age" validate:"required"`
}

type ctYInlineItems struct {
	Extra map[string]ctYExtraItem `yaml:",inline"`
}

type ctYInlineSecrets struct {
	Secrets map[string]int `yaml:",inline"`
}

type ctYEmbAge struct {
	Age int `yaml:"age" validate:"required"`
}

type ctYInlineEmbed struct {
	ctYEmbAge `yaml:",inline"`
}

type ctEmbedCrossBase struct {
	A string `json:"a" validate:"ct_cross_iface"`
}

type ctEmbedCross struct {
	ctEmbedCrossBase
}

type ctEmbedPtrBase struct {
	X int `json:"x" validate:"required"`
}

type ctEmbedPtr struct {
	*ctEmbedPtrBase
}

type ctCfgString struct {
	Cfg string `json:"cfg"`
}

type ctShadowEmbed struct {
	Pass    string `json:"pass"`
	Confirm string `json:"confirm" validate:"ct_same_pass"`
}

type ctShadowOuter struct {
	ctShadowEmbed
	Pass string `json:"outer_pass"`
}

type ctAnyKeyInts struct {
	Meta map[interface{}]int `yaml:"meta"`
}

type ctAnyKeyInner struct {
	UserName string `json:"user_name" validate:"required"`
}

type ctAnyKeyStructs struct {
	M map[interface{}]ctAnyKeyInner `yaml:"m"`
}

type ctCoerceTwo struct {
	Age  int    `json:"age"`
	Name string `json:"name" validate:"required"`
}

type ctInnerPass struct {
	Pass string
}

type ctPtrEmbedRule struct {
	*ctInnerPass
	Confirm string `validate:"ct_same_pass"`
}

type ctPtrEmbedOuter struct {
	ctPtrEmbedRule
}

type ctSiblingRule struct {
	secret  string
	Confirm string `validate:"ct_same_secret"`
}

type ctSiblingOuter struct {
	ctSiblingRule
}

type ctAnyKeyItems struct {
	M map[interface{}]ctItem `yaml:"m"`
}

type ctAnyKeyAges struct {
	M map[interface{}]ctYExtraItem `yaml:"m"`
}

type ctYKeyMap struct {
	Attrs map[string]int `yaml:"attrs"`
}

type ctYKeyInline struct {
	Name  string         `yaml:"name"`
	Extra map[string]int `yaml:",inline"`
}

type ctYAnyStrings struct {
	M map[interface{}]string `yaml:"m"`
}

type ctYStrings struct {
	M map[string]string `yaml:"m"`
}

type ctPwUser struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

type ctPwUsers struct {
	Users []ctPwUser `json:"users" validate:"min=2"`
}

type ctLoopRule struct {
	Name string      `validate:"required"`
	Next *ctLoopRule `json:"next"`
}

type ctNegativeCache struct {
	A string `json:"a" validate:"required"`
}

type ctRequiredArray struct {
	X int     `json:"x"`
	H [4]byte `json:"h" validate:"required"`
}

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

func wantValidationRedacted(err error) error {
	var list model.ErrorList
	if !errors.As(err, &list) || len(list.ValidationErrors()) != 1 {
		return fmt.Errorf("unexpected error %v", err)
	}
	if got := list.ValidationErrors()[0].SanitizedValue(); got != model.RedactedValue {
		return fmt.Errorf("SanitizedValue returned %v", got)
	}
	return nil
}

func wantYAMLError[T any](input string) error {
	got, err := model.ParseIntoWithFormat[T]([]byte(input), model.FormatYAML)
	if err == nil {
		return fmt.Errorf("accepted %q as %+v", input, got)
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
		{"numeric", "YAML float does not truncate into an int", 0, func() error {
			return wantYAMLError[ctYInt]("v: 1.9\n")
		}},
		{"numeric", "YAML float does not truncate into an int8", 0, func() error {
			return wantYAMLError[ctYI8]("v: 100.5\n")
		}},
		{"numeric", "YAML negative float does not wrap into a uint64", 0, func() error {
			return wantYAMLError[ctYU64]("v: -1.0\n")
		}},
		{"numeric", "YAML float32 overflow rejected", 0, func() error {
			return wantYAMLError[ctYF32]("v: 1e39\n")
		}},
		{"numeric", "YAML int8 overflow rejected", 0, func() error {
			return wantYAMLError[ctYI8]("v: 300\n")
		}},
		{"numeric", "YAML whole float still converts", 0, func() error {
			got, err := model.ParseIntoWithFormat[ctYInt]([]byte("v: 4.0\n"), model.FormatYAML)
			if err != nil || got.V != 4 {
				return fmt.Errorf("got %+v, err %v", got, err)
			}
			return nil
		}},
		{"numeric", "YAML hex integer still decodes", 0, func() error {
			got, err := model.ParseIntoWithFormat[ctYInt]([]byte("v: 0x10\n"), model.FormatYAML)
			if err != nil || got.V != 16 {
				return fmt.Errorf("got %+v, err %v", got, err)
			}
			return nil
		}},
		{"numeric", "YAML maximum uint64 stays exact", 0, func() error {
			got, err := model.ParseIntoWithFormat[ctYU64]([]byte("v: 18446744073709551615\n"), model.FormatYAML)
			if err != nil || got.V != math.MaxUint64 {
				return fmt.Errorf("got %+v, err %v", got, err)
			}
			return nil
		}},
		{"numeric", "time rejects a huge number", 0, func() error { return wantError[ctTimeField](`{"at":1e300}`) }},
		{"numeric", "time rejects YAML NaN", 0, func() error { return wantYAMLError[ctTimeField]("at: .nan\n") }},
		{"numeric", "time rejects YAML infinity", 0, func() error { return wantYAMLError[ctTimeField]("at: .inf\n") }},
		{"numeric", "time keeps an ordinary Unix timestamp", 0, func() error {
			got, err := model.ParseInto[ctTimeField]([]byte(`{"at":1704067200}`))
			if err != nil || got.At.Unix() != 1704067200 {
				return fmt.Errorf("got %v, err %v", got.At, err)
			}
			return nil
		}},
		{"numeric", "YAML duration string still decodes", 0, func() error {
			got, err := model.ParseIntoWithFormat[ctYDuration]([]byte("d: 30s\n"), model.FormatYAML)
			if err != nil || got.D != 30*time.Second {
				return fmt.Errorf("got %+v, err %v", got, err)
			}
			return nil
		}},
		{"numeric", "YAML float does not truncate into a duration", 0, func() error {
			return wantYAMLError[ctYDuration]("d: 1.5\n")
		}},
		{"numeric", "YAML uintptr still decodes", 0, func() error {
			got, err := model.ParseIntoWithFormat[ctYUintptr]([]byte("p: 5\n"), model.FormatYAML)
			if err != nil || got.P != 5 {
				return fmt.Errorf("got %+v, err %v", got, err)
			}
			return nil
		}},
		{"numeric", "YAML NaN does not become a bool", 0, func() error { return wantYAMLError[ctYBool]("v: .nan\n") }},
		{"numeric", "an invalid json.Number is rejected", 0, func() error { return wantError[ctNumber](`{"n":"abc"}`) }},
		{"numeric", ",string rejects trailing junk in a number", 0, func() error { return wantError[ctQuotedInt](`{"a":"12 junk"}`) }},
		{"numeric", ",string rejects trailing junk in a bool", 0, func() error { return wantError[ctQuotedInt](`{"b":"true]"}`) }},
		{"numeric", ",string does not recurse into its contents", 0, func() error {
			return wantError[ctQuotedInt](`{"a":"` + strings.Repeat("[", 5<<20) + `"}`)
		}},
		{"numeric", "an invalid YAML json.Number is rejected", 0, func() error {
			if err := wantYAMLError[ctNumber]("n: abc\n"); err != nil {
				return err
			}
			return wantYAMLError[ctNumber]("n: true\n")
		}},
		{"numeric", "a YAML json.Number keeps a valid number", 0, func() error {
			got, err := model.ParseIntoWithFormat[ctNumber]([]byte("n: 12\n"), model.FormatYAML)
			if err != nil || got.N != "12" {
				return fmt.Errorf("got %+v, err %v", got, err)
			}
			return nil
		}},
		{"numeric", "a json.Number with trailing space is rejected", 0, func() error {
			if err := wantError[ctNumber](`{"n":"12 "}`); err != nil {
				return err
			}
			return wantError[ctNumber](`{"n":"12\n"}`)
		}},
		{"numeric", "a YAML json.Number keeps its digits", 0, func() error {
			for _, want := range []string{"123456789012345678901234567890", "1.10", "1e3"} {
				got, err := model.ParseIntoWithFormat[ctNumber]([]byte("n: "+want+"\n"), model.FormatYAML)
				if err != nil || string(got.N) != want {
					return fmt.Errorf("%s: got %+v, err %v", want, got, err)
				}
			}
			return nil
		}},
		{"numeric", "a YAML json.Number takes every number YAML reads", 0, func() error {
			for input, want := range map[string]string{"0x10": "16", "1_000": "1000", "0o17": "15", ".5": "0.5", "007": "7", "+5": "5"} {
				got, err := model.ParseIntoWithFormat[ctNumber]([]byte("n: "+input+"\n"), model.FormatYAML)
				if err != nil || string(got.N) != want {
					return fmt.Errorf("%s: got %+v, err %v", input, got, err)
				}
			}
			return wantYAMLError[ctNumber]("n: .inf\n")
		}},
		{"numeric", ",string rejects padding", 0, func() error { return wantError[ctQuotedInt](`{"a":" 12 "}`) }},
		{"numeric", "a time with no date is rejected", 0, func() error {
			return wantError[ctTimeField](`{"at":"12:30:00"}`)
		}},
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
		{"divergence", "null reaches an UnmarshalJSON method", 0, func() error {
			return wantSameAsJSON[ctNullField](`{"trigger":1,"o":null}`, `{"trigger":"1","o":null}`)
		}},
		{"divergence", "the ,string option", 0, func() error {
			const body = `"name":"\"x\"","n":"7","b":"true","f":"1.5"}`
			return wantSameAsJSON[ctQuoted](`{"trigger":1,`+body, `{"trigger":"1",`+body)
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
		{"yaml", "inline map behind a nil inline pointer", 0, func() error {
			got, err := model.ParseIntoWithFormat[ctYPtrInline]([]byte("name: a\nother: b\n"), model.FormatYAML)
			if err != nil || got.In == nil || got.In.Extra["other"] != "b" {
				return fmt.Errorf("got %+v, err %v", got, err)
			}
			return nil
		}},
		{"yaml", "unexported inline struct is ignored", 0, func() error {
			_, err := model.ParseIntoWithFormat[ctYHiddenInlineStruct]([]byte("name: a\n"), model.FormatYAML)
			return err
		}},
		{"yaml", "unexported inline map is ignored", 0, func() error {
			got, err := model.ParseIntoWithFormat[ctYHiddenInlineMap]([]byte("a: 1\nb: c\n"), model.FormatYAML)
			if err != nil || got.A != 1 {
				return fmt.Errorf("got %+v, err %v", got, err)
			}
			return nil
		}},
		{"yaml", "func-signature UnmarshalYAML runs", 0, func() error {
			got, err := model.ParseIntoWithFormat[ctOldYAMLField]([]byte("c: hi\n"), model.FormatYAML)
			if err != nil || got.C.Got != "hi" {
				return fmt.Errorf("got %+v, err %v", got, err)
			}
			return nil
		}},
		{"yaml", "yaml.Node field keeps the node", 0, func() error {
			got, err := model.ParseIntoWithFormat[ctYAMLNodeField]([]byte("cfg: {a: 1}\n"), model.FormatYAML)
			if err != nil || got.Cfg.Kind != yaml.MappingNode || len(got.Cfg.Content) != 2 {
				return fmt.Errorf("got kind %v with %d children, err %v", got.Cfg.Kind, len(got.Cfg.Content), err)
			}
			return nil
		}},
		{"yaml", "pointer to an inline map is ignored", 0, func() error {
			got, err := model.ParseIntoWithFormat[ctYPtrInlineMap]([]byte("name: x\nother: 1\n"), model.FormatYAML)
			if err != nil || got.Name != "x" {
				return fmt.Errorf("got %+v, err %v", got, err)
			}
			return nil
		}},
		{"yaml", "interface map keys decode", 0, func() error {
			got, err := model.ParseIntoWithFormat[ctYAnyKeys]([]byte("meta:\n  a: 1\n  2: b\n"), model.FormatYAML)
			if err != nil || got.Meta["a"] != 1 || got.Meta[2] != "b" {
				return fmt.Errorf("got %+v, err %v", got, err)
			}
			return nil
		}},
		{"yaml", "empty document leaves a self-decoding root alone", 0, func() error {
			for _, input := range []string{"", "# comment\n"} {
				if _, err := model.ParseIntoWithFormat[time.Time]([]byte(input), model.FormatYAML); err != nil {
					return fmt.Errorf("%q: %w", input, err)
				}
			}
			return nil
		}},
		{"yaml", "null keeps its yaml.Node", 0, func() error {
			got, err := model.ParseIntoWithFormat[ctYAMLNodeField]([]byte("cfg: null\n"), model.FormatYAML)
			if err != nil || got.Cfg.Kind != yaml.ScalarNode {
				return fmt.Errorf("got kind %v, err %v", got.Cfg.Kind, err)
			}
			return nil
		}},
		{"yaml", "inline map errors keep the field in their path", 0, func() error {
			_, err := model.ParseIntoWithFormat[ctYInlineItems]([]byte("k: {age: x}\n"), model.FormatYAML)
			var list model.ErrorList
			if !errors.As(err, &list) || len(list) != 1 {
				return fmt.Errorf("got %v", err)
			}
			var parseErr *model.ParseError
			if !errors.As(list[0], &parseErr) || parseErr.Field != "Extra[k].Age" {
				return fmt.Errorf("got %v", list[0])
			}
			return nil
		}},
		{"yaml", "unexported inline struct reports one fault", 0, func() error {
			_, err := model.ParseIntoWithFormat[ctYInlineEmbed]([]byte("age: x\n"), model.FormatYAML)
			var list model.ErrorList
			if !errors.As(err, &list) || len(list) != 1 {
				return fmt.Errorf("got %v", err)
			}
			return nil
		}},
		{"yaml", "interface map values still coerce", 0, func() error {
			got, err := model.ParseIntoWithFormat[ctAnyKeyInts]([]byte("meta:\n  a: \"12\"\n"), model.FormatYAML)
			if err != nil || got.Meta["a"] != 12 {
				return fmt.Errorf("got %+v, err %v", got, err)
			}
			return nil
		}},
		{"yaml", "interface map values honour json tags", 0, func() error {
			got, err := model.ParseIntoWithFormat[ctAnyKeyStructs]([]byte("m:\n  a: {user_name: bob}\n"), model.FormatYAML)
			if err != nil || got.M["a"].UserName != "bob" {
				return fmt.Errorf("got %+v, err %v", got, err)
			}
			return nil
		}},
		{"yaml", "interface map errors name the entry", 0, func() error {
			_, err := model.ParseIntoWithFormat[ctAnyKeyInts]([]byte("meta:\n  a: x\n"), model.FormatYAML)
			var parseErr *model.ParseError
			if !errors.As(err, &parseErr) || parseErr.Field != "Meta[a]" {
				return fmt.Errorf("got %v", err)
			}
			return nil
		}},
		{"yaml", "interface map takes a null key", 0, func() error {
			got, err := model.ParseIntoWithFormat[ctAnyKeyInts]([]byte("meta:\n  ~: 1\n"), model.FormatYAML)
			if err != nil || got.Meta[nil] != 1 {
				return fmt.Errorf("got %+v, err %v", got, err)
			}
			return nil
		}},
		{"yaml", "interface map rejects a key that cannot be hashed", 0, func() error {
			return wantYAMLError[ctAnyKeyInts]("meta:\n  ? [1, 2]\n  : 1\n")
		}},
		{"yaml", "typed interface keys report a failed entry once", 0, func() error {
			for _, key := range []string{"0x10", "~", "1.50"} {
				_, err := model.ParseIntoWithFormat[ctAnyKeyAges]([]byte("m:\n  "+key+": {age: x}\n"), model.FormatYAML)
				var list model.ErrorList
				if !errors.As(err, &list) || len(list) != 1 {
					return fmt.Errorf("%s: got %v", key, err)
				}
			}
			return nil
		}},
		{"yaml", "a later aliased key wins over an earlier key", 0, func() error {
			got, err := model.ParseIntoWithFormat[ctYKeyMap]([]byte("x: &k a\nattrs: {a: 1, *k : 2}\n"), model.FormatYAML)
			if err != nil || got.Attrs["a"] != 2 {
				return fmt.Errorf("got %+v, err %v", got, err)
			}
			return nil
		}},
		{"yaml", "a null key cannot go in a string map", 0, func() error {
			return wantYAMLError[ctYKeyMap]("attrs: {~: 1, a: 3}\n")
		}},
		{"yaml", "a null key cannot go in an inline map", 0, func() error {
			return wantYAMLError[ctYKeyInline]("name: x\n~: 1\n")
		}},
		{"yaml", "a null key is an unknown key on a struct", 0, func() error {
			got, err := model.ParseIntoWithFormat[ctYMerge]([]byte("~: 1\ntrigger: 2\n"), model.FormatYAML)
			if err != nil || got.Trigger != 2 {
				return fmt.Errorf("got %+v, err %v", got, err)
			}
			return nil
		}},
		{"yaml", "merged keys of two types cannot share an interface map", 0, func() error {
			return wantYAMLError[ctYAnyStrings]("b: &b {\"1\": str}\nm:\n  <<: *b\n  1: int\n")
		}},
		{"yaml", "an explicit key still beats a merged one in a string map", 0, func() error {
			got, err := model.ParseIntoWithFormat[ctYStrings]([]byte("b: &b {\"1\": str}\nm:\n  <<: *b\n  1: int\n"), model.FormatYAML)
			if err != nil || got.M["1"] != "int" {
				return fmt.Errorf("got %+v, err %v", got, err)
			}
			return nil
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
		{"validation", "input cannot set a json:\"-\" field, so its rule waits", 0, func() error {
			return wantValue(`{}`, ctDashRule{})
		}},
		{"validation", "the coercion path agrees", 0, func() error {
			return wantValue(`{"n":"1"}`, ctDashCoerced{N: 1})
		}},
		{"validation", "Validate runs the rule on a json:\"-\" field", 0, func() error {
			if err := model.Validate(&ctDashRule{}); err == nil {
				return errors.New("empty field passed required")
			}
			return model.Validate(&ctDashRule{A: "set by the caller"})
		}},
		{"validation", "a field no decoder sets is treated alike", 0, func() error {
			if err := wantValue(`{}`, ctDashBoth{}); err != nil {
				return err
			}
			if err := model.Validate(&ctDashBoth{}); err == nil {
				return errors.New("empty field passed required")
			}
			return nil
		}},
		{"validation", "input does not choose the format that skips a rule", 0, func() error {
			if err := wantError[ctDashYAMLRule](`{}`); err != nil {
				return fmt.Errorf("JSON input: %w", err)
			}
			if _, err := model.ParseIntoWithFormat[ctDashYAMLRule]([]byte("b: 1\n"), model.FormatYAML); err == nil {
				return errors.New("YAML input: no error")
			}
			_, err := model.ParseIntoWithFormat[ctDashYAMLRule]([]byte("a: x\n"), model.FormatYAML)
			return err
		}},
		{"validation", "a field only JSON sets keeps its rule on YAML", 0, func() error {
			if _, err := model.ParseIntoWithFormat[ctDashYAMLOff]([]byte("b: 1\n"), model.FormatYAML); err == nil {
				return errors.New("no error")
			}
			return nil
		}},
		{"validation", "rules inside a struct only YAML sets", 0, func() error {
			_, err := model.ParseIntoWithFormat[ctDashNested]([]byte("inner: {name: \"\"}\n"), model.FormatYAML)
			if err == nil {
				return errors.New("no error")
			}
			return nil
		}},
		{"validation", "rules inside a slice only YAML sets", 0, func() error {
			_, err := model.ParseIntoWithFormat[ctDashList]([]byte("items: [{name: \"\"}]\n"), model.FormatYAML)
			if err == nil {
				return errors.New("no error")
			}
			return nil
		}},
		{"validation", "rules inside an inline struct", 0, func() error {
			_, err := model.ParseIntoWithFormat[ctDashInline]([]byte("name: \"\"\n"), model.FormatYAML)
			if err == nil {
				return errors.New("no error")
			}
			return nil
		}},
		{"validation", "Validate stops at a back-pointer", 0, func() error {
			root := &ctDashTree{Name: "root"}
			root.Children = []*ctDashTree{{Name: "child", Parent: root}}
			return model.Validate(root)
		}},
		{"validation", "hidden embedded struct waits for Validate", 0, func() error {
			if err := wantValue(`{"n":1}`, ctDashHiddenEmbed{N: 1}); err != nil {
				return err
			}
			if err := model.Validate(&ctDashHiddenEmbed{}); err == nil {
				return errors.New("embedded rule did not run")
			}
			return nil
		}},
		{"validation", "a map loop holding no struct does not overflow the stack", 0, func() error {
			loop := ctLoopMap{}
			loop["a"] = loop
			return model.Validate(&ctLoopHolder{ID: 1, Kids: loop})
		}},
		{"validation", "a slice loop holding no struct does not overflow the stack", 0, func() error {
			loop := ctLoopSlice{nil}
			loop[0] = loop
			return model.Validate(&ctLoopSliceHolder{ID: 1, Kids: loop})
		}},
		{"validation", "NaN fails required on its own", 0, func() error {
			if err := model.Validate(&ctRequiredFloat{V: math.NaN()}); err == nil {
				return errors.New("NaN passed required")
			}
			return wantYAMLError[ctRequiredFloat]("v: .nan\n")
		}},
		{"validation", "CoerceValue runs rules on a struct target", 0, func() error {
			_, err := model.CoerceValue(map[string]interface{}{"age": 3}, reflect.TypeOf(ctCoerceTarget{}), "u")
			if err == nil {
				return errors.New("age 3 passed min=18")
			}
			return nil
		}},
		{"validation", "cross-field rule in an unexported embed can read its struct", 0, func() error {
			if _, err := model.ParseInto[ctEmbedCross]([]byte(`{"a":"x"}`)); err != nil {
				return err
			}
			if err := model.Validate(&ctEmbedCross{ctEmbedCrossBase{A: "x"}}); err != nil {
				return err
			}
			return nil
		}},
		{"validation", "cross-field rule in an embed reads the embed, not a shadowing field", 0, func() error {
			if _, err := model.ParseInto[ctShadowOuter]([]byte(`{"pass":"a","confirm":"a","outer_pass":"z"}`)); err != nil {
				return err
			}
			if _, err := model.ParseInto[ctShadowOuter]([]byte(`{"pass":"a","confirm":"b","outer_pass":"b"}`)); err == nil {
				return errors.New("mismatch passed")
			}
			return nil
		}},
		{"validation", "cross-field rule sees an embedded pointer in its embed", 0, func() error {
			outer := ctPtrEmbedOuter{ctPtrEmbedRule{&ctInnerPass{Pass: "a"}, "a"}}
			return model.Validate(&outer)
		}},
		{"validation", "cross-field rule sees an unexported sibling in its embed", 0, func() error {
			outer := ctSiblingOuter{ctSiblingRule{"s", "s"}}
			return model.Validate(&outer)
		}},
		{"validation", "cross-field rule in an embed under a map value", 0, func() error {
			_, err := model.ParseInto[map[string]ctEmbedCross]([]byte(`{"k":{"a":"x"}}`))
			return err
		}},
		{"validation", "a NaN map key cannot hide its entry from the rules", 0, func() error {
			return wantYAMLError[ctAnyKeyItems]("m: {.nan: {}}\n")
		}},
		{"validation", "rules behind an unexported embedded pointer run", 0, func() error {
			if err := model.Validate(&ctEmbedPtr{&ctEmbedPtrBase{}}); err == nil {
				return errors.New("zero X passed required")
			}
			return model.Validate(&ctEmbedPtr{})
		}},
		{"validation", "a root of the wrong shape reports one fault", 0, func() error {
			_, err := model.ParseInto[ctItem]([]byte(`[1]`))
			var list model.ErrorList
			if !errors.As(err, &list) || len(list) != 1 {
				return fmt.Errorf("got %v", err)
			}
			return nil
		}},
		{"validation", "CoerceValue keeps rules on the other fields", 0, func() error {
			_, err := model.CoerceValue(map[string]interface{}{"age": "x"}, reflect.TypeOf(ctCoerceTwo{}), "u")
			var list model.ErrorList
			if !errors.As(err, &list) || len(list) != 2 || len(list.ValidationErrors()) != 1 {
				return fmt.Errorf("got %v", err)
			}
			return nil
		}},
		{"validation", "CoerceValue reports a failed field once", 0, func() error {
			_, err := model.CoerceValue(map[string]interface{}{"age": "x"}, reflect.TypeOf(ctCoerceTarget{}), "u")
			var list model.ErrorList
			if errors.As(err, &list) && len(list) != 1 {
				return fmt.Errorf("got %v", err)
			}
			return nil
		}},
		{"validation", "exactly 100 errors are not called too many", 0, func() error {
			_, err := model.ParseInto[[]ctItem]([]byte(`[` + strings.TrimSuffix(strings.Repeat(`{},`, 100), ",") + `]`))
			var list model.ErrorList
			if !errors.As(err, &list) || len(list) != 100 {
				return fmt.Errorf("got %d errors", len(list))
			}
			return nil
		}},
		{"validation", "101 errors keep 100 and say so", 0, func() error {
			_, err := model.ParseInto[[]ctItem]([]byte(`[` + strings.TrimSuffix(strings.Repeat(`{},`, 101), ",") + `]`))
			var list model.ErrorList
			if !errors.As(err, &list) || len(list) != 101 || !strings.Contains(list[100].Error(), "too many errors") {
				return fmt.Errorf("got %d errors", len(list))
			}
			return nil
		}},
		{"validation", "a depth of 0 cannot turn the limit off", 0, func() error {
			orig := model.GetMaxValidationDepth()
			model.SetMaxValidationDepth(0)
			defer model.SetMaxValidationDepth(orig)
			loop := &ctLoopRule{Name: "x"}
			loop.Next = loop
			err := model.Validate(loop)
			if err == nil || !strings.Contains(err.Error(), "depth") {
				return fmt.Errorf("got %v", err)
			}
			return nil
		}},
		{"validation", "a negative cache size does not panic", 0, func() error {
			orig := model.GetMaxCacheSize()
			model.SetMaxCacheSize(-1)
			defer model.SetMaxCacheSize(orig)
			model.ClearValidationCache()
			return wantValue(`{"a":"x"}`, ctNegativeCache{A: "x"})
		}},
		{"validation", "required fails on a zero fixed array", 0, func() error {
			if err := wantError[ctRequiredArray](`{"x":1}`); err != nil {
				return err
			}
			_, err := model.ParseInto[ctRequiredArray]([]byte(`{"h":[1,0,0,0]}`))
			return err
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
		{"tags", "bad tag on a json:\"-\" field", 0, func() error { return wantTagError[ctDashTypo](`{}`) }},
		{"tags", "bad tag behind a field only YAML sets", 0, func() error { return wantTagError[ctDashYAMLTypo](`{}`) }},
		{"tags", "foreign tags behind a field no decoder sets", 0, func() error {
			if err := wantValue(`{"name":"x"}`, ctDashForeign{Name: "x"}); err != nil {
				return err
			}
			return model.Validate(&ctDashForeign{Owner: &ctForeign{}})
		}},
		{"tags", "json:\"-\" field with no rule stays allowed", 0, func() error {
			return wantValue(`{"b":"x"}`, ctDashNoRule{B: "x"})
		}},
		{"tags", "rule on the key named by json:\"-,\" runs", 0, func() error { return wantError[ctDashKey](`{"other":1}`) }},
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
		{"panics", "CachedParser of an interface type", 0, func() error {
			cached := model.NewCachedParser[any](nil)
			cached.Close()
			return nil
		}},
		{"panics", "CachedParser closed twice", 0, func() error {
			cached := model.NewCachedParser[ctItem](nil)
			cached.Close()
			cached.Close()
			return nil
		}},

		// Cache isolation.
		{"cache", "mutating a hit does not reach the next caller", 0, func() error {
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
		{"cache", "mutating a miss does not reach the next caller", 0, func() error {
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
		{"errors", "a json:\"-\" value stays out of the structured report", 0, func() error {
			err := model.Validate(&ctDashHash{Hash: "abc"})
			var list model.ErrorList
			if !errors.As(err, &list) || len(list.ValidationErrors()) != 1 {
				return fmt.Errorf("unexpected error %v", err)
			}
			if got := list.ValidationErrors()[0].SanitizedValue(); got != model.RedactedValue {
				return fmt.Errorf("SanitizedValue returned %v", got)
			}
			if got := list.ValidationErrors()[0].Value; got != "abc" {
				return fmt.Errorf("Value is %v", got)
			}
			report, jsonErr := list.ToJSON()
			if jsonErr != nil || strings.Contains(string(report), "abc") {
				return fmt.Errorf("report %s, err %v", report, jsonErr)
			}
			return nil
		}},
		{"errors", "a value under a json:\"-\" parent is redacted", 0, func() error {
			_, err := model.ParseIntoWithFormat[ctDashHashParent]([]byte("inner: {code: abc}\n"), model.FormatYAML)
			var list model.ErrorList
			if !errors.As(err, &list) || len(list.ValidationErrors()) != 1 {
				return fmt.Errorf("unexpected error %v", err)
			}
			if got := list.ValidationErrors()[0].SanitizedValue(); got != model.RedactedValue {
				return fmt.Errorf("SanitizedValue returned %v", got)
			}
			return nil
		}},
		{"errors", "a json:\"-\" value stays out of a YAML ParseError", 0, func() error {
			_, err := model.ParseIntoWithFormat[ctDashYAMLInt]([]byte("n: not-a-number-4321\n"), model.FormatYAML)
			if err == nil || strings.Contains(err.Error(), "4321") {
				return fmt.Errorf("got %v", err)
			}
			return nil
		}},
		{"errors", "a value in a hidden embedded struct is redacted", 0, func() error {
			if err := wantValidationRedacted(model.Validate(&ctDashHiddenEmbedHash{ctDashEmbedHash{Code: "abc"}})); err != nil {
				return fmt.Errorf("Validate: %w", err)
			}
			_, err := model.ParseIntoWithFormat[ctDashHiddenEmbedHash]([]byte("code: abc\n"), model.FormatYAML)
			if err := wantValidationRedacted(err); err != nil {
				return fmt.Errorf("YAML: %w", err)
			}
			return nil
		}},
		{"errors", "a hidden embedded path renders without a segment", 0, func() error {
			err := model.Validate(&ctDashHiddenEmbedHash{ctDashEmbedHash{Code: "abc"}})
			var list model.ErrorList
			if !errors.As(err, &list) || len(list.ValidationErrors()) != 1 {
				return fmt.Errorf("unexpected error %v", err)
			}
			if got := list.ValidationErrors()[0].FieldPath; got != "Code" {
				return fmt.Errorf("FieldPath %q", got)
			}
			return nil
		}},
		{"errors", "a value in a hidden inline struct stays out of ParseError", 0, func() error {
			_, err := model.ParseIntoWithFormat[ctDashInlineInt]([]byte("n: not-a-number-4321\n"), model.FormatYAML)
			if err == nil || strings.Contains(err.Error(), "4321") {
				return fmt.Errorf("got %v", err)
			}
			return nil
		}},
		{"errors", "a value in a hidden inline map stays out of ParseError", 0, func() error {
			_, err := model.ParseIntoWithFormat[ctDashInlineMap]([]byte("other: not-a-number-4321\n"), model.FormatYAML)
			if err == nil || strings.Contains(err.Error(), "4321") {
				return fmt.Errorf("got %v", err)
			}
			return nil
		}},
		{"errors", "an ordinary validation value is not redacted", 0, func() error {
			err := model.Validate(&ctBounded{V: 500})
			var list model.ErrorList
			if !errors.As(err, &list) || len(list.ValidationErrors()) != 1 {
				return fmt.Errorf("unexpected error %v", err)
			}
			if got := list.ValidationErrors()[0].SanitizedValue(); got != float64(500) {
				return fmt.Errorf("SanitizedValue returned %v", got)
			}
			return nil
		}},
		{"errors", "errors.As reaches a ParseError inside the list", 0, func() error {
			_, err := model.ParseInto[ctInt]([]byte(`{"v":"x"}`))
			var parseErr *model.ParseError
			if !errors.As(err, &parseErr) {
				return fmt.Errorf("got %T: %v", err, err)
			}
			return nil
		}},
		{"errors", "errors.As reaches a ValidationError inside the list", 0, func() error {
			_, err := model.ParseInto[ctItem]([]byte(`{"name":""}`))
			var validationErr *model.ValidationError
			if !errors.As(err, &validationErr) {
				return fmt.Errorf("got %T: %v", err, err)
			}
			return nil
		}},
		{"errors", "a sensitive JSON key redacts a validation value", 0, func() error {
			return wantValidationRedacted(model.Validate(&ctKeySensitive{Code: "hunter2"}))
		}},
		{"errors", "a sensitive YAML key redacts a validation value", 0, func() error {
			return wantValidationRedacted(model.Validate(&ctKeySensitiveYAML{Code: "hunter2"}))
		}},
		{"errors", "a sensitive parent key redacts a validation value", 0, func() error {
			var v ctKeySensitiveParent
			v.Inner.Code = "hunter2"
			return wantValidationRedacted(model.Validate(&v))
		}},
		{"errors", "a sensitive inline map is redacted", 0, func() error {
			_, err := model.ParseIntoWithFormat[ctYInlineSecrets]([]byte("foo: hunter2\n"), model.FormatYAML)
			if err == nil || strings.Contains(err.Error(), "hunter2") {
				return fmt.Errorf("got %v", err)
			}
			var parseErr *model.ParseError
			if !errors.As(err, &parseErr) || parseErr.SanitizedValue() != model.RedactedValue {
				return fmt.Errorf("got %v", err)
			}
			return nil
		}},
		{"errors", "a sensitive key in an interface map is redacted", 0, func() error {
			_, err := model.ParseIntoWithFormat[ctAnyKeyInts]([]byte("meta:\n  password: hunter2\n"), model.FormatYAML)
			if err == nil || strings.Contains(err.Error(), "hunter2") {
				return fmt.Errorf("got %v", err)
			}
			return nil
		}},
		{"errors", "a rejected object does not carry its contents", 0, func() error {
			_, err := model.ParseInto[ctCfgString]([]byte(`{"cfg":{"password":"hunter2"}}`))
			var parseErr *model.ParseError
			if !errors.As(err, &parseErr) {
				return fmt.Errorf("got %v", err)
			}
			if got := fmt.Sprint(parseErr.SanitizedValue()); strings.Contains(got, "hunter2") {
				return fmt.Errorf("SanitizedValue returned %s", got)
			}
			return nil
		}},
		{"errors", "a failed rule on a container keeps its contents out of the report", 0, func() error {
			_, err := model.ParseInto[ctPwUsers]([]byte(`{"users":[{"name":"a","password":"hunter2"}]}`))
			var list model.ErrorList
			if !errors.As(err, &list) {
				return fmt.Errorf("got %v", err)
			}
			report, jsonErr := list.ToJSON()
			if jsonErr != nil || strings.Contains(string(report), "hunter2") {
				return fmt.Errorf("report %s, err %v", report, jsonErr)
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

	model.RegisterGlobalCrossFieldFunc("ct_same_pass", func(field string, value interface{}, parent reflect.Value, _ map[string]interface{}) error {
		if parent.Interface() == nil || parent.FieldByName("Pass").String() != value.(string) {
			return model.NewValidationError(field, value, "ct_same_pass", "fields differ")
		}
		return nil
	})
	model.RegisterGlobalCrossFieldFunc("ct_same_secret", func(field string, value interface{}, parent reflect.Value, _ map[string]interface{}) error {
		if parent.FieldByName("secret").String() != value.(string) {
			return model.NewValidationError(field, value, "ct_same_secret", "fields differ")
		}
		return nil
	})
	model.RegisterGlobalCrossFieldFunc("ct_cross_iface", func(field string, value interface{}, parent reflect.Value, _ map[string]interface{}) error {
		if parent.Interface() == nil || parent.FieldByName("A").String() != value.(string) {
			return model.NewValidationError(field, value, "ct_cross_iface", "cannot read the struct")
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
