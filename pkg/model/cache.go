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
	hits        atomic.Uint64
	misses      atomic.Uint64
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
		cp.misses.Add(1)
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
		cp.misses.Add(1)
		var zero T
		return zero, false
	}

	// Check TTL
	if time.Since(entry.timestamp) > cp.config.TTL {
		// Entry expired, clean up with write lock
		cp.mu.Lock()
		delete(cp.cache, key)
		cp.mu.Unlock()
		cp.misses.Add(1)
		var zero T
		return zero, false
	}

	// This can fail on a value that cloned when stored: the depth reached depends on map order,
	// and a failed clone still shares references with the cache.
	result, ok := cp.clone(entry.value)
	if !ok {
		cp.misses.Add(1)
		var zero T
		return zero, false
	}
	cp.hits.Add(1)
	return result, true
}

// clone returns a value sharing nothing with the original.
func (cp *CachedParser[T]) clone(value T) (T, bool) {
	if !cp.class.holdsRefs {
		return value, true
	}
	var d detacher
	ok := d.refs(reflect.ValueOf(&value).Elem(), 0)
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

	hits := cp.hits.Load()
	misses := cp.misses.Load()
	total := hits + misses

	if total == 0 {
		return size, cp.config.MaxEntries, 0.0
	}

	hitRate = float64(hits) / float64(total)
	return size, cp.config.MaxEntries, hitRate
}

// maxCopyDepth bounds the recursion in detacher: overflowing the stack cannot be recovered. A
// value this deep is not cached.
const maxCopyDepth = 1024

// refClass says what copying a value of a type involves.
type refClass struct {
	holdsRefs  bool
	uncopyable bool // Holds a reference detacher cannot reach
}

// refClasses caches classOf. It runs for every interface value in a result, so a type must not
// be classified again each time.
var refClasses sync.Map // map[reflect.Type]refClass

func classOf(typ reflect.Type) refClass {
	if class, ok := refClasses.Load(typ); ok {
		return class.(refClass)
	}
	class := refClass{holdsRefs: holdsRefs(typ), uncopyable: isUncopyable(typ, make(map[reflect.Type]bool))}
	refClasses.Store(typ, class)
	return class
}

// holdsRefs reports whether a value of typ contains a reference. A struct can only contain
// itself through one, and a reference stops the recursion, so no cycle guard is needed.
func holdsRefs(typ reflect.Type) bool {
	switch typ.Kind() {
	case reflect.Ptr, reflect.Slice, reflect.Map, reflect.Interface, reflect.Chan, reflect.Func, reflect.UnsafePointer:
		return true
	case reflect.Array:
		return holdsRefs(typ.Elem())
	case reflect.Struct:
		// time.Time holds a *Location that is never changed once it is built.
		if typ == timeType {
			return false
		}
		for i := 0; i < typ.NumField(); i++ {
			if holdsRefs(typ.Field(i).Type) {
				return true
			}
		}
	}
	return false
}

// isUncopyable reports whether anything typ reaches holds a reference out of detacher's reach.
// It follows references, so seen guards cycles.
func isUncopyable(typ reflect.Type, seen map[reflect.Type]bool) bool {
	if seen[typ] {
		return false
	}
	seen[typ] = true

	switch typ.Kind() {
	case reflect.Chan, reflect.Func, reflect.UnsafePointer:
		return true
	case reflect.Ptr, reflect.Slice, reflect.Array:
		return isUncopyable(typ.Elem(), seen)
	case reflect.Map:
		return isUncopyable(typ.Key(), seen) || isUncopyable(typ.Elem(), seen)
	case reflect.Struct:
		return hasUncopyableField(typ, seen)
	}
	return false
}

func hasUncopyableField(typ reflect.Type, seen map[reflect.Type]bool) bool {
	if typ == timeType {
		return false
	}
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if isCopied(field) {
			if isUncopyable(field.Type, seen) {
				return true
			}
		} else if holdsRefs(field.Type) {
			return true
		}
	}
	return false
}

// isCopied reports whether detacher enters a field. The exported fields of an unexported
// embedded struct stay settable.
func isCopied(field reflect.StructField) bool {
	return field.IsExported() || (field.Anonymous && field.Type.Kind() == reflect.Struct)
}

// refKey identifies one reference. A slice needs its length, since two slices of one array
// differ.
type refKey struct {
	typ    reflect.Type
	ptr    uintptr
	length int
}

// detacher replaces every reference inside a shallow copy with its own copy. seen maps each
// reference to its copy, so a value reached twice is copied once. Without it a shared pointer
// costs a copy per path, which is exponential in a DAG and in YAML aliases held by a yaml.Node.
type detacher struct {
	seen map[refKey]reflect.Value
}

func (d *detacher) remember(key refKey, copied reflect.Value) {
	if d.seen == nil {
		d.seen = make(map[refKey]reflect.Value)
	}
	d.seen[key] = copied
}

// refs reports false when something cannot be copied.
func (d *detacher) refs(v reflect.Value, depth int) bool {
	if depth > maxCopyDepth {
		return false
	}

	switch v.Kind() {
	case reflect.Ptr:
		return d.ptr(v, depth)
	case reflect.Slice:
		return d.slice(v, depth)
	case reflect.Map:
		return d.mapRefs(v, depth)
	case reflect.Interface:
		return d.iface(v, depth)
	case reflect.Array:
		for i := 0; i < v.Len(); i++ {
			if !d.refs(v.Index(i), depth+1) {
				return false
			}
		}
	case reflect.Struct:
		return d.structRefs(v, depth)
	}
	return true
}

func (d *detacher) ptr(v reflect.Value, depth int) bool {
	if v.IsNil() {
		return true
	}
	key := refKey{typ: v.Type(), ptr: v.Pointer()}
	if copied, ok := d.seen[key]; ok {
		v.Set(copied)
		return true
	}

	fresh := reflect.New(v.Type().Elem())
	d.remember(key, fresh)
	fresh.Elem().Set(v.Elem())
	if !d.refs(fresh.Elem(), depth+1) {
		return false
	}
	v.Set(fresh)
	return true
}

func (d *detacher) slice(v reflect.Value, depth int) bool {
	if v.IsNil() {
		return true
	}
	fresh := reflect.MakeSlice(v.Type(), v.Len(), v.Len())
	reflect.Copy(fresh, v)
	if !classOf(v.Type().Elem()).holdsRefs {
		v.Set(fresh)
		return true
	}

	key := refKey{typ: v.Type(), ptr: v.Pointer(), length: v.Len()}
	if copied, ok := d.seen[key]; ok {
		v.Set(copied)
		return true
	}
	d.remember(key, fresh)
	for i := 0; i < fresh.Len(); i++ {
		if !d.refs(fresh.Index(i), depth+1) {
			return false
		}
	}
	v.Set(fresh)
	return true
}

func (d *detacher) mapRefs(v reflect.Value, depth int) bool {
	if v.IsNil() {
		return true
	}
	key := refKey{typ: v.Type(), ptr: v.Pointer()}
	if copied, ok := d.seen[key]; ok {
		v.Set(copied)
		return true
	}

	typ := v.Type()
	fresh := reflect.MakeMapWithSize(typ, v.Len())
	d.remember(key, fresh)
	for iter := v.MapRange(); iter.Next(); {
		entryKey := reflect.New(typ.Key()).Elem()
		entryKey.SetIterKey(iter)
		value := reflect.New(typ.Elem()).Elem()
		value.SetIterValue(iter)
		if !d.refs(entryKey, depth+1) || !d.refs(value, depth+1) {
			return false
		}
		fresh.SetMapIndex(entryKey, value)
	}
	v.Set(fresh)
	return true
}

// iface classifies the dynamic type here: the static type says nothing about it.
func (d *detacher) iface(v reflect.Value, depth int) bool {
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
	if !d.refs(fresh, depth+1) {
		return false
	}
	v.Set(fresh)
	return true
}

func (d *detacher) structRefs(v reflect.Value, depth int) bool {
	typ := v.Type()
	if typ == timeType {
		return true
	}
	for i := 0; i < typ.NumField(); i++ {
		if isCopied(typ.Field(i)) && !d.refs(v.Field(i), depth+1) {
			return false
		}
	}
	return true
}
