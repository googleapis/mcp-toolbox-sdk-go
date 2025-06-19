package core

import (
	"reflect"
	"sort"
	"testing"

	"golang.org/x/oauth2"
)

func TestFindUnusedKeys(t *testing.T) {
	testCases := []struct {
		name     string
		provided map[string]struct{}
		used     map[string]struct{}
		expected []string
	}{
		{
			name:     "Finds one unused key",
			provided: map[string]struct{}{"a": {}, "b": {}},
			used:     map[string]struct{}{"a": {}},
			expected: []string{"b"},
		},
		{
			name:     "Finds multiple unused keys",
			provided: map[string]struct{}{"a": {}, "b": {}, "c": {}},
			used:     map[string]struct{}{"a": {}},
			expected: []string{"b", "c"},
		},
		{
			name:     "Finds no unused keys",
			provided: map[string]struct{}{"a": {}, "b": {}},
			used:     map[string]struct{}{"a": {}, "b": {}},
			expected: []string{},
		},
		{
			name:     "Handles empty provided set",
			provided: map[string]struct{}{},
			used:     map[string]struct{}{"a": {}},
			expected: []string{},
		},
		{
			name:     "Handles empty used set",
			provided: map[string]struct{}{"a": {}, "b": {}},
			used:     map[string]struct{}{},
			expected: []string{"a", "b"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := findUnusedKeys(tc.provided, tc.used)
			sort.Strings(result)
			sort.Strings(tc.expected)
			if !reflect.DeepEqual(result, tc.expected) {
				t.Errorf("expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestIdentifyAuthRequirements(t *testing.T) {
	t.Run("Satisfies authn and authz", func(t *testing.T) {
		reqAuthn := map[string][]string{"paramA": {"google"}}
		reqAuthz := []string{"github"}
		sources := map[string]oauth2.TokenSource{"google": nil, "github": nil}

		unmetAuthn, unmetAuthz, used := identifyAuthRequirements(reqAuthn, reqAuthz, sources)

		if len(unmetAuthn) != 0 {
			t.Errorf("Expected 0 unmet authn params, got %d", len(unmetAuthn))
		}
		if len(unmetAuthz) != 0 {
			t.Errorf("Expected 0 unmet authz tokens, got %d", len(unmetAuthz))
		}
		sort.Strings(used)
		if !reflect.DeepEqual(used, []string{"github", "google"}) {
			t.Errorf("Expected used keys [github, google], got %v", used)
		}
	})

	t.Run("Negative Test - Fails to satisfy authn", func(t *testing.T) {
		reqAuthn := map[string][]string{"paramA": {"google"}}
		sources := map[string]oauth2.TokenSource{"github": nil}

		unmetAuthn, _, _ := identifyAuthRequirements(reqAuthn, nil, sources)

		if len(unmetAuthn) != 1 {
			t.Fatal("Expected 1 unmet authn param")
		}
		if _, ok := unmetAuthn["paramA"]; !ok {
			t.Error("Expected 'paramA' to be in the unmet set")
		}
	})

	t.Run("Negative Test - Fails to satisfy authz", func(t *testing.T) {
		reqAuthz := []string{"github"}
		sources := map[string]oauth2.TokenSource{"google": nil}

		_, unmetAuthz, _ := identifyAuthRequirements(nil, reqAuthz, sources)

		if !reflect.DeepEqual(unmetAuthz, []string{"github"}) {
			t.Errorf("Expected unmet authz to be [github], got %v", unmetAuthz)
		}
	})
}
