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
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"
	"google.golang.org/api/option"
)

// mockAuthTokenSource is a mock implementation of the oauth2.TokenSource interface.
// It allows us to control the token and error returned during tests.
type mockAuthTokenSource struct {
	tokenToReturn *oauth2.Token
	errorToReturn error
}

// Token is the method that satisfies the TokenSource interface.
func (m *mockAuthTokenSource) Token() (*oauth2.Token, error) {
	return m.tokenToReturn, m.errorToReturn
}

// setup is a helper to reset the cache and the newTokenSource variable for each test.
func setup(t *testing.T) {
	// Reset the global cache for a clean state.
	cacheMutex.Lock()
	tokenSourceCache = make(map[string]oauth2.TokenSource)
	cacheMutex.Unlock()

	// After the test, restore the original function.
	originalNewTokenSource := newTokenSource
	t.Cleanup(func() {
		newTokenSource = originalNewTokenSource
	})
}

func TestGetGoogleIDToken_Success(t *testing.T) {
	setup(t)
	const mockToken = "mock-id-token-123"
	const audience = "https://test-service.com"

	// Replace the package-level variable with the mock function.
	newTokenSource = func(ctx context.Context, aud string, opts ...option.ClientOption) (oauth2.TokenSource, error) {
		// This mock will return our custom token source.
		return &mockAuthTokenSource{
			tokenToReturn: &oauth2.Token{
				AccessToken: mockToken,
				Expiry:      time.Now().Add(time.Hour),
			},
		}, nil
	}

	token, err := GetGoogleIDToken(context.Background(), audience)

	if err != nil {
		t.Fatalf("Expected no error, but got: %v", err)
	}

	expectedToken := "Bearer " + mockToken
	if token != expectedToken {
		t.Errorf("Expected token '%s', but got '%s'", expectedToken, token)
	}
}

func TestGetGoogleIDToken_Caching(t *testing.T) {
	setup(t)
	callCount := 0

	// Replace the variable with a mock that tracks how many times it's called.
	newTokenSource = func(ctx context.Context, aud string, opts ...option.ClientOption) (oauth2.TokenSource, error) {
		callCount++
		return &mockAuthTokenSource{
			tokenToReturn: &oauth2.Token{AccessToken: "some-token"},
		}, nil
	}

	_, _ = GetGoogleIDToken(context.Background(), "https://some-audience.com")
	_, _ = GetGoogleIDToken(context.Background(), "https://some-audience.com")

	// The mock should only be called once because the result is cached.
	if callCount != 1 {
		t.Errorf("Expected newTokenSource to be called 1 time due to caching, but was called %d times", callCount)
	}
}

func TestGetGoogleIDToken_NewTokenSourceError(t *testing.T) {
	setup(t)
	expectedErr := errors.New("failed to create source")

	// This mock simulates an error during the creation of the token source itself.
	newTokenSource = func(ctx context.Context, aud string, opts ...option.ClientOption) (oauth2.TokenSource, error) {
		return nil, expectedErr
	}

	_, err := GetGoogleIDToken(context.Background(), "https://some-audience.com")

	if err == nil {
		t.Fatal("Expected an error, but got nil")
	}
	if !strings.Contains(err.Error(), expectedErr.Error()) {
		t.Errorf("Expected error message to contain '%s', but got: %v", expectedErr.Error(), err)
	}
}

func TestGetGoogleIDToken_TokenFetchError(t *testing.T) {
	setup(t)
	expectedErr := errors.New("failed to fetch token")

	// This mock successfully creates a source, but the source itself will fail
	// when we try to get a token from it.
	newTokenSource = func(ctx context.Context, aud string, opts ...option.ClientOption) (oauth2.TokenSource, error) {
		return &mockAuthTokenSource{
			errorToReturn: expectedErr,
		}, nil
	}

	_, err := GetGoogleIDToken(context.Background(), "https://some-audience.com")

	if err == nil {
		t.Fatal("Expected an error, but got nil")
	}
	if !strings.Contains(err.Error(), expectedErr.Error()) {
		t.Errorf("Expected error message to contain '%s', but got: %v", expectedErr.Error(), err)
	}
}
