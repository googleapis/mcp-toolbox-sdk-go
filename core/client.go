package core

import (
	"net/http"

	"golang.org/x/oauth2"
)

// The synchronous interface for a Toolbox service client.
type ToolboxClient struct {
	baseURL             string
	httpClient          *http.Client
	clientHeaderSources map[string]oauth2.TokenSource
	defaultToolOptions  []ToolOption
}
