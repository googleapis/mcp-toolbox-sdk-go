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

package tbadk

import (
	"context"
	"fmt"

	"github.com/googleapis/mcp-toolbox-sdk-go/core"
)

// The MCP Toolbox for Databases Client.
type ToolboxClient struct {
	*core.ToolboxClient
}

// NewToolboxClient creates and configures a new, immutable client for interacting with a
// Toolbox server.
//
// Inputs:
//   - url: The base URL of the Toolbox server.
//   - opts: A variadic list of core.ClientOption functions to configure the client,
//     such as setting a custom http.Client or default headers.
//
// Returns:
//
//	A configured ToolboxClient and a nil error on success, or an empty client
//	and an error if configuration fails.
func NewToolboxClient(url string, opts ...core.ClientOption) (ToolboxClient, error) {
	finalOpts := append(opts, core.WithClientName("toolbox-adk-go"))
	coreClient, err := core.NewToolboxClient(url, finalOpts...)
	if err != nil {
		return ToolboxClient{}, err
	}
	return ToolboxClient{ToolboxClient: coreClient}, nil
}

// LoadToolset fetches  a collection of tools from the Toolbox server
//
// Inputs:
//   - name: Name of the toolset to be loaded.Set this arg to "" to load the default toolset
//   - ctx: The context to control the lifecycle of the request.
//   - opts: A variadic list of ToolOption functions. These can include WithStrict
//     and options for auth or bound params that may apply to tools in the set.
//
// Returns:
//
//	A slice of configured *ToolboxTool and a nil error on success, or a nil
//	slice and an error if loading or validation fails.
func (tc ToolboxClient) LoadToolset(name string, ctx context.Context, opts ...core.ToolOption) ([]ToolboxTool, error) {
	coreTools, err := tc.ToolboxClient.LoadToolset(name, ctx, opts...)
	if err != nil {
		return nil, err
	}

	tbadkTools := make([]ToolboxTool, 0, len(coreTools))

	for _, coreTool := range coreTools {
		adaptedTool, err := toADKTool(coreTool)
		if err != nil {
			return nil, fmt.Errorf("failed to adapt tool '%s': %w", coreTool.Name(), err)
		}

		tbadkTools = append(tbadkTools, adaptedTool)
	}

	return tbadkTools, nil

}

// LoadTool fetches a single tool from the Toolbox server
//
// Inputs:
//   - name: The specific name of the tool to load.
//   - ctx: The context to control the lifecycle of the request.
//   - opts: A variadic list of ToolOption functions to configure auth tokens
//     or bind parameters for this tool.
//
// Returns:
//
//	A configured *ToolboxTool and a nil error on success, or an empty tool and
//	an error if loading or validation fails.
func (tc ToolboxClient) LoadTool(name string, ctx context.Context, opts ...core.ToolOption) (ToolboxTool, error) {
	coreTool, err := tc.ToolboxClient.LoadTool(name, ctx, opts...)
	if err != nil {
		return ToolboxTool{}, err
	}

	return toADKTool(coreTool)
}
