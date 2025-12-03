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

package toolboxtransport_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/googleapis/mcp-toolbox-sdk-go/core/transport"
	"github.com/googleapis/mcp-toolbox-sdk-go/core/transport/toolboxtransport"
	"golang.org/x/oauth2"
)

const (
	testBaseURL  = "http://fake-toolbox-server.com"
	testToolName = "test_tool"
)

// failingTokenSource is a token source that always returns an error, for testing failure paths.
type failingTokenSource struct{}

func (f *failingTokenSource) Token() (*oauth2.Token, error) {
	return nil, errors.New("token source failed as designed")
}

// makeTokenSources is a helper to create the auth map required by the interface.
func makeTokenSources(headers map[string]string) map[string]oauth2.TokenSource {
	if headers == nil {
		return nil
	}
	res := make(map[string]oauth2.TokenSource)
	for k, v := range headers {
		res[k] = oauth2.StaticTokenSource(&oauth2.Token{AccessToken: v})
	}
	return res
}

func TestBaseURL(t *testing.T) {
	tr := toolboxtransport.New(testBaseURL, http.DefaultClient)
	if tr.BaseURL() != testBaseURL {
		t.Errorf("expected BaseURL %q, got %q", testBaseURL, tr.BaseURL())
	}
}

func TestGetTool_Success(t *testing.T) {
	// Mock Manifest Response
	mockManifest := transport.ManifestSchema{
		ServerVersion: "1.0.0",
		Tools: map[string]transport.ToolSchema{
			testToolName: {
				Description: "A test tool",
				Parameters: []transport.ParameterSchema{
					{Name: "param1", Type: "string", Description: "The first parameter.", Required: true},
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify URL
		if r.URL.Path != "/api/tool/"+testToolName {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		// Verify Headers
		if r.Header.Get("X-Test-Header") != "value" {
			t.Errorf("missing or incorrect header X-Test-Header")
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(mockManifest)
	}))
	defer server.Close()

	tr := toolboxtransport.New(server.URL, server.Client())
	headers := makeTokenSources(map[string]string{"X-Test-Header": "value"})

	result, err := tr.GetTool(context.Background(), testToolName, headers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ServerVersion != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", result.ServerVersion)
	}
	if tool, ok := result.Tools[testToolName]; !ok {
		t.Errorf("tool %s not found in result", testToolName)
	} else if tool.Description != "A test tool" {
		t.Errorf("expected description 'A test tool', got %q", tool.Description)
	}
}

func TestGetTool_Failure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	tr := toolboxtransport.New(server.URL, server.Client())
	_, err := tr.GetTool(context.Background(), testToolName, nil)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "500") || !strings.Contains(err.Error(), "Internal Server Error") {
		t.Errorf("expected error message to contain 500 and Internal Server Error, got: %v", err)
	}
}

func TestListTools_Success(t *testing.T) {
	mockManifest := transport.ManifestSchema{ServerVersion: "1.0.0", Tools: map[string]transport.ToolSchema{}}

	testCases := []struct {
		name         string
		toolsetName  string
		expectedPath string
	}{
		{"With Toolset", "my_toolset", "/api/toolset/my_toolset"},
		{"Without Toolset", "", "/api/toolset/"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != tc.expectedPath {
					t.Errorf("expected path %q, got %q", tc.expectedPath, r.URL.Path)
				}
				_ = json.NewEncoder(w).Encode(mockManifest)
			}))
			defer server.Close()

			tr := toolboxtransport.New(server.URL, server.Client())
			_, err := tr.ListTools(context.Background(), tc.toolsetName, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestInvokeTool_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify Path & Method
		if r.URL.Path != "/api/tool/"+testToolName+"/invoke" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("unexpected method: %s", r.Method)
		}
		// Verify Headers
		if r.Header.Get("Authorization") != "Bearer token" {
			t.Errorf("missing or incorrect Authorization header")
		}
		// Verify Body
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["param1"] != "value1" {
			t.Errorf("unexpected body param1: %v", body["param1"])
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("success"))
	}))
	defer server.Close()

	tr := toolboxtransport.New(server.URL, server.Client())
	payload := map[string]any{"param1": "value1"}
	headers := makeTokenSources(map[string]string{"Authorization": "Bearer token"})

	result, err := tr.InvokeTool(context.Background(), testToolName, payload, headers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "success"

	if result != expected {
		t.Errorf("expected result %s, got %s", expected, result)
	}
}

func TestInvokeTool_Failure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error": "Invalid arguments"}`))
	}))
	defer server.Close()

	tr := toolboxtransport.New(server.URL, server.Client())
	_, err := tr.InvokeTool(context.Background(), testToolName, map[string]any{}, nil)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "Invalid arguments") {
		t.Errorf("expected error to contain 'Invalid arguments', got: %v", err)
	}
}

type mockTransport struct {
	RoundTripFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.RoundTripFunc(req)
}

func TestInvokeTool_HTTPWarning(t *testing.T) {
	// Capture logs to verify the warning
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	// Mock a successful response so InvokeTool completes (or fails gracefully after the log)
	dummyResponse := func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewBufferString("ok")),
			Header:     make(http.Header),
		}, nil
	}

	testCases := []struct {
		name       string
		baseURL    string
		shouldWarn bool
	}{
		{"HTTP", "http://insecure.com", true},
		{"HTTPS", "https://secure.com", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buf.Reset()

			client := &http.Client{
				Transport: &mockTransport{RoundTripFunc: dummyResponse},
			}

			tr := toolboxtransport.New(tc.baseURL, client)

			payload := map[string]any{"param": "val"}

			_, _ = tr.InvokeTool(context.Background(), testToolName, payload, nil)

			logOutput := buf.String()
			hasWarning := strings.Contains(logOutput, "Sending ID token over HTTP")

			if tc.shouldWarn && !hasWarning {
				t.Errorf("expected warning for %s, but got none", tc.name)
			}
			if !tc.shouldWarn && hasWarning {
				t.Errorf("unexpected warning for %s", tc.name)
			}
		})
	}
}

// --- Mock for Token Testing ---
type mockTokenSource struct {
	token *oauth2.Token
	err   error
}

func (m *mockTokenSource) Token() (*oauth2.Token, error) {
	return m.token, m.err
}

func TestResolveAndApplyHeaders(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "http://example.com", nil)
		sources := map[string]oauth2.TokenSource{
			"Authorization": &mockTokenSource{token: &oauth2.Token{AccessToken: "secret"}},
			"X-Custom":      &mockTokenSource{token: &oauth2.Token{AccessToken: "custom-val"}},
		}

		err := toolboxtransport.ResolveAndApplyHeaders(req, sources)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if req.Header.Get("Authorization") != "secret" {
			t.Errorf("Authorization header incorrect, got %s", req.Header.Get("Authorization"))
		}
		if req.Header.Get("X-Custom") != "custom-val" {
			t.Errorf("X-Custom header incorrect, got %s", req.Header.Get("X-Custom"))
		}
	})

	t.Run("Failure_TokenError", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "http://example.com", nil)
		sources := map[string]oauth2.TokenSource{
			"Authorization": &mockTokenSource{err: errors.New("token fetch failed")},
		}

		err := toolboxtransport.ResolveAndApplyHeaders(req, sources)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "token fetch failed") {
			t.Errorf("expected error to contain underlying error, got: %v", err)
		}
	})
}

func TestLoadManifest(t *testing.T) {
	mockJSON := `{"serverVersion":"1.0.0","tools":{"test":{"description":"foo"}}}`

	t.Run("Success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") != "Bearer token" {
				t.Errorf("Missing Authorization header")
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(mockJSON))
		}))
		defer server.Close()

		transportConcrete := toolboxtransport.New(server.URL, server.Client()).(*toolboxtransport.ToolboxTransport)

		sources := makeTokenSources(map[string]string{"Authorization": "Bearer token"})

		manifest, err := transportConcrete.LoadManifest(context.Background(), server.URL+"/some/path", sources)
		if err != nil {
			t.Fatalf("LoadManifest failed: %v", err)
		}

		if manifest.ServerVersion != "1.0.0" {
			t.Errorf("unexpected version: %s", manifest.ServerVersion)
		}
	})

	t.Run("Failure_Status500", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("oops"))
		}))
		defer server.Close()

		transportConcrete := toolboxtransport.New(server.URL, server.Client()).(*toolboxtransport.ToolboxTransport)

		_, err := transportConcrete.LoadManifest(context.Background(), server.URL, nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "500") {
			t.Errorf("expected 500 error, got: %v", err)
		}
	})

	t.Run("Failure_BadJSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{bad json`))
		}))
		defer server.Close()

		transportConcrete := toolboxtransport.New(server.URL, server.Client()).(*toolboxtransport.ToolboxTransport)

		_, err := transportConcrete.LoadManifest(context.Background(), server.URL, nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "unable to parse manifest") {
			t.Errorf("expected parse error, got: %v", err)
		}
	})
}

func TestLoadManifest_EdgeCases(t *testing.T) {
	t.Run("Unreadable JSON Response", func(t *testing.T) {
		// Server returns 200 OK but invalid JSON body
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{broken-manifest`))
		}))
		defer server.Close()

		tr := toolboxtransport.New(server.URL, server.Client())
		_, err := tr.GetTool(context.Background(), testToolName, nil)

		if err == nil {
			t.Fatal("expected error, got nil")
		}
		// Matches: "unable to parse manifest correctly"
		if !strings.Contains(err.Error(), "unable to parse manifest") {
			t.Errorf("expected parse error, got: %v", err)
		}
	})

	t.Run("Network Error (Server Down)", func(t *testing.T) {
		// Start a server to get a valid URL, then immediately close it
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		url := server.URL
		server.Close() // Kill the server

		tr := toolboxtransport.New(url, server.Client())
		_, err := tr.GetTool(context.Background(), testToolName, nil)

		if err == nil {
			t.Fatal("expected network error, got nil")
		}
		// Matches: "failed to make HTTP request"
		if !strings.Contains(err.Error(), "failed to make HTTP request") {
			t.Errorf("expected request error, got: %v", err)
		}
	})

	t.Run("HTTP 500 with Non-JSON Body", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("Fatal Server Error"))
		}))
		defer server.Close()

		tr := toolboxtransport.New(server.URL, server.Client())
		_, err := tr.GetTool(context.Background(), testToolName, nil)

		if err == nil {
			t.Fatal("expected error, got nil")
		}
		// Matches: "server returned non-OK status: 500 ... body: Fatal Server Error"
		if !strings.Contains(err.Error(), "500") || !strings.Contains(err.Error(), "Fatal Server Error") {
			t.Errorf("expected error to contain status and raw body, got: %v", err)
		}
	})

	t.Run("NewRequest Error (Bad URL)", func(t *testing.T) {
		// Pass a URL with control characters to trigger http.NewRequest failure
		tr := toolboxtransport.New("http://bad\nurl.com", http.DefaultClient)

		_, err := tr.GetTool(context.Background(), testToolName, nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		// Matches: "failed to create HTTP request"
		if !strings.Contains(err.Error(), "failed to create HTTP request") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("Header Resolution Failure", func(t *testing.T) {
		// Use a failing token source
		failingSource := map[string]oauth2.TokenSource{
			"Authorization": &failingTokenSource{},
		}

		tr := toolboxtransport.New(testBaseURL, http.DefaultClient)
		_, err := tr.GetTool(context.Background(), testToolName, failingSource)

		if err == nil {
			t.Fatal("expected error, got nil")
		}
		// Matches: "failed to apply client headers"
		if !strings.Contains(err.Error(), "failed to apply client headers") {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestInvokeTool_EdgeCases(t *testing.T) {
	ctx := context.Background()
	t.Run("Nil_HTTP_Client", func(t *testing.T) {
		// Create transport with nil http client
		tr := toolboxtransport.New(testBaseURL, nil)
		_, err := tr.InvokeTool(ctx, "tool", map[string]any{}, nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "http client is not set") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("Unreadable JSON Response", func(t *testing.T) {
		// Server returns 200 OK but invalid JSON body
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{broken-json`))
		}))
		defer server.Close()

		tr := toolboxtransport.New(server.URL, server.Client())
		result, err := tr.InvokeTool(context.Background(), testToolName, map[string]any{}, nil)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// If JSON parsing fails, the transport is designed to return the raw body as string
		// This matches the logic: "Fallback for non-enveloped responses" or malformed result envelopes
		if resStr, ok := result.(string); !ok || resStr != `{broken-json` {
			t.Errorf("expected raw string '{broken-json', got %v", result)
		}
	})

	t.Run("Network Error (Server Down)", func(t *testing.T) {
		// Start a server to get a valid URL, then immediately close it
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		url := server.URL
		server.Close() // Kill the server

		tr := toolboxtransport.New(url, server.Client())
		_, err := tr.InvokeTool(context.Background(), testToolName, map[string]any{}, nil)

		if err == nil {
			t.Fatal("expected network error, got nil")
		}
		// Error should come from http.Client.Do
		if !strings.Contains(err.Error(), "connection refused") && !strings.Contains(err.Error(), "HTTP call to tool") {
			t.Errorf("expected connection error, got: %v", err)
		}
	})

	t.Run("HTTP 500 with Non-JSON Body", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("Fatal Database Error"))
		}))
		defer server.Close()

		tr := toolboxtransport.New(server.URL, server.Client())
		_, err := tr.InvokeTool(context.Background(), testToolName, map[string]any{}, nil)

		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "500") || !strings.Contains(err.Error(), "Fatal Database Error") {
			t.Errorf("expected error to contain status and raw body, got: %v", err)
		}
	})

	t.Run("ApplyHeaders_Error", func(t *testing.T) {
		tr := toolboxtransport.New(testBaseURL, http.DefaultClient)
		sources := map[string]oauth2.TokenSource{
			"Auth": &failingTokenSource{},
		}
		_, err := tr.InvokeTool(ctx, "tool", map[string]any{}, sources)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		// This error comes from ResolveAndApplyHeaders directly
		if !strings.Contains(err.Error(), "failed to resolve token") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("Marshal_Error", func(t *testing.T) {
		tr := toolboxtransport.New(testBaseURL, http.DefaultClient)
		// Pass a channel which cannot be marshaled to JSON
		payload := map[string]any{"bad": make(chan int)}
		_, err := tr.InvokeTool(ctx, "tool", payload, nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "failed to marshal tool payload") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("NewRequest_Error", func(t *testing.T) {
		tr := toolboxtransport.New("http://bad\nurl.com", http.DefaultClient)
		_, err := tr.InvokeTool(ctx, "tool", map[string]any{}, nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "failed to create API request") {
			t.Errorf("unexpected error: %v", err)
		}
	})
}
