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

import (
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

// mockTokenSource is a simple implementation of oauth2.TokenSource for testing.
type mockTokenSource struct {
	token *oauth2.Token
}

func (m *mockTokenSource) Token() (*oauth2.Token, error) {
	return m.token, nil
}

// Enforcing the TokenSource type on the mockTokenSource
var _ oauth2.TokenSource = &mockTokenSource{}

// Helper to create a new client for each test, ensuring a clean state.
func newTestClient() *ToolboxClient {
	return &ToolboxClient{
		clientHeaderSources: make(map[string]oauth2.TokenSource),
	}
}

func TestWithHTTPClient(t *testing.T) {
	t.Run("Success case", func(t *testing.T) {
		client := newTestClient()
		customHTTPClient := &http.Client{Timeout: 30 * time.Second}
		opt := WithHTTPClient(customHTTPClient)
		err := opt(client)

		if err != nil {
			t.Errorf("Expected no error, but got: %v", err)
		}
		if client.httpClient != customHTTPClient {
			t.Error("httpClient was not set correctly")
		}
	})

	t.Run("Failure on nil client", func(t *testing.T) {
		client := newTestClient()
		opt := WithHTTPClient(nil)
		err := opt(client)
		if err == nil {
			t.Error("Expected an error for nil http.Client, but got none")
		}
	})
}

func TestWithClientHeaderString(t *testing.T) {
	t.Run("Success case", func(t *testing.T) {
		client := newTestClient()
		headerName := "X-Api-Key"
		headerValue := "static-secret-value"
		opt := WithClientHeaderString(headerName, headerValue)
		err := opt(client)

		if err != nil {
			t.Errorf("Expected no error, but got: %v", err)
		}

		source, ok := client.clientHeaderSources[headerName]
		if !ok {
			t.Fatalf("Header source for '%s' was not set", headerName)
		}
		token, _ := source.Token()
		if token.AccessToken != headerValue {
			t.Errorf("Expected token value '%s', but got '%s'", headerValue, token.AccessToken)
		}
	})

	t.Run("Failure on duplicate header", func(t *testing.T) {
		client := newTestClient()
		headerName := "X-Api-Key"
		opt := WithClientHeaderString(headerName, "value1")
		_ = opt(client) // Apply once

		err := opt(client) // Apply again
		if err == nil {
			t.Error("Expected an error for duplicate header, but got none")
		}
	})
}

func TestWithClientHeaderTokenSource(t *testing.T) {
	mockTokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "dynamic-token"})

	t.Run("Success case", func(t *testing.T) {
		client := newTestClient()
		headerName := "Authorization"
		opt := WithClientHeaderTokenSource(headerName, mockTokenSource)
		err := opt(client)

		if err != nil {
			t.Errorf("Expected no error, but got: %v", err)
		}
		if _, ok := client.clientHeaderSources[headerName]; !ok {
			t.Errorf("TokenSource for header '%s' was not set", headerName)
		}
	})

	t.Run("Failure on nil token source", func(t *testing.T) {
		client := newTestClient()
		opt := WithClientHeaderTokenSource("Authorization", nil)
		err := opt(client)
		if err == nil {
			t.Error("Expected an error for nil TokenSource, but got none")
		}
	})

	t.Run("Failure on duplicate header", func(t *testing.T) {
		client := newTestClient()
		headerName := "Authorization"
		opt := WithClientHeaderTokenSource(headerName, mockTokenSource)
		_ = opt(client) // Apply once

		err := opt(client) // Apply again
		if err == nil {
			t.Error("Expected an error for duplicate header, but got none")
		}
	})
}

func TestWithDefaultToolOptions(t *testing.T) {
	// A dummy ToolOption for testing purposes.
	dummyOpt := func(c *ToolConfig) error { return nil }

	t.Run("Success case", func(t *testing.T) {
		client := newTestClient()
		opt := WithDefaultToolOptions(dummyOpt, dummyOpt)
		err := opt(client)

		if err != nil {
			t.Errorf("Expected no error, but got: %v", err)
		}
		if len(client.defaultToolOptions) != 2 {
			t.Errorf("Expected 2 default options, but got %d", len(client.defaultToolOptions))
		}
		if !client.defaultOptionsSet {
			t.Error("defaultOptionsSet flag was not set to true")
		}
	})

	t.Run("Failure on setting twice", func(t *testing.T) {
		client := newTestClient()
		opt := WithDefaultToolOptions(dummyOpt)
		_ = opt(client) // Apply once

		err := opt(client) // Apply again
		if err == nil {
			t.Error("Expected an error when setting default options twice, but got none")
		}
	})
}

func TestToolOptions(t *testing.T) {
	newTestConfig := func() *ToolConfig {
		return newToolConfig()
	}

	t.Run("WithName", func(t *testing.T) {
		config := newTestConfig()
		opt := WithName("MyTool")

		if err := opt(config); err != nil {
			t.Fatalf("WithName returned an unexpected error: %v", err)
		}
		if config.Name != "MyTool" {
			t.Errorf("WithName() failed: expected Name 'MyTool', got %q", config.Name)
		}
	})

	t.Run("WithStrict", func(t *testing.T) {
		config := newTestConfig()
		opt := WithStrict(true)
		if err := opt(config); err != nil {
			t.Fatalf("WithStrict returned an unexpected error: %v", err)
		}
		if !config.Strict {
			t.Error("WithStrict(true) failed: expected Strict to be true")
		}
	})

	t.Run("WithAuthTokenSource", func(t *testing.T) {
		config := newTestConfig()
		mockSource := &mockTokenSource{token: &oauth2.Token{AccessToken: "test-token"}}

		opt := WithAuthTokenSource("google", mockSource)
		if err := opt(config); err != nil {
			t.Fatalf("WithAuthTokenSource returned an unexpected error: %v", err)
		}

		if config.AuthTokenSources == nil {
			t.Fatal("AuthTokenSources map was not initialized")
		}
		if source, ok := config.AuthTokenSources["google"]; !ok || source != mockSource {
			t.Error("WithAuthTokenSource did not set the token source correctly")
		}
	})

	t.Run("WithAuthTokenString", func(t *testing.T) {
		config := newTestConfig()
		opt := WithAuthTokenString("google", "token-123")

		if err := opt(config); err != nil {
			t.Fatalf("WithAuthTokenString returned an unexpected error: %v", err)
		}

		if config.AuthTokenSources == nil {
			t.Fatal("AuthTokenSources map was not initialized")
		}
		source, ok := config.AuthTokenSources["google"]
		if !ok {
			t.Fatal("WithAuthTokenString did not add the source")
		}
		token, err := source.Token()
		if err != nil || token.AccessToken != "token-123" {
			t.Errorf("Expected static token 'token-123', got %q, err: %v", token.AccessToken, err)
		}
	})

	t.Run("Parameter Binding - Static Values", func(t *testing.T) {
		config := newTestConfig()

		// Setup and apply all static options
		_ = WithBindParamString("username", "john_doe")(config)
		_ = WithBindParamInt("age", 42)(config)
		_ = WithBindParamInt("port", uint16(8080))(config)
		_ = WithBindParamFloat("price", 99.99)(config)
		_ = WithBindParamFloat("tax", float32(0.08))(config)
		_ = WithBindParamBool("isAdmin", true)(config)
		_ = WithBindParamStringArray("tags", []string{"a", "b"})(config)
		_ = WithBindParamIntArray("scores", []int{10, 20})(config)
		_ = WithBindParamFloatArray("coords", []float64{1.1, 2.2})(config)
		_ = WithBindParamBoolArray("flags", []bool{true, false})(config)

		// Assertions
		if config.BoundParams == nil {
			t.Fatal("BoundParams map was not initialized (Negative Test)")
		}
		if val, ok := config.BoundParams["username"].(string); !ok || val != "john_doe" {
			t.Errorf("String binding failed. Got: %v", config.BoundParams["username"])
		}
		if val, ok := config.BoundParams["age"].(int); !ok || val != 42 {
			t.Errorf("Int binding failed. Got: %v", config.BoundParams["age"])
		}
		if val, ok := config.BoundParams["port"].(uint16); !ok || val != 8080 {
			t.Errorf("Generic int (uint16) binding failed. Got: %v", config.BoundParams["port"])
		}
		if val, ok := config.BoundParams["price"].(float64); !ok || val != 99.99 {
			t.Errorf("Float binding failed. Got: %v", config.BoundParams["price"])
		}
		if val, ok := config.BoundParams["tax"].(float32); !ok || val != 0.08 {
			t.Errorf("Generic float (float32) binding failed. Got: %v", config.BoundParams["tax"])
		}
		if val, ok := config.BoundParams["isAdmin"].(bool); !ok || !val {
			t.Errorf("Bool binding failed. Got: %v", config.BoundParams["isAdmin"])
		}
		if val, ok := config.BoundParams["tags"].([]string); !ok || !reflect.DeepEqual(val, []string{"a", "b"}) {
			t.Errorf("StringArray binding failed. Got: %v", config.BoundParams["tags"])
		}
		if val, ok := config.BoundParams["scores"].([]int); !ok || !reflect.DeepEqual(val, []int{10, 20}) {
			t.Errorf("IntArray binding failed. Got: %v", config.BoundParams["scores"])
		}
		if val, ok := config.BoundParams["coords"].([]float64); !ok || !reflect.DeepEqual(val, []float64{1.1, 2.2}) {
			t.Errorf("FloatArray binding failed. Got: %v", config.BoundParams["coords"])
		}
		if val, ok := config.BoundParams["flags"].([]bool); !ok || !reflect.DeepEqual(val, []bool{true, false}) {
			t.Errorf("BoolArray binding failed. Got: %v", config.BoundParams["flags"])
		}

	})

	t.Run("Parameter Binding - Function Values", func(t *testing.T) {
		config := newTestConfig()

		_ = WithBindParamStringFunc("requestID", func() (string, error) { return "req-123", nil })(config)
		_ = WithBindParamIntFunc("userID", func() (int, error) { return 42, nil })(config)
		_ = WithBindParamBoolFunc("isLoggedIn", func() (bool, error) { return true, nil })(config)
		_ = WithBindParamStringArrayFunc("roles", func() ([]string, error) { return []string{"admin", "user"}, nil })(config)

		if fn, ok := config.BoundParams["requestID"].(func() (string, error)); !ok {
			t.Fatal("StringFunc was not stored correctly")
		} else if val, err := fn(); err != nil || val != "req-123" {
			t.Errorf("Executing stored StringFunc failed. Got val=%q, err=%v", val, err)
		}

		if fn, ok := config.BoundParams["userID"].(func() (int, error)); !ok {
			t.Fatal("IntFunc was not stored correctly")
		} else if val, err := fn(); err != nil || val != 42 {
			t.Errorf("Executing stored IntFunc failed. Got val=%d, err=%v", val, err)
		}

		if fn, ok := config.BoundParams["isLoggedIn"].(func() (bool, error)); !ok {
			t.Fatal("BoolFunc was not stored correctly")
		} else if val, err := fn(); err != nil || !val {
			t.Errorf("Executing stored BoolFunc failed. Got val=%v, err=%v", val, err)
		}

		if fn, ok := config.BoundParams["roles"].(func() ([]string, error)); !ok {
			t.Fatal("StringArrayFunc was not stored correctly")
		} else if val, err := fn(); err != nil || !reflect.DeepEqual(val, []string{"admin", "user"}) {
			t.Errorf("Executing stored StringArrayFunc failed. Got val=%v, err=%v", val, err)
		}
	})

	t.Run("Negative Tests - Preventing Overwrites", func(t *testing.T) {
		t.Run("WithName", func(t *testing.T) {
			config := newTestConfig()
			_ = WithName("first-name")(config)
			err := WithName("second-name")(config)
			if err == nil {
				t.Error("Expected an error when setting Name twice, but got nil")
			}
		})

		t.Run("WithStrict", func(t *testing.T) {
			config := newTestConfig()
			_ = WithStrict(true)(config)
			err := WithStrict(false)(config)
			if err == nil {
				t.Error("Expected an error when setting Strict twice, but got nil")
			}
		})

		t.Run("WithAuthTokenSource", func(t *testing.T) {
			config := newTestConfig()
			_ = WithAuthTokenString("google", "token-v1")(config)
			err := WithAuthTokenSource("google", &mockTokenSource{})
			if err == nil {
				t.Error("Expected an error when setting auth source 'google' twice, but got nil")
			}
		})

		t.Run("WithBindParam", func(t *testing.T) {
			config := newTestConfig()
			_ = WithBindParamString("user_id", "user-a")(config)
			err := WithBindParamInt("user_id", 123)(config)
			if err == nil {
				t.Error("Expected an error when binding parameter 'user_id' twice, but got nil")
			}
		})
	})
}

func TestArrayAndArrayFuncOptions(t *testing.T) {
	newTestConfig := func() *ToolConfig {
		return newToolConfig()
	}

	t.Run("Static Array Parameter Binding", func(t *testing.T) {
		config := newTestConfig()

		// Test happy path for different array types
		_ = WithBindParamStringArray("tags", []string{"go", "test"})(config)
		_ = WithBindParamIntArray("ids", []int64{101, 202})(config)

		// Assert string array
		if val, ok := config.BoundParams["tags"].([]string); !ok || !reflect.DeepEqual(val, []string{"go", "test"}) {
			t.Errorf("StringArray binding failed. Got: %v", config.BoundParams["tags"])
		}
		// Assert generic int array
		if val, ok := config.BoundParams["ids"].([]int64); !ok || !reflect.DeepEqual(val, []int64{101, 202}) {
			t.Errorf("IntArray binding failed. Got: %v", config.BoundParams["ids"])
		}
	})

	t.Run("Function Array Parameter Binding", func(t *testing.T) {
		config := newTestConfig()

		stringArrayFunc := func() ([]string, error) { return []string{"a", "b"}, nil }
		_ = WithBindParamStringArrayFunc("labels", stringArrayFunc)(config)

		if fn, ok := config.BoundParams["labels"].(func() ([]string, error)); !ok {
			t.Fatal("StringArrayFunc was not stored correctly")
		} else if val, err := fn(); err != nil || !reflect.DeepEqual(val, []string{"a", "b"}) {
			t.Errorf("Executing stored StringArrayFunc failed. Got val=%v, err=%v", val, err)
		}
	})

	t.Run("Negative Test - Prevents Overwriting Array Parameters", func(t *testing.T) {
		config := newTestConfig()

		err1 := WithBindParamIntArray("scores", []int{99, 88})(config)
		if err1 != nil {
			t.Fatalf("Setting initial array parameter failed: %v", err1)
		}

		err2 := WithBindParamIntArray("scores", []int{77, 66})(config)

		if err2 == nil {
			t.Error("Expected an error when binding an array parameter twice, but got nil")
		} else if !strings.Contains(err2.Error(), "duplicate parameter binding") {
			t.Errorf("Error message for duplicate array parameter is incorrect, got: %v", err2)
		}
	})

	t.Run("Negative Test - Prevents Overwriting Func Array Parameters", func(t *testing.T) {
		config := newTestConfig()

		fn1 := func() ([]int, error) { return []int{1}, nil }
		err1 := WithBindParamIntArrayFunc("data", fn1)(config)
		if err1 != nil {
			t.Fatalf("Setting initial func array parameter failed: %v", err1)
		}

		fn2 := func() ([]int, error) { return []int{2}, nil }
		err2 := WithBindParamIntArrayFunc("data", fn2)(config)

		if err2 == nil {
			t.Error("Expected an error when binding a func array parameter twice, but got nil")
		} else if !strings.Contains(err2.Error(), "duplicate parameter binding") {
			t.Errorf("Error message for duplicate func array parameter is incorrect, got: %v", err2)
		}
	})
}

// TestFunctionParameterBinding covers the less common function-based binding options.
func TestFunctionParameterBinding(t *testing.T) {
	config := newToolConfig()

	// Bind different function types
	_ = WithBindParamFloatFunc("price", func() (float64, error) { return 99.50, nil })(config)
	_ = WithBindParamFloatArrayFunc("vector", func() ([]float32, error) { return []float32{1.1, 2.2}, nil })(config)
	_ = WithBindParamBoolArrayFunc("flags", func() ([]bool, error) { return []bool{true, false, true}, nil })(config)

	// Assert FloatFunc
	if fn, ok := config.BoundParams["price"].(func() (float64, error)); !ok {
		t.Fatal("FloatFunc was not stored correctly")
	} else if val, err := fn(); err != nil || val != 99.50 {
		t.Errorf("Executing stored FloatFunc failed. Got val=%v, err=%v", val, err)
	}

	// Assert FloatArrayFunc
	if fn, ok := config.BoundParams["vector"].(func() ([]float32, error)); !ok {
		t.Fatal("FloatArrayFunc was not stored correctly")
	} else if val, err := fn(); err != nil || !reflect.DeepEqual(val, []float32{1.1, 2.2}) {
		t.Errorf("Executing stored FloatArrayFunc failed. Got val=%v, err=%v", val, err)
	}

	// Assert BoolArrayFunc
	if fn, ok := config.BoundParams["flags"].(func() ([]bool, error)); !ok {
		t.Fatal("BoolArrayFunc was not stored correctly")
	} else if val, err := fn(); err != nil || !reflect.DeepEqual(val, []bool{true, false, true}) {
		t.Errorf("Executing stored BoolArrayFunc failed. Got val=%v, err=%v", val, err)
	}
}

func TestNewToolConfig(t *testing.T) {
	// Call the function to get a new config.
	config := newToolConfig()

	// Check that the returned pointer is not nil.
	if config == nil {
		t.Fatal("NewToolConfig() returned a nil pointer")
	}

	//Check that the maps are initialized (not nil).
	if config.AuthTokenSources == nil {
		t.Error("Expected AuthTokenSources map to be initialized, but it was nil")
	}

	if config.BoundParams == nil {
		t.Error("Expected BoundParams map to be initialized, but it was nil")
	}

	// Check that other fields have their correct zero-values.
	if config.Name != "" {
		t.Errorf("Expected Name to be empty, but got '%s'", config.Name)
	}

	if config.Strict != false {
		t.Errorf("Expected Strict to be false, but got %t", config.Strict)
	}
}
