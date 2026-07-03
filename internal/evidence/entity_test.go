package evidence

import (
	"encoding/json"
	"strings"
	"testing"

	"casemind/internal/privacy"
)

// TestPublicEvidenceNeverLeaksInternalTruth fails if the public evidence
// JSON exposes internal_truth (07_SECURITY_MODEL.md).
func TestPublicEvidenceNeverLeaksInternalTruth(t *testing.T) {
	e := &Evidence{
		Title:         "Bronze maquette",
		Type:          "weapon",
		PublicData:    json.RawMessage(`{"observation":"slightly off its dust ring"}`),
		InternalTruth: json.RawMessage(`{"actual":"murder weapon with blood traces"}`),
		Discovered:    true,
	}
	body, err := json.Marshal(map[string]any{"evidence": []PublicEvidence{e.Public()}})
	if err != nil {
		t.Fatal(err)
	}
	if err := privacy.MustBeClean(body); err != nil {
		t.Fatalf("public evidence leaks truth layer: %v", err)
	}
	if strings.Contains(string(body), "internal_truth") || strings.Contains(string(body), "murder weapon") {
		t.Fatalf("public evidence JSON leaks internal truth: %s", body)
	}
}
