//go:build e2e

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

package core_test

import (
	"context"
	"errors"
	"log"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	"github.com/googleapis/mcp-toolbox-sdk-go/core"
)

// Global variables to hold session-scoped "fixtures"
var (
	projectID       string = getEnvVar("GOOGLE_CLOUD_PROJECT")
	toolboxVersion  string = getEnvVar("TOOLBOX_VERSION")
	authToken1      string
	authToken2      string
	manifestVersion string = getEnvVar("TOOLBOX_MANIFEST_VERSION")
)

func TestMain(m *testing.M) {
	ctx := context.Background()
	log.Println("Starting E2E test setup...")

	// Get secrets and auth tokens
	log.Println("Fetching secrets and auth tokens...")
	toolsManifestContent := accessSecretVersion(ctx, projectID, "sdk_testing_tools", manifestVersion)
	clientID1 := accessSecretVersion(ctx, projectID, "sdk_testing_client1", "latest")
	clientID2 := accessSecretVersion(ctx, projectID, "sdk_testing_client2", "latest")
	authToken1 = getAuthToken(ctx, clientID1)
	authToken2 = getAuthToken(ctx, clientID2)

	// Create a temporary file for the tools manifest
	toolsFile, err := os.CreateTemp("", "tools-*.json")
	if err != nil {
		log.Fatalf("Failed to create temp file for tools: %v", err)
	}
	if _, err := toolsFile.WriteString(toolsManifestContent); err != nil {
		log.Fatalf("Failed to write to temp file: %v", err)
	}
	toolsFile.Close()
	toolsFilePath := toolsFile.Name()
	defer os.Remove(toolsFilePath) // Ensure cleanup

	// Download and start the toolbox server
	cmd := setupAndStartToolboxServer(ctx, toolboxVersion, toolsFilePath)

	// --- 2. Run Tests ---
	log.Println("Setup complete. Running tests...")
	exitCode := m.Run()

	// --- 3. Teardown Phase ---
	log.Println("Tearing down toolbox server...")
	if err := cmd.Process.Kill(); err != nil {
		log.Printf("Failed to kill toolbox server process: %v", err)
	}
	_ = cmd.Wait() // Clean up the process resources

	os.Exit(exitCode)
}

func TestE2E_Basic(t *testing.T) {
	// Helper to create a new client for each sub-test, like a function-scoped fixture
	newClient := func(t *testing.T) *core.ToolboxClient {
		client, err := core.NewToolboxClient("http://localhost:5000")
		require.NoError(t, err, "Failed to create ToolboxClient")
		return client
	}

	// Helper to load the get-n-rows tool, like the get_n_rows_tool fixture
	getNRowsTool := func(t *testing.T, client *core.ToolboxClient) *core.ToolboxTool {
		tool, err := client.LoadTool("get-n-rows", context.Background())
		require.NoError(t, err, "Failed to load tool 'get-n-rows'")
		require.Equal(t, "get-n-rows", tool.Name())
		return tool
	}

	t.Run("test_load_toolset_specific", func(t *testing.T) {
		testCases := []struct {
			name           string
			toolsetName    string
			expectedLength int
			expectedTools  []string
		}{
			{"my-toolset", "my-toolset", 1, []string{"get-row-by-id"}},
			{"my-toolset-2", "my-toolset-2", 2, []string{"get-n-rows", "get-row-by-id"}},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				client := newClient(t)
				toolset, err := client.LoadToolset(tc.toolsetName, context.Background())

				require.NoError(t, err)
				assert.Len(t, toolset, tc.expectedLength)

				toolNames := make(map[string]struct{})
				for _, tool := range toolset {
					toolNames[tool.Name()] = struct{}{}
				}
				expectedToolsSet := make(map[string]struct{})
				for _, name := range tc.expectedTools {
					expectedToolsSet[name] = struct{}{}
				}
				assert.Equal(t, expectedToolsSet, toolNames)
			})
		}
	})

	t.Run("test_load_toolset_default", func(t *testing.T) {
		client := newClient(t)
		toolset, err := client.LoadToolset("", context.Background())
		require.NoError(t, err)

		assert.Len(t, toolset, 7)
		toolNames := make(map[string]struct{})
		for _, tool := range toolset {
			toolNames[tool.Name()] = struct{}{}
		}
		expectedTools := map[string]struct{}{
			"get-row-by-content-auth": {},
			"get-row-by-email-auth":   {},
			"get-row-by-id-auth":      {},
			"get-row-by-id":           {},
			"get-n-rows":              {},
			"search-rows":             {},
			"process-data":            {},
		}
		assert.Equal(t, expectedTools, toolNames)
	})

	t.Run("test_run_tool", func(t *testing.T) {
		client := newClient(t)
		tool := getNRowsTool(t, client)

		response, err := tool.Invoke(context.Background(), map[string]any{"num_rows": "2"})
		require.NoError(t, err)

		respStr, ok := response.(string)
		require.True(t, ok, "Response should be a string")
		assert.Contains(t, respStr, "row1")
		assert.Contains(t, respStr, "row2")
		assert.NotContains(t, respStr, "row3")
	})

	t.Run("test_run_tool_missing_params", func(t *testing.T) {
		client := newClient(t)
		tool := getNRowsTool(t, client)

		// The Go SDK performs validation inside Invoke, so we check the error there.
		_, err := tool.Invoke(context.Background(), map[string]any{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing required parameter 'num_rows'")
	})

	t.Run("test_run_tool_wrong_param_type", func(t *testing.T) {
		client := newClient(t)
		tool := getNRowsTool(t, client)

		_, err := tool.Invoke(context.Background(), map[string]any{"num_rows": 2}) // Pass int instead of string
		require.Error(t, err)
		assert.Contains(t, err.Error(), "parameter 'num_rows' expects a string, but got int")
	})
}

func TestE2E_LoadErrors(t *testing.T) {
	newClient := func(t *testing.T) *core.ToolboxClient {
		client, err := core.NewToolboxClient("http://localhost:5000")
		require.NoError(t, err, "Failed to create ToolboxClient")
		return client
	}

	t.Run("test_load_non_existent_tool", func(t *testing.T) {
		client := newClient(t)
		_, err := client.LoadTool("non-existent-tool", context.Background())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "server returned non-OK status: 404")
	})

	t.Run("test_load_non_existent_toolset", func(t *testing.T) {
		client := newClient(t)
		_, err := client.LoadToolset("non-existent-toolset", context.Background())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "server returned non-OK status: 404")
	})

	t.Run("test_new_client_with_nil_option", func(t *testing.T) {
		_, err := core.NewToolboxClient("http://localhost:5000", nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "received a nil ClientOption")
	})

	t.Run("test_load_tool_with_nil_option", func(t *testing.T) {
		client := newClient(t)
		_, err := client.LoadTool("get-n-rows", context.Background(), nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "received a nil ToolOption")
	})
}

func TestE2E_BindParams(t *testing.T) {
	newClient := func(t *testing.T) *core.ToolboxClient {
		client, err := core.NewToolboxClient("http://localhost:5000")
		require.NoError(t, err)
		return client
	}
	getNRowsTool := func(t *testing.T, client *core.ToolboxClient) *core.ToolboxTool {
		tool, err := client.LoadTool("get-n-rows", context.Background())
		require.NoError(t, err)
		return tool
	}

	t.Run("test_bind_params", func(t *testing.T) {
		client := newClient(t)
		tool := getNRowsTool(t, client)

		newTool, err := tool.ToolFrom(core.WithBindParamString("num_rows", "3"))
		require.NoError(t, err)

		response, err := newTool.Invoke(context.Background(), map[string]any{})
		require.NoError(t, err)

		respStr, ok := response.(string)
		require.True(t, ok)
		assert.Contains(t, respStr, "row1")
		assert.Contains(t, respStr, "row2")
		assert.Contains(t, respStr, "row3")
		assert.NotContains(t, respStr, "row4")
	})

	t.Run("test_bind_params_callable", func(t *testing.T) {
		client := newClient(t)
		tool := getNRowsTool(t, client)

		callable := func() (string, error) {
			return "3", nil
		}

		newTool, err := tool.ToolFrom(core.WithBindParamStringFunc("num_rows", callable))
		require.NoError(t, err)

		response, err := newTool.Invoke(context.Background(), map[string]any{})
		require.NoError(t, err)

		respStr, ok := response.(string)
		require.True(t, ok)
		assert.Contains(t, respStr, "row1")
		assert.Contains(t, respStr, "row2")
		assert.Contains(t, respStr, "row3")
		assert.NotContains(t, respStr, "row4")
	})
}

func TestE2E_BindParamErrors(t *testing.T) {
	client, err := core.NewToolboxClient("http://localhost:5000")
	require.NoError(t, err)
	tool, err := client.LoadTool("get-n-rows", context.Background())
	require.NoError(t, err)

	t.Run("test_bind_non_existent_param", func(t *testing.T) {
		_, err := tool.ToolFrom(core.WithBindParamString("non-existent-param", "3"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unable to bind parameter: no parameter named 'non-existent-param' on the tool")
	})

	t.Run("test_override_bound_param", func(t *testing.T) {
		newTool, err := tool.ToolFrom(core.WithBindParamString("num_rows", "2"))
		require.NoError(t, err)

		_, err = newTool.ToolFrom(core.WithBindParamString("num_rows", "3"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot override existing bound parameter: 'num_rows'")
	})
}

func TestE2E_Auth(t *testing.T) {
	newClient := func(t *testing.T) *core.ToolboxClient {
		client, err := core.NewToolboxClient("http://localhost:5000")
		require.NoError(t, err)
		return client
	}

	// Helper to create a static token source from a string token
	staticTokenSource := func(token string) oauth2.TokenSource {
		return oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	}

	t.Run("test_run_tool_unauth_with_auth", func(t *testing.T) {
		client := newClient(t)
		_, err := client.LoadTool("get-row-by-id", context.Background(),
			core.WithAuthTokenSource("my-test-auth", staticTokenSource(authToken2)),
		)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "validation failed for tool 'get-row-by-id': unused auth tokens: my-test-auth")
	})

	t.Run("test_run_tool_no_auth", func(t *testing.T) {
		client := newClient(t)
		tool, err := client.LoadTool("get-row-by-id-auth", context.Background())
		require.NoError(t, err)

		_, err = tool.Invoke(context.Background(), map[string]any{"id": "2"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "permission error: auth service 'my-test-auth' is required")
	})

	t.Run("test_run_tool_wrong_auth", func(t *testing.T) {
		client := newClient(t)
		tool, err := client.LoadTool("get-row-by-id-auth", context.Background())
		require.NoError(t, err)

		authedTool, err := tool.ToolFrom(
			core.WithAuthTokenSource("my-test-auth", staticTokenSource(authToken2)),
		)
		require.NoError(t, err)

		_, err = authedTool.Invoke(context.Background(), map[string]any{"id": "2"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "tool invocation not authorized")
	})

	t.Run("test_run_tool_auth", func(t *testing.T) {
		client := newClient(t)
		tool, err := client.LoadTool("get-row-by-id-auth", context.Background(),
			core.WithAuthTokenSource("my-test-auth", staticTokenSource(authToken1)),
		)
		require.NoError(t, err)

		response, err := tool.Invoke(context.Background(), map[string]any{"id": "2"})
		require.NoError(t, err)

		respStr, ok := response.(string)
		require.True(t, ok)
		assert.Contains(t, respStr, "row2")
	})

	t.Run("test_run_tool_param_auth_no_auth", func(t *testing.T) {
		client := newClient(t)
		tool, err := client.LoadTool("get-row-by-email-auth", context.Background())
		require.NoError(t, err)

		_, err = tool.Invoke(context.Background(), map[string]any{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "permission error: auth service 'my-test-auth' is required")
	})

	t.Run("test_run_tool_param_auth", func(t *testing.T) {
		client := newClient(t)
		tool, err := client.LoadTool("get-row-by-email-auth", context.Background(),
			core.WithAuthTokenSource("my-test-auth", staticTokenSource(authToken1)),
		)
		require.NoError(t, err)

		response, err := tool.Invoke(context.Background(), map[string]any{})
		require.NoError(t, err)

		respStr, ok := response.(string)
		require.True(t, ok)
		assert.Contains(t, respStr, "row4")
		assert.Contains(t, respStr, "row5")
		assert.Contains(t, respStr, "row6")
	})

	t.Run("test_run_tool_param_auth_no_field", func(t *testing.T) {
		client := newClient(t)
		tool, err := client.LoadTool("get-row-by-content-auth", context.Background(),
			core.WithAuthTokenSource("my-test-auth", staticTokenSource(authToken1)),
		)
		require.NoError(t, err)

		_, err = tool.Invoke(context.Background(), map[string]any{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no field named row_data in claims")
	})

	t.Run("test_run_tool_with_failing_token_source", func(t *testing.T) {
		client := newClient(t)
		tool, err := client.LoadTool("get-row-by-id-auth", context.Background(),
			core.WithAuthTokenSource("my-test-auth", &failingTokenSource{}),
		)
		require.NoError(t, err)

		_, err = tool.Invoke(context.Background(), map[string]any{"id": "2"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get token for service 'my-test-auth'")
		assert.Contains(t, err.Error(), "token source failed as designed")
	})
}

func TestE2E_OptionalParams(t *testing.T) {
	// Helper to create a new client
	newClient := func(t *testing.T) *core.ToolboxClient {
		client, err := core.NewToolboxClient("http://localhost:5000")
		require.NoError(t, err, "Failed to create ToolboxClient")
		return client
	}

	// Helper to load the search-rows tool
	searchRowsTool := func(t *testing.T, client *core.ToolboxClient) *core.ToolboxTool {
		tool, err := client.LoadTool("search-rows", context.Background())
		require.NoError(t, err, "Failed to load tool 'search-rows'")
		return tool
	}

	t.Run("test_tool_schema_is_correct", func(t *testing.T) {
		client := newClient(t)
		tool := searchRowsTool(t, client)
		params := tool.Parameters()

		// Convert slice to map for easy lookup
		paramMap := make(map[string]core.ParameterSchema)
		for _, p := range params {
			paramMap[p.Name] = p
		}

		// Check required parameter 'email'
		emailParam, ok := paramMap["email"]
		require.True(t, ok, "email parameter should exist")
		assert.True(t, emailParam.Required, "'email' should be required")
		assert.Equal(t, "string", emailParam.Type)

		// Check optional parameter 'data'
		dataParam, ok := paramMap["data"]
		require.True(t, ok, "data parameter should exist")
		assert.False(t, dataParam.Required, "'data' should be optional")
		assert.Equal(t, "string", dataParam.Type)

		// Check optional parameter 'id'
		idParam, ok := paramMap["id"]
		require.True(t, ok, "id parameter should exist")
		assert.False(t, idParam.Required, "'id' should be optional")
		assert.Equal(t, "integer", idParam.Type)
	})

	t.Run("test_run_tool_omitting_optionals", func(t *testing.T) {
		client := newClient(t)
		tool := searchRowsTool(t, client)

		// Test case 1: Optional params are completely omitted
		response1, err1 := tool.Invoke(context.Background(), map[string]any{
			"email": "twishabansal@google.com",
		})
		require.NoError(t, err1)
		respStr1, ok1 := response1.(string)
		require.True(t, ok1)
		assert.Contains(t, respStr1, `"email":"twishabansal@google.com"`)
		assert.Contains(t, respStr1, "row2")
		assert.NotContains(t, respStr1, "row3")

		// Test case 2: Optional params are explicitly nil
		// This should produce the same result as omitting them
		response2, err2 := tool.Invoke(context.Background(), map[string]any{
			"email": "twishabansal@google.com",
			"data":  nil,
			"id":    nil,
		})
		require.NoError(t, err2)
		respStr2, ok2 := response2.(string)
		require.True(t, ok2)
		assert.Equal(t, respStr1, respStr2)
	})

	t.Run("test_run_tool_with_all_params_provided", func(t *testing.T) {
		client := newClient(t)
		tool := searchRowsTool(t, client)
		response, err := tool.Invoke(context.Background(), map[string]any{
			"email": "twishabansal@google.com",
			"data":  "row3",
			"id":    3,
		})
		require.NoError(t, err)
		respStr, ok := response.(string)
		require.True(t, ok)
		assert.Contains(t, respStr, `"email":"twishabansal@google.com"`)
		assert.Contains(t, respStr, `"id":3`)
		assert.Contains(t, respStr, "row3")
		assert.NotContains(t, respStr, "row2")
	})

	t.Run("test_run_tool_missing_required_param", func(t *testing.T) {
		client := newClient(t)
		tool := searchRowsTool(t, client)
		_, err := tool.Invoke(context.Background(), map[string]any{
			"data": "row5",
			"id":   5,
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing required parameter 'email'")
	})

	t.Run("test_run_tool_required_param_is_nil", func(t *testing.T) {
		client := newClient(t)
		tool := searchRowsTool(t, client)
		_, err := tool.Invoke(context.Background(), map[string]any{
			"email": nil,
			"id":    5,
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "parameter 'email' is required but received a nil value")
	})

	// Corresponds to tests that check server-side logic by providing data that doesn't match
	t.Run("test_run_tool_with_non_matching_data", func(t *testing.T) {
		client := newClient(t)
		tool := searchRowsTool(t, client)

		// Test with a different email
		response, err := tool.Invoke(context.Background(), map[string]any{
			"email": "anubhavdhawan@google.com",
			"id":    3,
			"data":  "row3",
		})
		require.NoError(t, err)
		assert.Equal(t, "null", response, "Response should be null for non-matching email")

		// Test with different data
		response, err = tool.Invoke(context.Background(), map[string]any{
			"email": "twishabansal@google.com",
			"id":    3,
			"data":  "row4", // This data doesn't match the id
		})
		require.NoError(t, err)
		assert.Equal(t, "null", response, "Response should be null for non-matching data")
	})

	t.Run("test_run_tool_wrong_type_for_integer", func(t *testing.T) {
		client := newClient(t)
		tool := searchRowsTool(t, client)

		_, err := tool.Invoke(context.Background(), map[string]any{
			"email": "twishabansal@google.com",
			"id":    "not-an-integer",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "parameter 'id' expects an integer, but got string")
	})
}

func TestE2E_MapParams(t *testing.T) {
	// Helper to create a new client
	newClient := func(t *testing.T) *core.ToolboxClient {
		client, err := core.NewToolboxClient("http://localhost:5000")
		require.NoError(t, err, "Failed to create ToolboxClient")
		return client
	}

	// Helper to load the process-data tool
	processDataTool := func(t *testing.T, client *core.ToolboxClient) *core.ToolboxTool {
		tool, err := client.LoadTool("process-data", context.Background())
		require.NoError(t, err, "Failed to load tool 'process-data'")
		return tool
	}

	t.Run("test_tool_schema_is_correct", func(t *testing.T) {
		client := newClient(t)
		tool := processDataTool(t, client)
		params := tool.Parameters()

		// Convert slice to map for easy lookup
		paramMap := make(map[string]core.ParameterSchema)
		for _, p := range params {
			paramMap[p.Name] = p
		}

		// Verify 'execution_context' parameter.
		execCtxParam, ok := paramMap["execution_context"]
		require.True(t, ok, "'execution_context' parameter should exist")
		assert.True(t, execCtxParam.Required, "'execution_context' should be required")
		assert.Equal(t, "object", execCtxParam.Type, "'execution_context' type should be object")

		// Verify 'user_scores' parameter.
		userScoresParam, ok := paramMap["user_scores"]
		require.True(t, ok, "'user_scores' parameter should exist")
		assert.True(t, userScoresParam.Required, "'user_scores' should be required")
		assert.Equal(t, "object", userScoresParam.Type, "'user_scores' type should be object")

		// Verify 'feature_flags' parameter.
		featureFlagsParam, ok := paramMap["feature_flags"]
		require.True(t, ok, "'feature_flags' parameter should exist")
		assert.False(t, featureFlagsParam.Required, "'feature_flags' should be optional")
		assert.Equal(t, "object", featureFlagsParam.Type, "'feature_flags' type should be object")
	})

	t.Run("test_run_tool_with_all_map_params", func(t *testing.T) {
		client := newClient(t)
		tool := processDataTool(t, client)

		// Invoke the tool with valid map parameters.
		response, err := tool.Invoke(context.Background(), map[string]any{
			"execution_context": map[string]any{
				"env":  "prod",
				"id":   1234,
				"user": 1234.5,
			},
			"user_scores": map[string]any{
				"user1": 100,
				"user2": 200,
			},
			"feature_flags": map[string]any{
				"new_feature": true,
			},
		})
		require.NoError(t, err)
		respStr, ok := response.(string)
		require.True(t, ok, "Response should be a string")

		assert.Contains(t, respStr, `"execution_context":{"env":"prod","id":1234,"user":1234.5}`)
		assert.Contains(t, respStr, `"user_scores":{"user1":100,"user2":200}`)
		assert.Contains(t, respStr, `"feature_flags":{"new_feature":true}`)
	})

	t.Run("test_run_tool_omitting_optional_map", func(t *testing.T) {
		client := newClient(t)
		tool := processDataTool(t, client)

		// Invoke the tool without the optional 'feature_flags' parameter.
		response, err := tool.Invoke(context.Background(), map[string]any{
			"execution_context": map[string]any{"env": "dev"},
			"user_scores":       map[string]any{"user3": 300},
		})
		require.NoError(t, err)
		respStr, ok := response.(string)
		require.True(t, ok, "Response should be a string")

		assert.Contains(t, respStr, `"execution_context":{"env":"dev"}`)
		assert.Contains(t, respStr, `"user_scores":{"user3":300}`)
		assert.Contains(t, respStr, `"feature_flags":null`)
	})

	t.Run("test_run_tool_with_wrong_map_value_type", func(t *testing.T) {
		client := newClient(t)
		tool := processDataTool(t, client)

		// Attempt to invoke the tool with an incorrect type in a map value.
		_, err := tool.Invoke(context.Background(), map[string]any{
			"execution_context": map[string]any{"env": "staging"},
			"user_scores": map[string]any{
				"user4": "not-an-integer",
			},
		})

		// Assert that an error was returned.
		require.Error(t, err, "Expected an error for wrong map value type")
		assert.Contains(t, err.Error(), "expects an integer, but got string", "Error message should indicate a validation failure")
	})
}

func TestE2E_ContextHandling(t *testing.T) {
	newClient := func(t *testing.T) *core.ToolboxClient {
		client, err := core.NewToolboxClient("http://localhost:5000")
		require.NoError(t, err, "Failed to create ToolboxClient")
		return client
	}

	t.Run("test_load_tool_with_cancelled_context", func(t *testing.T) {
		client := newClient(t)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := client.LoadTool("get-n-rows", ctx)
		require.Error(t, err)
		assert.ErrorIs(t, err, context.Canceled)
	})

	t.Run("test_invoke_tool_with_timed_out_context", func(t *testing.T) {
		client := newClient(t)
		tool, err := client.LoadTool("get-n-rows", context.Background())
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()
		time.Sleep(1 * time.Millisecond)

		_, err = tool.Invoke(ctx, map[string]any{"num_rows": "1"})
		require.Error(t, err)
		assert.ErrorIs(t, err, context.DeadlineExceeded)
	})
}

type failingTokenSource struct{}

func (f *failingTokenSource) Token() (*oauth2.Token, error) {
	return nil, errors.New("token source failed as designed")
}
