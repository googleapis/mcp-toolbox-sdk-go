// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
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
	"sort"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func TestFindUnusedKeys(t *testing.T) {
	testCases := []struct {
		name     string
		provided map[string]struct{}
		used     map[string]struct{}
		expected []string
	}{
		{
			name:     "Finds one unused key",
			provided: map[string]struct{}{"a": {}, "b": {}},
			used:     map[string]struct{}{"a": {}},
			expected: []string{"b"},
		},
		{
			name:     "Finds multiple unused keys",
			provided: map[string]struct{}{"a": {}, "b": {}, "c": {}},
			used:     map[string]struct{}{"a": {}},
			expected: []string{"b", "c"},
		},
		{
			name:     "Finds no unused keys",
			provided: map[string]struct{}{"a": {}, "b": {}},
			used:     map[string]struct{}{"a": {}, "b": {}},
			expected: []string{},
		},
		{
			name:     "Handles empty provided set",
			provided: map[string]struct{}{},
			used:     map[string]struct{}{"a": {}},
			expected: []string{},
		},
		{
			name:     "Handles empty used set",
			provided: map[string]struct{}{"a": {}, "b": {}},
			used:     map[string]struct{}{},
			expected: []string{"a", "b"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := findUnusedKeys(tc.provided, tc.used)
			sort.Strings(result)
			sort.Strings(tc.expected)
			if !reflect.DeepEqual(result, tc.expected) {
				t.Errorf("expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestIdentifyAuthRequirements(t *testing.T) {
	t.Run("Satisfies authn and authz", func(t *testing.T) {
		reqAuthn := map[string][]string{"paramA": {"google"}}
		reqAuthz := []string{"github"}
		sources := map[string]oauth2.TokenSource{"google": nil, "github": nil}

		unmetAuthn, unmetAuthz, used := identifyAuthRequirements(reqAuthn, reqAuthz, sources)

		if len(unmetAuthn) != 0 {
			t.Errorf("Expected 0 unmet authn params, got %d", len(unmetAuthn))
		}
		if len(unmetAuthz) != 0 {
			t.Errorf("Expected 0 unmet authz tokens, got %d", len(unmetAuthz))
		}
		sort.Strings(used)
		if !reflect.DeepEqual(used, []string{"github", "google"}) {
			t.Errorf("Expected used keys [github, google], got %v", used)
		}
	})

	t.Run("Negative Test - Fails to satisfy authn", func(t *testing.T) {
		reqAuthn := map[string][]string{"paramA": {"google"}}
		sources := map[string]oauth2.TokenSource{"github": nil}

		unmetAuthn, _, _ := identifyAuthRequirements(reqAuthn, nil, sources)

		if len(unmetAuthn) != 1 {
			t.Fatal("Expected 1 unmet authn param")
		}
		if _, ok := unmetAuthn["paramA"]; !ok {
			t.Error("Expected 'paramA' to be in the unmet set")
		}
	})

	t.Run("Negative Test - Fails to satisfy authz", func(t *testing.T) {
		reqAuthz := []string{"github"}
		sources := map[string]oauth2.TokenSource{"google": nil}

		_, unmetAuthz, _ := identifyAuthRequirements(nil, reqAuthz, sources)

		if !reflect.DeepEqual(unmetAuthz, []string{"github"}) {
			t.Errorf("Expected unmet authz to be [github], got %v", unmetAuthz)
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
		err := resolveAndApplyHeaders(client.clientHeaderSources, req)

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

		err := resolveAndApplyHeaders(client.clientHeaderSources, req)

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

		manifest, err := loadManifest(context.Background(), server.URL, client.httpClient, client.clientHeaderSources)

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
		_, err := loadManifest(context.Background(), "http://example.com", client.httpClient, client.clientHeaderSources)

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

		_, err := loadManifest(context.Background(), server.URL, client.httpClient, client.clientHeaderSources)

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

		_, err := loadManifest(context.Background(), server.URL, client.httpClient, client.clientHeaderSources)

		if err == nil {
			t.Fatal("Expected an error due to JSON unmarshal failure, but got nil")
		}
		if !strings.Contains(err.Error(), "unable to parse manifest correctly") {
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
		_, err := loadManifest(ctx, server.URL, client.httpClient, client.clientHeaderSources)

		// Assert
		if err == nil {
			t.Fatal("Expected an error due to context cancellation, but got nil")
		}
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("Expected context.DeadlineExceeded error, but got a different error: %v", err)
		}
	})
}

func TestCustomTokenSource(t *testing.T) {
	t.Run("successful token retrieval", func(t *testing.T) {
		expectedToken := "my-secret-test-token-12345"
		mockProvider := func() string {
			return expectedToken
		}
		tokenSource := NewCustomTokenSource(mockProvider)

		token, err := tokenSource.Token()

		if err != nil {
			t.Fatalf("Token() returned an unexpected error: %v", err)
		}
		if token == nil {
			t.Fatal("Token() returned a nil token, but a valid one was expected.")
		}
		if token.AccessToken != expectedToken {
			t.Errorf("Expected access token '%s', but got '%s'", expectedToken, token.AccessToken)
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
		err := resolveAndApplyHeaders(client.clientHeaderSources, req)

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

		err := resolveAndApplyHeaders(client.clientHeaderSources, req)

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

		manifest, err := loadManifest(context.Background(), server.URL, client.httpClient, client.clientHeaderSources)

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
		_, err := loadManifest(context.Background(), "http://example.com", client.httpClient, client.clientHeaderSources)

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

		_, err := loadManifest(context.Background(), server.URL, client.httpClient, client.clientHeaderSources)

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

		_, err := loadManifest(context.Background(), server.URL, client.httpClient, client.clientHeaderSources)

		if err == nil {
			t.Fatal("Expected an error due to JSON unmarshal failure, but got nil")
		}
		if !strings.Contains(err.Error(), "unable to parse manifest correctly") {
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
		_, err := loadManifest(ctx, server.URL, client.httpClient, client.clientHeaderSources)

		// Assert
		if err == nil {
			t.Fatal("Expected an error due to context cancellation, but got nil")
		}
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("Expected context.DeadlineExceeded error, but got a different error: %v", err)
		}
	})
}
func TestCustomTokenSource(t *testing.T) {
	t.Run("successful token retrieval", func(t *testing.T) {
		expectedToken := "my-secret-test-token-12345"
		mockProvider := func() string {
			return expectedToken
		}
		tokenSource := NewCustomTokenSource(mockProvider)

		token, err := tokenSource.Token()

		if err != nil {
			t.Fatalf("Token() returned an unexpected error: %v", err)
		}
		if token == nil {
			t.Fatal("Token() returned a nil token, but a valid one was expected.")
		}
		if token.AccessToken != expectedToken {
			t.Errorf("Expected access token '%s', but got '%s'", expectedToken, token.AccessToken)
		}
	})
}
