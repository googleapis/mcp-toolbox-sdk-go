// Copyright 2026 Google LLC
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

package mcp

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/googleapis/mcp-toolbox-sdk-go/core/transport"
)

// BaseMcpTransport holds the common state and logic for MCP HTTP transports.
type BaseMcpTransport struct {
	baseURL       string
	HTTPClient    *http.Client
	ServerVersion string
	initOnce      sync.Once
	initErr       error

	// HandshakeHook is the abstract method _initialize_session.
	// The specific version implementation will assign this function.
	HandshakeHook func(context.Context) error
}

// BaseURL returns the base URL for the transport.
func (b *BaseMcpTransport) BaseURL() string {
	return b.baseURL
}

// NewBaseTransport creates a new base transport.
func NewBaseTransport(baseURL string, client *http.Client) (*BaseMcpTransport, error) {
	if client == nil {
		client = &http.Client{}
	}
	var fullURL string
	var err error
	// Normalize by removing trailing slash first
	cleanBaseURL := strings.TrimRight(baseURL, "/")

	// Only append "/mcp/" if it is not already present
	if strings.HasSuffix(cleanBaseURL, "/mcp") {
		// It's already correct, just use it
		fullURL = cleanBaseURL
	} else {
		// It's missing, so join it safely
		// url.JoinPath handles the slash insertion automatically
		fullURL, err = url.JoinPath(cleanBaseURL, "mcp")
		if err != nil {
			return nil, err
		}
	}

	// Ensure trailing slash
	fullURL += "/"

	return &BaseMcpTransport{
		baseURL:    fullURL,
		HTTPClient: client,
	}, nil
}

// EnsureInitialized guarantees the session is ready before making requests.
func (b *BaseMcpTransport) EnsureInitialized(ctx context.Context) error {
	b.initOnce.Do(func() {
		if b.HandshakeHook != nil {
			b.initErr = b.HandshakeHook(ctx)
		} else {
			b.initErr = fmt.Errorf("transport initialization logic (HandshakeHook) not defined")
		}
	})
	return b.initErr
}

// ConvertToolDefinition converts the raw tool dictionary into a transport.ToolSchema.
func (b *BaseMcpTransport) ConvertToolDefinition(toolData map[string]any) (transport.ToolSchema, error) {
	var paramAuth map[string]any
	var invokeAuth []string

	if meta, ok := toolData["_meta"].(map[string]any); ok {
		if pa, ok := meta["toolbox/authParam"].(map[string]any); ok {
			paramAuth = pa
		}
		if ia, ok := meta["toolbox/authInvoke"].([]any); ok {
			invokeAuth = make([]string, 0, len(ia))
			for _, v := range ia {
				if s, ok := v.(string); ok {
					invokeAuth = append(invokeAuth, s)
				}
			}
		}
	}

	description, _ := toolData["description"].(string)
	inputSchema, _ := toolData["inputSchema"].(map[string]any)
	properties, _ := inputSchema["properties"].(map[string]any)

	// Create lookup set for required fields
	requiredSet := make(map[string]bool)
	if reqList, ok := inputSchema["required"].([]any); ok {
		for _, r := range reqList {
			if s, ok := r.(string); ok {
				requiredSet[s] = true
			}
		}
	}

	// Build Parameter List
	parameters := make([]transport.ParameterSchema, 0, len(properties))

	for propertyName, definition := range properties {
		definitionMap, ok := definition.(map[string]any)
		if !ok {
			continue
		}

		// Handle Auth Sources for this specific parameter
		var authSources []string
		if paramAuth != nil {
			if sourcesRaw, ok := paramAuth[propertyName]; ok {
				if sourcesList, ok := sourcesRaw.([]any); ok {
					authSources = make([]string, 0, len(sourcesList))
					for _, s := range sourcesList {
						if str, ok := s.(string); ok {
							authSources = append(authSources, str)
						}
					}
				}
			}
		}

		// Recursively parse the property
		param := parseProperty(propertyName, definitionMap, requiredSet[propertyName])
		param.AuthSources = authSources

		parameters = append(parameters, param)
	}

	return transport.ToolSchema{
		Description:  description,
		Parameters:   parameters,
		AuthRequired: invokeAuth,
	}, nil
}

// parseProperty is the recursive helper to create ParameterSchema
func parseProperty(name string, definitionMap map[string]any, isRequired bool) transport.ParameterSchema {
	param := transport.ParameterSchema{
		Name:        name,
		Type:        getString(definitionMap, "type"),
		Description: getString(definitionMap, "description"),
		Required:    isRequired,
	}

	switch param.Type {
	case "object":
		if ap, ok := definitionMap["additionalProperties"]; ok {
			switch v := ap.(type) {
			case bool:
				param.AdditionalProperties = v
			case map[string]any:
				schema := parseProperty("", v, false)
				param.AdditionalProperties = &schema
			}
		}

	case "array":
		if itemsMap, ok := definitionMap["items"].(map[string]any); ok {
			itemSchema := parseProperty("", itemsMap, false)
			param.Items = &itemSchema
		}
	}

	return param
}

// Helper to safely extract string values from map
func getString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
