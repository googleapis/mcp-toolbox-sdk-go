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
	"net/http"

	"golang.org/x/oauth2"
)

// ----- Client Options -----

// ClientOption configures a ToolboxClient at creation time.
type ClientOption func(*ToolboxClient) error

// Constructor for a newToolConfig which initializes the maps for auth token sources and bound parameters
func newToolConfig() *ToolConfig {
	return &ToolConfig{
		AuthTokenSources: make(map[string]oauth2.TokenSource),
		BoundParams:      make(map[string]any),
	}
}

// WithHTTPClient provides a custom http.Client to the ToolboxClient.
func WithHTTPClient(client *http.Client) ClientOption {
	return func(tc *ToolboxClient) error {
		if client == nil {
			return fmt.Errorf("WithHTTPClient: provided http.Client cannot be nil")
		}
		tc.httpClient = client
		return nil
	}
}

// WithClientHeaderString adds a static string value as a client-wide HTTP header.
func WithClientHeaderString(headerName string, value string) ClientOption {
	return func(tc *ToolboxClient) error {
		if _, exists := tc.clientHeaderSources[headerName]; exists {
			return fmt.Errorf("client header '%s' is already set and cannot be overridden", headerName)
		}
		staticToken := &oauth2.Token{AccessToken: value}
		tc.clientHeaderSources[headerName] = oauth2.StaticTokenSource(staticToken)
		return nil
	}
}

// WithClientHeaderTokenSource adds a dynamic client-wide HTTP header from a TokenSource.
func WithClientHeaderTokenSource(headerName string, value oauth2.TokenSource) ClientOption {
	return func(tc *ToolboxClient) error {
		if _, exists := tc.clientHeaderSources[headerName]; exists {
			return fmt.Errorf("client header '%s' is already set and cannot be overridden", headerName)
		}
		if value == nil {
			return fmt.Errorf("WithClientHeaderTokenSource: provided oauth2.TokenSource for header '%s' cannot be nil", headerName)
		}
		tc.clientHeaderSources[headerName] = value
		return nil
	}
}

// WithDefaultToolOptions provides default Options that will be applied to every tool
// loaded by this client.
func WithDefaultToolOptions(opts ...ToolOption) ClientOption {
	return func(tc *ToolboxClient) error {
		if tc.defaultOptionsSet {
			return fmt.Errorf("default tool options have already been set and cannot be modified")
		}
		tc.defaultToolOptions = append(tc.defaultToolOptions, opts...)
		tc.defaultOptionsSet = true
		return nil
	}
}

// ----- Tool Options -----

// ToolConfig holds all configurable aspects for creating or deriving a tool.
type ToolConfig struct {
	AuthTokenSources map[string]oauth2.TokenSource
	BoundParams      map[string]any
	Strict           bool
	strictSet        bool
}

// ToolOption defines a single, universal type for a functional option that configures a tool.
type ToolOption func(*ToolConfig) error

type Integer interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

type Float interface {
	~float32 | ~float64
}

// WithStrict provides an option to enable strict validation for LoadToolset.
func WithStrict(strict bool) ToolOption {
	return func(c *ToolConfig) error {
		if c.strictSet {
			return fmt.Errorf("strict mode is already set and cannot be overridden")
		}
		c.Strict = strict
		c.strictSet = true // Set the flag after successful assignment
		return nil
	}
}

// WithAuthTokenSource provides an authentication token from a standard TokenSource.
func WithAuthTokenSource(authSourceName string, idToken oauth2.TokenSource) ToolOption {
	return func(c *ToolConfig) error {
		if _, exists := c.AuthTokenSources[authSourceName]; exists {
			return fmt.Errorf("authentication source '%s' is already set and cannot be overridden", authSourceName)
		}
		c.AuthTokenSources[authSourceName] = idToken
		return nil
	}
}

// WithAuthTokenString provides a static string authentication token.
func WithAuthTokenString(authSourceName string, idToken string) ToolOption {
	return func(c *ToolConfig) error {
		if _, exists := c.AuthTokenSources[authSourceName]; exists {
			return fmt.Errorf("authentication source '%s' is already set and cannot be overridden", authSourceName)
		}
		staticToken := &oauth2.Token{AccessToken: idToken}
		c.AuthTokenSources[authSourceName] = oauth2.StaticTokenSource(staticToken)
		return nil
	}
}

// Helper function
func createBoundParamToolOption(name string, value any) ToolOption {
	return func(c *ToolConfig) error {
		if _, exists := c.BoundParams[name]; exists {
			return fmt.Errorf("duplicate parameter binding: parameter '%s' is already set", name)
		}
		c.BoundParams[name] = value
		return nil
	}
}

// WithBindParamString binds a static string value to a parameter.
func WithBindParamString(name string, value string) ToolOption {
	return createBoundParamToolOption(name, value)
}

// WithBindParamStringFunc binds a function that returns a string to a parameter.
func WithBindParamStringFunc(name string, fn func() (string, error)) ToolOption {
	return createBoundParamToolOption(name, fn)
}

// WithBindParamInt binds a static integer value to a parameter.
func WithBindParamInt[T Integer](name string, value T) ToolOption {
	return createBoundParamToolOption(name, value)
}

// WithBindParamIntFunc binds a function that returns an integer to a parameter.
func WithBindParamIntFunc[T Integer](name string, fn func() (T, error)) ToolOption {
	return createBoundParamToolOption(name, fn)
}

// WithBindParamFloat binds a static float value to a parameter.
func WithBindParamFloat[T Float](name string, value T) ToolOption {
	return createBoundParamToolOption(name, value)
}

// WithBindParamFloatFunc binds a function that returns a float to a parameter.
func WithBindParamFloatFunc[T Float](name string, fn func() (T, error)) ToolOption {
	return createBoundParamToolOption(name, fn)
}

// WithBindParamBool binds a static boolean value to a parameter.
func WithBindParamBool(name string, value bool) ToolOption {
	return createBoundParamToolOption(name, value)
}

// WithBindParamBoolFunc binds a function that returns a boolean to a parameter.
func WithBindParamBoolFunc(name string, fn func() (bool, error)) ToolOption {
	return createBoundParamToolOption(name, fn)
}

// WithBindParamStringArray binds a static slice of strings to a parameter.
func WithBindParamStringArray(name string, value []string) ToolOption {
	return createBoundParamToolOption(name, value)
}

// WithBindParamStringArrayFunc binds a function that returns a slice of strings.
func WithBindParamStringArrayFunc(name string, fn func() ([]string, error)) ToolOption {
	return createBoundParamToolOption(name, fn)
}

// WithBindParamIntArray binds a static slice of integers to a parameter.
func WithBindParamIntArray[T Integer](name string, value []T) ToolOption {
	return createBoundParamToolOption(name, value)
}

// WithBindParamIntArrayFunc binds a function that returns a slice of integers.
func WithBindParamIntArrayFunc[T Integer](name string, fn func() ([]T, error)) ToolOption {
	return createBoundParamToolOption(name, fn)
}

// WithBindParamFloatArray binds a static slice of floats to a parameter.
func WithBindParamFloatArray[T Float](name string, value []T) ToolOption {
	return createBoundParamToolOption(name, value)
}

// WithBindParamFloatArrayFunc binds a function that returns a slice of floats.
func WithBindParamFloatArrayFunc[T Float](name string, fn func() ([]T, error)) ToolOption {
	return createBoundParamToolOption(name, fn)
}

// WithBindParamBoolArray binds a static slice of booleans to a parameter.
func WithBindParamBoolArray(name string, value []bool) ToolOption {
	return createBoundParamToolOption(name, value)
}

// WithBindParamBoolArrayFunc binds a function that returns a slice of booleans.
func WithBindParamBoolArrayFunc(name string, fn func() ([]bool, error)) ToolOption {
	return createBoundParamToolOption(name, fn)
}
