// Copyright 2024 Google LLC
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
	"testing"
	"time"
)

// Tests ParameterSchema with type 'int'.
func TestParameterSchemaInteger(t *testing.T) {

	schema := ParameterSchema{
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

	schema := ParameterSchema{
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

	schema := ParameterSchema{
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

	schema := ParameterSchema{
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

	itemSchema := ParameterSchema{
		Name:        "item",
		Type:        "string",
		Description: "item of the array",
	}

	paramSchema := ParameterSchema{
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

	paramSchema := ParameterSchema{
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

	paramSchema := ParameterSchema{
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
