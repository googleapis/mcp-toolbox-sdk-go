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
