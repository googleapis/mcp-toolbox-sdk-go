//go:build unit

// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package core

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// Tests ParameterSchema with type 'int'.
func TestParameterSchemaInteger(t *testing.T) {

	schema := ParameterSchema{
		Name:        "param_name",
		Type:        "integer",
		Description: "integer parameter",
	}

	t.Run("Test int param", func(t *testing.T) {
		value := 1

		err := schema.validateType(value)

		if err != nil {
			t.Fatal(err.Error())
		}
	})
	t.Run("Test int8 param", func(t *testing.T) {
		var value int8 = 1

		err := schema.validateType(value)

		if err != nil {
			t.Fatal(err.Error())
		}
	})
	t.Run("Test int16 param", func(t *testing.T) {
		var value int16 = 1

		err := schema.validateType(value)

		if err != nil {
			t.Fatal(err.Error())
		}
	})
	t.Run("Test int32 param", func(t *testing.T) {
		var value int32 = 1

		err := schema.validateType(value)

		if err != nil {
			t.Fatal(err.Error())
		}
	})
	t.Run("Test int64 param", func(t *testing.T) {
		var value int64 = 1

		err := schema.validateType(value)

		if err != nil {
			t.Fatal(err.Error())
		}
	})
	t.Run("Test uint param", func(t *testing.T) {
		var value uint = 1

		err := schema.validateType(value)

		if err != nil {
			t.Fatal(err.Error())
		}
	})
	t.Run("Test uint8 param", func(t *testing.T) {
		var value uint8 = 1

		err := schema.validateType(value)

		if err != nil {
			t.Fatal(err.Error())
		}
	})
	t.Run("Test uint16 param", func(t *testing.T) {
		var value uint16 = 1

		err := schema.validateType(value)

		if err != nil {
			t.Fatal(err.Error())
		}
	})
	t.Run("Test uint32 param", func(t *testing.T) {
		var value uint32 = 1

		err := schema.validateType(value)

		if err != nil {
			t.Fatal(err.Error())
		}
	})
	t.Run("Test uint64 param", func(t *testing.T) {
		var value uint64 = 1

		err := schema.validateType(value)

		if err != nil {
			t.Fatal(err.Error())
		}
	})

}

// Tests ParameterSchema with type 'string'.
func TestParameterSchemaString(t *testing.T) {

	schema := ParameterSchema{
		Name:        "param_name",
		Type:        "string",
		Description: "string parameter",
	}

	value := "abc"

	err := schema.validateType(value)

	if err != nil {
		t.Fatal(err.Error())
	}

}

// Tests ParameterSchema with type 'boolean'.
func TestParameterSchemaBoolean(t *testing.T) {

	schema := ParameterSchema{
		Name:        "param_name",
		Type:        "boolean",
		Description: "boolean parameter",
	}

	value := true

	err := schema.validateType(value)

	if err != nil {
		t.Fatal(err.Error())
	}

}

// Tests ParameterSchema with type 'float'.
func TestParameterSchemaFloat(t *testing.T) {

	schema := ParameterSchema{
		Name:        "param_name",
		Type:        "float",
		Description: "float parameter",
	}

	t.Run("Test float32 param", func(t *testing.T) {
		var value float32 = 3.14

		err := schema.validateType(value)

		if err != nil {
			t.Fatal(err.Error())
		}
	})
	t.Run("Test float64 param", func(t *testing.T) {
		value := 3.14

		err := schema.validateType(value)

		if err != nil {
			t.Fatal(err.Error())
		}
	})

}

// Tests ParameterSchema with type 'array'.
func TestParameterSchemaStringArray(t *testing.T) {

	itemSchema := ParameterSchema{
		Name:        "item",
		Type:        "string",
		Description: "item of the array",
	}

	paramSchema := ParameterSchema{
		Name:        "param_name",
		Type:        "array",
		Description: "array parameter",
		Items:       &itemSchema,
	}

	value := []string{"abc", "def"}

	err := paramSchema.validateType(value)

	if err != nil {
		t.Fatal(err.Error())
	}

}

// Tests ParameterSchema with an undefined type.
func TestParameterSchemaUndefinedType(t *testing.T) {

	paramSchema := ParameterSchema{
		Name:        "param_name",
		Type:        "time",
		Description: "time parameter",
	}

	value := time.Now()

	err := paramSchema.validateType(value)

	if err == nil {
		t.Fatal("Expected an error, but got nil")
	}

}

func TestOptionalStringParameter(t *testing.T) {
	schema := ParameterSchema{
		Name:        "nickname",
		Type:        "string",
		Description: "An optional nickname",
		Required:    false, // Explicitly optional
	}

	t.Run("allows nil value for optional parameter", func(t *testing.T) {
		err := schema.validateType(nil)
		if err != nil {
			t.Errorf("validateType() with nil should not return an error for an optional parameter, but got: %v", err)
		}
	})

	t.Run("allows valid string value", func(t *testing.T) {
		err := schema.validateType("my-name")
		if err != nil {
			t.Errorf("validateType() should not return an error for a valid string, but got: %v", err)
		}
	})
}

func TestRequiredParameter(t *testing.T) {
	schema := ParameterSchema{
		Name:        "id",
		Type:        "integer",
		Description: "A required ID",
		Required:    true, // Explicitly required
	}

	t.Run("rejects nil value for required parameter", func(t *testing.T) {
		err := schema.validateType(nil)
		if err == nil {
			t.Errorf("validateType() with nil should return an error for a required parameter, but it didn't")
		}
	})

	t.Run("allows valid integer value", func(t *testing.T) {
		err := schema.validateType(12345)
		if err != nil {
			t.Errorf("validateType() should not return an error for a valid integer, but got: %v", err)
		}
	})
}

func TestOptionalArrayParameter(t *testing.T) {
	schema := ParameterSchema{
		Name:        "optional_scores",
		Type:        "array",
		Description: "An optional list of scores",
		Required:    false,
		Items: &ParameterSchema{
			Type: "integer",
		},
	}

	t.Run("allows nil value for optional array", func(t *testing.T) {
		err := schema.validateType(nil)
		if err != nil {
			t.Errorf("validateType() with nil should not return an error for an optional array, but got: %v", err)
		}
	})

	t.Run("allows valid integer slice", func(t *testing.T) {
		err := schema.validateType([]int{95, 100})
		if err != nil {
			t.Errorf("validateType() should not return an error for a valid slice, but got: %v", err)
		}
	})

	t.Run("rejects slice with wrong item type", func(t *testing.T) {
		err := schema.validateType([]string{"not", "an", "int"})
		if err == nil {
			t.Errorf("validateType() should have returned an error for a slice with incorrect item types, but it didn't")
		}
	})
}

func TestValidateTypeObject(t *testing.T) {
	t.Run("generic object allows any value types", func(t *testing.T) {
		schema := ParameterSchema{
			Name:                 "metadata",
			Type:                 "object",
			AdditionalProperties: true, // or nil
		}

		// A map with mixed value types should be valid.
		validInput := map[string]any{
			"key_string": "a string",
			"key_int":    123,
			"key_bool":   true,
		}
		if err := schema.validateType(validInput); err != nil {
			t.Errorf("Expected no error for generic object, but got: %v", err)
		}

		// A value that is not a map should be invalid.
		invalidInput := "I am a string, not an object"
		if err := schema.validateType(invalidInput); err == nil {
			t.Errorf("Expected an error for non-map input, but got nil")
		}
		
		// Nested maps are invalid
		nestedMapInput := map[string]any {
			"key_map": map[string]any{"id": 1}
		}
		if err := schema.validateType(invalidInput); err == nil {
			t.Errorf("Expected an error for nested map input, but got nil")
		}

		// Nested arrays in maps are invalid
		nestedArrayInput := map[string]any {
			"key_map": []string{"id", "number"}
		}
		if err := schema.validateType(invalidInput); err == nil {
			t.Errorf("Expected an error for array in map input, but got nil")
		}
	})

	t.Run("typed object validation", func(t *testing.T) {
		testCases := []struct {
			name         string
			valueType    string
			validInput   map[string]any
			invalidInput map[string]any
		}{
			{
				name:         "string values",
				valueType:    "string",
				validInput:   map[string]any{"header": "application/json"},
				invalidInput: map[string]any{"bad_header": 123},
			},
			{
				name:         "integer values",
				valueType:    "integer",
				validInput:   map[string]any{"user_score": 100},
				invalidInput: map[string]any{"bad_score": "100"},
			},
			{
				name:         "float values",
				valueType:    "float",
				validInput:   map[string]any{"item_price": 99.99},
				invalidInput: map[string]any{"bad_price": 99},
			},
			{
				name:         "boolean values",
				valueType:    "boolean",
				validInput:   map[string]any{"feature_flag": true},
				invalidInput: map[string]any{"bad_flag": "true"},
			},
			{
				name:         "map or object values",
				valueType:    "object",
				validInput:   map[string]any{"feature_flag": true},
				invalidInput: map[string]any{"feature_flag": map[string]any{"id": "123"}},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				schema := ParameterSchema{
					Name:                 "test_map",
					Type:                 "object",
					AdditionalProperties: &ParameterSchema{Type: tc.valueType},
				}

				// Test that valid input passes
				if err := schema.validateType(tc.validInput); err != nil {
					t.Errorf("Expected no error for valid input, got: %v", err)
				}

				// Test that invalid input fails
				if err := schema.validateType(tc.invalidInput); err == nil {
					t.Errorf("Expected an error for invalid input, but got nil")
				}
			})
		}
	})

	t.Run("Fail for object valueType maps", func(t *testing.T) {

		// This schema itself is invalid so there is no valid test case
		schema := ParameterSchema{
			Name:                 "test_map",
			Type:                 "object",
			AdditionalProperties: &ParameterSchema{Type: "object"},
		}

		invalidInput := map[string]any{"feature_flag": map[string]any{"id": "123"}}
		// Test that invalid input fails
		if err := schema.validateType(invalidInput); err == nil {
			t.Errorf("Expected an error for invalid input, but got nil")
		}
	})

	t.Run("optional and required objects", func(t *testing.T) {
		// An optional object can be nil
		optionalSchema := ParameterSchema{Name: "optional_metadata", Type: "object", Required: false}
		if err := optionalSchema.validateType(nil); err != nil {
			t.Errorf("Expected no error for nil on optional object, but got: %v", err)
		}

		// A required object cannot be nil
		requiredSchema := ParameterSchema{Name: "required_metadata", Type: "object", Required: true}
		if err := requiredSchema.validateType(nil); err == nil {
			t.Error("Expected an error for nil on required object, but got nil")
		}
	})

	t.Run("object with unsupported value type in schema", func(t *testing.T) {
		unsupportedType := "custom_object"
		schema := ParameterSchema{
			Name:                 "custom_data",
			Type:                 "object",
			AdditionalProperties: &ParameterSchema{Type: unsupportedType},
		}

		input := map[string]any{"key": "some value"}
		err := schema.validateType(input)

		if err == nil {
			t.Fatal("Expected an error for unsupported sub-schema type, but got nil")
		}

		// Check if the error message contains the expected text.
		expectedErrorMsg := fmt.Sprintf("unknown type '%s'", unsupportedType)
		if !strings.Contains(err.Error(), expectedErrorMsg) {
			t.Errorf("Expected error to contain '%s', but got '%v'", expectedErrorMsg, err)
		}
	})
}

func TestParameterSchema_ValidateDefinition(t *testing.T) {
	t.Run("should succeed for simple valid types", func(t *testing.T) {
		testCases := []struct {
			name   string
			schema *ParameterSchema
		}{
			{"String", &ParameterSchema{Name: "p_string", Type: "string"}},
			{"Integer", &ParameterSchema{Name: "p_int", Type: "integer"}},
			{"Float", &ParameterSchema{Name: "p_float", Type: "float"}},
			{"Boolean", &ParameterSchema{Name: "p_bool", Type: "boolean"}},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				if err := tc.schema.ValidateDefinition(); err != nil {
					t.Errorf("expected no error, but got: %v", err)
				}
			})
		}
	})

	t.Run("should succeed for a valid array schema", func(t *testing.T) {
		schema := &ParameterSchema{
			Name:  "p_array",
			Type:  "array",
			Items: &ParameterSchema{Type: "string"},
		}
		if err := schema.ValidateDefinition(); err != nil {
			t.Errorf("expected no error for valid array, but got: %v", err)
		}
	})

	t.Run("should succeed for valid object schemas", func(t *testing.T) {
		testCases := []struct {
			name   string
			schema *ParameterSchema
		}{
			{
				"Typed Object",
				&ParameterSchema{
					Name:                 "p_obj_typed",
					Type:                 "object",
					AdditionalProperties: &ParameterSchema{Type: "integer"},
				},
			},
			{
				"Generic Object (bool)",
				&ParameterSchema{
					Name:                 "p_obj_bool",
					Type:                 "object",
					AdditionalProperties: true,
				},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				if err := tc.schema.ValidateDefinition(); err != nil {
					t.Errorf("expected no error, but got: %v", err)
				}
			})
		}
	})

	t.Run("should fail when type is missing", func(t *testing.T) {
		schema := &ParameterSchema{Name: "p_missing_type", Type: "object", AdditionalProperties: &ParameterSchema{Type: ""}}
		err := schema.ValidateDefinition()
		if err == nil {
			t.Fatal("expected an error for missing type, but got nil")
		}
		if !strings.Contains(err.Error(), "type is missing") {
			t.Errorf("error message should mention 'type is missing', but was: %s", err)
		}
	})

	t.Run("should fail when type is unknown", func(t *testing.T) {
		schema := &ParameterSchema{Name: "p_unknown", Type: "object", AdditionalProperties: &ParameterSchema{Type: "some-custom-type"}}
		err := schema.ValidateDefinition()
		if err == nil {
			t.Fatal("expected an error for unknown type, but got nil")
		}
		if !strings.Contains(err.Error(), "unknown schema type") {
			t.Errorf("error message should mention 'unknown schema type', but was: %s", err)
		}
	})

	t.Run("should fail for array with missing items property", func(t *testing.T) {
		schema := &ParameterSchema{Name: "p_bad_array", Type: "array", Items: nil}
		err := schema.ValidateDefinition()
		if err == nil {
			t.Fatal("expected an error for array with nil items, but got nil")
		}
		if !strings.Contains(err.Error(), "missing item type definition") {
			t.Errorf("error message should mention 'missing item type definition', but was: %s", err)
		}
	})

	t.Run("should fail for object with invalid AdditionalProperties type", func(t *testing.T) {
		schema := &ParameterSchema{
			Name:                 "p_bad_object",
			Type:                 "object",
			AdditionalProperties: "a-string-is-not-valid",
		}
		err := schema.ValidateDefinition()
		if err == nil {
			t.Fatal("expected an error for invalid AdditionalProperties, but got nil")
		}
		if !strings.Contains(err.Error(), "must be a boolean or a schema") {
			t.Errorf("error message should mention 'must be a boolean or a schema', but was: %s", err)
		}
	})
}
