# gopantic

Type-safe JSON/YAML parsing with validation for Go.

Automatic format detection, type coercion, and struct tag validation in a single `ParseInto[T]()` call.

## Quick Example

```go
type User struct {
    ID    int    `json:"id" validate:"required,min=1"`
    Name  string `json:"name" validate:"required,min=2"`
    Email string `json:"email" validate:"required,email"`
}

data := []byte(`{"id": "123", "name": "Alice", "email": "alice@example.com"}`)

user, err := model.ParseInto[User](data)
// String "123" coerced to int 123, validation applied
```

## Features

| Feature | Description |
|---------|-------------|
| Format Detection | JSON or YAML detected automatically |
| Type Coercion | `"123"` to `123`, `"true"` to `true` |
| Validation | Struct tags: `validate:"required,email,min=5"` |
| Cross-Field | Compare fields: `validate:"gtfield=Min"` |
| Caching | Optional, 5x+ speedup on repeated data |

## Installation

```bash
go get github.com/1mb-dev/gopantic
```

Requires Go 1.23+.

## When to Use

Good fit:

- API request parsing with validation
- Configuration files (JSON/YAML)
- Data import with type conversion

Not ideal:

- Latency-critical hot paths
- Simple JSON without validation
- Binary protocols

## Next

- [Getting Started](getting-started.md)
- [Validation Guide](guide/validation.md)
- [API Reference](reference/api.md)
