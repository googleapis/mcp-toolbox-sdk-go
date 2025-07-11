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
//
// Inputs:
//   - reqAuthnParams: A mapping of parameter names to list of required
//     authentication services for those parameters.
//   - reqAuthzTokens: A slice of strings representing all authorization
//     tokens that are required to invoke the current tool.
//   - authTokenSources: An iterable of authentication/authorization service
//     names for which token getters are available.
//
// Returns:
//   - requiredAuthnParams: A map representing the subset of required authentication
//     parameters that are not covered by the
//     provided authTokenSources.
//   - requiredAuthzTokens: A slice of authorization tokens that were not satisfied
//     by any of the provided authTokenSources.
//   - usedServices: A slice of service names from authTokenSources that were used
//     to satisfy one or more authentication or authorization requirements.
func identifyAuthRequirements(
	reqAuthnParams map[string][]string,
	reqAuthzTokens []string,
	authTokenSources map[string]oauth2.TokenSource,
) (map[string][]string, []string, []string) {

	// This map will be populated with authentication parameters that are NOT met.
	requiredAuthnParams := make(map[string][]string)
	// This map is used as a "set" to track every available service that was
	// used to meet ANY requirement.
	usedServices := make(map[string]struct{})

	// Find which of the required authn params are covered by available services.
	for param, services := range reqAuthnParams {

		// First, just check IF the requirement can be met by any available service.
		if isServiceProvided(services, authTokenSources) {
			for _, service := range services {
				// Record all available services that satisfy the requirement.
				if _, ok := authTokenSources[service]; ok {
					usedServices[service] = struct{}{}
				}
			}
		} else {
			// If no match was found, this parameter is still required by the user.
			requiredAuthnParams[param] = services
		}
	}

	// Find which of the required authz tokens are covered by available services.
	var requiredAuthzTokens []string
	isAuthzMet := false
	for _, reqToken := range reqAuthzTokens {
		// If an available service can satisfy one of the token requirements mark
		// the authorization requirement as met and record the service that was used.
		if _, ok := authTokenSources[reqToken]; ok {
			isAuthzMet = true
			usedServices[reqToken] = struct{}{}
		}
	}

	// After checking all tokens, if the authorization requirement was still not met...
	// ...then ALL original tokens are still required.
	if !isAuthzMet {
		requiredAuthzTokens = reqAuthzTokens
	}

	// Convert the `usedServices` map (acting as a set) into a slice for the return value.
	usedServicesSlice := make([]string, 0, len(usedServices))
	for service := range usedServices {
		usedServicesSlice = append(usedServicesSlice, service)
	}
	return requiredAuthnParams, requiredAuthzTokens, usedServicesSlice
}

// isServiceProvided checks if any of the required services are available in the
// provided token sources. It returns true on the first match.
func isServiceProvided(requiredServices []string, providedTokenSources map[string]oauth2.TokenSource) bool {
	for _, service := range requiredServices {
		if _, ok := providedTokenSources[service]; ok {
			return true
		}
	}
	return false
}

// findUnusedKeys calculates the set difference between a provided set of keys
// and a used set of keys. It returns a slice of strings containing keys that
// are in the `provided` map but not in the `used` map.
func findUnusedKeys(provided, used map[string]struct{}) []string {
	unused := make([]string, 0)
	for k := range provided {
		if _, ok := used[k]; !ok {
			unused = append(unused, k)
		}
	}
	return unused
}

// stringTokenSource is a custom type that implements the oauth2.TokenSource interface.
type customTokenSource struct {
	provider func() string
}

// This function converts a custom function that returns a string into an oauth2.TokenSource type.
//
// Inputs:
//   - provider: A custom function that returns a token as a string.
//
// Returns:
//   - An oauth2.TokenSource that wraps the custom function.
func NewCustomTokenSource(provider func() string) oauth2.TokenSource {
	return &customTokenSource{
		provider: provider,
	}
}

func (s *customTokenSource) Token() (*oauth2.Token, error) {
	tokenStr := s.provider()
	return &oauth2.Token{
		AccessToken: tokenStr,
	}, nil
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

// schemaToMap recursively converts a ParameterSchema to a map with it's type and description.
func schemaToMap(p *ParameterSchema) map[string]any {
	// Basic schema with type and description
	schema := map[string]any{
		"type":        p.Type,
		"description": p.Description,
	}

	// If the type is "array", recursively define what's in the array.
	if p.Type == "array" && p.Items != nil {
		schema["items"] = schemaToMap(p.Items)
	}

	return schema
}
