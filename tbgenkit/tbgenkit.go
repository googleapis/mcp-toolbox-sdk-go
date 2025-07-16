// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package tbgenkit

import (
	"encoding/json"
	"fmt"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/googleapis/mcp-toolbox-sdk-go/core"
	"github.com/invopop/jsonschema"
)

// ToGenkitTool converts a custom ToolboxTool into a genkit ai.Tool
// Inputs:
//
//	tool: A pointer to the custom `core.ToolboxTool` to be converted.
//	g:    A pointer to the `genkit.Genkit` instance to register the tool.
//
// Returns:
//
//	An `ai.Tool` interface instance representing the Genkit-compatible tool.
//	Returns `nil` if there are critical errors during the conversion process.
func ToGenkitTool(tool *core.ToolboxTool, g *genkit.Genkit) (ai.Tool, error) {
	// Robustness Checks
	if tool == nil {
		err := fmt.Errorf("error: ToGenkitTool received a nil core.ToolboxTool pointer")
		return nil, err
	}
	if g == nil {
		err := fmt.Errorf("error: ToGenkitTool received a nil genkit.Genkit pointer")
		return nil, err
	}

	// Retrieve the JSON schema bytes from the custom tool.
	jsonBytes, err := tool.InputSchema()
	if err != nil {
		return nil, err
	}

	// Unmarshal the JSON schema bytes into a jsonschema.Schema object.
	var schema *jsonschema.Schema
	if err := json.Unmarshal(jsonBytes, &schema); err != nil {
		return nil, err
	}

	// Define the execution function for the Genkit tool.
	// This function acts as a wrapper around the core.ToolboxTool's Invoke method.
	// It conforms to the `func(ctx *ai.ToolContext, input any) (string, error)` signature
	// required by Genkit's tool definition.
	executeFn := func(ctx *ai.ToolContext, input any) (string, error) {
		// Perform a safe type assertion for the input.
		inputMap, ok := input.(map[string]any)
		if !ok {
			// If the input is not a map, return an error indicating the type mismatch.
			return "", fmt.Errorf("tool input expected map[string]any, got %T", input)
		}
		// Invoke the underlying custom tool with the provided context and input.
		result, err := tool.Invoke(ctx, inputMap)
		if err != nil {
			// Propagate any errors that occurred during the custom tool's invocation.
			return "", fmt.Errorf("error invoking core tool %s: %w", tool.Name(), err)
		}

		// Convert the result from the custom tool's invocation to a string.
		strResult := fmt.Sprintf("%v", result)
		return strResult, nil
	}

	// Create a Genkit Tool
	return genkit.DefineToolWithInputSchema(
		g,
		tool.Name(),
		tool.Description(),
		schema,
		executeFn,
	), nil
}
