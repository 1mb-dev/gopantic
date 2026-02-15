package tests

import (
	"sync"
	"testing"

	"github.com/1mb-dev/gopantic/pkg/model"
)

// TestConcurrency_ParallelParsing tests that ParseInto is safe for concurrent use
func TestConcurrency_ParallelParsing(t *testing.T) {
	userData := `{
		"id": 123,
		"name": "ConcurrentUser",
		"email": "concurrent@example.com",
		"age": 25
	}`

	const goroutines = 100
	var wg sync.WaitGroup
	errors := make(chan error, goroutines)

	// Launch multiple goroutines parsing simultaneously
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			user, err := model.ParseInto[User]([]byte(userData))
			if err != nil {
				errors <- err
				return
			}
			if user.Name != "ConcurrentUser" {
				t.Errorf("Expected name 'ConcurrentUser', got '%s'", user.Name)
			}
		}()
	}

	wg.Wait()
	close(errors)

	// Check for any errors
	for err := range errors {
		t.Errorf("Concurrent parsing error: %v", err)
	}
}

// TestConcurrency_ParallelValidation tests that Validate is safe for concurrent use
func TestConcurrency_ParallelValidation(t *testing.T) {
	user := User{
		ID:    42,
		Name:  "ValidUser",
		Email: "valid@example.com",
		Age:   30,
	}

	const goroutines = 100
	var wg sync.WaitGroup
	errors := make(chan error, goroutines)

	// Launch multiple goroutines validating simultaneously
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := model.Validate(&user)
			if err != nil {
				errors <- err
			}
		}()
	}

	wg.Wait()
	close(errors)

	// Check for any errors
	for err := range errors {
		t.Errorf("Concurrent validation error: %v", err)
	}
}

// TestConcurrency_CachedParser tests that CachedParser is safe for concurrent use
func TestConcurrency_CachedParser(t *testing.T) {
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

	const goroutines = 100
	var wg sync.WaitGroup
	errors := make(chan error, goroutines)
	results := make(chan string, goroutines)

	// Launch multiple goroutines using cached parser simultaneously
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			user, err := parser.Parse([]byte(userData))
			if err != nil {
				errors <- err
				return
			}
			results <- user.Username
		}()
	}

	wg.Wait()
	close(errors)
	close(results)

	// Check for any errors
	for err := range errors {
		t.Errorf("Concurrent cached parsing error: %v", err)
	}

	// Verify all results are consistent
	for username := range results {
		if username != "cacheduser" {
			t.Errorf("Expected username 'cacheduser', got '%s'", username)
		}
	}
}

// TestConcurrency_MixedOperations tests concurrent mixed operations (parse + validate)
func TestConcurrency_MixedOperations(t *testing.T) {
	validData := `{"id":100, "name":"MixedUser", "email":"mixed@example.com", "age":28}`
	invalidData := `{"id":-1, "name":"", "email":"invalid", "age":999}`

	const goroutines = 50 // 50 valid + 50 invalid = 100 total
	var wg sync.WaitGroup

	// Launch goroutines with valid data
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			user, err := model.ParseInto[User]([]byte(validData))
			if err != nil {
				t.Errorf("Valid data should parse without error: %v", err)
				return
			}
			if user.Name != "MixedUser" {
				t.Errorf("Expected name 'MixedUser', got '%s'", user.Name)
			}
		}()
	}

	// Launch goroutines with invalid data
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := model.ParseInto[User]([]byte(invalidData))
			// We expect this to succeed parsing (no validation on User struct)
			// This tests that concurrent parsing doesn't interfere
			if err != nil {
				// Parsing itself should work even if data is logically invalid
				t.Logf("Parsing error (acceptable): %v", err)
			}
		}()
	}

	wg.Wait()
}
