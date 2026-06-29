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

package v20260618

import "encoding/json"

// jsonRPCRequest represents a standard JSON-RPC 2.0 request.
type jsonRPCRequest struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	ID      any    `json:"id,omitempty"`     // string or int
	Params  any    `json:"params,omitempty"` // map or struct
}

// jsonRPCResponse represents a standard JSON-RPC 2.0 response.
type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
}

// jsonRPCError represents the error object inside a JSON-RPC response.
type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// mcpTool represents a single tool definition from the server.
type mcpTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	InputSchema map[string]any `json:"inputSchema"`
	Meta        map[string]any `json:"_meta,omitempty"`
}

// listToolsResult holds the response from the 'tools/list' method.
type listToolsResult struct {
	Tools []mcpTool `json:"tools"`
}

// callToolRequestParams holds the parameters for the 'tools/call' method.
type callToolRequestParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
	Meta      map[string]any `json:"_meta,omitempty"`
}

// textContent represents a single text block in a tool's output.
type textContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// callToolResult holds the response from the 'tools/call' method.
type callToolResult struct {
	Content []textContent `json:"content"`
	IsError bool          `json:"isError"`
}
