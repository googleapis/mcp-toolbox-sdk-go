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

	"github.com/googleapis/mcp-toolbox-sdk-go/core/transport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListToolsAndHeaders(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DRAFT-2026-v1", r.Header.Get("MCP-Protocol-Version"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var req map[string]any
		require.NoError(t, json.Unmarshal(body, &req))
		assert.Equal(t, "tools/list", req["method"])

		params := req["params"].(map[string]any)
		meta := params["_meta"].(map[string]any)
		assert.Equal(t, "DRAFT-2026-v1", meta["io.modelcontextprotocol/protocolVersion"])

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

func TestInvokeToolAndHeaders(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DRAFT-2026-v1", r.Header.Get("MCP-Protocol-Version"))

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
		assert.Equal(t, "DRAFT-2026-v1", r.Header.Get("MCP-Protocol-Version"))
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
		assert.Equal(t, "DRAFT-2026-v1", r.Header.Get("MCP-Protocol-Version"))
		assert.Equal(t, "prompts/get", r.Header.Get("Mcp-Method"))
		assert.Equal(t, "test_prompt", r.Header.Get("Mcp-Name"))

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":"1","result":{}}`))
	}))
	defer ts.Close()

	tr, err := New(ts.URL, ts.Client(), "test-client", "1.0.0")
	require.NoError(t, err)

	reqPayload := map[string]any{
		"jsonrpc": "2.0",
		"id":      "1",
		"method":  "prompts/get",
		"params": map[string]any{
			"name": "test_prompt",
		},
	}

	err = tr.sendRequest(context.Background(), ts.URL, reqPayload, nil, nil, "prompts/get", "test_prompt")
	require.NoError(t, err)
}

func TestSendRequest_AddsMcpNameHeaderForResourcesRead(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DRAFT-2026-v1", r.Header.Get("MCP-Protocol-Version"))
		assert.Equal(t, "resources/read", r.Header.Get("Mcp-Method"))
		assert.Equal(t, "file:///test.txt", r.Header.Get("Mcp-Name"))

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":"1","result":{}}`))
	}))
	defer ts.Close()

	tr, err := New(ts.URL, ts.Client(), "test-client", "1.0.0")
	require.NoError(t, err)

	reqPayload := map[string]any{
		"jsonrpc": "2.0",
		"id":      "1",
		"method":  "resources/read",
		"params": map[string]any{
			"uri": "file:///test.txt",
		},
	}

	err = tr.sendRequest(context.Background(), ts.URL, reqPayload, nil, nil, "resources/read", "file:///test.txt")
	require.NoError(t, err)
}
