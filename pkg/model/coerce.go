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
	coerced, err := coerceValue(value, targetType, fieldName, format)
	if parseErr, ok := err.(*ParseError); ok && IsSensitiveField(fieldName) {
		parseErr.redact()
	}
	return coerced, err
}

func coerceValue(value interface{}, targetType reflect.Type, fieldName string, format Format) (interface{}, error) {
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
	case reflect.Slice, reflect.Array, reflect.Struct, reflect.Ptr, reflect.Map:
		return coerceContainer(value, targetType, fieldName, format)
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
		return unixTime(float64(v), value, fieldName)
	case json.Number:
		f, err := v.Float64()
		if err != nil {
			return time.Time{}, NewParseError(fieldName, value, "time.Time", fmt.Sprintf("cannot parse number %q: %v", v, err))
		}
		return coerceToTime(f, fieldName)
	case float64:
		return unixTime(v, value, fieldName)
	case int:
		return unixTime(float64(v), value, fieldName)
	default:
		return time.Time{}, NewParseError(fieldName, value, "time.Time",
			fmt.Sprintf("cannot coerce %T to time.Time", value))
	}
}

// Unix seconds at the start of year 1 and the end of year 9999, the years RFC 3339 can write.
const (
	minUnixSeconds = -62135596800
	maxUnixSeconds = 253402300799
)

// unixTime converts Unix seconds, which may be fractional.
func unixTime(seconds float64, value interface{}, fieldName string) (time.Time, error) {
	if math.IsNaN(seconds) || seconds < minUnixSeconds || seconds > maxUnixSeconds {
		return time.Time{}, NewParseError(fieldName, value, "time.Time", "timestamp is out of range")
	}
	whole, fraction := math.Modf(seconds)
	return time.Unix(int64(whole), int64(fraction*1e9)), nil
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
