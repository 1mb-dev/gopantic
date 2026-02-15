package tests

import (
	"encoding/json"
	"testing"

	"github.com/1mb-dev/gopantic/pkg/model"
)

func TestParseInto_JSONRawMessage(t *testing.T) {
	type Request struct {
		Name        string          `json:"name" validate:"required"`
		MetadataRaw json.RawMessage `json:"metadata,omitempty"`
	}

	tests := []struct {
		name    string
		input   string
		wantErr bool
		check   func(t *testing.T, req Request)
	}{
		{
			name: "json.RawMessage with nested object",
			input: `{
				"name": "test",
				"metadata": {"key": "value", "nested": {"foo": "bar"}}
			}`,
			wantErr: false,
			check: func(t *testing.T, req Request) {
				if req.Name != "test" {
					t.Errorf("Expected name 'test', got '%s'", req.Name)
				}

				// Verify metadata is preserved as raw JSON
				var metadata map[string]interface{}
				if err := json.Unmarshal(req.MetadataRaw, &metadata); err != nil {
					t.Fatalf("Failed to unmarshal metadata: %v", err)
				}

				if metadata["key"] != "value" {
					t.Errorf("Expected metadata.key='value', got '%v'", metadata["key"])
				}

				nested, ok := metadata["nested"].(map[string]interface{})
				if !ok {
					t.Fatal("Expected nested to be a map")
				}
				if nested["foo"] != "bar" {
					t.Errorf("Expected nested.foo='bar', got '%v'", nested["foo"])
				}
			},
		},
		{
			name: "json.RawMessage with array",
			input: `{
				"name": "test",
				"metadata": [1, 2, 3, "four"]
			}`,
			wantErr: false,
			check: func(t *testing.T, req Request) {
				var metadata []interface{}
				if err := json.Unmarshal(req.MetadataRaw, &metadata); err != nil {
					t.Fatalf("Failed to unmarshal metadata: %v", err)
				}

				if len(metadata) != 4 {
					t.Errorf("Expected 4 items, got %d", len(metadata))
				}
			},
		},
		{
			name: "json.RawMessage with null",
			input: `{
				"name": "test",
				"metadata": null
			}`,
			wantErr: false,
			check: func(t *testing.T, req Request) {
				if len(req.MetadataRaw) != 4 { // "null" is 4 bytes
					t.Errorf("Expected metadata to be 'null', got len=%d", len(req.MetadataRaw))
				}
			},
		},
		{
			name: "json.RawMessage omitted",
			input: `{
				"name": "test"
			}`,
			wantErr: false,
			check: func(t *testing.T, req Request) {
				if len(req.MetadataRaw) != 0 {
					t.Errorf("Expected metadata to be empty, got len=%d", len(req.MetadataRaw))
				}
			},
		},
		{
			name: "validation still works",
			input: `{
				"metadata": {"key": "value"}
			}`,
			wantErr: true, // Missing required 'name' field
			check:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := model.ParseInto[Request]([]byte(tt.input))

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseInto() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.check != nil {
				tt.check(t, req)
			}
		})
	}
}

func TestParseInto_JSONRawMessage_PostgreSQLPattern(t *testing.T) {
	// Common PostgreSQL JSONB pattern
	type Account struct {
		ID          string          `json:"id" validate:"required"`
		Name        string          `json:"name" validate:"required,min=2"`
		MetadataRaw json.RawMessage `json:"metadata,omitempty"`
	}

	input := []byte(`{
		"id": "acc_123",
		"name": "John Doe",
		"metadata": {
			"preferences": {
				"theme": "dark",
				"language": "en"
			},
			"tags": ["vip", "premium"],
			"custom_fields": {
				"department": "Engineering",
				"level": 5
			}
		}
	}`)

	account, err := model.ParseInto[Account](input)
	if err != nil {
		t.Fatalf("ParseInto() failed: %v", err)
	}

	// Verify validation worked
	if account.ID != "acc_123" {
		t.Errorf("Expected id='acc_123', got '%s'", account.ID)
	}
	if account.Name != "John Doe" {
		t.Errorf("Expected name='John Doe', got '%s'", account.Name)
	}

	// Verify raw JSON is preserved
	var metadata map[string]interface{}
	if err := json.Unmarshal(account.MetadataRaw, &metadata); err != nil {
		t.Fatalf("Failed to unmarshal metadata: %v", err)
	}

	prefs, ok := metadata["preferences"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected preferences to be a map")
	}
	if prefs["theme"] != "dark" {
		t.Errorf("Expected theme='dark', got '%v'", prefs["theme"])
	}

	tags, ok := metadata["tags"].([]interface{})
	if !ok {
		t.Fatal("Expected tags to be an array")
	}
	if len(tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(tags))
	}
}

func TestParseInto_MultipleRawMessages(t *testing.T) {
	type ComplexStruct struct {
		Name      string          `json:"name" validate:"required"`
		Config    json.RawMessage `json:"config,omitempty"`
		Metadata  json.RawMessage `json:"metadata,omitempty"`
		ExtraData json.RawMessage `json:"extra_data,omitempty"`
	}

	input := []byte(`{
		"name": "test",
		"config": {"enabled": true, "timeout": 30},
		"metadata": ["tag1", "tag2"],
		"extra_data": "just a string"
	}`)

	result, err := model.ParseInto[ComplexStruct](input)
	if err != nil {
		t.Fatalf("ParseInto() failed: %v", err)
	}

	// Verify all raw messages are preserved
	var config map[string]interface{}
	if err := json.Unmarshal(result.Config, &config); err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}
	if config["enabled"] != true {
		t.Error("Config not preserved correctly")
	}

	var metadata []interface{}
	if err := json.Unmarshal(result.Metadata, &metadata); err != nil {
		t.Fatalf("Failed to unmarshal metadata: %v", err)
	}
	if len(metadata) != 2 {
		t.Errorf("Expected 2 metadata items, got %d", len(metadata))
	}

	var extraData string
	if err := json.Unmarshal(result.ExtraData, &extraData); err != nil {
		t.Fatalf("Failed to unmarshal extra_data: %v", err)
	}
	if extraData != "just a string" {
		t.Errorf("Expected 'just a string', got '%s'", extraData)
	}
}

func TestValidate_WithJSONRawMessage(t *testing.T) {
	// Test that Validate works with json.RawMessage fields
	type Request struct {
		Name        string          `json:"name" validate:"required"`
		MetadataRaw json.RawMessage `json:"metadata,omitempty"`
	}

	// Populate struct directly (not via ParseInto)
	req := Request{
		Name:        "test",
		MetadataRaw: json.RawMessage(`{"key": "value"}`),
	}

	err := model.Validate(&req)
	if err != nil {
		t.Errorf("Validate() failed: %v", err)
	}

	// Test validation failure
	invalidReq := Request{
		MetadataRaw: json.RawMessage(`{"key": "value"}`),
	}

	err = model.Validate(&invalidReq)
	if err == nil {
		t.Error("Expected validation error for missing required field")
	}
}

func TestValidate_WithStandardUnmarshal(t *testing.T) {
	// Test the recommended pattern from Issue #11:
	// Use standard json.Unmarshal + gopantic Validate
	type Request struct {
		Name        string          `json:"name" validate:"required,min=2"`
		MetadataRaw json.RawMessage `json:"metadata,omitempty"`
	}

	input := []byte(`{
		"name": "test",
		"metadata": {"complex": {"nested": "data"}}
	}`)

	// Step 1: Standard unmarshal
	var req Request
	if err := json.Unmarshal(input, &req); err != nil {
		t.Fatalf("json.Unmarshal() failed: %v", err)
	}

	// Step 2: Validate with gopantic
	if err := model.Validate(&req); err != nil {
		t.Fatalf("Validate() failed: %v", err)
	}

	// Verify metadata is preserved
	var metadata map[string]interface{}
	if err := json.Unmarshal(req.MetadataRaw, &metadata); err != nil {
		t.Fatalf("Failed to unmarshal metadata: %v", err)
	}

	complex, ok := metadata["complex"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected complex to be a map")
	}
	if complex["nested"] != "data" {
		t.Errorf("Metadata not preserved correctly")
	}
}
