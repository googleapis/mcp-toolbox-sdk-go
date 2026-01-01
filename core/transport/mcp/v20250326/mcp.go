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

package mcp20250326

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
	"golang.org/x/oauth2"
)

const (
	ProtocolVersion = "2025-03-26"
	ClientName      = "toolbox-go-sdk"
	ClientVersion   = mcp.SDKVersion
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
		var err error
		requestURL, err = url.JoinPath(requestURL, toolsetName)
		if err != nil {
			return nil, fmt.Errorf("failed to construct toolset URL: %w", err)
		}
	}

	var result listToolsResult
	if _, err := t.sendRequest(ctx, requestURL, "tools/list", map[string]any{}, finalHeaders, &result); err != nil {
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

// InvokeTool executes a tool
func (t *McpTransport) InvokeTool(ctx context.Context, toolName string, payload map[string]any, headers map[string]oauth2.TokenSource) (any, error) {
	if err := t.EnsureInitialized(ctx); err != nil {
		return "", err
	}

	finalHeaders, err := t.resolveHeaders(headers)
	if err != nil {
		return "", err
	}

	params := callToolRequestParams{
		Name:      toolName,
		Arguments: payload,
	}
	var result callToolResult
	if _, err := t.sendRequest(ctx, t.BaseURL(), "tools/call", params, finalHeaders, &result); err != nil {
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

// initializeSession performs the initial handshake and extracts the Session ID.
func (t *McpTransport) initializeSession(ctx context.Context) error {
	params := initializeRequestParams{
		ProtocolVersion: t.protocolVersion,
		Capabilities:    clientCapabilities{},
		ClientInfo: implementation{
			Name:    ClientName,
			Version: ClientVersion,
		},
	}
	var result initializeResult
	req := jsonRPCRequest{
		JSONRPC: "2.0",
		Method:  "initialize",
		ID:      uuid.New().String(),
		Params:  params,
	}

	// Capture headers to check for Session ID
	respHeaders, err := t.doRPC(ctx, t.BaseURL(), req, nil, &result)
	if err != nil {
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

	// Session ID Extraction: Check the Headers.
	sessionId := respHeaders.Get("Mcp-Session-Id")

	if sessionId == "" {
		return fmt.Errorf("server did not return a Mcp-Session-Id in the headers")
	}
	t.sessionId = sessionId

	// Confirm Handshake
	_, err = t.sendNotification(ctx, "notifications/initialized", map[string]any{})
	return err
}

// resolveHeaders converts a map of TokenSources into standard HTTP headers.
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
		headers[headerKey] = token.AccessToken
	}
	return headers, nil
}

// sendRequest sends a JSON-RPC request and injects the Session ID if active.
func (t *McpTransport) sendRequest(ctx context.Context, url string, method string, params any, headers map[string]string, dest any) (http.Header, error) {

	// Initialize headers map if it is nil
	if headers == nil {
		headers = make(map[string]string)
	}

	// Spec Requirement: Include Mcp-Session-Id in the HEADER for all subsequent requests
	if method != "initialize" && t.sessionId != "" {
		headers["Mcp-Session-Id"] = t.sessionId
	}

	// Construct the standard JSON-RPC request (Params are NOT modified)
	req := jsonRPCRequest{
		JSONRPC: "2.0",
		Method:  method,
		ID:      uuid.New().String(),
		Params:  params,
	}

	return t.doRPC(ctx, url, req, headers, dest)
}

// sendNotification sends a JSON-RPC notification and injects the Session ID if active.
func (t *McpTransport) sendNotification(ctx context.Context, method string, params any) (http.Header, error) {

	// Initialize headers map
	headers := make(map[string]string)

	// Spec Requirement: Inject Session ID as a HEADER
	if t.sessionId != "" {
		headers["Mcp-Session-Id"] = t.sessionId
	}

	// Construct the standard JSON-RPC notification
	req := jsonRPCNotification{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}

	// Pass the headers to doRPC
	return t.doRPC(ctx, t.BaseURL(), req, headers, nil)
}

// doRPC performs the HTTP POST, returns headers, and handles JSON-RPC wrapping.
func (t *McpTransport) doRPC(ctx context.Context, url string, reqBody any, headers map[string]string, dest any) (http.Header, error) {
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal failed: %w", err)
	}

	// Create Request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// Apply resolved headers
	for k, v := range headers {
		httpReq.Header.Set(k, v)
	}

	resp, err := t.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		// Continue to body parsing
	} else if (resp.StatusCode == http.StatusAccepted || resp.StatusCode == http.StatusNoContent) && dest == nil {
		return resp.Header, nil // Valid notification success
	} else {
		// Any other code, OR a 202/204 when we expected a result, is a failure.
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	if dest == nil {
		return resp.Header, nil
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body failed: %w", err)
	}
	var rpcResp jsonRPCResponse
	if err := json.Unmarshal(bodyBytes, &rpcResp); err != nil {
		return nil, fmt.Errorf("response unmarshal failed: %w", err)
	}

	// Check RPC Error
	if rpcResp.Error != nil {
		return nil, fmt.Errorf("MCP request failed with code %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	// Decode Result into specific struct
	resultBytes, _ := json.Marshal(rpcResp.Result)
	if err := json.Unmarshal(resultBytes, dest); err != nil {
		return nil, fmt.Errorf("failed to parse result data: %w", err)
	}

	return resp.Header, nil
}
