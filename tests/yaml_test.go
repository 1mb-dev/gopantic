package tests

import (
	"reflect"
	"testing"
	"time"

	"github.com/1mb-dev/gopantic/pkg/model"
)

// Test structs for YAML parsing
type YAMLUser struct {
	ID    int    `yaml:"id" validate:"required,min=1"`
	Name  string `yaml:"name" validate:"required,min=2"`
	Email string `yaml:"email" validate:"required,email"`
	Age   int    `yaml:"age" validate:"min=18,max=120"`
}

type YAMLConfig struct {
	Database struct {
		Host     string `yaml:"host" validate:"required"`
		Port     int    `yaml:"port" validate:"min=1,max=65535"`
		Username string `yaml:"username" validate:"required"`
		Password string `yaml:"password" validate:"required"`
		SSL      bool   `yaml:"ssl"`
	} `yaml:"database" validate:"required"`
	Server struct {
		Port    int      `yaml:"port" validate:"min=1000,max=65535"`
		Workers int      `yaml:"workers" validate:"min=1,max=100"`
		Hosts   []string `yaml:"hosts"`
	} `yaml:"server" validate:"required"`
	Debug bool `yaml:"debug"`
}

func TestParseIntoWithFormat_YAML_Basic(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		want    YAMLUser
		wantErr bool
	}{
		{
			name: "valid YAML user",
			input: []byte(`
id: 123
name: "John Doe"
email: "john@example.com"
age: 30
`),
			want: YAMLUser{
				ID:    123,
				Name:  "John Doe",
				Email: "john@example.com",
				Age:   30,
			},
			wantErr: false,
		},
		{
			name: "YAML with type coercion",
			input: []byte(`
id: "456"
name: Alice Smith
email: alice@test.com
age: "25"
`),
			want: YAMLUser{
				ID:    456,
				Name:  "Alice Smith",
				Email: "alice@test.com",
				Age:   25,
			},
			wantErr: false,
		},
		{
			name: "invalid YAML - missing required field",
			input: []byte(`
id: 789
name: Bob
# missing email
age: 35
`),
			wantErr: true,
		},
		{
			name: "invalid YAML - validation failure",
			input: []byte(`
id: 0
name: "A"
email: "invalid-email"
age: 15
`),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := model.ParseIntoWithFormat[YAMLUser](tt.input, model.FormatYAML)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseIntoWithFormat() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseIntoWithFormat() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseInto_YAML_AutoDetection(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		want    YAMLUser
		wantErr bool
	}{
		{
			name: "auto-detect YAML format",
			input: []byte(`
id: 123
name: John Doe
email: john@example.com
age: 30
`),
			want: YAMLUser{
				ID:    123,
				Name:  "John Doe",
				Email: "john@example.com",
				Age:   30,
			},
			wantErr: false,
		},
		{
			name: "YAML with document separator",
			input: []byte(`---
id: 456
name: Alice
email: alice@test.com
age: 28
`),
			want: YAMLUser{
				ID:    456,
				Name:  "Alice",
				Email: "alice@test.com",
				Age:   28,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test auto-detection
			got, err := model.ParseInto[YAMLUser](tt.input)
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

func TestParseInto_YAML_ComplexConfiguration(t *testing.T) {
	yamlConfig := []byte(`
database:
  host: localhost
  port: 5432
  username: admin
  password: secret123
  ssl: true

server:
  port: 8080
  workers: 10
  hosts:
    - api.example.com
    - cdn.example.com
    - static.example.com

debug: false
`)

	expected := YAMLConfig{
		Database: struct {
			Host     string `yaml:"host" validate:"required"`
			Port     int    `yaml:"port" validate:"min=1,max=65535"`
			Username string `yaml:"username" validate:"required"`
			Password string `yaml:"password" validate:"required"`
			SSL      bool   `yaml:"ssl"`
		}{
			Host:     "localhost",
			Port:     5432,
			Username: "admin",
			Password: "secret123",
			SSL:      true,
		},
		Server: struct {
			Port    int      `yaml:"port" validate:"min=1000,max=65535"`
			Workers int      `yaml:"workers" validate:"min=1,max=100"`
			Hosts   []string `yaml:"hosts"`
		}{
			Port:    8080,
			Workers: 10,
			Hosts:   []string{"api.example.com", "cdn.example.com", "static.example.com"},
		},
		Debug: false,
	}

	result, err := model.ParseInto[YAMLConfig](yamlConfig)
	if err != nil {
		t.Fatalf("ParseInto() error = %v", err)
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("ParseInto() = %v, want %v", result, expected)
	}
}

func TestParseInto_YAML_WithTimeFields(t *testing.T) {
	type EventYAML struct {
		Name      string    `yaml:"name" validate:"required"`
		StartTime time.Time `yaml:"start_time"`
		EndTime   time.Time `yaml:"end_time"`
		Duration  int       `yaml:"duration" validate:"min=1"`
	}

	yamlData := []byte(`
name: "Tech Conference"
start_time: "2023-12-25T10:30:00Z"
end_time: 1703523000
duration: "300"
`)

	result, err := model.ParseInto[EventYAML](yamlData)
	if err != nil {
		t.Fatalf("ParseInto() error = %v", err)
	}

	if result.Name != "Tech Conference" {
		t.Errorf("Name = %v, want %v", result.Name, "Tech Conference")
	}

	expectedStart := time.Date(2023, 12, 25, 10, 30, 0, 0, time.UTC)
	if !result.StartTime.Equal(expectedStart) {
		t.Errorf("StartTime = %v, want %v", result.StartTime, expectedStart)
	}

	if result.Duration != 300 {
		t.Errorf("Duration = %v, want %v", result.Duration, 300)
	}
}
