package core

import (
	"fmt"
	"reflect"
)

// Schema for a tool parameter.
type Parameter struct {
	Name        string     `json:"name"`
	Type        string     `json:"type"`
	Description string     `json:"description"`
	AuthSources []string   `json:"authSources,omitempty"`
	Items       *Parameter `json:"items,omitempty"`
}

// validateType is a helper for manual type checking.
func (p *Parameter) validateType(value any) error {
	if value == nil {
		return fmt.Errorf("parameter '%s' received a nil value", p.Name)
	}

	switch p.Type {
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("parameter '%s' expects a string, but got %T", p.Name, value)
		}
	case "integer":
		_, ok := value.(int)
		if !ok {
			return fmt.Errorf("parameter '%s' expects an integer, but got %T", p.Name, value)
		}
	case "float":
		if _, ok := value.(float64); !ok {
			return fmt.Errorf("parameter '%s' expects a number, but got %T", p.Name, value)
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("parameter '%s' expects a boolean, but got %T", p.Name, value)
		}
	case "array":
		v := reflect.ValueOf(value)
		if v.Kind() != reflect.Slice {
			return fmt.Errorf("parameter '%s' expects an array/slice, but got %T", p.Name, value)
		}
		if p.Items == nil {
			return fmt.Errorf("parameter '%s' is an array but is missing item type definition", p.Name)
		}
		for i := range v.Len() {
			item := v.Index(i).Interface()

			if err := p.Items.validateType(item); err != nil {
				return fmt.Errorf("error in array '%s' at index %d: %w", p.Name, i, err)
			}
		}
	default:
		return fmt.Errorf("unknown type '%s' in schema for parameter '%s'", p.Type, p.Name)
	}
	return nil
}

// Schema for a tool.
type ToolSchema struct {
	Description  string      `json:"description"`
	Parameters   []Parameter `json:"parameters"`
	AuthRequired []string    `json:"authRequired,omitempty"`
}

// Schema for the Toolbox manifest.
type ManifestSchema struct {
	ServerVersion string                `json:"serverVersion"`
	Tools         map[string]ToolSchema `json:"tools"`
}
