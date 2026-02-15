package main

import (
	"fmt"

	"github.com/1mb-dev/gopantic/pkg/model"
)

// DatabaseConfig demonstrates nested YAML configuration
type DatabaseConfig struct {
	Host     string `yaml:"host" validate:"required"`
	Port     int    `yaml:"port" validate:"min=1,max=65535"`
	Username string `yaml:"username" validate:"required"`
	Password string `yaml:"password" validate:"required"`
	Database string `yaml:"database" validate:"required"`
	SSL      bool   `yaml:"ssl"`
}

// ServerConfig demonstrates configuration with arrays
type ServerConfig struct {
	Port    int      `yaml:"port" validate:"min=1000,max=65535"`
	Workers int      `yaml:"workers" validate:"min=1,max=100"`
	Hosts   []string `yaml:"hosts"`
	Debug   bool     `yaml:"debug"`
}

// AppConfig demonstrates complete application configuration
type AppConfig struct {
	Name     string         `yaml:"name" validate:"required"`
	Version  string         `yaml:"version" validate:"required"`
	Database DatabaseConfig `yaml:"database" validate:"required"`
	Server   ServerConfig   `yaml:"server" validate:"required"`
}

func main() {
	fmt.Println("gopantic - YAML Configuration Examples")
	fmt.Println("========================================")
	fmt.Println()

	// Example 1: Basic YAML configuration
	fmt.Println("1. Basic YAML Configuration:")
	yamlConfig := []byte(`
name: MyApplication
version: "1.0.0"

database:
  host: localhost
  port: 5432
  username: admin
  password: secret123
  database: myapp
  ssl: true

server:
  port: 8080
  workers: 10
  hosts:
    - api.example.com
    - cdn.example.com
  debug: false
`)

	config, err := model.ParseInto[AppConfig](yamlConfig)
	if err != nil {
		fmt.Printf("   Error: %v\n", err)
	} else {
		fmt.Printf("   App: %s v%s\n", config.Name, config.Version)
		fmt.Printf("   Database: %s@%s:%d\n", config.Database.Username, config.Database.Host, config.Database.Port)
		fmt.Printf("   Server: %d workers on port %d\n", config.Server.Workers, config.Server.Port)
	}

	// Example 2: Auto-detection (YAML vs JSON)
	fmt.Println("\n2. Format Auto-Detection:")

	// Same structure, different format
	jsonConfig := []byte(`{
		"name": "MyApp",
		"version": "2.0.0",
		"database": {
			"host": "db.example.com",
			"port": 5432,
			"username": "user",
			"password": "pass1234",
			"database": "prod",
			"ssl": true
		},
		"server": {
			"port": 9000,
			"workers": 20,
			"hosts": ["prod.example.com"],
			"debug": false
		}
	}`)

	fmt.Printf("   YAML format detected: %v\n", model.DetectFormat(yamlConfig) == model.FormatYAML)
	fmt.Printf("   JSON format detected: %v\n", model.DetectFormat(jsonConfig) == model.FormatJSON)

	configFromJSON, err := model.ParseInto[AppConfig](jsonConfig)
	if err != nil {
		fmt.Printf("   Error: %v\n", err)
	} else {
		fmt.Printf("   Parsed JSON: %s v%s\n", configFromJSON.Name, configFromJSON.Version)
	}

	// Example 3: Type coercion with YAML
	fmt.Println("\n3. Type Coercion in YAML:")
	coercionYAML := []byte(`
name: TestApp
version: "1.5.0"

database:
  host: testdb
  port: "5432"
  username: test
  password: testpass
  database: test
  ssl: "true"

server:
  port: "3000"
  workers: "5"
  hosts: []
  debug: 1
`)

	coercedConfig, err := model.ParseInto[AppConfig](coercionYAML)
	if err != nil {
		fmt.Printf("   Error: %v\n", err)
	} else {
		fmt.Printf("   String '5432' → int %d\n", coercedConfig.Database.Port)
		fmt.Printf("   String '3000' → int %d\n", coercedConfig.Server.Port)
		fmt.Printf("   Number 1 → bool %t\n", coercedConfig.Server.Debug)
	}

	// Example 4: Validation errors
	fmt.Println("\n4. Configuration Validation:")
	invalidYAML := []byte(`
name: ""
version: "1.0.0"

database:
  host: ""
  port: 99999
  username: ""
  password: ""
  database: ""
  ssl: false

server:
  port: 500
  workers: 0
  hosts: []
  debug: false
`)

	_, err = model.ParseInto[AppConfig](invalidYAML)
	if err != nil {
		fmt.Printf("   Multiple validation errors:\n   %v\n", err)
	}

	fmt.Println("\nYAML examples completed!")
	fmt.Println("gopantic handles both YAML and JSON with the same API")
}
