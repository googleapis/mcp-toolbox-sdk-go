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
	if err := t.sendRequest(ctx, targetURL, reqPayload, &result, headers, ""); err != nil {
		return nil, err
	}

	toolsMap := make(map[string]transport.ToolSchema)
	for _, toolRaw := range result.Tools {
		toolJSON, err := json.Marshal(toolRaw)
		if err != nil {
			continue
		}
		var tool transport.ToolSchema
		if err := json.Unmarshal(toolJSON, &tool); err != nil {
			continue
		}
		if name, ok := toolRaw["name"].(string); ok {
			toolsMap[name] = tool
		}
	}

	return &transport.ManifestSchema{
		ServerVersion: "1.0.0",
		Tools:         toolsMap,
	}, nil
}

// InvokeTool executes a tool on the server.
func (t *McpTransport) InvokeTool(ctx context.Context, toolName string, payload map[string]any, headers map[string]string) (any, error) {
	reqPayload := CallToolRequest{
		JSONRPC: "2.0",
		ID:      "1",
		Method:  "tools/call",
		Params: CallToolParams{
			Name:      toolName,
			Arguments: payload,
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
	if err := t.sendRequest(ctx, t.BaseURL(), reqPayload, &result, headers, toolName); err != nil {
		return nil, err
	}

	if result.IsError {
		errMsg := "tool invocation returned error"
		if len(result.Content) > 0 {
			errMsg = result.Content[0].Text
		}
		return nil, fmt.Errorf("%s", errMsg)
	}

	if len(result.Content) == 0 {
		return "", nil
	}

	if len(result.Content) == 1 {
		var parsedJSON any
		if err := json.Unmarshal([]byte(result.Content[0].Text), &parsedJSON); err == nil {
			return parsedJSON, nil
		}
		return result.Content[0].Text, nil
	}

	var mergedObjects []any
	allObjects := true
	for _, content := range result.Content {
		var parsedJSON any
		if err := json.Unmarshal([]byte(content.Text), &parsedJSON); err == nil {
			if _, isMap := parsedJSON.(map[string]any); isMap {
				mergedObjects = append(mergedObjects, parsedJSON)
			} else {
				allObjects = false
				break
			}
		} else {
			allObjects = false
			break
		}
	}

	if allObjects && len(mergedObjects) > 0 {
		return mergedObjects, nil
	}

	var sb strings.Builder
	for _, content := range result.Content {
		sb.WriteString(content.Text)
	}
	return sb.String(), nil
}

func (t *McpTransport) sendRequest(ctx context.Context, reqURL string, bodyPayload any, dest any, headers map[string]string, mcpName string) error {
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
	if mcpName != "" {
		httpReq.Header.Set("X-Goog-Toolbox-Target-Format", "mcp")
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
		var rpcResp jsonRPCResponse
		if err := json.Unmarshal(body, &rpcResp); err == nil && rpcResp.Error != nil {
			if rpcResp.Error.Code == -32004 || rpcResp.Error.Code == -32022 {
				data, ok := rpcResp.Error.Data.(map[string]any)
				if ok {
					if supported, ok := data["supported"].([]any); ok && len(supported) > 0 {
						if fallbackStr, ok := supported[0].(string); ok {
							return &transport.ProtocolNegotiationError{FallbackVersion: fallbackStr}
						}
					}
				}
				return &transport.ProtocolNegotiationError{FallbackVersion: "2025-11-25"}
			}
			return fmt.Errorf("MCP request failed with code %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
		}
		if strings.Contains(string(body), "invalid protocol version") {
			return &transport.ProtocolNegotiationError{FallbackVersion: "2025-11-25"}
		}
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var rpcResp jsonRPCResponse
	if err := json.Unmarshal(body, &rpcResp); err != nil {
		return fmt.Errorf("response unmarshal failed: %w", err)
	}

	if rpcResp.Error != nil {
		return fmt.Errorf("MCP request failed with code %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	resultBytes, err := json.Marshal(rpcResp.Result)
	if err != nil {
		return fmt.Errorf("failed to marshal result payload: %w", err)
	}

	if err := json.Unmarshal(resultBytes, dest); err != nil {
		return fmt.Errorf("failed to unmarshal result payload into target: %w", err)
	}

	return nil
}
