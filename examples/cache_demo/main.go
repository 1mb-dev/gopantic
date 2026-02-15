package main

import (
	"fmt"
	"log"
	"time"

	"github.com/1mb-dev/gopantic/pkg/model"
)

// APIResponse represents a typical API response structure
type APIResponse struct {
	ID        int       `json:"id"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	Status    string    `json:"status"`
}

// Config represents application configuration that might be parsed repeatedly
type Config struct {
	Host    string `json:"host"`
	Port    int    `json:"port"`
	Timeout int    `json:"timeout_ms"`
	Enabled bool   `json:"enabled"`
}

func main() {
	fmt.Println("gopantic - CachedParser Demo")
	fmt.Println("================================")

	// Example 1: Basic caching with default configuration
	fmt.Println("\n1. Basic cached parsing with default config:")
	parser := model.NewCachedParser[Config](nil) // Uses default config
	defer parser.Close()                         // Clean up background goroutine

	configJSON := []byte(`{"host": "localhost", "port": "8080", "timeout_ms": "5000", "enabled": "true"}`)

	// First parse - cache miss
	fmt.Println("   First parse (cache miss):")
	start := time.Now()
	config1, err := parser.Parse(configJSON)
	elapsed1 := time.Since(start)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Parsed: %+v (took %v)\n", config1, elapsed1)

	// Second parse - cache hit
	fmt.Println("   Second parse (cache hit - same data):")
	start = time.Now()
	config2, err := parser.Parse(configJSON)
	elapsed2 := time.Since(start)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Parsed: %+v (took %v)\n", config2, elapsed2)
	fmt.Printf("   Speedup: %.2fx faster\n", float64(elapsed1)/float64(elapsed2))

	// Display cache statistics
	size, maxSize, hitRate := parser.Stats()
	fmt.Printf("   Cache stats: Size=%d/%d, Hit Rate=%.1f%%\n",
		size, maxSize, hitRate*100)

	// Example 2: Custom cache configuration
	fmt.Println("\n2. Custom cache configuration:")
	customConfig := &model.CacheConfig{
		TTL:             5 * time.Minute,
		MaxEntries:      100,
		CleanupInterval: 1 * time.Minute,
	}
	customParser := model.NewCachedParser[APIResponse](customConfig)
	defer customParser.Close()

	fmt.Printf("   TTL: %v\n", customConfig.TTL)
	fmt.Printf("   MaxEntries: %d\n", customConfig.MaxEntries)
	fmt.Printf("   CleanupInterval: %v\n", customConfig.CleanupInterval)

	responseJSON := []byte(`{
		"id": 42,
		"message": "Success",
		"timestamp": "2025-01-15T10:30:00Z",
		"status": "OK"
	}`)

	response, err := customParser.Parse(responseJSON)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("   Parsed response: %+v\n", response)

	// Example 3: Demonstrating cache key sensitivity
	fmt.Println("\n3. Cache key sensitivity (different data = cache miss):")
	jsonV1 := []byte(`{"host": "localhost", "port": 8080}`)
	jsonV2 := []byte(`{"host": "localhost", "port": 8081}`) // Different port

	keyParser := model.NewCachedParser[Config](nil)
	defer keyParser.Close()

	_, _ = keyParser.Parse(jsonV1)
	fmt.Println("   Parsed version 1 (cache miss)")

	_, _ = keyParser.Parse(jsonV2)
	fmt.Println("   Parsed version 2 (cache miss - data differs by 1 character)")

	size, maxSize, hitRate = keyParser.Stats()
	fmt.Printf("   Cache stats: Size=%d/%d, Hit Rate=%.1f%% (both are misses)\n",
		size, maxSize, hitRate*100)

	// Example 4: Use case - Parsing static config repeatedly
	fmt.Println("\n4. Practical use case - Static config file parsed multiple times:")
	staticConfigJSON := []byte(`{
		"host": "api.example.com",
		"port": 443,
		"timeout_ms": 30000,
		"enabled": true
	}`)

	staticParser := model.NewCachedParser[Config](nil)
	defer staticParser.Close()

	// Simulate multiple reads of the same config
	for i := 1; i <= 5; i++ {
		_, err := staticParser.Parse(staticConfigJSON)
		if err != nil {
			log.Fatal(err)
		}
		_, _, hitRate := staticParser.Stats()
		fmt.Printf("   Read #%d: Hit Rate=%.1f%%\n", i, hitRate*100)
	}

	// Example 5: Clearing cache
	fmt.Println("\n5. Cache clearing:")
	clearParser := model.NewCachedParser[Config](nil)
	defer clearParser.Close()

	testJSON := []byte(`{"host": "test.com", "port": 80}`)

	_, _ = clearParser.Parse(testJSON)
	fmt.Println("   Parsed data (cache miss)")

	_, _ = clearParser.Parse(testJSON)
	fmt.Println("   Parsed again (cache hit)")

	size, _, hitRate = clearParser.Stats()
	fmt.Printf("   Before clear: Cache Size=%d, Hit Rate=%.1f%%\n", size, hitRate*100)

	clearParser.ClearCache()
	fmt.Println("   Cache cleared!")

	_, _ = clearParser.Parse(testJSON)
	fmt.Println("   Parsed after clear (cache miss)")

	size, _, hitRate = clearParser.Stats()
	fmt.Printf("   After clear: Cache Size=%d, Hit Rate=%.1f%%\n", size, hitRate*100)

	// Summary
	fmt.Println("\n================================")
	fmt.Println("Cache Demo Summary:")
	fmt.Println("- Use CachedParser for repeated parsing of identical data")
	fmt.Println("- Perfect for: config files, static API responses, retry scenarios")
	fmt.Println("- Limited benefit for: unique API requests with same structure")
	fmt.Println("- Always call defer parser.Close() to cleanup background goroutines")
	fmt.Println("- Cache keys use SHA256, so even 1 byte difference = cache miss")
	fmt.Println("================================")
}
