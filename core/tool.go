package core

import (
	"net/http"

	"golang.org/x/oauth2"
)

// ToolboxTool represents an immutable, universal definition of a Toolbox tool.
type ToolboxTool struct {
	name                string
	description         string
	parameters          []Parameter
	invocationURL       string
	httpClient          *http.Client
	authTokenSources    map[string]oauth2.TokenSource
	boundParams         map[string]interface{}
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
func (tt *ToolboxTool) Parameters() []Parameter {
	paramsCopy := make([]Parameter, len(tt.parameters))
	copy(paramsCopy, tt.parameters)
	return paramsCopy
}
