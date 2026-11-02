package model

import (
	"sync"
)

// Thread-safe configuration accessors
//
// The package-level variables (MaxInputSize, MaxCacheSize, MaxValidationDepth)
// are exported for backwards compatibility but are NOT thread-safe for concurrent
// modification. For concurrent access, use the Get/Set functions below.
//
// Typical usage patterns:
//   - Startup configuration (before goroutines): Direct variable access is fine
//   - Runtime modification (with goroutines): Use Get/Set functions
//
// Note: The Get/Set functions maintain their own copy of the values. After using
// SetMaxInputSize(), internal code will use the new value on subsequent calls.

var (
	configMu   sync.RWMutex
	configOnce sync.Once
)

// configValues holds thread-safe copies of configuration values.
// These are initialized from the exported variables on first access.
var configValues struct {
	maxInputSize           int
	maxCacheSize           int
	maxValidationDepth     int
	maxStructureDepth      int
	maxYAMLKeyComparisons  int
	sensitiveFieldPatterns []string
}

// initConfigOnce ensures configuration is initialized from exported variables
func initConfigOnce() {
	configOnce.Do(func() {
		configValues.maxInputSize = MaxInputSize
		configValues.maxCacheSize = MaxCacheSize
		configValues.maxValidationDepth = MaxValidationDepth
		configValues.maxStructureDepth = MaxStructureDepth
		configValues.maxYAMLKeyComparisons = MaxYAMLKeyComparisons
		configValues.sensitiveFieldPatterns = append([]string{}, DefaultSensitivePatterns...)
	})
}

// GetMaxInputSize returns the maximum input size in a thread-safe manner.
// Default: 10MB (10 * 1024 * 1024 bytes). Set to 0 to disable size checking.
func GetMaxInputSize() int {
	initConfigOnce()
	configMu.RLock()
	defer configMu.RUnlock()
	return configValues.maxInputSize
}

// SetMaxInputSize sets the maximum input size in a thread-safe manner.
// Set to 0 to disable size checking.
//
// Note: This also updates the exported MaxInputSize variable for compatibility,
// but that update is not atomic with respect to direct variable reads.
func SetMaxInputSize(size int) {
	initConfigOnce()
	configMu.Lock()
	defer configMu.Unlock()
	configValues.maxInputSize = size
	MaxInputSize = size
}

// GetMaxCacheSize returns the maximum cache size in a thread-safe manner.
// Default: 1000 types.
func GetMaxCacheSize() int {
	initConfigOnce()
	configMu.RLock()
	defer configMu.RUnlock()
	if configValues.maxCacheSize < 0 {
		return defaultMaxCacheSize
	}
	return configValues.maxCacheSize
}

// SetMaxCacheSize sets the maximum cache size in a thread-safe manner.
// Set to 0 for unlimited caching (not recommended for long-running services).
//
// Note: This also updates the exported MaxCacheSize variable for compatibility,
// but that update is not atomic with respect to direct variable reads.
func SetMaxCacheSize(size int) {
	initConfigOnce()
	configMu.Lock()
	defer configMu.Unlock()
	configValues.maxCacheSize = size
	MaxCacheSize = size
}

// GetMaxValidationDepth returns the maximum validation depth in a thread-safe manner.
// Default: 32 levels.
func GetMaxValidationDepth() int {
	initConfigOnce()
	configMu.RLock()
	defer configMu.RUnlock()
	if configValues.maxValidationDepth <= 0 {
		return defaultMaxValidationDepth
	}
	return configValues.maxValidationDepth
}

// SetMaxValidationDepth sets the maximum validation depth in a thread-safe manner.
// This prevents stack overflow and DoS attacks from deeply nested structures.
//
// Note: This also updates the exported MaxValidationDepth variable for compatibility,
// but that update is not atomic with respect to direct variable reads.
func SetMaxValidationDepth(depth int) {
	initConfigOnce()
	configMu.Lock()
	defer configMu.Unlock()
	configValues.maxValidationDepth = depth
	MaxValidationDepth = depth
}

// GetMaxYAMLKeyComparisons returns the YAML key comparison budget in a thread-safe manner.
// Default: 50,000,000. 0 means no budget, and a negative value counts as the default.
func GetMaxYAMLKeyComparisons() int {
	initConfigOnce()
	configMu.RLock()
	defer configMu.RUnlock()
	if configValues.maxYAMLKeyComparisons < 0 {
		return defaultMaxYAMLKeyComparisons
	}
	return configValues.maxYAMLKeyComparisons
}

// SetMaxYAMLKeyComparisons sets the YAML key comparison budget in a thread-safe manner.
// Set to 0 to disable the budget. Duplicate keys are rejected either way.
func SetMaxYAMLKeyComparisons(limit int) {
	initConfigOnce()
	configMu.Lock()
	defer configMu.Unlock()
	configValues.maxYAMLKeyComparisons = limit
	MaxYAMLKeyComparisons = limit
}

// GetMaxStructureDepth returns the maximum structure nesting depth in a thread-safe manner.
// Default: 64 levels. Set to 0 to disable depth checking.
func GetMaxStructureDepth() int {
	initConfigOnce()
	configMu.RLock()
	defer configMu.RUnlock()
	return configValues.maxStructureDepth
}

// SetMaxStructureDepth sets the maximum structure nesting depth in a thread-safe manner.
// This prevents resource exhaustion from deeply nested JSON/YAML structures.
//
// Note: This also updates the exported MaxStructureDepth variable for compatibility,
// but that update is not atomic with respect to direct variable reads.
func SetMaxStructureDepth(depth int) {
	initConfigOnce()
	configMu.Lock()
	defer configMu.Unlock()
	configValues.maxStructureDepth = depth
	MaxStructureDepth = depth
}

// DefaultSensitivePatterns contains field name patterns that indicate sensitive data.
// These patterns are matched case-insensitively as substrings of field names.
// Fields matching these patterns will have their values redacted in error output.
var DefaultSensitivePatterns = []string{
	"password", "passwd", "secret", "token",
	"key", "credential", "auth", "api_key",
	"apikey", "private", "bearer",
}

// GetSensitiveFieldPatterns returns the current sensitive field patterns in a thread-safe manner.
// Returns a copy of the patterns slice to prevent external modification.
func GetSensitiveFieldPatterns() []string {
	initConfigOnce()
	configMu.RLock()
	defer configMu.RUnlock()
	// Return a copy to prevent external modification
	result := make([]string, len(configValues.sensitiveFieldPatterns))
	copy(result, configValues.sensitiveFieldPatterns)
	return result
}

// SetSensitiveFieldPatterns sets the sensitive field patterns in a thread-safe manner.
// Pass nil or empty slice to disable sensitive field detection.
// Pass custom patterns to override the defaults.
//
// Example:
//
//	model.SetSensitiveFieldPatterns([]string{"password", "secret", "ssn", "credit_card"})
func SetSensitiveFieldPatterns(patterns []string) {
	initConfigOnce()
	configMu.Lock()
	defer configMu.Unlock()
	if patterns == nil {
		configValues.sensitiveFieldPatterns = nil
	} else {
		// Store a copy to prevent external modification
		configValues.sensitiveFieldPatterns = make([]string, len(patterns))
		copy(configValues.sensitiveFieldPatterns, patterns)
	}
}

// AddSensitiveFieldPattern adds a pattern to the sensitive field patterns in a thread-safe manner.
// The pattern is matched case-insensitively as a substring of field names.
func AddSensitiveFieldPattern(pattern string) {
	initConfigOnce()
	configMu.Lock()
	defer configMu.Unlock()
	configValues.sensitiveFieldPatterns = append(configValues.sensitiveFieldPatterns, pattern)
}
