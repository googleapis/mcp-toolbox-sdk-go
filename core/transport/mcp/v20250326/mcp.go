// Copyright 2025 Google LLC
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

package mcp20250326

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
	"golang.org/x/oauth2"
)

const (
	ProtocolVersion = "2025-03-26"
	ClientName      = "toolbox-go-sdk"
	ClientVersion   = "0.1.0"
)

// Ensure that McpTransport implements the Transport interface.
var _ transport.Transport = &McpTransport{}

// McpTransport implements the MCP v2025-03-26 protocol.
type McpTransport struct {
	*mcp.BaseMcpTransport

	protocolVersion string
	sessionId       string // Unique session ID for v2025-03-26
}

// New creates a new version-specific transport instance.
func New(baseURL string, client *http.Client) *McpTransport {
	t := &McpTransport{
		BaseMcpTransport: mcp.NewBaseTransport(baseURL, client),
		protocolVersion:  ProtocolVersion,
	}
	t.BaseMcpTransport.HandshakeHook = t.initializeSession

	return t
}

// ListTools fetches tools from the server and converts them to the ManifestSchema.
func (t *McpTransport) ListTools(ctx context.Context, toolsetName string, headers map[string]oauth2.TokenSource) (*transport.ManifestSchema, error) {
	if err := t.EnsureInitialized(ctx); err != nil {
		return nil, err
	}

	finalHeaders, err := t.resolveHeaders(headers)
	if err != nil {
		return nil, err
	}

	// Append toolset name to base URL if provided
	requestURL := t.BaseURL()
	if toolsetName != "" {
		requestURL += toolsetName
	}

	var result ListToolsResult
	if err := t.sendRequest(ctx, requestURL, "tools/list", map[string]any{}, finalHeaders, &result); err != nil {
		return nil, fmt.Errorf("failed to list tools: %w", err)
	}

	manifest := &transport.ManifestSchema{
		ServerVersion: t.ServerVersion,
		Tools:         make(map[string]transport.ToolSchema),
	}

	for i, mcpTool := range result.Tools {
		if mcpTool.Name == "" {
			return nil, fmt.Errorf("received invalid tool definition at index %d: missing 'name' field", i)
		}

		rawTool := map[string]any{
			"name":        mcpTool.Name,
			"description": mcpTool.Description,
			"inputSchema": mcpTool.InputSchema,
		}
		if mcpTool.Meta != nil {
			rawTool["_meta"] = mcpTool.Meta
		}

		toolSchema, err := t.ConvertToolDefinition(rawTool)
		if err != nil {
			return nil, fmt.Errorf("failed to convert schema for tool %s: %w", mcpTool.Name, err)
		}

		manifest.Tools[mcpTool.Name] = toolSchema
	}

	return manifest, nil
}

// GetTool fetches a single tool definition.
func (t *McpTransport) GetTool(ctx context.Context, toolName string, headers map[string]oauth2.TokenSource) (*transport.ManifestSchema, error) {
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

// InvokeTool calls a specific tool on the server and returns the text result.
func (t *McpTransport) InvokeTool(ctx context.Context, toolName string, args map[string]any, headers map[string]oauth2.TokenSource) (any, error) {
	if err := t.EnsureInitialized(ctx); err != nil {
		return "", err
	}

	finalHeaders, err := t.resolveHeaders(headers)
	if err != nil {
		return "", err
	}

	params := CallToolRequestParams{
		Name:      toolName,
		Arguments: args,
	}

	var result CallToolResult
	if err := t.sendRequest(ctx, t.BaseURL(), "tools/call", params, finalHeaders, &result); err != nil {
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
		return "null", nil
	}
	return output, nil
}

// initializeSession is the concrete implementation of the handshake hook.
func (t *McpTransport) initializeSession(ctx context.Context) error {
	params := InitializeRequestParams{
		ProtocolVersion: t.protocolVersion,
		Capabilities:    ClientCapabilities{},
		ClientInfo: Implementation{
			Name:    ClientName,
			Version: ClientVersion,
		},
	}

	var result InitializeResult

	if err := t.sendRequest(ctx, t.BaseURL(), "initialize", params, nil, &result); err != nil {
		return err
	}

	// Protocol Version Check
	if result.ProtocolVersion != t.protocolVersion {
		return fmt.Errorf("MCP version mismatch: client (%s) != server (%s)",
			t.protocolVersion, result.ProtocolVersion)
	}

	// Capabilities Check
	if result.Capabilities.Tools == nil {
		return fmt.Errorf("server does not support the 'tools' capability")
	}

	t.ServerVersion = result.ServerInfo.Version

	// Extract Session ID (v2025-03-26 specific)
	if result.McpSessionId == "" {
		return fmt.Errorf("server did not return a Mcp-Session-Id during initialization")
	}
	t.sessionId = result.McpSessionId

	// Confirm Handshake
	return t.sendNotification(ctx, "notifications/initialized", map[string]any{})
}

// resolveHeaders converts a map of TokenSources into standard HTTP headers (map[string]string).
func (t *McpTransport) resolveHeaders(sources map[string]oauth2.TokenSource) (map[string]string, error) {
	if sources == nil {
		return nil, nil
	}

	headers := make(map[string]string, len(sources))
	for headerKey, source := range sources {
		if source == nil {
			continue
		}

		token, err := source.Token()
		if err != nil {
			return nil, fmt.Errorf("failed to get token for header %s: %w", headerKey, err)
		}
		val := token.AccessToken

		headers[headerKey] = val
	}
	return headers, nil
}

// sendRequest sends a standard JSON-RPC request and injects the session ID if present.
func (t *McpTransport) sendRequest(ctx context.Context, url string, method string, params any, headers map[string]string, dest any) error {

	// Inject Session ID for non-initialize requests (v2025-03-26 specific)
	finalParams := params
	if method != "initialize" && t.sessionId != "" {
		paramBytes, _ := json.Marshal(params)
		var paramMap map[string]any
		if err := json.Unmarshal(paramBytes, &paramMap); err == nil {
			if paramMap == nil {
				paramMap = make(map[string]any)
			}
			paramMap["Mcp-Session-Id"] = t.sessionId
			finalParams = paramMap
		}
	}

	req := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  method,
		ID:      uuid.New().String(),
		Params:  finalParams,
	}
	return t.doRPC(ctx, url, req, headers, dest)
}

// sendNotification sends a standard JSON-RPC notification and injects the session ID if present.
func (t *McpTransport) sendNotification(ctx context.Context, method string, params any) error {

	// Inject Session ID (v2025-03-26 specific)
	finalParams := params
	if t.sessionId != "" {
		paramBytes, _ := json.Marshal(params)
		var paramMap map[string]any
		if err := json.Unmarshal(paramBytes, &paramMap); err == nil {
			if paramMap == nil {
				paramMap = make(map[string]any)
			}
			paramMap["Mcp-Session-Id"] = t.sessionId
			finalParams = paramMap
		}
	}

	req := JSONRPCNotification{
		JSONRPC: "2.0",
		Method:  method,
		Params:  finalParams,
	}
	return t.doRPC(ctx, t.BaseURL(), req, nil, nil)
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
	var rpcResp JSONRPCResponse
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
