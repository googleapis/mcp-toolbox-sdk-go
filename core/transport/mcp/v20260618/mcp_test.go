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

package v20260618

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/googleapis/mcp-toolbox-sdk-go/core"
	"github.com/googleapis/mcp-toolbox-sdk-go/core/transport/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type capturedRequest struct {
	Body    jsonRPCRequest
	Headers http.Header
}

type mockMCPServer struct {
	*httptest.Server
	handlers map[string]func(params json.RawMessage) (any, error)
	requests []capturedRequest
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

		m.requests = append(m.requests, capturedRequest{
			Body:    req,
			Headers: r.Header.Clone(),
		})

		handler, ok := m.handlers[req.Method]
		if !ok {
			http.Error(w, "method not found", http.StatusNotFound)
			return
		}

		result, err := handler(req.Params.(json.RawMessage))
		if err != nil {
			// Mock protocol fallback error code
			if err.Error() == "fallback" {
				w.WriteHeader(http.StatusBadRequest)
				errResp := jsonRPCResponse{
					JSONRPC: "2.0",
					ID:      req.ID,
					Error: &jsonRPCError{
						Code:    -32004,
						Message: "Protocol fallback",
						Data: map[string]any{
							"supported": []any{"2025-11-25"},
						},
					},
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(errResp)
				return
			}
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

	return m
}

func asRawMessage(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

func TestListToolsAndHeaders(t *testing.T) {
	server := newMockMCPServer(t)
	defer server.Close()

	server.handlers["tools/list"] = func(params json.RawMessage) (any, error) {
		return listToolsResult{
			Tools: []mcpTool{
				{
					Name:        "get_weather",
					Description: "Get weather for a location",
					InputSchema: map[string]any{},
				},
			},
		}, nil
	}

	client, _ := New(server.URL, server.Client(), "test-client", "1.0.0")
	ctx := context.Background()

	manifest, err := client.ListTools(ctx, "", nil)
	require.NoError(t, err)
	require.NotNil(t, manifest)

	require.Len(t, server.requests, 1)
	req := server.requests[0]

	// Verify Headers
	assert.Equal(t, "DRAFT-2026-v1", req.Headers.Get("MCP-Protocol-Version"))
	assert.Equal(t, "tools/list", req.Headers.Get("Mcp-Method"))
	assert.Empty(t, req.Headers.Get("Mcp-Name"))

	// Verify _meta
	var params map[string]any
	json.Unmarshal(req.Body.Params.(json.RawMessage), &params)
	meta, ok := params["_meta"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "DRAFT-2026-v1", meta["protocolVersion"])
}

func TestInvokeToolAndHeaders(t *testing.T) {
	server := newMockMCPServer(t)
	defer server.Close()

	server.handlers["tools/call"] = func(params json.RawMessage) (any, error) {
		return callToolResult{
			Content: []textContent{
				{Type: "text", Text: "Echo"},
			},
			IsError: false,
		}, nil
	}

	client, _ := New(server.URL, server.Client(), "test-client", "1.0.0")
	ctx := context.Background()

	res, err := client.InvokeTool(ctx, "echo", map[string]any{}, nil)
	require.NoError(t, err)
	assert.Equal(t, "Echo", res)

	require.Len(t, server.requests, 1)
	req := server.requests[0]

	// Verify Headers
	assert.Equal(t, "DRAFT-2026-v1", req.Headers.Get("MCP-Protocol-Version"))
	assert.Equal(t, "tools/call", req.Headers.Get("Mcp-Method"))
	assert.Equal(t, "echo", req.Headers.Get("Mcp-Name"))

	// Verify _meta
	var params map[string]any
	json.Unmarshal(req.Body.Params.(json.RawMessage), &params)
	meta, ok := params["_meta"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "DRAFT-2026-v1", meta["protocolVersion"])
}

func TestProtocolFallback(t *testing.T) {
	server := newMockMCPServer(t)
	defer server.Close()

	server.handlers["tools/list"] = func(params json.RawMessage) (any, error) {
		return nil, errors.New("fallback")
	}

	client, _ := New(server.URL, server.Client(), "test-client", "1.0.0")
	ctx := context.Background()

	_, err := client.ListTools(ctx, "", nil)
	require.Error(t, err)

	var negErr *core.ProtocolNegotiationError
	require.True(t, errors.As(err, &negErr))
	assert.Equal(t, "2025-11-25", negErr.FallbackVersion)
}

func TestPrepareHeadersMcpName(t *testing.T) {
	// Test tools/call with struct
	headers1 := prepareHeaders("tools/call", callToolRequestParams{Name: "struct_tool"}, nil)
	assert.Equal(t, "struct_tool", headers1["Mcp-Name"])

	// Test tools/call with map
	headers2 := prepareHeaders("tools/call", map[string]any{"name": "map_tool"}, nil)
	assert.Equal(t, "map_tool", headers2["Mcp-Name"])

	// Test prompts/get with map
	headers3 := prepareHeaders("prompts/get", map[string]any{"name": "test_prompt"}, nil)
	assert.Equal(t, "test_prompt", headers3["Mcp-Name"])

	// Test resources/read with map
	headers4 := prepareHeaders("resources/read", map[string]any{"uri": "file:///test"}, nil)
	assert.Equal(t, "file:///test", headers4["Mcp-Name"])
}
