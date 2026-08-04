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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/googleapis/mcp-toolbox-sdk-go/core/transport"
	"github.com/googleapis/mcp-toolbox-sdk-go/core/transport/mcp"
	mcp20241105 "github.com/googleapis/mcp-toolbox-sdk-go/core/transport/mcp/v20241105"
	mcp20250326 "github.com/googleapis/mcp-toolbox-sdk-go/core/transport/mcp/v20250326"
	mcp20250618 "github.com/googleapis/mcp-toolbox-sdk-go/core/transport/mcp/v20250618"
	mcp20251125 "github.com/googleapis/mcp-toolbox-sdk-go/core/transport/mcp/v20251125"
)

const (
	ProtocolVersion = transport.MCPv20260728
)

// Ensure that McpTransport implements the Transport interface.
var _ transport.Transport = &McpTransport{}

// McpTransport implements the MCP 2026-07-28 protocol (Stateless MCP).
type McpTransport struct {
	*mcp.BaseMcpTransport
	protocolVersion string
	clientName      string
	clientVersion   string
}

// New creates a new version-specific transport instance.
func New(baseURL string, client *http.Client, clientName string, clientVersion string) (*McpTransport, error) {
	baseTransport, err := mcp.NewBaseTransport(baseURL, client)
	if err != nil {
		return nil, err
	}
	baseTransport.ProtocolVersion = ProtocolVersion
	if clientVersion == "" {
		clientVersion = mcp.SDKVersion
	}

	t := &McpTransport{
		BaseMcpTransport: baseTransport,
		protocolVersion:  ProtocolVersion,
		clientName:       clientName,
		clientVersion:    clientVersion,
	}
	// MCP 2026-07-28 is stateless (SEP-2575) so no initialization handshake is required.
	t.HandshakeHook = func(ctx context.Context, headers map[string]string) error {
		return nil
	}

	return t, nil
}

// ListTools fetches available tools
func (t *McpTransport) ListTools(ctx context.Context, toolsetName string, headers map[string]string) (*transport.ManifestSchema, error) {
	requestURL, err := mcp.AppendToolsetPath(t.BaseURL(), toolsetName)
	if err != nil {
		return nil, fmt.Errorf("failed to construct toolset URL: %w", err)
	}

	var result listToolsResult
	if err := t.sendRequest(ctx, requestURL, "tools/list", map[string]any{}, headers, &result); err != nil {
		return nil, fmt.Errorf("failed to list tools: %w", err)
	}

	if result.ResultType == "" {
		result.ResultType = "complete"
	}

	if result.Meta != nil && result.Meta.ServerInfo != nil && result.Meta.ServerInfo.Version != "" {
		t.ServerVersion = result.Meta.ServerInfo.Version
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
	if payload == nil {
		payload = make(map[string]any)
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

	output := t.ProcessToolResultContent(result.Content)

	return output, nil
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
// v20260728 (Stateless MCP): Injects '_meta' in params and sets 'Mcp-Method' and 'Mcp-Name' headers.
func (t *McpTransport) doRPC(ctx context.Context, url string, reqBody any, headers map[string]string, dest any) error {
	var method string

	meta := mcpMeta{
		ProtocolVersion: t.protocolVersion,
		ClientInfo: implementation{
			Name:    t.clientName,
			Version: t.clientVersion,
		},
		ClientCapabilities: clientCapabilities{},
	}

	toolName := extractMcpName(reqBody)

	switch r := reqBody.(type) {
	case jsonRPCRequest:
		method = r.Method
		r.Params = injectMeta(r.Params, meta)
		reqBody = r
	case jsonRPCNotification:
		method = r.Method
		r.Params = injectMeta(r.Params, meta)
		reqBody = r
	}

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
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("MCP-Protocol-Version", t.protocolVersion)
	if method != "" {
		httpReq.Header.Set("Mcp-Method", method)
	}
	if toolName != "" {
		httpReq.Header.Set("Mcp-Name", toolName)
	}

	// Apply resolved headers
	for k, v := range headers {
		httpReq.Header.Set(k, v)
	}

	resp, err := t.HTTPClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	supportedVersionsPriority := []string{
		ProtocolVersion,
		mcp20251125.ProtocolVersion,
		mcp20250618.ProtocolVersion,
		mcp20250326.ProtocolVersion,
		mcp20241105.ProtocolVersion,
	}

	checkRPCError := func(rpcErr *jsonRPCError) error {
		if rpcErr == nil {
			return nil
		}
		if rpcErr.Code == -32004 || rpcErr.Code == -32022 {
			if data, ok := rpcErr.Data.(map[string]any); ok {
				if supported, ok := data["supported"].([]any); ok && len(supported) > 0 {
					supportedSet := make(map[string]struct{})
					for _, s := range supported {
						if str, ok := s.(string); ok {
							supportedSet[str] = struct{}{}
						}
					}
					for _, v := range supportedVersionsPriority {
						if _, exists := supportedSet[v]; exists {
							return &transport.ProtocolNegotiationError{FallbackVersion: v}
						}
					}
				}
			}
			return &transport.ProtocolNegotiationError{FallbackVersion: mcp20251125.ProtocolVersion}
		}
		errMsgLower := strings.ToLower(rpcErr.Message)
		if strings.Contains(errMsgLower, "invalid protocol version") || strings.Contains(errMsgLower, "unsupported protocol version") {
			return &transport.ProtocolNegotiationError{FallbackVersion: mcp20251125.ProtocolVersion}
		}
		return fmt.Errorf("MCP request failed with code %d: %s", rpcErr.Code, rpcErr.Message)
	}

	if (resp.StatusCode == http.StatusAccepted || resp.StatusCode == http.StatusNoContent) && dest == nil {
		return nil // Valid notification success
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		var rpcResp jsonRPCResponse
		if err := json.Unmarshal(body, &rpcResp); err == nil && rpcResp.Error != nil {
			if err := checkRPCError(rpcResp.Error); err != nil {
				return err
			}
		}
		bodyStrLower := strings.ToLower(string(body))
		if strings.Contains(bodyStrLower, "invalid protocol version") || strings.Contains(bodyStrLower, "unsupported protocol version") {
			return &transport.ProtocolNegotiationError{FallbackVersion: mcp20251125.ProtocolVersion}
		}
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
		if err := checkRPCError(rpcResp.Error); err != nil {
			return err
		}
	}

	// Decode Result into specific struct
	resultBytes, _ := json.Marshal(rpcResp.Result)
	if err := json.Unmarshal(resultBytes, dest); err != nil {
		return fmt.Errorf("failed to parse result data: %w", err)
	}

	return nil
}

// injectMeta wraps or injects _meta into JSON-RPC params.
func injectMeta(params any, meta mcpMeta) map[string]any {
	res := make(map[string]any)
	if params != nil {
		b, err := json.Marshal(params)
		if err == nil {
			_ = json.Unmarshal(b, &res)
		}
	}
	res["_meta"] = meta
	return res
}

// extractMcpName extracts the tool name from request payload for the Mcp-Name header.
func extractMcpName(reqBody any) string {
	var params any
	switch r := reqBody.(type) {
	case jsonRPCRequest:
		params = r.Params
	case jsonRPCNotification:
		params = r.Params
	default:
		params = reqBody
	}

	if params == nil {
		return ""
	}

	switch p := params.(type) {
	case callToolRequestParams:
		return p.Name
	case map[string]any:
		if n, ok := p["name"].(string); ok {
			return n
		}
	}

	return ""
}
