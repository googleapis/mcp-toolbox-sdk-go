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
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
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
		{
			name:     "Preserves Query Parameters",
			baseURL:  "http://example.com?num_rows=2",
			expected: "http://example.com/mcp/?num_rows=2",
		},
		{
			name:     "Multiple Query Parameters",
			baseURL:  "http://api.com?proj=xyz&env=prod",
			expected: "http://api.com/mcp/?proj=xyz&env=prod",
		},
		{
			name:     "Escaped Query Parameters",
			baseURL:  "http://api.com/mcp?q=a%20b&flag=true",
			expected: "http://api.com/mcp/?q=a%20b&flag=true",
		},
		{
			name:     "Repeated Query Parameters",
			baseURL:  "http://api.com?tag=1&tag=2",
			expected: "http://api.com/mcp/?tag=1&tag=2",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tr, _ := NewBaseTransport(tc.baseURL, nil)
			if tr.BaseURL() != tc.expected {
				t.Errorf("Expected URL %s, got %s", tc.expected, tr.BaseURL())
			}
			if tr.HTTPClient == nil {
				t.Error("Expected HTTPClient to be initialized, got nil")
			}
		})
	}
}

func TestAppendToolsetPath(t *testing.T) {
	const base = "http://example.com/mcp/"
	tests := []struct {
		name        string
		baseURL     string
		toolsetName string
		expected    string
		wantErr     string
	}{
		{
			name:        "Empty toolset name",
			baseURL:     "http://example.com/mcp/",
			toolsetName: "",
			expected:    "http://example.com/mcp/",
		},
		{
			name:        "Append toolset name without query",
			baseURL:     "http://example.com/mcp/",
			toolsetName: "my_toolset",
			expected:    "http://example.com/mcp/my_toolset",
		},
		{
			name:        "Append toolset name with query parameters preserved",
			baseURL:     "http://example.com/mcp/?num_rows=2&env=test",
			toolsetName: "my_toolset",
			expected:    "http://example.com/mcp/my_toolset?num_rows=2&env=test",
		},
		{
			name:        "Append toolset name with spaces and special characters",
			baseURL:     "http://example.com/mcp/?num_rows=2",
			toolsetName: "my toolset",
			expected:    "http://example.com/mcp/my%20toolset?num_rows=2",
		},
		// "/" is escaped so that the name stays a single path segment.
		{name: "Slash", baseURL: base, toolsetName: "a/b", expected: base + "a%2Fb"},
		{name: "Traversal attempt", baseURL: base, toolsetName: "../../admin", expected: base + "..%2F..%2Fadmin"},
		{name: "Traversal attempt with query parameters", baseURL: base + "?num_rows=2&env=test", toolsetName: "../../admin", expected: base + "..%2F..%2Fadmin?num_rows=2&env=test"},
		{name: "Slash and space with escaped base path", baseURL: "http://example.com/my%20api/mcp/?env=test", toolsetName: "a/b c", expected: "http://example.com/my%20api/mcp/a%2Fb%20c?env=test"},
		// Other characters keep Go's default path escaping.
		{name: "Backslash", baseURL: base, toolsetName: `..\admin`, expected: base + "..%5Cadmin"},
		{name: "Percent", baseURL: base, toolsetName: "%2e%2e", expected: base + "%252e%252e"},
		{name: "Space", baseURL: base, toolsetName: "a b", expected: base + "a%20b"},
		{name: "Question mark, hash and percent", baseURL: base + "?env=test", toolsetName: "a?b#c%d", expected: base + "a%3Fb%23c%25d?env=test"},
		{name: "Non-ASCII", baseURL: base, toolsetName: "café", expected: base + "caf%C3%A9"},
		{name: "Tab between dots", baseURL: base, toolsetName: ".\t.", expected: base + ".%09."},
		{name: "Tab before dots", baseURL: base, toolsetName: "\t..", expected: base + "%09.."},
		{name: "Newline after dots", baseURL: base, toolsetName: "..\n", expected: base + "..%0A"},
		{name: "Carriage return between dots", baseURL: base, toolsetName: ".\r.", expected: base + ".%0D."},
		{name: "Tab", baseURL: base, toolsetName: "a\tb", expected: base + "a%09b"},
		{name: "NUL", baseURL: base, toolsetName: "\x00", expected: base + "%00"},
		{name: "Control character 0x1F", baseURL: base, toolsetName: "\x1f", expected: base + "%1F"},
		{name: "DEL", baseURL: base, toolsetName: "\x7f", expected: base + "%7F"},
		{name: "Semicolon and comma are not escaped", baseURL: base, toolsetName: "a;b,c", expected: base + "a;b,c"},
		{name: "Semicolon and comma with query parameters", baseURL: base + "?env=test", toolsetName: "a;b,c", expected: base + "a;b,c?env=test"},
		{name: "Sub-delimiters, colon and at sign are not escaped", baseURL: base, toolsetName: "a+b:c@d=e$f&g", expected: base + "a+b:c@d=e$f&g"},
		// "." and ".." would be read as dot segments.
		{name: "Dot is rejected", baseURL: base, toolsetName: ".", wantErr: `invalid toolset name "."`},
		{name: "Dot-dot is rejected", baseURL: base + "?env=test", toolsetName: "..", wantErr: `invalid toolset name ".."`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res, err := AppendToolsetPath(tc.baseURL, tc.toolsetName)
			if tc.wantErr != "" {
				if err == nil || err.Error() != tc.wantErr {
					t.Fatalf("Expected error %q, got result %q and error %v", tc.wantErr, res, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if res != tc.expected {
				t.Errorf("Expected %s, got %s", tc.expected, res)
			}
		})
	}
}

func TestNewBaseTransport_DoesNotFollowRedirects(t *testing.T) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("Failed to create cookie jar: %v", err)
	}
	callerTransport := &http.Transport{}
	defer callerTransport.CloseIdleConnections()

	tests := []struct {
		name   string
		client func(policyCalled *atomic.Bool) *http.Client
	}{
		{
			name:   "Default client",
			client: func(*atomic.Bool) *http.Client { return nil },
		},
		{
			name: "Caller client keeps Transport, Timeout and Jar",
			client: func(*atomic.Bool) *http.Client {
				return &http.Client{Transport: callerTransport, Timeout: 5 * time.Second, Jar: jar}
			},
		},
		{
			name: "Caller client redirect policy is overridden",
			client: func(policyCalled *atomic.Bool) *http.Client {
				return &http.Client{
					Timeout: 5 * time.Second,
					CheckRedirect: func(*http.Request, []*http.Request) error {
						policyCalled.Store(true)
						return nil // Would follow every redirect.
					},
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var targetHits atomic.Int32
			target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				targetHits.Add(1)
			}))
			defer target.Close()
			redirector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Location", target.URL)
				w.WriteHeader(http.StatusTemporaryRedirect)
			}))
			defer redirector.Close()

			var policyCalled atomic.Bool
			callerClient := tc.client(&policyCalled)
			var before http.Client
			if callerClient != nil {
				before = *callerClient
			}

			tr, err := NewBaseTransport(redirector.URL, callerClient)
			if err != nil {
				t.Fatalf("NewBaseTransport failed: %v", err)
			}
			if tr.HTTPClient == nil || tr.HTTPClient == callerClient {
				t.Fatal("Expected the transport to use its own copy of the HTTP client")
			}
			if callerClient != nil {
				if tr.HTTPClient.Transport != before.Transport || tr.HTTPClient.Timeout != before.Timeout || tr.HTTPClient.Jar != before.Jar {
					t.Error("Expected the copy to keep the caller's Transport, Timeout and Jar")
				}
			}

			req, err := http.NewRequest(http.MethodPost, tr.BaseURL(), strings.NewReader(`{"secureArguments":{"api_key":"secret"}}`))
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			req.Header.Set("my-service_token", "secret-token")
			resp, err := tr.HTTPClient.Do(req)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}
			_ = resp.Body.Close()

			if resp.StatusCode != http.StatusTemporaryRedirect || resp.Header.Get("Location") != target.URL {
				t.Errorf("Expected the 307 redirect response to be returned, got %d with Location %q", resp.StatusCode, resp.Header.Get("Location"))
			}
			if hits := targetHits.Load(); hits != 0 {
				t.Errorf("Expected the redirect target to receive no requests, got %d", hits)
			}

			if callerClient != nil {
				if policyCalled.Load() {
					t.Error("Expected the caller's CheckRedirect not to be consulted")
				}
				if callerClient.Transport != before.Transport || callerClient.Timeout != before.Timeout || callerClient.Jar != before.Jar {
					t.Error("Expected the caller's http.Client to be left unmodified")
				}
				if (before.CheckRedirect == nil) != (callerClient.CheckRedirect == nil) {
					t.Fatal("Expected the caller's CheckRedirect to be left unmodified")
				}
				if callerClient.CheckRedirect != nil && callerClient.CheckRedirect(nil, nil) != nil {
					t.Error("Expected the caller's CheckRedirect to still be the caller's function")
				}
			}
		})
	}
}

func TestEnsureInitialized(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		tr, _ := NewBaseTransport("http://example.com", nil)
		called := 0

		testHeaders := map[string]string{"Authorization": "Bearer test"}
		tr.HandshakeHook = func(ctx context.Context, headers map[string]string) error {
			called++

			// Verify headers were passed through
			if headers["Authorization"] != "Bearer test" {
				t.Errorf("Expected Authorization header 'Bearer test', got %s", headers["Authorization"])
			}
			return nil
		}

		// First call should trigger hook
		if err := tr.EnsureInitialized(context.Background(), testHeaders); err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		// Second call should NOT trigger hook
		if err := tr.EnsureInitialized(context.Background(), testHeaders); err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if called != 1 {
			t.Errorf("Expected hook to be called once, got %d", called)
		}
	})

	t.Run("Failure", func(t *testing.T) {
		tr, _ := NewBaseTransport("http://example.com", nil)
		expectedErr := errors.New("handshake failed")
		tr.HandshakeHook = func(ctx context.Context, headers map[string]string) error {
			return expectedErr
		}

		if err := tr.EnsureInitialized(context.Background(), nil); err != expectedErr {
			t.Errorf("Expected error %v, got %v", expectedErr, err)
		}

		// verify error is cached
		if err := tr.EnsureInitialized(context.Background(), nil); err != expectedErr {
			t.Errorf("Expected cached error %v, got %v", expectedErr, err)
		}
	})

	t.Run("MissingHook", func(t *testing.T) {
		tr, _ := NewBaseTransport("http://example.com", nil)
		// No hook defined
		err := tr.EnsureInitialized(context.Background(), nil)
		if err == nil {
			t.Error("Expected error when HandshakeHook is missing, got nil")
		}
	})
}

func TestConvertToolDefinition(t *testing.T) {
	t.Run("ComplexSchema", func(t *testing.T) {
		tr, _ := NewBaseTransport("http://example.com", nil)

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
					"missing_type_param": map[string]any{
						"description": "Should default to string",
					},
					"generic_array": map[string]any{
						"type": "array",
					},
					"generic_object": map[string]any{
						"type": "object",
					},
				},
				"required": []any{"simple_str"},
			},
			"_meta": map[string]any{
				"com.google.cloud/authParam": map[string]any{
					"simple_str": []any{"header:x-api-key"},
				},
				"com.google.cloud/authInvoke": []any{"oauth2"},
			},
		}

		schema, err := tr.ConvertToolDefinition(rawTool)
		if err != nil {
			t.Fatalf("ConvertToolDefinition failed: %v", err)
		}

		if schema.Description != "A test tool" {
			t.Errorf("Expected description 'A test tool', got '%s'", schema.Description)
		}

		if len(schema.AuthRequired) != 1 || schema.AuthRequired[0] != "oauth2" {
			t.Errorf("Expected AuthRequired=['oauth2'], got %v", schema.AuthRequired)
		}

		if len(schema.Parameters) != 6 {
			t.Fatalf("Expected 6 parameters, got %d", len(schema.Parameters))
		}

		foundSimple := false
		for _, p := range schema.Parameters {
			switch p.Name {
			case "simple_str":
				foundSimple = true
				if !p.Required {
					t.Error("Expected simple_str to be required")
				}
				if len(p.AuthSources) != 1 || p.AuthSources[0] != "header:x-api-key" {
					t.Errorf("Expected AuthSources=['header:x-api-key'], got %v", p.AuthSources)
				}
			case "nested_obj":
				if p.Type != "object" {
					t.Errorf("Expected nested_obj type object, got %s", p.Type)
				}
				if p.AdditionalProperties == nil {
					t.Error("Expected nested_obj to have AdditionalProperties schema")
				}
			case "str_array":
				if p.Type != "array" {
					t.Errorf("Expected str_array type array, got %s", p.Type)
				}
				if p.Items == nil || p.Items.Type != "string" {
					t.Error("Expected str_array items to be type string")
				}
			case "missing_type_param":
				if p.Type != "string" {
					t.Errorf("Expected missing_type_param type string, got %s", p.Type)
				}
			case "generic_array":
				if p.Type != "array" {
					t.Errorf("Expected generic_array type array, got %s", p.Type)
				}
				if p.Items != nil {
					t.Error("Expected generic_array items to be nil")
				}
			case "generic_object":
				if p.Type != "object" {
					t.Errorf("Expected generic_object type object, got %s", p.Type)
				}
				if p.AdditionalProperties != nil {
					t.Error("Expected generic_object AdditionalProperties to be nil")
				}
			}
		}

		if !foundSimple {
			t.Error("Parameter 'simple_str' not found in converted schema")
		}
	})

	t.Run("WithDefaults", func(t *testing.T) {
		tr, _ := NewBaseTransport("http://example.com", nil)

		rawTool := map[string]any{
			"name": "default_tool",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"count": map[string]any{
						"type":    "integer",
						"default": 10,
					},
					"text": map[string]any{
						"type":    "string",
						"default": "hello",
					},
				},
			},
		}

		schema, err := tr.ConvertToolDefinition(rawTool)
		if err != nil {
			t.Fatalf("ConvertToolDefinition failed: %v", err)
		}

		if len(schema.Parameters) != 2 {
			t.Fatalf("Expected 2 parameters, got %d", len(schema.Parameters))
		}

		foundCount := false
		foundText := false

		for _, p := range schema.Parameters {
			switch p.Name {
			case "count":
				foundCount = true
				if p.Default == nil {
					t.Error("Expected count to have a default value, got nil")
				} else if val, ok := p.Default.(int); !ok || val != 10 {
					t.Errorf("Expected count default 10, got %v (%T)", p.Default, p.Default)
				}
			case "text":
				foundText = true
				if p.Default == nil {
					t.Error("Expected text to have a default value, got nil")
				} else if val, ok := p.Default.(string); !ok || val != "hello" {
					t.Errorf("Expected text default 'hello', got %v (%T)", p.Default, p.Default)
				}
			}
		}

		if !foundCount || !foundText {
			t.Errorf("Missing expected parameters: foundCount=%v, foundText=%v", foundCount, foundText)
		}
	})

	t.Run("Pre2026", func(t *testing.T) {
		tr, _ := NewBaseTransport("http://example.com", nil)
		tr.ProtocolVersion = "2025-11-25"

		rawTool := map[string]any{
			"name":        "legacyTool",
			"description": "Legacy tool test",
			"inputSchema": map[string]any{
				"properties": map[string]any{
					"param": map[string]any{"type": "string"},
				},
			},
			"_meta": map[string]any{
				"toolbox/authParam": map[string]any{
					"param": []any{"legacy-auth"},
				},
				"toolbox/authInvoke": []any{"legacy-invoke"},
			},
		}

		schema, err := tr.ConvertToolDefinition(rawTool)
		if err != nil {
			t.Fatalf("ConvertToolDefinition failed: %v", err)
		}

		if len(schema.AuthRequired) != 1 || schema.AuthRequired[0] != "legacy-invoke" {
			t.Errorf("Expected AuthRequired=['legacy-invoke'], got %v", schema.AuthRequired)
		}
		if len(schema.Parameters) != 1 || len(schema.Parameters[0].AuthSources) != 1 || schema.Parameters[0].AuthSources[0] != "legacy-auth" {
			t.Errorf("Expected AuthSources=['legacy-auth'], got %v", schema.Parameters[0].AuthSources)
		}
	})

	t.Run("LegacyIgnoredOn2026", func(t *testing.T) {
		tr, _ := NewBaseTransport("http://example.com", nil)
		tr.ProtocolVersion = "2026-07-28"

		rawTool := map[string]any{
			"name":        "legacyToolOn2026",
			"description": "Legacy tool on 2026 version test",
			"inputSchema": map[string]any{
				"properties": map[string]any{
					"param": map[string]any{"type": "string"},
				},
			},
			"_meta": map[string]any{
				"toolbox/authParam": map[string]any{
					"param": []any{"legacy-auth"},
				},
				"toolbox/authInvoke":  []any{"legacy-invoke"},
			},
		}

		schema, err := tr.ConvertToolDefinition(rawTool)
		if err != nil {
			t.Fatalf("ConvertToolDefinition failed: %v", err)
		}

		if len(schema.AuthRequired) != 0 {
			t.Errorf("Expected empty AuthRequired, got %v", schema.AuthRequired)
		}
		if len(schema.Parameters) != 1 || len(schema.Parameters[0].AuthSources) != 0 {
			t.Errorf("Expected empty AuthSources, got %v", schema.Parameters[0].AuthSources)
		}
	})

	t.Run("MalformedMeta", func(t *testing.T) {
		tr, _ := NewBaseTransport("http://example.com", nil)
		tr.ProtocolVersion = "2026-07-28"

		rawToolMalformed := map[string]any{
			"name": "toolMalformed",
			"_meta": map[string]any{
				"com.google.cloud/authParam":  "invalid_string_instead_of_dict",
				"com.google.cloud/authInvoke": 12345,
			},
		}

		schema, err := tr.ConvertToolDefinition(rawToolMalformed)
		if err != nil {
			t.Fatalf("ConvertToolDefinition should handle malformed _meta without error, got: %v", err)
		}
		if len(schema.AuthRequired) != 0 {
			t.Errorf("Expected empty AuthRequired, got %v", schema.AuthRequired)
		}
	})

	t.Run("InvalidProtocolVersion", func(t *testing.T) {
		tr, _ := NewBaseTransport("http://example.com", nil)
		tr.ProtocolVersion = "invalid-version-string"

		rawTool := map[string]any{
			"name": "testTool",
		}

		_, err := tr.ConvertToolDefinition(rawTool)
		if err == nil {
			t.Fatal("Expected ConvertToolDefinition to fail with invalid protocol version, got nil error")
		}
	})
}
