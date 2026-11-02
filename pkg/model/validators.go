package model

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"regexp"
	"strings"
)

// numberValue gives a json.Number its numeric value: its kind is string, so the bounds would
// otherwise compare its length. One that is not a number becomes NaN, which fails both bounds.
func numberValue(val reflect.Value) reflect.Value {
	if val.Type() != jsonNumberType {
		return val
	}
	f, err := json.Number(val.String()).Float64()
	if err != nil {
		return reflect.ValueOf(math.NaN())
	}
	return reflect.ValueOf(f)
}

// RequiredValidator checks that a field has a non-zero value
type RequiredValidator struct{}

// Name returns the validator name
func (v *RequiredValidator) Name() string {
	return "required"
}

// Validate checks that a field has a non-zero value
func (v *RequiredValidator) Validate(fieldName string, value interface{}) error {
	if value == nil {
		return NewValidationError(fieldName, value, "required", "field is required")
	}

	// Check for zero values based on type
	val := reflect.ValueOf(value)
	switch val.Kind() {
	case reflect.String:
		if val.String() == "" {
			return NewValidationError(fieldName, value, "required", "field is required")
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if val.Int() == 0 {
			return NewValidationError(fieldName, value, "required", "field is required")
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if val.Uint() == 0 {
			return NewValidationError(fieldName, value, "required", "field is required")
		}
	case reflect.Float32, reflect.Float64:
		if f := val.Float(); f == 0.0 || math.IsNaN(f) {
			return NewValidationError(fieldName, value, "required", "field is required")
		}
	case reflect.Bool:
		// For booleans, false is considered a valid value, so we don't fail
		// This matches common validation library behavior
		return nil
	case reflect.Slice, reflect.Map:
		if val.Len() == 0 {
			return NewValidationError(fieldName, value, "required", "field is required")
		}
	case reflect.Array:
		if val.IsZero() {
			return NewValidationError(fieldName, value, "required", "field is required")
		}
	case reflect.Ptr, reflect.Interface:
		if val.IsNil() {
			return NewValidationError(fieldName, value, "required", "field is required")
		}
	case reflect.Struct:
		// Check if struct is a zero value (all fields are zero)
		if val.IsZero() {
			return NewValidationError(fieldName, value, "required", "field is required")
		}
	}

	return nil
}

// MinValidator checks that a numeric value or string length is at least the minimum
type MinValidator struct {
	Min float64
}

// Name returns the validator name
func (v *MinValidator) Name() string {
	return "min"
}

// Validate checks that a numeric value or string length is at least the minimum
func (v *MinValidator) Validate(fieldName string, value interface{}) error {
	if value == nil {
		return nil // nil values are handled by required validator
	}

	val := reflect.ValueOf(value)

	// Handle pointer types by dereferencing them
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil // nil pointers are not validated
		}
		val = val.Elem()
	}
	val = numberValue(val)

	switch val.Kind() {
	case reflect.String:
		if float64(len(val.String())) < v.Min {
			return NewValidationError(fieldName, value, "min",
				fmt.Sprintf("string length must be at least %.0f characters", v.Min))
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if float64(val.Int()) < v.Min {
			return NewValidationError(fieldName, value, "min",
				fmt.Sprintf("value must be at least %.0f", v.Min))
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if float64(val.Uint()) < v.Min {
			return NewValidationError(fieldName, value, "min",
				fmt.Sprintf("value must be at least %.0f", v.Min))
		}
	case reflect.Float32, reflect.Float64:
		if !(val.Float() >= v.Min) { // written so NaN fails
			return NewValidationError(fieldName, value, "min",
				fmt.Sprintf("value must be at least %g", v.Min))
		}
	case reflect.Slice, reflect.Array:
		if float64(val.Len()) < v.Min {
			return NewValidationError(fieldName, value, "min",
				fmt.Sprintf("array length must be at least %.0f", v.Min))
		}
	default:
		return NewValidationError(fieldName, value, "min",
			fmt.Sprintf("min validation not supported for type %T", value))
	}

	return nil
}

// MaxValidator checks that a numeric value or string length is at most the maximum
type MaxValidator struct {
	Max float64
}

// Name returns the validator name
func (v *MaxValidator) Name() string {
	return "max"
}

// Validate checks that a numeric value or string length is at most the maximum
func (v *MaxValidator) Validate(fieldName string, value interface{}) error {
	if value == nil {
		return nil // nil values are handled by required validator
	}

	val := reflect.ValueOf(value)

	// Handle pointer types by dereferencing them
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil // nil pointers are not validated
		}
		val = val.Elem()
	}
	val = numberValue(val)

	switch val.Kind() {
	case reflect.String:
		if float64(len(val.String())) > v.Max {
			return NewValidationError(fieldName, value, "max",
				fmt.Sprintf("string length must be at most %.0f characters", v.Max))
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if float64(val.Int()) > v.Max {
			return NewValidationError(fieldName, value, "max",
				fmt.Sprintf("value must be at most %.0f", v.Max))
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if float64(val.Uint()) > v.Max {
			return NewValidationError(fieldName, value, "max",
				fmt.Sprintf("value must be at most %.0f", v.Max))
		}
	case reflect.Float32, reflect.Float64:
		if !(val.Float() <= v.Max) { // written so NaN fails
			return NewValidationError(fieldName, value, "max",
				fmt.Sprintf("value must be at most %g", v.Max))
		}
	case reflect.Slice, reflect.Array:
		if float64(val.Len()) > v.Max {
			return NewValidationError(fieldName, value, "max",
				fmt.Sprintf("array length must be at most %.0f", v.Max))
		}
	default:
		return NewValidationError(fieldName, value, "max",
			fmt.Sprintf("max validation not supported for type %T", value))
	}

	return nil
}

// EmailValidator validates email addresses using a simple but practical regex
type EmailValidator struct{}

// Name returns the validator name
func (v *EmailValidator) Name() string {
	return "email"
}

// Email validation regex - simple but covers most practical cases
// This is intentionally not RFC 5322 compliant for simplicity and performance
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// Validate checks if the value is a valid email address
func (v *EmailValidator) Validate(fieldName string, value interface{}) error {
	if value == nil {
		return nil // nil values are handled by required validator
	}

	// Handle pointer types by dereferencing them
	actualValue := value
	val := reflect.ValueOf(value)
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil // nil pointers are not validated
		}
		actualValue = val.Elem().Interface()
	}

	// Convert to string
	var email string
	switch v := actualValue.(type) {
	case string:
		email = v
	default:
		email = fmt.Sprintf("%v", actualValue)
	}

	// Skip validation for empty strings (handled by required validator)
	if email == "" {
		return nil
	}

	// Basic length check
	if len(email) > 254 {
		return NewValidationError(fieldName, value, "email", "email address is too long")
	}

	// Regex validation
	if !emailRegex.MatchString(email) {
		return NewValidationError(fieldName, value, "email", "invalid email address format")
	}

	// Additional checks for common issues
	if strings.Contains(email, "..") {
		return NewValidationError(fieldName, value, "email", "email address cannot contain consecutive dots")
	}

	if strings.HasPrefix(email, ".") || strings.HasSuffix(email, ".") {
		return NewValidationError(fieldName, value, "email", "email address cannot start or end with a dot")
	}

	// Check for valid domain part
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return NewValidationError(fieldName, value, "email", "invalid email address format")
	}

	localPart, domain := parts[0], parts[1]

	// Local part checks
	if localPart == "" || len(localPart) > 64 {
		return NewValidationError(fieldName, value, "email", "email local part must be 1-64 characters")
	}

	if strings.HasPrefix(localPart, ".") || strings.HasSuffix(localPart, ".") {
		return NewValidationError(fieldName, value, "email", "email local part cannot start or end with a dot")
	}

	// Domain checks
	if domain == "" || len(domain) > 253 {
		return NewValidationError(fieldName, value, "email", "email domain must be 1-253 characters")
	}

	if strings.HasPrefix(domain, ".") || strings.HasSuffix(domain, ".") {
		return NewValidationError(fieldName, value, "email", "email domain cannot start or end with a dot")
	}

	return nil
}

// LengthValidator checks exact length for strings and arrays
type LengthValidator struct {
	Length int
}

// Name returns the validator name
func (v *LengthValidator) Name() string {
	return "length"
}

// Validate checks if the value has the exact required length
func (v *LengthValidator) Validate(fieldName string, value interface{}) error {
	if value == nil {
		return nil // nil values are handled by required validator
	}

	val := reflect.ValueOf(value)

	// Handle pointer types by dereferencing them
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil // nil pointers are not validated
		}
		val = val.Elem()
	}

	var actualLength int

	switch val.Kind() {
	case reflect.String:
		actualLength = len(val.String())
	case reflect.Slice, reflect.Array:
		actualLength = val.Len()
	default:
		return NewValidationError(fieldName, value, "length",
			fmt.Sprintf("length validation not supported for type %T", value))
	}

	if actualLength != v.Length {
		return NewValidationError(fieldName, value, "length",
			fmt.Sprintf("length must be exactly %d", v.Length))
	}

	return nil
}

// AlphaValidator checks that a string contains only alphabetic characters
type AlphaValidator struct{}

// Name returns the validator name
func (v *AlphaValidator) Name() string {
	return "alpha"
}

var alphaRegex = regexp.MustCompile(`^[a-zA-Z]+$`)

// Validate checks if the value contains only alphabetic characters
func (v *AlphaValidator) Validate(fieldName string, value interface{}) error {
	if value == nil {
		return nil // nil values are handled by required validator
	}

	// Handle pointer types by dereferencing them
	actualValue := value
	val := reflect.ValueOf(value)
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil // nil pointers are not validated
		}
		actualValue = val.Elem().Interface()
	}

	str, ok := actualValue.(string)
	if !ok {
		return NewValidationError(fieldName, value, "alpha", "value must be a string")
	}

	if str == "" {
		return nil // empty strings are handled by required validator
	}

	if !alphaRegex.MatchString(str) {
		return NewValidationError(fieldName, value, "alpha", "value must contain only alphabetic characters")
	}

	return nil
}

// AlphanumValidator checks that a string contains only alphanumeric characters
type AlphanumValidator struct{}

// Name returns the validator name
func (v *AlphanumValidator) Name() string {
	return "alphanum"
}

var alphanumRegex = regexp.MustCompile(`^[a-zA-Z0-9]+$`)

// Validate checks if the value contains only alphanumeric characters
func (v *AlphanumValidator) Validate(fieldName string, value interface{}) error {
	if value == nil {
		return nil // nil values are handled by required validator
	}

	// Handle pointer types by dereferencing them
	actualValue := value
	val := reflect.ValueOf(value)
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil // nil pointers are not validated
		}
		actualValue = val.Elem().Interface()
	}

	str, ok := actualValue.(string)
	if !ok {
		return NewValidationError(fieldName, value, "alphanum", "value must be a string")
	}

	if str == "" {
		return nil // empty strings are handled by required validator
	}

	if !alphanumRegex.MatchString(str) {
		return NewValidationError(fieldName, value, "alphanum", "value must contain only alphanumeric characters")
	}

	return nil
}
