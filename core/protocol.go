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

import "github.com/googleapis/mcp-toolbox-sdk-go/core/transport"

// Protocol defines underlying transport protocols.
type Protocol string

const (
	// Toolbox represents the Native Toolbox protocol.
	Toolbox Protocol = "toolbox"

	// MCP Version Constants
	MCPv20250618 Protocol = "2025-06-18"
	MCPv20250326 Protocol = "2025-03-26"
	MCPv20241105 Protocol = "2024-11-05"

	// ProtocolMCP is the default alias pointing to the newest supported version.
	ProtocolMCP = MCPv20250618
)

// GetSupportedMcpVersions returns a list of supported MCP protocol versions.
func GetSupportedMcpVersions() []string {
	return []string{
		string(MCPv20250618),
		string(MCPv20250326),
		string(MCPv20241105),
	}
}

type ManifestSchema = transport.ManifestSchema

// ToolSchema defines a single tool in the manifest.
type ToolSchema = transport.ToolSchema

// ParameterSchema defines the structure and validation logic for tool parameters.
type ParameterSchema = transport.ParameterSchema
