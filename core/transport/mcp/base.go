package mcp

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/googleapis/mcp-toolbox-sdk-go/core/transport"
)

// BaseMcpTransport holds the common state and logic for MCP HTTP transports.
type BaseMcpTransport struct {
	baseURL            string
	HTTPClient         *http.Client
	ProtocolVer        string
	ServerVersion      string
	ServerCapabilities map[string]any
	initOnce           sync.Once
	initErr            error

	// HandshakeHook is the abstract method _initialize_session.
	// The specific version implementation will assign this function.
	HandshakeHook func(context.Context) error
}

// BaseURL returns the base URL for the transport.
func (b *BaseMcpTransport) BaseURL() string {
	return b.baseURL
}

// NewBaseTransport creates a new base transport.
func NewBaseTransport(baseURL string, client *http.Client) *BaseMcpTransport {
	if client == nil {
		client = &http.Client{}
	}
	fullURL := baseURL
	if len(fullURL) > 0 && fullURL[len(fullURL)-1] != '/' {
		fullURL += "/"
	}
	fullURL += "mcp/"

	return &BaseMcpTransport{
		baseURL:    fullURL,
		HTTPClient: client,
	}
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
	var parameters []transport.ParameterSchema

	for propertyName, definition := range properties {
		definitionMap, ok := definition.(map[string]any)
		if !ok {
			continue
		}

		// Handle Auth Sources for this specific parameter
		var authSources []string
		if paramAuth != nil {
			// Check if this parameter name exists in the auth map
			if sourcesRaw, ok := paramAuth[propertyName]; ok {
				if sourcesList, ok := sourcesRaw.([]any); ok {
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
		if raw, ok := definitionMap["additionalProperties"]; ok {
			if b, isBool := raw.(bool); isBool {
				param.AdditionalProperties = b
			} else if m, isMap := raw.(map[string]any); isMap {
				schema := parseProperty("", m, false)
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

func getString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
