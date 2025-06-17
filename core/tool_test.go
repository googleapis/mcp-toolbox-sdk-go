package core

import (
	"reflect"
	"testing"
)

func TestToolboxTool_Getters(t *testing.T) {
	sampleParams := []Parameter{
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
			// reflect.DeepEqual is the standard way to compare complex types like slices of structs.
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
				parameters: []Parameter{},
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
