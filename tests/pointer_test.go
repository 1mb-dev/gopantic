package tests

import (
	"reflect"
	"testing"

	"github.com/1mb-dev/gopantic/pkg/model"
)

// Structs for pointer testing
type PersonWithPointers struct {
	Name   string   `json:"name" validate:"required"`
	Age    *int     `json:"age"`
	Email  *string  `json:"email"`
	Height *float64 `json:"height"`
	Active *bool    `json:"active"`
	Bio    *string  `json:"bio"`
}

func TestParseInto_PointerTypes(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		want    PersonWithPointers
		wantErr bool
	}{
		{
			name: "all pointer fields present",
			input: []byte(`{
				"name": "Alice",
				"age": "25",
				"email": "alice@example.com",
				"height": "5.6",
				"active": "true",
				"bio": "Software engineer"
			}`),
			want: PersonWithPointers{
				Name:   "Alice",
				Age:    intPtr(25),
				Email:  stringPtr("alice@example.com"),
				Height: float64Ptr(5.6),
				Active: boolPtr(true),
				Bio:    stringPtr("Software engineer"),
			},
			wantErr: false,
		},
		{
			name: "some pointer fields missing (should be nil)",
			input: []byte(`{
				"name": "Bob",
				"age": 30,
				"height": 6.0
			}`),
			want: PersonWithPointers{
				Name:   "Bob",
				Age:    intPtr(30),
				Email:  nil,
				Height: float64Ptr(6.0),
				Active: nil,
				Bio:    nil,
			},
			wantErr: false,
		},
		{
			name: "explicit null values for pointers",
			input: []byte(`{
				"name": "Charlie",
				"age": null,
				"email": null,
				"height": null,
				"active": null,
				"bio": null
			}`),
			want: PersonWithPointers{
				Name:   "Charlie",
				Age:    nil,
				Email:  nil,
				Height: nil,
				Active: nil,
				Bio:    nil,
			},
			wantErr: false,
		},
		{
			name: "type coercion with pointers",
			input: []byte(`{
				"name": "Diana",
				"age": "35",
				"email": "diana@test.com",
				"height": "5.4",
				"active": 1,
				"bio": 42
			}`),
			want: PersonWithPointers{
				Name:   "Diana",
				Age:    intPtr(35),
				Email:  stringPtr("diana@test.com"),
				Height: float64Ptr(5.4),
				Active: boolPtr(true),
				Bio:    stringPtr("42"),
			},
			wantErr: false,
		},
		{
			name:    "invalid coercion for pointer field",
			input:   []byte(`{"name": "Error", "age": "not-a-number"}`),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := model.ParseInto[PersonWithPointers](tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseInto() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseInto() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Test pointer validation
func TestParseInto_PointerValidation(t *testing.T) {
	type PersonWithValidatedPointers struct {
		Name  string  `json:"name" validate:"required"`
		Age   *int    `json:"age" validate:"min=18"`
		Email *string `json:"email" validate:"email"`
	}

	tests := []struct {
		name    string
		input   []byte
		wantErr bool
	}{
		{
			name: "valid pointer values",
			input: []byte(`{
				"name": "Valid Person",
				"age": 25,
				"email": "valid@example.com"
			}`),
			wantErr: false,
		},
		{
			name: "nil pointers (no validation error)",
			input: []byte(`{
				"name": "Person with nil pointers"
			}`),
			wantErr: false,
		},
		{
			name:    "invalid age in pointer",
			input:   []byte(`{"name": "Invalid Age", "age": 15}`),
			wantErr: true,
		},
		{
			name:    "invalid email in pointer",
			input:   []byte(`{"name": "Invalid Email", "email": "not-an-email"}`),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := model.ParseInto[PersonWithValidatedPointers](tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseInto() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Helper functions to create pointers
func intPtr(v int) *int {
	return &v
}

func stringPtr(v string) *string {
	return &v
}

func float64Ptr(v float64) *float64 {
	return &v
}

func boolPtr(v bool) *bool {
	return &v
}
