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
	"fmt"

	"github.com/googleapis/mcp-toolbox-sdk-go/core"
	"google.golang.org/adk/model"
	"google.golang.org/adk/tool"
	"google.golang.org/genai"
)

type ToolboxTool struct {
	*core.ToolboxTool
	funcDeclaration *genai.FunctionDeclaration
}

var (
	_ tool.Tool = ToolboxTool{}
)

// IsLongRunning indicates whether the tool is a long-running operation
// TBADK will not support long running tools for now
func (tt ToolboxTool) IsLongRunning() bool {
	return false
}

// This function is copied from ADK Go
// if there is already a tool with a function declaration,
// it appends another to it; otherwise, it creates a new genai tool.
func (tt ToolboxTool) ProcessRequest(ctx tool.Context, req *model.LLMRequest) error {
	if req.Tools == nil {
		req.Tools = make(map[string]any)
	}

	name := tt.Name()

	if _, ok := req.Tools[name]; ok {
		return fmt.Errorf("duplicate tool: %q", name)
	}
	req.Tools[name] = tt

	if req.Config == nil {
		req.Config = &genai.GenerateContentConfig{}
	}
	if decl := tt.Declaration(); decl == nil {
		return nil
	}
	// Find an existing genai.Tool with FunctionDeclarations
	var funcTool *genai.Tool
	for _, tool := range req.Config.Tools {
		if tool != nil && tool.FunctionDeclarations != nil {
			funcTool = tool
			break
		}
	}
	if funcTool == nil {
		req.Config.Tools = append(req.Config.Tools, &genai.Tool{
			FunctionDeclarations: []*genai.FunctionDeclaration{tt.Declaration()},
		})
	} else {
		funcTool.FunctionDeclarations = append(funcTool.FunctionDeclarations, tt.Declaration())
	}
	return nil
}

// Declaration returns the tool's function declaration.
func (tt ToolboxTool) Declaration() *genai.FunctionDeclaration {
	return tt.funcDeclaration
}

// Run executes the tool with the given input.
//
// Inputs:
//   - ctx: The tool context to control the lifecycle of the API request.
//   - args: A map of parameter names to values provided by the user for this
//     specific invocation.
//
// Returns:
//
//	The result from the API call, in the form of a map[string]any with the result
//	in the 'output' field.
func (tt ToolboxTool) Run(ctx tool.Context, args any) (result map[string]any, err error) {
	// Perform a safe type assertion for the input.
	inputMap, ok := args.(map[string]any)
	if !ok {
		// If the input is not a map, return an error indicating the type mismatch.
		return nil, fmt.Errorf("tool input expected map[string]any, got %T", args)
	}
	// Invoke the underlying tool with the provided context and input.
	toolresult, err := tt.Invoke(ctx, inputMap)
	if err != nil {
		// Propagate any errors that occurred during the custom tool's invocation.
		return nil, fmt.Errorf("error invoking the tool %s: %w", tt.Name(), err)
	}

	// Convert the result from the custom tool's invocation to a string.
	strResult := fmt.Sprintf("%v", toolresult)
	return map[string]any{
		"output": strResult,
	}, nil
}

// ToolFrom creates a new, more specialized tool from an existing one by applying
// additional options. This is useful for creating variations of a tool with
// different bound parameters / auth tokens without modifying the original and
// all provided options must be applicable.
//
// Inputs:
//   - opts: A variadic list of ToolOption functions to further configure the
//     new tool, such as binding more parameters.
//
// Returns:
//
//	A new, specialized ToolboxTool and a nil error, or an empty tool and an
//	error if the new options are invalid or conflict with existing settings.
func (tt ToolboxTool) ToolFrom(opts ...core.ToolOption) (ToolboxTool, error) {
	coreTool, err := tt.ToolboxTool.ToolFrom(opts...)
	if err != nil {
		return ToolboxTool{}, err
	}

	return toADKTool(coreTool)

}
