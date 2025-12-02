// Copyright 2025 Google LLC
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

package toolboxtransport

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/googleapis/mcp-toolbox-sdk-go/core/transport"
	"golang.org/x/oauth2"
)

type ToolboxTransport struct {
	baseURL    string
	httpClient *http.Client
}

func New(baseURL string, client *http.Client) transport.Transport {
	return &ToolboxTransport{baseURL: baseURL, httpClient: client}
}

func (t *ToolboxTransport) BaseURL() string { return t.baseURL }

func (t *ToolboxTransport) GetTool(ctx context.Context, toolName string, tokenSources map[string]oauth2.TokenSource) (*transport.ManifestSchema, error) {
	url := fmt.Sprintf("%s/api/tool/%s", t.baseURL, toolName)
	return t.fetchManifest(ctx, url, tokenSources)
}

func (t *ToolboxTransport) ListTools(ctx context.Context, toolsetName string, tokenSources map[string]oauth2.TokenSource) (*transport.ManifestSchema, error) {
	url := fmt.Sprintf("%s/api/toolset/%s", t.baseURL, toolsetName)
	return t.fetchManifest(ctx, url, tokenSources)
}

// loadManifest is an internal helper for fetching manifests from the Toolbox server.
// Inputs:
//   - ctx: The context to control the lifecycle of the HTTP request, including
//     cancellation.
//   - url: The specific URL from which to fetch the manifest.
//   - tokenSources: A map of token sources to be resolved and applied as
//     headers to the request.
//
// Returns:
//
//	A pointer to the successfully parsed ManifestSchema and a nil error, or a
//	nil ManifestSchema and a descriptive error if any part of the process fails.
func (t *ToolboxTransport) fetchManifest(ctx context.Context, url string, tokenSources map[string]oauth2.TokenSource) (*transport.ManifestSchema, error) {
	// Create a new GET request with a context for cancellation.
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request to %s: %w", url, err)
	}

	// Add all client-level headers to the request
	if err := resolveAndApplyHeaders(req, tokenSources); err != nil {
		return nil, fmt.Errorf("failed to apply client headers: %w", err)
	}

	//  Execute the HTTP request.
	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make HTTP request to %s: %w", url, err)
	}
	defer resp.Body.Close()

	// Check for non-successful status codes and include the response body
	// for better debugging.
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned non-OK status: %d %s, body: %s", resp.StatusCode, resp.Status, string(bodyBytes))
	}

	// Read the response body.
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Unmarshal the JSON body into the ManifestSchema struct.
	var manifest transport.ManifestSchema
	if err = json.Unmarshal(body, &manifest); err != nil {
		return nil, fmt.Errorf("unable to parse manifest correctly: %w", err)
	}
	return &manifest, nil
}

func (t *ToolboxTransport) InvokeTool(ctx context.Context, toolName string, payload map[string]any, tokenSources map[string]oauth2.TokenSource) (any, error) {
	if !strings.HasPrefix(t.baseURL, "https://") {
		log.Println("WARNING: Sending ID token over HTTP. User data may be exposed. Use HTTPS for secure communication.")
	}

	if t.httpClient == nil {
		return nil, fmt.Errorf("http client is not set for toolbox tool '%s'", toolName)
	}

	payloadBytes, err := json.Marshal(payload)
	invocationURL := fmt.Sprintf("%s/api/tool/%s/invoke", t.baseURL, toolName)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal tool payload for API call: %w", err)
	}

	// Assemble the API request
	req, err := http.NewRequestWithContext(ctx, "POST", invocationURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create API request for tool '%s': %w", toolName, err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Resolve and apply headers.
	if err := resolveAndApplyHeaders(req, tokenSources); err != nil {
		return nil, err
	}

	// API call execution
	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP call to tool '%s' failed: %w", toolName, err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body for tool '%s': %w", toolName, err)
	}

	// Handle non-successful status codes
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errorResponse map[string]any
		if jsonErr := json.Unmarshal(responseBody, &errorResponse); jsonErr == nil {
			if errMsg, ok := errorResponse["error"].(string); ok {
				return nil, fmt.Errorf("tool '%s' API returned error status %d: %s", toolName, resp.StatusCode, errMsg)
			}
		}
		return nil, fmt.Errorf("tool '%s' API returned unexpected status: %d %s, body: %s", toolName, resp.StatusCode, resp.Status, string(responseBody))
	}

	// For successful responses, attempt to extract the 'result' field.
	var apiResult map[string]any
	if err := json.Unmarshal(responseBody, &apiResult); err == nil {
		if result, ok := apiResult["result"]; ok {
			return result, nil
		}
	}
	return string(responseBody), nil
}

// resolveAndApplyHeaders iterates through a map of token sources, retrieves a
// token from each, and applies it as a header to the given HTTP request.
//
// Inputs:
//   - req: The HTTP request to which the headers will be added. This request is
//     modified in place.
//   - tokenSources: A map where the key is the HTTP header name and the
//     value is the TokenSource that provides the header's value.
//
// Returns:
//
//	An error if retrieving a token from any source fails, otherwise nil.
func resolveAndApplyHeaders(req *http.Request, tokenSources map[string]oauth2.TokenSource) error {
	for key, source := range tokenSources {
		token, err := source.Token()
		if err != nil {
			return fmt.Errorf("failed to resolve token for header '%s': %w", key, err)
		}
		req.Header.Set(key, token.AccessToken)
	}
	return nil
}
