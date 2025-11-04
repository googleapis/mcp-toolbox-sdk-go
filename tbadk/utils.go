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
// limitations under the License.s

package tbadk

import (
	"encoding/json"
	"fmt"

	"github.com/googleapis/mcp-toolbox-sdk-go/core"
	"google.golang.org/genai"
)

func toADKTool(t *core.ToolboxTool) (ToolboxTool, error) {
	if t == nil {
		return ToolboxTool{}, fmt.Errorf("nil tool recieved")
	}

	paramsJSON, err := t.InputSchema()
	if err != nil {
		return ToolboxTool{}, fmt.Errorf("could not generate input schema from core tool: %w", err)
	}

	fullFunctionDef := struct {
		Name        string          `json:"name"`
		Description string          `json:"description"`
		Parameters  json.RawMessage `json:"parameters"`
	}{
		Name:        t.Name(),
		Description: t.Description(),
		Parameters:  paramsJSON,
	}

	finalJSON, err := json.Marshal(fullFunctionDef)
	if err != nil {
		return ToolboxTool{}, fmt.Errorf("failed to marshal final function declaration: %w", err)
	}

	var funcDecl genai.FunctionDeclaration
	if err := json.Unmarshal(finalJSON, &funcDecl); err != nil {
		return ToolboxTool{}, fmt.Errorf("failed to unmarshal to FunctionDeclaration: %w", err)
	}

	adaptedTool := ToolboxTool{
		ToolboxTool:     t,
		funcDeclaration: &funcDecl,
	}

	return adaptedTool, nil
}
