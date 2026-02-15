package tests

import (
	"encoding/json"
	"testing"

	"github.com/1mb-dev/gopantic/pkg/model"
	"github.com/go-playground/validator/v10"
)

// Benchmark test structures - use Bench prefix to avoid conflicts

// BenchSimpleUser represents a basic user struct for validation benchmarks
type BenchSimpleUser struct {
	ID    int    `json:"id" validate:"required,min=1"`
	Name  string `json:"name" validate:"required,min=2,max=100"`
	Email string `json:"email" validate:"required,email"`
	Age   int    `json:"age" validate:"min=0,max=150"`
}

// BenchComplexUser represents a more complex struct with nested validation
type BenchComplexUser struct {
	ID        int            `json:"id" validate:"required,min=1"`
	Name      string         `json:"name" validate:"required,min=2,max=100"`
	Email     string         `json:"email" validate:"required,email"`
	Age       int            `json:"age" validate:"min=0,max=150"`
	Active    bool           `json:"active"`
	Tags      []string       `json:"tags" validate:"max=10"`
	Profile   BenchProfile   `json:"profile"`
	Addresses []BenchAddress `json:"addresses" validate:"max=5"`
}

type BenchProfile struct {
	Bio     string `json:"bio" validate:"max=500"`
	Website string `json:"website"`
}

type BenchAddress struct {
	Street  string `json:"street" validate:"required"`
	City    string `json:"city" validate:"required"`
	Country string `json:"country" validate:"required,alpha"`
	Zip     string `json:"zip" validate:"required,alphanum"`
}

// Test data
var benchSimpleUserJSON = []byte(`{"id": 1, "name": "John Doe", "email": "john@example.com", "age": 30}`)
var benchSimpleUserValid = BenchSimpleUser{ID: 1, Name: "John Doe", Email: "john@example.com", Age: 30}

var benchComplexUserJSON = []byte(`{
	"id": 1,
	"name": "John Doe",
	"email": "john@example.com",
	"age": 30,
	"active": true,
	"tags": ["developer", "golang"],
	"profile": {
		"bio": "Software developer",
		"website": "https://example.com"
	},
	"addresses": [
		{"street": "123 Main St", "city": "Boston", "country": "USA", "zip": "02101"}
	]
}`)

var benchComplexUserValid = BenchComplexUser{
	ID:     1,
	Name:   "John Doe",
	Email:  "john@example.com",
	Age:    30,
	Active: true,
	Tags:   []string{"developer", "golang"},
	Profile: BenchProfile{
		Bio:     "Software developer",
		Website: "https://example.com",
	},
	Addresses: []BenchAddress{
		{Street: "123 Main St", City: "Boston", Country: "USA", Zip: "02101"},
	},
}

// ============================================================================
// Validation-Only Benchmarks
// ============================================================================

// BenchmarkValidation_Gopantic_Simple benchmarks gopantic validation
func BenchmarkValidation_Gopantic_Simple(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = model.Validate(&benchSimpleUserValid)
	}
}

// BenchmarkValidation_Playground_Simple benchmarks go-playground/validator
func BenchmarkValidation_Playground_Simple(b *testing.B) {
	validate := validator.New()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validate.Struct(&benchSimpleUserValid)
	}
}

// BenchmarkValidation_Gopantic_Complex benchmarks gopantic with complex struct
func BenchmarkValidation_Gopantic_Complex(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = model.Validate(&benchComplexUserValid)
	}
}

// BenchmarkValidation_Playground_Complex benchmarks go-playground/validator with complex struct
func BenchmarkValidation_Playground_Complex(b *testing.B) {
	validate := validator.New()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validate.Struct(&benchComplexUserValid)
	}
}

// ============================================================================
// Parse + Validate Benchmarks (gopantic's strength)
// ============================================================================

// BenchmarkParseValidate_Gopantic_Simple benchmarks gopantic ParseInto (parse + validate)
func BenchmarkParseValidate_Gopantic_Simple(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = model.ParseInto[BenchSimpleUser](benchSimpleUserJSON)
	}
}

// BenchmarkParseValidate_StdJSON_Simple benchmarks encoding/json + playground/validator
func BenchmarkParseValidate_StdJSON_Simple(b *testing.B) {
	validate := validator.New()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var user BenchSimpleUser
		_ = json.Unmarshal(benchSimpleUserJSON, &user)
		_ = validate.Struct(&user)
	}
}

// BenchmarkParseValidate_Gopantic_Complex benchmarks gopantic with complex struct
func BenchmarkParseValidate_Gopantic_Complex(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = model.ParseInto[BenchComplexUser](benchComplexUserJSON)
	}
}

// BenchmarkParseValidate_StdJSON_Complex benchmarks encoding/json + playground/validator
func BenchmarkParseValidate_StdJSON_Complex(b *testing.B) {
	validate := validator.New()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var user BenchComplexUser
		_ = json.Unmarshal(benchComplexUserJSON, &user)
		_ = validate.Struct(&user)
	}
}

// ============================================================================
// Parse-Only Benchmarks (no validation)
// ============================================================================

// BenchSimpleUserNoValidation has no validation tags
type BenchSimpleUserNoValidation struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Age   int    `json:"age"`
}

// BenchmarkParse_Gopantic_NoValidation benchmarks gopantic without validation
func BenchmarkParse_Gopantic_NoValidation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = model.ParseInto[BenchSimpleUserNoValidation](benchSimpleUserJSON)
	}
}

// BenchmarkParse_StdJSON benchmarks encoding/json directly
func BenchmarkParse_StdJSON(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var user BenchSimpleUserNoValidation
		_ = json.Unmarshal(benchSimpleUserJSON, &user)
	}
}

// ============================================================================
// Type Coercion Benchmarks (gopantic unique feature)
// ============================================================================

var benchCoercionJSON = []byte(`{"id": "123", "name": "John", "email": "john@example.com", "age": "30"}`)

// BenchmarkCoercion_Gopantic benchmarks gopantic with type coercion
func BenchmarkCoercion_Gopantic(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = model.ParseInto[BenchSimpleUser](benchCoercionJSON)
	}
}

// Note: encoding/json would fail on this input since types don't match
// gopantic automatically coerces "123" -> 123 and "30" -> 30

// ============================================================================
// Format Detection Benchmarks
// ============================================================================

var benchYamlData = []byte(`
id: 1
name: John Doe
email: john@example.com
age: 30
`)

// BenchmarkFormat_JSON benchmarks JSON format parsing
func BenchmarkFormat_JSON(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = model.ParseInto[BenchSimpleUser](benchSimpleUserJSON)
	}
}

// BenchmarkFormat_YAML benchmarks YAML format parsing
func BenchmarkFormat_YAML(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = model.ParseInto[BenchSimpleUser](benchYamlData)
	}
}

// BenchmarkFormatDetect benchmarks format auto-detection
func BenchmarkFormatDetect(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = model.DetectFormat(benchSimpleUserJSON)
	}
}

// ============================================================================
// Parallel Benchmarks
// ============================================================================

// BenchmarkParallel_Gopantic_Simple benchmarks gopantic under concurrent load
func BenchmarkParallel_Gopantic_Simple(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = model.ParseInto[BenchSimpleUser](benchSimpleUserJSON)
		}
	})
}

// BenchmarkParallel_StdJSON_Simple benchmarks encoding/json + validator under concurrent load
func BenchmarkParallel_StdJSON_Simple(b *testing.B) {
	validate := validator.New()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var user BenchSimpleUser
			_ = json.Unmarshal(benchSimpleUserJSON, &user)
			_ = validate.Struct(&user)
		}
	})
}
