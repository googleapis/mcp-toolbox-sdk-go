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
	"fmt"
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

// Parameters returns the list of parameters that must be provided by a user
// at invocation time.
func (tt *ToolboxTool) Parameters() []ParameterSchema {
	paramsCopy := make([]ParameterSchema, len(tt.parameters))
	copy(paramsCopy, tt.parameters)
	return paramsCopy
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

// cloneToolboxTool creates a deep copy of the ToolboxTool instance to ensure
// that derivative tools created with ToolFrom cannot mutate the parent.
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
