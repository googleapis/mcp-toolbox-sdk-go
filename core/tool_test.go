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
	"reflect"
	"strings"
	"testing"

	"golang.org/x/oauth2"
)

func TestToolboxTool_Getters(t *testing.T) {
	sampleParams := []ParameterSchema{
		{Name: "param_one", Type: "string"},
		{Name: "param_two", Type: "integer"},
	}

	tool := &ToolboxTool{
		name:        "my-test-tool",
		description: "A tool specifically for testing purposes.",
		parameters:  sampleParams,
	}

	t.Run("Name Method Returns Correct Value", func(t *testing.T) {
		expected := "my-test-tool"
		if got := tool.Name(); got != expected {
			t.Fatalf("Expected Name() to be '%s', but got '%s'", expected, got)
		}
	})

	t.Run("Description Method Returns Correct Value", func(t *testing.T) {
		expected := "A tool specifically for testing purposes."
		if got := tool.Description(); got != expected {
			t.Fatalf("Expected Description() to be '%s', but got '%s'", expected, got)
		}
	})

	t.Run("Parameters Method Behavior", func(t *testing.T) {
		t.Run("Returns Correct Slice Content", func(t *testing.T) {
			params := tool.Parameters()
			if !reflect.DeepEqual(params, sampleParams) {
				t.Fatalf("Parameters() returned incorrect slice.\nExpected: %+v\nGot: %+v", sampleParams, params)
			}
		})

		t.Run("Returns A Safe Copy, Not a Reference", func(t *testing.T) {
			paramsFromMethod := tool.Parameters()

			paramsFromMethod[0].Name = "MODIFIED"

			internalParams := tool.parameters
			if internalParams[0].Name == "MODIFIED" {
				t.Fatalf("Parameters() returned a direct reference to the internal slice, not a copy. Modifying the returned slice dangerously changed the tool's internal state.")
			}
		})

		t.Run("Handles Case With No Parameters", func(t *testing.T) {
			emptyTool := &ToolboxTool{
				parameters: []ParameterSchema{},
			}

			params := emptyTool.Parameters()

			if params == nil {
				t.Fatalf("Parameters() should return a non-nil, empty slice for a tool with no parameters, but got nil.")
			}
			if len(params) != 0 {
				t.Fatalf("Expected an empty slice from Parameters(), but got a slice of length %d", len(params))
			}
		})
	})
}

func TestDescribeParameters(t *testing.T) {
	testCases := []struct {
		name     string
		tool     *ToolboxTool
		expected string
	}{
		{
			name:     "Tool with no parameters",
			tool:     &ToolboxTool{parameters: []ParameterSchema{}},
			expected: "",
		},
		{
			name: "Tool with one parameter",
			tool: &ToolboxTool{
				parameters: []ParameterSchema{
					{Name: "city", Type: "string"},
				},
			},
			expected: "'city' (type: string, description: )",
		},
		{
			name: "Tool with multiple parameters",
			tool: &ToolboxTool{
				parameters: []ParameterSchema{
					{Name: "location", Type: "string"},
					{Name: "days", Type: "integer"},
					{Name: "include_extended", Type: "boolean"},
				},
			},
			expected: "'location' (type: string, description: ), 'days' (type: integer, description: ), 'include_extended' (type: boolean, description: )",
		},
		{
			name: "Tool with empty parameter slice",
			tool: &ToolboxTool{
				parameters: nil,
			},
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Action
			result := tc.tool.DescribeParameters()

			// Assert
			if result != tc.expected {
				t.Errorf("expected %q, but got %q", tc.expected, result)
			}
		})
	}
}

func TestToolFrom(t *testing.T) {
	// Base tool used for creating test instances.
	baseTool := &ToolboxTool{
		name:        "weather",
		description: "gets the weather",
		parameters: []ParameterSchema{
			{Name: "city", Type: "string"},
			{Name: "days", Type: "integer"},
		},
		boundParams: map[string]any{
			"units": "celsius", // Parameter already bound on the parent
		},
		authTokenSources: map[string]oauth2.TokenSource{
			"google": &mockTokenSource{}, // Auth source already set on parent
		},
	}

	getTestTool := func() *ToolboxTool {
		return baseTool.cloneToolboxTool()
	}

	t.Run("Binding a new parameter - Success", func(t *testing.T) {
		tool := getTestTool()
		newTool, err := tool.ToolFrom(WithBindParamString("city", "London"))
		if err != nil {
			t.Fatalf("ToolFrom failed unexpectedly: %v", err)
		}
		if val, ok := newTool.boundParams["city"]; !ok || val != "London" {
			t.Errorf("Expected 'city' to be bound to 'London', but it was not")
		}
		if len(newTool.parameters) != 1 || newTool.parameters[0].Name != "days" {
			t.Error("Expected 'city' to be removed from the unbound parameters list")
		}
	})

	t.Run("Negative Test - fails when overriding an existing bound parameter", func(t *testing.T) {
		tool := getTestTool()
		_, err := tool.ToolFrom(WithBindParamString("units", "fahrenheit"))
		if err == nil {
			t.Fatal("Expected an error when overriding 'units' parameter, but got nil")
		}
		if !strings.Contains(err.Error(), "cannot override existing bound parameter: 'units'") {
			t.Errorf("Incorrect error message for override. Got: %v", err)
		}
	})

	t.Run("Negative Test - fails when overriding an existing auth source", func(t *testing.T) {
		tool := getTestTool()
		_, err := tool.ToolFrom(WithAuthTokenString("google", "new-token"))
		if err == nil {
			t.Fatal("Expected an error when overriding 'google' auth source, but got nil")
		}
		if !strings.Contains(err.Error(), "cannot override existing auth token source: 'google'") {
			t.Errorf("Incorrect error message for override. Got: %v", err)
		}
	})

	t.Run("Negative Test - fails when using WithStrict option", func(t *testing.T) {
		tool := getTestTool()
		_, err := tool.ToolFrom(WithStrict(true))
		if err == nil {
			t.Fatal("Expected an error when using WithStrict, but got nil")
		}
		if !strings.Contains(err.Error(), "WithStrict option is not applicable") {
			t.Errorf("Incorrect error message for WithStrict. Got: %v", err)
		}
	})

	t.Run("Negative Test - binding a completely unknown parameter", func(t *testing.T) {
		tool := getTestTool()
		_, err := tool.ToolFrom(WithBindParamString("country", "UK"))
		if err == nil {
			t.Fatal("Expected an error when binding an unknown parameter, but got nil")
		}
		if !strings.Contains(err.Error(), "no parameter named 'country'") {
			t.Errorf("Incorrect error message for unknown parameter. Got: %q", err.Error())
		}
	})

	t.Run("Negative Test - conflicting options are provided", func(t *testing.T) {
		tool := getTestTool()
		_, err := tool.ToolFrom(
			WithBindParamString("city", "A"),
			WithBindParamString("city", "B"),
		)
		if err == nil {
			t.Fatal("Expected an error from a duplicate option, but got nil")
		}
		if !strings.Contains(err.Error(), "duplicate parameter binding") {
			t.Errorf("Incorrect error message for conflicting options. Got: %q", err.Error())
		}
	})
}

func TestCloneToolboxTool(t *testing.T) {
	// 1. Setup an original tool with populated maps and slices to test deep copying.
	originalTool := &ToolboxTool{
		name:        "original_tool",
		description: "An original tool to be cloned.",
		parameters: []ParameterSchema{
			{Name: "p1", Type: "string"},
		},
		boundParams: map[string]any{
			"b1":        "value1",
			"callbacks": []string{"original_func"},
		},
		authTokenSources: map[string]oauth2.TokenSource{
			"auth1": &mockTokenSource{},
		},
		requiredAuthnParams: map[string][]string{
			"req1": {"google", "github"},
		},
		requiredAuthzTokens: []string{"system_token"},
		clientHeaderSources: map[string]oauth2.TokenSource{
			"header1": &mockTokenSource{},
		},
	}

	clone := originalTool.cloneToolboxTool()

	if originalTool == clone {
		t.Fatal("Clone should not be the same instance (pointer) as the original")
	}
	if !reflect.DeepEqual(originalTool, clone) {
		t.Fatal("Initial clone is not deeply equal to the original")
	}

	t.Run("Negative Test - modifying clone's boundParams map", func(t *testing.T) {
		clone.boundParams["b2"] = "newValue"
		delete(clone.boundParams, "b1")

		if _, exists := originalTool.boundParams["b2"]; exists {
			t.Error("Modifying clone's boundParams added a key to the original's map")
		}
		if _, exists := originalTool.boundParams["b1"]; !exists {
			t.Error("Modifying clone's boundParams deleted a key from the original's map")
		}
	})

	t.Run("Negative Test - modifying clone's parameters slice", func(t *testing.T) {
		clone.parameters = append(clone.parameters, ParameterSchema{Name: "p2"})

		if len(originalTool.parameters) != 1 {
			t.Errorf("Appending to clone's parameters slice changed the length of the original. Got length %d, want 1", len(originalTool.parameters))
		}
	})

	t.Run("Negative Test - modifying nested slice in requiredAuthnParams map", func(t *testing.T) {
		clone.requiredAuthnParams["req1"][0] = "overwritten_value"
		clone.requiredAuthnParams["req1"] = append(clone.requiredAuthnParams["req1"], "new_service")

		originalSlice := originalTool.requiredAuthnParams["req1"]
		if originalSlice[0] != "google" {
			t.Errorf("Modifying a nested slice in the clone's map changed a value in the original's slice. Got %q, want 'google'", originalSlice[0])
		}
		if len(originalSlice) != 2 {
			t.Errorf("Appending to a nested slice in the clone's map changed the length of the original's slice. Got length %d, want 2", len(originalSlice))
		}
	})

	t.Run("Negative Test - modifying a slice within boundParams map", func(t *testing.T) {
		// This test verifies that reference types within the boundParams map are not shared.
		// Note: The current cloneToolboxTool implementation performs a shallow copy of this map's
		// values, so this test would fail unless the clone function is updated to deep copy them.

		// Action: Modify the slice inside the clone's map.
		clonedSlice := clone.boundParams["callbacks"].([]string)
		clonedSlice[0] = "modified_func"

		// Assert: Check if the original tool's slice was affected.
		originalSlice := originalTool.boundParams["callbacks"].([]string)
		if originalSlice[0] != "original_func" {
			t.Error("Modifying a slice in the clone's boundParams affected the original (shallow copy bug)")
		}
	})

	t.Run("Negative Test - modifying clone's authTokenSources map", func(t *testing.T) {
		clone.authTokenSources["auth2"] = &mockTokenSource{}

		if len(originalTool.authTokenSources) != 1 {
			t.Errorf("Modifying clone's authTokenSources map changed the length of the original. Got length %d, want 1", len(originalTool.authTokenSources))
		}
	})
}
