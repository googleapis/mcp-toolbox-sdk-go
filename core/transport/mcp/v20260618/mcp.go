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

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/google/uuid"
	"github.com/googleapis/mcp-toolbox-sdk-go/core/transport"
	"github.com/googleapis/mcp-toolbox-sdk-go/core/transport/mcp"
)

const (
	ProtocolVersion = "DRAFT-2026-v1"
)

// Ensure that McpTransport implements the Transport interface.
var _ transport.Transport = &McpTransport{}

// McpTransport implements the MCP DRAFT-2026-v1 protocol (Stateless MCP).
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
	if clientVersion == "" {
		clientVersion = mcp.SDKVersion
	}

	t := &McpTransport{
		BaseMcpTransport: baseTransport,
		protocolVersion:  ProtocolVersion,
		clientName:       clientName,
		clientVersion:    clientVersion,
	}
	t.HandshakeHook = t.initializeSession

	return t, nil
}

func (t *McpTransport) getMeta() map[string]any {
	return map[string]any{
		"protocolVersion": t.protocolVersion,
		"clientInfo": map[string]any{
			"name":    t.clientName,
			"version": t.clientVersion,
		},
		"clientCapabilities": map[string]any{},
	}
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
	params := map[string]any{
		"_meta": t.getMeta(),
	}
	if err := t.sendRequest(ctx, requestURL, "tools/list", params, headers, &result); err != nil {
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
		Meta:      t.getMeta(),
	}

	var result callToolResult
	if err := t.sendRequest(ctx, t.BaseURL(), "tools/call", params, headers, &result); err != nil {
		return "", fmt.Errorf("failed to invoke tool '%s': %w", toolName, err)
	}

	if result.IsError {
		return "", fmt.Errorf("tool execution resulted in error")
	}

	baseContent := make([]mcp.ToolContent, len(result.Content))
	for i, item := range result.Content {
		baseContent[i] = mcp.ToolContent{
			Type: item.Type,
			Text: item.Text,
		}
	}

	output := t.ProcessToolResultContent(baseContent)

	return output, nil
}

// initializeSession is a no-op in stateless MCP
func (t *McpTransport) initializeSession(ctx context.Context, headers map[string]string) error {
	// Stateless MCP does not handshake. We assume server version is latest or unknown.
	t.ServerVersion = "unknown"
	return nil
}

func prepareHeaders(method string, params any, headers map[string]string) map[string]string {
	if headers == nil {
		headers = make(map[string]string)
	}
	newHeaders := make(map[string]string)
	for k, v := range headers {
		newHeaders[k] = v
	}
	newHeaders["Mcp-Method"] = method
	switch p := params.(type) {
	case callToolRequestParams:
		if method == "tools/call" || method == "prompts/get" {
			newHeaders["Mcp-Name"] = p.Name
		}
	case map[string]any:
		switch method {
		case "tools/call", "prompts/get":
			if name, ok := p["name"].(string); ok {
				newHeaders["Mcp-Name"] = name
			}
		case "resources/read":
			if uri, ok := p["uri"].(string); ok {
				newHeaders["Mcp-Name"] = uri
			}
		}
	}
	return newHeaders
}

// sendRequest sends a standard JSON-RPC request to the server.
func (t *McpTransport) sendRequest(ctx context.Context, url string, method string, params any, headers map[string]string, dest any) error {
	headers = prepareHeaders(method, params, headers)
	req := jsonRPCRequest{
		JSONRPC: "2.0",
		Method:  method,
		ID:      uuid.New().String(),
		Params:  params,
	}
	return t.doRPC(ctx, url, req, headers, dest)
}

// doRPC performs the low-level HTTP POST and handles JSON-RPC wrapping/unwrapping.
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
	// Set Accept header, we only accept application/json
	httpReq.Header.Set("Accept", "application/json")
	// DRAFT-2026-v1 Specific: Inject Protocol Version Header
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
	} else if resp.StatusCode == http.StatusBadRequest {
		// Possibly a ProtocolNegotiationError
		body, _ := io.ReadAll(resp.Body)
		var rpcResp jsonRPCResponse
		if err := json.Unmarshal(body, &rpcResp); err == nil && rpcResp.Error != nil {
			if rpcResp.Error.Code == -32004 {
				data, ok := rpcResp.Error.Data.(map[string]any)
				if ok {
					if supported, ok := data["supported"].([]any); ok && len(supported) > 0 {
						if fallbackStr, ok := supported[0].(string); ok {
							return &transport.ProtocolNegotiationError{FallbackVersion: fallbackStr}
						}
					}
				}
			}
			return fmt.Errorf("MCP request failed with code %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
		}
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	} else {
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
		if rpcResp.Error.Code == -32004 {
			data, ok := rpcResp.Error.Data.(map[string]any)
			if ok {
				if supported, ok := data["supported"].([]any); ok && len(supported) > 0 {
					if fallbackStr, ok := supported[0].(string); ok {
						return &transport.ProtocolNegotiationError{FallbackVersion: fallbackStr}
					}
				}
			}
		}
		return fmt.Errorf("MCP request failed with code %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	// Decode Result into specific struct
	resultBytes, _ := json.Marshal(rpcResp.Result)
	if err := json.Unmarshal(resultBytes, dest); err != nil {
		return fmt.Errorf("failed to parse result data: %w", err)
	}

	return nil
}
