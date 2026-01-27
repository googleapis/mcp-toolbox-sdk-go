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
	"testing"

	"github.com/googleapis/mcp-toolbox-sdk-go/core"
)

// convertParamsToJSONSchema reconstructs a raw JSON schema from the SDK's internal ParameterSchema.
// This is needed because the Mock Server must send "raw" JSON, which the Client then parses back into structs.
func convertParamsToJSONSchema(params []core.ParameterSchema) map[string]any {
	properties := make(map[string]any)
	required := []string{}

	for _, p := range params {
		prop := map[string]any{
			"type":        p.Type,
			"description": p.Description,
		}
		properties[p.Name] = prop
		if p.Required {
			required = append(required, p.Name)
		}
	}

	return map[string]any{
		"type":       "object",
		"properties": properties,
		"required":   required,
	}
}

func createCoreTool(t *testing.T, toolName string, schema core.ToolSchema) (*core.ToolboxTool, *httptest.Server) {
	t.Helper()

	// Prepare the Tool definition in MCP JSON format
	mcpToolDef := map[string]any{
		"name":        toolName,
		"description": schema.Description,
		"inputSchema": convertParamsToJSONSchema(schema.Parameters),
	}

	// Setup a Mock MCP Server (JSON-RPC 2.0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			JSONRPC string `json:"jsonrpc"`
			Method  string `json:"method"`
			ID      any    `json:"id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		var result any

		// Handle MCP Protocol Lifecycle
		switch req.Method {
		case "initialize":
			// Handshake
			result = map[string]any{
				"protocolVersion": "2025-11-25", // Matches latest default
				"capabilities":    map[string]any{"tools": map[string]any{}},
				"serverInfo": map[string]any{
					"name":    "mock-server",
					"version": "1.0.0",
				},
			}
		case "notifications/initialized":
			// Confirmation (No response needed)
			return
		case "tools/list":
			// List available tools
			result = map[string]any{
				"tools": []any{mcpToolDef},
			}
		default:
			// Ignore other methods for this test
			return
		}

		// Send JSON-RPC Response
		resp := map[string]any{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"result":  result,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))

	//  Create Client, defaults to Latest MCP (v2025-06-18)
	client, err := core.NewToolboxClient(server.URL, core.WithHTTPClient(server.Client()))
	if err != nil {
		server.Close()
		t.Fatalf("Failed to create ToolboxClient: %v", err)
	}

	// 4. Load the tool (Triggers initialize -> tools/list)
	tool, err := client.LoadTool(toolName, context.Background())
	if err != nil {
		server.Close()
		t.Fatalf("Failed to load tool '%s': %v", toolName, err)
	}

	return tool, server
}

func TestToADKTool(t *testing.T) {

	t.Run("Success - Happy Path with parameters", func(t *testing.T) {
		toolSchema := core.ToolSchema{
			Description: "Get the weather",
			Parameters: []core.ParameterSchema{
				{Name: "location", Type: "string", Description: "The city", Required: true},
				{Name: "unit", Type: "string", Description: "celsius or fahrenheit"},
			},
		}

		// Create Core Tool via MCP Mock
		coreTool, server := createCoreTool(t, "getWeather", toolSchema)
		defer server.Close()

		// Convert to ADK Tool
		adkTool, err := toADKTool(coreTool)

		if err != nil {
			t.Fatalf("toADKTool() unexpected error = %v", err)
		}
		if adkTool.funcDeclaration == nil {
			t.Fatal("adkTool.funcDeclaration is nil")
		}

		// Verify Basic Fields
		if got, want := adkTool.funcDeclaration.Name, "getWeather"; got != want {
			t.Errorf("funcDeclaration.Name = %q, want %q", got, want)
		}
		if got, want := adkTool.funcDeclaration.Description, "Get the weather"; got != want {
			t.Errorf("funcDeclaration.Description = %q, want %q", got, want)
		}

		// Verify Schema Conversion
		var params map[string]any
		schema, err := adkTool.InputSchema()
		if err != nil {
			t.Error("Failed to fetch input schema", err)
		}
		if err := json.Unmarshal(schema, &params); err != nil {
			t.Fatalf("Failed to unmarshal generated parameters schema: %v", err)
		}

		// Expected JSON Structure
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
		toolSchema := core.ToolSchema{
			Description: "A tool with no params",
			Parameters:  nil,
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
		_ = json.Unmarshal(schema, &params)

		expectedParamsJSON := `{"type": "object", "properties": {}}`
		var expectedParams map[string]any
		_ = json.Unmarshal([]byte(expectedParamsJSON), &expectedParams)

		if !reflect.DeepEqual(expectedParams, params) {
			t.Errorf("Generated parameter schema not empty.\nGot: %v\nWant: %v", params, expectedParams)
		}
	})

	t.Run("Failure - Nil Input", func(t *testing.T) {
		_, err := toADKTool(nil)
		if err == nil {
			t.Errorf("toADKTool(nil) expects error but got nil")
		}
	})

}
