package core

import "golang.org/x/oauth2"

// ClientOption configures a ToolboxClient at creation time.
type ClientOption func(*ToolboxClient)

// ToolConfig holds all configurable aspects for creating or deriving a tool.
type ToolConfig struct {
	AuthTokenSources map[string]oauth2.TokenSource
	BoundParams      map[string]any
	Name             string
	Strict           bool
}

// ToolOption defines a single, universal type for a functional option that configures a tool.
type ToolOption func(*ToolConfig)
