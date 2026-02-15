# Getting Started

## Installation

```bash
go get github.com/1mb-dev/gopantic
```

## Basic Usage

Define a struct with JSON tags and validation rules:

```go
type User struct {
    ID       int    `json:"id" validate:"required,min=1"`
    Username string `json:"username" validate:"required,min=3,max=50"`
    Email    string `json:"email" validate:"required,email"`
    Age      int    `json:"age" validate:"min=0,max=150"`
}
```

Parse data:

```go
import "github.com/1mb-dev/gopantic/pkg/model"

data := []byte(`{"id": "123", "username": "alice", "email": "alice@example.com", "age": "25"}`)
user, err := model.ParseInto[User](data)
```

String `"123"` is coerced to int `123`.

## YAML

Format detected automatically:

```go
yamlData := []byte(`
id: 456
username: bob
email: bob@example.com
age: 30
`)

user, err := model.ParseInto[User](yamlData)
```

Explicit format when known:

```go
user, err := model.ParseIntoWithFormat[User](data, model.FormatJSON)
user, err := model.ParseIntoWithFormat[User](yamlData, model.FormatYAML)
```

## Validation

Built-in validators:

| Validator | Example | Description |
|-----------|---------|-------------|
| `required` | `validate:"required"` | Non-zero value |
| `min` | `validate:"min=5"` | Min value or length |
| `max` | `validate:"max=100"` | Max value or length |
| `length` | `validate:"length=10"` | Exact length |
| `email` | `validate:"email"` | Email format |
| `alpha` | `validate:"alpha"` | Letters only |
| `alphanum` | `validate:"alphanum"` | Letters and numbers |

Combine with commas:

```go
Email string `json:"email" validate:"required,email"`
Age   int    `json:"age" validate:"required,min=18,max=120"`
```

See [Validation Guide](guide/validation.md) for all validators.

## Error Handling

```go
user, err := model.ParseInto[User](data)
if err != nil {
    // "field 'email' validation failed: invalid email format"
    log.Printf("Error: %v", err)
}
```

Error messages include field values. Do not expose to untrusted clients.

## Caching

For repeated parsing of identical data:

```go
parser := model.NewCachedParser[User](nil)
defer parser.Close()

user1, _ := parser.Parse(data)  // Cache miss
user2, _ := parser.Parse(data)  // Cache hit
```

See [Caching Guide](guide/caching.md).

## Configuration

Thread-safe accessors:

```go
model.SetMaxInputSize(5 * 1024 * 1024)  // 5MB (default: 10MB)
model.SetMaxValidationDepth(16)          // default: 32
model.SetMaxCacheSize(500)               // default: 1000
```

## Next

- [Validation Guide](guide/validation.md)
- [Caching Guide](guide/caching.md)
- [API Reference](reference/api.md)
