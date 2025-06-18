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

import "golang.org/x/oauth2"

// ClientOption configures a ToolboxClient at creation time.
type ClientOption func(*ToolboxClient)

// ToolConfig holds all configurable aspects for creating or deriving a tool.
type ToolConfig struct {
	AuthTokenSources map[string]oauth2.TokenSource
	BoundParams      map[string]any
	Name             string
	Strict           bool
}

// ToolOption defines a single, universal type for a functional option that configures a tool.
type ToolOption func(*ToolConfig)
