# API

This page groups the package by job. [pkg.go.dev](https://pkg.go.dev/github.com/1mb-dev/gopantic/pkg/model) has every symbol.

## Parsing

```go
func ParseInto[T any](raw []byte) (T, error)
func ParseIntoWithFormat[T any](raw []byte, format Format) (T, error)
func DetectFormat(raw []byte) Format
```

`T` can be a struct, a pointer, a slice, an array or a map. On error the returned value is the zero value. `format` is `FormatJSON` or `FormatYAML`. `DetectFormat` calls input starting with `{` or `[` JSON, and otherwise looks for YAML markers, falling back to JSON.

## Validating

```go
func Validate[T any](v *T) error
```

Runs the rules on a struct you already hold. Unlike a parse, it also runs the rules on fields no decoder can set; see [Fields input cannot set](../guide/validation.md#fields-input-cannot-set).

## Rules

```go
func RegisterGlobalFunc(name string, fn ValidatorFunc)
func RegisterGlobalCrossFieldFunc(name string, fn CrossFieldValidatorFunc)
func GetDefaultRegistry() *ValidatorRegistry
func ParseValidationTags(typ reflect.Type) *StructValidation
func ClearValidationCache()

type ValidatorFunc func(fieldName string, value interface{}, params map[string]interface{}) error
type CrossFieldValidatorFunc func(fieldName string, fieldValue interface{}, structValue reflect.Value, params map[string]interface{}) error
```

Registering on the default registry, by these functions or its own `Register`, `RegisterFunc` and `RegisterCrossFieldFunc` methods, is safe at any time and clears cached rules. `ClearValidationCache` is only needed if something else changed a type's rules, which ordinary code doesn't do.

`ParseValidationTags` returns the rules for a type. Its `TagErrors` field lists the tags that can't run, for this type and every type it reaches, so a test can fail on them before any input arrives:

```go
func TestTags(t *testing.T) {
    if errs := model.ParseValidationTags(reflect.TypeOf(Order{})).TagErrors; len(errs) > 0 {
        t.Fatal(errs)
    }
}
```

## Coercing one value

```go
func CoerceValue(value interface{}, target reflect.Type, fieldName string) (interface{}, error)
func CoerceValueWithFormat(value interface{}, target reflect.Type, fieldName string, format Format) (interface{}, error)
```

Applies the [coercion rules](../guide/coercion.md) to a single value. Integer kinds come back as `int64` or `uint64`, and floats as `float64`, checked against the range of `target`.

## Caching

```go
func NewCachedParser[T any](config *CacheConfig) *CachedParser[T]
func DefaultCacheConfig() *CacheConfig

func (cp *CachedParser[T]) Parse(data []byte) (T, error)
func (cp *CachedParser[T]) ParseWithFormat(data []byte, format Format) (T, error)
func (cp *CachedParser[T]) Stats() (size, maxSize int, hitRate float64)
func (cp *CachedParser[T]) ClearCache()
func (cp *CachedParser[T]) Close()
```

See the [caching guide](../guide/caching.md).

## Errors

```go
type ParseError struct {
    Field, Type, Message string
    Value                interface{}
}

type ValidationError struct {
    Field     string // Leaf name: "Name"
    FieldPath string // Full path: "Items[1].Name"
    Value     interface{}
    Rule      string
    Message   string
    Details   map[string]interface{}
}
```

Both have `SanitizedValue()`. A failed parse or `Validate` returns an `ErrorList` holding them, plus plain errors for tags that can't run or a depth limit. `ErrorList` works with `errors.As` and `errors.Is`, and has `ValidationErrors`, `GroupByField`, `ToStructuredReport` and `ToJSON`. Syntax errors and oversized input come back as plain errors.

Custom rules build their errors with `NewValidationError`, or `NewValidationErrorWithDetails` to attach `Details`.

## Limits and settings

| Setting | Default | Meaning |
|---------|---------|---------|
| `SetMaxInputSize` | 10 MB | Larger input is rejected; 0 turns the check off |
| `SetMaxStructureDepth` | 64 | Deeper input is rejected; 0 turns the check off |
| `SetMaxYAMLKeyComparisons` | 50,000,000 | YAML needing more key comparisons is rejected; 0 turns the budget off |
| `SetMaxValidationDepth` | 32 | Rules reached deeper than this are an error |
| `SetMaxCacheSize` | 1000 | Types whose rules stay cached; 0 is unlimited |

Each has a matching getter. They are safe for concurrent use. The package variables of the same names aren't, so use these.

`SetSensitiveFieldPatterns`, `AddSensitiveFieldPattern`, `GetSensitiveFieldPatterns` and `IsSensitiveField` control [sensitive values](../guide/validation.md#sensitive-values).
