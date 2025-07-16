//go:build tbgenkit

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
	"fmt"
	"strings"
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

// mockTokenSource is a simple implementation of oauth2.TokenSource for testing.
type mockTokenSource struct {
	token *oauth2.Token
}

func (m *mockTokenSource) Token() (*oauth2.Token, error) {
	return m.token, nil
}

func TestToGenkitTool(t *testing.T) {
	// Helper to create a new client for each sub-test, like a function-scoped fixture
	newClient := func(t *testing.T) *core.ToolboxClient {
		client, err := core.NewToolboxClient("http://localhost:5000")
		require.NoError(t, err, "Failed to create ToolboxClient")
		return client
	}

	ctx := context.Background()
	g, _ := genkit.Init(ctx)

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
		genkitTool, err := tbgenkit.ToGenkitTool(nil, g)
		if err == nil {
			t.Fatal("Expected error when tool is nil, got nil")
		}
		if genkitTool != nil {
			t.Error("Expected nil tool when tool is nil, got non-nil")
		}
		expectedErrStr := "Error: ToGenkitTool received a nil core.ToolboxTool pointer."
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
		expectedErrStr := "Error: ToGenkitTool received a nil genkit.Genkit pointer."
		if err.Error() != expectedErrStr {
			t.Errorf("Unexpected error message for nil Genkit: got %q, want %q", err.Error(), expectedErrStr)
		}
	})

	// --- Test Case 6: executeFn input is not map[string]any ---
	t.Run("ExecuteFnNonMapInput", func(t *testing.T) {
		client := newClient(t)
		tool := getNRowsTool(t, client)

		genkitTool, err := tbgenkit.ToGenkitTool(tool, g)
		if err != nil {
			t.Fatalf("ToGenkitTool failed: %v", err)
		}

		// Call Execute with a non-map input (e.g., a string)
		_, execErr := genkitTool.RunRaw(ctx, "this is a string, not a map")
		if execErr == nil {
			t.Fatal("Expected error from executeFn for non-map input, got nil")
		}
		expectedErrStr := "tool input expected map[string]any, got string"
		if execErr.Error() != expectedErrStr {
			t.Errorf("Unexpected error message for non-map input: got %q, want %q", execErr.Error(), expectedErrStr)
		}
	})

	// --- Test Case 7: tool.Invoke() returns error ---
	t.Run("InvokeErrorPropagation", func(t *testing.T) {
		client := newClient(t)
		tool := getNRowsTool(t, client)

		genkitTool, err := tbgenkit.ToGenkitTool(tool, g)
		if err != nil {
			t.Fatalf("ToGenkitTool failed: %v", err)
		}

		_, execErr := genkitTool.RunRaw(ctx, map[string]any{"input": "some_value"})
		if execErr == nil {
			t.Fatal("Expected error from executeFn when Invoke returns error, got nil")
		}

		expectedErrPrefix := fmt.Sprintf("error invoking core tool %s: ", tool.Name())
		if !strings.HasPrefix(execErr.Error(), expectedErrPrefix) {
			t.Errorf("Error message prefix mismatch: got %q, want prefix %q", execErr.Error(), expectedErrPrefix)
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
	g, _ := genkit.Init(ctx)

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

		boundTool, err := tool.ToolFrom(core.WithBindParamString("num_rows", "3"))
		require.NoError(t, err)

		genkitTool, err := tbgenkit.ToGenkitTool(boundTool, g)
		if err != nil {
			t.Fatalf("ToGenkitTool failed: %v", err)
		}

		description := genkitTool.Definition().Description

		expectedDescription := `{
                "type": "object",
                "properties": {}
            }`

		if description != expectedDescription {
			t.Fatal("Expected error from executeFn when Invoke returns error, got nil")
		}

		_, execErr := genkitTool.RunRaw(ctx, map[string]any{})
		if execErr == nil {
			t.Fatal("Expected error from executeFn when Invoke returns error, got nil")
		}

		expectedErrPrefix := fmt.Sprintf("error invoking core tool %s: ", tool.Name())
		if !strings.HasPrefix(execErr.Error(), expectedErrPrefix) {
			t.Errorf("Error message prefix mismatch: got %q, want prefix %q", execErr.Error(), expectedErrPrefix)
		}
	})

	t.Run("WithBoundParams Callable", func(t *testing.T) {
		client := newClient(t)
		tool := getNRowsTool(t, client)
		callable := func() (string, error) {
			return "3", nil
		}

		boundTool, err := tool.ToolFrom(core.WithBindParamStringFunc("num_rows", callable))
		require.NoError(t, err)

		genkitTool, err := tbgenkit.ToGenkitTool(boundTool, g)
		if err != nil {
			t.Fatalf("ToGenkitTool failed: %v", err)
		}

		description := genkitTool.Definition().Description

		expectedDescription := `{
                "type": "object",
                "properties": {}
            }`

		if description != expectedDescription {
			t.Fatal("Expected error from executeFn when Invoke returns error, got nil")
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
	g, _ := genkit.Init(ctx)

	t.Run("test_run_tool_no_auth", func(t *testing.T) {
		client := newClient(t)
		tool, err := client.LoadTool("get-row-by-id-auth", ctx)
		require.NoError(t, err)

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
	g, _ := genkit.Init(ctx)

	// Helper to load the search-rows tool
	searchRowsTool := func(t *testing.T, client *core.ToolboxClient) *core.ToolboxTool {
		tool, err := client.LoadTool("search-rows", ctx)
		require.NoError(t, err, "Failed to load tool 'search-rows'")
		return tool
	}

	t.Run("test_tool_schema_is_correct", func(t *testing.T) {
		client := newClient(t)
		tool := searchRowsTool(t, client)

		genkitTool, err := tbgenkit.ToGenkitTool(tool, g)
		if err != nil {
			t.Fatalf("ToGenkitTool failed: %v", err)
		}

		expectedDescription := `{
                "type": "object",
                "properties": {
                    "email": {
                        "type": "string",
                        "description": "City and state"
                    },
										"data": {
                        "type": "string",
                        "description": "City and state"
                    },
                    "id": {
                        "type": "integer",
                        "description": "Number of days"
                    }
                },
                "required": ["email"]
            }`
		description := genkitTool.Definition().Description

		assert.Equal(t, description, expectedDescription)
	})

	t.Run("test_run_tool_omitting_optionals", func(t *testing.T) {
		client := newClient(t)
		tool := searchRowsTool(t, client)

		genkitTool, err := tbgenkit.ToGenkitTool(tool, g)
		if err != nil {
			t.Fatalf("ToGenkitTool failed: %v", err)
		}

		// Test case 1: Optional params are completely omitted
		response1, err1 := genkitTool.RunRaw(ctx, map[string]any{
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
		response2, err2 := genkitTool.RunRaw(ctx, map[string]any{
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

		genkitTool, err := tbgenkit.ToGenkitTool(tool, g)
		if err != nil {
			t.Fatalf("ToGenkitTool failed: %v", err)
		}
		response, err := genkitTool.RunRaw(ctx, map[string]any{
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

		genkitTool, err := tbgenkit.ToGenkitTool(tool, g)
		if err != nil {
			t.Fatalf("ToGenkitTool failed: %v", err)
		}
		_, err = genkitTool.RunRaw(ctx, map[string]any{
			"data": "row5",
			"id":   5,
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing required parameter 'email'")
	})

}
