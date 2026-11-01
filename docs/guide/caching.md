# Caching

`CachedParser` remembers the result for each exact input. It helps when the same bytes arrive again: a config file read on every reload, a retried request, a redelivered queue message. If each input is different, it costs a hash per call and returns nothing.

```go
parser := model.NewCachedParser[Config](nil)
defer parser.Close()

config, err := parser.Parse(data) // parses
config, err = parser.Parse(data)  // same bytes: returned from the cache
```

A parser holds one type and is safe for concurrent use, so create one and share it. Only successful parses are cached. `Close` stops the background cleanup and can be called more than once.

## Results are copies

Each caller gets its own copy, so changing a returned slice or map can't affect the cache or anyone else. That copy is the price of a hit: a struct of plain values costs nothing extra, while one holding slices and maps pays to copy them each time.

A type holding references gopantic can't copy, such as an unexported slice, a `*big.Int` or a `netip.Addr`, is parsed on every call, and `Stats` shows a hit rate of zero. So is a value nested past 1,024 levels of copying.

## Configuration

```go
parser := model.NewCachedParser[Config](&model.CacheConfig{
    TTL:             5 * time.Minute,
    MaxEntries:      100,
    CleanupInterval: time.Minute,
})
```

| Field | Default | Meaning |
|-------|---------|---------|
| `TTL` | 1 hour | How long an entry lives |
| `MaxEntries` | 1000 | Entries kept; the oldest goes first |
| `CleanupInterval` | 30 minutes | How often expired entries are swept; 0 turns the sweep off |

`MaxEntries` counts entries, not bytes. A cache of large documents holds that many large documents. An entry is evicted by age, whether or not it's still being hit.

## Checking it helps

```go
size, maxSize, hitRate := parser.Stats()
```

A low hit rate means the inputs aren't repeating, and a plain `ParseInto` will serve you better. `ClearCache` empties the cache.

## Keys and staleness

The key is a SHA-256 of the input plus its format, so one changed byte is a different entry. A hit doesn't run validation again. If you replace a validator under an existing name, entries already cached keep the old verdict until they expire or you call `ClearCache`.

The [benchmarks](../benchmarks.md) show what a hit saves.
