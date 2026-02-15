# Benchmark Comparison

Performance comparison between gopantic and alternatives.

## Methodology

**Hardware:** Intel Core i5-8257U CPU @ 1.40GHz (8 cores)
**Go Version:** 1.24.0
**OS:** macOS (Darwin)
**Date:** 2026-01-19

All benchmarks run with:
```bash
go test -bench=. -benchmem ./tests/... -run=^$
```

## Results Summary

### Validation-Only (struct already parsed)

| Library | Simple Struct | Complex Struct |
|---------|--------------|----------------|
| **gopantic** | 1311 ns/op, 81 B/op, 5 allocs | 2450 ns/op, 203 B/op, 11 allocs |
| **go-playground/validator** | 1552 ns/op, 88 B/op, 5 allocs | 1740 ns/op, 89 B/op, 5 allocs |

**Takeaway:** gopantic is ~15% faster for simple validation but go-playground/validator is faster for complex nested structs due to optimized struct traversal.

### Parse + Validate (JSON to validated struct)

| Approach | Simple Struct | Complex Struct |
|----------|--------------|----------------|
| **gopantic ParseInto** | 4264 ns/op, 988 B/op, 28 allocs | 12720 ns/op, 2642 B/op, 85 allocs |
| **encoding/json + validator** | 2928 ns/op, 382 B/op, 12 allocs | 7796 ns/op, 772 B/op, 26 allocs |

**Takeaway:** Standard library combination is faster for pure JSON-to-struct. gopantic's strength is in type coercion and multi-format support.

### Parsing Only (no validation)

| Library | Simple Struct |
|---------|--------------|
| **gopantic** | 2853 ns/op, 896 B/op, 23 allocs |
| **encoding/json** | 1340 ns/op, 288 B/op, 7 allocs |

**Takeaway:** encoding/json is ~2x faster for pure JSON parsing. gopantic adds overhead for format detection and coercion infrastructure.

### Cached Parsing (gopantic unique feature)

| Mode | Performance |
|------|-------------|
| **Cached** | 645.6 ns/op, 112 B/op, 6 allocs |
| **Uncached** | 6068 ns/op, 1118 B/op, 35 allocs |

**Takeaway:** CachedParser provides ~10x performance improvement for repeated parsing of the same data.

### Type Coercion (gopantic unique feature)

```go
// Input: {"id": "123", "name": "John", "age": "30"}
// Output: struct with int id=123, int age=30
```

| Library | Performance |
|---------|-------------|
| **gopantic** | 6491 ns/op, 1882 B/op, 50 allocs |
| **encoding/json** | ❌ Fails (type mismatch error) |

**Takeaway:** gopantic automatically coerces string numbers to integers. Standard library requires exact type matching.

### JSON vs YAML Parsing

| Format | Performance |
|--------|-------------|
| **JSON** | 4263 ns/op, 989 B/op, 28 allocs |
| **YAML** | 24194 ns/op, 17885 B/op, 161 allocs |

**Takeaway:** JSON is ~6x faster than YAML. Use JSON when performance matters.

### Format Detection

| Operation | Performance |
|-----------|-------------|
| **DetectFormat** | 2.588 ns/op, 0 B/op, 0 allocs |

**Takeaway:** Format detection is essentially free (sub-3ns).

### Parallel Performance

| Approach | Performance |
|----------|-------------|
| **gopantic** | 1986 ns/op, 1002 B/op, 28 allocs |
| **encoding/json + validator** | 1634 ns/op, 391 B/op, 12 allocs |

**Takeaway:** Both scale well under concurrent load. Standard library combination has lower overhead.

## When to Use gopantic

**Choose gopantic when:**
- You need automatic type coercion (string "123" → int 123)
- You work with both JSON and YAML formats
- You want integrated parse + validate in one step
- You use the CachedParser for repeated parsing
- You're building configuration parsers, API gateways, or data pipelines

**Choose encoding/json + validator when:**
- Pure performance is critical
- You only use JSON format
- Your data always has exact type matching
- You're processing high-volume, well-typed streams

## Detailed Results

```
BenchmarkValidation_Gopantic_Simple-8             	  921296	      1311 ns/op	      81 B/op	       5 allocs/op
BenchmarkValidation_Playground_Simple-8           	  799057	      1552 ns/op	      88 B/op	       5 allocs/op
BenchmarkValidation_Gopantic_Complex-8            	  457716	      2450 ns/op	     203 B/op	      11 allocs/op
BenchmarkValidation_Playground_Complex-8          	  686307	      1740 ns/op	      89 B/op	       5 allocs/op

BenchmarkParseValidate_Gopantic_Simple-8          	  279391	      4264 ns/op	     988 B/op	      28 allocs/op
BenchmarkParseValidate_StdJSON_Simple-8           	  404677	      2928 ns/op	     382 B/op	      12 allocs/op
BenchmarkParseValidate_Gopantic_Complex-8         	   95326	     12720 ns/op	    2642 B/op	      85 allocs/op
BenchmarkParseValidate_StdJSON_Complex-8          	  178778	      7796 ns/op	     772 B/op	      26 allocs/op

BenchmarkParse_Gopantic_NoValidation-8            	  414764	      2853 ns/op	     896 B/op	      23 allocs/op
BenchmarkParse_StdJSON-8                          	  891786	      1340 ns/op	     288 B/op	       7 allocs/op

BenchmarkCachedParsing-8                          	 1799991	       645.6 ns/op	     112 B/op	       6 allocs/op
BenchmarkCoercion_Gopantic-8                      	  181592	      6491 ns/op	    1882 B/op	      50 allocs/op

BenchmarkFormat_JSON-8                            	  274134	      4263 ns/op	     989 B/op	      28 allocs/op
BenchmarkFormat_YAML-8                            	   49485	     24194 ns/op	   17885 B/op	     161 allocs/op
BenchmarkFormatDetect-8                           	464196988	         2.588 ns/op	       0 B/op	       0 allocs/op

BenchmarkParallel_Gopantic_Simple-8               	  653206	      1986 ns/op	    1002 B/op	      28 allocs/op
BenchmarkParallel_StdJSON_Simple-8                	  948778	      1634 ns/op	     391 B/op	      12 allocs/op
```

## Reproducing Benchmarks

```bash
# Clone the repository
git clone https://github.com/1mb-dev/gopantic.git
cd gopantic

# Run all benchmarks
go test -bench=. -benchmem ./tests/... -run=^$

# Run specific benchmark categories
go test -bench=Validation -benchmem ./tests/...
go test -bench=ParseValidate -benchmem ./tests/...
go test -bench=Parallel -benchmem ./tests/...
```

## Notes

1. **Benchmark stability:** Results may vary by ±10% between runs due to CPU frequency scaling and system load.

2. **Validation scope:** go-playground/validator supports more validation rules out of the box. gopantic focuses on core validators with extensibility.

3. **Memory vs CPU:** gopantic trades some memory for features like type coercion and format flexibility.

4. **Real-world usage:** Microbenchmarks don't capture all production scenarios. Profile your specific workload.
