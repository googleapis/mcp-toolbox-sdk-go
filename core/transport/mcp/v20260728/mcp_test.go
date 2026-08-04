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

package v20260728

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/googleapis/mcp-toolbox-sdk-go/core/transport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListToolsAndHeaders(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "2026-07-28", r.Header.Get("MCP-Protocol-Version"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "application/json", r.Header.Get("Accept"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var req map[string]any
		require.NoError(t, json.Unmarshal(body, &req))
		assert.Equal(t, "tools/list", req["method"])

		params := req["params"].(map[string]any)
		meta := params["_meta"].(map[string]any)
		assert.Equal(t, "2026-07-28", meta["io.modelcontextprotocol/protocolVersion"])

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"jsonrpc": "2.0",
			"id": "1",
			"result": {
				"tools": [
					{
						"name": "test_tool",
						"description": "A test tool"
					}
				]
			}
		}`))
	}))
	defer ts.Close()

	tr, err := New(ts.URL, ts.Client(), "test-client", "1.0.0")
	require.NoError(t, err)

	manifest, err := tr.ListTools(context.Background(), "", nil)
	require.NoError(t, err)
	assert.Contains(t, manifest.Tools, "test_tool")
}

func TestListTools_ServerVersionFromMetadata(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"jsonrpc": "2.0",
			"id": "1",
			"result": {
				"_meta": {
					"io.modelcontextprotocol/serverInfo": {
						"name": "test-server",
						"version": "1.2.3"
					}
				},
				"tools": [
					{
						"name": "test_tool"
					}
				]
			}
		}`))
	}))
	defer ts.Close()

	tr, err := New(ts.URL, ts.Client(), "test-client", "1.0.0")
	require.NoError(t, err)

	manifest, err := tr.ListTools(context.Background(), "", nil)
	require.NoError(t, err)
	assert.Equal(t, "1.2.3", manifest.ServerVersion)
	assert.Equal(t, "1.2.3", tr.ServerVersion)
}


func TestInvokeToolAndHeaders(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "2026-07-28", r.Header.Get("MCP-Protocol-Version"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var req map[string]any
		require.NoError(t, json.Unmarshal(body, &req))
		assert.Equal(t, "tools/call", req["method"])

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"jsonrpc": "2.0",
			"id": "1",
			"result": {
				"content": [
					{"type": "text", "text": "hello"}
				]
			}
		}`))
	}))
	defer ts.Close()

	tr, err := New(ts.URL, ts.Client(), "test-client", "1.0.0")
	require.NoError(t, err)

	res, err := tr.InvokeTool(context.Background(), "test_tool", nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "hello", res)
}

func TestInvokeTool_NilArgumentsSerializedAsObject(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var req map[string]any
		require.NoError(t, json.Unmarshal(body, &req))
		params, ok := req["params"].(map[string]any)
		require.True(t, ok)
		args, exists := params["arguments"]
		require.True(t, exists, "params.arguments should exist")
		require.NotNil(t, args, "params.arguments must not be null")
		_, isMap := args.(map[string]any)
		require.True(t, isMap, "params.arguments must be an object map")

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"jsonrpc": "2.0",
			"id": "1",
			"result": {
				"content": [{"type": "text", "text": "ok"}]
			}
		}`))
	}))
	defer ts.Close()

	tr, err := New(ts.URL, ts.Client(), "test-client", "1.0.0")
	require.NoError(t, err)

	res, err := tr.InvokeTool(context.Background(), "test_tool", nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "ok", res)
}

func TestPrepareHeadersMcpName(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "2026-07-28", r.Header.Get("MCP-Protocol-Version"))
		assert.Equal(t, "tools/call", r.Header.Get("Mcp-Method"))
		assert.Equal(t, "my_tool", r.Header.Get("Mcp-Name"))

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"jsonrpc": "2.0",
			"id": "1",
			"result": {
				"content": [
					{"type": "text", "text": "ok"}
				]
			}
		}`))
	}))
	defer ts.Close()

	tr, err := New(ts.URL, ts.Client(), "test-client", "1.0.0")
	require.NoError(t, err)

	res, err := tr.InvokeTool(context.Background(), "my_tool", nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "ok", res)
}

func TestToolsList_UsesServerInfoFromMeta(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"jsonrpc": "2.0",
			"id": "1",
			"result": {
				"resultType": "complete",
				"tools": [{"name": "sample_tool", "description": "Sample"}],
				"_meta": {
					"io.modelcontextprotocol/serverInfo": {
						"name": "ToolboxServer",
						"version": "2.5.0"
					}
				}
			}
		}`))
	}))
	defer ts.Close()

	tr, err := New(ts.URL, ts.Client(), "test-client", "1.0.0")
	require.NoError(t, err)

	manifest, err := tr.ListTools(context.Background(), "", nil)
	require.NoError(t, err)
	assert.Equal(t, "2.5.0", manifest.ServerVersion)
	assert.Contains(t, manifest.Tools, "sample_tool")
}

func TestToolsList_ResultMetaOptionalServerInfo(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"jsonrpc": "2.0",
			"id": "1",
			"result": {
				"tools": [{"name": "sample_tool", "description": "Sample"}]
			}
		}`))
	}))
	defer ts.Close()

	tr, err := New(ts.URL, ts.Client(), "test-client", "1.0.0")
	require.NoError(t, err)

	manifest, err := tr.ListTools(context.Background(), "", nil)
	require.NoError(t, err)
	assert.Equal(t, "", manifest.ServerVersion)
	assert.Contains(t, manifest.Tools, "sample_tool")
}

func TestResultTypeParsingAndFallback(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"jsonrpc": "2.0",
			"id": "1",
			"result": {
				"content": [{"type": "text", "text": "test output"}]
			}
		}`))
	}))
	defer ts.Close()

	tr, err := New(ts.URL, ts.Client(), "test-client", "1.0.0")
	require.NoError(t, err)

	res, err := tr.InvokeTool(context.Background(), "test_tool", nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "test output", res)
}

func TestListTools_ParsesInputSchemaParameters(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"jsonrpc": "2.0",
			"id": "1",
			"result": {
				"tools": [
					{
						"name": "param_tool",
						"description": "Tool with params",
						"inputSchema": {
							"type": "object",
							"properties": {
								"location": {
									"type": "string",
									"description": "City name"
								}
							},
							"required": ["location"]
						}
					}
				]
			}
		}`))
	}))
	defer ts.Close()

	tr, err := New(ts.URL, ts.Client(), "test-client", "1.0.0")
	require.NoError(t, err)

	manifest, err := tr.ListTools(context.Background(), "", nil)
	require.NoError(t, err)
	require.Contains(t, manifest.Tools, "param_tool")

	toolSchema := manifest.Tools["param_tool"]
	require.NotEmpty(t, toolSchema.Parameters, "expected parameters to be parsed from inputSchema, but got empty slice")
	assert.Equal(t, "location", toolSchema.Parameters[0].Name)
	assert.Equal(t, "string", toolSchema.Parameters[0].Type)
	assert.True(t, toolSchema.Parameters[0].Required)
}

func TestJSONRPCError_HTTP200_ProtocolNegotiation(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"jsonrpc": "2.0",
			"id": "1",
			"error": {
				"code": -32022,
				"message": "unsupported protocol version",
				"data": {
					"supported": ["2025-11-25"]
				}
			}
		}`))
	}))
	defer ts.Close()

	tr, err := New(ts.URL, ts.Client(), "test-client", "1.0.0")
	require.NoError(t, err)

	_, err = tr.ListTools(context.Background(), "", nil)
	require.Error(t, err)

	var negErr *transport.ProtocolNegotiationError
	require.True(t, errors.As(err, &negErr), "expected ProtocolNegotiationError for HTTP 200 RPC error -32022")
	assert.Equal(t, "2025-11-25", negErr.FallbackVersion)
}

func TestSendRequest_AddsMcpNameHeaderForPromptsGet(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "2026-07-28", r.Header.Get("MCP-Protocol-Version"))
		assert.Equal(t, "prompts/get", r.Header.Get("Mcp-Method"))
		assert.Equal(t, "test_prompt", r.Header.Get("Mcp-Name"))

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":"1","result":{}}`))
	}))
	defer ts.Close()

	tr, err := New(ts.URL, ts.Client(), "test-client", "1.0.0")
	require.NoError(t, err)

	params := map[string]any{
		"name": "test_prompt",
	}

	err = tr.sendRequest(context.Background(), ts.URL, "prompts/get", params, nil, nil)
	require.NoError(t, err)
}


func TestRequestID_GeneratesUniqueUUIDs(t *testing.T) {
	var capturedIDs []string

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var req map[string]any
		require.NoError(t, json.Unmarshal(body, &req))

		reqID, ok := req["id"].(string)
		require.True(t, ok, "request ID should be a string")
		require.NotEmpty(t, reqID, "request ID should not be empty")
		require.NotEqual(t, "1", reqID, "request ID should not be hardcoded '1'")

		capturedIDs = append(capturedIDs, reqID)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":"` + reqID + `","result":{"tools":[]}}`))
	}))
	defer ts.Close()

	tr, err := New(ts.URL, ts.Client(), "test-client", "1.0.0")
	require.NoError(t, err)

	_, err = tr.ListTools(context.Background(), "", nil)
	require.NoError(t, err)

	_, err = tr.ListTools(context.Background(), "", nil)
	require.NoError(t, err)

	require.Len(t, capturedIDs, 2)
	assert.NotEqual(t, capturedIDs[0], capturedIDs[1], "consecutive request IDs must be unique UUIDs")
}

func TestListTools_ReturnsErrorOnMissingToolName(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"jsonrpc": "2.0",
			"id": "1",
			"result": {
				"tools": [
					{"description": "Missing name field"}
				]
			}
		}`))
	}))
	defer ts.Close()

	tr, err := New(ts.URL, ts.Client(), "test-client", "1.0.0")
	require.NoError(t, err)

	_, err = tr.ListTools(context.Background(), "", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing 'name' field")
}

func TestInvokeTool_ErrorResult(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"jsonrpc": "2.0",
			"id": "1",
			"result": {
				"content": [{"type": "text", "text": "Something went wrong"}],
				"isError": true
			}
		}`))
	}))
	defer ts.Close()

	tr, err := New(ts.URL, ts.Client(), "test-client", "1.0.0")
	require.NoError(t, err)

	_, err = tr.InvokeTool(context.Background(), "tool", nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tool execution resulted in error")
}

func TestInvokeTool_RPCError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`Internal Server Error`))
	}))
	defer ts.Close()

	tr, err := New(ts.URL, ts.Client(), "test-client", "1.0.0")
	require.NoError(t, err)

	_, err = tr.InvokeTool(context.Background(), "tool", nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to invoke tool 'tool':")
}

func TestGetTool_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"jsonrpc": "2.0",
			"id": "1",
			"result": {
				"tools": [
					{"name": "tool_a", "description": "Tool A"},
					{"name": "tool_b", "description": "Tool B"}
				]
			}
		}`))
	}))
	defer ts.Close()

	tr, err := New(ts.URL, ts.Client(), "test-client", "1.0.0")
	require.NoError(t, err)

	manifest, err := tr.GetTool(context.Background(), "tool_a", nil)
	require.NoError(t, err)
	assert.Contains(t, manifest.Tools, "tool_a")
	assert.NotContains(t, manifest.Tools, "tool_b")
}

func TestGetTool_NotFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"jsonrpc": "2.0",
			"id": "1",
			"result": {
				"tools": []
			}
		}`))
	}))
	defer ts.Close()

	tr, err := New(ts.URL, ts.Client(), "test-client", "1.0.0")
	require.NoError(t, err)

	_, err = tr.GetTool(context.Background(), "missing_tool", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestListTools_WithToolset(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/mcp/my-toolset", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"jsonrpc": "2.0",
			"id": "1",
			"result": {
				"tools": [{"name": "tool_a"}]
			}
		}`))
	}))
	defer ts.Close()

	tr, err := New(ts.URL, ts.Client(), "test-client", "1.0.0")
	require.NoError(t, err)

	manifest, err := tr.ListTools(context.Background(), "my-toolset", nil)
	require.NoError(t, err)
	assert.Contains(t, manifest.Tools, "tool_a")
}

func TestInvokeTool_ComplexContent(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"jsonrpc": "2.0",
			"id": "1",
			"result": {
				"content": [
					{"type": "text", "text": "Part 1 "},
					{"type": "image", "text": "base64data"},
					{"type": "text", "text": "Part 2"}
				]
			}
		}`))
	}))
	defer ts.Close()

	tr, err := New(ts.URL, ts.Client(), "test-client", "1.0.0")
	require.NoError(t, err)

	res, err := tr.InvokeTool(context.Background(), "t", nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "Part 1 Part 2", res)
}

func TestInvokeTool_EmptyResult(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"jsonrpc": "2.0",
			"id": "1",
			"result": {
				"content": []
			}
		}`))
	}))
	defer ts.Close()

	tr, err := New(ts.URL, ts.Client(), "test-client", "1.0.0")
	require.NoError(t, err)

	res, err := tr.InvokeTool(context.Background(), "t", nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "null", res)
}

func TestInvokeTool_ContentProcessing_Scenarios(t *testing.T) {
	t.Run("Multiple JSON Objects (Merge to Array)", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"jsonrpc": "2.0",
				"id": "1",
				"result": {
					"content": [
						{"type": "text", "text": "{\"foo\":\"bar\"}"},
						{"type": "text", "text": "{\"foo\":\"quux\"}"}
					]
				}
			}`))
		}))
		defer ts.Close()

		tr, err := New(ts.URL, ts.Client(), "test-client", "1.0.0")
		require.NoError(t, err)

		res, err := tr.InvokeTool(context.Background(), "tool", nil, nil)
		require.NoError(t, err)
		assert.Equal(t, `[{"foo":"bar"},{"foo":"quux"}]`, res)
	})
}

func TestNew_ClientVersion(t *testing.T) {
	t.Run("Explicit version", func(t *testing.T) {
		tr, err := New("http://example.com", nil, "test-client", "2.0.0")
		require.NoError(t, err)
		assert.Equal(t, "2.0.0", tr.clientVersion)
	})

	t.Run("Empty version uses SDKVersion", func(t *testing.T) {
		tr, err := New("http://example.com", nil, "test-client", "")
		require.NoError(t, err)
		assert.NotEmpty(t, tr.clientVersion)
	})
}

func TestSendNotification(t *testing.T) {
	var capturedMethod string
	var capturedMeta map[string]any

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "2026-07-28", r.Header.Get("MCP-Protocol-Version"))
		assert.Equal(t, "notifications/initialized", r.Header.Get("Mcp-Method"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var req map[string]any
		require.NoError(t, json.Unmarshal(body, &req))
		capturedMethod, _ = req["method"].(string)

		params, ok := req["params"].(map[string]any)
		if ok {
			capturedMeta, _ = params["_meta"].(map[string]any)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	tr, err := New(ts.URL, ts.Client(), "test-client", "1.0.0")
	require.NoError(t, err)

	err = tr.sendNotification(context.Background(), "notifications/initialized", map[string]any{}, nil)
	require.NoError(t, err)
	assert.Equal(t, "notifications/initialized", capturedMethod)
	require.NotNil(t, capturedMeta)
	assert.Equal(t, "2026-07-28", capturedMeta["io.modelcontextprotocol/protocolVersion"])
}

func TestJSONRPCError_ProtocolNegotiation(t *testing.T) {
	testCases := []struct {
		name             string
		code             int
		message          string
		httpStatus       int
		body             string
		supported        []string
		expectedFallback string
	}{
		{
			name:             "Code -32022 default fallback",
			code:             -32022,
			message:          "version mismatch",
			httpStatus:       http.StatusOK,
			expectedFallback: "2025-11-25",
		},
		{
			name:             "Code -32004 default fallback",
			code:             -32004,
			message:          "version mismatch",
			httpStatus:       http.StatusOK,
			expectedFallback: "2025-11-25",
		},
		{
			name:             "Message invalid protocol version",
			code:             -32000,
			message:          "invalid protocol version",
			httpStatus:       http.StatusOK,
			expectedFallback: "2025-11-25",
		},
		{
			name:             "Message unsupported protocol version",
			code:             -32000,
			message:          "unsupported protocol version",
			httpStatus:       http.StatusOK,
			expectedFallback: "2025-11-25",
		},
		{
			name:             "HTTP status error containing unsupported protocol version",
			httpStatus:       http.StatusBadRequest,
			body:             "unsupported protocol version",
			expectedFallback: "2025-11-25",
		},
		{
			name:             "Code -32022 with supported versions priority selection",
			code:             -32022,
			message:          "unsupported protocol version",
			httpStatus:       http.StatusOK,
			supported:        []string{"2025-03-26", "2025-06-18"},
			expectedFallback: "2025-06-18",
		},
		{
			name:             "Code -32022 with single supported version",
			code:             -32022,
			message:          "unsupported protocol version",
			httpStatus:       http.StatusOK,
			supported:        []string{"2025-11-25"},
			expectedFallback: "2025-11-25",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			status := tc.httpStatus
			if status == 0 {
				status = http.StatusOK
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(status)
				if status != http.StatusOK && tc.body != "" {
					_, _ = w.Write([]byte(tc.body))
					return
				}
				var data map[string]any
				if len(tc.supported) > 0 {
					supportedAnys := make([]any, len(tc.supported))
					for i, s := range tc.supported {
						supportedAnys[i] = s
					}
					data = map[string]any{"supported": supportedAnys}
				}
				resp := jsonRPCResponse{
					JSONRPC: "2.0",
					ID:      "1",
					Error: &jsonRPCError{
						Code:    tc.code,
						Message: tc.message,
						Data:    data,
					},
				}
				_ = json.NewEncoder(w).Encode(resp)
			}))
			defer server.Close()

			tr, err := New(server.URL, server.Client(), "test-client", "1.0.0")
			require.NoError(t, err)

			_, err = tr.ListTools(context.Background(), "", nil)
			require.Error(t, err)
			var negErr *transport.ProtocolNegotiationError
			require.True(t, errors.As(err, &negErr))
			assert.Equal(t, tc.expectedFallback, negErr.FallbackVersion)
		})
	}
}




