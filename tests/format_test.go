package tests

import (
	"testing"

	"github.com/1mb-dev/gopantic/pkg/model"
)

func TestDetectFormat(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected model.Format
	}{
		{
			name:     "JSON object",
			input:    []byte(`{"name": "John", "age": 30}`),
			expected: model.FormatJSON,
		},
		{
			name:     "JSON array",
			input:    []byte(`[{"name": "John"}, {"name": "Jane"}]`),
			expected: model.FormatJSON,
		},
		{
			name:     "YAML with document separator",
			input:    []byte("---\nname: John\nage: 30"),
			expected: model.FormatYAML,
		},
		{
			name:     "YAML without document separator",
			input:    []byte("name: John\nage: 30\nemail: john@example.com"),
			expected: model.FormatYAML,
		},
		{
			name:     "YAML with nested structure",
			input:    []byte("database:\n  host: localhost\n  port: 5432"),
			expected: model.FormatYAML,
		},
		{
			name:     "Empty input",
			input:    []byte(""),
			expected: model.FormatJSON,
		},
		{
			name:     "Whitespace only",
			input:    []byte("   \n  \t  "),
			expected: model.FormatJSON,
		},
		{
			name:     "JSON with leading whitespace",
			input:    []byte("  \n  {\n  \"name\": \"John\"\n}"),
			expected: model.FormatJSON,
		},
		{
			name:     "YAML list",
			input:    []byte("hosts:\n  - api.example.com\n  - cdn.example.com"),
			expected: model.FormatYAML,
		},
		{
			name:     "Ambiguous case defaults to JSON",
			input:    []byte("simple string"),
			expected: model.FormatJSON,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := model.DetectFormat(tt.input)
			if result != tt.expected {
				t.Errorf("DetectFormat() = %v, want %v for input: %q", result, tt.expected, string(tt.input))
			}
		})
	}
}

func TestGetParser(t *testing.T) {
	tests := []struct {
		format   model.Format
		expected model.Format
	}{
		{model.FormatJSON, model.FormatJSON},
		{model.FormatYAML, model.FormatYAML},
	}

	for _, tt := range tests {
		parser := model.GetParser(tt.format)
		if parser.Format() != tt.expected {
			t.Errorf("GetParser(%v).Format() = %v, want %v", tt.format, parser.Format(), tt.expected)
		}
	}
}

func TestFormatParserFunctionality(t *testing.T) {
	jsonData := []byte(`{"name": "John", "age": 30}`)
	yamlData := []byte(`name: John
age: 30`)

	// Test JSON parser
	jsonParser := model.GetParser(model.FormatJSON)
	jsonResult, err := jsonParser.Parse(jsonData)
	if err != nil {
		t.Fatalf("JSON parser failed: %v", err)
	}
	jsonMap, ok := jsonResult.(map[string]interface{})
	if !ok {
		t.Fatalf("JSON parser should return map[string]interface{}, got %T", jsonResult)
	}
	if jsonMap["name"] != "John" {
		t.Errorf("JSON parser result incorrect: got %v, want John", jsonMap["name"])
	}

	// Test YAML parser
	yamlParser := model.GetParser(model.FormatYAML)
	yamlResult, err := yamlParser.Parse(yamlData)
	if err != nil {
		t.Fatalf("YAML parser failed: %v", err)
	}
	yamlMap, ok := yamlResult.(map[string]interface{})
	if !ok {
		t.Fatalf("YAML parser should return map[string]interface{}, got %T", yamlResult)
	}
	if yamlMap["name"] != "John" {
		t.Errorf("YAML parser result incorrect: got %v, want John", yamlMap["name"])
	}
}
