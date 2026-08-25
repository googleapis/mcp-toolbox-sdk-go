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

import "testing"

func TestGetSupportedMcpVersions(t *testing.T) {
	versions := GetSupportedMcpVersions()

	// Verify we get exactly 5 versions
	if len(versions) != 5 {
		t.Errorf("Expected 5 supported versions, got %d", len(versions))
	}

	// Verify the content matches our constants
	expected := []string{
		string(MCPv20260728),
		string(MCPv20251125),
		string(MCPv20250618),
		string(MCPv20250326),
		string(MCPv20241105),
	}

	for i, v := range versions {
		if v != expected[i] {
			t.Errorf("Index %d: expected version %s, got %s", i, expected[i], v)
		}
	}
}

func TestProtocol_AtLeast(t *testing.T) {
	tests := []struct {
		name           string
		currentVersion Protocol
		minVersion     Protocol
		wantResult     bool
	}{
		{
			name:           "Equal version",
			currentVersion: MCPv20260728,
			minVersion:     MCPv20260728,
			wantResult:     true,
		},
		{
			name:           "Newer current version vs older min version",
			currentVersion: MCPv20260728,
			minVersion:     MCPv20251125,
			wantResult:     true,
		},
		{
			name:           "Older current version vs newer min version",
			currentVersion: MCPv20250326,
			minVersion:     MCPv20260728,
			wantResult:     false,
		},
		{
			name:           "Unrecognized current version returns false",
			currentVersion: Protocol("invalid-version"),
			minVersion:     MCPv20260728,
			wantResult:     false,
		},
		{
			name:           "Unrecognized min version returns false",
			currentVersion: MCPv20260728,
			minVersion:     Protocol("unknown-target"),
			wantResult:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.currentVersion.AtLeast(tt.minVersion)
			if got != tt.wantResult {
				t.Errorf("%q.AtLeast(%q) = %v, want %v", tt.currentVersion, tt.minVersion, got, tt.wantResult)
			}
		})
	}
}

