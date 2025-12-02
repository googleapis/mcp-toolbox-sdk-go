package transport

import (
	"context"

	"golang.org/x/oauth2"
)

type Transport interface {
	BaseURL() string

	// GetTool fetches a single tool manifest.
	GetTool(ctx context.Context, toolName string, tokenSources map[string]oauth2.TokenSource) (*ManifestSchema, error)

	// ListTools fetches available tools.
	ListTools(ctx context.Context, toolsetName string, tokenSources map[string]oauth2.TokenSource) (*ManifestSchema, error)

	// InvokeTool executes a tool.
	InvokeTool(ctx context.Context, toolName string, payload map[string]any, tokenSources map[string]oauth2.TokenSource) (any, error)
}
