# gopantic

**Practical JSON/YAML parsing with validation for Go.**

Inspired by Python's Pydantic, gopantic provides type-safe parsing, coercion, and validation with idiomatic Go APIs.

[![Go Reference](https://pkg.go.dev/badge/github.com/1mb-dev/gopantic.svg)](https://pkg.go.dev/github.com/1mb-dev/gopantic)
[![Go Version](https://img.shields.io/badge/go-1.23+-blue.svg)](https://golang.org/dl/)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![CI](https://github.com/1mb-dev/gopantic/actions/workflows/ci.yml/badge.svg)](https://github.com/1mb-dev/gopantic/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/1mb-dev/gopantic/branch/main/graph/badge.svg)](https://codecov.io/gh/1mb-dev/gopantic)

> **Full Documentation**: [1mb-dev.github.io/gopantic](https://1mb-dev.github.io/gopantic/)

## Quick Start

```bash
go get github.com/1mb-dev/gopantic
```

```go
package main

import (
    "fmt"
    "log"
    "github.com/1mb-dev/gopantic/pkg/model"
)

type User struct {
    ID    int    `json:"id" validate:"required,min=1"`
    Name  string `json:"name" validate:"required,min=2"`
    Email string `json:"email" validate:"required,email"`
    Age   int    `json:"age" validate:"min=18,max=120"`
}

func main() {
    raw := []byte(`{"id": "42", "name": "Alice", "email": "alice@example.com", "age": "28"}`)
    
    user, err := model.ParseInto[User](raw)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("%+v\n", user) // {ID:42 Name:Alice Email:alice@example.com Age:28}
}
```

## Features

- **JSON/YAML parsing** with automatic format detection
- **Type coercion** (`"123"` → `123`, `"true"` → `true`)
- **Validation** using struct tags (`validate:"required,email,min=5"`)
- **Standalone validation** - use `Validate()` independently of parsing
- **Cross-field validation** (password confirmation, field comparisons)
- **Built-in validators**: `required`, `min`, `max`, `email`, `alpha`, `alphanum`, `length`
- **Nested structs** and arrays with full validation
- **Time parsing** (RFC3339, Unix timestamps, custom formats)
- **Pointer support** for optional fields (`*string`, `*int`)
- **json.RawMessage support** for flexible/deferred JSON parsing
- **Caching support** for repeated identical inputs (optional)
- **Thread-safe** concurrent parsing
- **Zero dependencies** (except optional YAML support)
- **Generic API** for type-safe parsing with compile-time guarantees

## YAML Support

Works seamlessly with YAML - same API, automatic detection:

```go
yamlData := []byte(`
id: 42
name: Alice
email: alice@example.com
age: 28
`)

user, err := model.ParseInto[User](yamlData) // Automatic YAML detection
```

## json.RawMessage Support

gopantic seamlessly handles `json.RawMessage` fields for flexible metadata and JSONB database columns:

```go
type Account struct {
    ID          string          `json:"id" validate:"required"`
    Name        string          `json:"name" validate:"required,min=2"`
    MetadataRaw json.RawMessage `json:"metadata,omitempty"`
}

input := []byte(`{
    "id": "acc_123",
    "name": "John Doe",
    "metadata": {"preferences": {"theme": "dark"}, "tags": ["vip"]}
}`)

account, err := model.ParseInto[Account](input)
// MetadataRaw preserves the raw JSON for later parsing
```

## Standalone Validation

Use `Validate()` to validate structs from any source (database, environment, etc.):

```go
type Config struct {
    Host string `json:"host" validate:"required"`
    Port int    `json:"port" validate:"min=1,max=65535"`
}

// Populate from environment or database
config := Config{
    Host: os.Getenv("HOST"),
    Port: 8080,
}

// Validate independently
if err := model.Validate(&config); err != nil {
    log.Fatal(err)
}
```

This enables flexible patterns:

```go
// Pattern 1: Standard library unmarshal + gopantic validation
var req Request
json.Unmarshal(body, &req)  // Handles json.RawMessage correctly
model.Validate(&req)         // Apply gopantic validation rules

// Pattern 2: All-in-one (parsing + validation)
req, err := model.ParseInto[Request](body)
```

## Validation

```go
type Product struct {
    SKU   string  `json:"sku" validate:"required,length=8"`
    Price float64 `json:"price" validate:"required,min=0.01"`
}

// Multiple validation errors are aggregated
product, err := model.ParseInto[Product](invalidData)
// Error: "multiple errors: validation error on field 'SKU': ...; validation error on field 'Price': ..."
```

### Cross-Field Validation

Built-in support for cross-field validation - compare fields against each other:

```go
type UserRegistration struct {
    Password        string `json:"password" validate:"required,min=8"`
    ConfirmPassword string `json:"confirm_password" validate:"required,password_match"`
    Email           string `json:"email" validate:"required,email"`
    NotificationEmail string `json:"notification_email" validate:"email,email_different"`
}

// Register custom cross-field validators
model.RegisterGlobalCrossFieldFunc("password_match", func(fieldName string, fieldValue interface{}, structValue reflect.Value, params map[string]interface{}) error {
    confirmPassword := fieldValue.(string)
    password := structValue.FieldByName("Password").String()
    if confirmPassword != password {
        return model.NewValidationError(fieldName, fieldValue, "password_match", "passwords do not match")
    }
    return nil
})
```

## Performance

**Optimized for production use** - recent improvements deliver 40-65% faster parsing with 64% less memory:

| Metric | Standard Parsing | Cached Parsing |
|--------|-----------------|----------------|
| Speed vs stdlib JSON | 1.7x slower | **5.4x faster** |
| Memory | 435 B/op | 112 B/op |
| Allocations | 14/op | 6/op |

Use **cached parser** for repeated identical inputs (configs, retries):

```go
parser := model.NewCachedParser[User](nil)
user, _ := parser.Parse(data) // 5.4x faster for cache hits
```

**Note**: YAML is 4x slower than JSON due to parser complexity. Use JSON for performance-critical paths.

## Why Choose gopantic?

### vs. Standard Library (`encoding/json`)
- **Built-in validation** - No separate validation step needed
- **Type coercion** - Handles `"123"` → `123` automatically
- **Better errors** - Structured error reporting with field paths
- **YAML support** - Automatic format detection
- **Cross-field validation** - Compare fields against each other

### vs. Validation Libraries (`go-playground/validator`)
- **Integrated parsing** - Parse and validate in one step
- **Type coercion** - No manual string conversion needed
- **Format agnostic** - Works with JSON and YAML seamlessly
- **Generics support** - Type-safe with `ParseInto[T]()`
- **Performance** - Built-in caching for repeated operations

### vs. Code Generation (`easyjson`, `ffjson`)
- **Zero code generation** - No build step or generated files
- **Dynamic validation** - Runtime validation rule changes
- **Simpler workflow** - Standard Go development process
- **Faster iteration** - No regeneration on struct changes
- **Cross-field validation** - Complex validation logic support

### vs. Schema Libraries (`jsonschema`, `gojsonschema`)
- **Native Go structs** - Use existing struct definitions
- **Compile-time safety** - Type checking at compile time
- **Better performance** - Direct struct mapping vs. schema validation
- **IDE support** - Full autocompletion and refactoring
- **Integrated coercion** - Automatic type conversion

## Development

### Pre-commit Hook

A practical pre-commit hook is included that automatically:
- Checks for potential secrets in staged files
- Formats Go code with `go fmt`
- Runs `go vet` for static analysis
- Performs fast linting with `golangci-lint`
- Runs tests if test files are modified

Install the hook:
```bash
make hooks
```

## Documentation

**Full documentation**: [1mb-dev.github.io/gopantic](https://1mb-dev.github.io/gopantic/)

Quick links:
- [Getting Started](https://1mb-dev.github.io/gopantic/getting-started/) - Installation and basic usage
- [API Reference](https://1mb-dev.github.io/gopantic/reference/api/) - Complete API documentation
- [Validation Guide](https://1mb-dev.github.io/gopantic/guide/validation/) - All validation options
- [Migration Guide](https://1mb-dev.github.io/gopantic/migration/) - Coming from other libraries
- [Examples](examples/) - Runnable code examples

**Resources**:
- [Changelog](https://1mb-dev.github.io/gopantic/changelog/) - Version history
- [Security Policy](.github/SECURITY.md) - Vulnerability reporting

## License

MIT License - see [LICENSE](LICENSE) for details.