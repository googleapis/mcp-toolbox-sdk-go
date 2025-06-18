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
	"net/http"

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
