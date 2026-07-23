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

package v20260618_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	v20260618 "github.com/googleapis/mcp-toolbox-sdk-go/core/transport/mcp/v20260618"
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

	tr, err := v20260618.New(ts.URL, ts.Client(), "test-client", "1.0.0")
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

	tr, err := v20260618.New(ts.URL, ts.Client(), "test-client", "1.0.0")
	require.NoError(t, err)

	res, err := tr.InvokeTool(context.Background(), "test_tool", nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "hello", res)
}

func TestPrepareHeadersMcpName(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "mcp", r.Header.Get("X-Goog-Toolbox-Target-Format"))

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

	tr, err := v20260618.New(ts.URL, ts.Client(), "test-client", "1.0.0")
	require.NoError(t, err)

	res, err := tr.InvokeTool(context.Background(), "my_tool", nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "ok", res)
}
