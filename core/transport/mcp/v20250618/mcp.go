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

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"github.com/googleapis/mcp-toolbox-sdk-go/core/transport"
	"github.com/googleapis/mcp-toolbox-sdk-go/core/transport/mcp"
)

const (
	ProtocolVersion = "2025-06-18"
	ClientName      = "toolbox-go-sdk"
	ClientVersion   = mcp.SDKVersion
)

// Ensure that McpTransport implements the Transport interface.
var _ transport.Transport = &McpTransport{}

// McpTransport implements the MCP v2025-06-18 protocol.
type McpTransport struct {
	*mcp.BaseMcpTransport
	protocolVersion string
}

// New creates a new version-specific transport instance.
func New(baseURL string, client *http.Client) (*McpTransport, error) {
	baseTransport, err := mcp.NewBaseTransport(baseURL, client)
	if err != nil {
		return nil, err
	}
	t := &McpTransport{
		BaseMcpTransport: baseTransport,
		protocolVersion:  ProtocolVersion,
	}
	t.BaseMcpTransport.HandshakeHook = t.initializeSession

	return t, nil
}

// ListTools fetches available tools
func (t *McpTransport) ListTools(ctx context.Context, toolsetName string, headers map[string]string) (*transport.ManifestSchema, error) {
	if err := t.EnsureInitialized(ctx, headers); err != nil {
		return nil, err
	}

	requestURL := t.BaseURL()
	if toolsetName != "" {
		var err error
		requestURL, err = url.JoinPath(requestURL, toolsetName)
		if err != nil {
			return nil, fmt.Errorf("failed to construct toolset URL: %w", err)
		}
	}

	var result listToolsResult
	if err := t.sendRequest(ctx, requestURL, "tools/list", map[string]any{}, headers, &result); err != nil {
		return nil, fmt.Errorf("failed to list tools: %w", err)
	}

	manifest := &transport.ManifestSchema{
		ServerVersion: t.ServerVersion,
		Tools:         make(map[string]transport.ToolSchema),
	}

	for i, tool := range result.Tools {
		if tool.Name == "" {
			return nil, fmt.Errorf("received invalid tool definition at index %d: missing 'name' field", i)
		}

		rawTool := map[string]any{
			"name":        tool.Name,
			"description": tool.Description,
			"inputSchema": tool.InputSchema,
		}
		if tool.Meta != nil {
			rawTool["_meta"] = tool.Meta
		}

		toolSchema, err := t.ConvertToolDefinition(rawTool)
		if err != nil {
			return nil, fmt.Errorf("failed to convert schema for tool %s: %w", tool.Name, err)
		}

		manifest.Tools[tool.Name] = toolSchema
	}

	return manifest, nil
}

// GetTool fetches a single tool
func (t *McpTransport) GetTool(ctx context.Context, toolName string, headers map[string]string) (*transport.ManifestSchema, error) {
	manifest, err := t.ListTools(ctx, "", headers)
	if err != nil {
		return nil, err
	}

	tool, exists := manifest.Tools[toolName]
	if !exists {
		return nil, fmt.Errorf("tool '%s' not found", toolName)
	}

	return &transport.ManifestSchema{
		ServerVersion: manifest.ServerVersion,
		Tools:         map[string]transport.ToolSchema{toolName: tool},
	}, nil
}

// InvokeTool executes a tool
func (t *McpTransport) InvokeTool(ctx context.Context, toolName string, payload map[string]any, headers map[string]string) (any, error) {
	if err := t.EnsureInitialized(ctx, headers); err != nil {
		return "", err
	}
	params := callToolRequestParams{
		Name:      toolName,
		Arguments: payload,
	}

	var result callToolResult
	if err := t.sendRequest(ctx, t.BaseURL(), "tools/call", params, headers, &result); err != nil {
		return "", fmt.Errorf("failed to invoke tool '%s': %w", toolName, err)
	}

	if result.IsError {
		return "", fmt.Errorf("tool execution resulted in error")
	}

	// Concatenate all text content blocks
	var sb strings.Builder
	for _, content := range result.Content {
		if content.Type == "text" {
			sb.WriteString(content.Text)
		}
	}

	output := sb.String()
	if output == "" {
		// Return null if no text content found but not an error
		return "null", nil
	}
	return output, nil
}

// initializeSession performs the initial handshake with the server.
func (t *McpTransport) initializeSession(ctx context.Context, headers map[string]string) error {
	params := initializeRequestParams{
		ProtocolVersion: t.protocolVersion,
		Capabilities:    clientCapabilities{},
		ClientInfo: implementation{
			Name:    ClientName,
			Version: ClientVersion,
		},
	}

	var result initializeResult
	if err := t.sendRequest(ctx, t.BaseURL(), "initialize", params, headers, &result); err != nil {
		return err
	}

	// Protocol Version Check
	if result.ProtocolVersion != t.protocolVersion {
		return fmt.Errorf("MCP version mismatch: client (%s) != server (%s)", t.protocolVersion, result.ProtocolVersion)
	}

	// Capabilities Check
	if result.Capabilities.Tools == nil {
		return fmt.Errorf("server does not support the 'tools' capability")
	}

	t.ServerVersion = result.ServerInfo.Version

	// Confirm Handshake
	return t.sendNotification(ctx, "notifications/initialized", map[string]any{}, headers)
}

// sendRequest sends a standard JSON-RPC request to the server.
func (t *McpTransport) sendRequest(ctx context.Context, url string, method string, params any, headers map[string]string, dest any) error {
	req := jsonRPCRequest{
		JSONRPC: "2.0",
		Method:  method,
		ID:      uuid.New().String(),
		Params:  params,
	}
	return t.doRPC(ctx, url, req, headers, dest)
}

// sendNotification sends a standard JSON-RPC notification (no response expected).
func (t *McpTransport) sendNotification(ctx context.Context, method string, params any, headers map[string]string) error {
	req := jsonRPCNotification{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}
	return t.doRPC(ctx, t.BaseURL(), req, headers, nil)
}

// doRPC performs the low-level HTTP POST and handles JSON-RPC wrapping/unwrapping.
// v2025-06-18: Injects 'MCP-Protocol-Version' header.
func (t *McpTransport) doRPC(ctx context.Context, url string, reqBody any, headers map[string]string, dest any) error {
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal failed: %w", err)
	}

	// Create Request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("create request failed: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	// Set Accept header for MCP Spec 2025-03-26
	// Since SSE is not supported, we only accept application/json
	httpReq.Header.Set("Accept", "application/json")
	// v2025-06-18 Specific: Inject Protocol Version Header
	httpReq.Header.Set("MCP-Protocol-Version", t.protocolVersion)

	// Apply resolved headers
	for k, v := range headers {
		httpReq.Header.Set(k, v)
	}

	resp, err := t.HTTPClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		// Continue to body parsing
	} else if (resp.StatusCode == http.StatusAccepted || resp.StatusCode == http.StatusNoContent) && dest == nil {
		return nil // Valid notification success
	} else {
		// Any other code, OR a 202/204 when we expected a result, is a failure.
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	if dest == nil {
		return nil
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read body failed: %w", err)
	}

	// Decode RPC Envelope
	var rpcResp jsonRPCResponse
	if err := json.Unmarshal(bodyBytes, &rpcResp); err != nil {
		return fmt.Errorf("response unmarshal failed: %w", err)
	}

	// Check RPC Error
	if rpcResp.Error != nil {
		return fmt.Errorf("MCP request failed with code %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	// Decode Result into specific struct
	resultBytes, _ := json.Marshal(rpcResp.Result)
	if err := json.Unmarshal(resultBytes, dest); err != nil {
		return fmt.Errorf("failed to parse result data: %w", err)
	}

	return nil
}
