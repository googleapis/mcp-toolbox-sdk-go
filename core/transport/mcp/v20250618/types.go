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

package v20250618

import "encoding/json"

// jsonRPCRequest represents a standard JSON-RPC 2.0 request.
type jsonRPCRequest struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	ID      any    `json:"id,omitempty"`
	Params  any    `json:"params,omitempty"`
}

// jsonRPCNotification represents a standard JSON-RPC 2.0 notification (no ID).
type jsonRPCNotification struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
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

// implementation describes the name and version of the client.
type implementation struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// clientCapabilities describes the features supported by the client.
type clientCapabilities map[string]any

// serverCapabilities describes the features supported by the server.
type serverCapabilities struct {
	Prompts map[string]any `json:"prompts,omitempty"`
	Tools   map[string]any `json:"tools,omitempty"`
}

// initializeRequestParams holds the parameters for the 'initialize' handshake.
type initializeRequestParams struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    clientCapabilities `json:"capabilities"`
	ClientInfo      implementation     `json:"clientInfo"`
}

// initializeResult holds the response from the 'initialize' handshake.
type initializeResult struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    serverCapabilities `json:"capabilities"`
	ServerInfo      implementation     `json:"serverInfo"`
	Instructions    string             `json:"instructions,omitempty"`
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
