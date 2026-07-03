package suspect

import (
	"encoding/json"
	"testing"

	"casemind/internal/privacy"
)

// TestPublicSuspectNeverLeaksTruthLayer fails if the public suspect JSON
// contains any internal field (07_SECURITY_MODEL.md).
func TestPublicSuspectNeverLeaksTruthLayer(t *testing.T) {
	s := &Suspect{
		Name:          "Elena Voss",
		IsCulprit:     true,
		Secrets:       json.RawMessage(`["she forged the paintings"]`),
		LieProfile:    json.RawMessage(`{"will_lie_about":["the audit"]}`),
		PrivateMemory: json.RawMessage(`["claimed she left at 22:45"]`),
		PublicProfile: json.RawMessage(`{"appearance":"composed"}`),
		KnownFacts:    json.RawMessage(`["restored two paintings"]`),
		Personality:   json.RawMessage(`{"traits":["meticulous"]}`),
	}
	body, err := json.Marshal(map[string]any{"suspects": []PublicSuspect{s.Public()}})
	if err != nil {
		t.Fatal(err)
	}
	if err := privacy.MustBeClean(body); err != nil {
		t.Fatalf("public suspect leaks truth layer: %v", err)
	}
	for _, forbidden := range []string{"is_culprit", "secrets", "lie_profile", "private_memory", "forged"} {
		if containsInsensitive(string(body), forbidden) {
			t.Errorf("public suspect JSON contains %q: %s", forbidden, body)
		}
	}
}

func containsInsensitive(haystack, needle string) bool {
	return json.Valid([]byte(haystack)) && stringContains(haystack, needle)
}

func stringContains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
