package tests

import (
	"strings"
	"testing"

	"github.com/1mb-dev/gopantic/pkg/model"
)

// Core validation test structs
type ValidatedUser struct {
	ID       int    `json:"id" validate:"required,min=1"`
	Username string `json:"username" validate:"required,min=3,max=20,alphanum"`
	Email    string `json:"email" validate:"required,email"`
	Age      int    `json:"age" validate:"min=18,max=120"`
	Name     string `json:"name" validate:"required,min=2,alpha"`
}

type ValidatedProduct struct {
	SKU   string  `json:"sku" validate:"required,length=8"`
	Name  string  `json:"name" validate:"required,min=1,max=100"`
	Price float64 `json:"price" validate:"required,min=0.01"`
}

func TestValidation_Success(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
	}{
		{
			name:  "valid user",
			input: []byte(`{"id":1,"username":"john123","email":"john@example.com","age":25,"name":"John"}`),
		},
		{
			name:  "valid product",
			input: []byte(`{"sku":"ABC12345","name":"Widget","price":29.99}`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "valid user" {
				_, err := model.ParseInto[ValidatedUser](tt.input)
				if err != nil {
					t.Errorf("ParseInto() unexpected error = %v", err)
				}
			} else {
				_, err := model.ParseInto[ValidatedProduct](tt.input)
				if err != nil {
					t.Errorf("ParseInto() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestValidation_Failures(t *testing.T) {
	tests := []struct {
		name            string
		input           []byte
		wantErrContains string
	}{
		{
			name:            "required field missing",
			input:           []byte(`{"username":"john","email":"john@example.com","age":25,"name":"John"}`), // missing id
			wantErrContains: "required",
		},
		{
			name:            "min validation failure",
			input:           []byte(`{"id":0,"username":"john","email":"john@example.com","age":25,"name":"John"}`), // id < 1
			wantErrContains: "at least",
		},
		{
			name:            "max validation failure",
			input:           []byte(`{"id":1,"username":"verylongusernamethatexceedslimit","email":"john@example.com","age":25,"name":"John"}`),
			wantErrContains: "at most",
		},
		{
			name:            "email validation failure",
			input:           []byte(`{"id":1,"username":"john","email":"invalid-email","age":25,"name":"John"}`),
			wantErrContains: "email",
		},
		{
			name:            "alpha validation failure",
			input:           []byte(`{"id":1,"username":"john123","email":"john@example.com","age":25,"name":"John123"}`), // name has numbers
			wantErrContains: "alpha",
		},
		{
			name:            "alphanum validation failure",
			input:           []byte(`{"id":1,"username":"john@#$","email":"john@example.com","age":25,"name":"John"}`), // username has special chars
			wantErrContains: "alphanum",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := model.ParseInto[ValidatedUser](tt.input)
			if err == nil {
				t.Errorf("ParseInto() expected error but got none")
				return
			}
			if !strings.Contains(strings.ToLower(err.Error()), tt.wantErrContains) {
				t.Errorf("ParseInto() error = %v, want error containing %v", err, tt.wantErrContains)
			}
		})
	}
}

func TestValidation_WithCoercion(t *testing.T) {
	// Test that validation happens after type coercion
	input := []byte(`{"id":"1","username":"john123","email":"john@example.com","age":"25","name":"John"}`)

	_, err := model.ParseInto[ValidatedUser](input)
	if err != nil {
		t.Errorf("ParseInto() unexpected error = %v", err)
	}
}

func TestValidation_MultipleErrors(t *testing.T) {
	input := []byte(`{"id":0,"username":"ab","email":"invalid","age":150,"name":""}`) // Multiple validation failures

	_, err := model.ParseInto[ValidatedUser](input)
	if err == nil {
		t.Errorf("ParseInto() expected error but got none")
		return
	}

	errorStr := strings.ToLower(err.Error())
	if !strings.Contains(errorStr, "multiple errors") {
		t.Errorf("ParseInto() expected 'multiple errors' in error message, got: %v", err)
	}
}

func TestValidation_LengthValidator(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		wantErr bool
	}{
		{
			name:    "exact length valid",
			input:   []byte(`{"sku":"ABC12345","name":"Widget","price":1.0}`), // SKU is exactly 8 chars
			wantErr: false,
		},
		{
			name:    "length too short",
			input:   []byte(`{"sku":"ABC123","name":"Widget","price":1.0}`), // SKU is 6 chars
			wantErr: true,
		},
		{
			name:    "length too long",
			input:   []byte(`{"sku":"ABC123456","name":"Widget","price":1.0}`), // SKU is 9 chars
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := model.ParseInto[ValidatedProduct](tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseInto() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
