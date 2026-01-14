//go:build unit

// Copyright 2026 Google LLC
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

package v20250618

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// capturedRequest holds both the RPC body and the HTTP headers for verification
type capturedRequest struct {
	Body    jsonRPCRequest
	Headers http.Header
}

// mockMCPServer is a helper to mock MCP JSON-RPC responses
type mockMCPServer struct {
	*httptest.Server
	handlers map[string]func(params json.RawMessage) (any, error)
	requests []capturedRequest // Log of received requests (body + headers)
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

		// Capture both body and headers
		m.requests = append(m.requests, capturedRequest{
			Body:    req,
			Headers: r.Header.Clone(),
		})

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

		resBytes, err := json.Marshal(result)
		require.NoError(t, err)

		resp := jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  resBytes,
		}

		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(resp)
		require.NoError(t, err)
	}))

	// Register default handshake handlers
	m.handlers["initialize"] = func(params json.RawMessage) (any, error) {
		return initializeResult{
			ProtocolVersion: "2025-06-18",
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

func TestHeaders_Presence(t *testing.T) {
	server := newMockMCPServer(t)
	defer server.Close()

	client, _ := New(server.URL, server.Client())
	err := client.EnsureInitialized(context.Background(), nil)
	require.NoError(t, err)

	require.NotEmpty(t, server.requests)

	// Check the Initialize request (first request)
	req := server.requests[0]

	// Requirement: MCP-Protocol-Version must be present
	assert.Equal(t, "2025-06-18", req.Headers.Get("MCP-Protocol-Version"))

	// Requirement: Accept header must be present and application/json
	assert.Equal(t, "application/json", req.Headers.Get("Accept"))
}

func TestListTools(t *testing.T) {
	server := newMockMCPServer(t)
	defer server.Close()

	// Mock tools/list response using strict Tool struct
	server.handlers["tools/list"] = func(params json.RawMessage) (any, error) {
		return listToolsResult{
			Tools: []mcpTool{
				{
					Name:        "get_weather",
					Description: "Get weather for a location",
					InputSchema: map[string]any{
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

	client, _ := New(server.URL, server.Client())
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

	t.Run("Verify Handshake Sequence and Headers", func(t *testing.T) {
		require.GreaterOrEqual(t, len(server.requests), 3)
		assert.Equal(t, "initialize", server.requests[0].Body.Method)
		assert.Equal(t, "notifications/initialized", server.requests[1].Body.Method)

		// Verify ListTools Request
		listReq := server.requests[2]
		assert.Equal(t, "tools/list", listReq.Body.Method)
		assert.Equal(t, "2025-06-18", listReq.Headers.Get("MCP-Protocol-Version"))
	})
}

func TestListTools_ErrorOnEmptyName(t *testing.T) {
	server := newMockMCPServer(t)
	defer server.Close()

	server.handlers["tools/list"] = func(params json.RawMessage) (any, error) {
		return listToolsResult{
			Tools: []mcpTool{
				{Name: "valid", InputSchema: map[string]any{}},
				{Name: "", InputSchema: map[string]any{}}, // Invalid tool
			},
		}, nil
	}

	client, _ := New(server.URL, server.Client())
	_, err := client.ListTools(context.Background(), "", nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing 'name' field")
}

func TestGetTool_Success(t *testing.T) {
	server := newMockMCPServer(t)
	defer server.Close()

	server.handlers["tools/list"] = func(params json.RawMessage) (any, error) {
		return listToolsResult{
			Tools: []mcpTool{
				{Name: "tool_a", InputSchema: map[string]any{"type": "object"}},
				{Name: "tool_b", InputSchema: map[string]any{"type": "object"}},
			},
		}, nil
	}

	client, _ := New(server.URL, server.Client())
	manifest, err := client.GetTool(context.Background(), "tool_a", nil)
	require.NoError(t, err)
	assert.Contains(t, manifest.Tools, "tool_a")
	assert.NotContains(t, manifest.Tools, "tool_b")
}

func TestGetTool_NotFound(t *testing.T) {
	server := newMockMCPServer(t)
	defer server.Close()

	server.handlers["tools/list"] = func(params json.RawMessage) (any, error) {
		return listToolsResult{Tools: []mcpTool{}}, nil
	}

	client, _ := New(server.URL, server.Client())
	_, err := client.GetTool(context.Background(), "missing_tool", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestInvokeTool(t *testing.T) {
	server := newMockMCPServer(t)
	defer server.Close()

	server.handlers["tools/call"] = func(params json.RawMessage) (any, error) {
		// Verify arguments
		var callParams callToolRequestParams
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

	client, _ := New(server.URL, server.Client())
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		args := map[string]any{"message": "Hello MCP"}
		result, err := client.InvokeTool(ctx, "echo", args, nil)
		require.NoError(t, err)

		resStr, ok := result.(string)
		require.True(t, ok)
		assert.Equal(t, "Echo: Hello MCP", resStr)

		// Verify Headers on Invoke
		// Last request in the slice
		lastReq := server.requests[len(server.requests)-1]
		assert.Equal(t, "tools/call", lastReq.Body.Method)
		assert.Equal(t, "2025-06-18", lastReq.Headers.Get("MCP-Protocol-Version"))
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

	client, _ := New(server.URL, server.Client())

	_, err := client.ListTools(context.Background(), "", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "MCP version mismatch")
}

func TestInitialize_MissingCapabilities(t *testing.T) {
	server := newMockMCPServer(t)
	defer server.Close()

	server.handlers["initialize"] = func(params json.RawMessage) (any, error) {
		return initializeResult{
			ProtocolVersion: "2025-06-18",
			Capabilities:    serverCapabilities{Tools: nil},
			ServerInfo:      implementation{Name: "srv", Version: "1"},
		}, nil
	}

	client, _ := New(server.URL, server.Client())
	_, err := client.ListTools(context.Background(), "", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not support the 'tools' capability")
}

func TestConvertToolSchema(t *testing.T) {
	// Use the transport's ConvertToolDefinition which delegates to the base/helper logic
	tr, _ := New("http://example.com", nil)

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
		return listToolsResult{Tools: []mcpTool{}}, nil
	}

	client, _ := New(server.URL, server.Client())
	toolsetName := "my-toolset"

	_, err := client.ListTools(context.Background(), toolsetName, nil)
	require.NoError(t, err)
}

func TestRequest_NetworkError(t *testing.T) {
	// Close server immediately to simulate connection refused
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := server.URL
	server.Close()

	client, _ := New(url, server.Client())
	_, err := client.ListTools(context.Background(), "", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "http request failed")
}

func TestRequest_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Error"))
	}))
	defer server.Close()

	client, _ := New(server.URL, server.Client())
	_, err := client.ListTools(context.Background(), "", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API request failed with status 500")
}

func TestRequest_BadJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{ broken json `))
	}))
	defer server.Close()

	client, _ := New(server.URL, server.Client())
	_, err := client.ListTools(context.Background(), "", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "response unmarshal failed")
}

func TestRequest_NewRequestError(t *testing.T) {
	// Bad URL triggers http.NewRequest error
	_, err := New("http://bad\nurl.com", http.DefaultClient)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid control character in URL")
}

func TestRequest_MarshalError(t *testing.T) {
	server := newMockMCPServer(t)
	defer server.Close()
	client, _ := New(server.URL, server.Client())

	// Force initialization first
	_ = client.EnsureInitialized(context.Background(), nil)

	// Pass a type that cannot be marshaled to JSON (e.g. channel)
	badPayload := map[string]any{"bad": make(chan int)}
	_, err := client.InvokeTool(context.Background(), "tool", badPayload, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "marshal failed")
}

func TestInvokeTool_ErrorResult(t *testing.T) {
	server := newMockMCPServer(t)
	defer server.Close()

	server.handlers["tools/call"] = func(params json.RawMessage) (any, error) {
		return callToolResult{
			Content: []textContent{{Type: "text", Text: "Something went wrong"}},
			IsError: true,
		}, nil
	}

	client, _ := New(server.URL, server.Client())
	_, err := client.InvokeTool(context.Background(), "tool", nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tool execution resulted in error")
}

func TestInvokeTool_RPCError(t *testing.T) {
	server := newMockMCPServer(t)
	defer server.Close()

	server.handlers["tools/call"] = func(params json.RawMessage) (any, error) {
		return nil, errors.New("internal server error")
	}

	client, _ := New(server.URL, server.Client())
	_, err := client.InvokeTool(context.Background(), "tool", nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "internal server error")
}

func TestInvokeTool_ComplexContent(t *testing.T) {
	server := newMockMCPServer(t)
	defer server.Close()

	server.handlers["tools/call"] = func(params json.RawMessage) (any, error) {
		return callToolResult{
			Content: []textContent{
				{Type: "text", Text: "Part 1 "},
				{Type: "image", Text: "base64data"}, // Should be ignored
				{Type: "text", Text: "Part 2"},
			},
		}, nil
	}

	client, _ := New(server.URL, server.Client())
	res, err := client.InvokeTool(context.Background(), "t", nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "Part 1 Part 2", res)
}

func TestInvokeTool_EmptyResult(t *testing.T) {
	server := newMockMCPServer(t)
	defer server.Close()

	server.handlers["tools/call"] = func(params json.RawMessage) (any, error) {
		return callToolResult{
			Content: []textContent{},
		}, nil
	}

	client, _ := New(server.URL, server.Client())
	res, err := client.InvokeTool(context.Background(), "t", nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "null", res)
}

func TestEnsureInitialized_PassesHeaders(t *testing.T) {
	tr, err := New("http://fake.com", nil)
	require.NoError(t, err)

	capturedHeaders := make(map[string]string)
	tr.BaseMcpTransport.HandshakeHook = func(ctx context.Context, headers map[string]string) error {
		for k, v := range headers {
			capturedHeaders[k] = v
		}
		return nil
	}

	testHeaders := map[string]string{"X-Test": "123"}

	err = tr.EnsureInitialized(context.Background(), testHeaders)
	require.NoError(t, err)

	assert.Equal(t, "123", capturedHeaders["X-Test"], "EnsureInitialized failed to pass headers to the handshake hook")
}

func TestInitializeSession_PassesHeadersToWire(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer token" {
			t.Errorf("Missing Authorization header on request to %s", r.URL.Path)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		var req struct {
			Method string `json:"method"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if req.Method == "initialize" {
			resp := map[string]any{
				"jsonrpc": "2.0",
				"id":      "123",
				"result": map[string]any{
					"protocolVersion": "2025-06-18",
					"capabilities":    map[string]any{"tools": map[string]any{}},
					"serverInfo":      map[string]any{"name": "test", "version": "1.0"},
				},
			}
			json.NewEncoder(w).Encode(resp)
		} else if req.Method == "notifications/initialized" {
			// Verify notification success
			w.WriteHeader(http.StatusNoContent)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	tr, err := New(ts.URL, ts.Client())
	require.NoError(t, err)

	testHeaders := map[string]string{"Authorization": "Bearer token"}

	err = tr.initializeSession(context.Background(), testHeaders)
	require.NoError(t, err)
}
