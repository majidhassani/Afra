// Package privacy enforces the Truth Layer boundary: it scans outgoing JSON
// for field names that must never reach the client. It is used both by the
// HTTP privacy-guard middleware and by tests.
package privacy

import (
	"encoding/json"
	"fmt"
)

// ForbiddenKeys are JSON object keys that expose Truth Layer data.
// The security model (07_SECURITY_MODEL.md) requires responses to fail
// if any of these appear.
var ForbiddenKeys = []string{
	"case_bible",
	"culprit_id",
	"is_culprit",
	"secrets",
	"lie_profile",
	"private_memory",
	"internal_truth",
	"hidden_facts",
	"evidence_truth",
	"suspect_secrets",
	"real_timeline",
	"false_leads",
	"truth",
	// Image-generation prompts are internal inputs; the public visual
	// contract is avatar_url/image_url + status only.
	"avatar_prompt",
	"thumbnail_prompt",
	"visual_prompt",
	"avatar_or_thumbnail_prompt",
	// WorldBible / hidden mission state must never reach clients.
	"world_bible",
	"hidden_state",
	"private_state",
}

var forbidden = func() map[string]bool {
	m := make(map[string]bool, len(ForbiddenKeys))
	for _, k := range ForbiddenKeys {
		m[k] = true
	}
	return m
}()

// ScanJSON returns the forbidden keys found anywhere (nested included) in
// the given JSON document. A non-JSON body returns no findings.
func ScanJSON(body []byte) []string {
	var doc any
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil
	}
	seen := map[string]bool{}
	walk(doc, seen)
	if len(seen) == 0 {
		return nil
	}
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	return out
}

func walk(v any, seen map[string]bool) {
	switch t := v.(type) {
	case map[string]any:
		for k, child := range t {
			if forbidden[k] {
				seen[k] = true
			}
			walk(child, seen)
		}
	case []any:
		for _, child := range t {
			walk(child, seen)
		}
	}
}

// MustBeClean returns an error if body contains forbidden keys.
func MustBeClean(body []byte) error {
	if leaks := ScanJSON(body); len(leaks) > 0 {
		return fmt.Errorf("privacy violation: forbidden fields in response: %v", leaks)
	}
	return nil
}
