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
	name                    string
	description             string
	parameters              []ParameterSchema
	secureParameters        []ParameterSchema
	transport               transport.Transport
	authTokenSources        map[string]oauth2.TokenSource
	boundParams             map[string]any
	boundParamSchemas       map[string]ParameterSchema
	boundSecureParams       map[string]any
	boundSecureParamSchemas map[string]ParameterSchema
	requiredAuthnParams     map[string][]string
	requiredAuthzTokens     []string
	clientHeaderSources     map[string]oauth2.TokenSource
	clientName              string
	clientVersion           string
	supportedProtocols      []string
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

// SecureParameters returns the list of unbound secure parameters for the tool.
func (tt *ToolboxTool) SecureParameters() []ParameterSchema {
	secCopy := make([]ParameterSchema, len(tt.secureParameters))
	copy(secCopy, tt.secureParameters)
	return secCopy
}

// InputSchema generates an OpenAPI JSON Schema for the tool's input parameters and returns it as raw bytes.
// Secure parameters are strictly isolated and never included in InputSchema.
func (tt *ToolboxTool) InputSchema() ([]byte, error) {
	properties := make(map[string]any)
	required := make([]string, 0)

	for _, p := range tt.parameters {
		var err error
		// Convert each parameter to its map representation and add to properties.
		properties[p.Name], err = schemaToMap(&p)
		if err != nil {
			return nil, fmt.Errorf("failed to convert parameter '%s' to schema map: %w", p.Name, err)
		}

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
// descriptions. Secure parameters are strictly isolated and excluded.
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
// different bound parameters or secure parameters without modifying the original,
// and all provided options must be applicable.
//
// Inputs:
//   - opts: A variadic list of ToolOption functions to further configure the
//     new tool, such as binding parameters, binding secure parameters, or
//     providing auth tokens.
//
// Returns:
//
//	A new, specialized *ToolboxTool and a nil error, or a nil tool and an
//	error if the new options are invalid or conflict with existing settings.
func (tt *ToolboxTool) ToolFrom(opts ...ToolOption) (*ToolboxTool, error) {
	// Create a config and apply the new options, checking for internal duplicates.
	config := newToolConfig()
	for _, opt := range opts {
		if opt == nil {
			return nil, fmt.Errorf("ToolFrom: received a nil ToolOption in options list")
		}
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
			if newTt.authTokenSources == nil {
				newTt.authTokenSources = make(map[string]oauth2.TokenSource)
			}
			newTt.authTokenSources[name] = source
		}
	}

	paramNames := make(map[string]ParameterSchema)
	for _, p := range tt.parameters {
		paramNames[p.Name] = p
	}

	secParamNames := make(map[string]ParameterSchema)
	for _, p := range tt.secureParameters {
		secParamNames[p.Name] = p
	}

	// Validate and merge new BoundParams
	for name, val := range config.BoundParams {
		if _, exists := secParamNames[name]; exists {
			return nil, fmt.Errorf("parameter %q is a secure parameter; use WithBindSecureParam* instead", name)
		}
		if _, exists := tt.boundSecureParams[name]; exists {
			return nil, fmt.Errorf("parameter %q is a secure parameter; use WithBindSecureParam* instead", name)
		}

		schema, exists := paramNames[name]
		if !exists {
			if _, existsInParent := tt.boundParams[name]; existsInParent {
				return nil, fmt.Errorf("cannot override existing bound parameter: '%s'", name)
			}
			return nil, fmt.Errorf("unable to bind parameter: no parameter named '%s' on the tool", name)
		}

		if newTt.boundParamSchemas == nil {
			newTt.boundParamSchemas = make(map[string]ParameterSchema)
		}
		if newTt.boundParams == nil {
			newTt.boundParams = make(map[string]any)
		}

		newTt.boundParamSchemas[name] = schema
		newTt.boundParams[name] = val
	}

	// Recalculate the remaining unbound parameters for the new tool.
	newParams := make([]ParameterSchema, 0)
	for _, p := range tt.parameters {
		if _, exists := newTt.boundParams[p.Name]; !exists {
			newParams = append(newParams, p)
		}
	}
	newTt.parameters = newParams

	// Validate and merge new SecureParams
	for name, val := range config.SecureParams {
		if _, exists := paramNames[name]; exists {
			return nil, fmt.Errorf("parameter %q is a regular parameter; use WithBindParam* instead", name)
		}
		if _, exists := tt.boundParams[name]; exists {
			return nil, fmt.Errorf("parameter %q is a regular parameter; use WithBindParam* instead", name)
		}

		schema, exists := secParamNames[name]
		if !exists {
			if _, existsInParent := tt.boundSecureParams[name]; existsInParent {
				return nil, fmt.Errorf("cannot override existing bound secure parameter: %q", name)
			}
			return nil, fmt.Errorf("unable to bind secure parameter: no secure parameter named %q on the tool", name)
		}

		if newTt.boundSecureParamSchemas == nil {
			newTt.boundSecureParamSchemas = make(map[string]ParameterSchema)
		}
		if newTt.boundSecureParams == nil {
			newTt.boundSecureParams = make(map[string]any)
		}

		newTt.boundSecureParamSchemas[name] = schema
		newTt.boundSecureParams[name] = val
	}

	// Recalculate remaining unbound secure parameters
	newSecParams := make([]ParameterSchema, 0)
	for _, p := range tt.secureParameters {
		if _, exists := newTt.boundSecureParams[p.Name]; !exists {
			newSecParams = append(newSecParams, p)
		}
	}
	newTt.secureParameters = newSecParams

	return newTt, nil
}

// cloneToolboxTool creates a deep copy of the ToolboxTool instance to ensure
// that derivative tools created with ToolFrom cannot mutate the parent.
func (tt *ToolboxTool) cloneToolboxTool() *ToolboxTool {
	newTt := &ToolboxTool{
		name:          tt.name,
		description:   tt.description,
		transport:     tt.transport,
		clientName:    tt.clientName,
		clientVersion: tt.clientVersion,
	}

	if tt.parameters != nil {
		newTt.parameters = make([]ParameterSchema, len(tt.parameters))
		copy(newTt.parameters, tt.parameters)
	}
	if tt.secureParameters != nil {
		newTt.secureParameters = make([]ParameterSchema, len(tt.secureParameters))
		copy(newTt.secureParameters, tt.secureParameters)
	}
	if tt.authTokenSources != nil {
		newTt.authTokenSources = make(map[string]oauth2.TokenSource, len(tt.authTokenSources))
		maps.Copy(newTt.authTokenSources, tt.authTokenSources)
	}
	if tt.boundParams != nil {
		newTt.boundParams = make(map[string]any, len(tt.boundParams))
		for k, v := range tt.boundParams {
			val := reflect.ValueOf(v)
			if val.Kind() == reflect.Slice {
				newSlice := reflect.MakeSlice(val.Type(), val.Len(), val.Cap())
				reflect.Copy(newSlice, val)
				newTt.boundParams[k] = newSlice.Interface()
			} else {
				newTt.boundParams[k] = v
			}
		}
	}
	if tt.boundParamSchemas != nil {
		newTt.boundParamSchemas = make(map[string]ParameterSchema, len(tt.boundParamSchemas))
		maps.Copy(newTt.boundParamSchemas, tt.boundParamSchemas)
	}
	if tt.boundSecureParams != nil {
		newTt.boundSecureParams = make(map[string]any, len(tt.boundSecureParams))
		for k, v := range tt.boundSecureParams {
			val := reflect.ValueOf(v)
			if val.Kind() == reflect.Slice {
				newSlice := reflect.MakeSlice(val.Type(), val.Len(), val.Cap())
				reflect.Copy(newSlice, val)
				newTt.boundSecureParams[k] = newSlice.Interface()
			} else {
				newTt.boundSecureParams[k] = v
			}
		}
	}
	if tt.boundSecureParamSchemas != nil {
		newTt.boundSecureParamSchemas = make(map[string]ParameterSchema, len(tt.boundSecureParamSchemas))
		maps.Copy(newTt.boundSecureParamSchemas, tt.boundSecureParamSchemas)
	}
	if tt.requiredAuthnParams != nil {
		newTt.requiredAuthnParams = make(map[string][]string, len(tt.requiredAuthnParams))
		for k, v := range tt.requiredAuthnParams {
			newSlice := make([]string, len(v))
			copy(newSlice, v)
			newTt.requiredAuthnParams[k] = newSlice
		}
	}
	if tt.requiredAuthzTokens != nil {
		newTt.requiredAuthzTokens = make([]string, len(tt.requiredAuthzTokens))
		copy(newTt.requiredAuthzTokens, tt.requiredAuthzTokens)
	}
	if tt.clientHeaderSources != nil {
		newTt.clientHeaderSources = make(map[string]oauth2.TokenSource, len(tt.clientHeaderSources))
		maps.Copy(newTt.clientHeaderSources, tt.clientHeaderSources)
	}
	if tt.supportedProtocols != nil {
		newTt.supportedProtocols = make([]string, len(tt.supportedProtocols))
		copy(newTt.supportedProtocols, tt.supportedProtocols)
	}

	return newTt
}

// Invoke executes the tool with the given input.
//
// Inputs:
//   - ctx: The context to control the lifecycle of the API request.
//   - input: A map of parameter names to values provided by the user for this
//     specific invocation. Secure parameters cannot be passed in input and
//     must be pre-configured on the tool.
//
// Returns:
//
//	The result from the API call, which can be a structured object (from a JSON
//	'result' field) or a raw string. Returns an error if any step of the
//	process fails, including if required secure parameters are missing.
func (tt *ToolboxTool) Invoke(ctx context.Context, input map[string]any) (any, error) {
	// 1. Fast-fail: validate missing required secure parameters before anything else
	var missingSecure []string
	for _, p := range tt.secureParameters {
		if p.Required && p.Default == nil {
			if _, isBound := tt.boundSecureParams[p.Name]; !isBound {
				missingSecure = append(missingSecure, p.Name)
			}
		}
	}
	if len(missingSecure) > 0 {
		return nil, fmt.Errorf("missing required secure parameter(s) [%s] for tool %q", strings.Join(missingSecure, ", "), tt.name)
	}

	// 2. Ensure all authentication tokens required by the tool are available.
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

	// 3. Validate user input and build finalPayload.
	finalPayload, err := tt.validateAndBuildPayload(input)
	if err != nil {
		return nil, fmt.Errorf("tool payload processing failed: %w", err)
	}

	// 4. Resolve and build securePayload.
	securePayload, err := tt.resolveAndBuildSecurePayload()
	if err != nil {
		return nil, err
	}

	// 5. Headers.
	resolvedHeaders := make(map[string]string)

	for k, source := range tt.clientHeaderSources {
		token, err := source.Token()
		if err != nil {
			return nil, fmt.Errorf("failed to resolve client header %s: %w", k, err)
		}
		resolvedHeaders[k] = token.AccessToken
	}

	for name, source := range tt.authTokenSources {
		token, err := source.Token()
		if err != nil {
			return nil, fmt.Errorf("failed to resolve auth token %s: %w", name, err)
		}
		headerName := fmt.Sprintf("%s_token", name)
		resolvedHeaders[headerName] = token.AccessToken
	}

	checkSecureHeaders(tt.transport.BaseURL(), len(tt.authTokenSources) > 0)

	response, err := tt.transport.InvokeTool(ctx, tt.name, finalPayload, securePayload, resolvedHeaders)
	if err != nil {
		return nil, err
	}

	return response, nil
}

// resolveValue resolves a static value or a dynamic function getter.
func resolveValue(boundVal any) (any, error) {
	switch v := boundVal.(type) {
	case func() (string, error):
		return v()
	case func() (int, error):
		return v()
	case func() (float64, error):
		return v()
	case func() (bool, error):
		return v()
	case func() ([]string, error):
		return v()
	case func() ([]int, error):
		return v()
	case func() ([]float64, error):
		return v()
	case func() ([]bool, error):
		return v()
	case func() (map[string]string, error):
		return v()
	case func() (map[string]int, error):
		return v()
	case func() (map[string]float64, error):
		return v()
	case func() (map[string]bool, error):
		return v()
	case func() (map[string]any, error):
		return v()
	case func() (any, error):
		return v()
	default:
		return boundVal, nil
	}
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
	paramSchema := make(map[string]ParameterSchema)
	for _, p := range tt.parameters {
		paramSchema[p.Name] = p
	}

	// Validate user input against the public parameter schema.
	for key, value := range input {
		param, isUnbound := paramSchema[key]
		_, isBound := tt.boundParams[key]

		if !isUnbound || isBound {
			return nil, fmt.Errorf("unexpected parameter '%s' provided", key)
		}

		if isUnbound {
			if err := param.ValidateType(value); err != nil {
				return nil, err
			}
		}
	}

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
		resolvedValue, resolveErr := resolveValue(boundVal)
		if resolveErr != nil {
			return nil, fmt.Errorf("failed to resolve bound parameter function for '%s': %w", paramName, resolveErr)
		}

		// Apply delayed schema validation
		if schema, ok := tt.boundParamSchemas[paramName]; ok {
			if err := schema.ValidateType(resolvedValue); err != nil {
				return nil, fmt.Errorf("resolved bound parameter '%s' failed validation: %w", paramName, err)
			}
		}

		if resolvedValue != nil {
			finalPayload[paramName] = resolvedValue
		}
	}

	return finalPayload, nil
}

// resolveAndBuildSecurePayload resolves bound secure parameter values, validates their types,
// and applies defaults for unbound secure parameters.
//
// Returns:
//
//	A map representing the final, validated secure parameter payload, or an error if
//	validation or parameter resolution fails. Returns nil if there are no secure parameters.
func (tt *ToolboxTool) resolveAndBuildSecurePayload() (map[string]any, error) {
	if len(tt.boundSecureParams) == 0 && len(tt.secureParameters) == 0 {
		return nil, nil
	}

	var securePayload map[string]any
	if len(tt.boundSecureParams) > 0 {
		securePayload = make(map[string]any, len(tt.boundSecureParams))
		for paramName, boundVal := range tt.boundSecureParams {
			resolvedValue, resolveErr := resolveValue(boundVal)
			if resolveErr != nil {
				return nil, fmt.Errorf("failed to resolve bound secure parameter function for '%s': %w", paramName, resolveErr)
			}
			if schema, ok := tt.boundSecureParamSchemas[paramName]; ok {
				if err := schema.ValidateType(resolvedValue); err != nil {
					return nil, fmt.Errorf("resolved bound secure parameter '%s' failed validation: %w", paramName, err)
				}
			}
			if resolvedValue != nil {
				securePayload[paramName] = resolvedValue
			}
		}
	}

	// Apply defaults for unbound secure parameters
	for _, p := range tt.secureParameters {
		if _, isBound := tt.boundSecureParams[p.Name]; !isBound && p.Default != nil {
			if securePayload == nil {
				securePayload = make(map[string]any)
			}
			securePayload[p.Name] = p.Default
		}
	}

	if len(securePayload) == 0 {
		securePayload = nil
	}

	return securePayload, nil
}
