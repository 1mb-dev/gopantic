package tests

import (
	"strings"
	"testing"

	"github.com/1mb-dev/gopantic/pkg/model"
)

// TestMaxStructureDepth_Config verifies the configuration getter/setter
func TestMaxStructureDepth_Config(t *testing.T) {
	// Save original value
	orig := model.GetMaxStructureDepth()
	defer model.SetMaxStructureDepth(orig)

	t.Run("default value is 64", func(t *testing.T) {
		// Reset to default
		model.SetMaxStructureDepth(64)
		got := model.GetMaxStructureDepth()
		if got != 64 {
			t.Errorf("GetMaxStructureDepth() = %d, want 64", got)
		}
	})

	t.Run("setter updates value", func(t *testing.T) {
		model.SetMaxStructureDepth(32)
		got := model.GetMaxStructureDepth()
		if got != 32 {
			t.Errorf("GetMaxStructureDepth() = %d, want 32", got)
		}
	})
}

// TestStructureDepth_JSON verifies depth checking for JSON
func TestStructureDepth_JSON(t *testing.T) {
	// Save original value
	orig := model.GetMaxStructureDepth()
	defer model.SetMaxStructureDepth(orig)

	type Nested struct {
		Value string                 `json:"value"`
		Child map[string]interface{} `json:"child,omitempty"`
	}

	t.Run("shallow structure passes", func(t *testing.T) {
		model.SetMaxStructureDepth(10)
		data := []byte(`{"value": "test", "child": {"value": "nested"}}`)
		_, err := model.ParseInto[Nested](data)
		if err != nil {
			t.Errorf("shallow structure should pass: %v", err)
		}
	})

	t.Run("deeply nested structure fails", func(t *testing.T) {
		model.SetMaxStructureDepth(3)
		// Create a structure with depth 5: {a:{b:{c:{d:{e:"x"}}}}}
		data := []byte(`{"a":{"b":{"c":{"d":{"e":"x"}}}}}`)
		_, err := model.ParseInto[map[string]interface{}](data)
		if err == nil {
			t.Error("deeply nested structure should fail")
		}
		if !strings.Contains(err.Error(), "structure depth") {
			t.Errorf("error should mention structure depth: %v", err)
		}
	})

	t.Run("depth check disabled when set to 0", func(t *testing.T) {
		model.SetMaxStructureDepth(0)
		// Deep structure should pass when checking is disabled
		data := []byte(`{"a":{"b":{"c":{"d":{"e":{"f":{"g":"x"}}}}}}}`)
		_, err := model.ParseInto[map[string]interface{}](data)
		if err != nil {
			t.Errorf("depth check should be disabled: %v", err)
		}
	})

	t.Run("array depth counts", func(t *testing.T) {
		model.SetMaxStructureDepth(3)
		// Array with nested objects: [[[{"x":1}]]]
		data := []byte(`[[[[{"x":1}]]]]`)
		_, err := model.ParseInto[[][][][]map[string]int](data)
		if err == nil {
			t.Error("deeply nested array should fail")
		}
	})
}

// TestStructureDepth_YAML verifies depth checking for YAML
func TestStructureDepth_YAML(t *testing.T) {
	// Save original value
	orig := model.GetMaxStructureDepth()
	defer model.SetMaxStructureDepth(orig)

	t.Run("shallow YAML passes", func(t *testing.T) {
		model.SetMaxStructureDepth(10)
		data := []byte(`
level1:
  level2:
    value: test
`)
		_, err := model.ParseIntoWithFormat[map[string]interface{}](data, model.FormatYAML)
		if err != nil {
			t.Errorf("shallow YAML should pass: %v", err)
		}
	})

	t.Run("deeply nested YAML fails", func(t *testing.T) {
		model.SetMaxStructureDepth(3)
		data := []byte(`
a:
  b:
    c:
      d:
        e: value
`)
		_, err := model.ParseIntoWithFormat[map[string]interface{}](data, model.FormatYAML)
		if err == nil {
			t.Error("deeply nested YAML should fail")
		}
		if !strings.Contains(err.Error(), "structure depth") {
			t.Errorf("error should mention structure depth: %v", err)
		}
	})
}

// TestStructureDepth_EdgeCases tests edge cases
func TestStructureDepth_EdgeCases(t *testing.T) {
	orig := model.GetMaxStructureDepth()
	defer model.SetMaxStructureDepth(orig)

	t.Run("empty object passes", func(t *testing.T) {
		model.SetMaxStructureDepth(1)
		data := []byte(`{}`)
		_, err := model.ParseInto[map[string]interface{}](data)
		if err != nil {
			t.Errorf("empty object should pass: %v", err)
		}
	})

	t.Run("empty array passes", func(t *testing.T) {
		model.SetMaxStructureDepth(1)
		data := []byte(`[]`)
		_, err := model.ParseInto[[]interface{}](data)
		if err != nil {
			t.Errorf("empty array should pass: %v", err)
		}
	})

	t.Run("flat structure with many keys passes", func(t *testing.T) {
		model.SetMaxStructureDepth(2)
		data := []byte(`{"a":1,"b":2,"c":3,"d":4,"e":5}`)
		_, err := model.ParseInto[map[string]int](data)
		if err != nil {
			t.Errorf("flat structure should pass: %v", err)
		}
	})

	t.Run("exact depth limit passes", func(t *testing.T) {
		model.SetMaxStructureDepth(3)
		// Depth of 3: {a:{b:{c:1}}}
		data := []byte(`{"a":{"b":{"c":1}}}`)
		_, err := model.ParseInto[map[string]interface{}](data)
		if err != nil {
			t.Errorf("exact depth limit should pass: %v", err)
		}
	})

	t.Run("one over depth limit fails", func(t *testing.T) {
		model.SetMaxStructureDepth(3)
		// Depth of 4: {a:{b:{c:{d:1}}}}
		data := []byte(`{"a":{"b":{"c":{"d":1}}}}`)
		_, err := model.ParseInto[map[string]interface{}](data)
		if err == nil {
			t.Error("one over depth limit should fail")
		}
	})
}
