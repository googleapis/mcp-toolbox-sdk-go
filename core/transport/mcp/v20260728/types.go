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
	"encoding/json"

	"github.com/googleapis/mcp-toolbox-sdk-go/core/transport/mcp"
)

type Implementation struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type ClientCapabilities struct{}

type MCPMeta struct {
	ProtocolVersion    string             `json:"io.modelcontextprotocol/protocolVersion"`
	ClientInfo         Implementation     `json:"io.modelcontextprotocol/clientInfo"`
	ClientCapabilities ClientCapabilities `json:"io.modelcontextprotocol/clientCapabilities"`
}

type ListToolsParams struct {
	Meta MCPMeta `json:"_meta"`
}

type ListToolsRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      string          `json:"id"`
	Method  string          `json:"method"`
	Params  ListToolsParams `json:"params"`
}

type CallToolParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
	Meta      MCPMeta        `json:"_meta"`
}

type CallToolRequest struct {
	JSONRPC string         `json:"jsonrpc"`
	ID      string         `json:"id"`
	Method  string         `json:"method"`
	Params  CallToolParams `json:"params"`
}

type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      string          `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

type MCPResultMeta struct {
	ServerInfo *Implementation `json:"io.modelcontextprotocol/serverInfo,omitempty"`
}

type MCPResult struct {
	ResultType string         `json:"resultType,omitempty"`
	Meta       *MCPResultMeta `json:"_meta,omitempty"`
}

type ListToolsResult struct {
	MCPResult
	Tools []map[string]any `json:"tools"`
}

type CallToolResult struct {
	MCPResult
	Content []mcp.ToolContent `json:"content"`
	IsError bool              `json:"isError,omitempty"`
}
