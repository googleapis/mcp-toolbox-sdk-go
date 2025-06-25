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
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

// Test Helpers & Mocks

// failingTokenSource is a token source that always returns an error, for testing failure paths.
type failingTokenSource struct{}

func (f *failingTokenSource) Token() (*oauth2.Token, error) {
	return nil, errors.New("token source failed as designed")
}

// mockNonClosingTransport is a custom http.RoundTripper for testing the Close() method.
type mockNonClosingTransport struct{}

func (m *mockNonClosingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	return nil, nil
}

// TestNewToolboxClient verifies the constructor's core functionality,
// including default values and panic handling.
func TestNewToolboxClient(t *testing.T) {
	t.Run("Creates client with default settings", func(t *testing.T) {
		// Assuming the timeout is restored in NewToolboxClient
		client, err := NewToolboxClient("https://api.example.com")
		if err != nil {
			t.Fatalf("NewToolboxClient() with no options returned an error: %v", err)
		}
		if client == nil {
			t.Fatal("NewToolboxClient returned nil")
		}
		// This test will now correctly fail if you forget the timeout
		if client.httpClient.Timeout != 0 {
			t.Errorf("expected no timeout, got %v", client.httpClient.Timeout)
		}
	})

	t.Run("Returns error when a nil option is provided", func(t *testing.T) {
		_, err := NewToolboxClient("https://toolbox.example.com", nil)
		if err == nil {
			t.Error("Expected an error, but got nil")
		}
	})

	t.Run("Returns error when an option fails", func(t *testing.T) {
		// This test confirms that errors from options are propagated correctly.
		_, err := NewToolboxClient("url",
			WithClientHeaderString("auth-a", "token-a"),
			WithClientHeaderString("auth-a", "token-b"),
		)
		if err == nil {
			t.Fatal("Expected an error from a duplicate option, but got nil")
		}
		if !strings.Contains(err.Error(), "client header 'auth-a' is already set") {
			t.Errorf("Expected an error, but got: %v", err)
		}
	})

}

// TestClientOptions contains unit tests for each ClientOption constructor
func TestClientOptions(t *testing.T) {
	t.Run("WithHTTPClient", func(t *testing.T) {
		// Setup
		customClient := &http.Client{Timeout: 30 * time.Second}
		client, _ := NewToolboxClient("test-url")

		// Action
		opt := WithHTTPClient(customClient)
		if err := opt(client); err != nil {
			t.Fatalf("WithHTTPClient returned an unexpected error: %v", err)
		}

		// Assert
		if client.httpClient != customClient {
			t.Error("WithHTTPClient did not set the http.Client correctly.")
		}
		if client.httpClient.Timeout != 30*time.Second {
			t.Errorf("Expected http client timeout to be 30s, got %v", client.httpClient.Timeout)
		}
	})

	t.Run("WithClientHeaderString", func(t *testing.T) {
		// Setup
		client, _ := NewToolboxClient("test-url")

		// Action
		opt := WithClientHeaderString("Authorization", "my-secret-token")
		if err := opt(client); err != nil {
			t.Fatalf("WithHTTPClient returned an unexpected error: %v", err)
		}

		// Assert
		source, ok := client.clientHeaderSources["Authorization"]
		if !ok {
			t.Fatal("WithClientHeaderString did not add the header source.")
		}

		token, err := source.Token()
		if err != nil {
			t.Fatalf("TokenSource returned an unexpected error: %v", err)
		}
		if token.AccessToken != "my-secret-token" {
			t.Errorf("Expected token value 'my-secret-token', got %q", token.AccessToken)
		}
	})

	t.Run("WithClientHeaderTokenSource", func(t *testing.T) {
		// Setup
		client, _ := NewToolboxClient("test-url")
		mockSource := &mockTokenSource{token: &oauth2.Token{AccessToken: "dynamic-token"}}

		// Action
		opt := WithClientHeaderTokenSource("X-Api-Key", mockSource)
		if err := opt(client); err != nil {
			t.Fatalf("WithHTTPClient returned an unexpected error: %v", err)
		}

		// Assert
		source, ok := client.clientHeaderSources["X-Api-Key"]
		if !ok {
			t.Fatal("WithClientHeaderTokenSource did not add the header source.")
		}
		if source != mockSource {
			t.Error("The stored token source is not the one that was provided.")
		}
		token, _ := source.Token()
		if token.AccessToken != "dynamic-token" {
			t.Errorf("Expected token from source to be 'dynamic-token', got %q", token.AccessToken)
		}
	})

	t.Run("WithDefaultToolOptions", func(t *testing.T) {
		// Setup
		client, _ := NewToolboxClient("test-url")
		// Create two distinct tool options for testing
		opt1 := func(tc *ToolConfig) error {
			tc.Strict = true
			return nil
		}
		opt2 := func(tc *ToolConfig) error {
			tc.Name = "TestTool"
			return nil
		}

		// Action
		clientOpt := WithDefaultToolOptions(opt1, opt2)
		if err := clientOpt(client); err != nil {
			t.Fatalf("WithDefaultToolOptions returned an unexpected error: %v", err)
		}

		// Assert
		if len(client.defaultToolOptions) != 2 {
			t.Fatalf("Expected 2 default tool options, got %d", len(client.defaultToolOptions))
		}

		// To verify the correct options were added, apply them and check the result.
		testConfig := &ToolConfig{}
		if err := client.defaultToolOptions[0](testConfig); err != nil {
			t.Fatalf("Executing first stored ToolOption returned an unexpected error: %v", err)
		}
		if !testConfig.Strict {
			t.Error("The first tool option (Strict=true) was not stored correctly.")
		}

		if err := client.defaultToolOptions[1](testConfig); err != nil {
			t.Fatalf("Executing second stored ToolOption returned an unexpected error: %v", err)
		}

		_ = client.defaultToolOptions[1](testConfig)
		if testConfig.Name != "TestTool" {
			t.Error("The second tool option (Name=TestTool) was not stored correctly.")
		}
	})

	// Test that options are correctly applied during construction
	t.Run("Applies options during construction", func(t *testing.T) {
		customClient := &http.Client{Timeout: 5 * time.Second}
		client, _ := NewToolboxClient("test-url",
			WithHTTPClient(customClient),
			WithClientHeaderString("X-Request-Id", "abc-123"),
		)

		if client.httpClient != customClient {
			t.Error("WithHTTPClient was not applied during construction.")
		}
		if _, ok := client.clientHeaderSources["X-Request-Id"]; !ok {
			t.Error("WithClientHeaderString was not applied during construction.")
		}
	})
}

func TestResolveAndApplyHeaders(t *testing.T) {
	t.Run("Successfully applies headers", func(t *testing.T) {
		// Setup
		client, _ := NewToolboxClient("test-url")
		client.clientHeaderSources["Authorization"] = &mockTokenSource{token: &oauth2.Token{AccessToken: "token123"}}
		client.clientHeaderSources["X-Api-Key"] = &mockTokenSource{token: &oauth2.Token{AccessToken: "key456"}}

		req, _ := http.NewRequest("GET", "https://toolbox.example.com", nil)

		// Action
		err := client.resolveAndApplyHeaders(req)

		// Assert
		if err != nil {
			t.Fatalf("Expected no error, but got: %v", err)
		}
		if auth := req.Header.Get("Authorization"); auth != "token123" {
			t.Errorf("Expected Authorization header 'token123', got %q", auth)
		}
		if key := req.Header.Get("X-Api-Key"); key != "key456" {
			t.Errorf("Expected X-Api-Key header 'key456', got %q", key)
		}
	})

	t.Run("Returns error when a token source fails", func(t *testing.T) {
		client, _ := NewToolboxClient("test-url")
		client.clientHeaderSources["Authorization"] = &failingTokenSource{}

		req, _ := http.NewRequest("GET", "https://toolbox.example.com", nil)

		err := client.resolveAndApplyHeaders(req)

		if err == nil {
			t.Fatal("Expected an error, but got nil")
		}
		if !strings.Contains(err.Error(), "failed to resolve header 'Authorization'") {
			t.Errorf("Error message missing expected text. Got: %s", err.Error())
		}
		if !strings.Contains(err.Error(), "token source failed as designed") {
			t.Errorf("Error message did not wrap the underlying error. Got: %s", err.Error())
		}
	})
}

func TestApplyOptions(t *testing.T) {

	// Test Case 1: The "happy path" where all options are valid and should succeed.
	t.Run("SuccessWithValidOptions", func(t *testing.T) {
		// Arrange
		config := &ToolConfig{}
		opts := []ToolOption{
			WithName("MyAwesomeTool"),
			WithStrict(true),
			WithBindParamString("user", "jane.doe"),
			WithBindParamInt("retries", 3),
		}
		expectedParams := map[string]any{
			"user":    "jane.doe",
			"retries": 3,
		}

		err := applyOptions(config, opts)

		if err != nil {
			t.Fatalf("applyOptions failed unexpectedly: %v", err)
		}
		if config.Name != "MyAwesomeTool" {
			t.Errorf("Expected Name to be 'MyAwesomeTool', got '%s'", config.Name)
		}
		if !config.Strict {
			t.Error("Expected Strict to be true, but it was false")
		}
		if !reflect.DeepEqual(config.BoundParams, expectedParams) {
			t.Errorf("BoundParams mismatch.\nGot:  %v\nWant: %v", config.BoundParams, expectedParams)
		}
	})

	// An option returns an error, applyOptions should stop and propagate it.
	t.Run("StopsWhenOptionReturnsError", func(t *testing.T) {
		config := &ToolConfig{}
		opts := []ToolOption{
			WithName("MyTool"),
			WithBindParamString("user", "test"),
			WithName("AnotherName"),
			WithStrict(true),
		}
		expectedErr := errors.New("name is already set and cannot be overridden")

		// Act
		err := applyOptions(config, opts)

		// Assert
		if err == nil {
			t.Fatal("Expected an error from a failing option, but got nil")
		}
		if err.Error() != expectedErr.Error() {
			t.Errorf("Expected error message '%s', got '%s'", expectedErr, err.Error())
		}
		// Verify that processing stopped at the point of error.
		if config.Name != "MyTool" {
			t.Errorf("Expected Name to be 'MyTool', got '%s'", config.Name)
		}
		if config.strictSet {
			t.Error("WithStrict should not have been applied after the error")
		}
		if _, ok := config.BoundParams["user"]; !ok {
			t.Error("Expected parameter 'user' to be set, but it wasn't")
		}
	})

	// The options slice contains a nil value, applyOptions should fail.
	t.Run("StopsWhenOptionIsNil", func(t *testing.T) {
		// Arrange
		config := &ToolConfig{}
		opts := []ToolOption{
			WithName("MyTool"), // This should be applied.
			nil,                // This should cause an error.
			WithStrict(true),   // This should NOT be applied.
		}
		expectedErr := errors.New("received a nil option")

		// Act
		err := applyOptions(config, opts)

		// Assert
		if err == nil {
			t.Fatal("Expected an error for a nil option, but got nil")
		}
		if err.Error() != expectedErr.Error() {
			t.Errorf("Expected error message '%s', got '%s'", expectedErr, err.Error())
		}
		if config.Name != "MyTool" {
			t.Error("Option before nil was not applied")
		}
		if config.strictSet {
			t.Error("Option after nil should not have been applied")
		}
	})
}

func TestLoadManifest(t *testing.T) {
	validManifest := ManifestSchema{
		ServerVersion: "v1",
		Tools: map[string]ToolSchema{
			"toolA": {Description: "Does a thing"},
		},
	}
	validManifestJSON, _ := json.Marshal(validManifest)

	t.Run("Successfully loads and unmarshals manifest", func(t *testing.T) {
		// Setup mock server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") != "Bearer test-token" {
				t.Errorf("Server did not receive expected Authorization header")
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write(validManifestJSON); err != nil {
				t.Fatalf("Mock server failed to write response: %v", err)
			}
		}))
		defer server.Close()

		client, _ := NewToolboxClient(server.URL, WithHTTPClient(server.Client()))
		client.clientHeaderSources["Authorization"] = oauth2.StaticTokenSource(&oauth2.Token{
			AccessToken: "Bearer test-token",
		})

		manifest, err := client.loadManifest(context.Background(), server.URL)

		if err != nil {
			t.Fatalf("Expected no error, but got: %v", err)
		}
		if !reflect.DeepEqual(*manifest, validManifest) {
			t.Errorf("Returned manifest does not match expected value")
		}
	})

	t.Run("Fails when header resolution fails", func(t *testing.T) {
		// Setup client with a failing token source
		client, _ := NewToolboxClient("any-url")
		client.clientHeaderSources["Authorization"] = &failingTokenSource{} // Use the failing mock

		// Action
		_, err := client.loadManifest(context.Background(), "http://example.com")

		// Assert
		if err == nil {
			t.Fatal("Expected an error due to header resolution failure, but got nil")
		}
		if !strings.Contains(err.Error(), "failed to apply client headers") {
			t.Errorf("Error message missing expected text. Got: %s", err.Error())
		}
	})

	t.Run("Fails when server returns non-200 status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			if _, err := w.Write([]byte("internal server error")); err != nil {
				t.Fatalf("Mock server failed to write response: %v", err)
			}
		}))
		defer server.Close()

		client, _ := NewToolboxClient(server.URL, WithHTTPClient(server.Client()))

		_, err := client.loadManifest(context.Background(), server.URL)

		if err == nil {
			t.Fatal("Expected an error due to non-OK status, but got nil")
		}
		if !strings.Contains(err.Error(), "server returned non-OK status: 500") {
			t.Errorf("Error message missing expected status code. Got: %s", err.Error())
		}
	})

	t.Run("Fails when response body is invalid JSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte(`{"serverVersion": "bad-json",`)); err != nil {
				t.Fatalf("Mock server failed to write response: %v", err)
			}
		}))
		defer server.Close()

		client, _ := NewToolboxClient(server.URL, WithHTTPClient(server.Client()))

		_, err := client.loadManifest(context.Background(), server.URL)

		if err == nil {
			t.Fatal("Expected an error due to JSON unmarshal failure, but got nil")
		}
		if !strings.Contains(err.Error(), "invalid manifest structure received") {
			t.Errorf("Error message missing expected text. Got: %s", err.Error())
		}
	})

	t.Run("Fails when context is canceled", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(100 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client, _ := NewToolboxClient(server.URL, WithHTTPClient(server.Client()))

		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		defer cancel()

		// Action
		_, err := client.loadManifest(ctx, server.URL)

		// Assert
		if err == nil {
			t.Fatal("Expected an error due to context cancellation, but got nil")
		}
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("Expected context.DeadlineExceeded error, but got a different error: %v", err)
		}
	})
}

func TestLoadToolAndLoadToolset(t *testing.T) {
	// Setup a valid manifest for the mock server
	manifest := ManifestSchema{
		ServerVersion: "v1",
		Tools: map[string]ToolSchema{
			"toolA": {
				Description: "This is tool A",
				Parameters: []ParameterSchema{
					{Name: "param1", Type: "string"},
					{Name: "param2", Type: "string", AuthSources: []string{"google"}},
				},
			},
			"toolB": {
				Description:  "Tool B",
				AuthRequired: []string{"github"},
			},
		},
	}
	manifestJSON, _ := json.Marshal(manifest)

	// Setup a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(manifestJSON); err != nil {
			t.Fatalf("Mock server failed to write error response: %v", err)
		}
	}))
	defer server.Close()

	t.Run("LoadTool - Success", func(t *testing.T) {
		client, _ := NewToolboxClient(server.URL, WithHTTPClient(server.Client()))
		tool, err := client.LoadTool("toolA",
			WithBindParamString("param1", "value1"),
			WithAuthTokenString("google", "token-google"),
		)
		if err != nil {
			t.Fatalf("LoadTool failed unexpectedly: %v", err)
		}
		if tool.name != "toolA" {
			t.Errorf("Expected tool name 'toolA', got %q", tool.name)
		}
	})

	t.Run("LoadTool - Negative Test - Unused bound parameter", func(t *testing.T) {
		client, _ := NewToolboxClient(server.URL, WithHTTPClient(server.Client()))
		_, err := client.LoadTool("toolA",
			WithBindParamString("param1", "value1"),
			WithBindParamString("unused_param", "value-unused"),
		)
		if err == nil {
			t.Fatal("Expected an error for unused bound parameter, but got nil")
		}
		if !strings.Contains(err.Error(), "unable to bind parameter: no parameter named 'unused_param' found on tool 'toolA'") {
			t.Errorf("Incorrect error for unused bound parameter. Got: %v", err)
		}
	})

	t.Run("LoadToolset - Success with non-strict mode", func(t *testing.T) {
		client, _ := NewToolboxClient(server.URL, WithHTTPClient(server.Client()))
		tools, err := client.LoadToolset(
			WithBindParamString("param1", "value1"),
			WithAuthTokenString("google", "token-google"),
			WithAuthTokenString("github", "token-github"),
		)
		if err != nil {
			t.Fatalf("LoadToolset failed unexpectedly: %v", err)
		}
		if len(tools) != 2 {
			t.Errorf("Expected to load 2 tools, got %d", len(tools))
		}
	})

	t.Run("LoadToolset - Negative Test - Unused parameter in non-strict mode", func(t *testing.T) {
		client, _ := NewToolboxClient(server.URL, WithHTTPClient(server.Client()))
		_, err := client.LoadToolset(
			WithBindParamString("param1", "value1"),
			WithAuthTokenString("unknown-auth", "token-unknown"),
		)
		if err == nil {
			t.Fatal("Expected an error for unused auth token, but got nil")
		}
		if !strings.Contains(err.Error(), "unused auth tokens could not be applied to any tool: unknown-auth") {
			t.Errorf("Incorrect error for unused auth token. Got: %v", err)
		}
	})

	t.Run("LoadToolset - Negative Test - Unused parameter in strict mode", func(t *testing.T) {
		client, _ := NewToolboxClient(server.URL, WithHTTPClient(server.Client()))
		_, err := client.LoadToolset(
			WithStrict(true), // Enable strict mode
			WithBindParamString("param1", "value1"),
			WithAuthTokenString("google", "token-google"),
			WithAuthTokenString("github", "token-github"),
			WithAuthTokenString("unused-auth", "token-unused"),
		)
		if err == nil {
			t.Fatal("Expected an error for unused auth token in strict mode, but got nil")
		}
		// In strict mode, the error is reported for the first tool it doesn't apply to
		if !strings.Contains(err.Error(), "validation failed for tool") {
			t.Errorf("Incorrect error for unused auth token in strict mode. Got: %v", err)
		}
	})
}

func TestDefaultOptionOverwriting(t *testing.T) {
	// Setup a mock server
	manifest := ManifestSchema{
		ServerVersion: "v1",
		Tools: map[string]ToolSchema{
			"toolWithParams": {
				Description: "A tool that uses the parameters being tested",
				Parameters: []ParameterSchema{
					{Name: "user_id", Type: "string"},
				},
				AuthRequired: []string{"google"},
			},
		},
	}
	manifestJSON, _ := json.Marshal(manifest)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(manifestJSON); err != nil {
			t.Fatalf("Mock server failed to write error response: %v", err)
		}
	}))
	defer server.Close()

	t.Run("LoadTool - Fails when overriding a default bound parameter", func(t *testing.T) {
		client, err := NewToolboxClient(server.URL,
			WithHTTPClient(server.Client()),
			WithDefaultToolOptions(
				WithBindParamString("user_id", "default_user"),
			),
		)
		if err != nil {
			t.Fatalf("Client creation with default options failed unexpectedly: %v", err)
		}

		_, err = client.LoadTool("toolWithParams",
			WithBindParamString("user_id", "override_user"),
		)

		if err == nil {
			t.Fatal("Expected an error when overriding a default bound parameter, but got nil")
		}

		expectedErrorMsg := "duplicate parameter binding: parameter 'user_id' is already set"
		if !strings.Contains(err.Error(), expectedErrorMsg) {
			t.Errorf("Expected error message to contain %q, but got: %v", expectedErrorMsg, err)
		}
	})

	t.Run("LoadTool - Fails when overriding a default auth token", func(t *testing.T) {

		client, err := NewToolboxClient(server.URL,
			WithHTTPClient(server.Client()),
			WithDefaultToolOptions(
				WithAuthTokenString("google", "default_google_token"),
			),
		)
		if err != nil {
			t.Fatalf("Client creation with default options failed unexpectedly: %v", err)
		}

		_, err = client.LoadTool("toolWithParams",
			WithAuthTokenString("google", "override_google_token"),
		)

		if err == nil {
			t.Fatal("Expected an error when overriding a default auth token, but got nil")
		}

		expectedErrorMsg := "authentication source 'google' is already set"
		if !strings.Contains(err.Error(), expectedErrorMsg) {
			t.Errorf("Expected error message to contain %q, but got: %v", expectedErrorMsg, err)
		}
	})
}

func TestNegativeAndEdgeCases(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	}))
	defer server.Close()

	t.Run("LoadTool fails when a nil ToolOption is provided", func(t *testing.T) {

		client, _ := NewToolboxClient(server.URL)

		_, err := client.LoadTool("any-tool", WithName("a-name"), nil)

		if err == nil {
			t.Fatal("Expected an error when a nil option is passed to LoadTool, but got nil")
		}
		if !strings.Contains(err.Error(), "received a nil option") {
			t.Errorf("Expected nil option error, got: %v", err)
		}
	})

	t.Run("Client options fail fast with nil arguments", func(t *testing.T) {

		// Test WithHTTPClient(nil)
		_, err := NewToolboxClient(server.URL, WithHTTPClient(nil))
		if err == nil {
			t.Error("Expected error from WithHTTPClient(nil), but got nil")
		} else if !strings.Contains(err.Error(), "http.Client cannot be nil") {
			t.Errorf("Incorrect error message for nil http client. Got: %v", err)
		}

		// Test WithClientHeaderTokenSource(name, nil)
		_, err = NewToolboxClient(server.URL, WithClientHeaderTokenSource("any", nil))
		if err == nil {
			t.Error("Expected error from WithClientHeaderTokenSource(name, nil), but got nil")
		} else if !strings.Contains(err.Error(), "oauth2.TokenSource for header 'any' cannot be nil") {
			t.Errorf("Incorrect error message for nil token source. Got: %v", err)
		}
	})

	t.Run("LoadTool fails gracefully if manifest has no tools", func(t *testing.T) {
		// This server returns a valid manifest, but the "tools" map is missing/empty.
		manifestWithNoTools := `{"serverVersion": "v1"}`
		serverWithNoTools := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte(manifestWithNoTools)); err != nil {
				t.Fatalf("Mock server failed to write error response: %v", err)
			}
		}))
		defer serverWithNoTools.Close()

		client, _ := NewToolboxClient(serverWithNoTools.URL, WithHTTPClient(serverWithNoTools.Client()))

		// This call would panic if the code doesn't check for a nil map.
		_, err := client.LoadTool("any-tool")

		if err == nil {
			t.Fatal("Expected an error when manifest has no tools, but got nil")
		}
		if !strings.Contains(err.Error(), "tool 'any-tool' not found") {
			t.Errorf("Expected 'tool not found' error, got: %v", err)
		}
	})
}

// TestOptionDuplicateAndEdgeCases covers scenarios where options are used incorrectly.
func TestOptionDuplicateAndEdgeCases(t *testing.T) {
	t.Run("Fails when trying to add default tool options twice", func(t *testing.T) {
		// Action: Try to configure a client with the same option type twice.
		_, err := NewToolboxClient("url",
			WithDefaultToolOptions(WithName("a")), // First call
			WithDefaultToolOptions(WithName("b")), // Second call should fail
		)

		// Assert
		if err == nil {
			t.Fatal("Expected an error when setting default tool options twice, but got nil")
		}
		if !strings.Contains(err.Error(), "default tool options have already been set") {
			t.Errorf("Incorrect error message for duplicate default options. Got: %v", err)
		}
	})

	t.Run("Fails when ClientHeaderTokenSource tries to overwrite", func(t *testing.T) {
		_, err := NewToolboxClient("url",
			WithClientHeaderString("Authorization", "token-a"),
			WithClientHeaderTokenSource("Authorization", &mockTokenSource{}), // Overwrite attempt
		)

		if err == nil {
			t.Fatal("Expected an error when overwriting a client header, but got nil")
		}
		if !strings.Contains(err.Error(), "client header 'Authorization' is already set") {
			t.Errorf("Incorrect error message for duplicate client header. Got: %v", err)
		}
	})

	t.Run("Fails when WithAuthTokenSource tries to overwrite", func(t *testing.T) {
		// Note: This check happens at application time, not client creation time.
		config := &ToolConfig{}
		_ = WithAuthTokenString("google", "token-a")(config)             // Set it once
		err := WithAuthTokenSource("google", &mockTokenSource{})(config) // Try to overwrite

		if err == nil {
			t.Fatal("Expected an error when overwriting an auth token source, but got nil")
		}
		if !strings.Contains(err.Error(), "authentication source 'google' is already set") {
			t.Errorf("Incorrect error message for duplicate auth token. Got: %v", err)
		}
	})
}

// TestToolboxClient_Close verifies the Close method's safety.
func TestToolboxClient_Close(t *testing.T) {
	t.Run("Safely closes client with default transport", func(t *testing.T) {
		client, _ := NewToolboxClient("url")
		client.Close()
	})

	t.Run("Safely does nothing for client with non-standard transport", func(t *testing.T) {
		httpClient := &http.Client{Transport: &mockNonClosingTransport{}}
		client, _ := NewToolboxClient("url", WithHTTPClient(httpClient))
		client.Close()
	})
}

// TestLoadToolAndLoadToolset_ErrorPaths covers various failure scenarios for the main functions.
func TestLoadToolAndLoadToolset_ErrorPaths(t *testing.T) {
	// --- Setup a mock server and manifest for reuse ---
	manifest := ManifestSchema{
		ServerVersion: "v1",
		Tools: map[string]ToolSchema{
			"toolA": {
				Description: "Tool A",
				Parameters: []ParameterSchema{
					{Name: "param1", Type: "string"},
					{Name: "auth_param", Type: "string", AuthSources: []string{"google"}},
				},
			},
			"toolB": {Description: "Tool B"},
		},
	}
	manifestJSON, _ := json.Marshal(manifest)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(manifestJSON); err != nil {
			t.Fatalf("Mock server failed to write error response: %v", err)
		}
	}))
	defer server.Close()

	t.Run("LoadTool fails when a default option is invalid", func(t *testing.T) {
		// Setup client with duplicate default options
		client, _ := NewToolboxClient(server.URL,
			WithHTTPClient(server.Client()),
			WithDefaultToolOptions(
				WithName("set-a"),
				WithName("set-b"),
			),
		)

		// Action: Applying the defaults inside LoadTool should fail
		_, err := client.LoadTool("toolA")

		// Assert
		if err == nil {
			t.Fatal("Expected an error from duplicate default options, but got nil")
		}
		if !strings.Contains(err.Error(), "name is already set") {
			t.Errorf("Incorrect error for duplicate default option. Got: %v", err)
		}
	})

	t.Run("LoadTool fails when tool is not in the manifest", func(t *testing.T) {
		client, _ := NewToolboxClient(server.URL, WithHTTPClient(server.Client()))
		_, err := client.LoadTool("tool-that-does-not-exist")

		if err == nil {
			t.Fatal("Expected an error for a missing tool, but got nil")
		}
		if !strings.Contains(err.Error(), "tool 'tool-that-does-not-exist' not found") {
			t.Errorf("Incorrect error for missing tool. Got: %v", err)
		}
	})

	t.Run("LoadTool fails when loadManifest returns an error", func(t *testing.T) {
		// Create a server that is immediately closed to simulate a network error
		errorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		errorServer.Close()

		client, _ := NewToolboxClient(errorServer.URL, WithHTTPClient(errorServer.Client()))
		_, err := client.LoadTool("any-tool")

		if err == nil {
			t.Fatal("Expected an error from a failed manifest load, but got nil")
		}
		if !strings.Contains(err.Error(), "failed to load tool manifest") {
			t.Errorf("Incorrect error wrapping for manifest load failure. Got: %v", err)
		}
	})

	t.Run("LoadTool fails with unused auth tokens", func(t *testing.T) {
		client, _ := NewToolboxClient(server.URL, WithHTTPClient(server.Client()))
		_, err := client.LoadTool("toolA",
			WithAuthTokenString("unused-auth", "token"), // This auth is not needed by toolA
		)
		if err == nil {
			t.Fatal("Expected an error for unused auth token, but got nil")
		}
		if !strings.Contains(err.Error(), "unused auth tokens: unused-auth") {
			t.Errorf("Incorrect error for unused auth token. Got: %v", err)
		}
	})

	t.Run("LoadTool fails with unused bound parameters", func(t *testing.T) {
		client, _ := NewToolboxClient(server.URL, WithHTTPClient(server.Client()))
		_, err := client.LoadTool("toolA",
			WithBindParamString("unused-param", "value"), // This param is not defined on toolA
		)

		if err == nil {
			t.Fatal("Expected an error for unused bound parameter, but got nil")
		}
		// Note: This error comes from newToolboxTool, so the wrapping is different
		if !strings.Contains(err.Error(), "no parameter named 'unused-param' found on tool 'toolA'") {
			t.Errorf("Incorrect error for unused bound parameter. Got: %v", err)
		}
	})

	t.Run("LoadToolset fails with unused parameters in strict mode", func(t *testing.T) {
		client, _ := NewToolboxClient(server.URL, WithHTTPClient(server.Client()))
		_, err := client.LoadToolset(
			WithStrict(true),
			WithBindParamString("param1", "value-for-tool-a"),
		)

		if err == nil {
			t.Fatal("Expected an error in strict mode for a param not on all tools, but got nil")
		}
		// The failure should happen when processing toolB
		if !strings.Contains(err.Error(), "failed to create tool 'toolB'") {
			t.Errorf("Expected failure on tool 'toolB'. Got: %v", err)
		}
	})

	t.Run("LoadToolset fails with unused parameters in non-strict mode", func(t *testing.T) {
		client, _ := NewToolboxClient(server.URL, WithHTTPClient(server.Client()))
		_, err := client.LoadToolset(
			WithStrict(false),
			WithBindParamString("completely-unused-param", "value"),
		)

		if err == nil {
			t.Fatal("Expected an error for a param used by no tools, but got nil")
		}
		if !strings.Contains(err.Error(), "unused bound parameters could not be applied to any tool") {
			t.Errorf("Incorrect error for completely unused param. Got: %v", err)
		}
	})
}
