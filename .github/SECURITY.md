# Security Policy

## Supported Versions

Currently supported versions with security updates:

| Version | Supported          |
| ------- | ------------------ |
| 1.3.x   | :white_check_mark: |
| 1.2.x   | :white_check_mark: |
| < 1.2   | :x:                |

## Reporting a Vulnerability

If you discover a security vulnerability in gopantic, please report it responsibly:

**Do NOT file public issues for security vulnerabilities.**

1. **Open a GitHub Security Advisory** at https://github.com/1mb-dev/gopantic/security/advisories/new
2. Alternatively, contact the maintainer directly via the email in the GitHub profile

Please include:
- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Affected versions
- Suggested fix (if available)

### Response Timeline

- Initial response: Within 48 hours
- Status update: Within 7 days
- Fix timeline: Depends on severity (critical issues prioritized)

## Security Considerations

### Input Size Limits

gopantic enforces input size limits to prevent resource exhaustion attacks:

```go
// Default: 10MB
model.MaxInputSize = 10 * 1024 * 1024

// For thread-safe modification at runtime:
model.SetMaxInputSize(5 * 1024 * 1024)  // 5MB

// Disable size limit (not recommended for untrusted input):
model.SetMaxInputSize(0)
```

**Recommendation**: Set appropriate limits based on your expected input sizes. Large limits increase memory exhaustion risk.

### Validation Depth

Nested struct validation has a depth limit to prevent stack overflow:

```go
// Default: 32 levels
model.MaxValidationDepth = 32

// For thread-safe modification at runtime:
model.SetMaxValidationDepth(16)  // Stricter limit
```

**Recommendation**: Keep the default unless you have deeply nested structures that require it.

### Cache Size

The validation cache has a size limit to prevent unbounded memory growth:

```go
// Default: 1000 types
model.MaxCacheSize = 1000

// For thread-safe modification at runtime:
model.SetMaxCacheSize(500)

// Disable caching (not recommended):
model.SetMaxCacheSize(0)
```

**Recommendation**: The default is appropriate for most applications. Adjust only if you're working with an unusually large number of struct types.

### Error Message Content

Error messages include field names and may include input values:

```go
result, err := model.ParseInto[User](data)
if err != nil {
    // err may contain: "field 'password' validation failed: min length 8"
    // Do NOT expose raw errors to untrusted clients
    log.Printf("Parse error: %v", err)  // OK for server logs
    http.Error(w, "Invalid input", 400)  // Sanitized for client
}
```

**Recommendation**: Always sanitize errors before returning to untrusted clients. Log full errors server-side only.

### YAML Security

gopantic uses `gopkg.in/yaml.v3` with safe defaults:
- No arbitrary code execution
- Safe handling of YAML-specific constructs

However, YAML parsing can be more memory-intensive than JSON:

```go
// For untrusted YAML input, consider stricter limits:
model.SetMaxInputSize(1 * 1024 * 1024)  // 1MB for YAML
```

### Reflection Usage

The library uses reflection for type coercion and validation:
- Only exported struct fields are accessed
- Field tags control behavior (`json`, `validate`)
- Ensure struct definitions don't expose unintended fields

### Dependency Security

gopantic maintains minimal production dependencies:
- `gopkg.in/yaml.v3` - YAML parsing

Run regular security checks:
```bash
# Check for known vulnerabilities
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...

# Verify module integrity
go mod verify
```

## Thread Safety

All public APIs are thread-safe for concurrent use:

| API | Thread-Safety |
|-----|---------------|
| `ParseInto[T]` / `ParseIntoWithFormat[T]` | Safe (stateless) |
| `Validate[T]` | Safe (stateless) |
| `CachedParser[T]` | Safe (internal RWMutex) |
| `GetMaxInputSize()` / `SetMaxInputSize()` | Safe (synchronized) |
| `GetMaxCacheSize()` / `SetMaxCacheSize()` | Safe (synchronized) |
| `GetMaxValidationDepth()` / `SetMaxValidationDepth()` | Safe (synchronized) |

**Note**: Direct modification of `MaxInputSize`, `MaxCacheSize`, and `MaxValidationDepth` variables is NOT thread-safe. Use the Get/Set functions for runtime configuration changes.

## Security Changelog

### v1.3.0
- Fixed cache key hash collision vulnerability (truncated SHA256)
- Added thread-safe configuration accessors

### v1.2.0
- Added input size limits
- Added validation depth limits
