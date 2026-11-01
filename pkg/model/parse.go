package model

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// Performance optimization caches
var (
	// fieldKeyCache stores pre-computed field keys per type and format
	fieldKeyCache sync.Map // map[fieldKeyCacheKey]map[int]string
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

	// Check structure depth to prevent resource exhaustion from deeply nested input
	if err := checkRawStructureDepth(raw, format); err != nil {
		return zero, err
	}

	// Strategy: Try standard unmarshal first (handles json.RawMessage, custom UnmarshalJSON, etc.)
	// If that succeeds, apply selective coercion only where needed
	// If it fails (due to type mismatches), fall back to map-based coercion

	var result T
	unmarshalErr := unmarshalByFormat(raw, &result, format)

	if unmarshalErr == nil {
		if err := validateParsed(reflect.ValueOf(&result).Elem(), nil, nil); err != nil {
			return zero, err
		}
		return result, nil
	}

	// Standard unmarshal failed, fall back to map-based coercion approach
	// This handles cases where the input has type mismatches that need coercion
	return parseWithMapCoercion[T](raw, format)
}

// unmarshalByFormat unmarshals raw bytes into a value using the appropriate decoder
func unmarshalByFormat(raw []byte, v interface{}, format Format) error {
	switch format {
	case FormatJSON:
		return json.Unmarshal(raw, v)
	case FormatYAML:
		return yaml.Unmarshal(raw, v)
	default:
		return fmt.Errorf("unsupported format: %v", format)
	}
}

// checkRawStructureDepth parses raw bytes and checks if the structure depth exceeds the limit.
// This is called early in parsing to reject deeply nested input before expensive processing.
func checkRawStructureDepth(raw []byte, format Format) error {
	maxDepth := GetMaxStructureDepth()
	if maxDepth <= 0 {
		return nil // depth checking disabled
	}

	// Parse into generic interface{} to check structure depth
	// Note: parser.Parse already calls checkStructureDepth internally,
	// so this will return the depth error if the structure is too deep
	parser := GetParser(format)
	_, err := parser.Parse(raw)
	return err
}

// parseWithMapCoercion is the fallback parser that uses map-based coercion
// This is the original gopantic parsing logic
func parseWithMapCoercion[T any](raw []byte, format Format) (T, error) {
	var zero T
	var errors ErrorList

	// Get the appropriate parser for the format
	parser := GetParser(format)

	// Parse into a generic interface{} structure
	data, err := parser.Parse(raw)
	if err != nil {
		errors.Add(err)
		return zero, errors.AsError()
	}

	// Create new instance of T
	resultValue := reflect.New(reflect.TypeOf(zero)).Elem()
	resultType := resultValue.Type()

	// Handle different target types
	if resultType.Kind() == reflect.Slice || resultType.Kind() == reflect.Array {
		// Handle array/slice parsing
		result, err := parseIntoSlice[T](data, resultType, format)
		if err != nil {
			return zero, err
		}
		if err := validateParsed(reflect.ValueOf(&result).Elem(), nil, nil); err != nil {
			return zero, err
		}
		return result, nil
	}

	// Ensure data is a map for struct parsing
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		errors.Add(fmt.Errorf("cannot parse non-object data into struct"))
		return zero, errors.AsError()
	}

	// Process each field in the struct (parsing and coercion pass)
	for i := 0; i < resultType.NumField(); i++ {
		field := resultType.Field(i)
		fieldValue := resultValue.Field(i)

		// Skip unexported fields
		if !fieldValue.CanSet() {
			continue
		}

		// Get field key from appropriate tag (json or yaml), fallback to field name
		fieldKey := getFieldKey(field, format)
		if fieldKey == "-" {
			continue // Skip fields with tag:"-"
		}

		// Get value from data map
		rawValue, exists := dataMap[fieldKey]
		if !exists {
			// Field not present in data, leave as zero value
			rawValue = nil
		}

		// Coerce and set the value
		if err := setFieldValue(fieldValue, rawValue, field.Name, format); err != nil {
			errors.Add(err)
		}
	}

	if err := validateParsed(resultValue, errors, failedPaths(errors)); err != nil {
		return zero, err
	}

	return resultValue.Interface().(T), nil
}

// setFieldValue coerces and sets a value on a struct field
func setFieldValue(fieldValue reflect.Value, rawValue interface{}, fieldName string, format Format) error {
	fieldType := fieldValue.Type()
	fieldKind := fieldType.Kind()

	// Handle direct assignment for matching types first
	if rawValue != nil && reflect.TypeOf(rawValue).AssignableTo(fieldType) {
		fieldValue.Set(reflect.ValueOf(rawValue))
		return nil
	}

	// Handle specific types that need special treatment
	if fieldType == reflect.TypeOf(time.Time{}) {
		coercedValue, err := CoerceValueWithFormat(rawValue, fieldType, fieldName, format)
		if err != nil {
			return err
		}
		fieldValue.Set(reflect.ValueOf(coercedValue))
		return nil
	}

	// Use coercion for basic type conversion
	coercedValue, err := CoerceValueWithFormat(rawValue, fieldType, fieldName, format)
	if err != nil {
		return err
	}

	// Set the coerced value based on the field kind
	switch fieldKind {
	case reflect.String:
		fieldValue.SetString(coercedValue.(string))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		fieldValue.SetInt(coercedValue.(int64))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		fieldValue.SetUint(coercedValue.(uint64))
	case reflect.Float32, reflect.Float64:
		fieldValue.SetFloat(coercedValue.(float64))
	case reflect.Bool:
		fieldValue.SetBool(coercedValue.(bool))
	case reflect.Slice, reflect.Array:
		fieldValue.Set(reflect.ValueOf(coercedValue))
	case reflect.Struct:
		fieldValue.Set(reflect.ValueOf(coercedValue))
	case reflect.Ptr:
		fieldValue.Set(reflect.ValueOf(coercedValue))
	default:
		return NewParseError(fieldName, rawValue, fieldType.String(),
			fmt.Sprintf("unsupported field type: %s", fieldType))
	}

	return nil
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

	return validateParsed(val, nil, nil)
}

// validateParsed validates a decoded value of any kind, continuing errs from decoding. Paths
// in skip failed to decode, so their zero values are not reported a second time.
func validateParsed(val reflect.Value, errs ErrorList, skip map[string]bool) error {
	walk := validationWalk{errs: errs, skip: skip}
	walk.value(val, nil, 0)
	return walk.errs.AsError()
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
			key := segment.key
			if len(key) > maxPathKey {
				key = strings.ToValidUTF8(key[:maxPathKey], "") + "..."
			}
			fmt.Fprintf(&b, "[%s]", key)
		}
	}
	return b.String()
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
}

func (w *validationWalk) value(val reflect.Value, path *fieldPath, depth int) {
	if w.errs.full() {
		return
	}

	switch val.Kind() {
	case reflect.Struct:
		if val.Type() != reflect.TypeOf(time.Time{}) {
			w.structFields(val, path, depth)
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

func (w *validationWalk) structFields(val reflect.Value, path *fieldPath, depth int) {
	if maxDepth := GetMaxValidationDepth(); maxDepth > 0 && depth > maxDepth {
		w.errs.addCapped(fmt.Errorf("validation depth exceeded maximum of %d levels", maxDepth))
		return
	}

	typ := val.Type()
	validation := ParseValidationTags(typ)
	for i := 0; i < val.NumField() && !w.errs.full(); i++ {
		w.field(val, i, validation, path, depth)
	}
}

func (w *validationWalk) field(parent reflect.Value, i int, validation *StructValidation, path *fieldPath, depth int) {
	field := parent.Type().Field(i)
	fieldVal := parent.Field(i)

	// The exported fields of an unexported embedded struct stay reachable.
	if field.Anonymous && !field.IsExported() && fieldVal.Kind() == reflect.Struct {
		w.value(fieldVal, path, depth+1)
		return
	}

	rules := rulesFor(validation, field.Name)
	walkable := mayHoldRules(field.Type)
	if len(rules) == 0 && !walkable && w.skip == nil {
		return
	}
	if !fieldVal.CanInterface() || getFieldKey(field, FormatJSON) == "-" {
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

// parseIntoSlice handles parsing of array/slice data into slice/array types
func parseIntoSlice[T any](data interface{}, resultType reflect.Type, format Format) (T, error) {
	var zero T
	var errors ErrorList

	// Ensure data is an array
	dataSlice, ok := data.([]interface{})
	if !ok {
		errors.Add(fmt.Errorf("cannot parse non-array data into slice/array"))
		return zero, errors.AsError()
	}

	if resultType.Kind() == reflect.Slice {
		// Handle slice parsing
		slice := reflect.MakeSlice(resultType, len(dataSlice), len(dataSlice))

		for i, item := range dataSlice {
			elemValue := slice.Index(i)
			if err := setFieldValue(elemValue, item, fmt.Sprintf("[%d]", i), format); err != nil {
				errors.Add(err)
			}
		}

		if len(errors) > 0 {
			return zero, errors.AsError()
		}

		return slice.Interface().(T), nil
	} else if resultType.Kind() == reflect.Array {
		// Handle array parsing
		arrayLen := resultType.Len()
		if len(dataSlice) != arrayLen {
			errors.Add(fmt.Errorf("array length mismatch: expected %d elements, got %d", arrayLen, len(dataSlice)))
			return zero, errors.AsError()
		}

		array := reflect.New(resultType).Elem()

		for i, item := range dataSlice {
			elemValue := array.Index(i)
			if err := setFieldValue(elemValue, item, fmt.Sprintf("[%d]", i), format); err != nil {
				errors.Add(err)
			}
		}

		if len(errors) > 0 {
			return zero, errors.AsError()
		}

		return array.Interface().(T), nil
	}

	errors.Add(fmt.Errorf("unsupported type: %s", resultType.Kind()))
	return zero, errors.AsError()
}
