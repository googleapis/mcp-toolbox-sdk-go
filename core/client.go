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

package core

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"golang.org/x/oauth2"
)

// The synchronous interface for a Toolbox service client.
type ToolboxClient struct {
	baseURL             string
	httpClient          *http.Client
	clientHeaderSources map[string]oauth2.TokenSource
	defaultToolOptions  []ToolOption
	defaultOptionsSet   bool
}

// NewToolboxClient creates a new, immutable synchronous ToolboxClient.
func NewToolboxClient(url string, opts ...ClientOption) (*ToolboxClient, error) {
	tc := &ToolboxClient{
		baseURL:             url,
		httpClient:          &http.Client{},
		clientHeaderSources: make(map[string]oauth2.TokenSource),
		defaultToolOptions:  []ToolOption{},
	}

	for _, opt := range opts {
		if opt == nil {
			return nil, fmt.Errorf("NewToolboxClient: received a nil ClientOption")
		}
		if err := opt(tc); err != nil {
			return nil, err
		}
	}

	return tc, nil
}

// Close closes the underlying client session's idle connections.
func (tc *ToolboxClient) Close() {
	if tr, ok := tc.httpClient.Transport.(*http.Transport); ok {
		tr.CloseIdleConnections()
	}
}

// resolveAndApplyHeaders resolves dynamic header values from TokenSources.
func (tc *ToolboxClient) resolveAndApplyHeaders(req *http.Request) error {
	for name, source := range tc.clientHeaderSources {
		token, err := source.Token()
		if err != nil {
			return fmt.Errorf("failed to resolve header '%s': %w", name, err)
		}
		req.Header.Set(name, token.AccessToken)
	}
	return nil
}

// loadManifest is an internal helper for fetching manifests from the Toolbox server.
func (tc *ToolboxClient) loadManifest(ctx context.Context, url string) (*ManifestSchema, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request to %s: %w", url, err)
	}

	if err := tc.resolveAndApplyHeaders(req); err != nil {
		return nil, fmt.Errorf("failed to apply client headers: %w", err)
	}

	resp, err := tc.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make HTTP request to %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned non-OK status: %d %s, body: %s", resp.StatusCode, resp.Status, string(bodyBytes))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var manifest ManifestSchema
	if err = json.Unmarshal(body, &manifest); err != nil {
		return nil, fmt.Errorf("failed to load tools: %w", err)
	}
	return &manifest, nil
}

func (tc *ToolboxClient) newToolboxTool(
	name string,
	schema ToolSchema,
	finalConfig *ToolConfig,
	isStrict bool,
) (*ToolboxTool, []string, []string, error) {

	finalParameters := make([]ParameterSchema, 0)
	authnParams := make(map[string][]string)
	paramSchema := make(map[string]struct{})

	// This map will store only the bound parameters that are actually applicable
	// to this tool, after considering AuthSources.
	localBoundParams := make(map[string]any)

	for _, p := range schema.Parameters {
		paramSchema[p.Name] = struct{}{}

		if len(p.AuthSources) > 0 {
			authnParams[p.Name] = p.AuthSources
		} else if val, isBound := finalConfig.BoundParams[p.Name]; isBound {
			localBoundParams[p.Name] = val
		} else {
			finalParameters = append(finalParameters, p)
		}
	}

	if isStrict {
		for boundName := range finalConfig.BoundParams {
			if _, exists := paramSchema[boundName]; !exists {
				return nil, nil, nil, fmt.Errorf("unable to bind parameter: no parameter named '%s' found on tool '%s'", boundName, name)
			}
		}
	}

	var usedBoundKeys []string
	for k := range localBoundParams {
		usedBoundKeys = append(usedBoundKeys, k)
	}

	remainingAuthnParams, remainingAuthzTokens, usedAuthKeys := identifyAuthRequirements(
		authnParams,
		schema.AuthRequired,
		finalConfig.AuthTokenSources,
	)

	tt := &ToolboxTool{
		name:                name,
		description:         schema.Description,
		parameters:          finalParameters,
		invocationURL:       fmt.Sprintf("%s/api/tool/%s%s", tc.baseURL, name, toolInvokeSuffix),
		httpClient:          tc.httpClient,
		authTokenSources:    finalConfig.AuthTokenSources,
		boundParams:         localBoundParams,
		requiredAuthnParams: remainingAuthnParams,
		requiredAuthzTokens: remainingAuthzTokens,
		clientHeaderSources: tc.clientHeaderSources,
	}

	return tt, usedAuthKeys, usedBoundKeys, nil
}

// LoadTool synchronously fetches and loads a single tool.
func (tc *ToolboxClient) LoadTool(name string, opts ...ToolOption) (*ToolboxTool, error) {
	finalConfig := &ToolConfig{}
	for _, opt := range tc.defaultToolOptions {
		if err := opt(finalConfig); err != nil {
			return nil, err
		}
	}
	for _, opt := range opts {
		if opt == nil {
			return nil, fmt.Errorf("LoadTool: received a nil ToolOption in options list")
		}
		if err := opt(finalConfig); err != nil {
			return nil, err
		}
	}

	ctx := context.Background()
	url := fmt.Sprintf("%s/api/tool/%s", tc.baseURL, name)
	manifest, err := tc.loadManifest(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("failed to load tool manifest for '%s': %w", name, err)
	}
	if manifest.Tools == nil {
		return nil, fmt.Errorf("tool '%s' not found (manifest contains no tools)", name)
	}
	schema, ok := manifest.Tools[name]
	if !ok {
		return nil, fmt.Errorf("tool '%s' not found", name)
	}

	tool, usedAuthKeys, usedBoundKeys, err := tc.newToolboxTool(name, schema, finalConfig, true)
	if err != nil {
		return nil, fmt.Errorf("failed to create toolbox tool from schema for '%s': %w", name, err)
	}

	providedAuthKeys := make(map[string]struct{})
	for k := range finalConfig.AuthTokenSources {
		providedAuthKeys[k] = struct{}{}
	}
	providedBoundKeys := make(map[string]struct{})
	for k := range finalConfig.BoundParams {
		providedBoundKeys[k] = struct{}{}
	}

	usedAuthSet := make(map[string]struct{})
	for _, k := range usedAuthKeys {
		usedAuthSet[k] = struct{}{}
	}
	usedBoundSet := make(map[string]struct{})
	for _, k := range usedBoundKeys {
		usedBoundSet[k] = struct{}{}
	}

	var errorMessages []string
	unusedAuth := findUnusedKeys(providedAuthKeys, usedAuthSet)
	unusedBound := findUnusedKeys(providedBoundKeys, usedBoundSet)

	if len(unusedAuth) > 0 {
		errorMessages = append(errorMessages, fmt.Sprintf("unused auth tokens: %s", strings.Join(unusedAuth, ", ")))
	}
	if len(unusedBound) > 0 {
		errorMessages = append(errorMessages, fmt.Sprintf("unused bound parameters: %s", strings.Join(unusedBound, ", ")))
	}
	if len(errorMessages) > 0 {
		return nil, fmt.Errorf("validation failed for tool '%s': %s", name, strings.Join(errorMessages, "; "))
	}

	return tool, nil
}

// LoadToolset synchronously fetches and loads all tools in a toolset.
func (tc *ToolboxClient) LoadToolset(opts ...ToolOption) ([]*ToolboxTool, error) {
	finalConfig := &ToolConfig{}
	for _, opt := range tc.defaultToolOptions {
		if err := opt(finalConfig); err != nil {
			return nil, err
		}
	}
	for _, opt := range opts {
		if opt == nil {
			return nil, fmt.Errorf("LoadToolset: received a nil ToolOption in options list")
		}
		if err := opt(finalConfig); err != nil {
			return nil, err
		}
	}

	ctx := context.Background()
	var url string
	if finalConfig.Name == "" {
		url = fmt.Sprintf("%s/api/toolset/", tc.baseURL)
	} else {
		url = fmt.Sprintf("%s/api/toolset/%s", tc.baseURL, finalConfig.Name)
	}
	manifest, err := tc.loadManifest(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("failed to load toolset manifest for '%s': %w", finalConfig.Name, err)
	}
	if manifest.Tools == nil {
		return nil, fmt.Errorf("toolset '%s' not found (manifest contains no tools)", finalConfig.Name)
	}

	var tools []*ToolboxTool
	overallUsedAuthKeys := make(map[string]struct{})
	overallUsedBoundParams := make(map[string]struct{})

	providedAuthKeys := make(map[string]struct{})
	for k := range finalConfig.AuthTokenSources {
		providedAuthKeys[k] = struct{}{}
	}
	providedBoundKeys := make(map[string]struct{})
	for k := range finalConfig.BoundParams {
		providedBoundKeys[k] = struct{}{}
	}

	for toolName, schema := range manifest.Tools {
		tool, usedAuthKeys, usedBoundKeys, err := tc.newToolboxTool(toolName, schema, finalConfig, finalConfig.Strict)
		if err != nil {
			return nil, fmt.Errorf("failed to create tool '%s': %w", toolName, err)
		}
		tools = append(tools, tool)

		if finalConfig.Strict {
			usedAuthSet := make(map[string]struct{})
			for _, k := range usedAuthKeys {
				usedAuthSet[k] = struct{}{}
			}
			usedBoundSet := make(map[string]struct{})
			for _, k := range usedBoundKeys {
				usedBoundSet[k] = struct{}{}
			}

			unusedAuth := findUnusedKeys(providedAuthKeys, usedAuthSet)
			unusedBound := findUnusedKeys(providedBoundKeys, usedBoundSet)

			var errorMessages []string
			if len(unusedAuth) > 0 {
				errorMessages = append(errorMessages, fmt.Sprintf("unused auth tokens: %s", strings.Join(unusedAuth, ", ")))
			}
			if len(unusedBound) > 0 {
				errorMessages = append(errorMessages, fmt.Sprintf("unused bound parameters: %s", strings.Join(unusedBound, ", ")))
			}
			if len(errorMessages) > 0 {
				return nil, fmt.Errorf("validation failed for tool '%s': %s", toolName, strings.Join(errorMessages, "; "))
			}
		} else {
			for _, k := range usedAuthKeys {
				overallUsedAuthKeys[k] = struct{}{}
			}
			for _, k := range usedBoundKeys {
				overallUsedBoundParams[k] = struct{}{}
			}
		}
	}

	if !finalConfig.Strict {
		unusedAuth := findUnusedKeys(providedAuthKeys, overallUsedAuthKeys)
		unusedBound := findUnusedKeys(providedBoundKeys, overallUsedBoundParams)

		var errorMessages []string
		if len(unusedAuth) > 0 {
			errorMessages = append(errorMessages, fmt.Sprintf("unused auth tokens could not be applied to any tool: %s", strings.Join(unusedAuth, ", ")))
		}
		if len(unusedBound) > 0 {
			errorMessages = append(errorMessages, fmt.Sprintf("unused bound parameters could not be applied to any tool: %s", strings.Join(unusedBound, ", ")))
		}
		if len(errorMessages) > 0 {
			name := finalConfig.Name
			if name == "" {
				name = "default"
			}
			return nil, fmt.Errorf("validation failed for toolset '%s': %s", name, strings.Join(errorMessages, "; "))
		}
	}

	return tools, nil
}
