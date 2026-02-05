//go:build unit

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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/googleapis/mcp-toolbox-sdk-go/core/transport"
	"github.com/googleapis/mcp-toolbox-sdk-go/core/transport/toolboxtransport"
	"golang.org/x/oauth2"
)

// Dummy transport for tests
type dummyTransport struct{}

func (d *dummyTransport) BaseURL() string { return "" }
func (d *dummyTransport) GetTool(ctx context.Context, name string, h map[string]string) (*transport.ManifestSchema, error) {
	return nil, nil
}
func (d *dummyTransport) ListTools(ctx context.Context, set string, h map[string]string) (*transport.ManifestSchema, error) {
	return nil, nil
}
func (d *dummyTransport) InvokeTool(ctx context.Context, name string, p map[string]any, h map[string]string) (any, error) {
	return nil, nil
}

func TestToolboxTool_Getters(t *testing.T) {
	sampleParams := []ParameterSchema{
		{Name: "param_one", Type: "string"},
		{Name: "param_two", Type: "integer"},
	}

	tool := &ToolboxTool{
		name:        "my-test-tool",
		description: "A tool specifically for testing purposes.",
		parameters:  sampleParams,
		transport:   &dummyTransport{},
	}

	t.Run("Name Method Returns Correct Value", func(t *testing.T) {
		expected := "my-test-tool"
		if got := tool.Name(); got != expected {
			t.Fatalf("Expected Name() to be '%s', but got '%s'", expected, got)
		}
	})

	t.Run("Description Method Returns Correct Value", func(t *testing.T) {
		expected := "A tool specifically for testing purposes."
		if got := tool.Description(); got != expected {
			t.Fatalf("Expected Description() to be '%s', but got '%s'", expected, got)
		}
	})

	t.Run("Parameters Method Behavior", func(t *testing.T) {
		t.Run("Returns Correct Slice Content", func(t *testing.T) {
			params := tool.Parameters()
			if !reflect.DeepEqual(params, sampleParams) {
				t.Fatalf("Parameters() returned incorrect slice.\nExpected: %+v\nGot: %+v", sampleParams, params)
			}
		})

		t.Run("Returns A Safe Copy, Not a Reference", func(t *testing.T) {
			paramsFromMethod := tool.Parameters()

			paramsFromMethod[0].Name = "MODIFIED"

			internalParams := tool.parameters
			if internalParams[0].Name == "MODIFIED" {
				t.Fatalf("Parameters() returned a direct reference to the internal slice, not a copy. Modifying the returned slice dangerously changed the tool's internal state.")
			}
		})

		t.Run("Handles Case With No Parameters", func(t *testing.T) {
			emptyTool := &ToolboxTool{
				parameters: []ParameterSchema{},
				transport:  &dummyTransport{},
			}

			params := emptyTool.Parameters()

			if params == nil {
				t.Fatalf("Parameters() should return a non-nil, empty slice for a tool with no parameters, but got nil.")
			}
			if len(params) != 0 {
				t.Fatalf("Expected an empty slice from Parameters(), but got a slice of length %d", len(params))
			}
		})
	})
}

func TestDescribeParameters(t *testing.T) {
	testCases := []struct {
		name     string
		tool     *ToolboxTool
		expected string
	}{
		{
			name:     "Tool with no parameters",
			tool:     &ToolboxTool{parameters: []ParameterSchema{}},
			expected: "",
		},
		{
			name: "Tool with one parameter",
			tool: &ToolboxTool{
				parameters: []ParameterSchema{
					{Name: "city", Type: "string"},
				},
			},
			expected: "'city' (type: string, description: )",
		},
		{
			name: "Tool with multiple parameters",
			tool: &ToolboxTool{
				parameters: []ParameterSchema{
					{Name: "location", Type: "string"},
					{Name: "days", Type: "integer"},
					{Name: "include_extended", Type: "boolean"},
				},
			},
			expected: "'location' (type: string, description: ), 'days' (type: integer, description: ), 'include_extended' (type: boolean, description: )",
		},
		{
			name: "Tool with empty parameter slice",
			tool: &ToolboxTool{
				parameters: nil,
			},
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Action
			result := tc.tool.DescribeParameters()

			// Assert
			if result != tc.expected {
				t.Errorf("expected %q, but got %q", tc.expected, result)
			}
		})
	}
}

func TestToolFrom(t *testing.T) {
	// Base tool used for creating test instances.
	baseTool := &ToolboxTool{
		name:        "weather",
		description: "gets the weather",
		parameters: []ParameterSchema{
			{Name: "city", Type: "string"},
			{Name: "days", Type: "integer"},
		},
		boundParams: map[string]any{
			"units": "celsius", // Parameter already bound on the parent
		},
		authTokenSources: map[string]oauth2.TokenSource{
			"google": &mockTokenSource{}, // Auth source already set on parent
		},
		transport: &dummyTransport{},
	}

	getTestTool := func() *ToolboxTool {
		return baseTool.cloneToolboxTool()
	}

	t.Run("Binding a new parameter - Success", func(t *testing.T) {
		tool := getTestTool()
		newTool, err := tool.ToolFrom(WithBindParamString("city", "London"))
		if err != nil {
			t.Fatalf("ToolFrom failed unexpectedly: %v", err)
		}
		if val, ok := newTool.boundParams["city"]; !ok || val != "London" {
			t.Errorf("Expected 'city' to be bound to 'London', but it was not")
		}
		if len(newTool.parameters) != 1 || newTool.parameters[0].Name != "days" {
			t.Error("Expected 'city' to be removed from the unbound parameters list")
		}
	})

	t.Run("Negative Test - fails when overriding an existing bound parameter", func(t *testing.T) {
		tool := getTestTool()
		_, err := tool.ToolFrom(WithBindParamString("units", "fahrenheit"))
		if err == nil {
			t.Fatal("Expected an error when overriding 'units' parameter, but got nil")
		}
		if !strings.Contains(err.Error(), "cannot override existing bound parameter: 'units'") {
			t.Errorf("Incorrect error message for override. Got: %v", err)
		}
	})

	t.Run("Negative Test - fails when overriding an existing auth source", func(t *testing.T) {
		tool := getTestTool()
		_, err := tool.ToolFrom(WithAuthTokenString("google", "new-token"))
		if err == nil {
			t.Fatal("Expected an error when overriding 'google' auth source, but got nil")
		}
		if !strings.Contains(err.Error(), "cannot override existing auth token source: 'google'") {
			t.Errorf("Incorrect error message for override. Got: %v", err)
		}
	})

	t.Run("Negative Test - fails when using WithStrict option", func(t *testing.T) {
		tool := getTestTool()
		_, err := tool.ToolFrom(WithStrict(true))
		if err == nil {
			t.Fatal("Expected an error when using WithStrict, but got nil")
		}
		if !strings.Contains(err.Error(), "WithStrict option is not applicable") {
			t.Errorf("Incorrect error message for WithStrict. Got: %v", err)
		}
	})

	t.Run("Negative Test - binding a completely unknown parameter", func(t *testing.T) {
		tool := getTestTool()
		_, err := tool.ToolFrom(WithBindParamString("country", "UK"))
		if err == nil {
			t.Fatal("Expected an error when binding an unknown parameter, but got nil")
		}
		if !strings.Contains(err.Error(), "no parameter named 'country'") {
			t.Errorf("Incorrect error message for unknown parameter. Got: %q", err.Error())
		}
	})

	t.Run("Negative Test - conflicting options are provided", func(t *testing.T) {
		tool := getTestTool()
		_, err := tool.ToolFrom(
			WithBindParamString("city", "A"),
			WithBindParamString("city", "B"),
		)
		if err == nil {
			t.Fatal("Expected an error from a duplicate option, but got nil")
		}
		if !strings.Contains(err.Error(), "duplicate parameter binding") {
			t.Errorf("Incorrect error message for conflicting options. Got: %q", err.Error())
		}
	})
}

func TestCloneToolboxTool(t *testing.T) {
	// 1. Setup an original tool with populated maps and slices to test deep copying.
	originalTransport := &dummyTransport{}
	originalTool := &ToolboxTool{
		name:        "original_tool",
		description: "An original tool to be cloned.",
		transport:   originalTransport,
		parameters: []ParameterSchema{
			{Name: "p1", Type: "string"},
		},
		boundParams: map[string]any{
			"b1":        "value1",
			"callbacks": []string{"original_func"},
		},
		authTokenSources: map[string]oauth2.TokenSource{
			"auth1": &mockTokenSource{},
		},
		requiredAuthnParams: map[string][]string{
			"req1": {"google", "github"},
		},
		requiredAuthzTokens: []string{"system_token"},
		clientHeaderSources: map[string]oauth2.TokenSource{
			"header1": &mockTokenSource{},
		},
	}

	clone := originalTool.cloneToolboxTool()

	if originalTool == clone {
		t.Fatal("Clone should not be the same instance (pointer) as the original")
	}
	if !reflect.DeepEqual(originalTool, clone) {
		t.Fatal("Initial clone is not deeply equal to the original")
	}

	if clone.transport != originalTool.transport {
		t.Error("Clone should share the same transport reference")
	}

	t.Run("Negative Test - modifying clone's boundParams map", func(t *testing.T) {
		clone.boundParams["b2"] = "newValue"
		delete(clone.boundParams, "b1")

		if _, exists := originalTool.boundParams["b2"]; exists {
			t.Error("Modifying clone's boundParams added a key to the original's map")
		}
		if _, exists := originalTool.boundParams["b1"]; !exists {
			t.Error("Modifying clone's boundParams deleted a key from the original's map")
		}
	})

	t.Run("Negative Test - modifying clone's parameters slice", func(t *testing.T) {
		clone.parameters = append(clone.parameters, ParameterSchema{Name: "p2"})

		if len(originalTool.parameters) != 1 {
			t.Errorf("Appending to clone's parameters slice changed the length of the original. Got length %d, want 1", len(originalTool.parameters))
		}
	})

	t.Run("Negative Test - modifying nested slice in requiredAuthnParams map", func(t *testing.T) {
		clone.requiredAuthnParams["req1"][0] = "overwritten_value"
		clone.requiredAuthnParams["req1"] = append(clone.requiredAuthnParams["req1"], "new_service")

		originalSlice := originalTool.requiredAuthnParams["req1"]
		if originalSlice[0] != "google" {
			t.Errorf("Modifying a nested slice in the clone's map changed a value in the original's slice. Got %q, want 'google'", originalSlice[0])
		}
		if len(originalSlice) != 2 {
			t.Errorf("Appending to a nested slice in the clone's map changed the length of the original's slice. Got length %d, want 2", len(originalSlice))
		}
	})

	t.Run("Negative Test - modifying a slice within boundParams map", func(t *testing.T) {
		// This test verifies that reference types within the boundParams map are not shared.
		// Note: The current cloneToolboxTool implementation performs a shallow copy of this map's
		// values, so this test would fail unless the clone function is updated to deep copy them.

		// Action: Modify the slice inside the clone's map.
		clonedSlice := clone.boundParams["callbacks"].([]string)
		clonedSlice[0] = "modified_func"

		// Assert: Check if the original tool's slice was affected.
		originalSlice := originalTool.boundParams["callbacks"].([]string)
		if originalSlice[0] != "original_func" {
			t.Error("Modifying a slice in the clone's boundParams affected the original (shallow copy bug)")
		}
	})

	t.Run("Negative Test - modifying clone's authTokenSources map", func(t *testing.T) {
		clone.authTokenSources["auth2"] = &mockTokenSource{}

		if len(originalTool.authTokenSources) != 1 {
			t.Errorf("Modifying clone's authTokenSources map changed the length of the original. Got length %d, want 1", len(originalTool.authTokenSources))
		}
	})
}

func TestValidateAndBuildPayload(t *testing.T) {
	// A base tool where some parameters are unbound and others are bound.
	// This setup is now logically consistent.
	baseTool := &ToolboxTool{
		parameters: []ParameterSchema{
			{Name: "city", Type: "string"},
			{Name: "days", Type: "integer"},
			// "units" and "api_key" are NOT in this slice because they are bound.
		},
		boundParams: map[string]any{
			"units": "metric", // A static bound parameter
			"api_key": func() (string, error) { // A function-based bound parameter
				return "secret-key", nil
			},
		},
	}

	t.Run("Happy Path - combines user input and bound params", func(t *testing.T) {
		input := map[string]any{
			"city": "London",
			"days": 5,
		}

		payload, err := baseTool.validateAndBuildPayload(input)
		if err != nil {
			t.Fatalf("validateAndBuildPayload failed unexpectedly: %v", err)
		}

		expectedPayload := map[string]any{
			"city":    "London",
			"days":    5,
			"units":   "metric",
			"api_key": "secret-key",
		}

		if !reflect.DeepEqual(payload, expectedPayload) {
			t.Errorf("Payload mismatch.\nExpected: %v\nGot:      %v", expectedPayload, payload)
		}
	})

	t.Run("Negative Test - fails on type validation error", func(t *testing.T) {
		input := map[string]any{
			"city": "Paris",
			"days": "five", // Incorrect type
		}

		_, err := baseTool.validateAndBuildPayload(input)

		if err == nil {
			t.Fatal("Expected a type validation error, but got nil")
		}
		if !strings.Contains(err.Error(), "expects an integer, but got string") {
			t.Errorf("Incorrect error message for type mismatch. Got: %v", err)
		}
	})

	t.Run("Negative Test - fails on extra parameter provided in input", func(t *testing.T) {
		input := map[string]any{
			"city":        "Tokyo",
			"extra_param": "this should now cause an error",
		}

		_, err := baseTool.validateAndBuildPayload(input)

		if err == nil {
			t.Fatal("Expected an error for extra parameter, but got nil")
		}
		if !strings.Contains(err.Error(), "unexpected parameter 'extra_param' provided") {
			t.Errorf("Incorrect error message for extra parameter. Got: %v", err)
		}
	})

	t.Run("Negative Test - fails when bound function returns an error", func(t *testing.T) {
		toolWithFailingFunc := &ToolboxTool{
			boundParams: map[string]any{
				"api_key": func() (string, error) {
					return "", errors.New("failed to retrieve key")
				},
			},
		}

		_, err := toolWithFailingFunc.validateAndBuildPayload(map[string]any{})

		if err == nil {
			t.Fatal("Expected an error from a failing bound function, but got nil")
		}
		if !strings.Contains(err.Error(), "failed to resolve bound parameter function for 'api_key'") {
			t.Errorf("Incorrect error message for function resolution failure. Got: %v", err)
		}
	})

	t.Run("Bound parameters overwrite user input for the same key", func(t *testing.T) {
		// This test now uses a tool where "units" is bound, so the user's input
		// for "units" will be ignored and then overwritten.
		toolWithBoundUnits := &ToolboxTool{
			parameters: []ParameterSchema{{Name: "city", Type: "string"}},
			boundParams: map[string]any{
				"units": "metric",
			},
		}

		input := map[string]any{
			"city":  "UserCity",
			"units": "imperial", // User tries to provide a value for a bound param
		}

		payload, err := toolWithBoundUnits.validateAndBuildPayload(input)
		if err != nil {
			t.Fatalf("validateAndBuildPayload failed unexpectedly: %v", err)
		}

		// Assert that the bound value 'metric' won, not the user's 'imperial'.
		if payload["units"] != "metric" {
			t.Errorf("Expected bound parameter 'units' to overwrite user input. Got '%v', want 'metric'", payload["units"])
		}
		if payload["city"] != "UserCity" {
			t.Error("User-provided parameter 'city' was not included in final payload")
		}
	})
}

type errorReader struct{}

func (er *errorReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("simulated read error")
}

// failingTransport is a custom transport to inject a failing reader
type failingTransport struct{}

func (ft *failingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(&errorReader{}),
	}, nil
}

func TestToolboxTool_Invoke(t *testing.T) {
	// A base tool for successful invocations
	createBaseTool := func(httpClient *http.Client, baseURL string) *ToolboxTool {
		tr := toolboxtransport.New(baseURL, httpClient)

		return &ToolboxTool{
			name:        "weather",
			description: "Get the weather",
			transport:   tr,
			parameters: []ParameterSchema{
				{Name: "city", Type: "string"},
			},
			boundParams: map[string]any{
				"units": "metric",
			},
			authTokenSources: map[string]oauth2.TokenSource{
				"weather_api": oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "api-token-123"}),
			},
			clientHeaderSources: map[string]oauth2.TokenSource{
				"X-Client-Version": oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "v1.0.0"}),
			},
			requiredAuthzTokens: []string{"weather_api"},
		}
	}

	t.Run("Successful invocation", func(t *testing.T) {
		// Setup a mock server that validates the incoming request
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Content-Type") != "application/json" {
				t.Error("Request missing Content-Type header")
			}
			if r.Header.Get("X-Client-Version") != "v1.0.0" {
				t.Error("Request missing client version header")
			}
			if r.Header.Get("weather_api_token") != "api-token-123" {
				t.Error("Request missing auth token header")
			}

			body, _ := io.ReadAll(r.Body)
			var payload map[string]any
			_ = json.Unmarshal(body, &payload)
			if payload["city"] != "London" || payload["units"] != "metric" {
				t.Errorf("Received incorrect payload: %v", payload)
			}

			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{"result": "sunny"})
		}))
		defer server.Close()

		tool := createBaseTool(server.Client(), server.URL)
		result, err := tool.Invoke(context.Background(), map[string]any{"city": "London"})

		if err != nil {
			t.Fatalf("Invoke failed unexpectedly: %v", err)
		}
		if result != "sunny" {
			t.Errorf("Expected result 'sunny', got '%v'", result)
		}
	})

	t.Run("Applies correct _token suffix to auth headers but not client headers", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Assert client header is present without suffix
			if r.Header.Get("X-Custom-Header") != "client-val" {
				t.Errorf("Expected client header 'X-Custom-Header' to be 'client-val', got %q", r.Header.Get("X-Custom-Header"))
			}

			// Assert auth token header is present with suffix
			if r.Header.Get("my_auth_token") != "auth-val" {
				t.Errorf("Expected auth header 'my_auth_token' to be 'auth-val', got %q", r.Header.Get("my_auth_token"))
			}

			// Assert auth token header is NOT present without suffix
			if r.Header.Get("my_auth") != "" {
				t.Errorf("Auth header 'my_auth' should not exist, but it does")
			}

			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{"result": "ok"})
		}))
		defer server.Close()

		tool := createBaseTool(server.Client(), server.URL)
		// Add the specific headers for this test
		tool.clientHeaderSources["X-Custom-Header"] = oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "client-val"})
		tool.authTokenSources["my_auth"] = oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "auth-val"})

		_, err := tool.Invoke(context.Background(), map[string]any{"city": "London"})
		if err != nil {
			t.Fatalf("Invoke failed unexpectedly: %v", err)
		}
	})

	t.Run("Auth token headers overwrite client headers with the same final name", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("special_api_token") != "auth_value_should_win" {
				t.Errorf("Expected header 'special_api_token' to be overwritten by auth token source, but it was not. Got: %q", r.Header.Get("special_api_token"))
			}
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{"result": "ok"})
		}))
		defer server.Close()

		tool := createBaseTool(server.Client(), server.URL)
		tool.clientHeaderSources["special_api_token"] = oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "client_value_should_lose"})
		tool.authTokenSources["special_api"] = oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "auth_value_should_win"})

		_, err := tool.Invoke(context.Background(), map[string]any{"city": "London"})
		if err != nil {
			t.Fatalf("Invoke failed unexpectedly: %v", err)
		}
	})

	t.Run("Negative Test - Fails when http client is nil", func(t *testing.T) {
		tool := createBaseTool(nil, "") // httpClient is nil
		_, err := tool.Invoke(context.Background(), nil)

		if err == nil {
			t.Fatal("Expected an error for nil http client, but got nil")
		}
		if !strings.Contains(err.Error(), "http client is not set") {
			t.Errorf("Incorrect error message for nil client. Got: %v", err)
		}
	})

	t.Run("Negative Test - Fails when required auth is missing", func(t *testing.T) {
		tool := createBaseTool(http.DefaultClient, "")
		tool.requiredAuthzTokens = []string{"required_service"} // This service is not in authTokenSources

		_, err := tool.Invoke(context.Background(), nil)

		if err == nil {
			t.Fatal("Expected an error for missing auth service, but got nil")
		}
		if !strings.Contains(err.Error(), "permission error: auth service 'required_service' is required") {
			t.Errorf("Incorrect error message for missing auth. Got: %v", err)
		}
	})

	t.Run("Negative Test - Fails when payload validation fails", func(t *testing.T) {
		tool := createBaseTool(http.DefaultClient, "")

		// Pass an extra parameter, which should be rejected
		_, err := tool.Invoke(context.Background(), map[string]any{"extra": "param"})

		if err == nil {
			t.Fatal("Expected an error from payload validation, but got nil")
		}
		if !strings.Contains(err.Error(), "unexpected parameter 'extra' provided") {
			t.Errorf("Incorrect error message for payload validation. Got: %v", err)
		}
	})

	t.Run("Negative Test - Fails when server returns an error status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": "invalid city format"})
		}))
		defer server.Close()

		tool := createBaseTool(server.Client(), server.URL)
		_, err := tool.Invoke(context.Background(), map[string]any{"city": "London"})

		if err == nil {
			t.Fatal("Expected an error from a non-200 server response, but got nil")
		}
		if !strings.Contains(err.Error(), "API returned error status 400: invalid city format") {
			t.Errorf("Incorrect error message for server error. Got: %v", err)
		}
	})

	t.Run("Success Path - Handles non-json successful response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("Plain text success message"))
		}))
		defer server.Close()

		tool := createBaseTool(server.Client(), server.URL)
		result, err := tool.Invoke(context.Background(), map[string]any{"city": "London"})

		if err != nil {
			t.Fatalf("Invoke failed unexpectedly for plain text response: %v", err)
		}
		if result != "Plain text success message" {
			t.Errorf("Expected plain text result, got '%v'", result)
		}
	})

	t.Run("Negative Test - Fails when required AuthN (param-level) is missing", func(t *testing.T) {
		tool := createBaseTool(http.DefaultClient, "")
		// This tool requires a 'google' token for one of its parameters.
		tool.requiredAuthnParams = map[string][]string{
			"user_location": {"google"},
		}
		// The base tool does not provide the 'google' token source.

		_, err := tool.Invoke(context.Background(), map[string]any{"city": "London"})

		if err == nil {
			t.Fatal("Expected an error for missing AuthN service, but got nil")
		}
		if !strings.Contains(err.Error(), "permission error: auth service 'google' is required") {
			t.Errorf("Incorrect error message for missing param-level auth. Got: %v", err)
		}
	})

	t.Run("Negative Test - Fails when required AuthZ (tool-level) is missing", func(t *testing.T) {
		tool := createBaseTool(http.DefaultClient, "")
		tool.requiredAuthzTokens = []string{"required_service"} // This service is not in authTokenSources

		_, err := tool.Invoke(context.Background(), nil)

		if err == nil {
			t.Fatal("Expected an error for missing auth service, but got nil")
		}
		if !strings.Contains(err.Error(), "permission error: auth service 'required_service' is required") {
			t.Errorf("Incorrect error message for missing tool-level auth. Got: %v", err)
		}
	})

	t.Run("Negative Test - Fails when server returns an error status with non-JSON body", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("Internal Server Error"))
		}))
		defer server.Close()

		tool := createBaseTool(server.Client(), server.URL)
		_, err := tool.Invoke(context.Background(), map[string]any{"city": "London"})

		if err == nil {
			t.Fatal("Expected an error from a non-200 server response, but got nil")
		}
		if !strings.Contains(err.Error(), "API returned unexpected status: 500") {
			t.Errorf("Incorrect error message for non-JSON server error. Got: %v", err)
		}
	})

	t.Run("Negative Test - Fails when API call itself fails", func(t *testing.T) {
		// Create a server and immediately close it to simulate a network error
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		server.Close()

		tool := createBaseTool(server.Client(), server.URL)
		_, err := tool.Invoke(context.Background(), map[string]any{"city": "London"})

		if err == nil {
			t.Fatal("Expected an error from a failed API call, but got nil")
		}
		if !strings.Contains(err.Error(), "HTTP call to tool 'weather' failed") {
			t.Errorf("Incorrect error message for failed API call. Got: %v", err)
		}
	})

	t.Run("Negative Test - Fails when API call itself fails", func(t *testing.T) {
		// Create a server and immediately close it to simulate a network error
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		server.Close()

		tool := createBaseTool(server.Client(), server.URL)
		_, err := tool.Invoke(context.Background(), map[string]any{"city": "London"})

		if err == nil {
			t.Fatal("Expected an error from a failed API call, but got nil")
		}
		if !strings.Contains(err.Error(), "HTTP call to tool 'weather' failed") {
			t.Errorf("Incorrect error message for failed API call. Got: %v", err)
		}
	})

	t.Run("Negative Test - Fails when reading response body fails", func(t *testing.T) {
		// The mock server for this test case is intentionally minimal,
		// as the failure is injected via the custom http.Client.
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		defer server.Close()

		failingClient := &http.Client{Transport: &failingTransport{}}
		tool := createBaseTool(failingClient, server.URL)

		_, err := tool.Invoke(context.Background(), map[string]any{"city": "London"})
		if err == nil {
			t.Fatal("Expected an error from a failing response body read, but got nil")
		}
		if !strings.Contains(err.Error(), "failed to read response body") {
			t.Errorf("Incorrect error message for failed body read. Got: %v", err)
		}
	})

}
func TestToolboxTool_Invoke_HttpsWarning(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(nil)
	mockTokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "secret-token"})

	tests := []struct {
		name          string
		baseURL       string
		expectWarning bool
	}{
		{
			name:          "Warning triggered for HTTP",
			baseURL:       "http://api.example.com",
			expectWarning: true,
		},
		{
			name:          "No warning for HTTPS",
			baseURL:       "https://api.example.com",
			expectWarning: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()

			tool := &ToolboxTool{
				name:      "test-tool",
				transport: &dummyTransport{baseURL: tt.baseURL},
				authTokenSources: map[string]oauth2.TokenSource{
					"service_a": mockTokenSource,
				},
				boundParams: make(map[string]any),
			}

			_, err := tool.Invoke(context.Background(), nil)
			if err != nil {
				t.Fatalf("Invoke failed: %v", err)
			}

			logOutput := buf.String()
			hasWarning := strings.Contains(logOutput, "WARNING: Sending ID token over HTTP")

			if tt.expectWarning && !hasWarning {
				t.Errorf("Expected warning for URL %s, but none was logged", tt.baseURL)
			}
			if !tt.expectWarning && hasWarning {
				t.Errorf("Did not expect warning for URL %s, but one was logged: %s", tt.baseURL, logOutput)
			}
		})
	}
}

// TestInputSchema tests the JSON output of the InputSchema method.
func TestInputSchema(t *testing.T) {
	testCases := []struct {
		name         string
		tool         *ToolboxTool
		expectedJSON string
	}{
		{
			name: "Tool with simple and required parameters",
			tool: &ToolboxTool{
				parameters: []ParameterSchema{
					{Name: "location", Type: "string", Description: "City and state", Required: true},
					{Name: "days", Type: "integer", Description: "Number of days", Required: false},
				},
			},
			expectedJSON: `{
                "type": "object",
                "properties": {
                    "location": {
                        "type": "string",
                        "description": "City and state"
                    },
                    "days": {
                        "type": "integer",
                        "description": "Number of days"
                    }
                },
                "required": ["location"]
            }`,
		},
		{
			name: "Tool with a nested array parameter",
			tool: &ToolboxTool{
				parameters: []ParameterSchema{
					{
						Name:        "items",
						Type:        "array",
						Description: "A list of items",
						Required:    true,
						Items: &ParameterSchema{
							Type:        "string",
							Description: "An item name",
						},
					},
				},
			},
			expectedJSON: `{
                "type": "object",
                "properties": {
                    "items": {
                        "type": "array",
                        "description": "A list of items",
                        "items": {
                            "type": "string",
                            "description": "An item name"
                        }
                    }
                },
                "required": ["items"]
            }`,
		},
		{
			name: "Tool with no required parameters",
			tool: &ToolboxTool{
				parameters: []ParameterSchema{
					{Name: "query", Type: "string", Description: "A search query", Required: false},
				},
			},
			expectedJSON: `{
                "type": "object",
                "properties": {
                    "query": {
                        "type": "string",
                        "description": "A search query"
                    }
                }
            }`,
		},
		{
			name: "Tool with no parameters",
			tool: &ToolboxTool{
				parameters: []ParameterSchema{},
			},
			expectedJSON: `{
                "type": "object",
                "properties": {}
            }`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Get the actual JSON from the method
			actualBytes, err := tc.tool.InputSchema()
			if err != nil {
				t.Fatalf("InputSchema() returned an unexpected error: %v", err)
			}

			// Unmarshal both actual and expected into maps for a robust comparison
			var actualMap, expectedMap map[string]interface{}

			if err := json.Unmarshal(actualBytes, &actualMap); err != nil {
				t.Fatalf("Failed to unmarshal actual JSON: %v", err)
			}
			if err := json.Unmarshal([]byte(tc.expectedJSON), &expectedMap); err != nil {
				t.Fatalf("Failed to unmarshal expected JSON: %v", err)
			}

			// Compare the resulting maps
			if !reflect.DeepEqual(actualMap, expectedMap) {
				t.Errorf("Mismatched JSON output.\nGot:\n%s\n\nWant:\n%s", string(actualBytes), tc.expectedJSON)
			}
		})
	}
}
