package model

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"time"
)

// CoerceValue attempts to coerce a value to the target type with intelligent type conversion.
// Uses JSON format assumptions by default. Supports conversion between strings, numbers,
// booleans, time.Time, slices, arrays, and nested structs.
//
// Example:
//
//	result, err := model.CoerceValue("123", reflect.TypeOf(0), "user_id")
//	// result will be int(123)
func CoerceValue(value interface{}, targetType reflect.Type, fieldName string) (interface{}, error) {
	return CoerceValueWithFormat(value, targetType, fieldName, FormatJSON)
}

// CoerceValueWithFormat attempts to coerce a value to the target type with format-specific awareness.
// Different formats may have different type coercion rules and conventions.
// This is the core type coercion function used by all parsing operations.
//
// Supported conversions include:
// - String <-> numeric types (int, float, etc.)
// - String <-> bool (true/false, 1/0, yes/no, etc.)
// - String/numeric -> time.Time (various formats)
// - Array/slice element coercion
// - Map -> struct conversion with nested coercion
func CoerceValueWithFormat(value interface{}, targetType reflect.Type, fieldName string, format Format) (interface{}, error) {
	if value == nil {
		return getZeroValueForType(targetType), nil
	}

	// Handle specific struct types first
	if targetType == reflect.TypeOf(time.Time{}) {
		return coerceToTime(value, fieldName)
	}

	// Fall back to kind-based coercion
	targetKind := targetType.Kind()
	switch targetKind {
	case reflect.String:
		return coerceToString(value, fieldName)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return coerceToInt(value, targetType, fieldName)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return coerceToUint(value, targetType, fieldName)
	case reflect.Float32, reflect.Float64:
		return coerceToFloat(value, targetType, fieldName)
	case reflect.Bool:
		return coerceToBool(value, fieldName)
	case reflect.Slice:
		return coerceToSlice(value, targetType, fieldName)
	case reflect.Array:
		return coerceToArray(value, targetType, fieldName)
	case reflect.Struct:
		return coerceToStructWithFormat(value, targetType, fieldName, format)
	case reflect.Ptr:
		return coerceToPointer(value, targetType, fieldName)
	default:
		return nil, NewParseError(fieldName, value, targetType.String(),
			fmt.Sprintf("coercion to %s not supported", targetType))
	}
}

// coerceToString converts scalars to string. Containers are rejected, since formatting one
// would hand back Go syntax as if it were the input.
func coerceToString(value interface{}, fieldName string) (string, error) {
	switch v := value.(type) {
	case string:
		return v, nil
	case json.Number:
		return v.String(), nil
	case bool:
		return strconv.FormatBool(v), nil
	}

	rv := reflect.ValueOf(value)
	switch {
	case rv.CanInt():
		return strconv.FormatInt(rv.Int(), 10), nil
	case rv.CanUint():
		return strconv.FormatUint(rv.Uint(), 10), nil
	case rv.CanFloat():
		return strconv.FormatFloat(rv.Float(), 'f', -1, rv.Type().Bits()), nil
	}

	switch rv.Kind() {
	case reflect.Map, reflect.Slice, reflect.Array:
		return "", NewParseError(fieldName, value, "string", fmt.Sprintf("cannot coerce %T to string", value))
	}
	return fmt.Sprintf("%v", value), nil
}

// coerceToInt converts value to an integer that fits target.
func coerceToInt(value interface{}, target reflect.Type, fieldName string) (int64, error) {
	n, err := toInt64(value, fieldName)
	if err != nil {
		return 0, err
	}
	if target.OverflowInt(n) {
		return 0, NewParseError(fieldName, value, target.String(), fmt.Sprintf("value %d overflows %s", n, target))
	}
	return n, nil
}

func toInt64(value interface{}, fieldName string) (int64, error) {
	rv := reflect.ValueOf(value)
	switch {
	case rv.CanInt():
		return rv.Int(), nil
	case rv.CanUint():
		u := rv.Uint()
		if u > math.MaxInt64 {
			return 0, NewParseError(fieldName, value, "int64", "value too large for int64")
		}
		return int64(u), nil
	case rv.CanFloat():
		return floatToInt64(rv.Float(), value, fieldName)
	}

	switch v := value.(type) {
	case json.Number:
		if n, err := strconv.ParseInt(v.String(), 10, 64); err == nil {
			return n, nil
		}
		f, err := v.Float64()
		if err != nil {
			return 0, NewParseError(fieldName, value, "int64", fmt.Sprintf("cannot parse number %q: %v", v, err))
		}
		return floatToInt64(f, value, fieldName)
	case string:
		parsed, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return 0, NewParseError(fieldName, value, "int64",
				fmt.Sprintf("cannot parse string %q as integer: %v", v, err))
		}
		return parsed, nil
	case bool:
		if v {
			return 1, nil
		}
		return 0, nil
	default:
		return 0, NewParseError(fieldName, value, "int64",
			fmt.Sprintf("cannot coerce %T to int64", value))
	}
}

func floatToInt64(f float64, value interface{}, fieldName string) (int64, error) {
	if err := checkIntegralFloat(f, value, "int64", fieldName); err != nil {
		return 0, err
	}
	return int64(f), nil
}

// checkIntegralFloat rejects floats with no exact integer value. The 2^53 bound also keeps
// the later conversion in range, so no comparison against a rounded MaxInt64 is needed.
func checkIntegralFloat(f float64, value interface{}, targetType, fieldName string) error {
	switch {
	case math.IsNaN(f) || math.IsInf(f, 0):
		return NewParseError(fieldName, value, targetType, "value is not a finite number")
	case f != math.Trunc(f):
		return NewParseError(fieldName, value, targetType, "value has a fractional part")
	case math.Abs(f) >= 1<<53:
		return NewParseError(fieldName, value, targetType, "value is too large to convert exactly")
	}
	return nil
}

// coerceToUint converts value to an unsigned integer that fits target.
func coerceToUint(value interface{}, target reflect.Type, fieldName string) (uint64, error) {
	n, err := toUint64(value, fieldName)
	if err != nil {
		return 0, err
	}
	if target.OverflowUint(n) {
		return 0, NewParseError(fieldName, value, target.String(), fmt.Sprintf("value %d overflows %s", n, target))
	}
	return n, nil
}

func toUint64(value interface{}, fieldName string) (uint64, error) {
	rv := reflect.ValueOf(value)
	switch {
	case rv.CanInt():
		n := rv.Int()
		if n < 0 {
			return 0, NewParseError(fieldName, value, "uint64", "negative value cannot be coerced to uint64")
		}
		return uint64(n), nil
	case rv.CanUint():
		return rv.Uint(), nil
	case rv.CanFloat():
		return floatToUint64(rv.Float(), value, fieldName)
	}

	switch v := value.(type) {
	case json.Number:
		if n, err := strconv.ParseUint(v.String(), 10, 64); err == nil {
			return n, nil
		}
		f, err := v.Float64()
		if err != nil {
			return 0, NewParseError(fieldName, value, "uint64", fmt.Sprintf("cannot parse number %q: %v", v, err))
		}
		return floatToUint64(f, value, fieldName)
	case string:
		parsed, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			return 0, NewParseError(fieldName, value, "uint64",
				fmt.Sprintf("cannot parse string %q as unsigned integer: %v", v, err))
		}
		return parsed, nil
	case bool:
		if v {
			return 1, nil
		}
		return 0, nil
	default:
		return 0, NewParseError(fieldName, value, "uint64",
			fmt.Sprintf("cannot coerce %T to uint64", value))
	}
}

func floatToUint64(f float64, value interface{}, fieldName string) (uint64, error) {
	if f < 0 {
		return 0, NewParseError(fieldName, value, "uint64", "negative value cannot be coerced to uint64")
	}
	if err := checkIntegralFloat(f, value, "uint64", fieldName); err != nil {
		return 0, err
	}
	return uint64(f), nil
}

// coerceToFloat converts value to a float that fits target. NaN and infinity are refused from
// strings: they compare false against every bound, so a validator could not reject them.
func coerceToFloat(value interface{}, target reflect.Type, fieldName string) (float64, error) {
	f, err := toFloat(value, target, fieldName)
	if err != nil {
		return 0, err
	}
	if target.OverflowFloat(f) {
		return 0, NewParseError(fieldName, value, target.String(), fmt.Sprintf("value %g overflows %s", f, target))
	}
	return f, nil
}

func toFloat(value interface{}, target reflect.Type, fieldName string) (float64, error) {
	rv := reflect.ValueOf(value)
	switch {
	case rv.CanFloat():
		return rv.Float(), nil
	case rv.CanInt():
		return float64(rv.Int()), nil
	case rv.CanUint():
		return float64(rv.Uint()), nil
	}

	switch v := value.(type) {
	case json.Number:
		f, err := v.Float64()
		if err != nil {
			return 0, NewParseError(fieldName, value, "float64", fmt.Sprintf("cannot parse number %q: %v", v, err))
		}
		return f, nil
	case string:
		parsed, err := strconv.ParseFloat(v, target.Bits())
		if err != nil {
			return 0, NewParseError(fieldName, value, "float64",
				fmt.Sprintf("cannot parse string %q as float: %v", v, err))
		}
		if math.IsNaN(parsed) || math.IsInf(parsed, 0) {
			return 0, NewParseError(fieldName, value, "float64",
				fmt.Sprintf("string %q is not a finite number", v))
		}
		return parsed, nil
	case bool:
		if v {
			return 1.0, nil
		}
		return 0.0, nil
	default:
		return 0, NewParseError(fieldName, value, "float64",
			fmt.Sprintf("cannot coerce %T to float64", value))
	}
}

// coerceToBool converts various types to bool
func coerceToBool(value interface{}, fieldName string) (bool, error) {
	switch v := value.(type) {
	case bool:
		return v, nil
	case json.Number:
		f, err := v.Float64()
		if err != nil {
			return false, NewParseError(fieldName, value, "bool", fmt.Sprintf("cannot parse number %q: %v", v, err))
		}
		return f != 0, nil
	case string:
		switch v {
		case "true", "True", "TRUE", "1", "yes", "Yes", "YES", "on", "On", "ON":
			return true, nil
		case "false", "False", "FALSE", "0", "no", "No", "NO", "off", "Off", "OFF", "":
			return false, nil
		default:
			return false, NewParseError(fieldName, value, "bool",
				fmt.Sprintf("cannot parse string %q as boolean", v))
		}
	}

	rv := reflect.ValueOf(value)
	switch {
	case rv.CanInt():
		return rv.Int() != 0, nil
	case rv.CanUint():
		return rv.Uint() != 0, nil
	case rv.CanFloat():
		return rv.Float() != 0, nil
	}
	return false, NewParseError(fieldName, value, "bool", fmt.Sprintf("cannot coerce %T to bool", value))
}

// coerceToTime converts various types to time.Time
func coerceToTime(value interface{}, fieldName string) (time.Time, error) {
	switch v := value.(type) {
	case time.Time:
		return v, nil
	case string:
		return parseTimeFromString(v, fieldName)
	case int64:
		// Unix timestamp (seconds)
		return time.Unix(v, 0), nil
	case float64:
		// Unix timestamp (seconds, may have fractional part)
		sec := int64(v)
		nsec := int64((v - float64(sec)) * 1e9)
		return time.Unix(sec, nsec), nil
	case int:
		// Unix timestamp (seconds)
		return time.Unix(int64(v), 0), nil
	default:
		return time.Time{}, NewParseError(fieldName, value, "time.Time",
			fmt.Sprintf("cannot coerce %T to time.Time", value))
	}
}

// parseTimeFromString attempts to parse time from string using multiple formats.
// Formats are ordered by likelihood: RFC3339 variants first (most common in APIs),
// then ISO 8601, then common date/time formats.
func parseTimeFromString(s, fieldName string) (time.Time, error) {
	// Quick heuristic: if string has 'T' at position 10, likely ISO 8601/RFC3339
	// Try those formats first for better performance
	if len(s) > 10 && s[10] == 'T' {
		formats := []string{
			time.RFC3339,           // "2006-01-02T15:04:05Z07:00" - most common
			time.RFC3339Nano,       // "2006-01-02T15:04:05.999999999Z07:00"
			"2006-01-02T15:04:05Z", // ISO 8601 UTC
			"2006-01-02T15:04:05",  // ISO 8601 without timezone
		}
		for _, format := range formats {
			if t, err := time.Parse(format, s); err == nil {
				return t, nil
			}
		}
	}

	// Try remaining formats
	otherFormats := []string{
		"2006-01-02 15:04:05", // Common format with space
		"2006-01-02",          // Date only
		"15:04:05",            // Time only (today's date)
	}

	for _, format := range otherFormats {
		if t, err := time.Parse(format, s); err == nil {
			return t, nil
		}
	}

	return time.Time{}, NewParseError(fieldName, s, "time.Time",
		fmt.Sprintf("cannot parse string %q as time.Time using standard formats", s))
}

// coerceToSlice converts JSON arrays to Go slices with element coercion
func coerceToSlice(value interface{}, targetType reflect.Type, fieldName string) (interface{}, error) {
	if value == nil {
		// Return zero slice for nil
		return reflect.Zero(targetType).Interface(), nil
	}

	// Handle JSON arrays ([]interface{})
	sourceSlice, ok := value.([]interface{})
	if !ok {
		return nil, NewParseError(fieldName, value, targetType.String(),
			fmt.Sprintf("cannot coerce %T to slice", value))
	}

	elementType := targetType.Elem()
	sliceLen := len(sourceSlice)

	// Create new slice with proper type
	resultSlice := reflect.MakeSlice(targetType, sliceLen, sliceLen)

	// Coerce each element
	for i, elem := range sourceSlice {
		coercedElem, err := CoerceValue(elem, elementType, fmt.Sprintf("%s[%d]", fieldName, i))
		if err != nil {
			return nil, err
		}

		// Set the element in the result slice
		elemValue := reflect.ValueOf(coercedElem)
		if elemValue.Type().ConvertibleTo(elementType) {
			elemValue = elemValue.Convert(elementType)
		}
		resultSlice.Index(i).Set(elemValue)
	}

	return resultSlice.Interface(), nil
}

// coerceToArray converts JSON arrays to Go arrays with element coercion
func coerceToArray(value interface{}, targetType reflect.Type, fieldName string) (interface{}, error) {
	if value == nil {
		// Return zero array for nil
		return reflect.Zero(targetType).Interface(), nil
	}

	// Handle JSON arrays ([]interface{})
	sourceSlice, ok := value.([]interface{})
	if !ok {
		return nil, NewParseError(fieldName, value, targetType.String(),
			fmt.Sprintf("cannot coerce %T to array", value))
	}

	elementType := targetType.Elem()
	arrayLen := targetType.Len()
	sourceLen := len(sourceSlice)

	if sourceLen != arrayLen {
		return nil, NewParseError(fieldName, value, targetType.String(),
			fmt.Sprintf("array length mismatch: expected %d, got %d", arrayLen, sourceLen))
	}

	// Create new array with proper type
	resultArray := reflect.New(targetType).Elem()

	// Coerce each element
	for i, elem := range sourceSlice {
		coercedElem, err := CoerceValue(elem, elementType, fmt.Sprintf("%s[%d]", fieldName, i))
		if err != nil {
			return nil, err
		}

		// Set the element in the result array
		elemValue := reflect.ValueOf(coercedElem)
		if elemValue.Type().ConvertibleTo(elementType) {
			elemValue = elemValue.Convert(elementType)
		}
		resultArray.Index(i).Set(elemValue)
	}

	return resultArray.Interface(), nil
}

// coerceToStructWithFormat converts objects to Go structs recursively with format awareness
func coerceToStructWithFormat(value interface{}, targetType reflect.Type, fieldName string, format Format) (interface{}, error) {
	if value == nil {
		// Return zero value for nil
		return reflect.Zero(targetType).Interface(), nil
	}

	// Handle data objects (map[string]interface{})
	sourceMap, ok := value.(map[string]interface{})
	if !ok {
		return nil, NewParseError(fieldName, value, targetType.String(),
			fmt.Sprintf("cannot coerce %T to struct", value))
	}

	// Create new instance of the target struct
	resultValue := reflect.New(targetType).Elem()

	// Parse validation rules for this struct type
	validation := ParseValidationTags(targetType)
	var errors ErrorList

	// Process each field in the nested struct
	for i := 0; i < targetType.NumField(); i++ {
		field := targetType.Field(i)
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
		rawValue, exists := sourceMap[fieldKey]
		nestedFieldName := fmt.Sprintf("%s.%s", fieldName, field.Name)

		if !exists {
			// Field not present in data, leave as zero value
			rawValue = nil
		}

		// Recursively coerce and set the value
		if err := setFieldValue(fieldValue, rawValue, nestedFieldName, format); err != nil {
			errors.Add(err)
			continue // Skip validation if coercion failed
		}

		// Apply validation rules to nested fields
		if err := validateFieldValue(field.Name, fieldKey, fieldValue.Interface(), validation); err != nil {
			// Update error to include nested path
			updatedErr := updateFieldPaths(err, nestedFieldName, field.Name)
			errors.Add(updatedErr)
		}
	}

	if errors.HasErrors() {
		return nil, errors.AsError()
	}

	return resultValue.Interface(), nil
}

// getZeroValueForType returns the zero value for the given type
func getZeroValueForType(t reflect.Type) interface{} {
	if t == reflect.TypeOf(time.Time{}) {
		return time.Time{}
	}

	// Handle types that need the full type information
	switch t.Kind() {
	case reflect.Slice:
		return reflect.MakeSlice(t, 0, 0).Interface()
	case reflect.Array:
		return reflect.Zero(t).Interface()
	case reflect.Struct:
		return reflect.Zero(t).Interface()
	case reflect.Ptr:
		// For pointers, zero value is nil
		return reflect.Zero(t).Interface()
	default:
		// Fall back to kind-based zero values
		return getZeroValue(t.Kind())
	}
}

// getZeroValue returns the zero value for the given kind
func getZeroValue(kind reflect.Kind) interface{} {
	switch kind {
	case reflect.String:
		return ""
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return int64(0)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return uint64(0)
	case reflect.Float32, reflect.Float64:
		return 0.0
	case reflect.Bool:
		return false
	default:
		return nil
	}
}

// updateFieldPaths recursively updates field paths in validation errors to include nested paths
func updateFieldPaths(err error, nestedFieldName, _ string) error {
	switch e := err.(type) {
	case *ValidationError:
		// Create a copy to avoid modifying the original
		return &ValidationError{
			Field:   nestedFieldName,
			Value:   e.Value,
			Rule:    e.Rule,
			Message: e.Message,
		}
	case ErrorList:
		// Handle multiple validation errors
		var updatedErrors ErrorList
		for _, innerErr := range e {
			updatedErr := updateFieldPaths(innerErr, nestedFieldName, "")
			updatedErrors.Add(updatedErr)
		}
		return updatedErrors
	case *ParseError:
		// Update parse errors as well
		return &ParseError{
			Field:   nestedFieldName,
			Value:   e.Value,
			Type:    e.Type,
			Message: e.Message,
		}
	default:
		// For other error types, return as-is
		return err
	}
}

// coerceToPointer handles pointer types by coercing to the underlying type and creating a pointer
func coerceToPointer(value interface{}, targetType reflect.Type, fieldName string) (interface{}, error) {
	// If value is nil, return a nil pointer
	if value == nil {
		return reflect.Zero(targetType).Interface(), nil
	}

	// Get the element type (what the pointer points to)
	elemType := targetType.Elem()

	// Coerce the value to the element type
	coercedValue, err := CoerceValue(value, elemType, fieldName)
	if err != nil {
		return nil, err
	}

	// Create a pointer to the coerced value
	ptrValue := reflect.New(elemType)

	// Convert coercedValue to reflect.Value and ensure it matches the target type exactly
	coercedReflectValue := reflect.ValueOf(coercedValue)

	// Handle type conversion for numeric types if needed
	if coercedReflectValue.Type() != elemType {
		// If both are numeric types, convert
		if coercedReflectValue.Kind() >= reflect.Int && coercedReflectValue.Kind() <= reflect.Float64 &&
			elemType.Kind() >= reflect.Int && elemType.Kind() <= reflect.Float64 {
			convertedValue := coercedReflectValue.Convert(elemType)
			ptrValue.Elem().Set(convertedValue)
		} else {
			return nil, NewParseError(fieldName, value, targetType.String(),
				fmt.Sprintf("cannot assign %s to %s", coercedReflectValue.Type(), elemType))
		}
	} else {
		ptrValue.Elem().Set(coercedReflectValue)
	}

	return ptrValue.Interface(), nil
}
