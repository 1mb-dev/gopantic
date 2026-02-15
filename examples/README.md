# gopantic Examples

Runnable examples demonstrating gopantic features.

## Examples

### quickstart/
Basic parsing and validation. Start here.

### api_validation/
HTTP request validation patterns for production use.

### cross_field_validation/
Password confirmation, field comparisons, custom validators.

### cache_demo/
CachedParser for repeated parsing of identical inputs.

### yaml/
Configuration file parsing with automatic format detection.

### pointers/
Optional fields using pointers to distinguish missing from zero.

### postgresql_jsonb/
PostgreSQL JSONB integration with `json.RawMessage` for flexible metadata.

## Running

```bash
cd examples/quickstart
go run main.go
```

## Documentation

- [Getting Started](https://1mb-dev.github.io/gopantic/getting-started/)
- [API Reference](https://1mb-dev.github.io/gopantic/reference/api/)
- [Type Reference](https://1mb-dev.github.io/gopantic/reference/types/)
