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
	testToolName = "test_tool"
)

// makeTokenSources is a helper to create the auth map required by the interface.
func makeTokenSources(headers map[string]string) map[string]oauth2.TokenSource {
	res := make(map[string]oauth2.TokenSource)
	for k, v := range headers {
		res[k] = oauth2.StaticTokenSource(&oauth2.Token{AccessToken: v})
	}
	return res
}

func TestBaseURL(t *testing.T) {
	baseURL := "http://fake-toolbox-server.com"
	tr := toolboxtransport.New(baseURL, http.DefaultClient)
	if tr.BaseURL() != baseURL {
		t.Errorf("expected BaseURL %q, got %q", baseURL, tr.BaseURL())
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
		json.NewEncoder(w).Encode(mockManifest)
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
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	tr := toolboxtransport.New(server.URL, server.Client())
	_, err := tr.GetTool(context.Background(), testToolName, nil)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// The Go implementation wraps the body in the error message
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
				json.NewEncoder(w).Encode(mockManifest)
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
		json.NewDecoder(r.Body).Decode(&body)
		if body["param1"] != "value1" {
			t.Errorf("unexpected body param1: %v", body["param1"])
		}

		// Response
		w.Header().Set("Content-Type", "application/json")
		// The Toolbox Server wraps success in {"result": ...}
		w.Write([]byte(`{"result": "success"}`))
	}))
	defer server.Close()

	tr := toolboxtransport.New(server.URL, server.Client())
	payload := map[string]any{"param1": "value1"}
	headers := makeTokenSources(map[string]string{"Authorization": "Bearer token"})

	result, err := tr.InvokeTool(context.Background(), testToolName, payload, headers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The HTTP Transport returns the raw bytes of the 'result' field.
	// json.RawMessage("success") is effectively []byte(`"success"`)
	expected := "success"

	if result != expected {
		t.Errorf("expected result %s, got %s", expected, result)
	}
}

func TestInvokeTool_Failure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "Invalid arguments"}`))
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

// mockTransport allows us to intercept requests without a real network connection,
// useful for testing logic that depends on the URL scheme (http vs https).
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
			// Return valid JSON result envelope so unmarshal doesn't error out early
			Body:   io.NopCloser(bytes.NewBufferString(`{"result": "ok"}`)),
			Header: make(http.Header),
		}, nil
	}

	testCases := []struct {
		name       string
		baseURL    string
		shouldWarn bool
	}{
		{"HTTP URL", "http://insecure.com", true},
		{"HTTPS URL", "https://secure.com", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buf.Reset()

			client := &http.Client{
				Transport: &mockTransport{RoundTripFunc: dummyResponse},
			}

			tr := toolboxtransport.New(tc.baseURL, client)

			payload := map[string]any{"foo": "bar"}
			headers := makeTokenSources(map[string]string{"Authorization": "Bearer token"})

			// Execute
			_, _ = tr.InvokeTool(context.Background(), "test_tool", payload, headers)

			logOutput := buf.String()
			hasWarning := strings.Contains(logOutput, "Sending ID token over HTTP")

			if tc.shouldWarn && !hasWarning {
				t.Errorf("expected warning for URL %q, but got none", tc.baseURL)
			}
			if !tc.shouldWarn && hasWarning {
				t.Errorf("unexpected warning for URL %q", tc.baseURL)
			}
		})
	}
}
