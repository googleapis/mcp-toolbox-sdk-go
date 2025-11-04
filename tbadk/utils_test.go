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

package tbadk

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/googleapis/mcp-toolbox-sdk-go/core"
)

func createCoreTool(t *testing.T, toolName string, schema core.ToolSchema) (*core.ToolboxTool, *httptest.Server) {
	t.Helper()

	// Create a mock manifest
	manifest := core.ManifestSchema{
		ServerVersion: "v1",
		Tools: map[string]core.ToolSchema{
			toolName: schema,
		},
	}
	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("Failed to marshal mock manifest: %v", err)
	}

	// Setup a mock server to serve this manifest.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle the specific tool manifest request from LoadTool
		if strings.HasSuffix(r.URL.Path, "/api/tool/"+toolName) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(manifestJSON)
			return
		}
		http.NotFound(w, r)
	}))

	// Create a real client pointing to the mock server.
	client, err := core.NewToolboxClient(server.URL, core.WithHTTPClient(server.Client()))
	if err != nil {
		server.Close()
		t.Fatalf("Failed to create ToolboxClient: %v", err)
	}

	// Load the tool, which returns the real *core.ToolboxTool instance.
	tool, err := client.LoadTool(toolName, context.Background())
	if err != nil {
		server.Close()
		t.Fatalf("Failed to load tool '%s': %v", toolName, err)
	}

	return tool, server
}
func TesttoADKTool(t *testing.T) {

	t.Run("Success - Happy Path with parameters", func(t *testing.T) {
		toolSchema := core.ToolSchema{
			Description: "Get the weather",
			Parameters: []core.ParameterSchema{
				{Name: "location", Type: "string", Description: "The city", Required: true},
				{Name: "unit", Type: "string", Description: "celsius or fahrenheit"},
			},
		}

		// Create Core Tool
		coreTool, server := createCoreTool(t, "getWeather", toolSchema)
		defer server.Close() // Ensure server is closed after the test

		// Convert the Core tool to ADK Tool
		adkTool, err := toADKTool(coreTool)

		if err != nil {
			t.Fatalf("toADKTool() unexpected error = %v", err)
		}
		if adkTool.funcDeclaration == nil {
			t.Fatal("adkTool.funcDeclaration is nil")
		}
		if adkTool.ToolboxTool != coreTool {
			t.Error("adkTool.ToolboxTool does not point to the original tool")
		}

		// Verify the FunctionDeclaration fields
		if got, want := adkTool.funcDeclaration.Name, "getWeather"; got != want {
			t.Errorf("funcDeclaration.Name = %q, want %q", got, want)
		}
		if got, want := adkTool.funcDeclaration.Description, "Get the weather"; got != want {
			t.Errorf("funcDeclaration.Description = %q, want %q", got, want)
		}

		// Verify the parameters schema
		var params map[string]any
		schema, err := adkTool.InputSchema()
		if err != nil {
			t.Error("Failed to fetch input schema", err)
		}
		err = json.Unmarshal(schema, &params)
		if err != nil {
			t.Fatalf("Failed to unmarshal generated parameters schema: %v", err)
		}

		expectedParamsJSON := `
		{
			"type": "object",
			"properties": {
				"location": { "type": "string", "description": "The city" },
				"unit": { "type": "string", "description": "celsius or fahrenheit" }
			},
			"required": ["location"]
		}`
		var expectedParams map[string]any
		_ = json.Unmarshal([]byte(expectedParamsJSON), &expectedParams)

		if !reflect.DeepEqual(expectedParams, params) {
			t.Errorf("Generated parameter schema does not match expected.\nGot: %v\nWant: %v", params, expectedParams)
		}
	})

	t.Run("Success - No Parameters", func(t *testing.T) {
		// Define schema with no parameters
		toolSchema := core.ToolSchema{
			Description: "A tool with no params",
			Parameters:  nil, // Test nil slice
		}

		coreTool, server := createCoreTool(t, "noParams", toolSchema)
		defer server.Close()

		adkTool, err := toADKTool(coreTool)

		if err != nil {
			t.Fatalf("toADKTool() unexpected error = %v", err)
		}
		if got, want := adkTool.funcDeclaration.Name, "noParams"; got != want {
			t.Errorf("funcDeclaration.Name = %q, want %q", got, want)
		}

		var params map[string]any
		schema, err := adkTool.InputSchema()
		if err != nil {
			t.Error("Failed to fetch input schema", err)
		}
		err = json.Unmarshal(schema, &params)
		if err != nil {
			t.Fatalf("Failed to unmarshal generated parameters schema: %v", err)
		}

		// core.ToolboxTool.InputSchema() correctly returns an empty properties map
		expectedParamsJSON := `{"type": "object", "properties": {}}`
		var expectedParams map[string]any
		_ = json.Unmarshal([]byte(expectedParamsJSON), &expectedParams)

		if !reflect.DeepEqual(expectedParams, params) {
			t.Errorf("Generated parameter schema not empty.\nGot: %v\nWant: %v", params, expectedParams)
		}
	})

	t.Run("Nil Input", func(t *testing.T) {
		_, err := toADKTool(nil)
		if err != nil {
			t.Errorf("toADKTool(nil) error = %v, want nil", err)
		}
	})

}
