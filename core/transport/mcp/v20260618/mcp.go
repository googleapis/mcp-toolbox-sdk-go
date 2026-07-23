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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/googleapis/mcp-toolbox-sdk-go/core/transport"
	"github.com/googleapis/mcp-toolbox-sdk-go/core/transport/mcp"
	mcp20241105 "github.com/googleapis/mcp-toolbox-sdk-go/core/transport/mcp/v20241105"
	mcp20250326 "github.com/googleapis/mcp-toolbox-sdk-go/core/transport/mcp/v20250326"
	mcp20250618 "github.com/googleapis/mcp-toolbox-sdk-go/core/transport/mcp/v20250618"
	mcp20251125 "github.com/googleapis/mcp-toolbox-sdk-go/core/transport/mcp/v20251125"
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

	return &McpTransport{
		BaseMcpTransport: baseTransport,
		protocolVersion:  ProtocolVersion,
		clientName:       clientName,
		clientVersion:    clientVersion,
	}, nil
}

// GetTool fetches a single tool manifest.
func (t *McpTransport) GetTool(ctx context.Context, toolName string, headers map[string]string) (*transport.ManifestSchema, error) {
	manifest, err := t.ListTools(ctx, "", headers)
	if err != nil {
		return nil, err
	}
	if tool, exists := manifest.Tools[toolName]; exists {
		return &transport.ManifestSchema{
			ServerVersion: manifest.ServerVersion,
			Tools:         map[string]transport.ToolSchema{toolName: tool},
		}, nil
	}
	return nil, fmt.Errorf("tool '%s' not found in manifest", toolName)
}

// ListTools fetches available tools from the server.
func (t *McpTransport) ListTools(ctx context.Context, toolsetName string, headers map[string]string) (*transport.ManifestSchema, error) {
	var targetURL string
	var err error
	if toolsetName != "" {
		targetURL, err = url.JoinPath(t.BaseURL(), toolsetName)
		if err != nil {
			return nil, fmt.Errorf("failed to build toolset URL: %w", err)
		}
	} else {
		targetURL = t.BaseURL()
	}

	reqPayload := ListToolsRequest{
		JSONRPC: "2.0",
		ID:      "1",
		Method:  "tools/list",
		Params: ListToolsParams{
			Meta: MCPMeta{
				ProtocolVersion: t.protocolVersion,
				ClientInfo: Implementation{
					Name:    t.clientName,
					Version: t.clientVersion,
				},
				ClientCapabilities: ClientCapabilities{},
			},
		},
	}

	var result ListToolsResult
	if err := t.sendRequest(ctx, targetURL, reqPayload, &result, headers, "tools/list", ""); err != nil {
		return nil, err
	}

	toolsMap := make(map[string]transport.ToolSchema)
	for _, toolRaw := range result.Tools {
		toolSchema, err := t.ConvertToolDefinition(toolRaw)
		if err != nil {
			continue
		}
		if name, ok := toolRaw["name"].(string); ok {
			toolsMap[name] = toolSchema
		}
	}

	return &transport.ManifestSchema{
		ServerVersion: "1.0.0",
		Tools:         toolsMap,
	}, nil
}

// InvokeTool executes a tool on the server.
func (t *McpTransport) InvokeTool(ctx context.Context, toolName string, payload map[string]any, headers map[string]string) (any, error) {
	args := payload
	if args == nil {
		args = make(map[string]any)
	}

	reqPayload := CallToolRequest{
		JSONRPC: "2.0",
		ID:      "1",
		Method:  "tools/call",
		Params: CallToolParams{
			Name:      toolName,
			Arguments: args,
			Meta: MCPMeta{
				ProtocolVersion: t.protocolVersion,
				ClientInfo: Implementation{
					Name:    t.clientName,
					Version: t.clientVersion,
				},
				ClientCapabilities: ClientCapabilities{},
			},
		},
	}

	var result CallToolResult
	if err := t.sendRequest(ctx, t.BaseURL(), reqPayload, &result, headers, "tools/call", toolName); err != nil {
		return nil, err
	}

	if result.IsError {
		errMsg := "tool invocation returned error"
		if len(result.Content) > 0 {
			errMsg = result.Content[0].Text
		}
		return nil, fmt.Errorf("%s", errMsg)
	}

	baseContent := make([]mcp.ToolContent, len(result.Content))
	for i, item := range result.Content {
		baseContent[i] = mcp.ToolContent{
			Type: item.Type,
			Text: item.Text,
		}
	}

	return t.ProcessToolResultContent(baseContent), nil
}

var supportedVersionsPriority = []string{
	mcp20251125.ProtocolVersion,
	mcp20250618.ProtocolVersion,
	mcp20250326.ProtocolVersion,
	mcp20241105.ProtocolVersion,
}

func checkRPCError(rpcErr *jsonRPCError) error {
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

func (t *McpTransport) sendRequest(ctx context.Context, reqURL string, bodyPayload any, dest any, headers map[string]string, method string, mcpName string) error {
	jsonBytes, err := json.Marshal(bodyPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal request payload: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", reqURL, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("MCP-Protocol-Version", t.protocolVersion)
	if method != "" {
		httpReq.Header.Set("Mcp-Method", method)
	}
	if mcpName != "" {
		httpReq.Header.Set("Mcp-Name", mcpName)
	}
	for k, v := range headers {
		httpReq.Header.Set(k, v)
	}

	resp, err := t.HTTPClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		bodyStr := string(body)
		var rpcResp jsonRPCResponse
		if err := json.Unmarshal(body, &rpcResp); err == nil && rpcResp.Error != nil {
			if err := checkRPCError(rpcResp.Error); err != nil {
				return err
			}
		}
		bodyStrLower := strings.ToLower(bodyStr)
		if strings.Contains(bodyStrLower, "invalid protocol version") || strings.Contains(bodyStrLower, "unsupported protocol version") {
			return &transport.ProtocolNegotiationError{FallbackVersion: mcp20251125.ProtocolVersion}
		}
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, bodyStr)
	}

	var rpcResp jsonRPCResponse
	if err := json.Unmarshal(body, &rpcResp); err != nil {
		return fmt.Errorf("response unmarshal failed: %w", err)
	}

	if rpcResp.Error != nil {
		if err := checkRPCError(rpcResp.Error); err != nil {
			return err
		}
	}

	if dest != nil && len(rpcResp.Result) > 0 {
		if err := json.Unmarshal(rpcResp.Result, dest); err != nil {
			return fmt.Errorf("failed to unmarshal result payload into target: %w", err)
		}
	}

	return nil
}
