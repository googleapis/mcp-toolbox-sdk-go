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
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2"
)

// This function identifies authentication parameters and authorization tokens that are
// still required after considering the provided token sources.
func identifyAuthRequirements(
	reqAuthnParams map[string][]string,
	reqAuthzTokens []string,
	authTokenSources map[string]oauth2.TokenSource,
) (map[string][]string, []string, []string) {

	unmetAuthnParams := make(map[string][]string)
	usedServices := make(map[string]struct{})

	providedServiceNames := make(map[string]struct{}, len(authTokenSources))
	for name := range authTokenSources {
		providedServiceNames[name] = struct{}{}
	}

	// Find which of the required authn params are covered by available services.
	for param, services := range reqAuthnParams {
		isMet := false
		for _, service := range services {
			if _, ok := providedServiceNames[service]; ok {
				isMet = true
				usedServices[service] = struct{}{}
			}
		}
		if !isMet {
			unmetAuthnParams[param] = services
		}
	}

	// Find which of the required authz tokens are covered by available services.
	var unmetAuthzTokens []string
	isAuthzMet := false
	for _, reqToken := range reqAuthzTokens {
		if _, ok := providedServiceNames[reqToken]; ok {
			isAuthzMet = true
			usedServices[reqToken] = struct{}{}
		}
	}

	// If no match is found, all original tokens are still required.
	if !isAuthzMet {
		unmetAuthzTokens = reqAuthzTokens
	}

	// Convert usedServices set to a slice for the return value
	usedServicesSlice := make([]string, 0, len(usedServices))
	for service := range usedServices {
		usedServicesSlice = append(usedServicesSlice, service)
	}

	return unmetAuthnParams, unmetAuthzTokens, usedServicesSlice
}

// Finds unused keys from in a map given the used keys
func findUnusedKeys(provided, used map[string]struct{}) []string {
	unused := make([]string, 0)
	for k := range provided {
		if _, ok := used[k]; !ok {
			unused = append(unused, k)
		}
	}
	return unused
}

// resolveAndApplyHeaders iterates through a map of token sources, retrieves a
// token from each, and applies it as a header to the given HTTP request.
//
// Inputs:
//   - clientHeaderSources: A map where the key is the HTTP header name and the
//     value is the TokenSource that provides the header's value.
//   - req: The HTTP request to which the headers will be added. This request is
//     modified in place.
//
// Returns:
//
//	An error if retrieving a token from any source fails, otherwise nil.
func resolveAndApplyHeaders(
	clientHeaderSources map[string]oauth2.TokenSource,
	req *http.Request,
) error {
	for name, source := range clientHeaderSources {
		// Retrieve the token
		token, err := source.Token()
		if err != nil {
			return fmt.Errorf("failed to resolve header '%s': %w", name, err)
		}
		// Set the header on the request object.
		req.Header.Set(name, token.AccessToken)
	}
	return nil
}

// loadManifest is an internal helper for fetching manifests from the Toolbox server.
// Inputs:
//   - ctx: The context to control the lifecycle of the HTTP request, including
//     cancellation.
//   - url: The specific URL from which to fetch the manifest.
//   - httpClient: The http.Client used to execute the request.
//   - clientHeaderSources: A map of token sources to be resolved and applied as
//     headers to the request.
//
// Returns:
//
//	A pointer to the successfully parsed ManifestSchema and a nil error, or a
//	nil ManifestSchema and a descriptive error if any part of the process fails.
func loadManifest(ctx context.Context, url string, httpClient *http.Client,
	clientHeaderSources map[string]oauth2.TokenSource) (*ManifestSchema, error) {
	// Create a new GET request with a context for cancellation.
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request to %s: %w", url, err)
	}

	// Add all client-level headers to the request
	if err := resolveAndApplyHeaders(clientHeaderSources, req); err != nil {
		return nil, fmt.Errorf("failed to apply client headers: %w", err)
	}

	//  Execute the HTTP request.
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make HTTP request to %s: %w", url, err)
	}
	defer resp.Body.Close()

	// Check for non-successful status codes and include the response body
	// for better debugging.
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned non-OK status: %d %s, body: %s", resp.StatusCode, resp.Status, string(bodyBytes))
	}

	// Read the response body.
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Unmarshal the JSON body into the ManifestSchema struct.
	var manifest ManifestSchema
	if err = json.Unmarshal(body, &manifest); err != nil {
		return nil, fmt.Errorf("unable to parse manifest correctly: %w", err)
	}
	return &manifest, nil
}
