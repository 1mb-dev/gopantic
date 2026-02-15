# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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

## [1.0.1] - 2025-01-16

### Fixed
- **GitHub Actions workflows** - Resolved tar cache extraction failures in nightly performance monitoring
- **Benchmark targets** - Corrected Make targets to point to correct tests directory structure
- **CI pipeline** - Fixed lint and security job failures, added proper SARIF upload permissions
- **Repository security** - Removed non-existent security scan actions and corrected gosec paths

### Changed
- **Documentation script** - Streamlined generate-docs.sh for cleaner, minimal output
- **Build artifacts** - Enhanced .gitignore for better profiling and benchmark file management
- **Code quality** - Removed TODO comments from production code for release readiness

## [1.0.0] - 2025-01-13

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