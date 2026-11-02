# Benchmarks

Measured on 2026-11-02 at commit `194ec36`, with Go 1.27.3 on an Intel Core i5-8257U at 1.40 GHz under macOS. Each figure is the median of three runs, and runs of the same benchmark differed by up to 7%. Treat differences smaller than that as ties.

The small struct has four fields and nine rules. The complex one adds a nested struct and a list of addresses, for 18 rules.

## Correctly typed JSON, parsed and validated

| | gopantic | `encoding/json` + validator |
|---|---|---|
| Small struct | 2,217 ns, 10 allocs | 2,501 ns, 10 allocs |
| Complex struct | 6,365 ns, 29 allocs | 5,006 ns, 19 allocs |

These are close enough that speed should not decide between them. gopantic allocates more memory on both: 453 bytes against 186 on the small struct, and 1,531 against 520 on the complex one.

## What the extras cost

These use the small struct, against 2,217 ns for correctly typed JSON.

| Input | Time | Cost |
|---|---|---|
| Numbers sent as strings | 7,099 ns | 3.2x |
| YAML | 17,381 ns | 7.8x (against 2,215 ns for JSON through the same call) |

Coercion is the expensive path because the whole input is decoded a second time by the walker. One string in a number field costs the same as several. YAML always takes that path.

## Validation alone

`Validate` on a struct already in memory, against validator's `Struct`.

| | gopantic | validator |
|---|---|---|
| Small struct | 1,263 ns | 1,575 ns |
| Complex struct | 2,891 ns | 2,152 ns |

## Caching

For the same input, a parse took 2,918 ns and a cache hit 686 ns, about 4x less. A hit on this struct allocates once. A struct holding slices or maps pays to copy them on every hit, so its gain is smaller. See the [caching guide](guide/caching.md).

## Hostile input

From `tests/adversarial_bench_test.go`. Each input needs coercion, so each goes through the walker.

| Input | Time |
|---|---|
| 20,000 array items | 41 ms |
| 50,000 unknown keys | 25 ms |
| Nesting 30 levels deep | 72 µs |

## Running them

```bash
go test -run='^$' -bench=. -benchmem ./tests/
cd benchmarks && go test -run='^$' -bench=. -benchmem ./...
```

The comparisons against validator live in `benchmarks/`, a separate module, so that validator stays out of your dependency graph.
