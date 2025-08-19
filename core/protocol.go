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

import (
	"fmt"
	"reflect"
)

// Schema for a tool parameter.
type ParameterSchema struct {
	Name                 string           `json:"name"`
	Type                 string           `json:"type"`
	Required             bool             `json:"required,omitempty"`
	Description          string           `json:"description"`
	AuthSources          []string         `json:"authSources,omitempty"`
	Items                *ParameterSchema `json:"items,omitempty"`
	AdditionalProperties any              `json:"additionalProperties,omitempty"`
}

// validateType is a helper for manual type checking.
func (p *ParameterSchema) validateType(value any) error {
	if value == nil {
		if p.Required {
			return fmt.Errorf("parameter '%s' is required but received a nil value", p.Name)
		}
		return nil
	}

	switch p.Type {
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("parameter '%s' expects a string, but got %T", p.Name, value)
		}
	case "integer":
		switch value.(type) {
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		default:
			return fmt.Errorf("parameter '%s' expects an integer, but got %T", p.Name, value)
		}
	case "float":
		switch value.(type) {
		case float32, float64:
		default:
			return fmt.Errorf("parameter '%s' expects an float, but got %T", p.Name, value)
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("parameter '%s' expects a boolean, but got %T", p.Name, value)
		}
	case "array":
		v := reflect.ValueOf(value)
		if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
			return fmt.Errorf("parameter '%s' expects an array/slice, but got %T", p.Name, value)
		}
		for i := range v.Len() {
			item := v.Index(i).Interface()

			if err := p.Items.validateType(item); err != nil {
				return fmt.Errorf("error in array '%s' at index %d: %w", p.Name, i, err)
			}
		}
	case "object":
		// First, check that the value is a map with string keys.
		valMap, ok := value.(map[string]any)
		if !ok {
			return fmt.Errorf("parameter '%s' expects a map, but got %T", p.Name, value)
		}

		switch ap := p.AdditionalProperties.(type) {
		// No validation required, allows any type
		case bool:

		// Validate type for each value in map
		case *ParameterSchema:
			// Raise error if the input is a nested map / array
			if ap.Type == "object" || ap.Type == "array" {
				return fmt.Errorf("invalid schema for object '%s': values cannot be of type '%s'", p.Name, ap.Type)
			}
			for key, val := range valMap {
				if err := ap.validateType(val); err != nil {
					return fmt.Errorf("error in object '%s' for key '%s': %w", p.Name, key, err)
				}
			}

		default:
			// This is a schema / manifest error.
			return fmt.Errorf(
				"invalid schema for parameter '%s': AdditionalProperties must be a boolean or a map[string]any, but got %T",
				p.Name,
				ap,
			)
		}
	default:
		return fmt.Errorf("unknown type '%s' in schema for parameter '%s'", p.Type, p.Name)
	}
	return nil
}

// ValidateDefinition checks if the schema itself is well-formed.
func (p *ParameterSchema) ValidateDefinition() error {
	if p.Type == "" {
		return fmt.Errorf("schema validation failed for '%s': type is missing", p.Name)
	}

	switch p.Type {
	case "array":
		if p.Items == nil {
			return fmt.Errorf("parameter '%s' is an array but is missing item type definition", p.Name)
		}
		// Recursively validate the nested schema's definition.
		if err := p.Items.ValidateDefinition(); err != nil {
			return err
		}

	case "object":
		switch ap := p.AdditionalProperties.(type) {
		case bool:
			// Valid scenario
		case *ParameterSchema:
			if err := ap.ValidateDefinition(); err != nil {
				return err
			}
		default:
			// Any other type is an invalid schema definition.
			return fmt.Errorf(
				"invalid schema for parameter '%s': AdditionalProperties must be a boolean or a schema, but got %T",
				p.Name,
				ap,
			)
		}

	case "string", "integer", "float", "boolean":
		// No type-specific rules for these.
		break

	default:
		return fmt.Errorf("unknown schema type '%s' for parameter '%s'", p.Type, p.Name)
	}

	return nil
}

// Schema for a tool.
type ToolSchema struct {
	Description  string            `json:"description"`
	Parameters   []ParameterSchema `json:"parameters"`
	AuthRequired []string          `json:"authRequired,omitempty"`
}

// Schema for the Toolbox manifest.
type ManifestSchema struct {
	ServerVersion string                `json:"serverVersion"`
	Tools         map[string]ToolSchema `json:"tools"`
}
