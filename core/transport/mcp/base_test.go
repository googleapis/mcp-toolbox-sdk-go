//go:build unit

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
	"errors"
	"testing"
)

func TestNewBaseTransport(t *testing.T) {
	tests := []struct {
		name     string
		baseURL  string
		expected string
	}{
		{
			name:     "Clean URL",
			baseURL:  "http://example.com",
			expected: "http://example.com/mcp/",
		},
		{
			name:     "Trailing Slash",
			baseURL:  "http://example.com/",
			expected: "http://example.com/mcp/",
		},
		{
			name:     "Already Has MCP Suffix",
			baseURL:  "http://example.com/mcp",
			expected: "http://example.com/mcp/",
		},
		{
			name:     "Already Has MCP Suffix with Slash",
			baseURL:  "http://example.com/mcp/",
			expected: "http://example.com/mcp/",
		},
		{
			name:     "Deep Path",
			baseURL:  "http://example.com/api/v1",
			expected: "http://example.com/api/v1/mcp/",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tr := NewBaseTransport(tc.baseURL, nil)
			if tr.BaseURL() != tc.expected {
				t.Errorf("Expected URL %s, got %s", tc.expected, tr.BaseURL())
			}
			if tr.HTTPClient == nil {
				t.Error("Expected HTTPClient to be initialized, got nil")
			}
		})
	}
}

func TestEnsureInitialized(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		tr := NewBaseTransport("http://example.com", nil)
		called := 0
		tr.HandshakeHook = func(ctx context.Context) error {
			called++
			return nil
		}

		// First call should trigger hook
		if err := tr.EnsureInitialized(context.Background()); err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		// Second call should NOT trigger hook
		if err := tr.EnsureInitialized(context.Background()); err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if called != 1 {
			t.Errorf("Expected hook to be called once, got %d", called)
		}
	})

	t.Run("Failure", func(t *testing.T) {
		tr := NewBaseTransport("http://example.com", nil)
		expectedErr := errors.New("handshake failed")
		tr.HandshakeHook = func(ctx context.Context) error {
			return expectedErr
		}

		if err := tr.EnsureInitialized(context.Background()); err != expectedErr {
			t.Errorf("Expected error %v, got %v", expectedErr, err)
		}

		// verify error is cached
		if err := tr.EnsureInitialized(context.Background()); err != expectedErr {
			t.Errorf("Expected cached error %v, got %v", expectedErr, err)
		}
	})

	t.Run("MissingHook", func(t *testing.T) {
		tr := NewBaseTransport("http://example.com", nil)
		// No hook defined
		err := tr.EnsureInitialized(context.Background())
		if err == nil {
			t.Error("Expected error when HandshakeHook is missing, got nil")
		}
	})
}

func TestConvertToolDefinition(t *testing.T) {
	tr := NewBaseTransport("http://example.com", nil)

	rawTool := map[string]any{
		"name":        "complex_tool",
		"description": "A test tool",
		"inputSchema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"simple_str": map[string]any{
					"type":        "string",
					"description": "Simple string param",
				},
				"nested_obj": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"inner_int": map[string]any{"type": "integer"},
					},
					"additionalProperties": map[string]any{
						"type": "string",
					},
				},
				"str_array": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "string",
					},
				},
			},
			"required": []any{"simple_str"},
		},
		"_meta": map[string]any{
			"toolbox/authParam": map[string]any{
				"simple_str": []any{"header:x-api-key"},
			},
			"toolbox/authInvoke": []any{"oauth2"},
		},
	}

	schema, err := tr.ConvertToolDefinition(rawTool)
	if err != nil {
		t.Fatalf("ConvertToolDefinition failed: %v", err)
	}

	// Check Top-Level Metadata
	if schema.Description != "A test tool" {
		t.Errorf("Expected description 'A test tool', got '%s'", schema.Description)
	}

	// Check Auth Requirements
	if len(schema.AuthRequired) != 1 || schema.AuthRequired[0] != "oauth2" {
		t.Errorf("Expected AuthRequired=['oauth2'], got %v", schema.AuthRequired)
	}

	// Check Parameters
	if len(schema.Parameters) != 3 {
		t.Fatalf("Expected 3 parameters, got %d", len(schema.Parameters))
	}

	// Helper map to find params by name easily
	params := make(map[string]any)
	for _, p := range schema.Parameters {
		params[p.Name] = p
	}

	foundSimple := false
	for _, p := range schema.Parameters {
		if p.Name == "simple_str" {
			foundSimple = true
			if !p.Required {
				t.Error("Expected simple_str to be required")
			}
			if len(p.AuthSources) != 1 || p.AuthSources[0] != "header:x-api-key" {
				t.Errorf("Expected AuthSources=['header:x-api-key'], got %v", p.AuthSources)
			}
		} else if p.Name == "nested_obj" {
			if p.Type != "object" {
				t.Errorf("Expected nested_obj type object, got %s", p.Type)
			}
			if p.AdditionalProperties == nil {
				t.Error("Expected nested_obj to have AdditionalProperties schema")
			}
		} else if p.Name == "str_array" {
			if p.Type != "array" {
				t.Errorf("Expected str_array type array, got %s", p.Type)
			}
			if p.Items == nil || p.Items.Type != "string" {
				t.Error("Expected str_array items to be type string")
			}
		}
	}

	if !foundSimple {
		t.Error("Parameter 'simple_str' not found in converted schema")
	}
}
