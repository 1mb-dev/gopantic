# Benchmarks

Measured on 2026-11-01 at commit `86e26f1`, with Go 1.27.3 on an Intel Core i5-8257U at 1.40 GHz under macOS. Each figure is the median of three runs, and runs of the same benchmark differed by up to 20%. Treat differences smaller than that as ties.

The small struct has four fields and nine rules. The complex one adds a nested struct and a list of addresses, for 18 rules.

## Correctly typed JSON, parsed and validated

| | gopantic | `encoding/json` + validator |
|---|---|---|
| Small struct | 2,275 ns, 10 allocs | 2,596 ns, 10 allocs |
| Complex struct | 6,701 ns, 29 allocs | 5,224 ns, 19 allocs |

These are close enough that speed should not decide between them. gopantic allocates more memory on both: 453 bytes against 186 on the small struct, and 1,526 against 520 on the complex one.

## What the extras cost

These use the small struct, against 2,275 ns for correctly typed JSON.

| Input | Time | Cost |
|---|---|---|
| Numbers sent as strings | 6,958 ns | 3.1x |
| YAML | 18,639 ns | 8.4x (against 2,230 ns for JSON through the same call) |

Coercion is the expensive path because the whole input is decoded a second time by the walker. One string in a number field costs the same as several. YAML always takes that path.

## Validation alone

`Validate` on a struct already in memory, against validator's `Struct`.

| | gopantic | validator |
|---|---|---|
| Small struct | 1,218 ns | 1,615 ns |
| Complex struct | 2,941 ns | 2,207 ns |

## Caching

For the same input, a parse took 2,982 ns and a cache hit 692 ns, about 4x less. A hit on this struct allocates once. A struct holding slices or maps pays to copy them on every hit, so its gain is smaller. See the [caching guide](guide/caching.md).

## Hostile input

From `tests/adversarial_bench_test.go`. Each input needs coercion, so each goes through the walker.

| Input | Time |
|---|---|
| 20,000 array items | 40 ms |
| 50,000 unknown keys | 24 ms |
| Nesting 30 levels deep | 69 µs |

## Running them

```bash
go test -run='^$' -bench=. -benchmem ./tests/
cd benchmarks && go test -run='^$' -bench=. -benchmem ./...
```

The comparisons against validator live in `benchmarks/`, a separate module, so that validator stays out of your dependency graph.
