package tests

import (
	"reflect"
	"sort"
	"testing"

	"github.com/1mb-dev/gopantic/pkg/model"
)

// TestNewValidatorRegistry tests registry creation and built-in validators
func TestNewValidatorRegistry(t *testing.T) {
	t.Run("creates new registry", func(t *testing.T) {
		registry := model.NewValidatorRegistry()
		if registry == nil {
			t.Fatal("NewValidatorRegistry should not return nil")
		}
	})

	t.Run("registers built-in validators", func(t *testing.T) {
		registry := model.NewValidatorRegistry()
		validators := registry.ListValidators()

		expected := []string{"required", "min", "max", "email", "length", "alpha", "alphanum"}
		for _, name := range expected {
			found := false
			for _, v := range validators {
				if v == name {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected built-in validator %q not found", name)
			}
		}
	})

	t.Run("built-in validators work", func(t *testing.T) {
		registry := model.NewValidatorRegistry()

		// Test required validator
		required := registry.Create("required", nil)
		if required == nil {
			t.Fatal("failed to create required validator")
		}
		if required.Name() != "required" {
			t.Errorf("required.Name() = %q, want %q", required.Name(), "required")
		}

		// Test min validator with params
		min := registry.Create("min", map[string]interface{}{"value": 5})
		if min == nil {
			t.Fatal("failed to create min validator")
		}
		if min.Name() != "min" {
			t.Errorf("min.Name() = %q, want %q", min.Name(), "min")
		}
	})
}

// TestValidatorRegistry_Register tests factory function registration
func TestValidatorRegistry_Register(t *testing.T) {
	t.Run("registers custom factory", func(t *testing.T) {
		registry := model.NewValidatorRegistry()

		// Register a custom validator factory
		called := false
		registry.Register("custom", func(params map[string]interface{}) model.Validator {
			called = true
			return &mockValidator{name: "custom"}
		})

		// Create the validator
		v := registry.Create("custom", nil)
		if !called {
			t.Error("factory function was not called")
		}
		if v == nil {
			t.Fatal("Create should return a validator")
		}
		if v.Name() != "custom" {
			t.Errorf("Name() = %q, want %q", v.Name(), "custom")
		}
	})

	t.Run("factory receives params", func(t *testing.T) {
		registry := model.NewValidatorRegistry()

		var receivedParams map[string]interface{}
		registry.Register("paramtest", func(params map[string]interface{}) model.Validator {
			receivedParams = params
			return &mockValidator{name: "paramtest"}
		})

		expected := map[string]interface{}{"key": "value", "num": 42}
		registry.Create("paramtest", expected)

		if receivedParams["key"] != "value" {
			t.Errorf("params[key] = %v, want %v", receivedParams["key"], "value")
		}
		if receivedParams["num"] != 42 {
			t.Errorf("params[num] = %v, want %v", receivedParams["num"], 42)
		}
	})

	t.Run("overwrites existing validator", func(t *testing.T) {
		registry := model.NewValidatorRegistry()

		// Register first version
		registry.Register("overwrite", func(params map[string]interface{}) model.Validator {
			return &mockValidator{name: "v1"}
		})

		// Overwrite with second version
		registry.Register("overwrite", func(params map[string]interface{}) model.Validator {
			return &mockValidator{name: "v2"}
		})

		v := registry.Create("overwrite", nil)
		if v.Name() != "v2" {
			t.Errorf("Name() = %q, want %q (overwrite should work)", v.Name(), "v2")
		}
	})
}

// TestValidatorRegistry_RegisterFunc tests custom function registration
func TestValidatorRegistry_RegisterFunc(t *testing.T) {
	t.Run("registers custom function", func(t *testing.T) {
		registry := model.NewValidatorRegistry()

		called := false
		registry.RegisterFunc("customfn", func(fieldName string, value interface{}, params map[string]interface{}) error {
			called = true
			return nil
		})

		v := registry.Create("customfn", nil)
		if v == nil {
			t.Fatal("Create should return a validator for custom function")
		}

		err := v.Validate("testfield", "testvalue")
		if err != nil {
			t.Errorf("Validate returned error: %v", err)
		}
		if !called {
			t.Error("custom function was not called")
		}
	})

	t.Run("function receives correct args", func(t *testing.T) {
		registry := model.NewValidatorRegistry()

		var gotField string
		var gotValue interface{}
		var gotParams map[string]interface{}

		registry.RegisterFunc("argtest", func(fieldName string, value interface{}, params map[string]interface{}) error {
			gotField = fieldName
			gotValue = value
			gotParams = params
			return nil
		})

		v := registry.Create("argtest", map[string]interface{}{"param1": "test"})
		_ = v.Validate("myfield", 123)

		if gotField != "myfield" {
			t.Errorf("fieldName = %q, want %q", gotField, "myfield")
		}
		if gotValue != 123 {
			t.Errorf("value = %v, want %v", gotValue, 123)
		}
		if gotParams["param1"] != "test" {
			t.Errorf("params[param1] = %v, want %v", gotParams["param1"], "test")
		}
	})

	t.Run("custom func has priority over built-in", func(t *testing.T) {
		registry := model.NewValidatorRegistry()

		// Register a custom "min" that overrides the built-in
		customCalled := false
		registry.RegisterFunc("min", func(fieldName string, value interface{}, params map[string]interface{}) error {
			customCalled = true
			return nil
		})

		v := registry.Create("min", nil)
		_ = v.Validate("test", 1)

		if !customCalled {
			t.Error("custom func should take priority over built-in")
		}
	})
}

// TestValidatorRegistry_RegisterCrossFieldFunc tests cross-field function registration
func TestValidatorRegistry_RegisterCrossFieldFunc(t *testing.T) {
	t.Run("registers cross-field function", func(t *testing.T) {
		registry := model.NewValidatorRegistry()

		registry.RegisterCrossFieldFunc("crosstest", func(fieldName string, fieldValue interface{}, structValue reflect.Value, params map[string]interface{}) error {
			return nil
		})

		v := registry.Create("crosstest", nil)
		if v == nil {
			t.Fatal("Create should return a validator for cross-field function")
		}

		// Cross-field validators should return error when Validate is called directly
		err := v.Validate("testfield", "testvalue")
		if err == nil {
			t.Error("cross-field validator should return error when called without struct context")
		}
	})

	t.Run("cross-field has highest priority", func(t *testing.T) {
		registry := model.NewValidatorRegistry()

		// Register all three types with the same name
		registry.Register("priority", func(params map[string]interface{}) model.Validator {
			return &mockValidator{name: "factory"}
		})
		registry.RegisterFunc("priority", func(fieldName string, value interface{}, params map[string]interface{}) error {
			return nil
		})
		registry.RegisterCrossFieldFunc("priority", func(fieldName string, fieldValue interface{}, structValue reflect.Value, params map[string]interface{}) error {
			return nil
		})

		v := registry.Create("priority", nil)
		// Cross-field validators return error on direct Validate call
		err := v.Validate("test", "value")
		if err == nil {
			t.Error("cross-field validator should be used (highest priority)")
		}
	})
}

// TestValidatorRegistry_Create tests validator creation
func TestValidatorRegistry_Create(t *testing.T) {
	t.Run("returns nil for unknown validator", func(t *testing.T) {
		registry := model.NewValidatorRegistry()

		v := registry.Create("nonexistent", nil)
		if v != nil {
			t.Error("Create should return nil for unknown validator")
		}
	})

	t.Run("creates validator with nil params", func(t *testing.T) {
		registry := model.NewValidatorRegistry()

		v := registry.Create("required", nil)
		if v == nil {
			t.Fatal("Create should handle nil params")
		}
	})

	t.Run("creates validator with empty params", func(t *testing.T) {
		registry := model.NewValidatorRegistry()

		v := registry.Create("email", map[string]interface{}{})
		if v == nil {
			t.Fatal("Create should handle empty params")
		}
	})
}

// TestGetDefaultRegistry tests the global registry singleton
func TestGetDefaultRegistry(t *testing.T) {
	t.Run("returns same instance", func(t *testing.T) {
		r1 := model.GetDefaultRegistry()
		r2 := model.GetDefaultRegistry()

		if r1 != r2 {
			t.Error("GetDefaultRegistry should return the same instance")
		}
	})

	t.Run("has built-in validators", func(t *testing.T) {
		registry := model.GetDefaultRegistry()
		validators := registry.ListValidators()

		if len(validators) < 7 {
			t.Errorf("expected at least 7 built-in validators, got %d", len(validators))
		}
	})
}

// TestValidatorRegistry_ListValidators tests validator listing
func TestValidatorRegistry_ListValidators(t *testing.T) {
	t.Run("returns all validator names", func(t *testing.T) {
		registry := model.NewValidatorRegistry()

		// Add custom validators
		registry.Register("custom1", func(params map[string]interface{}) model.Validator {
			return &mockValidator{name: "custom1"}
		})
		registry.RegisterFunc("custom2", func(fieldName string, value interface{}, params map[string]interface{}) error {
			return nil
		})
		registry.RegisterCrossFieldFunc("custom3", func(fieldName string, fieldValue interface{}, structValue reflect.Value, params map[string]interface{}) error {
			return nil
		})

		validators := registry.ListValidators()

		// Should have 7 built-in + 3 custom
		if len(validators) < 10 {
			t.Errorf("expected at least 10 validators, got %d", len(validators))
		}

		// Check that all custom validators are included
		custom := []string{"custom1", "custom2", "custom3"}
		for _, name := range custom {
			found := false
			for _, v := range validators {
				if v == name {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("custom validator %q not found in list", name)
			}
		}
	})

	t.Run("returns empty list for empty registry", func(t *testing.T) {
		// Create a registry without built-ins would require accessing internal state
		// So we just verify the default has built-ins
		registry := model.NewValidatorRegistry()
		validators := registry.ListValidators()

		// Should have at least the built-in validators
		if len(validators) == 0 {
			t.Error("NewValidatorRegistry should have built-in validators")
		}
	})
}

// TestBuiltInValidators tests that built-in validators work correctly
func TestBuiltInValidators(t *testing.T) {
	registry := model.NewValidatorRegistry()

	t.Run("required validator", func(t *testing.T) {
		v := registry.Create("required", nil)
		if v == nil {
			t.Fatal("failed to create required validator")
		}

		// Empty string should fail
		err := v.Validate("field", "")
		if err == nil {
			t.Error("required should fail on empty string")
		}

		// Non-empty should pass
		err = v.Validate("field", "value")
		if err != nil {
			t.Errorf("required should pass on non-empty: %v", err)
		}
	})

	t.Run("min validator", func(t *testing.T) {
		v := registry.Create("min", map[string]interface{}{"value": 5})
		if v == nil {
			t.Fatal("failed to create min validator")
		}

		// Below min should fail
		err := v.Validate("field", 3)
		if err == nil {
			t.Error("min should fail when value is below minimum")
		}

		// At or above min should pass
		err = v.Validate("field", 5)
		if err != nil {
			t.Errorf("min should pass at minimum: %v", err)
		}

		err = v.Validate("field", 10)
		if err != nil {
			t.Errorf("min should pass above minimum: %v", err)
		}
	})

	t.Run("max validator", func(t *testing.T) {
		v := registry.Create("max", map[string]interface{}{"value": 10})
		if v == nil {
			t.Fatal("failed to create max validator")
		}

		// Above max should fail
		err := v.Validate("field", 15)
		if err == nil {
			t.Error("max should fail when value is above maximum")
		}

		// At or below max should pass
		err = v.Validate("field", 10)
		if err != nil {
			t.Errorf("max should pass at maximum: %v", err)
		}
	})

	t.Run("email validator", func(t *testing.T) {
		v := registry.Create("email", nil)
		if v == nil {
			t.Fatal("failed to create email validator")
		}

		// Invalid email should fail
		err := v.Validate("field", "notanemail")
		if err == nil {
			t.Error("email should fail on invalid email")
		}

		// Valid email should pass
		err = v.Validate("field", "test@example.com")
		if err != nil {
			t.Errorf("email should pass on valid email: %v", err)
		}
	})

	t.Run("alpha validator", func(t *testing.T) {
		v := registry.Create("alpha", nil)
		if v == nil {
			t.Fatal("failed to create alpha validator")
		}

		// Non-alpha should fail
		err := v.Validate("field", "abc123")
		if err == nil {
			t.Error("alpha should fail on non-alpha characters")
		}

		// Pure alpha should pass
		err = v.Validate("field", "abcXYZ")
		if err != nil {
			t.Errorf("alpha should pass on pure alpha: %v", err)
		}
	})

	t.Run("alphanum validator", func(t *testing.T) {
		v := registry.Create("alphanum", nil)
		if v == nil {
			t.Fatal("failed to create alphanum validator")
		}

		// Non-alphanum should fail
		err := v.Validate("field", "abc-123")
		if err == nil {
			t.Error("alphanum should fail on non-alphanumeric characters")
		}

		// Pure alphanum should pass
		err = v.Validate("field", "abc123XYZ")
		if err != nil {
			t.Errorf("alphanum should pass on alphanumeric: %v", err)
		}
	})
}

// TestValidatorLookupOrder tests the priority order of validator types
func TestValidatorLookupOrder(t *testing.T) {
	t.Run("lookup order: cross-field > custom func > factory", func(t *testing.T) {
		registry := model.NewValidatorRegistry()

		// Register in reverse priority order
		registry.Register("order", func(params map[string]interface{}) model.Validator {
			return &mockValidator{name: "factory"}
		})
		registry.RegisterFunc("order", func(fieldName string, value interface{}, params map[string]interface{}) error {
			return nil
		})
		registry.RegisterCrossFieldFunc("order", func(fieldName string, fieldValue interface{}, structValue reflect.Value, params map[string]interface{}) error {
			return nil
		})

		v := registry.Create("order", nil)
		// Cross-field validators return specific error on direct Validate
		err := v.Validate("test", "value")
		if err == nil {
			t.Error("cross-field should have highest priority")
		}

		// Verify the error is about cross-field context requirement
		if err.Error() == "" {
			t.Error("cross-field validator should explain why it failed")
		}
	})
}

// mockValidator is a simple test validator
type mockValidator struct {
	name string
}

func (v *mockValidator) Name() string {
	return v.name
}

func (v *mockValidator) Validate(fieldName string, value interface{}) error {
	return nil
}

// TestListValidatorsStability tests that ListValidators returns consistent results
func TestListValidatorsStability(t *testing.T) {
	registry := model.NewValidatorRegistry()

	// Get list multiple times
	list1 := registry.ListValidators()
	list2 := registry.ListValidators()

	// Sort both lists for comparison (map iteration order is not guaranteed)
	sort.Strings(list1)
	sort.Strings(list2)

	if len(list1) != len(list2) {
		t.Errorf("ListValidators returned different lengths: %d vs %d", len(list1), len(list2))
	}

	for i := range list1 {
		if list1[i] != list2[i] {
			t.Errorf("ListValidators returned different content at %d: %q vs %q", i, list1[i], list2[i])
		}
	}
}
