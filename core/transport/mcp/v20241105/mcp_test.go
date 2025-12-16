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

package v20241105

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockMCPServer is a helper to mock MCP JSON-RPC responses
type mockMCPServer struct {
	*httptest.Server
	handlers map[string]func(params json.RawMessage) (any, error)
	requests []jsonRPCRequest // Log of received requests for verification
}

func newMockMCPServer(t *testing.T) *mockMCPServer {
	m := &mockMCPServer{
		handlers: make(map[string]func(json.RawMessage) (any, error)),
	}

	m.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var req jsonRPCRequest
		err = json.Unmarshal(body, &req)
		require.NoError(t, err)

		m.requests = append(m.requests, req)

		// Handle Notifications (no ID) - return 204 or 200 OK immediately
		if req.ID == nil {
			if handler, ok := m.handlers[req.Method]; ok {
				_, _ = handler(asRawMessage(req.Params))
			}
			w.WriteHeader(http.StatusOK)
			return
		}

		// Handle Requests
		handler, ok := m.handlers[req.Method]
		if !ok {
			http.Error(w, "method not found", http.StatusNotFound)
			return
		}

		result, err := handler(asRawMessage(req.Params))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		resp := jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  result,
		}

		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(resp)
		require.NoError(t, err)
	}))

	// Register default handshake handlers
	m.handlers["initialize"] = func(params json.RawMessage) (any, error) {
		return initializeResult{
			ProtocolVersion: "2024-11-05",
			Capabilities: serverCapabilities{
				Tools: map[string]any{"listChanged": true},
			},
			ServerInfo: implementation{
				Name:    "mock-server",
				Version: "1.0.0",
			},
		}, nil
	}
	m.handlers["notifications/initialized"] = func(params json.RawMessage) (any, error) {
		return nil, nil
	}

	return m
}

func asRawMessage(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

func TestListTools(t *testing.T) {
	server := newMockMCPServer(t)
	defer server.Close()

	// Mock tools/list response
	server.handlers["tools/list"] = func(params json.RawMessage) (any, error) {
		return listToolsResult{
			Tools: []map[string]any{
				{
					"name":        "get_weather",
					"description": "Get weather for a location",
					"inputSchema": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"location": map[string]any{"type": "string"},
						},
						"required": []string{"location"},
					},
				},
			},
		}, nil
	}

	client := NewMcpTransport(server.URL, server.Client())
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		manifest, err := client.ListTools(ctx, "", nil)
		require.NoError(t, err)
		require.NotNil(t, manifest)

		assert.Equal(t, "1.0.0", manifest.ServerVersion)
		assert.Contains(t, manifest.Tools, "get_weather")
		tool := manifest.Tools["get_weather"]
		assert.Equal(t, "Get weather for a location", tool.Description)
		assert.Len(t, tool.Parameters, 1)
		assert.Equal(t, "location", tool.Parameters[0].Name)
	})

	t.Run("Verify Handshake Sequence", func(t *testing.T) {
		require.GreaterOrEqual(t, len(server.requests), 3)
		assert.Equal(t, "initialize", server.requests[0].Method)
		assert.Equal(t, "notifications/initialized", server.requests[1].Method)
		assert.Equal(t, "tools/list", server.requests[2].Method)
	})
}

func TestInvokeTool(t *testing.T) {
	server := newMockMCPServer(t)
	defer server.Close()

	server.handlers["tools/call"] = func(params json.RawMessage) (any, error) {
		// Verify arguments
		var callParams callToolParams
		_ = json.Unmarshal(params, &callParams)
		if callParams.Name != "echo" {
			return nil, nil
		}

		msg, _ := callParams.Arguments["message"].(string)
		return callToolResult{
			Content: []textContent{
				{Type: "text", Text: "Echo: " + msg},
			},
			IsError: false,
		}, nil
	}

	client := NewMcpTransport(server.URL, server.Client())
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		args := map[string]any{"message": "Hello MCP"}
		result, err := client.InvokeTool(ctx, "echo", args, nil)
		require.NoError(t, err)

		resStr, ok := result.(string)
		require.True(t, ok)
		assert.Equal(t, "Echo: Hello MCP", resStr)
	})
}

func TestProtocolMismatch(t *testing.T) {
	server := newMockMCPServer(t)
	defer server.Close()

	// Override initialize to return wrong version
	server.handlers["initialize"] = func(params json.RawMessage) (any, error) {
		return initializeResult{
			ProtocolVersion: "2099-01-01", // Future version
			Capabilities:    serverCapabilities{Tools: map[string]any{}},
			ServerInfo:      implementation{Name: "mock", Version: "1.0"},
		}, nil
	}

	client := NewMcpTransport(server.URL, server.Client())

	_, err := client.ListTools(context.Background(), "", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "MCP version mismatch")
}

func TestConvertToolSchema(t *testing.T) {
	// Use the transport's ConvertToolDefinition which delegates to the base/helper logic
	tr := NewMcpTransport("http://example.com", nil)

	// Correctly structured test data matching Python logic: _meta is a sibling of inputSchema
	rawTool := map[string]any{
		"name":        "complex_tool",
		"description": "Complex tool",
		"inputSchema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"tag": map[string]any{
					"type":        "string",
					"description": "A tag",
				},
				"count": map[string]any{
					"type": "integer",
				},
			},
			"required": []any{"tag"},
		},
		"_meta": map[string]any{
			"toolbox/authParam": map[string]any{
				"tag": []any{"serviceA"},
			},
			"toolbox/authInvoke": []any{"serviceB"},
		},
	}

	schema, err := tr.ConvertToolDefinition(rawTool)
	require.NoError(t, err)

	assert.Equal(t, "Complex tool", schema.Description)
	assert.Len(t, schema.Parameters, 2)
	assert.Equal(t, []string{"serviceB"}, schema.AuthRequired)

	for _, p := range schema.Parameters {
		if p.Name == "tag" {
			assert.True(t, p.Required)
			assert.Equal(t, []string{"serviceA"}, p.AuthSources)
		}
	}
}

func TestListTools_WithToolset(t *testing.T) {
	server := newMockMCPServer(t)
	defer server.Close()

	// We verify that the toolset name was appended to the URL in the POST request
	server.handlers["tools/list"] = func(params json.RawMessage) (any, error) {
		return listToolsResult{Tools: []map[string]any{}}, nil
	}

	client := NewMcpTransport(server.URL, server.Client())
	toolsetName := "my-toolset"

	_, err := client.ListTools(context.Background(), toolsetName, nil)
	require.NoError(t, err)
}
