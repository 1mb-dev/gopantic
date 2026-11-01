package model

import (
	"crypto/sha256"
	"reflect"
	"sync"
	"sync/atomic"
	"time"
)

// CacheConfig holds basic cache configuration
type CacheConfig struct {
	TTL             time.Duration // Time to live for cached entries (default: 1 hour)
	MaxEntries      int           // Maximum number of cached entries, not bytes (default: 1000)
	CleanupInterval time.Duration // How often to run cleanup (default: TTL/2, 0 to disable)
}

// DefaultCacheConfig returns sensible defaults for in-memory caching
func DefaultCacheConfig() *CacheConfig {
	return &CacheConfig{
		TTL:             time.Hour,
		MaxEntries:      1000,
		CleanupInterval: 30 * time.Minute, // Half of default TTL
	}
}

// cacheKey identifies an input. A parser holds one T, so the type is not part of the key.
type cacheKey struct {
	sum    [sha256.Size]byte
	format Format
}

// cacheEntry represents a single cached item
type cacheEntry[T any] struct {
	value     T
	timestamp time.Time
}

// CachedParser provides simple in-memory caching for parsing results.
//
// Every result is an independent copy, so changing one cannot reach a later caller. A T holding
// references this package cannot copy, such as an unexported slice, is parsed on every call and
// counted as a miss.
type CachedParser[T any] struct {
	cache       map[cacheKey]cacheEntry[T]
	mu          sync.RWMutex
	config      *CacheConfig
	class       refClass
	hits        uint64
	misses      uint64
	stopCleanup chan struct{}
	closeOnce   sync.Once
}

// NewCachedParser creates a new cached parser with optional configuration.
// If CleanupInterval > 0, a background goroutine will periodically clean expired entries.
// Call Close() when done to stop the cleanup goroutine.
func NewCachedParser[T any](config *CacheConfig) *CachedParser[T] {
	if config == nil {
		config = DefaultCacheConfig()
	}

	class := classOf(reflect.TypeFor[T]())

	cp := &CachedParser[T]{
		cache:       make(map[cacheKey]cacheEntry[T]),
		config:      config,
		class:       class,
		stopCleanup: make(chan struct{}),
	}

	// Start cleanup goroutine if interval is configured
	if config.CleanupInterval > 0 {
		go cp.cleanupLoop()
	}

	return cp
}

// Parse parses data with caching support
func (cp *CachedParser[T]) Parse(data []byte) (T, error) {
	if err := checkInputSize(data); err != nil {
		var zero T
		return zero, err
	}
	return cp.ParseWithFormat(data, DetectFormat(data))
}

// ParseWithFormat parses data with format specification and caching
func (cp *CachedParser[T]) ParseWithFormat(data []byte, format Format) (T, error) {
	if err := checkInputSize(data); err != nil {
		var zero T
		return zero, err
	}

	if cp.class.uncopyable {
		atomic.AddUint64(&cp.misses, 1)
		return ParseIntoWithFormat[T](data, format)
	}

	key := cacheKey{sum: sha256.Sum256(data), format: format}

	// Try cache first
	if cached, found := cp.get(key); found {
		return cached, nil
	}

	// Parse and cache
	result, err := ParseIntoWithFormat[T](data, format)
	if err != nil {
		var zero T
		return zero, err
	}

	cp.set(key, result)
	return result, nil
}

// get retrieves a value from cache with TTL check
func (cp *CachedParser[T]) get(key cacheKey) (T, bool) {
	cp.mu.RLock()
	entry, exists := cp.cache[key]
	cp.mu.RUnlock()

	if !exists {
		atomic.AddUint64(&cp.misses, 1)
		var zero T
		return zero, false
	}

	// Check TTL
	if time.Since(entry.timestamp) > cp.config.TTL {
		// Entry expired, clean up with write lock
		cp.mu.Lock()
		delete(cp.cache, key)
		cp.mu.Unlock()
		atomic.AddUint64(&cp.misses, 1)
		var zero T
		return zero, false
	}

	result, ok := cp.clone(entry.value)
	if !ok {
		atomic.AddUint64(&cp.misses, 1)
		var zero T
		return zero, false
	}
	atomic.AddUint64(&cp.hits, 1)
	return result, true
}

// clone returns a value sharing nothing with the original.
func (cp *CachedParser[T]) clone(value T) (T, bool) {
	if !cp.class.holdsRefs {
		return value, true
	}
	ok := detachRefs(reflect.ValueOf(&value).Elem(), 0)
	return value, ok
}

// set stores a value in cache with size limit enforcement
func (cp *CachedParser[T]) set(key cacheKey, value T) {
	stored, ok := cp.clone(value)
	if !ok {
		return
	}

	cp.mu.Lock()
	defer cp.mu.Unlock()

	// Evict if at capacity
	if len(cp.cache) >= cp.config.MaxEntries {
		cp.evictOldest()
	}

	cp.cache[key] = cacheEntry[T]{
		value:     stored,
		timestamp: time.Now(),
	}
}

// evictOldest removes the oldest entry from cache
func (cp *CachedParser[T]) evictOldest() {
	var oldestKey cacheKey
	var oldestTime time.Time
	found := false

	for key, entry := range cp.cache {
		if !found || entry.timestamp.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.timestamp
			found = true
		}
	}

	if found {
		delete(cp.cache, oldestKey)
	}
}

// ClearCache removes all cached entries
func (cp *CachedParser[T]) ClearCache() {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	cp.cache = make(map[cacheKey]cacheEntry[T])
}

// Close stops the background cleanup goroutine if running.
// After calling Close, the parser can still be used but expired entries
// will only be cleaned up on access rather than proactively.
func (cp *CachedParser[T]) Close() {
	cp.closeOnce.Do(func() {
		if cp.stopCleanup != nil {
			close(cp.stopCleanup)
		}
	})
}

// cleanupLoop runs periodically to remove expired entries
func (cp *CachedParser[T]) cleanupLoop() {
	ticker := time.NewTicker(cp.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cp.cleanupExpired()
		case <-cp.stopCleanup:
			return
		}
	}
}

// cleanupExpired removes all expired entries from the cache
func (cp *CachedParser[T]) cleanupExpired() {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	now := time.Now()
	for key, entry := range cp.cache {
		if now.Sub(entry.timestamp) > cp.config.TTL {
			delete(cp.cache, key)
		}
	}
}

// Stats returns cache statistics including size, max size, and hit rate
func (cp *CachedParser[T]) Stats() (size, maxSize int, hitRate float64) {
	cp.mu.RLock()
	size = len(cp.cache)
	cp.mu.RUnlock()

	hits := atomic.LoadUint64(&cp.hits)
	misses := atomic.LoadUint64(&cp.misses)
	total := hits + misses

	if total == 0 {
		return size, cp.config.MaxEntries, 0.0
	}

	hitRate = float64(hits) / float64(total)
	return size, cp.config.MaxEntries, hitRate
}

// maxCopyDepth bounds detachRefs: a custom unmarshaler can build a cycle, and overflowing the
// stack cannot be recovered. A value this deep is not cached.
const maxCopyDepth = 1024

// refClass says what copying a value of a type involves.
type refClass struct {
	holdsRefs  bool
	uncopyable bool // Holds a reference detachRefs cannot reach
}

// refClasses caches classOf. It runs on every interface value in a result, so a type must not
// be classified again each time.
var refClasses sync.Map // map[reflect.Type]refClass

func classOf(typ reflect.Type) refClass {
	if class, ok := refClasses.Load(typ); ok {
		return class.(refClass)
	}
	class := classifyRefs(typ, make(map[reflect.Type]refClass))
	refClasses.Store(typ, class)
	return class
}

// classifyRefs classifies typ. seen keeps results rather than a visited flag: a type reached a
// second time, through an unexported field, still has to answer. A type in progress answers
// with its kind, because recursion always passes through a reference kind, and an unexported
// field of that kind has to count as holding one.
func classifyRefs(typ reflect.Type, seen map[reflect.Type]refClass) refClass {
	if class, ok := seen[typ]; ok {
		return class
	}
	seen[typ] = refClass{holdsRefs: isReferenceKind(typ.Kind())}

	class := classifyKind(typ, seen)
	seen[typ] = class
	return class
}

func isReferenceKind(kind reflect.Kind) bool {
	switch kind {
	case reflect.Ptr, reflect.Slice, reflect.Map, reflect.Interface, reflect.Chan, reflect.Func, reflect.UnsafePointer:
		return true
	default:
		return false
	}
}

func classifyKind(typ reflect.Type, seen map[reflect.Type]refClass) refClass {
	switch typ.Kind() {
	case reflect.Chan, reflect.Func, reflect.UnsafePointer:
		return refClass{holdsRefs: true, uncopyable: true}
	case reflect.Interface:
		return refClass{holdsRefs: true}
	case reflect.Ptr, reflect.Slice:
		return refClass{holdsRefs: true, uncopyable: classifyRefs(typ.Elem(), seen).uncopyable}
	case reflect.Map:
		uncopyable := classifyRefs(typ.Key(), seen).uncopyable || classifyRefs(typ.Elem(), seen).uncopyable
		return refClass{holdsRefs: true, uncopyable: uncopyable}
	case reflect.Array:
		return classifyRefs(typ.Elem(), seen)
	case reflect.Struct:
		return classifyStruct(typ, seen)
	default:
		return refClass{}
	}
}

func classifyStruct(typ reflect.Type, seen map[reflect.Type]refClass) refClass {
	// time.Time holds a *Location that is never changed once it is built.
	if typ == timeType {
		return refClass{}
	}

	var class refClass
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldClass := classifyRefs(field.Type, seen)
		class.holdsRefs = class.holdsRefs || fieldClass.holdsRefs
		class.uncopyable = class.uncopyable || fieldClass.uncopyable || (fieldClass.holdsRefs && !isCopied(field))
	}
	return class
}

// isCopied reports whether detachRefs enters a field. The exported fields of an unexported
// embedded struct stay settable.
func isCopied(field reflect.StructField) bool {
	return field.IsExported() || (field.Anonymous && field.Type.Kind() == reflect.Struct)
}

// detachRefs replaces every reference inside v, a shallow copy, with its own copy. It reports
// false when something cannot be copied.
func detachRefs(v reflect.Value, depth int) bool {
	if depth > maxCopyDepth {
		return false
	}

	switch v.Kind() {
	case reflect.Ptr:
		return detachPtr(v, depth)
	case reflect.Slice:
		return detachSlice(v, depth)
	case reflect.Map:
		return detachMap(v, depth)
	case reflect.Interface:
		return detachInterface(v, depth)
	case reflect.Array:
		for i := 0; i < v.Len(); i++ {
			if !detachRefs(v.Index(i), depth+1) {
				return false
			}
		}
	case reflect.Struct:
		return detachStruct(v, depth)
	}
	return true
}

func detachPtr(v reflect.Value, depth int) bool {
	if v.IsNil() {
		return true
	}
	fresh := reflect.New(v.Type().Elem())
	fresh.Elem().Set(v.Elem())
	if !detachRefs(fresh.Elem(), depth+1) {
		return false
	}
	v.Set(fresh)
	return true
}

func detachSlice(v reflect.Value, depth int) bool {
	if v.IsNil() {
		return true
	}
	fresh := reflect.MakeSlice(v.Type(), v.Len(), v.Len())
	reflect.Copy(fresh, v)
	if !classOf(v.Type().Elem()).holdsRefs {
		v.Set(fresh)
		return true
	}
	for i := 0; i < fresh.Len(); i++ {
		if !detachRefs(fresh.Index(i), depth+1) {
			return false
		}
	}
	v.Set(fresh)
	return true
}

func detachMap(v reflect.Value, depth int) bool {
	if v.IsNil() {
		return true
	}
	typ := v.Type()
	fresh := reflect.MakeMapWithSize(typ, v.Len())
	for iter := v.MapRange(); iter.Next(); {
		key := reflect.New(typ.Key()).Elem()
		key.Set(iter.Key())
		value := reflect.New(typ.Elem()).Elem()
		value.Set(iter.Value())
		if !detachRefs(key, depth+1) || !detachRefs(value, depth+1) {
			return false
		}
		fresh.SetMapIndex(key, value)
	}
	v.Set(fresh)
	return true
}

// detachInterface classifies the dynamic type here: the static type says nothing about it.
func detachInterface(v reflect.Value, depth int) bool {
	if v.IsNil() {
		return true
	}
	inner := v.Elem()
	class := classOf(inner.Type())
	if class.uncopyable {
		return false
	}
	if !class.holdsRefs {
		return true
	}
	fresh := reflect.New(inner.Type()).Elem()
	fresh.Set(inner)
	if !detachRefs(fresh, depth+1) {
		return false
	}
	v.Set(fresh)
	return true
}

func detachStruct(v reflect.Value, depth int) bool {
	typ := v.Type()
	if typ == timeType {
		return true
	}
	for i := 0; i < typ.NumField(); i++ {
		if isCopied(typ.Field(i)) && !detachRefs(v.Field(i), depth+1) {
			return false
		}
	}
	return true
}
