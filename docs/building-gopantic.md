# Building gopantic

Why another parsing library? What problems does it solve?

## The Problem

Go's `encoding/json` deserializes. That's it. Everything else is your problem:

```go
// What you actually do for every API endpoint
var raw map[string]interface{}
json.Unmarshal(data, &raw)

id, ok := raw["id"].(float64)  // JSON numbers are float64
if !ok {
    return errors.New("invalid id")
}
user := User{ID: int(id)}  // Manual conversion

validate := validator.New()
if err := validate.Struct(user); err != nil {
    return err
}
```

This pattern repeats everywhere: parse, assert types, convert, validate. Four steps, three libraries, scattered error handling.

## The Solution

One function that answers: "Is this data valid and usable?"

```go
user, err := model.ParseInto[User](data)
```

Parse. Coerce types. Validate. Return all errors. Done.

## Design Decisions

### Generics for Type Safety

Returns `User`, not `interface{}`. Compile-time checking. IDE autocompletion works.

**Trade-off**: Requires Go 1.18+. Worth it.

### Automatic Type Coercion

APIs send `"123"` instead of `123`. Mobile apps send everything as strings. Configuration files mix types.

gopantic handles this transparently:

```go
// Input: {"id": "42", "active": "true"}
// Result: User{ID: 42, Active: true}
```

**When to disable**: Financial calculations requiring exact decimal types. Strict contracts where coercion masks errors.

### Tag-Based Validation

Validation rules live with type definitions:

```go
type Product struct {
    SKU   string  `json:"sku" validate:"required,len=8"`
    Price float64 `json:"price" validate:"min=0.01"`
}
```

No separate validation layer. No drift between struct and rules.

### Error Aggregation

Return all errors, not just the first:

```
multiple errors:
  field 'SKU': length must be exactly 8;
  field 'Price': must be at least 0.01
```

Users fix everything in one iteration.

### Optional Caching

For repeated parsing of identical data (config files, retries, deduplication):

```go
parser := model.NewCachedParser[Config](nil)
defer parser.Close()

config, _ := parser.Parse(data)  // Cache miss
config, _ := parser.Parse(data)  // Cache hit, 5-10x faster
```

FIFO eviction. TTL expiration. Thread-safe.

## Architecture

```
Raw Bytes → Format Detection → Parse → Coerce → Map to Struct → Validate → Result
```

**Format Detection**: JSON markers (`{`, `[`) or YAML markers (`---`, `:`). O(1).

**Coercion**: String to int/float/bool/time. Fails fast on invalid conversions.

**Validation**: Cached by `reflect.Type`. First parse pays reflection cost; subsequent parses reuse.

## Performance

| Scenario | vs stdlib | Notes |
|----------|-----------|-------|
| Parse only | 4-5x slower | Includes coercion |
| Parse + validate | 2.4x slower | Apples to apples |
| Cached parse | 5-10x faster | For identical inputs |

For most backend services, the convenience justifies the overhead. For ultra-high-frequency paths, profile first.

## When to Use

**Good fit**: API validation, config parsing, webhook handlers, data pipelines.

**Poor fit**: Ultra-low-latency trading, embedded systems, Protocol Buffers, streaming large datasets.

## Lessons Learned

1. **Cache reflection metadata** - Parsing tags on every request killed performance. Caching by type reduced overhead from 15% to 5%.

2. **Aggregate errors** - Early versions stopped at first error. Users hated the iteration loops.

3. **Generics over codegen** - Code generation (`easyjson`) is faster but complicates workflows. Generics provide 90% of the benefit with zero build overhead.

4. **Document cache effectiveness** - Content-based keys work for config files, not for unique API requests. Set proper expectations.

## Examples

See [examples/](https://github.com/1mb-dev/gopantic/tree/main/examples) for runnable code:

- **quickstart/** - Basic parsing and validation
- **api_validation/** - HTTP handler patterns
- **cross_field_validation/** - Password confirmation, field comparisons
- **cache_demo/** - Caching for repeated parsing
- **yaml/** - Configuration file parsing
- **pointers/** - Optional fields with nil handling
- **postgresql_jsonb/** - PostgreSQL JSONB with json.RawMessage

## Next

- [Getting Started](getting-started.md) - Installation and basic usage
- [API Reference](reference/api.md) - Complete API documentation
- [Validation Guide](guide/validation.md) - All validators and custom rules
