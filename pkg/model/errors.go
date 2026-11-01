// Package model provides core parsing and validation functionality for gopantic.
// It includes type coercion, error handling, and the main ParseInto function
// for converting JSON data into typed Go structs.
package model

import (
	"encoding/json"
	"fmt"
	"strings"
)

// RedactedValue is the placeholder used when sensitive field values are sanitized.
const RedactedValue = "[REDACTED]"

// ParseError represents an error that occurred during data parsing.
// Contains detailed information about the field, value, and target type that caused the error.
type ParseError struct {
	Field   string
	Value   interface{}
	Type    string
	Message string

	sensitive bool // Matched on the input key, which Field does not carry
}

func (e ParseError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("parse error on field %q: %s", e.Field, e.Message)
	}
	return fmt.Sprintf("parse error: %s", e.Message)
}

// NewParseError creates a new ParseError with detailed context information.
// Use this to create meaningful error messages for parsing failures.
func NewParseError(field string, value interface{}, targetType, message string) *ParseError {
	return &ParseError{
		Field:   field,
		Value:   value,
		Type:    targetType,
		Message: message,
	}
}

// SanitizedValue returns the error value with sensitive data redacted.
// If the field matches a sensitive pattern, returns RedactedValue ("[REDACTED]").
// Otherwise returns the original value unchanged.
func (e ParseError) SanitizedValue() interface{} {
	if e.sensitive || IsSensitiveField(e.Field) {
		return RedactedValue
	}
	return e.Value
}

// redact replaces the message, which can quote the input, and marks the value as sensitive.
func (e *ParseError) redact() {
	e.sensitive = true
	e.Message = "invalid value for " + e.Type
}

// ValidationError represents a validation failure with detailed field and rule information.
// Supports nested field paths and structured error details for comprehensive error reporting.
type ValidationError struct {
	Field     string
	FieldPath string // Full field path for nested structures (e.g., "User.Address.Street")
	Value     interface{}
	Rule      string
	Message   string
	Details   map[string]interface{} // Additional structured information
}

func (e ValidationError) Error() string {
	fieldName := e.Field
	if e.FieldPath != "" {
		fieldName = e.FieldPath
	}

	if fieldName != "" {
		return fmt.Sprintf("validation error on field %q: %s", fieldName, e.Message)
	}
	return fmt.Sprintf("validation error: %s", e.Message)
}

// NewValidationError creates a new ValidationError with basic field and rule information.
// This is the most commonly used constructor for validation errors.
func NewValidationError(field string, value interface{}, rule, message string) *ValidationError {
	return &ValidationError{
		Field:     field,
		FieldPath: field, // For backward compatibility
		Value:     value,
		Rule:      rule,
		Message:   message,
		Details:   make(map[string]interface{}),
	}
}

// NewValidationErrorWithPath creates a new ValidationError with explicit field path for nested structures.
// Use this when validating nested structs to provide clear error paths like "User.Address.Street".
func NewValidationErrorWithPath(field, fieldPath string, value interface{}, rule, message string) *ValidationError {
	return &ValidationError{
		Field:     field,
		FieldPath: fieldPath,
		Value:     value,
		Rule:      rule,
		Message:   message,
		Details:   make(map[string]interface{}),
	}
}

// NewValidationErrorWithDetails creates a new ValidationError with additional structured details.
// The details map can contain additional context information for advanced error reporting scenarios.
func NewValidationErrorWithDetails(field, fieldPath string, value interface{}, rule, message string, details map[string]interface{}) *ValidationError {
	if details == nil {
		details = make(map[string]interface{})
	}

	return &ValidationError{
		Field:     field,
		FieldPath: fieldPath,
		Value:     value,
		Rule:      rule,
		Message:   message,
		Details:   details,
	}
}

// SanitizedValue returns the error value with sensitive data redacted.
// If the field name matches any sensitive pattern, returns RedactedValue ("[REDACTED]").
// Otherwise returns the original value unchanged.
func (e ValidationError) SanitizedValue() interface{} {
	if IsSensitiveField(e.Field) || IsSensitiveField(e.FieldPath) {
		return RedactedValue
	}
	return e.Value
}

// IsSensitiveField checks if a field name matches any sensitive field pattern.
// Matching is case-insensitive and checks if the pattern is a substring of the field name.
// Returns true if the field should be considered sensitive and its value redacted.
//
// Example:
//
//	IsSensitiveField("Password")     // true (matches "password")
//	IsSensitiveField("user_api_key") // true (matches "api_key")
//	IsSensitiveField("username")     // false
func IsSensitiveField(fieldName string) bool {
	if fieldName == "" {
		return false
	}

	patterns := GetSensitiveFieldPatterns()
	if len(patterns) == 0 {
		return false
	}

	lowerField := strings.ToLower(fieldName)
	for _, pattern := range patterns {
		if strings.Contains(lowerField, strings.ToLower(pattern)) {
			return true
		}
	}
	return false
}

// ErrorList represents a collection of errors that can occur during parsing/validation.
// Provides aggregation, JSON serialization, and structured error reporting capabilities.
type ErrorList []error

func (el ErrorList) Error() string {
	if len(el) == 0 {
		return ""
	}
	if len(el) == 1 {
		return el[0].Error()
	}

	messages := make([]string, 0, len(el))
	for _, err := range el {
		messages = append(messages, err.Error())
	}
	return fmt.Sprintf("multiple errors: %s", strings.Join(messages, "; "))
}

// Add appends an error to the ErrorList
// If the error is itself an ErrorList, it flattens the errors to avoid nesting
func (el *ErrorList) Add(err error) {
	if err != nil {
		// Check if the error is another ErrorList and flatten it
		if nestedErrorList, ok := err.(ErrorList); ok {
			*el = append(*el, nestedErrorList...)
		} else {
			*el = append(*el, err)
		}
	}
}

// maxErrors caps the errors one parse reports, so a large input cannot allocate one per element.
const maxErrors = 100

var errTooManyErrors = fmt.Errorf("too many errors: stopped after %d", maxErrors)

// addCapped is Add with the maxErrors cap applied.
func (el *ErrorList) addCapped(err error) {
	if el.full() {
		return
	}
	el.Add(err)
	if len(*el) >= maxErrors {
		*el = append((*el)[:maxErrors:maxErrors], errTooManyErrors)
	}
}

// full reports whether the list has reached the cap and gained its closing entry.
func (el ErrorList) full() bool {
	return len(el) > maxErrors
}

// HasErrors returns true if the ErrorList contains any errors
func (el ErrorList) HasErrors() bool {
	return len(el) > 0
}

// AsError returns the ErrorList as an error if it contains any errors, nil otherwise
func (el ErrorList) AsError() error {
	if el.HasErrors() {
		return el
	}
	return nil
}

// ValidationErrors returns only the ValidationError instances from the ErrorList
func (el ErrorList) ValidationErrors() []*ValidationError {
	var validationErrors []*ValidationError
	for _, err := range el {
		if validationErr, ok := err.(*ValidationError); ok {
			validationErrors = append(validationErrors, validationErr)
		}
	}
	return validationErrors
}

// GroupByField groups validation errors by field path
func (el ErrorList) GroupByField() map[string][]*ValidationError {
	groups := make(map[string][]*ValidationError)
	for _, err := range el {
		if validationErr, ok := err.(*ValidationError); ok {
			fieldPath := validationErr.FieldPath
			if fieldPath == "" {
				fieldPath = validationErr.Field
			}
			groups[fieldPath] = append(groups[fieldPath], validationErr)
		}
	}
	return groups
}

// StructuredErrorReport represents a structured validation error report for JSON serialization.
// Provides a comprehensive, machine-readable format for validation errors suitable for APIs.
type StructuredErrorReport struct {
	Errors []FieldError `json:"errors"`
	Count  int          `json:"count"`
}

// FieldError represents a single field's validation errors.
// Groups all validation errors for a specific field with detailed context information.
type FieldError struct {
	Field     string                `json:"field"`
	FieldPath string                `json:"field_path"`
	Value     interface{}           `json:"value,omitempty"`
	Errors    []ValidationErrorInfo `json:"validation_errors"`
}

// ValidationErrorInfo represents detailed information about a validation error.
// Contains the rule name, message, and optional additional details for comprehensive error reporting.
type ValidationErrorInfo struct {
	Rule    string                 `json:"rule"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// ToStructuredReport converts an ErrorList to a structured error report for JSON serialization
func (el ErrorList) ToStructuredReport() *StructuredErrorReport {
	fieldGroups := el.GroupByField()
	fieldErrors := make([]FieldError, 0, len(fieldGroups))

	for fieldPath, validationErrors := range fieldGroups {
		errorInfos := make([]ValidationErrorInfo, 0, len(validationErrors))
		var field string
		var value interface{}

		for _, validationErr := range validationErrors {
			errorInfos = append(errorInfos, ValidationErrorInfo{
				Rule:    validationErr.Rule,
				Message: validationErr.Message,
				Details: validationErr.Details,
			})

			// Use the first error's field and sanitized value
			if field == "" {
				field = validationErr.Field
				value = validationErr.SanitizedValue()
			}
		}

		fieldErrors = append(fieldErrors, FieldError{
			Field:     field,
			FieldPath: fieldPath,
			Value:     value,
			Errors:    errorInfos,
		})
	}

	return &StructuredErrorReport{
		Errors: fieldErrors,
		Count:  len(fieldErrors),
	}
}

// ToJSON converts an ErrorList to JSON for API responses
func (el ErrorList) ToJSON() ([]byte, error) {
	report := el.ToStructuredReport()
	return json.Marshal(report)
}
