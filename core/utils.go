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

import "golang.org/x/oauth2"

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
