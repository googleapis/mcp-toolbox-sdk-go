//go:build unit

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
	"bytes"
	"errors"
	"log"
	"reflect"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// mockingTokenSource is a helper to simulate token generation behavior.
type mockingTokenSource struct {
	token *oauth2.Token
	err   error
}

func (m *mockingTokenSource) Token() (*oauth2.Token, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.token, nil
}

// Enforcing the TokenSource type on the mockingTokenSource
var _ oauth2.TokenSource = &mockingTokenSource{}

func TestResolveClientHeaders(t *testing.T) {
	t.Run("Success_MultipleHeaders", func(t *testing.T) {
		// Setup input map directly
		sources := map[string]oauth2.TokenSource{
			"Authorization":   &mockingTokenSource{token: &oauth2.Token{AccessToken: "bearer-token"}},
			"X-Custom-Header": &mockingTokenSource{token: &oauth2.Token{AccessToken: "custom-value"}},
		}

		// Execute function directly
		headers, err := resolveClientHeaders(sources)

		// Verify
		require.NoError(t, err)
		assert.Len(t, headers, 2)
		assert.Equal(t, "bearer-token", headers["Authorization"])
		assert.Equal(t, "custom-value", headers["X-Custom-Header"])
	})

	t.Run("Success_Empty", func(t *testing.T) {
		sources := make(map[string]oauth2.TokenSource)

		headers, err := resolveClientHeaders(sources)

		require.NoError(t, err)
		assert.Empty(t, headers)
		assert.NotNil(t, headers) // Ensure we get a map, not nil
	})

	t.Run("Failure_SingleSourceError", func(t *testing.T) {
		// Setup: One valid source, one failing source
		sources := map[string]oauth2.TokenSource{
			"Valid-Header":  &mockingTokenSource{token: &oauth2.Token{AccessToken: "ok"}},
			"Broken-Header": &mockingTokenSource{err: errors.New("network timeout")},
		}

		// Execute
		headers, err := resolveClientHeaders(sources)

		// Verify
		require.Error(t, err)
		assert.Nil(t, headers, "Should return nil map on error")

		// Check error wrapping
		assert.Contains(t, err.Error(), "failed to resolve client header 'Broken-Header'")
		assert.Contains(t, err.Error(), "network timeout")
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

func TestSchemaToMap(t *testing.T) {
	// Define test cases
	testCases := []struct {
		name     string
		input    *ParameterSchema
		expected map[string]any
	}{
		{
			name: "Simple String Parameter",
			input: &ParameterSchema{
				Type:        "string",
				Description: "A simple string input.",
			},
			expected: map[string]any{
				"type":        "string",
				"description": "A simple string input.",
			},
		},
		{
			name: "Array of Integers Parameter",
			input: &ParameterSchema{
				Type:        "array",
				Description: "A list of numbers.",
				Items: &ParameterSchema{
					Type:        "integer",
					Description: "A single number.",
				},
			},
			expected: map[string]any{
				"type":        "array",
				"description": "A list of numbers.",
				"items": map[string]any{
					"type":        "integer",
					"description": "A single number.",
				},
			},
		},
		{
			name: "Array with nil Items",
			input: &ParameterSchema{
				Type:        "array",
				Description: "An array with no defined item type.",
				Items:       nil,
			},
			expected: map[string]any{
				"type":        "array",
				"description": "An array with no defined item type.",
			},
		},
		{
			name: "Parameter with Empty Description",
			input: &ParameterSchema{
				Type:        "boolean",
				Description: "",
			},
			expected: map[string]any{
				"type": "boolean",
			},
		},
		{
			name: "Parameter with Default Value",
			input: &ParameterSchema{
				Type:        "string",
				Description: "Filter with a default",
				Default:     "active",
			},
			expected: map[string]any{
				"type":        "string",
				"description": "Filter with a default",
				"default":     "active",
			},
		},
	}

	// Run test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := schemaToMap(tc.input)
			if !reflect.DeepEqual(actual, tc.expected) {
				t.Errorf("schemaToMap() = %v, want %v", actual, tc.expected)
			}
		})
	}
}

func TestMapToSchema(t *testing.T) {
	testCases := []struct {
		name           string
		input          map[string]any
		expectedSchema *ParameterSchema
		expectErr      bool
	}{
		{
			name: "Success - Simple valid map",
			input: map[string]any{
				"name":        "location",
				"type":        "string",
				"description": "The city name.",
				"required":    true,
			},
			expectedSchema: &ParameterSchema{
				Name:        "location",
				Type:        "string",
				Description: "The city name.",
				Required:    true,
			},
			expectErr: false,
		},
		{
			name: "Success - Map with extra fields",
			input: map[string]any{
				"name":        "query",
				"type":        "string",
				"extra_field": "should be ignored",
			},
			expectedSchema: &ParameterSchema{
				Name: "query",
				Type: "string",
			},
			expectErr: false,
		},
		{
			name:           "Success - Empty map",
			input:          map[string]any{},
			expectedSchema: &ParameterSchema{},
			expectErr:      false,
		},
		{
			name:           "Success - Nil map",
			input:          nil,
			expectedSchema: &ParameterSchema{},
			expectErr:      false,
		},
		{
			name: "Failure - Invalid data type for field",
			input: map[string]any{
				"name":     "toggle",
				"type":     "boolean",
				"required": "yes",
			},
			expectedSchema: nil,
			expectErr:      true,
		},
		{
			name: "Failure - Unmarshallable map value",
			input: map[string]any{
				"name": "bad_map",
				"type": make(chan int),
			},
			expectedSchema: nil,
			expectErr:      true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actualSchema, err := mapToSchema(tc.input)

			if tc.expectErr {
				if err == nil {
					t.Errorf("expected an error, but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("did not expect an error, but got: %v", err)
				}
				if !reflect.DeepEqual(actualSchema, tc.expectedSchema) {
					t.Errorf("expected schema:\n%+v\nbut got:\n%+v", tc.expectedSchema, actualSchema)
				}
			}
		})
	}
}

func captureLogOutput(f func()) string {
	var buf bytes.Buffer
	original := log.Writer()
	log.SetOutput(&buf)
	defer log.SetOutput(original) // Restore original logger
	f()
	return buf.String()
}

func TestCheckSecureHeaders(t *testing.T) {
	t.Run("Logs warning when HTTP and sensitive data presence", func(t *testing.T) {
		output := captureLogOutput(func() {
			checkSecureHeaders("http://example.com", true)
		})
		assert.Contains(t, output, "WARNING: This connection is using HTTP")
	})

	t.Run("Does not log warning when HTTPS", func(t *testing.T) {
		output := captureLogOutput(func() {
			checkSecureHeaders("https://example.com", true)
		})
		assert.NotContains(t, output, "WARNING: This connection is using HTTP")
	})

	t.Run("Does not log warning when no sensitive data", func(t *testing.T) {
		output := captureLogOutput(func() {
			checkSecureHeaders("http://example.com", false)
		})
		assert.NotContains(t, output, "WARNING: This connection is using HTTP")
	})
}
