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
	"golang.org/x/oauth2"
)

// mockMCPServer is a helper to mock MCP JSON-RPC responses
type mockMCPServer struct {
	*httptest.Server
	handlers map[string]func(params json.RawMessage) (any, error)
	requests []JSONRPCRequest
}

func newMockMCPServer() *mockMCPServer {
	m := &mockMCPServer{
		handlers: make(map[string]func(json.RawMessage) (any, error)),
	}

	m.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "read body failed", http.StatusBadRequest)
			return
		}

		var req JSONRPCRequest
		if err := json.Unmarshal(body, &req); err != nil {
			http.Error(w, "json unmarshal failed", http.StatusBadRequest)
			return
		}

		m.requests = append(m.requests, req)

		// Handle Notifications (no ID)
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
			http.Error(w, "method not found: "+req.Method, http.StatusNotFound)
			return
		}

		result, err := handler(asRawMessage(req.Params))
		resp := JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
		}

		if err != nil {
			resp.Error = &JSONRPCError{
				Code:    -32000,
				Message: err.Error(),
			}
		} else {
			// Marshal result to RawMessage
			resBytes, _ := json.Marshal(result)
			resp.Result = resBytes
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))

	// Register default successful handshake
	m.handlers["initialize"] = func(params json.RawMessage) (any, error) {
		return InitializeResult{
			ProtocolVersion: ProtocolVersion,
			Capabilities: ServerCapabilities{
				Tools: map[string]any{"listChanged": true},
			},
			ServerInfo: Implementation{
				Name:    "mock-server",
				Version: "1.0.0",
			},
			McpSessionId: "session-12345", // Critical for this version
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

func TestInitialize_Success(t *testing.T) {
	server := newMockMCPServer()
	defer server.Close()

	client := New(server.URL, server.Client())

	// Trigger handshake via EnsureInitialized
	err := client.EnsureInitialized(context.Background())
	require.NoError(t, err)

	assert.Equal(t, "1.0.0", client.ServerVersion)
	assert.Equal(t, "session-12345", client.sessionId)
}

func TestInitialize_MissingSessionId(t *testing.T) {
	server := newMockMCPServer()
	defer server.Close()

	// Override initialize to return NO session ID
	server.handlers["initialize"] = func(params json.RawMessage) (any, error) {
		return InitializeResult{
			ProtocolVersion: ProtocolVersion,
			// Must provide non-empty tools so it isn't omitted by json omitempty
			Capabilities: ServerCapabilities{Tools: map[string]any{"listChanged": true}},
			ServerInfo:   Implementation{Name: "bad-server", Version: "1"},
			McpSessionId: "", // Missing
		}, nil
	}

	client := New(server.URL, server.Client())
	err := client.EnsureInitialized(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "did not return a Mcp-Session-Id")
}

func TestSessionId_Injection_InvokeTool(t *testing.T) {
	server := newMockMCPServer()
	defer server.Close()

	server.handlers["tools/call"] = func(params json.RawMessage) (any, error) {
		return CallToolResult{
			Content: []TextContent{{Type: "text", Text: "OK"}},
		}, nil
	}

	client := New(server.URL, server.Client())
	_, err := client.InvokeTool(context.Background(), "test-tool", map[string]any{"a": 1}, nil)
	require.NoError(t, err)

	// Verify requests
	// 0: initialize
	// 1: notifications/initialized
	// 2: tools/call
	require.Len(t, server.requests, 3)

	callReq := server.requests[2]
	assert.Equal(t, "tools/call", callReq.Method)

	// Verify Params contains the session ID
	var paramsMap map[string]any
	// Re-marshal to map to check keys
	json.Unmarshal(asRawMessage(callReq.Params), &paramsMap)

	assert.Equal(t, "session-12345", paramsMap["Mcp-Session-Id"])
	assert.Equal(t, "test-tool", paramsMap["name"])
}

func TestSessionId_Injection_ListTools(t *testing.T) {
	server := newMockMCPServer()
	defer server.Close()

	server.handlers["tools/list"] = func(params json.RawMessage) (any, error) {
		return ListToolsResult{Tools: []Tool{}}, nil
	}

	client := New(server.URL, server.Client())
	_, err := client.ListTools(context.Background(), "", nil)
	require.NoError(t, err)

	require.Len(t, server.requests, 3) // init, notified, list
	listReq := server.requests[2]
	assert.Equal(t, "tools/list", listReq.Method)

	var paramsMap map[string]any
	json.Unmarshal(asRawMessage(listReq.Params), &paramsMap)
	assert.Equal(t, "session-12345", paramsMap["Mcp-Session-Id"])
}

func TestListTools_MetaPreservation(t *testing.T) {
	server := newMockMCPServer()
	defer server.Close()

	server.handlers["tools/list"] = func(params json.RawMessage) (any, error) {
		return ListToolsResult{
			Tools: []Tool{
				{
					Name:        "auth_tool",
					Description: "Tool with auth",
					InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
					Meta: map[string]any{
						"toolbox/authInvoke": []string{"oauth-scope"},
					},
				},
			},
		}, nil
	}

	client := New(server.URL, server.Client())
	manifest, err := client.ListTools(context.Background(), "", nil)
	require.NoError(t, err)

	tool, ok := manifest.Tools["auth_tool"]
	require.True(t, ok)
	assert.Equal(t, []string{"oauth-scope"}, tool.AuthRequired)
}

func TestGetTool_Success(t *testing.T) {
	server := newMockMCPServer()
	defer server.Close()

	server.handlers["tools/list"] = func(params json.RawMessage) (any, error) {
		return ListToolsResult{
			Tools: []Tool{
				{Name: "wanted", InputSchema: map[string]any{}},
				{Name: "unwanted", InputSchema: map[string]any{}},
			},
		}, nil
	}

	client := New(server.URL, server.Client())
	manifest, err := client.GetTool(context.Background(), "wanted", nil)
	require.NoError(t, err)
	assert.Contains(t, manifest.Tools, "wanted")
	assert.NotContains(t, manifest.Tools, "unwanted")
}

func TestInvokeTool_ErrorResult(t *testing.T) {
	server := newMockMCPServer()
	defer server.Close()

	server.handlers["tools/call"] = func(params json.RawMessage) (any, error) {
		return CallToolResult{
			Content: []TextContent{{Type: "text", Text: "Something went wrong"}},
			IsError: true,
		}, nil
	}

	client := New(server.URL, server.Client())
	_, err := client.InvokeTool(context.Background(), "tool", nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tool execution resulted in error")
}

func TestInvokeTool_RPCError(t *testing.T) {
	server := newMockMCPServer()
	defer server.Close()

	server.handlers["tools/call"] = func(params json.RawMessage) (any, error) {
		return nil, errors.New("internal server error")
	}

	client := New(server.URL, server.Client())
	_, err := client.InvokeTool(context.Background(), "tool", nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "internal server error")
}

func TestListTools_WithAuthHeaders(t *testing.T) {
	server := newMockMCPServer()
	defer server.Close()

	server.handlers["tools/list"] = func(params json.RawMessage) (any, error) {
		return ListToolsResult{Tools: []Tool{}}, nil
	}

	client := New(server.URL, server.Client())
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "secret"})
	headers := map[string]oauth2.TokenSource{"Authorization": ts}

	_, err := client.ListTools(context.Background(), "", headers)
	require.NoError(t, err)
}

func TestProtocolVersionMismatch(t *testing.T) {
	server := newMockMCPServer()
	defer server.Close()

	server.handlers["initialize"] = func(params json.RawMessage) (any, error) {
		return InitializeResult{
			ProtocolVersion: "2099-01-01",
			Capabilities:    ServerCapabilities{Tools: map[string]any{}},
			ServerInfo:      Implementation{Name: "futuristic", Version: "1"},
			McpSessionId:    "s1",
		}, nil
	}

	client := New(server.URL, server.Client())
	err := client.EnsureInitialized(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "MCP version mismatch")
}

func TestInitialization_MissingCapabilities(t *testing.T) {
	server := newMockMCPServer()
	defer server.Close()

	server.handlers["initialize"] = func(params json.RawMessage) (any, error) {
		return InitializeResult{
			ProtocolVersion: ProtocolVersion,
			ServerInfo:      Implementation{Name: "bad", Version: "1"},
			McpSessionId:    "s1",
			// Tools capability missing
		}, nil
	}

	client := New(server.URL, server.Client())
	err := client.EnsureInitialized(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not support the 'tools' capability")
}

// --- Error Path Tests ---

func TestRequest_NetworkError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := server.URL
	server.Close()

	client := New(url, server.Client())
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

	client := New(server.URL, server.Client())
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

	client := New(server.URL, server.Client())
	_, err := client.ListTools(context.Background(), "", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "response unmarshal failed")
}

func TestRequest_NewRequestError(t *testing.T) {
	client := New("http://bad\nurl.com", http.DefaultClient)
	_, err := client.ListTools(context.Background(), "", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "create request failed")
}

func TestRequest_MarshalError(t *testing.T) {
	server := newMockMCPServer()
	defer server.Close()
	client := New(server.URL, server.Client())

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

	server.handlers["tools/list"] = func(params json.RawMessage) (any, error) {
		return ListToolsResult{Tools: []Tool{}}, nil
	}

	client := New(server.URL, server.Client())
	_, err := client.GetTool(context.Background(), "missing", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestListTools_InitFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := server.URL
	server.Close()

	client := New(url, server.Client())
	_, err := client.ListTools(context.Background(), "", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "http request failed")
}

// --- Extended Coverage Tests ---

type failingTokenSource struct{}

func (f *failingTokenSource) Token() (*oauth2.Token, error) {
	return nil, errors.New("token failure")
}

func TestHeaders_ResolutionError(t *testing.T) {
	// Fix: Use mock server to pass initialization so we hit the header resolution logic
	server := newMockMCPServer()
	defer server.Close()

	client := New(server.URL, server.Client())
	headers := map[string]oauth2.TokenSource{"auth": &failingTokenSource{}}

	// ListTools: EnsureInitialized succeeds, then header resolution fails
	_, err := client.ListTools(context.Background(), "", headers)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "token failure")

	// InvokeTool: EnsureInitialized succeeds, then header resolution fails
	_, err = client.InvokeTool(context.Background(), "tool", nil, headers)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "token failure")
}

func TestInit_NotificationFailure(t *testing.T) {
	// Fix: Use a custom server that returns 500 for the notification specifically.
	// doRPC swallows JSON-RPC error bodies for notifications (dest=nil), so we must rely on HTTP status codes.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req JSONRPCRequest
		// Read body to clear buffer, though we just check fields
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &req)

		if req.Method == "initialize" {
			// Success
			resp := JSONRPCResponse{
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

	client := New(server.URL, server.Client())
	err := client.EnsureInitialized(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestInvokeTool_ComplexContent(t *testing.T) {
	server := newMockMCPServer()
	defer server.Close()

	server.handlers["tools/call"] = func(params json.RawMessage) (any, error) {
		return CallToolResult{
			Content: []TextContent{
				{Type: "text", Text: "Part 1 "},
				{Type: "image", Text: "base64data"}, // Should be ignored based on text logic
				{Type: "text", Text: "Part 2"},
			},
		}, nil
	}

	client := New(server.URL, server.Client())
	res, err := client.InvokeTool(context.Background(), "t", nil, nil)
	require.NoError(t, err)
	// Only text types should be concatenated
	assert.Equal(t, "Part 1 Part 2", res)
}

func TestInvokeTool_EmptyResult(t *testing.T) {
	server := newMockMCPServer()
	defer server.Close()

	server.handlers["tools/call"] = func(params json.RawMessage) (any, error) {
		return CallToolResult{
			Content: []TextContent{},
		}, nil
	}

	client := New(server.URL, server.Client())
	res, err := client.InvokeTool(context.Background(), "t", nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "null", res)
}

func TestDoRPC_204_NoContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := New(server.URL, server.Client())
	err := client.sendNotification(context.Background(), "test", nil)
	require.NoError(t, err)
}

func TestListTools_ErrorOnEmptyName(t *testing.T) {
	server := newMockMCPServer()
	defer server.Close()

	server.handlers["tools/list"] = func(params json.RawMessage) (any, error) {
		return ListToolsResult{
			Tools: []Tool{
				{Name: "valid", InputSchema: map[string]any{}},
				{Name: "", InputSchema: map[string]any{}},
			},
		}, nil
	}

	client := New(server.URL, server.Client())
	_, err := client.ListTools(context.Background(), "", nil)

	// Assert that we get an error now
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing 'name' field")
}
