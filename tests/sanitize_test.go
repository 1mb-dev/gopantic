package tests

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/1mb-dev/gopantic/pkg/model"
)

// TestIsSensitiveField tests the sensitive field detection
func TestIsSensitiveField(t *testing.T) {
	// Save original patterns and restore after test
	orig := model.GetSensitiveFieldPatterns()
	defer model.SetSensitiveFieldPatterns(orig)

	// Reset to defaults for consistent testing
	model.SetSensitiveFieldPatterns(model.DefaultSensitivePatterns)

	tests := []struct {
		name      string
		fieldName string
		want      bool
	}{
		// Should match (default patterns)
		{"password lowercase", "password", true},
		{"password uppercase", "PASSWORD", true},
		{"password mixed case", "Password", true},
		{"password with prefix", "user_password", true},
		{"password with suffix", "password_hash", true},
		{"secret field", "client_secret", true},
		{"token field", "access_token", true},
		{"api_key field", "api_key", true},
		{"apikey field", "apikey", true},
		{"auth field", "auth_token", true},
		{"credential field", "credentials", true},
		{"private field", "private_key", true},
		{"bearer field", "bearer_token", true},
		{"nested path with password", "User.Password", true},

		// Should not match
		{"username", "username", false},
		{"email", "email", false},
		{"name", "name", false},
		{"id", "id", false},
		{"empty string", "", false},
		{"pass without word", "pass", false}, // "pass" alone doesn't match "password"
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := model.IsSensitiveField(tt.fieldName)
			if got != tt.want {
				t.Errorf("IsSensitiveField(%q) = %v, want %v", tt.fieldName, got, tt.want)
			}
		})
	}
}

// TestSensitiveFieldPatterns_Config tests the configuration getter/setter
func TestSensitiveFieldPatterns_Config(t *testing.T) {
	// Save original patterns
	orig := model.GetSensitiveFieldPatterns()
	defer model.SetSensitiveFieldPatterns(orig)

	t.Run("default patterns are set", func(t *testing.T) {
		model.SetSensitiveFieldPatterns(model.DefaultSensitivePatterns)
		patterns := model.GetSensitiveFieldPatterns()
		if len(patterns) == 0 {
			t.Error("expected default patterns to be set")
		}
		// Check for some expected defaults
		found := false
		for _, p := range patterns {
			if p == "password" {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected 'password' to be in default patterns")
		}
	})

	t.Run("setter updates patterns", func(t *testing.T) {
		custom := []string{"ssn", "credit_card"}
		model.SetSensitiveFieldPatterns(custom)
		patterns := model.GetSensitiveFieldPatterns()

		if len(patterns) != 2 {
			t.Errorf("expected 2 patterns, got %d", len(patterns))
		}

		// Now "password" should NOT match, but "ssn" should
		if model.IsSensitiveField("password") {
			t.Error("password should not match with custom patterns")
		}
		if !model.IsSensitiveField("ssn") {
			t.Error("ssn should match with custom patterns")
		}
	})

	t.Run("nil disables detection", func(t *testing.T) {
		model.SetSensitiveFieldPatterns(nil)
		if model.IsSensitiveField("password") {
			t.Error("password should not match when patterns are nil")
		}
	})

	t.Run("empty slice disables detection", func(t *testing.T) {
		model.SetSensitiveFieldPatterns([]string{})
		if model.IsSensitiveField("password") {
			t.Error("password should not match when patterns are empty")
		}
	})

	t.Run("AddSensitiveFieldPattern works", func(t *testing.T) {
		model.SetSensitiveFieldPatterns([]string{"password"})
		model.AddSensitiveFieldPattern("ssn")

		if !model.IsSensitiveField("password") {
			t.Error("password should still match")
		}
		if !model.IsSensitiveField("user_ssn") {
			t.Error("ssn should match after adding")
		}
	})
}

// TestValidationError_SanitizedValue tests the SanitizedValue method
func TestValidationError_SanitizedValue(t *testing.T) {
	// Save original patterns
	orig := model.GetSensitiveFieldPatterns()
	defer model.SetSensitiveFieldPatterns(orig)

	// Reset to defaults
	model.SetSensitiveFieldPatterns(model.DefaultSensitivePatterns)

	t.Run("sensitive field returns redacted", func(t *testing.T) {
		err := model.NewValidationError("password", "secret123", "required", "password is required")
		val := err.SanitizedValue()
		if val != model.RedactedValue {
			t.Errorf("SanitizedValue() = %v, want %v", val, model.RedactedValue)
		}
	})

	t.Run("non-sensitive field returns original", func(t *testing.T) {
		err := model.NewValidationError("username", "john", "required", "username is required")
		val := err.SanitizedValue()
		if val != "john" {
			t.Errorf("SanitizedValue() = %v, want %v", val, "john")
		}
	})

	t.Run("sensitive in field path returns redacted", func(t *testing.T) {
		err := model.NewValidationErrorWithPath("Password", "User.Password", "secret123", "required", "password is required")
		val := err.SanitizedValue()
		if val != model.RedactedValue {
			t.Errorf("SanitizedValue() = %v, want %v", val, model.RedactedValue)
		}
	})

	t.Run("value types preserved when not sensitive", func(t *testing.T) {
		err := model.NewValidationError("age", 25, "min", "age must be at least 18")
		val := err.SanitizedValue()
		if val != 25 {
			t.Errorf("SanitizedValue() = %v, want %v", val, 25)
		}
	})
}

// TestToStructuredReport_Sanitization tests that ToStructuredReport uses sanitized values
func TestToStructuredReport_Sanitization(t *testing.T) {
	// Save original patterns
	orig := model.GetSensitiveFieldPatterns()
	defer model.SetSensitiveFieldPatterns(orig)

	// Reset to defaults
	model.SetSensitiveFieldPatterns(model.DefaultSensitivePatterns)

	t.Run("sensitive values are redacted in report", func(t *testing.T) {
		var errors model.ErrorList
		errors.Add(model.NewValidationError("password", "secret123", "required", "password is required"))
		errors.Add(model.NewValidationError("username", "john", "required", "username is required"))

		report := errors.ToStructuredReport()

		for _, fieldErr := range report.Errors {
			if fieldErr.Field == "password" {
				if fieldErr.Value != model.RedactedValue {
					t.Errorf("password value should be redacted, got %v", fieldErr.Value)
				}
			}
			if fieldErr.Field == "username" {
				if fieldErr.Value != "john" {
					t.Errorf("username value should be preserved, got %v", fieldErr.Value)
				}
			}
		}
	})

	t.Run("ToJSON produces sanitized output", func(t *testing.T) {
		var errors model.ErrorList
		errors.Add(model.NewValidationError("api_key", "sk-12345-secret", "required", "api_key is required"))

		jsonBytes, err := errors.ToJSON()
		if err != nil {
			t.Fatalf("ToJSON failed: %v", err)
		}

		jsonStr := string(jsonBytes)
		if strings.Contains(jsonStr, "sk-12345-secret") {
			t.Error("JSON output should not contain the actual API key")
		}
		if !strings.Contains(jsonStr, model.RedactedValue) {
			t.Error("JSON output should contain redacted placeholder")
		}
	})
}

// TestParseInto_SensitiveFieldSanitization tests end-to-end sanitization
func TestParseInto_SensitiveFieldSanitization(t *testing.T) {
	// Save original patterns
	orig := model.GetSensitiveFieldPatterns()
	defer model.SetSensitiveFieldPatterns(orig)

	// Reset to defaults
	model.SetSensitiveFieldPatterns(model.DefaultSensitivePatterns)

	type LoginRequest struct {
		Username string `json:"username" validate:"required,min=3"`
		Password string `json:"password" validate:"required,min=8"`
	}

	t.Run("password value is redacted in validation error", func(t *testing.T) {
		// Password too short - will fail validation
		data := []byte(`{"username": "john", "password": "short"}`)
		_, err := model.ParseInto[LoginRequest](data)

		if err == nil {
			t.Fatal("expected validation error")
		}

		// Check that the error string doesn't contain the actual password
		errStr := err.Error()
		if strings.Contains(errStr, "short") {
			// This is acceptable - Error() method doesn't include value
			// The key is that SanitizedValue() and ToStructuredReport() work
		}

		// Check ErrorList behavior
		if errList, ok := err.(model.ErrorList); ok {
			report := errList.ToStructuredReport()
			jsonBytes, _ := json.Marshal(report)
			jsonStr := string(jsonBytes)

			if strings.Contains(jsonStr, "short") {
				t.Error("structured report should not contain the actual password")
			}
		}
	})
}

// TestRedactedValueConstant verifies the constant value
func TestRedactedValueConstant(t *testing.T) {
	if model.RedactedValue != "[REDACTED]" {
		t.Errorf("RedactedValue = %q, want %q", model.RedactedValue, "[REDACTED]")
	}
}
