# Migration Guide

## Quick Comparison

| Task | Standard Libraries | gopantic |
|------|-------------------|----------|
| Parse + Validate | `json.Unmarshal()` + `validate.Struct()` | `model.ParseInto[T]()` |
| Parse YAML | `yaml.Unmarshal()` + `validate.Struct()` | `model.ParseInto[T]()` |
| Type coercion | Manual conversion | Automatic |
| Validation | Separate step | Integrated |
| Custom validators | `RegisterValidation()` | `RegisterGlobalFunc()` |
| Error handling | `ValidationErrors` | Structured error types |

## Migration Examples

### From encoding/json + validator

**Before:**
```go
import (
    "encoding/json"
    "github.com/go-playground/validator/v10"
)

func parseUser(data []byte) (*User, error) {
    var user User
    if err := json.Unmarshal(data, &user); err != nil {
        return nil, err
    }
    validate := validator.New()
    if err := validate.Struct(user); err != nil {
        return nil, err
    }
    return &user, nil
}
```

**After:**
```go
import "github.com/1mb-dev/gopantic/pkg/model"

func parseUser(data []byte) (User, error) {
    return model.ParseInto[User](data)
}
```

**Key benefits:** Single call, automatic type coercion (`{"id": "42"}` works), generics return `User` directly, YAML support with same API.

### From encoding/json (manual validation)

**Before:**
```go
func loadConfig(data []byte) (*Config, error) {
    var cfg Config
    if err := json.Unmarshal(data, &cfg); err != nil {
        return nil, err
    }
    if cfg.Port < 1 || cfg.Port > 65535 {
        return nil, errors.New("invalid port")
    }
    if cfg.Host == "" {
        return nil, errors.New("host required")
    }
    return &cfg, nil
}
```

**After:**
```go
type Config struct {
    Port int    `json:"port" validate:"required,min=1,max=65535"`
    Host string `json:"host" validate:"required"`
}

func loadConfig(data []byte) (Config, error) {
    return model.ParseInto[Config](data)
}
```

**Key benefits:** Declarative validation via tags, automatic field names in errors, type coercion (`"8080"` → `8080`).

### From YAML libraries

Replace `yaml.Unmarshal()` + validator with `model.ParseInto[T]()`. Format auto-detected, validation integrated.

## Migration Checklist

### 1. Update imports

```go
- import "encoding/json"
- import "github.com/go-playground/validator/v10"
+ import "github.com/1mb-dev/gopantic/pkg/model"
```

### 2. Update validation tags

Most validator tags work as-is:

| validator | gopantic | Notes |
|-----------|----------|-------|
| `required`, `min=N`, `max=N` | Same | Compatible |
| `email` | `email` | Simplified regex |
| `len=8` | `len=8` | Same |
| `eqfield=Password` | `eqfield=Password` | Cross-field validators supported |
| `dive` | N/A | Nested validation automatic |

### 3. Handle type coercion

Remove manual string-to-type conversion. gopantic handles automatically:
```go
{"age": "25"}  // Now works with Age int field
```

### 4. Update error handling

**Before:**
```go
if validationErrs, ok := err.(validator.ValidationErrors); ok {
    for _, fieldErr := range validationErrs {
        fmt.Printf("Field: %s, Error: %s\n", fieldErr.Field(), fieldErr.Tag())
    }
}
```

**After:**
```go
// Errors already formatted: "validation error on field 'Email': must be a valid email"
log.Error(err)

// Or type-check structured errors
if parseErr, ok := err.(*model.ParseError); ok {
    fmt.Printf("Field: %s, Type: %s\n", parseErr.Field, parseErr.Type)
}
```

### 5. Update custom validators

**Before:**
```go
validate.RegisterValidation("strong_password", func(fl validator.FieldLevel) bool {
    password := fl.Field().String()
    return len(password) >= 8 && hasUpperLower(password)
})
```

**After:**
```go
model.RegisterGlobalFunc("strong_password", func(fieldName string, value interface{}, params map[string]interface{}) error {
    password, ok := value.(string)
    if !ok || len(password) < 8 || !hasUpperLower(password) {
        return model.NewValidationError(fieldName, value, "strong_password", "password must be 8+ chars with upper/lowercase")
    }
    return nil
})
```

## Common Gotchas

### Required vs zero values

```go
Port int `json:"port"`                  // Allows 0
Port int `json:"port" validate:"required"` // Rejects 0
```

### Optional fields

Use pointers for optional fields:
```go
Phone *string `json:"phone"` // nil = not provided, "" = empty string provided
```

### Nested validation

Automatic for nested structs. All `validate` tags in nested structs are checked:
```go
type User struct {
    Address Address `json:"address"` // Address fields validated automatically
}
```

### Tag order

Tags can appear in any order: `json:"name" validate:"required"` or `validate:"required" json:"name"`

## Performance Tips

### Caching for repeated parsing

```go
parser := model.NewCachedParser[User](nil)
defer parser.Close()

user1, _ := parser.Parse(data) // Cache miss
user2, _ := parser.Parse(data) // Cache hit
```

### Input size limits

```go
model.MaxInputSize = 5 * 1024 * 1024  // 5MB limit
model.MaxInputSize = 0                 // No limit
```

### Validation depth control

```go
model.MaxValidationDepth = 32  // Default, prevents stack overflow
```

## Getting Help

If you encounter migration issues:

1. Check [API documentation](reference/api.md) for function details
2. Review [examples](https://github.com/1mb-dev/gopantic/tree/main/examples) for common patterns
3. File issues at https://github.com/1mb-dev/gopantic/issues with:
   - Current code before migration
   - Error or unexpected behavior
   - Go version and gopantic version
