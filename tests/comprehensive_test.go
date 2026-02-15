package tests

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/1mb-dev/gopantic/pkg/model"
)

// Note: Test structures (E2EUser, Profile, Settings, APIResponse, AppConfig, etc.) are defined in fixtures.go

// Test comprehensive JSON parsing with validation
func TestComprehensive_CompleteUserParsing(t *testing.T) {
	testCases := []struct {
		name        string
		jsonData    string
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid complete user",
			jsonData: `{
				"id": 123,
				"username": "johndoe",
				"email": "john@example.com",
				"first_name": "John",
				"last_name": "Doe",
				"age": 30,
				"is_active": true,
				"profile": {
					"bio": "Software developer and tech enthusiast",
					"website": "https://johndoe.dev",
					"location": "San Francisco, CA",
					"skills": ["Go", "Python", "JavaScript"],
					"languages": ["English", "Spanish"]
				},
				"settings": {
					"theme": "dark",
					"notifications": {
						"email": true,
						"push": false,
						"sms": true
					},
					"privacy": {
						"profile_visible": true,
						"email_visible": false,
						"show_online": true
					}
				},
				"created_at": "2023-01-15T10:30:00Z"
			}`,
			expectError: false,
		},
		{
			name: "User with missing required nested field",
			jsonData: `{
				"id": 124,
				"username": "janedoe",
				"email": "jane@example.com",
				"first_name": "Jane",
				"last_name": "Doe",
				"age": 25,
				"is_active": true,
				"profile": {
					"bio": "Designer and artist",
					"location": "New York, NY",
					"skills": ["Design", "Art"]
				},
				"created_at": "2023-02-20T14:45:00Z"
			}`,
			expectError: true,
			errorMsg:    "Languages",
		},
		{
			name: "User with invalid email",
			jsonData: `{
				"id": 125,
				"username": "invaliduser",
				"email": "not-an-email",
				"first_name": "Invalid",
				"last_name": "User",
				"age": 20,
				"is_active": true,
				"profile": {
					"bio": "Test user",
					"location": "Test City",
					"skills": ["Testing"],
					"languages": ["English"]
				},
				"created_at": "2023-03-10T09:00:00Z"
			}`,
			expectError: true,
			errorMsg:    "email",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			user, err := model.ParseInto[E2EUser]([]byte(tc.jsonData))

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tc.errorMsg != "" && !strings.Contains(err.Error(), tc.errorMsg) {
					t.Errorf("Expected error to contain '%s', got: %v", tc.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				} else {
					// Validate parsing worked correctly
					if user.Username == "" {
						t.Error("Username should not be empty")
					}
					if user.Profile.Languages == nil {
						t.Error("Languages should not be nil")
					}
				}
			}
		})
	}
}

// Test API response parsing with generics
func TestComprehensive_APIResponseParsing(t *testing.T) {
	t.Run("Successful API response", func(t *testing.T) {
		responseData := `{
			"success": true,
			"data": {
				"id": 456,
				"username": "apiuser",
				"email": "api@example.com",
				"first_name": "API",
				"last_name": "User",
				"age": 28,
				"is_active": true,
				"profile": {
					"bio": "API testing user",
					"location": "Cloud",
					"skills": ["API Testing"],
					"languages": ["English"]
				},
				"created_at": "2023-06-01T12:00:00Z"
			},
			"error": null,
			"meta": {
				"request_id": "req-12345",
				"version": "v1.0",
				"process_time_ms": 150
			},
			"timestamp": "2023-06-01T12:00:01Z"
		}`

		response, err := model.ParseInto[APIResponse[E2EUser]]([]byte(responseData))
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if !response.Success {
			t.Error("Expected success to be true")
		}
		if response.Data == nil {
			t.Fatal("Expected data to not be nil")
		}
		if response.Data.Username != "apiuser" {
			t.Errorf("Expected username 'apiuser', got '%s'", response.Data.Username)
		}
		if response.Meta.RequestID != "req-12345" {
			t.Errorf("Expected request ID 'req-12345', got '%s'", response.Meta.RequestID)
		}
	})

	t.Run("Error API response", func(t *testing.T) {
		errorResponseData := `{
			"success": false,
			"data": null,
			"error": {
				"code": "VALIDATION_ERROR",
				"message": "Invalid user data provided",
				"details": {
					"field": "email",
					"reason": "invalid format"
				}
			},
			"meta": {
				"request_id": "req-67890",
				"version": "v1.0",
				"process_time_ms": 25
			},
			"timestamp": "2023-06-01T12:01:00Z"
		}`

		response, err := model.ParseInto[APIResponse[E2EUser]]([]byte(errorResponseData))
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if response.Success {
			t.Error("Expected success to be false")
		}
		if response.Data != nil {
			t.Error("Expected data to be nil")
		}
		if response.Error == nil {
			t.Fatal("Expected error to not be nil")
		}
		if response.Error.Code != "VALIDATION_ERROR" {
			t.Errorf("Expected error code 'VALIDATION_ERROR', got '%s'", response.Error.Code)
		}
	})
}

// Test configuration parsing from JSON
func TestComprehensive_ConfigurationParsing(t *testing.T) {
	configData := `{
		"name": "MyApp",
		"version": "1.2.3",
		"environment": "production",
		"database": {
			"host": "db.example.com",
			"port": 5432,
			"username": "app_user",
			"password": "secure_password123",
			"database": "myapp_db",
			"ssl": true,
			"timeout": 30000
		},
		"debug": false,
		"features": ["feature_a", "feature_b", "feature_c"]
	}`

	config, err := model.ParseInto[AppConfig]([]byte(configData))
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Validate parsed config
	if config.Name != "MyApp" {
		t.Errorf("Expected name 'MyApp', got '%s'", config.Name)
	}
	if config.Database.Host != "db.example.com" {
		t.Errorf("Expected database host 'db.example.com', got '%s'", config.Database.Host)
	}
	if config.Database.Port != 5432 {
		t.Errorf("Expected database port 5432, got %d", config.Database.Port)
	}
	if !config.Database.SSL {
		t.Error("Expected SSL to be true")
	}
	if len(config.Features) != 3 {
		t.Errorf("Expected 3 features, got %d", len(config.Features))
	}
}

// Test cross-field validation
func TestComprehensive_CrossFieldValidation(t *testing.T) {
	// Register a cross-field validator for testing
	model.RegisterGlobalCrossFieldFunc("profile_email_match", func(fieldName string, fieldValue interface{}, structValue reflect.Value, params map[string]interface{}) error {
		emailField := structValue.FieldByName("Email")
		if !emailField.IsValid() {
			return model.NewValidationError(fieldName, fieldValue, "profile_email_match", "Email field not found")
		}

		email := emailField.Interface().(string)
		profileEmail, ok := fieldValue.(string)
		if !ok {
			return nil // Skip if not string
		}

		if email != profileEmail {
			return model.NewValidationError(fieldName, fieldValue, "profile_email_match", "profile email must match user email")
		}

		return nil
	})

	type UserWithCrossValidation struct {
		Email        string `json:"email" validate:"required,email"`
		ProfileEmail string `json:"profile_email" validate:"required,profile_email_match"`
	}

	t.Run("Valid cross-field data", func(t *testing.T) {
		validData := `{
			"email": "test@example.com",
			"profile_email": "test@example.com"
		}`

		user, err := model.ParseInto[UserWithCrossValidation]([]byte(validData))
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if user.Email != "test@example.com" {
			t.Errorf("Expected email 'test@example.com', got '%s'", user.Email)
		}
	})

	t.Run("Invalid cross-field data", func(t *testing.T) {
		invalidData := `{
			"email": "test@example.com",
			"profile_email": "different@example.com"
		}`

		_, err := model.ParseInto[UserWithCrossValidation]([]byte(invalidData))
		if err == nil {
			t.Error("Expected cross-field validation error")
		} else if !strings.Contains(err.Error(), "must match") {
			t.Errorf("Expected 'must match' in error, got: %v", err)
		}
	})
}

// Test error reporting and serialization
func TestComprehensive_ErrorReporting(t *testing.T) {
	invalidUserData := `{
		"id": 0,
		"username": "ab",
		"email": "invalid-email",
		"first_name": "",
		"last_name": "",
		"age": 150,
		"is_active": true,
		"profile": {
			"bio": "",
			"location": "",
			"skills": [],
			"languages": []
		},
		"created_at": "invalid-date"
	}`

	_, err := model.ParseInto[E2EUser]([]byte(invalidUserData))
	if err == nil {
		t.Fatal("Expected validation errors")
	}

	// Test error serialization if ErrorList
	if errorList, ok := err.(model.ErrorList); ok {
		// Test JSON serialization
		jsonData, jsonErr := errorList.ToJSON()
		if jsonErr != nil {
			t.Fatalf("Failed to serialize errors to JSON: %v", jsonErr)
		}

		// Verify JSON structure
		var errorReport model.StructuredErrorReport
		if err := json.Unmarshal(jsonData, &errorReport); err != nil {
			t.Fatalf("Failed to parse error JSON: %v", err)
		}

		if errorReport.Count == 0 {
			t.Error("Expected error count to be greater than 0")
		}

		if len(errorReport.Errors) == 0 {
			t.Error("Expected errors array to not be empty")
		}

		t.Logf("Error report: %d field errors found", errorReport.Count)
	}
}
