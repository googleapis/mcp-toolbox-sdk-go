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
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"maps"

	"github.com/googleapis/mcp-toolbox-sdk-go/core/transport"
	"golang.org/x/oauth2"
)

// ToolboxTool represents an immutable, universal definition of a Toolbox tool.
type ToolboxTool struct {
	name                string
	description         string
	parameters          []ParameterSchema
	transport           transport.Transport
	authTokenSources    map[string]oauth2.TokenSource
	boundParams         map[string]any
	requiredAuthnParams map[string][]string
	requiredAuthzTokens []string
	clientHeaderSources map[string]oauth2.TokenSource
}

// Name returns the tool's name.
func (tt *ToolboxTool) Name() string {
	return tt.name
}

// Description returns the tool's description.
func (tt *ToolboxTool) Description() string {
	return tt.description
}

// Parameters returns the list of parameters that must be provided by a user
// at invocation time.
func (tt *ToolboxTool) Parameters() []ParameterSchema {
	paramsCopy := make([]ParameterSchema, len(tt.parameters))
	copy(paramsCopy, tt.parameters)
	return paramsCopy
}

// InputSchema generates an OpenAPI JSON Schema for the tool's input parameters and returns it as raw bytes.
func (tt *ToolboxTool) InputSchema() ([]byte, error) {
	properties := make(map[string]any)
	required := make([]string, 0)

	for _, p := range tt.parameters {
		// Convert each parameter to its map representation and add to properties.
		properties[p.Name] = schemaToMap(&p)

		// Collect the names of required parameters.
		if p.Required {
			required = append(required, p.Name)
		}
	}

	// Assemble the final object structure required by the LLM.
	finalSchema := map[string]any{
		"type":       "object",
		"properties": properties,
	}
	// Only add the 'required' field if there are required parameters.
	if len(required) > 0 {
		finalSchema["required"] = required
	}

	// Marshal the final map into an indented JSON string.
	return json.MarshalIndent(finalSchema, "", "  ")
}

// DescribeParameters returns a single, human-readable string that describes all
// of the tool's unbound parameters, including their names, types, and
// descriptions.
//
// Returns:
//
//	A formatted string of parameter descriptions, or an empty string if there
//	are no unbound parameters.
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

// ToolFrom creates a new, more specialized tool from an existing one by applying
// additional options. This is useful for creating variations of a tool with
// different bound parameters without modifying the original and
// all provided options must be applicable.
//
// Inputs:
//   - opts: A variadic list of ToolOption functions to further configure the
//     new tool, such as binding more parameters.
//
// Returns:
//
//	A new, specialized *ToolboxTool and a nil error, or a nil tool and an
//	error if the new options are invalid or conflict with existing settings.
func (tt *ToolboxTool) ToolFrom(opts ...ToolOption) (*ToolboxTool, error) {
	// Create a config and apply the new options, checking for internal duplicates.
	config := newToolConfig()
	for _, opt := range opts {
		if err := opt(config); err != nil {
			return nil, err
		}
	}

	// Validate that inapplicable options were not used.
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

// cloneToolboxTool creates a deep copy of the ToolboxTool instance to ensure
// that derivative tools created with ToolFrom cannot mutate the parent.
func (tt *ToolboxTool) cloneToolboxTool() *ToolboxTool {
	newTt := &ToolboxTool{
		name:                tt.name,
		description:         tt.description,
		transport:           tt.transport,
		parameters:          make([]ParameterSchema, len(tt.parameters)),
		authTokenSources:    make(map[string]oauth2.TokenSource, len(tt.authTokenSources)),
		boundParams:         make(map[string]any, len(tt.boundParams)),
		requiredAuthnParams: make(map[string][]string, len(tt.requiredAuthnParams)),
		requiredAuthzTokens: make([]string, len(tt.requiredAuthzTokens)),
		clientHeaderSources: make(map[string]oauth2.TokenSource, len(tt.clientHeaderSources)),
	}

	// Perform deep copies for slices and maps to prevent shared state.
	copy(newTt.parameters, tt.parameters)
	copy(newTt.requiredAuthzTokens, tt.requiredAuthzTokens)

	maps.Copy(newTt.authTokenSources, tt.authTokenSources)
	maps.Copy(newTt.clientHeaderSources, tt.clientHeaderSources)

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

	// Manually deep copy the map of string slices.
	for k, v := range tt.requiredAuthnParams {
		newSlice := make([]string, len(v))
		copy(newSlice, v)
		newTt.requiredAuthnParams[k] = newSlice
	}

	return newTt
}

// Invoke executes the tool with the given input.
//
// Inputs:
//   - ctx: The context to control the lifecycle of the API request.
//   - input: A map of parameter names to values provided by the user for this
//     specific invocation.
//
// Returns:
//
//	The result from the API call, which can be a structured object (from a JSON
//	'result' field) or a raw string. Returns an error if any step of the
//	process fails.
func (tt *ToolboxTool) Invoke(ctx context.Context, input map[string]any) (any, error) {

	// Ensure all authentication tokens required by the tool are available.
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

		// Check if each required service has a corresponding token source.
		for service := range reqAuthServices {
			if _, ok := tt.authTokenSources[service]; !ok {
				return nil, fmt.Errorf("permission error: auth service '%s' is required to invoke this tool but was not provided", service)
			}
		}
	}

	// Validate the user's input and merge it with pre-configured bound parameters.
	finalPayload, err := tt.validateAndBuildPayload(input)
	if err != nil {
		return nil, fmt.Errorf("tool payload processing failed: %w", err)
	}

	resolvedHeaders := make(map[string]string)

	// Resolve Client Headers
	for k, source := range tt.clientHeaderSources {
		token, err := source.Token()
		if err != nil {
			return nil, fmt.Errorf("failed to resolve client header %s: %w", k, err)
		}
		resolvedHeaders[k] = token.AccessToken
	}

	// Resolve Auth Headers
	for name, source := range tt.authTokenSources {
		token, err := source.Token()
		if err != nil {
			return nil, fmt.Errorf("failed to resolve auth token %s: %w", name, err)
		}
		// Toolbox HTTP protocol expects the suffix "_token"
		headerName := fmt.Sprintf("%s_token", name)
		resolvedHeaders[headerName] = token.AccessToken
	}

	checkSecureHeaders(tt.transport.BaseURL(), len(tt.authTokenSources) > 0)

	response, err := tt.transport.InvokeTool(ctx, tt.name, finalPayload, resolvedHeaders)
	if err != nil {
		return nil, err
	}

	return response, nil
}

// validateAndBuildPayload performs manual type validation and applies bound parameters.
//
// Inputs:
//   - input: The map of parameters provided by the user for this invocation.
//
// Returns:
//
//	A map representing the final, validated JSON payload, or an error if
//	validation or parameter resolution fails.
func (tt *ToolboxTool) validateAndBuildPayload(input map[string]any) (map[string]any, error) {
	// Create a map of the parameter schema for efficient lookups by name
	paramSchema := make(map[string]ParameterSchema)
	for _, p := range tt.parameters {
		paramSchema[p.Name] = p
	}

	// Validate user input against the schema.
	for key, value := range input {
		param, isUnbound := paramSchema[key]
		_, isBound := tt.boundParams[key]

		// An input key is invalid if it's neither an expected unbound parameter
		// nor a parameter that has been pre-configured (bound).
		if !isUnbound && !isBound {
			return nil, fmt.Errorf("unexpected parameter '%s' provided", key)
		}

		// If the parameter is a valid unbound parameter, validate its type.
		if isUnbound {
			if err := param.ValidateType(value); err != nil {
				return nil, err
			}
		}
	}

	// Initialize the final payload with the validated user input.
	finalPayload := make(map[string]any, len(input)+len(tt.boundParams))
	for k, v := range input {
		if _, ok := paramSchema[k]; ok && v != nil {
			finalPayload[k] = v
		}
	}

	for _, param := range tt.parameters {
		_, isProvided := finalPayload[param.Name]
		_, isBound := tt.boundParams[param.Name]

		if !isProvided && !isBound {
			if param.Default != nil {
				finalPayload[param.Name] = param.Default
			} else if param.Required {
				return nil, fmt.Errorf("missing required parameter '%s'", param.Name)
			}
		}
	}

	// Loop through the bound parameters and add them to the payload.
	for paramName, boundVal := range tt.boundParams {
		var resolvedValue any
		var resolveErr error
		// A bound parameter can be a static value or a function that must be
		// executed at invocation time to resolve the value.
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
