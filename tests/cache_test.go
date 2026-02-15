package tests

import (
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/1mb-dev/gopantic/pkg/model"
)

// Simple test struct for cache tests
type CacheTestUser struct {
	ID   int    `json:"id"`
	Name string `json:"name" validate:"required"`
}

// Test basic caching functionality
func TestCachedParsing(t *testing.T) {
	parser := model.NewCachedParser[E2EUser](nil)

	userData := `{
		"id": 789,
		"username": "cacheduser",
		"email": "cached@example.com",
		"first_name": "Cached",
		"last_name": "User",
		"age": 32,
		"is_active": true,
		"profile": {
			"bio": "Testing caching",
			"location": "Cache City",
			"skills": ["Caching"],
			"languages": ["Binary"]
		},
		"created_at": "2023-07-01T08:00:00Z"
	}`

	// First parse (cache miss)
	user1, err := parser.Parse([]byte(userData))
	if err != nil {
		t.Fatalf("First parse failed: %v", err)
	}

	// Second parse (cache hit)
	user2, err := parser.Parse([]byte(userData))
	if err != nil {
		t.Fatalf("Second parse failed: %v", err)
	}

	// Results should be identical
	if !reflect.DeepEqual(user1, user2) {
		t.Error("Cached result should be identical to original")
	}

	if user1.Username != "cacheduser" {
		t.Errorf("Expected username 'cacheduser', got '%s'", user1.Username)
	}
}

// TestCachedParser_FIFOEviction verifies that oldest entries are evicted first
// Note: This is FIFO eviction (oldest insertion), not LRU (least recently accessed)
func TestCachedParser_FIFOEviction(t *testing.T) {
	config := &model.CacheConfig{
		MaxEntries:      3,
		TTL:             time.Hour,
		CleanupInterval: 0, // Disable background cleanup
	}
	parser := model.NewCachedParser[CacheTestUser](config)
	defer parser.Close()

	// Add 3 entries
	data1 := []byte(`{"id": 1, "name": "First"}`)
	data2 := []byte(`{"id": 2, "name": "Second"}`)
	data3 := []byte(`{"id": 3, "name": "Third"}`)

	parser.Parse(data1)
	parser.Parse(data2)
	parser.Parse(data3)

	// Verify all 3 are cached
	size, _, _ := parser.Stats()
	if size != 3 {
		t.Errorf("Expected cache size 3, got %d", size)
	}

	// Add 4th entry - should evict the first (FIFO)
	data4 := []byte(`{"id": 4, "name": "Fourth"}`)
	parser.Parse(data4)

	size, _, _ = parser.Stats()
	if size != 3 {
		t.Errorf("Expected cache size 3 after eviction, got %d", size)
	}
}

// TestCachedParser_TTLExpiration verifies that expired entries return cache miss
func TestCachedParser_TTLExpiration(t *testing.T) {
	config := &model.CacheConfig{
		TTL:             100 * time.Millisecond,
		MaxEntries:      1000,
		CleanupInterval: 0, // Disable background cleanup for deterministic testing
	}
	parser := model.NewCachedParser[CacheTestUser](config)
	defer parser.Close()

	data := []byte(`{"id": 1, "name": "Test"}`)

	// First parse - cache miss
	parser.Parse(data)

	// Immediate second parse - should be cache hit
	_, _, hitRate := parser.Stats()
	if hitRate != 0.0 {
		t.Logf("Initial hit rate: %f (expected 0.0 after one miss)", hitRate)
	}

	// Parse again to get a hit
	parser.Parse(data)
	_, _, hitRate = parser.Stats()
	if hitRate != 0.5 {
		t.Errorf("Expected hit rate 0.5 (1 hit, 1 miss), got %f", hitRate)
	}

	// Wait for TTL to expire
	time.Sleep(150 * time.Millisecond)

	// Parse again - should be cache miss (expired)
	parser.Parse(data)

	// hitRate should decrease: 1 hit, 2 misses = 1/3 = 0.333
	_, _, hitRate = parser.Stats()
	expectedRate := 1.0 / 3.0
	if hitRate < expectedRate-0.01 || hitRate > expectedRate+0.01 {
		t.Errorf("Expected hit rate ~0.333 after expiration, got %f", hitRate)
	}
}

// TestCachedParser_CleanupGoroutine verifies background cleanup works
func TestCachedParser_CleanupGoroutine(t *testing.T) {
	config := &model.CacheConfig{
		TTL:             50 * time.Millisecond,
		MaxEntries:      1000,
		CleanupInterval: 25 * time.Millisecond,
	}
	parser := model.NewCachedParser[CacheTestUser](config)
	defer parser.Close()

	// Add entries
	for i := 0; i < 5; i++ {
		data := []byte(fmt.Sprintf(`{"id": %d, "name": "User%d"}`, i, i))
		parser.Parse(data)
	}

	size, _, _ := parser.Stats()
	if size != 5 {
		t.Errorf("Expected cache size 5, got %d", size)
	}

	// Wait for TTL + cleanup interval
	time.Sleep(100 * time.Millisecond)

	// Background cleanup should have removed expired entries
	size, _, _ = parser.Stats()
	if size != 0 {
		t.Errorf("Expected cache size 0 after cleanup, got %d", size)
	}
}

// TestCachedParser_Stats verifies stats accuracy
func TestCachedParser_Stats(t *testing.T) {
	parser := model.NewCachedParser[CacheTestUser](nil)
	defer parser.Close()

	// Initial stats
	size, maxSize, hitRate := parser.Stats()
	if size != 0 {
		t.Errorf("Expected initial size 0, got %d", size)
	}
	if maxSize != 1000 { // default MaxEntries
		t.Errorf("Expected maxSize 1000, got %d", maxSize)
	}
	if hitRate != 0.0 {
		t.Errorf("Expected initial hit rate 0.0, got %f", hitRate)
	}

	// Parse new data (cache miss)
	data1 := []byte(`{"id": 1, "name": "User1"}`)
	parser.Parse(data1)

	size, _, hitRate = parser.Stats()
	if size != 1 {
		t.Errorf("Expected size 1 after first parse, got %d", size)
	}
	if hitRate != 0.0 {
		t.Errorf("Expected hit rate 0.0 after first miss, got %f", hitRate)
	}

	// Parse same data (cache hit)
	parser.Parse(data1)
	_, _, hitRate = parser.Stats()
	if hitRate != 0.5 {
		t.Errorf("Expected hit rate 0.5 (1 hit / 2 total), got %f", hitRate)
	}

	// Parse different data (cache miss)
	data2 := []byte(`{"id": 2, "name": "User2"}`)
	parser.Parse(data2)

	size, _, hitRate = parser.Stats()
	if size != 2 {
		t.Errorf("Expected size 2, got %d", size)
	}
	// 1 hit, 2 misses = 1/3
	expectedRate := 1.0 / 3.0
	if hitRate < expectedRate-0.01 || hitRate > expectedRate+0.01 {
		t.Errorf("Expected hit rate ~0.333, got %f", hitRate)
	}
}

// TestCachedParser_ClearCache verifies cache clearing
func TestCachedParser_ClearCache(t *testing.T) {
	parser := model.NewCachedParser[CacheTestUser](nil)
	defer parser.Close()

	// Add entries
	for i := 0; i < 5; i++ {
		data := []byte(fmt.Sprintf(`{"id": %d, "name": "User%d"}`, i, i))
		parser.Parse(data)
	}

	size, _, _ := parser.Stats()
	if size != 5 {
		t.Errorf("Expected cache size 5, got %d", size)
	}

	// Clear cache
	parser.ClearCache()

	size, _, _ = parser.Stats()
	if size != 0 {
		t.Errorf("Expected cache size 0 after clear, got %d", size)
	}
}

// TestCachedParser_DifferentTypeSameData verifies type isolation in cache
func TestCachedParser_DifferentTypeSameData(t *testing.T) {
	type TypeA struct {
		ID int `json:"id"`
	}
	type TypeB struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	parserA := model.NewCachedParser[TypeA](nil)
	parserB := model.NewCachedParser[TypeB](nil)
	defer parserA.Close()
	defer parserB.Close()

	data := []byte(`{"id": 42, "name": "Test"}`)

	resultA, _ := parserA.Parse(data)
	resultB, _ := parserB.Parse(data)

	if resultA.ID != 42 {
		t.Errorf("TypeA.ID expected 42, got %d", resultA.ID)
	}
	if resultB.ID != 42 || resultB.Name != "Test" {
		t.Errorf("TypeB expected {42, Test}, got {%d, %s}", resultB.ID, resultB.Name)
	}

	// Cache entries should be separate
	sizeA, _, _ := parserA.Stats()
	sizeB, _, _ := parserB.Stats()
	if sizeA != 1 || sizeB != 1 {
		t.Errorf("Expected separate cache entries, got sizeA=%d, sizeB=%d", sizeA, sizeB)
	}
}

// TestCachedParser_LargeInputHashing verifies large inputs use correct hashing
func TestCachedParser_LargeInputHashing(t *testing.T) {
	parser := model.NewCachedParser[CacheTestUser](nil)
	defer parser.Close()

	// Create input > 1KB to trigger SHA256 path
	name := make([]byte, 2000)
	for i := range name {
		name[i] = 'a'
	}
	largeData := []byte(fmt.Sprintf(`{"id": 1, "name": "%s"}`, string(name)))

	// Should not panic and should parse correctly
	result, err := parser.Parse(largeData)
	if err != nil {
		t.Fatalf("Large input parse failed: %v", err)
	}

	if result.ID != 1 {
		t.Errorf("Expected ID 1, got %d", result.ID)
	}

	// Cache hit should work
	result2, _ := parser.Parse(largeData)
	if !reflect.DeepEqual(result, result2) {
		t.Error("Cached result for large input should match")
	}

	_, _, hitRate := parser.Stats()
	if hitRate != 0.5 {
		t.Errorf("Expected hit rate 0.5, got %f", hitRate)
	}
}

// TestCachedParser_LargeInputCollisionResistance verifies different large inputs get different cache entries
func TestCachedParser_LargeInputCollisionResistance(t *testing.T) {
	parser := model.NewCachedParser[CacheTestUser](nil)
	defer parser.Close()

	// Create two different large inputs (>1KB each)
	name1 := make([]byte, 2000)
	name2 := make([]byte, 2000)
	for i := range name1 {
		name1[i] = 'a'
		name2[i] = 'b' // Different content
	}
	data1 := []byte(fmt.Sprintf(`{"id": 1, "name": "%s"}`, string(name1)))
	data2 := []byte(fmt.Sprintf(`{"id": 2, "name": "%s"}`, string(name2)))

	// Parse first input
	result1, err := parser.Parse(data1)
	if err != nil {
		t.Fatalf("Parse data1 failed: %v", err)
	}
	if result1.ID != 1 {
		t.Errorf("Expected ID 1, got %d", result1.ID)
	}

	// Parse second input - should not return cached result from first
	result2, err := parser.Parse(data2)
	if err != nil {
		t.Fatalf("Parse data2 failed: %v", err)
	}
	if result2.ID != 2 {
		t.Errorf("Expected ID 2, got %d (cache collision detected)", result2.ID)
	}

	// Verify both are cached separately
	result1Again, _ := parser.Parse(data1)
	result2Again, _ := parser.Parse(data2)

	if result1Again.ID != 1 || result2Again.ID != 2 {
		t.Error("Cache should store different entries for different large inputs")
	}
}

// TestCachedParser_EmptyInput verifies empty input handling
func TestCachedParser_EmptyInput(t *testing.T) {
	parser := model.NewCachedParser[CacheTestUser](nil)
	defer parser.Close()

	// Empty input should return error
	_, err := parser.Parse([]byte{})
	if err == nil {
		t.Error("Expected error for empty input")
	}

	// Nil input should return error
	_, err = parser.Parse(nil)
	if err == nil {
		t.Error("Expected error for nil input")
	}
}

// TestCachedParser_InvalidJSON verifies invalid JSON handling
func TestCachedParser_InvalidJSON(t *testing.T) {
	parser := model.NewCachedParser[CacheTestUser](nil)
	defer parser.Close()

	// Invalid JSON should return error
	_, err := parser.Parse([]byte(`{"id": 1, "name":}`))
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}

	// Cache should not store failed parses
	size, _, _ := parser.Stats()
	if size != 0 {
		t.Errorf("Expected cache size 0 after failed parse, got %d", size)
	}
}

// TestCachedParser_ValidationError verifies validation errors don't cache
func TestCachedParser_ValidationError(t *testing.T) {
	parser := model.NewCachedParser[CacheTestUser](nil)
	defer parser.Close()

	// Missing required field
	_, err := parser.Parse([]byte(`{"id": 1}`))
	if err == nil {
		t.Error("Expected validation error for missing required field")
	}

	// Failed validations should not be cached
	size, _, _ := parser.Stats()
	if size != 0 {
		t.Errorf("Expected cache size 0 after validation failure, got %d", size)
	}
}

// TestCachedParser_ConcurrentMixedOperations tests concurrent access patterns
func TestCachedParser_ConcurrentMixedOperations(t *testing.T) {
	parser := model.NewCachedParser[CacheTestUser](nil)
	defer parser.Close()

	var wg sync.WaitGroup
	data := []byte(`{"id": 1, "name": "Test"}`)

	// Mix of operations
	for i := 0; i < 100; i++ {
		wg.Add(3)

		go func() {
			defer wg.Done()
			parser.Parse(data)
		}()

		go func() {
			defer wg.Done()
			parser.Stats()
		}()

		go func() {
			defer wg.Done()
			if i%10 == 0 {
				parser.ClearCache()
			}
		}()
	}

	wg.Wait()
	// If we get here without panic, the test passes
}

// TestCachedParser_ConcurrentEviction tests concurrent access during eviction
func TestCachedParser_ConcurrentEviction(t *testing.T) {
	config := &model.CacheConfig{
		MaxEntries:      10,
		TTL:             time.Hour,
		CleanupInterval: 0,
	}
	parser := model.NewCachedParser[CacheTestUser](config)
	defer parser.Close()

	var wg sync.WaitGroup

	// 50 goroutines each adding different data
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			data := []byte(fmt.Sprintf(`{"id": %d, "name": "User%d"}`, id, id))
			parser.Parse(data)
		}(i)
	}

	wg.Wait()

	// Cache should be at or near MaxEntries
	size, maxSize, _ := parser.Stats()
	if size > maxSize {
		t.Errorf("Cache size %d exceeds max %d", size, maxSize)
	}
}

// TestCachedParser_ParseWithFormat tests explicit format parsing
func TestCachedParser_ParseWithFormat(t *testing.T) {
	parser := model.NewCachedParser[CacheTestUser](nil)
	defer parser.Close()

	jsonData := []byte(`{"id": 1, "name": "JSON"}`)
	yamlData := []byte("id: 2\nname: YAML")

	// Parse JSON
	jsonResult, err := parser.ParseWithFormat(jsonData, model.FormatJSON)
	if err != nil {
		t.Fatalf("JSON parse failed: %v", err)
	}
	if jsonResult.Name != "JSON" {
		t.Errorf("Expected name 'JSON', got '%s'", jsonResult.Name)
	}

	// Parse YAML
	yamlResult, err := parser.ParseWithFormat(yamlData, model.FormatYAML)
	if err != nil {
		t.Fatalf("YAML parse failed: %v", err)
	}
	if yamlResult.Name != "YAML" {
		t.Errorf("Expected name 'YAML', got '%s'", yamlResult.Name)
	}

	// Both should be cached
	size, _, _ := parser.Stats()
	if size != 2 {
		t.Errorf("Expected 2 cached entries, got %d", size)
	}
}

// TestCachedParser_DefaultConfig verifies default configuration
func TestCachedParser_DefaultConfig(t *testing.T) {
	config := model.DefaultCacheConfig()

	if config.TTL != time.Hour {
		t.Errorf("Expected default TTL 1 hour, got %v", config.TTL)
	}
	if config.MaxEntries != 1000 {
		t.Errorf("Expected default MaxEntries 1000, got %d", config.MaxEntries)
	}
	if config.CleanupInterval != 30*time.Minute {
		t.Errorf("Expected default CleanupInterval 30 minutes, got %v", config.CleanupInterval)
	}
}
