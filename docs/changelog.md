# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.5.0] - 2026-11-01

A value gopantic returns without error is now the value the input meant, and every rule on a field either decoder can set has run. So input that v1.4 accepted can fail. If you need time to adapt, stay on v1.4.1 and work through **Behavior changes** below. Each item says what you will see and what to do, and names the row in `tests/contract_test.go` that holds it.

### Behavior changes

Input that used to pass can now be rejected, depending on the data:

- **Rules run on elements and values.** v1.4 skipped rules on slice elements, array elements, map values and top-level slices on some paths. You will see validation errors on data that passed before. Fix the data or loosen the rule. Tests: `validation/[]struct elements`, `validation/map[string]struct elements`, `validation/top-level []struct`.
- **Rules run on every parse.** v1.4 could stop checking nested structs for the rest of the process after the first parse of a type. You will see errors that came and went before. Tests: `validation/nested rule, second parse`, `validation/pointer rule, second parse`.
- **Numbers must fit.** `300` into an `int8`, `4.7` into an `int` and `-1` into a `uint` used to wrap, truncate or pass. They're now parse errors, in fields, slices, arrays and pointers, and in YAML as well as JSON. Use a wider type, or a float if fractions are real. Tests: the `numeric` group, and `numeric/YAML float does not truncate into an int`.
- **Unix timestamps must fall in years 1 to 9999.** `1e300` and YAML `.nan` into a `time.Time` used to give a garbage date. Tests: `numeric/time rejects a huge number`, `numeric/time rejects YAML NaN`.
- **`json.Number` must hold a number.** `{"n":"abc"}` used to store `abc`. YAML keeps the digits as written, and spellings only YAML has, like `0x10`, store their value. Tests: `numeric/an invalid json.Number is rejected`, `numeric/a YAML json.Number keeps its digits`.
- **Floats at or past 2^53 no longer convert to integers.** `1e16` into an `int` errors. Send the digits. Test: `numeric/exponent integer past 2^53 rejected`.
- **A time with no date is rejected.** `12:30:00` used to parse to year 0. Send a full date. Test: `numeric/a time with no date is rejected`.
- **`"NaN"` and `"Inf"` no longer convert to floats.** Send a number, or take a string field. Tests: `numeric/string NaN rejected`, `numeric/string Inf rejected`.
- **Objects no longer convert to strings.** v1.4 returned Go syntax such as `map[a:1]`. Use a struct, a map or `json.RawMessage`. Test: `numeric/object is not a string`.
- **`required` fails on a zero fixed array.** It always passed. Test: `validation/required fails on a zero fixed array`.
- **Rules past 32 levels of nesting are errors.** v1.4 skipped the limit on inputs that needed coercion. A rule that deep now fails rather than going unchecked; raise it with `SetMaxValidationDepth`. Types with no rules aren't affected. Test: `validation/rule below the depth limit fails closed`.

- **Large YAML mappings can be refused.** A document needing more than 50,000,000 key comparisons now errors before decoding: one mapping of 10,000 keys, or 100 of 1,000. Raise it with `SetMaxYAMLKeyComparisons`, or pass 0 to turn the budget off. Test: `TestYAMLGuardChargesAllComparisons`.

- **Some YAML keys are now errors.** A null key can't go in a typed map, and a key that appears with two types through a merge can't go in an interface-keyed map. Both used to store the wrong thing. A null key on a struct is still ignored like any unknown key. Tests: `yaml/a null key cannot go in a string map`, `yaml/merged keys of two types cannot share an interface map`.

Types that used to work can now fail on every parse:

- **A tag that can't run is an error**, for every parse and `Validate` of that type, even under an empty slice. This covers misspelled rules, a missing or malformed parameter such as `min` or `min=abc`, and stray commas. Rules that were never built in are the likely surprise: older docs showed `gtfield` as a feature, and it never existed. `eqfield` and `oneof` appeared as custom rules to register, so they fail only if you copied the tag without the registration. Register the rule or remove the tag. Use `ParseValidationTags(...).TagErrors` to find these in a test. Tests: the `tags` group.

`Validate` can now fail where it passed:

- **`Validate` runs rules on `json:"-"` fields.** v1.4 skipped them everywhere. Parsing still skips them when no decoder can set the field, but a field with `json:"-" yaml:"x"` keeps its rules on every parse, and `required` there now fails on JSON input. See [Fields input cannot set](guide/validation.md#fields-input-cannot-set). Tests: `validation/Validate runs the rule on a json:"-" field`, `validation/input does not choose the format that skips a rule`.

Values can change:

- **One field needing coercion no longer changes the others.** v1.4 switched to a weaker decoder for the whole input, which lost `json.RawMessage`, `UnmarshalJSON`, embedded fields, exact large integers and case-insensitive keys. All of these now match `encoding/json`. Code that depended on the old results will see the correct ones. Tests: the `divergence` group.
- **Fixed arrays follow `encoding/json`** when coercing: extra elements are dropped and missing ones are zero, where v1.4 returned an error. Use a slice with `len=N` to require a length. Test: seed `3bcd1ebd990ef103` under `FuzzWalkerDifferential`.
- **Absent slices stay nil**, not empty, when coercing. Test: `divergence/containers absent`.
- **YAML keys are exact.** `username:` no longer fills a field tagged `json:"user_name"`. The key comes from the `yaml` tag, then the `json` tag, then the lowercased field name. Tests: `yaml/json tag honoured without coercion`, `yaml/yaml tag honoured inside []struct`.

Errors look different:

- **`Field` is the leaf name and `FieldPath` the full path**, such as `Name` and `Items[1].Name`. Test: `validation/FieldPath is the full path`.
- **A parse reports at most 100 errors**, then one saying it stopped. Test: `validation/error count is capped on []struct`.
- **`ParseError` messages leave out values of sensitive fields.** Read `SanitizedValue()` or `Value` instead of parsing the message. Tests: the `errors` group.
- **A rejected object or array no longer sits in `ParseError.Value`.** It holds `"object"` or `"array"`, since a secret could be nested inside. Test: `errors/a rejected object does not carry its contents`.

- **`SanitizedValue` hides containers.** A failed rule on a slice or map returns `[REDACTED]`. `Value` still holds it. Test: `errors/a failed rule on a container keeps its contents out of the report`.

Settings:

- **`SetMaxValidationDepth(0)` no longer turns the limit off**, since a pointer loop would then overflow the stack. Zero or less means the default of 32. Test: `validation/a depth of 0 cannot turn the limit off`.
- **A negative `SetMaxCacheSize` means the default**, where it used to panic. Test: `validation/a negative cache size does not panic`.

`CachedParser` differs:

- **Results are copies.** Code that relied on two callers sharing a slice, which was never safe, will no longer see it. Types holding references gopantic can't copy are parsed on every call. Tests: `cache/mutating a hit does not reach the next caller`, `TestCachedParserBypassesUncopyableTypes`.

### Security

- A small YAML document could exhaust memory or CPU. `yaml.v3` compares every pair of keys and formats an error for each duplicate, so 15 KB of repeated keys allocated 800 MB and 1.2 MB of distinct keys took 2 minutes, far below `MaxInputSize`. `ParseInto` auto-detects YAML, so this reached endpoints that only expect JSON. Duplicates are now rejected in linear time and the remaining cost has a budget. Use `ParseIntoWithFormat` with `FormatJSON` for JSON-only input.
- Validation could be skipped: nested rules stopped running after the first parse of a type, and rules on slice elements, map values and top-level slices didn't always run. Input that should have been rejected was returned as valid.
- `CachedParser` returned shared slices and maps, so one caller could alter what the next received. Keys for input under 1 KB used a 64-bit FNV hash, which can be made to collide; they are now SHA-256.
- Integers wrapped silently on overflow, so a limit checked after parsing could be passed with a value that wrapped into range.
- `ParseError` messages quoted values of sensitive fields, and values of `json:"-"` fields and of nested fields inside a failed container reached `ToJSON`.
- The YAML depth limit missed maps with non-string keys.
- The validator registry had a data race under concurrent registration.

### Added

- `len` as another name for `length`.
- `SetMaxYAMLKeyComparisons`.
- `ParseError.SanitizedValue()`, and `StructValidation.TagErrors`.
- `ErrorList` works with `errors.As` and `errors.Is`. The documented `errors.As` pattern didn't work before.
- A contract test suite, and differential fuzzing against `encoding/json` and `yaml.v3`.

### Changed

- Rules on a registered name take effect at once: registering on the default registry clears cached rules, so `ClearValidationCache` is no longer needed for that.
- `yaml.v3` is the only dependency. `go-playground/validator` left the module graph; the comparison benchmarks are a nested module. The minimum Go version stays 1.24.
- Correctly typed JSON now parses and validates in about half the time of v1.4.1, because it no longer decodes twice. YAML takes about 23% less, and input needing coercion about 14% less. See [Benchmarks](benchmarks.md).

### Fixed

- `NewCachedParser[any]` and a second `Close` no longer panic, and neither do `map` or pointer targets needing coercion.
- `NaN` now fails `required`, `min` and `max`.
- The package builds on 32-bit platforms again, with a CI job to keep it that way.
- `CachedParser` checks input size before hashing.
- Docs rewritten against the code, and a test now holds their examples, tags and links to it.

## [1.4.1] - 2026-02-15

### Changed

- Module path moved to `github.com/1mb-dev/gopantic`.
- Minimum Go version is 1.24.0.
- `go-playground/validator` became a direct requirement, for the comparison tests. Removed in 1.5.0.

## [1.4.0] - 2026-01-19

### Security

- **Structure depth limit**: Added `MaxStructureDepth` configuration (default: 64) to protect against deeply nested JSON/YAML causing resource exhaustion
- **Sensitive field sanitization**: New `SanitizedValue()` method on `ValidationError` automatically redacts sensitive field values (passwords, tokens, API keys) in error output
- **Configurable patterns**: Thread-safe `GetSensitiveFieldPatterns()`, `SetSensitiveFieldPatterns()`, and `AddSensitiveFieldPattern()` for customizing sensitive field detection

### Added

- **Fuzzing tests**: 7 fuzz targets for parser and coercion edge case discovery
- **Validator registry tests**: Comprehensive coverage for all 7 registry methods
- **Depth limit tests**: Tests for JSON and YAML depth enforcement
- **Benchmark comparison document**: Performance comparison with go-playground/validator

### Changed

- **ToStructuredReport()**: Now uses sanitized values for sensitive fields
- **Error handling**: `IsSensitiveField()` exported for custom sanitization logic

### Documentation

- **Fixed validator documentation**: Corrected docs to accurately reflect built-in validators (removed non-existent validators like `url`, `uuid`, `oneof`)
- **Benchmark guide**: New `docs/benchmarks.md` with methodology and results
- **Validation guide**: Clarified custom validator registration patterns

### Testing

- Added `go-playground/validator/v10` as test dependency for comparison benchmarks
- Fuzzing tests exercise ParseInto and CoerceValue with various input patterns
- Registry tests verify validator registration and factory patterns

---

## [1.3.0] - 2026-01-19

### Security

- **Fixed cache key hash collision vulnerability**: Cache keys now use full SHA256 hash instead of truncated 64-bit hash for large inputs (≥1KB), preventing potential cache poisoning attacks
- **Thread-safe configuration**: Added Get/Set functions with synchronization for `MaxInputSize`, `MaxCacheSize`, and `MaxValidationDepth`
- **SECURITY.md**: Updated comprehensive security policy with detailed documentation

### Added

- **Documentation site**: New documentation at [1mb-dev.github.io/gopantic](https://1mb-dev.github.io/gopantic/) with MkDocs Material theme
  - Dark/light mode toggle
  - Full-text search
  - Organized guide and reference sections
- **"Why gopantic" narrative**: New building-gopantic.md explaining the problem, solution, and design decisions
- **Expanded cache tests**: Cache module test coverage significantly improved

### Changed

- **Documentation structure**: Reorganized docs into clear guide/ and reference/ hierarchy
- **Documentation quality**: All docs reviewed for accuracy and clarity
- **PostgreSQL tutorial**: Moved to examples/postgresql_jsonb/ as runnable code

### Fixed

- **Custom validator docs**: Corrected function signatures in validation.md to match actual API
- **architecture.md**: Fixed cache terminology (FIFO, not LRU) and package file list
- **migration.md**: Corrected validator references (`alphanum` not `alphanumeric`)

### Deprecation Notices

- Direct modification of `MaxInputSize`, `MaxCacheSize`, `MaxValidationDepth` is deprecated
- Use `GetMaxInputSize()`/`SetMaxInputSize()` and equivalent functions instead
- Old variables still work but may be removed in v2.0

---

## [1.2.0] - 2025-11-23

### Added

- **`json.RawMessage` support**: Full support for `json.RawMessage` fields, enabling flexible metadata storage and PostgreSQL JSONB integration
- **Standalone `Validate[T](*T)` function**: Validate structs independently of parsing, allowing validation of data from any source (database, environment variables, etc.)
- **Hybrid unmarshal strategy**: Uses standard `encoding/json` and `gopkg.in/yaml.v3` as base unmarshaler, then applies selective type coercion only where needed
- **Recursive nested struct validation**: Validation now properly handles deeply nested structs and pointer-to-struct fields
- **Performance optimizations**:
  - Validation tag caching for 10-20% faster repeated parsing
  - No-validation fast path for types without validation rules
  - FNV-1a hash for cache keys on small inputs (<1KB)
  - Result: 46% faster parsing, 64% less memory, 67% fewer allocations

### Changed

- **Parsing architecture**: Refactored to use standard library unmarshalers first, with fallback to map-based coercion for complex type conversion cases
- **Better ecosystem compatibility**: Works seamlessly with custom `UnmarshalJSON` methods and standard Go patterns
- **Performance vs stdlib JSON**: Gap reduced from 3.9x slower to 1.7x slower
- **Test suite consolidation**: Reorganized from 2 directories to flat structure, added concurrency and edge case tests (56→100 tests)

### Fixed

- **Issue #10**: `json.RawMessage` fields no longer cause "cannot coerce map to slice" errors
- **Nested struct validation**: Fixed validation not being applied to nested struct fields
- **Cross-field validation**: Improved handling of cross-field validators in complex struct hierarchies

### Code Quality

- **Removed dead code**: Eliminated 147 lines of unused functions (`applySelectiveCoercion`, `ParseIntoCached` wrappers)
- **Fixed magic numbers**: Replaced hardcoded values with `math.MaxInt64` constant (4 locations)
- **Consolidated duplicate functions**: Unified `getFieldKey` implementation, removed duplicate type converters
- **Reduced documentation verbosity**: Streamlined package docs from 100 to 29 lines while retaining essential information
- **Production-ready cleanup**: Removed AI-generated bloat, improved code maintainability and consistency

### Documentation

- Added `json.RawMessage` usage examples to README
- Added standalone `Validate()` function documentation
- Added PostgreSQL JSONB integration pattern examples
- Updated README with concise performance metrics
- Comprehensive test suite for `json.RawMessage` scenarios (tests/rawmessage_test.go)
- Consolidated and compacted all documentation for better readability

### Breaking Changes

None - this release is fully backward compatible. Existing code continues to work unchanged.

### Migration Notes

**New recommended pattern for `json.RawMessage` fields:**

```go
type Request struct {
    Name        string          `json:"name" validate:"required"`
    MetadataRaw json.RawMessage `json:"metadata,omitempty"`
}

// Now works seamlessly
req, err := model.ParseInto[Request](body)
```

**New standalone validation pattern:**

```go
var req Request
json.Unmarshal(body, &req)  // Use standard library
model.Validate(&req)         // Apply gopantic validation
```

## [1.1.0] - 2025-10-30

### Added

- **Validation metadata caching**: Struct validation rules now cached by type using `sync.Map` for 10-20% performance improvement on repeated parsing
- **Cache hit rate tracking**: `CachedParser.Stats()` now returns actual hit rate via atomic counters for better observability
- **Proactive cache cleanup**: Background goroutine periodically removes expired entries (configurable via `CleanupInterval`)
- **Input size protection**: Added `MaxInputSize` variable (default 10MB) to prevent DoS attacks from oversized inputs
- **Security policy**: Added SECURITY.md with vulnerability disclosure process and security best practices
- **Migration guide**: Comprehensive docs/migration.md covering migration from encoding/json, validator, and YAML libraries
- **Dependabot automation**: Weekly automated dependency updates for Go modules and GitHub Actions
- **`ClearValidationCache()` function**: Manual cache invalidation for dynamic validator registration scenarios
- **`Close()` method**: Properly stops background cleanup goroutine on CachedParser

### Changed

- **Optimized time parsing**: Heuristic-based format selection tries RFC3339/ISO 8601 first when 'T' detected at position 10
- **Cache configuration**: Added `CleanupInterval` field to `CacheConfig` (default: 30 minutes)
- **Documentation clarity**: Updated README and architecture docs to clarify cache use cases and limitations
- **golangci-lint v2 compatibility**: Updated configuration for golangci-lint v2 with version pinning for consistency

### Fixed

- **golangci-lint configuration**: Fixed YAML syntax and made compatible with v2.x
- **Test coverage reporting**: Added `-coverpkg=./pkg/...` flag to properly capture package coverage
- **CI cleanup**: Removed unused Redis service configurations from all GitHub Actions workflows
- **Linter version consistency**: Pinned golangci-lint to v2.5.0 in CI to match local development

### Performance

- 10-20% faster validation for repeated struct types via metadata caching
- Optimized time parsing for common RFC3339 timestamps
- Proactive cleanup prevents memory accumulation in long-running services

### Security

- DoS protection via configurable input size limits
- Documentation of error message content for secure handling of sensitive data
- Automated dependency vulnerability scanning

### Documentation

- New migration guide (200+ lines) covering common patterns
- Security policy with vulnerability reporting process
- Enhanced API docs with security considerations
- Clarified cache effectiveness and appropriate use cases

## [1.0.1] - 2025-09-16

### Fixed
- **GitHub Actions workflows** - Resolved tar cache extraction failures in nightly performance monitoring
- **Benchmark targets** - Corrected Make targets to point to correct tests directory structure
- **CI pipeline** - Fixed lint and security job failures, added proper SARIF upload permissions
- **Repository security** - Removed non-existent security scan actions and corrected gosec paths

### Changed
- **Documentation script** - Streamlined generate-docs.sh for cleaner, minimal output
- **Build artifacts** - Enhanced .gitignore for better profiling and benchmark file management
- **Code quality** - Removed TODO comments from production code for release readiness

## [1.0.0] - 2025-09-13

### Added
- **JSON/YAML parsing** with automatic format detection
- **Type coercion** - automatic conversion between compatible types (`"123"` → `123`, `"true"` → `true`)
- **Comprehensive validation** using struct tags (`validate:"required,email,min=5"`)
- **Cross-field validation** - validate fields against each other (password confirmation, field comparisons)
- **Built-in validators**: `required`, `min`, `max`, `email`, `alpha`, `alphanum`, `length`
- **Nested struct support** with full validation propagation
- **Array and slice parsing** with element validation
- **Time parsing** support (RFC3339, Unix timestamps, custom formats)
- **Pointer support** for optional fields (`*string`, `*int`)
- **High-performance caching** - 5-27x speedup for repeated parsing operations
- **Thread-safe** concurrent parsing operations
- **Generics support** for type-safe parsing with `ParseInto[T]()`
- **Structured error reporting** with field paths and detailed context
- **Custom validator registration** for domain-specific validation rules
- **Zero dependencies** (except optional YAML support via `gopkg.in/yaml.v3`)

### Features
- Single-function API: `model.ParseInto[T](data)` covers most use cases
- Automatic format detection (JSON vs YAML)
- Comprehensive example collection covering real-world scenarios
- Production-ready with extensive test coverage
- Compatible with Go 1.23+

### Examples
- Basic parsing with type coercion
- Validation examples with multiple error handling
- Cross-field validation (password confirmation, email differences)
- Time parsing with multiple format support
- YAML configuration parsing
- API request/response validation
- Pointer field handling for optional data
- High-performance caching usage

---

## Version Format
This project uses [Semantic Versioning](https://semver.org/):
- **MAJOR** version for incompatible API changes
- **MINOR** version for backwards-compatible functionality additions
- **PATCH** version for backwards-compatible bug fixes