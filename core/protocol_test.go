package core

import (
	"testing"
	"time"
)

// Tests ParameterSchema with type 'int'.
func TestParameterSchemaInteger(t *testing.T) {

	schema := Parameter{
		Name:        "param_name",
		Type:        "integer",
		Description: "integer parameter",
	}

	value := 1

	err := schema.validateType(value)

	if err != nil {
		t.Fatal(err.Error())
	}

}

// Tests ParameterSchema with type 'string'.
func TestParameterSchemaString(t *testing.T) {

	schema := Parameter{
		Name:        "param_name",
		Type:        "string",
		Description: "string parameter",
	}

	value := "abc"

	err := schema.validateType(value)

	if err != nil {
		t.Fatal(err.Error())
	}

}

// Tests ParameterSchema with type 'boolean'.
func TestParameterSchemaBoolean(t *testing.T) {

	schema := Parameter{
		Name:        "param_name",
		Type:        "boolean",
		Description: "boolean parameter",
	}

	value := true

	err := schema.validateType(value)

	if err != nil {
		t.Fatal(err.Error())
	}

}

// Tests ParameterSchema with type 'float'.
func TestParameterSchemaFloat(t *testing.T) {

	schema := Parameter{
		Name:        "param_name",
		Type:        "float",
		Description: "float parameter",
	}

	value := 3.14

	err := schema.validateType(value)

	if err != nil {
		t.Fatal(err.Error())
	}

}

// Tests ParameterSchema with type 'array'.
func TestParameterSchemaStringArray(t *testing.T) {

	itemSchema := Parameter{
		Name:        "item",
		Type:        "string",
		Description: "item of the array",
	}

	paramSchema := Parameter{
		Name:        "param_name",
		Type:        "array",
		Description: "array parameter",
		Items:       &itemSchema,
	}

	value := []string{"abc", "def"}

	err := paramSchema.validateType(value)

	if err != nil {
		t.Fatal(err.Error())
	}

}

// Tests ParameterSchema with type 'array' with no items.
func TestParameterSchemaArrayWithNoItems(t *testing.T) {

	paramSchema := Parameter{
		Name:        "param_name",
		Type:        "array",
		Description: "array parameter",
	}

	value := []string{"abc", "def"}

	err := paramSchema.validateType(value)

	if err == nil {
		t.Fatal("Expected an error, but got nil")
	}

}

// Tests ParameterSchema with an undefined type.
func TestParameterSchemaUndefinedType(t *testing.T) {

	paramSchema := Parameter{
		Name:        "param_name",
		Type:        "time",
		Description: "time parameter",
	}

	value := time.Now()

	err := paramSchema.validateType(value)

	if err == nil {
		t.Fatal("Expected an error, but got nil")
	}

}
