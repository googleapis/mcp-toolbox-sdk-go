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

package core

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"

	"maps"

	"golang.org/x/oauth2"
)

// ToolboxTool represents an immutable, universal definition of a Toolbox tool.
type ToolboxTool struct {
	name                string
	description         string
	parameters          []ParameterSchema
	invocationURL       string
	httpClient          *http.Client
	authTokenSources    map[string]oauth2.TokenSource
	boundParams         map[string]any
	requiredAuthnParams map[string][]string
	requiredAuthzTokens []string
	clientHeaderSources map[string]oauth2.TokenSource
}

const toolInvokeSuffix = "/invoke"

// Name returns the tool's name.
func (tt *ToolboxTool) Name() string {
	return tt.name
}

// Description returns the tool's description.
func (tt *ToolboxTool) Description() string {
	return tt.description
}

// Parameters returns the tool's unbound parameters.
func (tt *ToolboxTool) Parameters() []ParameterSchema {
	paramsCopy := make([]ParameterSchema, len(tt.parameters))
	copy(paramsCopy, tt.parameters)
	return paramsCopy
}

func (tt *ToolboxTool) DescribeParameters() string {
	if len(tt.parameters) == 0 {
		return ""
	}
	paramDescriptions := make([]string, len(tt.parameters))
	for i, p := range tt.parameters {
		paramDescriptions[i] = fmt.Sprintf("'%s' (type: %s, description: %s)", p.Name, p.Type, p.Description)
	}
	return strings.Join(paramDescriptions, ", ")
}

// ToolFrom creates a new, specialized tool from an existing one by applying additional options.
func (tt *ToolboxTool) ToolFrom(opts ...ToolOption) (*ToolboxTool, error) {
	// Create a config and apply the new options, checking for internal duplicates.
	config := &ToolConfig{}
	for _, opt := range opts {
		if err := opt(config); err != nil {
			return nil, err
		}
	}

	// Validate that inapplicable options were not used.
	if config.nameSet {
		return nil, fmt.Errorf("ToolFrom: WithName option is not applicable when creating a tool from an existing instance")
	}
	if config.strictSet {
		return nil, fmt.Errorf("ToolFrom: WithStrict option is not applicable as the behavior is always strict")
	}

	// Clone the parent tool to create a new, mutable instance.
	newTt := tt.cloneToolboxTool()

	// Validate and merge new AuthTokenSources, preventing overrides.
	if config.AuthTokenSources != nil {
		for name, source := range config.AuthTokenSources {
			if _, exists := newTt.authTokenSources[name]; exists {
				return nil, fmt.Errorf("cannot override existing auth token source: '%s'", name)
			}
			newTt.authTokenSources[name] = source
		}
	}

	// Validate and merge new BoundParams, preventing overrides.
	paramNames := make(map[string]struct{})
	for _, p := range tt.parameters {
		paramNames[p.Name] = struct{}{}
	}

	for name, val := range config.BoundParams {
		// A parameter is valid to bind if it exists in the unbound parameters list.
		if _, exists := paramNames[name]; !exists {
			// If it's not in the unbound list, check if it was already bound on the parent.
			// If it exists in neither, it's an unknown parameter.
			if _, existsInParent := tt.boundParams[name]; !existsInParent {
				return nil, fmt.Errorf("unable to bind parameter: no parameter named '%s' on the tool", name)
			}
			// If it exists in the parent's bound params, it's an attempt to override.
			return nil, fmt.Errorf("cannot override existing bound parameter: '%s'", name)
		}
		newTt.boundParams[name] = val
	}

	// Recalculate the remaining unbound parameters for the new tool.
	var newParams []ParameterSchema
	for _, p := range tt.parameters {
		if _, exists := newTt.boundParams[p.Name]; !exists {
			newParams = append(newParams, p)
		}
	}
	newTt.parameters = newParams

	return newTt, nil
}

// cloneToolboxTool creates a deep copy of the ToolboxTool instance.
func (tt *ToolboxTool) cloneToolboxTool() *ToolboxTool {
	newTt := &ToolboxTool{
		name:                tt.name,
		description:         tt.description,
		invocationURL:       tt.invocationURL,
		httpClient:          tt.httpClient,
		parameters:          make([]ParameterSchema, len(tt.parameters)),
		authTokenSources:    make(map[string]oauth2.TokenSource, len(tt.authTokenSources)),
		boundParams:         make(map[string]any, len(tt.boundParams)),
		requiredAuthnParams: make(map[string][]string, len(tt.requiredAuthnParams)),
		requiredAuthzTokens: make([]string, len(tt.requiredAuthzTokens)),
		clientHeaderSources: make(map[string]oauth2.TokenSource, len(tt.clientHeaderSources)),
	}

	copy(newTt.parameters, tt.parameters)
	copy(newTt.requiredAuthzTokens, tt.requiredAuthzTokens)

	maps.Copy(newTt.authTokenSources, tt.authTokenSources)

	for k, v := range tt.boundParams {
		val := reflect.ValueOf(v)
		if val.Kind() == reflect.Slice {
			// If it's a slice, create a new slice of the same type and length.
			newSlice := reflect.MakeSlice(val.Type(), val.Len(), val.Cap())
			// Copy the elements from the old slice to the new one.
			reflect.Copy(newSlice, val)
			// Assign the new, independent slice to the clone's map.
			newTt.boundParams[k] = newSlice.Interface()
		} else {
			// If it's not a slice, just copy the value directly.
			newTt.boundParams[k] = v
		}
	}

	for k, v := range tt.requiredAuthnParams {
		newSlice := make([]string, len(v))
		copy(newSlice, v)
		newTt.requiredAuthnParams[k] = newSlice
	}
	maps.Copy(newTt.clientHeaderSources, tt.clientHeaderSources)

	return newTt
}

// Invoke executes the tool after performing manual parameter validation.
func (tt *ToolboxTool) Invoke(ctx context.Context, input map[string]interface{}) (any, error) {
	if tt.httpClient == nil {
		return nil, fmt.Errorf("http client is not set for toolbox tool '%s'", tt.name)
	}

	if len(tt.requiredAuthnParams) > 0 || len(tt.requiredAuthzTokens) > 0 {
		reqAuthServices := make(map[string]struct{})
		for _, services := range tt.requiredAuthnParams {
			for _, service := range services {
				reqAuthServices[service] = struct{}{}
			}
		}
		for _, service := range tt.requiredAuthzTokens {
			reqAuthServices[service] = struct{}{}
		}

		for service := range reqAuthServices {
			if _, ok := tt.authTokenSources[service]; !ok {
				return nil, fmt.Errorf("permission error: auth service '%s' is required to invoke this tool but was not provided", service)
			}
		}
	}

	finalPayload, err := tt.validateAndBuildPayload(input)
	if err != nil {
		return nil, fmt.Errorf("tool payload processing failed: %w", err)
	}

	payloadBytes, err := json.Marshal(finalPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal tool payload for API call: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", tt.invocationURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create API request for tool '%s': %w", tt.name, err)
	}
	req.Header.Set("Content-Type", "application/json")

	for name, source := range tt.clientHeaderSources {
		token, tokenErr := source.Token()
		if tokenErr != nil {
			return nil, fmt.Errorf("failed to resolve client header '%s': %w", name, tokenErr)
		}
		req.Header.Set(name, token.AccessToken)
	}
	for authService, source := range tt.authTokenSources {
		token, tokenErr := source.Token()
		if tokenErr != nil {
			return nil, fmt.Errorf("failed to get token for service '%s' for tool '%s': %w", authService, tt.name, tokenErr)
		}
		headerName := fmt.Sprintf("%s_token", authService)
		req.Header.Set(headerName, token.AccessToken)
	}

	resp, err := tt.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API call to tool '%s' failed: %w", tt.name, err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read API response body for tool '%s': %w", tt.name, err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errorResponse map[string]any
		if jsonErr := json.Unmarshal(responseBody, &errorResponse); jsonErr == nil {
			if errMsg, ok := errorResponse["error"].(string); ok {
				return nil, fmt.Errorf("tool '%s' API returned error status %d: %s", tt.name, resp.StatusCode, errMsg)
			}
		}
		return nil, fmt.Errorf("tool '%s' API returned unexpected status: %d %s, body: %s", tt.name, resp.StatusCode, resp.Status, string(responseBody))
	}

	var apiResult map[string]any
	if err := json.Unmarshal(responseBody, &apiResult); err == nil {
		if result, ok := apiResult["result"]; ok {
			return result, nil
		}
	}
	return string(responseBody), nil
}

// validateAndBuildPayload performs manual type validation and applies bound parameters.
func (tt *ToolboxTool) validateAndBuildPayload(input map[string]any) (map[string]any, error) {
	paramSchema := make(map[string]ParameterSchema)
	for _, p := range tt.parameters {
		paramSchema[p.Name] = p
	}

	// Validate user input against the schema.
	for key, value := range input {
		param, isUnbound := paramSchema[key]
		_, isBound := tt.boundParams[key]

		if !isUnbound && !isBound {
			return nil, fmt.Errorf("unexpected parameter '%s' provided", key)
		}

		if isUnbound {
			if err := param.validateType(value); err != nil {
				return nil, err
			}
		}
	}

	finalPayload := make(map[string]any, len(input)+len(tt.boundParams))
	for k, v := range input {
		if _, ok := paramSchema[k]; ok {
			finalPayload[k] = v
		}
	}

	for paramName, boundVal := range tt.boundParams {
		var resolvedValue any
		var resolveErr error
		switch v := boundVal.(type) {
		case func() (string, error):
			resolvedValue, resolveErr = v()
		case func() (int, error):
			resolvedValue, resolveErr = v()
		case func() (float64, error):
			resolvedValue, resolveErr = v()
		case func() (bool, error):
			resolvedValue, resolveErr = v()
		case func() ([]string, error):
			resolvedValue, resolveErr = v()
		case func() ([]int, error):
			resolvedValue, resolveErr = v()
		case func() ([]float64, error):
			resolvedValue, resolveErr = v()
		case func() ([]bool, error):
			resolvedValue, resolveErr = v()
		default:
			resolvedValue = boundVal
		}
		if resolveErr != nil {
			return nil, fmt.Errorf("failed to resolve bound parameter function for '%s': %w", paramName, resolveErr)
		}
		finalPayload[paramName] = resolvedValue
	}

	return finalPayload, nil
}
