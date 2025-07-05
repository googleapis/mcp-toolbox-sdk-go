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
	"fmt"
	"sync"

	"golang.org/x/oauth2"
	"google.golang.org/api/idtoken"
)

var (
	// Caches the underlying token mechanism, keyed by audience, for efficiency.
	tokenSourceCache = make(map[string]oauth2.TokenSource)
	cacheMutex       = &sync.Mutex{}
	// By assigning the real function to a variable, we can replace it
	// during tests with a mock function.
	newTokenSource = idtoken.NewTokenSource
)

// GetGoogleIDToken fetches a Google ID token for a specific audience.
//
// Inputs:
//
//   - ctx: The context for the request, which can be used for cancellation or deadlines.
//   - audience: The recipient of the token, typically the URL of the secured service
//
// Returns:
//
//	A string in the format "Bearer <token>" on success, or an error if
//	the token could not be fetched.
func GetGoogleIDToken(ctx context.Context, audience string) (string, error) {
	cacheMutex.Lock()
	ts, ok := tokenSourceCache[audience]
	if !ok {
		// If not found in cache, create a new token source.
		var err error
		ts, err = newTokenSource(ctx, audience)
		if err != nil {
			cacheMutex.Unlock() // Unlock before returning the error.
			return "", fmt.Errorf("failed to create new token source: %w", err)
		}
		// Store the new source in the cache.
		tokenSourceCache[audience] = ts
	}
	cacheMutex.Unlock()

	// Use the token source to get a valid token.
	token, err := ts.Token()
	if err != nil {
		return "", fmt.Errorf("failed to retrieve token from source: %w", err)
	}

	// Return the token with the "Bearer " prefix.
	return "Bearer " + token.AccessToken, nil
}
