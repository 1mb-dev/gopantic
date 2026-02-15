# Caching Guide

Optional caching for repeated parsing of identical data. Provides 5x+ speedup on cache hits.

## When to Use Caching

**Good use cases:**

- Configuration files parsed on retries
- Message queue deduplication (same message reprocessed)
- Request retries with identical payloads
- Test fixtures parsed multiple times
- Any scenario with repeated identical input

**Not ideal for:**

- Unique API requests (every request is different)
- Streaming data
- Low-repetition scenarios (cache overhead > benefit)

## Basic Usage

```go
import "github.com/1mb-dev/gopantic/pkg/model"

// Create a cached parser with default config
parser := model.NewCachedParser[User](nil)
defer parser.Close()  // Important: stops background cleanup goroutine

// Parse data - first call is a cache miss
user1, err := parser.Parse(jsonData)

// Same data - cache hit (instant return)
user2, err := parser.Parse(jsonData)
```

## Configuration

```go
config := &model.CacheConfig{
    TTL:             time.Hour,        // How long entries stay valid
    MaxEntries:      1000,             // Maximum cached entries
    CleanupInterval: 30 * time.Minute, // Background cleanup frequency
}

parser := model.NewCachedParser[User](config)
defer parser.Close()
```

### Configuration Options

| Option | Default | Description |
|--------|---------|-------------|
| `TTL` | 1 hour | Time-to-live for cached entries |
| `MaxEntries` | 1000 | Maximum number of cached entries |
| `CleanupInterval` | 30 minutes | Background cleanup frequency (0 to disable) |

### Default Configuration

```go
config := model.DefaultCacheConfig()
// TTL: 1 hour
// MaxEntries: 1000
// CleanupInterval: 30 minutes
```

## Eviction Behavior

The cache uses **FIFO eviction** (First In, First Out):

- When `MaxEntries` is reached, the oldest entry is evicted
- Recently accessed entries are NOT prioritized (this is not LRU)
- Expired entries are removed on access or by background cleanup

## Cache Stats

Monitor cache performance:

```go
size, maxSize, hitRate := parser.Stats()
fmt.Printf("Cache: %d/%d entries, %.1f%% hit rate\n",
    size, maxSize, hitRate*100)
```

- `size`: Current number of cached entries
- `maxSize`: Maximum entries (from config)
- `hitRate`: Hits / (Hits + Misses), 0.0 to 1.0

## Cache Operations

```go
// Clear all cached entries
parser.ClearCache()

// Parse with explicit format
user, err := parser.ParseWithFormat(data, model.FormatJSON)

// Stop background cleanup (call when done)
parser.Close()
```

## Thread Safety

`CachedParser` is fully thread-safe:

- Multiple goroutines can call `Parse()` concurrently
- `ClearCache()` and `Stats()` are also safe
- Internal synchronization uses `sync.RWMutex`

```go
parser := model.NewCachedParser[User](nil)
defer parser.Close()

var wg sync.WaitGroup
for i := 0; i < 100; i++ {
    wg.Add(1)
    go func() {
        defer wg.Done()
        _, _ = parser.Parse(data)
    }()
}
wg.Wait()
```

## Performance Characteristics

| Operation | First Call | Cached Call |
|-----------|------------|-------------|
| Simple JSON | ~8-10 µs | ~1-2 µs |
| Complex JSON | ~25-30 µs | ~2-3 µs |
| With validation | +2-5 µs | (same as cached) |

**Note:** Actual performance depends on struct complexity, validation rules, and input size. Run your own benchmarks for accurate numbers.

## Cache Key Generation

Cache keys are generated from:

1. **Content hash**: FNV-1a for small inputs (<1KB), SHA256 for larger
2. **Type name**: Different types don't share cache entries
3. **Format**: Same data parsed as JSON vs YAML has different keys

```go
// These create separate cache entries:
parserA := model.NewCachedParser[TypeA](nil)
parserB := model.NewCachedParser[TypeB](nil)

parserA.Parse(data)  // Key: hash:TypeA:json
parserB.Parse(data)  // Key: hash:TypeB:json
```

## Best Practices

1. **Always call Close()**: Use `defer parser.Close()` to stop cleanup goroutine
2. **Reuse parsers**: Create once, use many times (parsers are thread-safe)
3. **Size appropriately**: Set `MaxEntries` based on expected unique inputs
4. **Monitor hit rate**: Low hit rates indicate caching isn't helping
5. **Consider TTL**: Longer TTL = more hits, but potentially stale data

## Example: API Handler

```go
var userParser = model.NewCachedParser[User](nil)

func HandleCreateUser(w http.ResponseWriter, r *http.Request) {
    body, _ := io.ReadAll(r.Body)

    user, err := userParser.Parse(body)
    if err != nil {
        http.Error(w, "Invalid user data", 400)
        return
    }

    // Use user...
}
```

## Example: Config Loading with Retries

```go
func LoadConfig(path string) (*Config, error) {
    parser := model.NewCachedParser[Config](&model.CacheConfig{
        TTL:        5 * time.Minute,
        MaxEntries: 10,
    })
    defer parser.Close()

    var lastErr error
    for i := 0; i < 3; i++ {
        data, err := os.ReadFile(path)
        if err != nil {
            lastErr = err
            time.Sleep(time.Second)
            continue
        }

        // On retry with same file content, this is a cache hit
        return parser.Parse(data)
    }

    return nil, fmt.Errorf("failed after 3 retries: %w", lastErr)
}
```
