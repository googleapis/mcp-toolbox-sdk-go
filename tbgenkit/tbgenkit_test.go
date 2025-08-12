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

package tbgenkit_test

import (
	"context"
	"log"
	"os"
	"reflect"
	"testing"

	"github.com/firebase/genkit/go/genkit"
	"github.com/googleapis/mcp-toolbox-sdk-go/core"
	"github.com/googleapis/mcp-toolbox-sdk-go/tbgenkit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

// Global variables to hold session-scoped "fixtures"
var (
	projectID      string = getEnvVar("GOOGLE_CLOUD_PROJECT")
	toolboxVersion string = getEnvVar("TOOLBOX_VERSION")
	authToken1     string
	authToken2     string
)

func TestMain(m *testing.M) {
	ctx := context.Background()
	log.Println("Starting E2E test setup...")

	// Get secrets and auth tokens
	log.Println("Fetching secrets and auth tokens...")
	toolsManifestContent := accessSecretVersion(ctx, projectID, "sdk_testing_tools", "34")
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

func TestToGenkitTool(t *testing.T) {
	// Helper to create a new client for each sub-test, like a function-scoped fixture
	newClient := func(t *testing.T) *core.ToolboxClient {
		client, err := core.NewToolboxClient("http://localhost:5000")
		require.NoError(t, err, "Failed to create ToolboxClient")
		return client
	}

	ctx := context.Background()

	newGenkit := func() *genkit.Genkit {
		g, _ := genkit.Init(ctx)
		return g
	}

	// Helper to load the get-n-rows tool
	getNRowsTool := func(t *testing.T, client *core.ToolboxClient) *core.ToolboxTool {
		tool, err := client.LoadTool("get-n-rows", ctx)
		require.NoError(t, err, "Failed to load tool 'get-n-rows'")
		require.Equal(t, "get-n-rows", tool.Name())
		return tool
	}

	t.Run("SuccessfulConversionAndExecution", func(t *testing.T) {
		client := newClient(t)
		tool := getNRowsTool(t, client)
		g := newGenkit()

		genkitTool, err := tbgenkit.ToGenkitTool(tool, g)
		if err != nil {
			t.Fatalf("ToGenkitTool returned unexpected error: %v", err)
		}
		if genkitTool == nil {
			t.Fatal("ToGenkitTool returned nil tool, expected as valid tool")
		}

		// Verify the properties of the returned genkitTool
		if genkitTool.Name() != tool.Name() {
			t.Errorf("Returned genkit tool name mismatch: got %q, want %q", genkitTool.Name(), tool.Name())
		}

		// Execute the wrapped function and verify its output
		inputForExecute := map[string]any{"num_rows": "2"}
		actualResult, execErr := genkitTool.RunRaw(ctx, inputForExecute)
		if execErr != nil {
			t.Errorf("ExecuteFn returned unexpected error: %v", execErr)
		}
		respStr, ok := actualResult.(string)
		require.True(t, ok, "Response should be a string")
		assert.Contains(t, respStr, "row1")
		assert.Contains(t, respStr, "row2")
		assert.NotContains(t, respStr, "row3")
	})

	// --- Test Case 2: tool is nil ---
	t.Run("NilTool", func(t *testing.T) {
		g := newGenkit()
		genkitTool, err := tbgenkit.ToGenkitTool(nil, g)
		if err == nil {
			t.Fatal("Expected error when tool is nil, got nil")
		}
		if genkitTool != nil {
			t.Error("Expected nil tool when tool is nil, got non-nil")
		}
		expectedErrStr := "error: ToGenkitTool received a nil core.ToolboxTool pointer"
		if err.Error() != expectedErrStr {
			t.Errorf("Unexpected error message for nil tool: got %q, want %q", err.Error(), expectedErrStr)
		}
	})

	// --- Test Case 3: g is nil ---
	t.Run("NilGenkit", func(t *testing.T) {
		client := newClient(t)
		tool := getNRowsTool(t, client)
		genkitTool, err := tbgenkit.ToGenkitTool(tool, nil)
		if err == nil {
			t.Fatal("Expected error when Genkit instance is nil, got nil")
		}
		if genkitTool != nil {
			t.Error("Expected nil tool when Genkit instance is nil, got non-nil")
		}
		expectedErrStr := "error: ToGenkitTool received a nil genkit.Genkit pointer"
		if err.Error() != expectedErrStr {
			t.Errorf("Unexpected error message for nil Genkit: got %q, want %q", err.Error(), expectedErrStr)
		}
	})

}

func TestToGenkitTool_BoundParams(t *testing.T) {
	// Helper to create a new client for each sub-test, like a function-scoped fixture
	newClient := func(t *testing.T) *core.ToolboxClient {
		client, err := core.NewToolboxClient("http://localhost:5000")
		require.NoError(t, err, "Failed to create ToolboxClient")
		return client
	}
	ctx := context.Background()
	newGenkit := func() *genkit.Genkit {
		g, _ := genkit.Init(ctx)
		return g
	}

	// Helper to load the get-n-rows tool
	getNRowsTool := func(t *testing.T, client *core.ToolboxClient) *core.ToolboxTool {
		tool, err := client.LoadTool("get-n-rows", ctx)
		require.NoError(t, err, "Failed to load tool 'get-n-rows'")
		require.Equal(t, "get-n-rows", tool.Name())
		return tool
	}

	t.Run("WithBoundParams", func(t *testing.T) {
		client := newClient(t)
		tool := getNRowsTool(t, client)
		g := newGenkit()

		boundTool, err := tool.ToolFrom(core.WithBindParamString("num_rows", "3"))
		require.NoError(t, err)

		genkitTool, err := tbgenkit.ToGenkitTool(boundTool, g)
		if err != nil {
			t.Fatalf("ToGenkitTool failed: %v", err)
		}

		schema := genkitTool.Definition().InputSchema

		expectedSchema := map[string]any{
			"type":       "object",
			"properties": struct{}{},
		}

		if reflect.DeepEqual(schema, expectedSchema) {
			t.Fatal("Input schema does not match")
		}

		result, err := genkitTool.RunRaw(ctx, map[string]any{})

		require.NoError(t, err)

		respStr, ok := result.(string)
		require.True(t, ok)
		assert.Contains(t, respStr, "row1")
		assert.Contains(t, respStr, "row2")
		assert.Contains(t, respStr, "row3")
		assert.NotContains(t, respStr, "row4")
	})

	t.Run("WithBoundParams Callable", func(t *testing.T) {
		client := newClient(t)
		tool := getNRowsTool(t, client)
		callable := func() (string, error) {
			return "3", nil
		}
		g := newGenkit()

		boundTool, err := tool.ToolFrom(core.WithBindParamStringFunc("num_rows", callable))
		require.NoError(t, err)

		genkitTool, err := tbgenkit.ToGenkitTool(boundTool, g)
		if err != nil {
			t.Fatalf("ToGenkitTool failed: %v", err)
		}

		schema := genkitTool.Definition().InputSchema

		expectedSchema := map[string]any{
			"type":       "object",
			"properties": struct{}{},
		}

		if reflect.DeepEqual(schema, expectedSchema) {
			t.Fatal("Input schema does not match")
		}

		result, err := genkitTool.RunRaw(ctx, map[string]any{})

		require.NoError(t, err)

		respStr, ok := result.(string)
		require.True(t, ok)
		assert.Contains(t, respStr, "row1")
		assert.Contains(t, respStr, "row2")
		assert.Contains(t, respStr, "row3")
		assert.NotContains(t, respStr, "row4")
	})
}

func TestToGenkitTool_Auth(t *testing.T) {
	// Helper to create a new client for each sub-test, like a function-scoped fixture
	newClient := func(t *testing.T) *core.ToolboxClient {
		client, err := core.NewToolboxClient("http://localhost:5000")
		require.NoError(t, err, "Failed to create ToolboxClient")
		return client
	}

	// Helper to create a static token source from a string token
	staticTokenSource := func(token string) oauth2.TokenSource {
		return oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	}
	ctx := context.Background()
	newGenkit := func() *genkit.Genkit {
		g, _ := genkit.Init(ctx)
		return g
	}

	t.Run("test_run_tool_no_auth", func(t *testing.T) {
		client := newClient(t)
		tool, err := client.LoadTool("get-row-by-id-auth", ctx)
		require.NoError(t, err)
		g := newGenkit()

		genkitTool, err := tbgenkit.ToGenkitTool(tool, g)
		if err != nil {
			t.Fatalf("ToGenkitTool failed: %v", err)
		}

		_, err = genkitTool.RunRaw(ctx, map[string]any{"id": "2"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "permission error: auth service 'my-test-auth' is required")
	})

	t.Run("test_run_tool_wrong_auth", func(t *testing.T) {
		client := newClient(t)
		tool, err := client.LoadTool("get-row-by-id-auth", ctx)
		require.NoError(t, err)
		g := newGenkit()

		authedTool, err := tool.ToolFrom(
			core.WithAuthTokenSource("my-test-auth", staticTokenSource(authToken2)),
		)
		require.NoError(t, err)

		genkitTool, err := tbgenkit.ToGenkitTool(authedTool, g)
		if err != nil {
			t.Fatalf("ToGenkitTool failed: %v", err)
		}

		_, err = genkitTool.RunRaw(ctx, map[string]any{"id": "2"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "tool invocation not authorized")
	})

	t.Run("test_run_tool_auth", func(t *testing.T) {
		client := newClient(t)
		tool, err := client.LoadTool("get-row-by-id-auth", ctx,
			core.WithAuthTokenSource("my-test-auth", staticTokenSource(authToken1)),
		)
		require.NoError(t, err)
		g := newGenkit()

		genkitTool, err := tbgenkit.ToGenkitTool(tool, g)
		if err != nil {
			t.Fatalf("ToGenkitTool failed: %v", err)
		}

		response, err := genkitTool.RunRaw(ctx, map[string]any{"id": "2"})
		require.NoError(t, err)

		respStr, ok := response.(string)
		require.True(t, ok)
		assert.Contains(t, respStr, "row2")
	})

	t.Run("test_run_tool_param_auth_no_auth", func(t *testing.T) {
		client := newClient(t)
		tool, err := client.LoadTool("get-row-by-email-auth", ctx)
		require.NoError(t, err)
		g := newGenkit()

		genkitTool, err := tbgenkit.ToGenkitTool(tool, g)
		if err != nil {
			t.Fatalf("ToGenkitTool failed: %v", err)
		}

		_, err = genkitTool.RunRaw(ctx, map[string]any{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "permission error: auth service 'my-test-auth' is required")
	})

	t.Run("test_run_tool_param_auth", func(t *testing.T) {
		client := newClient(t)
		tool, err := client.LoadTool("get-row-by-email-auth", ctx,
			core.WithAuthTokenSource("my-test-auth", staticTokenSource(authToken1)),
		)
		require.NoError(t, err)
		g := newGenkit()

		genkitTool, err := tbgenkit.ToGenkitTool(tool, g)
		if err != nil {
			t.Fatalf("ToGenkitTool failed: %v", err)
		}

		response, err := genkitTool.RunRaw(ctx, map[string]any{})
		require.NoError(t, err)

		respStr, ok := response.(string)
		require.True(t, ok)
		assert.Contains(t, respStr, "row4")
		assert.Contains(t, respStr, "row5")
		assert.Contains(t, respStr, "row6")
	})
}

func TestToGenkitTool_OptionalParams(t *testing.T) {
	// Helper to create a new client for each sub-test, like a function-scoped fixture
	newClient := func(t *testing.T) *core.ToolboxClient {
		client, err := core.NewToolboxClient("http://localhost:5000")
		require.NoError(t, err, "Failed to create ToolboxClient")
		return client
	}
	ctx := context.Background()
	newGenkit := func() *genkit.Genkit {
		g, _ := genkit.Init(ctx)
		return g
	}

	// Helper to load the search-rows tool
	searchRowsTool := func(t *testing.T, client *core.ToolboxClient) *core.ToolboxTool {
		tool, err := client.LoadTool("search-rows", ctx)
		require.NoError(t, err, "Failed to load tool 'search-rows'")
		return tool
	}

	t.Run("test_tool_schema_is_correct", func(t *testing.T) {
		client := newClient(t)
		tool := searchRowsTool(t, client)
		g := newGenkit()

		genkitTool, err := tbgenkit.ToGenkitTool(tool, g)
		if err != nil {
			t.Fatalf("ToGenkitTool failed: %v", err)
		}

		expectedSchema := map[string]any{
			"type": "object",
			"properties": map[string]any{
				"data": map[string]any{
					"description": "The row to narrow down the search.",
					"type":        "string",
				},
				"email": map[string]any{
					"description": "The email to search for.",
					"type":        "string",
				},
				"id": map[string]any{
					"description": "The id to narrow down the search.",
					"type":        "integer",
				}},
			"required": []any{"email"},
		}

		schema := genkitTool.Definition().InputSchema

		assert.Equal(t, schema, expectedSchema)
	})

	t.Run("test_run_tool_missing_required_param", func(t *testing.T) {
		client := newClient(t)
		tool := searchRowsTool(t, client)
		g := newGenkit()

		genkitTool, err := tbgenkit.ToGenkitTool(tool, g)
		if err != nil {
			t.Fatalf("ToGenkitTool failed: %v", err)
		}
		_, err = genkitTool.RunRaw(ctx, map[string]any{
			"data": "row5",
			"id":   5,
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "email is required")
	})

}

func TestToGenkitTool_MapParams(t *testing.T) {
	// Helper to create a new client for each sub-test, like a function-scoped fixture
	newClient := func(t *testing.T) *core.ToolboxClient {
		client, err := core.NewToolboxClient("http://localhost:5000")
		require.NoError(t, err, "Failed to create ToolboxClient")
		return client
	}
	ctx := context.Background()
	newGenkit := func() *genkit.Genkit {
		g, _ := genkit.Init(ctx)
		return g
	}

	// Helper to load the process-data tool
	processDataTool := func(t *testing.T, client *core.ToolboxClient) *core.ToolboxTool {
		tool, err := client.LoadTool("process-data", ctx)
		require.NoError(t, err, "Failed to load tool 'process-data'")
		return tool
	}

	t.Run("test_tool_schema_is_correct", func(t *testing.T) {
		client := newClient(t)
		tool := processDataTool(t, client)
		g := newGenkit()

		genkitTool, err := tbgenkit.ToGenkitTool(tool, g)
		if err != nil {
			t.Fatalf("ToGenkitTool failed: %v", err)
		}

		expectedSchema := map[string]any{
			"type": "object",
			"properties": map[string]any{
				"execution_context": map[string]any{
					"description":          "A map containing context about the execution environment.",
					"type":                 "object",
					"additionalProperties": true,
				},
				"user_scores": map[string]any{
					"description": "A map of user IDs to their integer scores.",
					"type":        "object",
					"additionalProperties": map[string]any{
						"type": "integer",
					},
				},
				"feature_flags": map[string]any{
					"description": "An optional map of feature flags.",
					"type":        "object",
					"additionalProperties": map[string]any{
						"type": "boolean",
					},
				}},
			"required": []any{"execution_context", "user_scores"},
		}

		schema := genkitTool.Definition().InputSchema

		assert.Equal(t, schema, expectedSchema)
	})

	t.Run("test_run_tool_with_all_map_params", func(t *testing.T) {
		client := newClient(t)
		tool := processDataTool(t, client)
		g := newGenkit()

		genkitTool, err := tbgenkit.ToGenkitTool(tool, g)
		if err != nil {
			t.Fatalf("ToGenkitTool failed: %v", err)
		}

		// Invoke the tool with valid map parameters.
		response, err := genkitTool.RunRaw(context.Background(), map[string]any{
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

}
