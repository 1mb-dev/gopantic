package model

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"
)

// Performance optimization caches
var (
	// fieldKeyCache stores resolved struct fields per type and format
	fieldKeyCache sync.Map // map[fieldKeyCacheKey]*fieldTable
)

// Note: validationCache is declared in validate.go

// fieldKeyCacheKey is the cache key for field name lookups
type fieldKeyCacheKey struct {
	typ    reflect.Type
	format Format
}

// MaxInputSize is the default maximum size for input data (10MB).
// Set to 0 to disable size checking. This prevents resource exhaustion
// from maliciously large inputs.
//
// WARNING: Direct modification of this variable is NOT thread-safe.
// For concurrent access, use GetMaxInputSize() and SetMaxInputSize().
var MaxInputSize = 10 * 1024 * 1024 // 10MB

// MaxStructureDepth is the default maximum nesting depth for parsed structures (64).
// Set to 0 to disable depth checking. This prevents resource exhaustion
// from deeply nested JSON/YAML structures.
//
// WARNING: Direct modification of this variable is NOT thread-safe.
// For concurrent access, use GetMaxStructureDepth() and SetMaxStructureDepth().
var MaxStructureDepth = 64

// ParseInto parses raw data into a struct of type T with automatic format detection, type coercion, and validation.
// The format is automatically detected (JSON or YAML) based on the content structure.
// This is the main entry point for parsing operations in gopantic.
//
// The function checks input size against MaxInputSize (default 10MB) to prevent resource exhaustion.
// Set MaxInputSize to 0 to disable size checking.
//
// Example:
//
//	type User struct {
//	    ID   int    `json:"id" validate:"required,min=1"`
//	    Name string `json:"name" validate:"required,min=2"`
//	}
//
//	user, err := model.ParseInto[User](jsonData)
//	if err != nil {
//	    log.Fatal(err)
//	}
func ParseInto[T any](raw []byte) (T, error) {
	// Check input size
	var zero T
	maxSize := GetMaxInputSize()
	if maxSize > 0 && len(raw) > maxSize {
		return zero, fmt.Errorf("input size %d bytes exceeds maximum allowed size %d bytes", len(raw), maxSize)
	}

	// Auto-detect format and use appropriate parser
	format := DetectFormat(raw)
	return ParseIntoWithFormat[T](raw, format)
}

// ParseIntoWithFormat parses raw data of a specific format into a struct of type T with type coercion and validation.
// Use this when you know the exact format or want to enforce a specific format.
// Supports JSON and YAML formats.
//
// The function checks input size against MaxInputSize (default 10MB) to prevent resource exhaustion.
// Set MaxInputSize to 0 to disable size checking.
//
// Example:
//
//	user, err := model.ParseIntoWithFormat[User](yamlData, model.FormatYAML)
//	if err != nil {
//	    log.Fatal(err)
//	}
func ParseIntoWithFormat[T any](raw []byte, format Format) (T, error) {
	var zero T

	// Check input size
	maxSize := GetMaxInputSize()
	if maxSize > 0 && len(raw) > maxSize {
		return zero, fmt.Errorf("input size %d bytes exceeds maximum allowed size %d bytes", len(raw), maxSize)
	}

	switch format {
	case FormatJSON:
		return parseJSON[T](raw)
	case FormatYAML:
		return parseYAML[T](raw)
	default:
		return zero, fmt.Errorf("unsupported format: %v", format)
	}
}

// getFieldKey extracts the appropriate field key based on the data format
func getFieldKey(field reflect.StructField, format Format) string {
	var tagName string

	// Determine which tag to use based on format
	switch format {
	case FormatYAML:
		tagName = "yaml"
	default:
		tagName = "json"
	}

	tag := field.Tag.Get(tagName)
	if tag == "" {
		// Fallback to json tag if yaml tag is not present
		if tagName == "yaml" {
			tag = field.Tag.Get("json")
		}

		// If still empty, use field name
		if tag == "" {
			return field.Name
		}
	}

	// Handle tag options like "name,omitempty"
	if tag == "-" {
		return "-"
	}

	// Split on comma and take first part (the name)
	for i, char := range tag {
		if char == ',' {
			return tag[:i]
		}
	}

	return tag
}

// Validate validates a struct using gopantic validation rules defined in struct tags.
// This function can be used independently of parsing, allowing you to validate
// structs that were populated from any source (JSON, YAML, database, environment variables, etc.).
//
// Example:
//
//	type User struct {
//	    ID   int    `json:"id" validate:"required,min=1"`
//	    Name string `json:"name" validate:"required,min=2"`
//	}
//
//	user := User{ID: 42, Name: "Alice"}
//	if err := model.Validate(&user); err != nil {
//	    log.Fatal(err)
//	}
func Validate[T any](v *T) error {
	if v == nil {
		return fmt.Errorf("Validate: nil pointer provided")
	}

	val := reflect.ValueOf(v).Elem()
	typ := val.Type()

	if typ.Kind() != reflect.Struct {
		return fmt.Errorf("Validate: expected struct, got %v", typ.Kind())
	}

	return validationWalk{populated: true}.run(val)
}

// validateParsed validates a decoded value of any kind, continuing errs from decoding. Paths
// in skip failed to decode, so their zero values are not reported a second time.
func validateParsed(val reflect.Value, errs ErrorList, skip map[string]bool) error {
	return validationWalk{errs: errs, skip: skip}.run(val)
}

// maxPathKey bounds a map key inside a rendered path.
const maxPathKey = 64

// fieldPath is a position in a decoded value, rendered only when an error needs it. Building
// strings during a walk would copy every enclosing map key once per element, which turns one
// long key above a large slice into quadratic work.
type fieldPath struct {
	parent *fieldPath
	name   string
	key    string
	index  int
	kind   pathKind
}

type pathKind int

const (
	pathField pathKind = iota
	pathIndex
	pathKey
)

func (p *fieldPath) String() string {
	var segments []*fieldPath
	for ; p != nil; p = p.parent {
		segments = append(segments, p)
	}

	var b strings.Builder
	for i := len(segments) - 1; i >= 0; i-- {
		switch segment := segments[i]; segment.kind {
		case pathField:
			if b.Len() > 0 {
				b.WriteByte('.')
			}
			b.WriteString(segment.name)
		case pathIndex:
			fmt.Fprintf(&b, "[%d]", segment.index)
		case pathKey:
			fmt.Fprintf(&b, "[%s]", boundedKey(segment.key))
		}
	}
	return b.String()
}

// boundedKey keeps input-sized map keys out of error text and sensitive-field matching, which
// run once per error.
func boundedKey(key string) string {
	if len(key) > maxPathKey {
		return strings.ToValidUTF8(key[:maxPathKey], "") + "..."
	}
	return key
}

// sensitive reports whether any segment names a sensitive field, by Go name or input key.
func (p *fieldPath) sensitive() bool {
	for ; p != nil; p = p.parent {
		if IsSensitiveField(p.name) || IsSensitiveField(boundedKey(p.key)) {
			return true
		}
	}
	return false
}

// failedPaths returns the paths decoding could not set.
func failedPaths(errs ErrorList) map[string]bool {
	if len(errs) == 0 {
		return nil
	}
	paths := make(map[string]bool, len(errs))
	for _, err := range errs {
		if parseErr, ok := err.(*ParseError); ok {
			paths[parseErr.Field] = true
		}
	}
	return paths
}

// validationWalk is the single validation path: every parse and Validate go through it, so
// the rules that run cannot depend on how the value was decoded.
type validationWalk struct {
	errs ErrorList
	skip map[string]bool

	// populated marks a value the caller filled in, so rules on fields no decoder sets apply.
	// Their containers stay closed: a back-pointer would cycle, and a foreign type's tags are
	// not ours to check.
	populated bool
}

func (w validationWalk) run(val reflect.Value) error {
	validation := ParseValidationTags(val.Type())
	if len(validation.TagErrors) > 0 {
		for _, err := range validation.TagErrors {
			w.errs.addCapped(err)
		}
		return w.errs.AsError()
	}
	if !validation.holdsRules {
		return w.errs.AsError()
	}

	w.value(val, nil, 0)
	return w.errs.AsError()
}

func (w *validationWalk) value(val reflect.Value, path *fieldPath, depth int) {
	if w.errs.full() {
		return
	}

	switch val.Kind() {
	case reflect.Struct:
		if validation := ParseValidationTags(val.Type()); validation.holdsRules {
			w.structFields(val, validation, path, depth)
		}
	case reflect.Ptr:
		if !val.IsNil() {
			w.value(val.Elem(), path, depth)
		}
	case reflect.Slice, reflect.Array:
		if !mayHoldRules(val.Type().Elem()) {
			return
		}
		for i := 0; i < val.Len() && !w.errs.full(); i++ {
			w.value(val.Index(i), &fieldPath{parent: path, kind: pathIndex, index: i}, depth)
		}
	case reflect.Map:
		if mayHoldRules(val.Type().Elem()) {
			w.mapValues(val, path, depth)
		}
	}
}

// mapValues visits entries in key order, so which errors survive the cap is stable.
func (w *validationWalk) mapValues(val reflect.Value, path *fieldPath, depth int) {
	type entry struct {
		label string
		key   reflect.Value
	}
	entries := make([]entry, 0, val.Len())
	for _, key := range val.MapKeys() {
		entries = append(entries, entry{label: fmt.Sprint(key.Interface()), key: key})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].label < entries[j].label })

	for i := 0; i < len(entries) && !w.errs.full(); i++ {
		segment := &fieldPath{parent: path, kind: pathKey, key: entries[i].label}
		w.value(val.MapIndex(entries[i].key), segment, depth)
	}
}

// mayHoldRules reports whether a container's elements can carry validation tags.
func mayHoldRules(elem reflect.Type) bool {
	switch elem.Kind() {
	case reflect.Struct, reflect.Ptr, reflect.Slice, reflect.Array, reflect.Map:
		return true
	default:
		return false
	}
}

func (w *validationWalk) structFields(val reflect.Value, validation *StructValidation, path *fieldPath, depth int) {
	if maxDepth := GetMaxValidationDepth(); maxDepth > 0 && depth > maxDepth {
		w.errs.addCapped(fmt.Errorf("validation depth exceeded maximum of %d levels", maxDepth))
		return
	}

	for i := 0; i < val.NumField() && !w.errs.full(); i++ {
		w.field(val, i, validation, path, depth)
	}
}

func (w *validationWalk) field(parent reflect.Value, i int, validation *StructValidation, path *fieldPath, depth int) {
	field := parent.Type().Field(i)
	fieldVal := parent.Field(i)

	decoded := validation.decodable[i]
	if !decoded && !w.populated {
		return
	}

	// The exported fields of an unexported embedded struct stay reachable.
	if field.Anonymous && !field.IsExported() && fieldVal.Kind() == reflect.Struct {
		w.value(fieldVal, path, depth+1)
		return
	}

	rules := rulesFor(validation, field.Name)
	walkable := enters(field, decoded)
	if len(rules) == 0 && !walkable && w.skip == nil {
		return
	}
	if !fieldVal.CanInterface() {
		return
	}

	here := &fieldPath{parent: path, kind: pathField, name: field.Name}
	if w.skip != nil && w.skip[here.String()] {
		return
	}
	if walkable {
		w.value(fieldVal, here, depth+1)
	}

	if len(rules) > 0 {
		err := ValidateValueWithStruct(field.Name, fieldVal.Interface(), rules, parent)
		if err != nil {
			w.errs.addCapped(withFieldPath(err, here.String()))
		}
	}
}

// enters reports whether the walk descends into a field. An embedded struct counts as the
// parent's own fields, whatever its tag says.
func enters(field reflect.StructField, decoded bool) bool {
	return (decoded || field.Anonymous) && mayHoldRules(field.Type)
}

func rulesFor(validation *StructValidation, fieldName string) []ValidationRule {
	for i := range validation.Fields {
		if validation.Fields[i].FieldName == fieldName {
			return validation.Fields[i].Rules
		}
	}
	return nil
}

// withFieldPath sets the full path on validation errors. Field stays the leaf name.
func withFieldPath(err error, path string) error {
	switch e := err.(type) {
	case *ValidationError:
		located := *e
		located.FieldPath = path
		return &located
	case ErrorList:
		located := make(ErrorList, len(e))
		for i, inner := range e {
			located[i] = withFieldPath(inner, path)
		}
		return located
	default:
		return err
	}
}
