//go:build unit

// Copyright 2026 Google LLC
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

package mcp20250326

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

// mockMCPServer is a helper to mock MCP JSON-RPC responses
type mockMCPServer struct {
	*httptest.Server
	handlers map[string]func(json.RawMessage) (any, map[string]string, error)
	requests []capturedRequest
}

type capturedRequest struct {
	Body    jsonRPCRequest
	Headers http.Header
}

func newMockMCPServer() *mockMCPServer {
	m := &mockMCPServer{
		handlers: make(map[string]func(json.RawMessage) (any, map[string]string, error)),
	}

	m.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "read body failed", http.StatusBadRequest)
			return
		}

		var req jsonRPCRequest
		if err := json.Unmarshal(body, &req); err != nil {
			http.Error(w, "json unmarshal failed", http.StatusBadRequest)
			return
		}

		// Capture the full request context (Body + Headers)
		m.requests = append(m.requests, capturedRequest{
			Body:    req,
			Headers: r.Header.Clone(),
		})

		// Handle Notifications (no ID)
		if req.ID == nil {
			if handler, ok := m.handlers[req.Method]; ok {
				_, _, _ = handler(asRawMessage(req.Params))
			}
			w.WriteHeader(http.StatusOK)
			return
		}

		// Handle Requests
		handler, ok := m.handlers[req.Method]
		if !ok {
			http.Error(w, "method not found: "+req.Method, http.StatusNotFound)
			return
		}

		result, headers, err := handler(asRawMessage(req.Params))

		resp := jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
		}

		if err != nil {
			resp.Error = &jsonRPCError{
				Code:    -32000,
				Message: err.Error(),
			}
		} else {
			// Marshal result to RawMessage
			resBytes, _ := json.Marshal(result)
			resp.Result = resBytes
		}

		w.Header().Set("Content-Type", "application/json")

		if headers != nil {
			for k, v := range headers {
				w.Header().Set(k, v)
			}
		}

		_ = json.NewEncoder(w).Encode(resp)
	}))

	// Register default successful handshake with a Session ID
	m.handlers["initialize"] = func(params json.RawMessage) (any, map[string]string, error) {
		sessionId := "session-12345"

		return initializeResult{
				ProtocolVersion: ProtocolVersion,
				Capabilities: serverCapabilities{
					Tools: map[string]any{"listChanged": true},
				},
				ServerInfo: implementation{
					Name:    "mock-server",
					Version: "1.0.0",
				},
			},
			map[string]string{
				"Mcp-Session-Id": sessionId,
			},
			nil
	}

	m.handlers["notifications/initialized"] = func(params json.RawMessage) (any, map[string]string, error) {
		return nil, nil, nil
	}

	return m
}

func asRawMessage(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

func TestInitialize_Success(t *testing.T) {
	server := newMockMCPServer()
	defer server.Close()

	client, _ := New(server.URL, server.Client())

	// Trigger handshake via EnsureInitialized
	err := client.EnsureInitialized(context.Background())
	require.NoError(t, err)

	assert.Equal(t, "1.0.0", client.ServerVersion)
	assert.Equal(t, "session-12345", client.sessionId)

	require.NotEmpty(t, server.requests)
	assert.Equal(t, "application/json", server.requests[0].Headers.Get("Accept"))
}

func TestInitialize_MissingSessionId(t *testing.T) {
	server := newMockMCPServer()
	defer server.Close()

	server.handlers["initialize"] = func(params json.RawMessage) (any, map[string]string, error) {
		return initializeResult{
			ProtocolVersion: ProtocolVersion,
			Capabilities:    serverCapabilities{Tools: map[string]any{"listChanged": true}},
			ServerInfo:      implementation{Name: "bad-server", Version: "1"},
		}, nil, nil
	}

	client, _ := New(server.URL, server.Client())
	err := client.EnsureInitialized(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "server did not return an Mcp-Session-Id")
}

func TestSessionId_Injection_InvokeTool(t *testing.T) {
	server := newMockMCPServer()
	defer server.Close()

	server.handlers["tools/call"] = func(params json.RawMessage) (any, map[string]string, error) {
		return callToolResult{
			Content: []textContent{{Type: "text", Text: "OK"}},
		}, nil, nil
	}

	client, _ := New(server.URL, server.Client())
	_, err := client.InvokeTool(context.Background(), "test-tool", map[string]any{"a": 1}, nil)
	require.NoError(t, err)

	// Verify requests
	// 0: initialize
	// 1: notifications/initialized
	// 2: tools/call
	require.Len(t, server.requests, 3)

	callReq := server.requests[2]
	assert.Equal(t, "tools/call", callReq.Body.Method)

	// Verify Session ID Header
	assert.Equal(t, "session-12345", callReq.Headers.Get("Mcp-Session-Id"), "Session ID header missing")

	// Verify Accept Header
	assert.Equal(t, "application/json", callReq.Headers.Get("Accept"), "Accept header missing or incorrect")
}

func TestSessionId_Injection_ListTools(t *testing.T) {
	server := newMockMCPServer()
	defer server.Close()

	server.handlers["tools/list"] = func(params json.RawMessage) (any, map[string]string, error) {
		return listToolsResult{Tools: []mcpTool{}}, nil, nil
	}

	client, _ := New(server.URL, server.Client())
	_, err := client.ListTools(context.Background(), "", nil)
	require.NoError(t, err)

	require.Len(t, server.requests, 3)
	listReq := server.requests[2]
	assert.Equal(t, "tools/list", listReq.Body.Method)

	// Verify Session ID Header
	assert.Equal(t, "session-12345", listReq.Headers.Get("Mcp-Session-Id"), "Session ID header missing")
}

func TestListTools_MetaPreservation(t *testing.T) {
	server := newMockMCPServer()
	defer server.Close()

	server.handlers["tools/list"] = func(params json.RawMessage) (any, map[string]string, error) {
		return listToolsResult{
			Tools: []mcpTool{
				{
					Name:        "auth_tool",
					Description: "mcpTool with auth",
					InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
					Meta: map[string]any{
						"toolbox/authInvoke": []string{"oauth-scope"},
					},
				},
			},
		}, nil, nil
	}

	client, _ := New(server.URL, server.Client())
	manifest, err := client.ListTools(context.Background(), "", nil)
	require.NoError(t, err)

	tool, ok := manifest.Tools["auth_tool"]
	require.True(t, ok)
	assert.Equal(t, []string{"oauth-scope"}, tool.AuthRequired)
}

func TestGetTool_Success(t *testing.T) {
	server := newMockMCPServer()
	defer server.Close()

	server.handlers["tools/list"] = func(params json.RawMessage) (any, map[string]string, error) {
		return listToolsResult{
			Tools: []mcpTool{
				{Name: "wanted", InputSchema: map[string]any{}},
				{Name: "unwanted", InputSchema: map[string]any{}},
			},
		}, nil, nil
	}

	client, _ := New(server.URL, server.Client())
	manifest, err := client.GetTool(context.Background(), "wanted", nil)
	require.NoError(t, err)
	assert.Contains(t, manifest.Tools, "wanted")
	assert.NotContains(t, manifest.Tools, "unwanted")
}

func TestInvokeTool_ErrorResult(t *testing.T) {
	server := newMockMCPServer()
	defer server.Close()

	server.handlers["tools/call"] = func(params json.RawMessage) (any, map[string]string, error) {
		return callToolResult{
			Content: []textContent{{Type: "text", Text: "Something went wrong"}},
			IsError: true,
		}, nil, nil
	}

	client, _ := New(server.URL, server.Client())
	_, err := client.InvokeTool(context.Background(), "tool", nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tool execution resulted in error")
}

func TestInvokeTool_RPCError(t *testing.T) {
	server := newMockMCPServer()
	defer server.Close()

	server.handlers["tools/call"] = func(params json.RawMessage) (any, map[string]string, error) {
		return nil, nil, errors.New("internal server error")
	}

	client, _ := New(server.URL, server.Client())
	_, err := client.InvokeTool(context.Background(), "tool", nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "internal server error")
}

func TestListTools_WithAuthHeaders(t *testing.T) {
	server := newMockMCPServer()
	defer server.Close()

	server.handlers["tools/list"] = func(params json.RawMessage) (any, map[string]string, error) {
		return listToolsResult{Tools: []mcpTool{}}, nil, nil
	}

	client, _ := New(server.URL, server.Client())
	headers := map[string]string{"Authorization": "secret"}

	_, err := client.ListTools(context.Background(), "", headers)
	require.NoError(t, err)
}

func TestProtocolVersionMismatch(t *testing.T) {
	server := newMockMCPServer()
	defer server.Close()

	server.handlers["initialize"] = func(params json.RawMessage) (any, map[string]string, error) {
		return initializeResult{
			ProtocolVersion: "2099-01-01",
			Capabilities:    serverCapabilities{Tools: map[string]any{}},
			ServerInfo:      implementation{Name: "futuristic", Version: "1"},
		}, nil, nil
	}

	client, _ := New(server.URL, server.Client())
	err := client.EnsureInitialized(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "MCP version mismatch")
}

func TestInitialization_MissingCapabilities(t *testing.T) {
	server := newMockMCPServer()
	defer server.Close()

	server.handlers["initialize"] = func(params json.RawMessage) (any, map[string]string, error) {
		return initializeResult{
			ProtocolVersion: ProtocolVersion,
			ServerInfo:      implementation{Name: "bad", Version: "1"},
		}, nil, nil
	}

	client, _ := New(server.URL, server.Client())
	err := client.EnsureInitialized(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not support the 'tools' capability")
}

// --- Error Path Tests ---

func TestRequest_NetworkError(t *testing.T) {
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
	_, err := New("http://bad\nurl.com", http.DefaultClient)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "invalid control character in URL")
}

func TestRequest_MarshalError(t *testing.T) {
	server := newMockMCPServer()
	defer server.Close()
	client, _ := New(server.URL, server.Client())

	// Force initialization first
	_ = client.EnsureInitialized(context.Background())

	badPayload := map[string]any{"bad": make(chan int)}
	_, err := client.InvokeTool(context.Background(), "tool", badPayload, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "marshal failed")
}

func TestGetTool_NotFound(t *testing.T) {
	server := newMockMCPServer()
	defer server.Close()

	server.handlers["tools/list"] = func(params json.RawMessage) (any, map[string]string, error) {
		return listToolsResult{Tools: []mcpTool{}}, nil, nil
	}

	client, _ := New(server.URL, server.Client())
	_, err := client.GetTool(context.Background(), "missing", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestListTools_InitFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := server.URL
	server.Close()

	client, _ := New(url, server.Client())
	_, err := client.ListTools(context.Background(), "", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "http request failed")
}

func TestInit_NotificationFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req jsonRPCRequest
		// Read body to clear buffer, though we just check fields
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &req)

		if req.Method == "initialize" {
			// Success
			resp := jsonRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result:  json.RawMessage(`{"protocolVersion":"2025-03-26","capabilities":{"tools":{}},"serverInfo":{"name":"mock","version":"1"},"Mcp-Session-Id":"s1"}`),
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}
		if req.Method == "notifications/initialized" {
			// Fail
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server Error"))
			return
		}
	}))
	defer server.Close()

	client, _ := New(server.URL, server.Client())
	err := client.EnsureInitialized(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "server did not return an Mcp-Session-Id")
}

func TestInvokeTool_ComplexContent(t *testing.T) {
	server := newMockMCPServer()
	defer server.Close()

	server.handlers["tools/call"] = func(params json.RawMessage) (any, map[string]string, error) {
		return callToolResult{
			Content: []textContent{
				{Type: "text", Text: "Part 1 "},
				{Type: "image", Text: "base64data"}, // Should be ignored
				{Type: "text", Text: "Part 2"},
			},
		}, nil, nil
	}

	client, _ := New(server.URL, server.Client())
	res, err := client.InvokeTool(context.Background(), "t", nil, nil)
	require.NoError(t, err)
	// Only text types should be concatenated
	assert.Equal(t, "Part 1 Part 2", res)
}

func TestInvokeTool_EmptyResult(t *testing.T) {
	server := newMockMCPServer()
	defer server.Close()

	server.handlers["tools/call"] = func(params json.RawMessage) (any, map[string]string, error) {
		return callToolResult{
			Content: []textContent{},
		}, nil, nil
	}

	client, _ := New(server.URL, server.Client())
	res, err := client.InvokeTool(context.Background(), "t", nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "null", res)
}

func TestDoRPC_204_NoContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, _ := New(server.URL, server.Client())
	_, err := client.sendNotification(context.Background(), "test", nil)
	require.NoError(t, err)
}

func TestListTools_ErrorOnEmptyName(t *testing.T) {
	server := newMockMCPServer()
	defer server.Close()

	server.handlers["tools/list"] = func(params json.RawMessage) (any, map[string]string, error) {
		return listToolsResult{
			Tools: []mcpTool{
				{Name: "valid", InputSchema: map[string]any{}},
				{Name: "", InputSchema: map[string]any{}},
			},
		}, nil, nil
	}

	client, _ := New(server.URL, server.Client())
	_, err := client.ListTools(context.Background(), "", nil)

	// Assert that we get an error now
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing 'name' field")
}

func TestInvokeTool_ContentProcessing_Scenarios(t *testing.T) {
	t.Run("Multiple JSON Objects (Merge to Array)", func(t *testing.T) {
		server := newMockMCPServer()
		defer server.Close()

		// Mock response with distinct JSON objects in separate text blocks
		server.handlers["tools/call"] = func(params json.RawMessage) (any, map[string]string, error) {
			return callToolResult{
				Content: []textContent{
					{Type: "text", Text: `{"foo":"bar", "baz": "qux"}`},
					{Type: "text", Text: `{"foo":"quux", "baz":"corge"}`},
				},
				IsError: false,
			}, nil, nil // Return nil for headers and nil for error
		}

		client, _ := New(server.URL, server.Client())
		result, err := client.InvokeTool(context.Background(), "tool", nil, nil)
		require.NoError(t, err)

		// Expectation: The transport should merge these into a single JSON array string
		expected := `[{"foo":"bar", "baz": "qux"},{"foo":"quux", "baz":"corge"}]`
		assert.Equal(t, expected, result)
	})

	t.Run("Split Text (Concatenate)", func(t *testing.T) {
		server := newMockMCPServer()
		defer server.Close()

		// Mock response where text is split across chunks but isn't JSON objects
		server.handlers["tools/call"] = func(params json.RawMessage) (any, map[string]string, error) {
			return callToolResult{
				Content: []textContent{
					{Type: "text", Text: "Hello "},
					{Type: "text", Text: "World"},
				},
				IsError: false,
			}, nil, nil
		}

		client, _ := New(server.URL, server.Client())
		result, err := client.InvokeTool(context.Background(), "tool", nil, nil)
		require.NoError(t, err)

		// Expectation: Simple concatenation
		assert.Equal(t, "Hello World", result)
	})

	t.Run("Split JSON Object (Concatenate)", func(t *testing.T) {
		server := newMockMCPServer()
		defer server.Close()

		// Mock response where a single JSON object is split across chunks.
		server.handlers["tools/call"] = func(params json.RawMessage) (any, map[string]string, error) {
			return callToolResult{
				Content: []textContent{
					{Type: "text", Text: `{"a": `},
					{Type: "text", Text: `1}`},
				},
				IsError: false,
			}, nil, nil
		}

		client, _ := New(server.URL, server.Client())
		result, err := client.InvokeTool(context.Background(), "tool", nil, nil)
		require.NoError(t, err)

		// Expectation: Concatenated to form the valid JSON string
		assert.Equal(t, `{"a": 1}`, result)
	})
}
