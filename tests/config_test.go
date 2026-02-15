package tests

import (
	"sync"
	"testing"

	"github.com/1mb-dev/gopantic/pkg/model"
)

// TestConfigGettersReturnDefaults verifies Get functions return expected defaults
func TestConfigGettersReturnDefaults(t *testing.T) {
	t.Run("GetMaxInputSize returns default", func(t *testing.T) {
		// Default is 10MB
		expected := 10 * 1024 * 1024
		got := model.GetMaxInputSize()
		if got != expected {
			t.Errorf("GetMaxInputSize() = %d, want %d", got, expected)
		}
	})

	t.Run("GetMaxCacheSize returns default", func(t *testing.T) {
		expected := 1000
		got := model.GetMaxCacheSize()
		if got != expected {
			t.Errorf("GetMaxCacheSize() = %d, want %d", got, expected)
		}
	})

	t.Run("GetMaxValidationDepth returns default", func(t *testing.T) {
		expected := 32
		got := model.GetMaxValidationDepth()
		if got != expected {
			t.Errorf("GetMaxValidationDepth() = %d, want %d", got, expected)
		}
	})
}

// TestConfigSettersUpdateValues verifies Set functions update values
func TestConfigSettersUpdateValues(t *testing.T) {
	// Save original values
	origInputSize := model.GetMaxInputSize()
	origCacheSize := model.GetMaxCacheSize()
	origValidationDepth := model.GetMaxValidationDepth()

	// Restore after test
	defer func() {
		model.SetMaxInputSize(origInputSize)
		model.SetMaxCacheSize(origCacheSize)
		model.SetMaxValidationDepth(origValidationDepth)
	}()

	t.Run("SetMaxInputSize updates value", func(t *testing.T) {
		newValue := 5 * 1024 * 1024 // 5MB
		model.SetMaxInputSize(newValue)
		got := model.GetMaxInputSize()
		if got != newValue {
			t.Errorf("After SetMaxInputSize(%d), GetMaxInputSize() = %d", newValue, got)
		}
	})

	t.Run("SetMaxCacheSize updates value", func(t *testing.T) {
		newValue := 500
		model.SetMaxCacheSize(newValue)
		got := model.GetMaxCacheSize()
		if got != newValue {
			t.Errorf("After SetMaxCacheSize(%d), GetMaxCacheSize() = %d", newValue, got)
		}
	})

	t.Run("SetMaxValidationDepth updates value", func(t *testing.T) {
		newValue := 64
		model.SetMaxValidationDepth(newValue)
		got := model.GetMaxValidationDepth()
		if got != newValue {
			t.Errorf("After SetMaxValidationDepth(%d), GetMaxValidationDepth() = %d", newValue, got)
		}
	})
}

// TestConfigConcurrentAccess verifies thread-safety of Get/Set functions
func TestConfigConcurrentAccess(t *testing.T) {
	// Save original value
	origInputSize := model.GetMaxInputSize()
	defer model.SetMaxInputSize(origInputSize)

	const goroutines = 100
	const iterations = 100

	var wg sync.WaitGroup
	wg.Add(goroutines * 2) // Half readers, half writers

	// Start concurrent readers
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				_ = model.GetMaxInputSize()
			}
		}()
	}

	// Start concurrent writers
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				model.SetMaxInputSize(id*1000 + j)
			}
		}(i)
	}

	wg.Wait()
	// If we get here without deadlock or panic, the test passes
}
