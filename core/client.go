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
	"fmt"
	"net/http"
	"strings"

	"github.com/googleapis/mcp-toolbox-sdk-go/core/transport"
	mcp20241105 "github.com/googleapis/mcp-toolbox-sdk-go/core/transport/mcp/v20241105"
	mcp20250326 "github.com/googleapis/mcp-toolbox-sdk-go/core/transport/mcp/v20250326"
	mcp20250618 "github.com/googleapis/mcp-toolbox-sdk-go/core/transport/mcp/v20250618"
	mcp20251125 "github.com/googleapis/mcp-toolbox-sdk-go/core/transport/mcp/v20251125"
	"github.com/googleapis/mcp-toolbox-sdk-go/core/transport/toolboxtransport"
	"golang.org/x/oauth2"
)

// The synchronous interface for a Toolbox service client.
type ToolboxClient struct {
	baseURL             string
	httpClient          *http.Client
	protocol            Protocol
	protocolSet         bool
	transport           transport.Transport
	clientHeaderSources map[string]oauth2.TokenSource
	defaultToolOptions  []ToolOption
	defaultOptionsSet   bool
}

// NewToolboxClient creates and configures a new, immutable client for interacting with a
// Toolbox server.
//
// Inputs:
//   - url: The base URL of the Toolbox server.
//   - opts: A variadic list of ClientOption functions to configure the client,
//     such as setting a custom http.Client, default headers, or the underlying protocol.
//
// Returns:
//
//	A configured *ToolboxClient and a nil error on success, or a nil client
//	and an error if configuration fails.
func NewToolboxClient(url string, opts ...ClientOption) (*ToolboxClient, error) {
	// Initialize the client with default values.
	// We default to MCP Protocol (the newest version alias) if not overridden.
	tc := &ToolboxClient{
		baseURL:             url,
		httpClient:          &http.Client{},
		protocol:            MCP, // Default
		clientHeaderSources: make(map[string]oauth2.TokenSource),
		defaultToolOptions:  []ToolOption{},
	}

	// Apply each functional option to customize the client configuration.
	for _, opt := range opts {
		if opt == nil {
			return nil, fmt.Errorf("NewToolboxClient: received a nil ClientOption")
		}
		if err := opt(tc); err != nil {
			return nil, err
		}
	}

	// Initialize the Transport based on the selected Protocol.
	var transportErr error = nil
	switch tc.protocol {
	case MCPv20251125:
		tc.transport, transportErr = mcp20251125.New(tc.baseURL, tc.httpClient)
	case MCPv20250618:
		tc.transport, transportErr = mcp20250618.New(tc.baseURL, tc.httpClient)
	case MCPv20250326:
		tc.transport, transportErr = mcp20250326.New(tc.baseURL, tc.httpClient)
	case MCPv20241105:
		tc.transport, transportErr = mcp20241105.New(tc.baseURL, tc.httpClient)
	case Toolbox:
		tc.transport = toolboxtransport.New(tc.baseURL, tc.httpClient)
	default:
		return nil, fmt.Errorf("unsupported protocol version: %s", tc.protocol)
	}

	return tc, transportErr
}

// newToolboxTool is an internal factory method that constructs a
// ToolboxTool from its schema and a final configuration.
//
// Inputs:
//   - name: The name of the tool being created.
//   - schema: The definition of the tool from the server manifest.
//   - finalConfig: The combined default and user-provided tool options.
//   - isStrict: A flag that, if true, errors if a bound parameter
//     config does not exist in the tool's schema.
//
// Returns:
//   - *ToolboxTool: The fully constructed tool, ready for invocation.
//   - []string: A slice of authentication source keys that were used by the tool.
//   - []string: A slice of bound parameter keys that were used by the tool.
//   - error: An error if validation fails (e.g., in strict mode).
func (tc *ToolboxClient) newToolboxTool(
	name string,
	schema ToolSchema,
	finalConfig *ToolConfig,
	isStrict bool,
	tr transport.Transport,
) (*ToolboxTool, []string, []string, error) {

	// These will be the parameters that the end-user must provide at invocation time.
	finalParameters := make([]ParameterSchema, 0)
	// This map collects parameters that require an auth token to be fulfilled.
	authnParams := make(map[string][]string)
	// This set tracks all parameter names defined in the schema for validation.
	paramSchema := make(map[string]struct{})
	// This map stores bound parameters that are applicable to this specific tool.
	localBoundParams := make(map[string]any)

	// Iterate over the tool's parameters from the schema to categorize them.
	for _, p := range schema.Parameters {

		if ap, ok := p.AdditionalProperties.(map[string]any); ok {
			apParam, err := mapToSchema(ap)
			if err != nil {
				return nil, nil, nil, err
			}
			p.AdditionalProperties = apParam
		}
		// Validate parameter schema
		if err := p.ValidateDefinition(); err != nil {
			// Return a detailed error indicating which tool failed validation.
			return nil, nil, nil, fmt.Errorf("invalid schema for tool '%s': %w", name, err)
		}
		paramSchema[p.Name] = struct{}{}

		if len(p.AuthSources) > 0 {
			// The parameter is satisfied by an authentication source.
			authnParams[p.Name] = p.AuthSources
		} else if val, isBound := finalConfig.BoundParams[p.Name]; isBound {
			// The parameter is satisfied by a pre-configured bound value.
			localBoundParams[p.Name] = val
		} else {
			// The parameter is not satisfied by auth or bindings, so it must
			// be provided by the user at invocation.
			finalParameters = append(finalParameters, p)
		}
	}

	// In strict mode, ensure that all provided bound parameters actually exist
	// on the tool's schema.
	if isStrict {
		for boundName := range finalConfig.BoundParams {
			if _, exists := paramSchema[boundName]; !exists {
				return nil, nil, nil, fmt.Errorf("unable to bind parameter: no parameter named '%s' found on tool '%s'", boundName, name)
			}
		}
	}

	// Collect the keys of the bound parameters that were actually used.
	var usedBoundKeys []string
	for k := range localBoundParams {
		usedBoundKeys = append(usedBoundKeys, k)
	}

	// Determine which auth requirements are still unmet after applying the provided tokens.
	remainingAuthnParams, remainingAuthzTokens, usedAuthKeys := identifyAuthRequirements(
		authnParams,
		schema.AuthRequired,
		finalConfig.AuthTokenSources,
	)

	// Construct the final tool object.
	tt := &ToolboxTool{
		name:                name,
		description:         schema.Description,
		parameters:          finalParameters,
		transport:           tr,
		authTokenSources:    finalConfig.AuthTokenSources,
		boundParams:         localBoundParams,
		requiredAuthnParams: remainingAuthnParams,
		requiredAuthzTokens: remainingAuthzTokens,
		clientHeaderSources: tc.clientHeaderSources,
	}

	return tt, usedAuthKeys, usedBoundKeys, nil
}

// LoadTool fetches a manifest for a single tool
//
// Inputs:
//   - name: The specific name of the tool to load.
//   - ctx: The context to control the lifecycle of the request.
//   - opts: A variadic list of ToolOption functions to configure auth tokens
//     or bind parameters for this tool.
//
// Returns:
//
//	A configured *ToolboxTool and a nil error on success, or a nil tool and
//	an error if loading or validation fails.
func (tc *ToolboxClient) LoadTool(name string, ctx context.Context, opts ...ToolOption) (*ToolboxTool, error) {
	finalConfig := newToolConfig()

	// Apply client-wide default options first.
	for _, opt := range tc.defaultToolOptions {
		if err := opt(finalConfig); err != nil {
			return nil, err
		}
	}

	// Then, apply the tool-specific options provided in this call.
	for _, opt := range opts {
		if opt == nil {
			return nil, fmt.Errorf("LoadTool: received a nil ToolOption in options list")
		}
		if err := opt(finalConfig); err != nil {
			return nil, err
		}
	}

	resolvedHeaders, err := resolveClientHeaders(tc.clientHeaderSources)
	if err != nil {
		return nil, err
	}

	// Fetch the manifest for the specified tool.
	manifest, err := tc.transport.GetTool(ctx, name, resolvedHeaders)

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

	// Construct the tool from its schema and the final configuration.
	tool, usedAuthKeys, usedBoundKeys, err := tc.newToolboxTool(name, schema, finalConfig, true, tc.transport)
	if err != nil {
		return nil, fmt.Errorf("failed to create toolbox tool from schema for '%s': %w", name, err)
	}

	// Create sets of provided and used keys for efficient lookup.
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

	// Find any provided options that were not consumed during tool creation.
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

// LoadToolset fetches a manifest for a collection of tools.
//
// Inputs:
//   - name: Name of the toolset to be loaded.Set this arg to "" to load the default toolset
//   - ctx: The context to control the lifecycle of the request.
//   - opts: A variadic list of ToolOption functions. These can include WithStrict
//     and options for auth or bound params that may apply to tools in the set.
//
// Returns:
//
//	A slice of configured *ToolboxTool and a nil error on success, or a nil
//	slice and an error if loading or validation fails.
func (tc *ToolboxClient) LoadToolset(name string, ctx context.Context, opts ...ToolOption) ([]*ToolboxTool, error) {
	finalConfig := newToolConfig()
	// Apply client-wide default options first.
	for _, opt := range tc.defaultToolOptions {
		if err := opt(finalConfig); err != nil {
			return nil, err
		}
	}

	// Then, apply the toolset-specific options provided in this call.
	for _, opt := range opts {
		if opt == nil {
			return nil, fmt.Errorf("LoadToolset: received a nil ToolOption in options list")
		}
		if err := opt(finalConfig); err != nil {
			return nil, err
		}
	}

	// Fetch the manifest for the toolset.
	resolvedHeaders, err := resolveClientHeaders(tc.clientHeaderSources)
	if err != nil {
		return nil, err
	}

	// Fetch Manifest via Transport
	manifest, err := tc.transport.ListTools(ctx, name, resolvedHeaders)
	if err != nil {
		return nil, fmt.Errorf("failed to load toolset manifest for '%s': %w", name, err)
	}
	if manifest.Tools == nil {
		return nil, fmt.Errorf("toolset '%s' not found (manifest contains no tools)", name)
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
		// Construct each tool from its schema and the shared configuration.
		tool, usedAuthKeys, usedBoundKeys, err := tc.newToolboxTool(toolName, schema, finalConfig, finalConfig.Strict, tc.transport)
		if err != nil {
			return nil, fmt.Errorf("failed to create tool '%s': %w", toolName, err)
		}
		tools = append(tools, tool)

		// Validation behavior depends on whether strict mode is enabled.
		if finalConfig.Strict {
			// In strict mode, validate each tool individually for unused options.
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
			// In non-strict mode, aggregate all used keys across all tools.
			// Validation will happen once at the end.
			for _, k := range usedAuthKeys {
				overallUsedAuthKeys[k] = struct{}{}
			}
			for _, k := range usedBoundKeys {
				overallUsedBoundParams[k] = struct{}{}
			}
		}
	}

	// For non-strict mode, perform a final validation to ensure all provided
	// options were used by at least one tool in the set.
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
			if name == "" {
				name = "default"
			}
			return nil, fmt.Errorf("validation failed for toolset '%s': %s", name, strings.Join(errorMessages, "; "))
		}
	}

	return tools, nil
}
