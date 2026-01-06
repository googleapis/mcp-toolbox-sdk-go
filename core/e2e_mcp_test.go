//go:build e2e

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

package core_test

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/googleapis/mcp-toolbox-sdk-go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

// protocolsToTest defines the matrix of MCP protocols we want to verify.
var protocolsToTest = []struct {
	name     string
	protocol core.Protocol
}{
	{"v20241105", core.MCPv20241105},
	{"v20250326", core.MCPv20250326},
	{"v20250618", core.MCPv20250618},
	{"Latest (alias MCP)", core.MCP},
}

// CapturingTransport wraps http.RoundTripper to capture headers from the latest request.
type CapturingTransport struct {
	base        http.RoundTripper
	lastHeaders http.Header
	mu          sync.Mutex
}

func (c *CapturingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	c.mu.Lock()
	c.lastHeaders = req.Header.Clone()
	c.mu.Unlock()

	// Delegate to the actual network transport
	base := c.base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(req)
}

func (c *CapturingTransport) CapturedHeaders() http.Header {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.lastHeaders
}

// helper factory to create a client with a specific protocol
func getNewMCPToolboxClient(t *testing.T, p core.Protocol) *core.ToolboxClient {
	client, err := core.NewToolboxClient("http://localhost:5000",
		core.WithProtocol(p))
	require.NoError(t, err, "Failed to create MCP ToolboxClient for protocol %s", p)
	return client
}

func TestMCP_Basic(t *testing.T) {
	for _, proto := range protocolsToTest {
		t.Run(proto.name, func(t *testing.T) {
			// Helper to create a new client for each sub-test
			newClient := func(t *testing.T) *core.ToolboxClient {
				return getNewMCPToolboxClient(t, proto.protocol)
			}

			// Helper to load the get-n-rows tool
			getNRowsTool := func(t *testing.T, client *core.ToolboxClient) *core.ToolboxTool {
				tool, err := client.LoadTool("get-n-rows", context.Background())
				require.NoError(t, err, "Failed to load tool 'get-n-rows'")
				require.Equal(t, "get-n-rows", tool.Name())
				return tool
			}

			t.Run("test_mcp_client_headers", func(t *testing.T) {
				// Setup the Transport to capture headers
				capturer := &CapturingTransport{}
				httpClient := &http.Client{
					Transport: capturer,
					Timeout:   30 * time.Second,
				}

				// Inject Transport into Client
				client, err := core.NewToolboxClient("http://localhost:5000",
					core.WithProtocol(proto.protocol),
					core.WithHTTPClient(httpClient),
				)
				require.NoError(t, err)

				// Trigger a request
				_, err = client.LoadTool("get-n-rows", context.Background())
				require.NoError(t, err)

				// Verify Transport Compliance
				headers := capturer.CapturedHeaders()

				switch proto.protocol {
				case core.MCPv20241105:
					// Should NOT have new headers
					assert.Empty(t, headers.Get("MCP-Protocol-Version"), "v20241105 should not send protocol version header")
					assert.Empty(t, headers.Get("Mcp-Session-Id"), "v20241105 should not include Mcp-Session-Id")

				case core.MCPv20250326:
					// v2025-03-26: Must send Accept: application/json
					assert.Equal(t, "application/json", headers.Get("Accept"), "v20250326 must request JSON only")
					assert.NotEmpty(t, headers.Get("Mcp-Session-Id"), "v20250326 should include Mcp-Session-Id")
					assert.Empty(t, headers.Get("MCP-Protocol-Version"), "v20250326 should not send protocol version header")

				case core.MCPv20250618:
					// v2025-06-18: Must send Accept AND Protocol Version
					assert.Equal(t, "application/json", headers.Get("Accept"), "v20250618 must request JSON only")
					assert.Empty(t, headers.Get("Mcp-Session-Id"), "v20250618 should not include Mcp-Session-Id")
					assert.Equal(t, "2025-06-18", headers.Get("MCP-Protocol-Version"), "v20250618 must send correct protocol version header")
				}
			})

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

				_, err := tool.Invoke(context.Background(), map[string]any{})
				require.Error(t, err)
				assert.Contains(t, err.Error(), "missing required parameter 'num_rows'")
			})

			t.Run("test_run_tool_wrong_param_type", func(t *testing.T) {
				client := newClient(t)
				tool := getNRowsTool(t, client)

				_, err := tool.Invoke(context.Background(), map[string]any{"num_rows": 2})
				require.Error(t, err)
				assert.Contains(t, err.Error(), "parameter 'num_rows' expects a string, but got int")
			})
		})
	}
}

func TestMCP_LoadErrors(t *testing.T) {
	for _, proto := range protocolsToTest {
		t.Run(proto.name, func(t *testing.T) {
			newClient := func(t *testing.T) *core.ToolboxClient {
				return getNewMCPToolboxClient(t, proto.protocol)
			}

			t.Run("test_load_non_existent_tool", func(t *testing.T) {
				client := newClient(t)
				_, err := client.LoadTool("non-existent-tool", context.Background())
				require.Error(t, err)
				assert.True(t, err != nil)
			})

			t.Run("test_load_non_existent_toolset", func(t *testing.T) {
				client := newClient(t)
				_, err := client.LoadToolset("non-existent-toolset", context.Background())
				require.Error(t, err)
			})
		})
	}

	t.Run("test_new_client_with_nil_option", func(t *testing.T) {
		_, err := core.NewToolboxClient("http://localhost:5000", nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "received a nil ClientOption")
	})

	t.Run("test_load_tool_with_nil_option", func(t *testing.T) {
		client := getNewMCPToolboxClient(t, protocolsToTest[0].protocol)
		_, err := client.LoadTool("get-n-rows", context.Background(), nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "received a nil ToolOption")
	})
}

func TestMCP_BindParams(t *testing.T) {
	for _, proto := range protocolsToTest {
		t.Run(proto.name, func(t *testing.T) {
			newClient := func(t *testing.T) *core.ToolboxClient {
				return getNewMCPToolboxClient(t, proto.protocol)
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
		})
	}
}

func TestMCP_BindParamErrors(t *testing.T) {
	for _, proto := range protocolsToTest {
		t.Run(proto.name, func(t *testing.T) {
			client := getNewMCPToolboxClient(t, proto.protocol)
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
		})
	}
}

func TestMCP_Auth(t *testing.T) {
	// Helper to create a static token source from a string token
	staticTokenSource := func(token string) oauth2.TokenSource {
		return oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	}

	for _, proto := range protocolsToTest {
		t.Run(proto.name, func(t *testing.T) {
			newClient := func(t *testing.T) *core.ToolboxClient {
				return getNewMCPToolboxClient(t, proto.protocol)
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
				assert.Contains(t, err.Error(), "unauthorized Tool call")
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
				assert.Contains(t, err.Error(), "failed to get token for header my-test-auth_token")
				assert.Contains(t, err.Error(), "token source failed as designed")
			})
		})
	}
}

func TestMCP_OptionalParams(t *testing.T) {
	for _, proto := range protocolsToTest {
		t.Run(proto.name, func(t *testing.T) {
			newClient := func(t *testing.T) *core.ToolboxClient {
				return getNewMCPToolboxClient(t, proto.protocol)
			}
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

				emailParam, ok := paramMap["email"]
				require.True(t, ok)
				assert.True(t, emailParam.Required)
				assert.Equal(t, "string", emailParam.Type)

				dataParam, ok := paramMap["data"]
				require.True(t, ok)
				assert.False(t, dataParam.Required)
				assert.Equal(t, "string", dataParam.Type)

				idParam, ok := paramMap["id"]
				require.True(t, ok)
				assert.False(t, idParam.Required)
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
					"data":  "row4",
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
		})
	}
}

func TestMCP_MapParams(t *testing.T) {
	for _, proto := range protocolsToTest {
		t.Run(proto.name, func(t *testing.T) {
			newClient := func(t *testing.T) *core.ToolboxClient {
				return getNewMCPToolboxClient(t, proto.protocol)
			}
			processDataTool := func(t *testing.T, client *core.ToolboxClient) *core.ToolboxTool {
				tool, err := client.LoadTool("process-data", context.Background())
				require.NoError(t, err, "Failed to load tool 'process-data'")
				return tool
			}

			t.Run("test_tool_schema_is_correct", func(t *testing.T) {
				client := newClient(t)
				tool := processDataTool(t, client)
				params := tool.Parameters()

				paramMap := make(map[string]core.ParameterSchema)
				for _, p := range params {
					paramMap[p.Name] = p
				}

				execCtxParam, ok := paramMap["execution_context"]
				require.True(t, ok)
				assert.True(t, execCtxParam.Required)
				assert.Equal(t, "object", execCtxParam.Type)

				userScoresParam, ok := paramMap["user_scores"]
				require.True(t, ok)
				assert.True(t, userScoresParam.Required)
				assert.Equal(t, "object", userScoresParam.Type)

				featureFlagsParam, ok := paramMap["feature_flags"]
				require.True(t, ok)
				assert.False(t, featureFlagsParam.Required)
				assert.Equal(t, "object", featureFlagsParam.Type)
			})

			t.Run("test_run_tool_with_all_map_params", func(t *testing.T) {
				client := newClient(t)
				tool := processDataTool(t, client)

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
				require.True(t, ok)

				assert.Contains(t, respStr, `"execution_context":{"env":"prod","id":1234,"user":1234.5}`)
				assert.Contains(t, respStr, `"user_scores":{"user1":100,"user2":200}`)
				assert.Contains(t, respStr, `"feature_flags":{"new_feature":true}`)
			})

			t.Run("test_run_tool_omitting_optional_map", func(t *testing.T) {
				client := newClient(t)
				tool := processDataTool(t, client)

				response, err := tool.Invoke(context.Background(), map[string]any{
					"execution_context": map[string]any{"env": "dev"},
					"user_scores":       map[string]any{"user3": 300},
				})
				require.NoError(t, err)
				respStr, ok := response.(string)
				require.True(t, ok)

				assert.Contains(t, respStr, `"execution_context":{"env":"dev"}`)
				assert.Contains(t, respStr, `"user_scores":{"user3":300}`)
				assert.Contains(t, respStr, `"feature_flags":null`)
			})

			t.Run("test_run_tool_with_wrong_map_value_type", func(t *testing.T) {
				client := newClient(t)
				tool := processDataTool(t, client)

				_, err := tool.Invoke(context.Background(), map[string]any{
					"execution_context": map[string]any{"env": "staging"},
					"user_scores": map[string]any{
						"user4": "not-an-integer",
					},
				})

				require.Error(t, err)
				assert.Contains(t, err.Error(), "expects an integer, but got string")
			})
		})
	}
}

func TestMCP_ContextHandling(t *testing.T) {
	for _, proto := range protocolsToTest {
		t.Run(proto.name, func(t *testing.T) {
			newClient := func(t *testing.T) *core.ToolboxClient {
				return getNewMCPToolboxClient(t, proto.protocol)
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
		})
	}
}
