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

func findUnusedKeys(provided, used map[string]struct{}) []string {
	unused := make([]string, 0)
	for k := range provided {
		if _, ok := used[k]; !ok {
			unused = append(unused, k)
		}
	}
	return unused
}
