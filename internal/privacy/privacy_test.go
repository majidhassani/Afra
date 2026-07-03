package privacy

import "testing"

func TestScanJSONFindsForbiddenKeysAtAnyDepth(t *testing.T) {
	body := []byte(`{
		"data": {
			"suspects": [
				{"name": "Elena", "nested": {"is_culprit": true}}
			],
			"case": {"deep": [{"deeper": {"lie_profile": {}}}]}
		}
	}`)
	leaks := ScanJSON(body)
	if len(leaks) != 2 {
		t.Fatalf("expected 2 leaks, got %v", leaks)
	}
	found := map[string]bool{}
	for _, l := range leaks {
		found[l] = true
	}
	if !found["is_culprit"] || !found["lie_profile"] {
		t.Fatalf("expected is_culprit and lie_profile, got %v", leaks)
	}
}

func TestScanJSONCleanBody(t *testing.T) {
	body := []byte(`{"data":{"suspects":[{"name":"Elena","stress_level":10}]}}`)
	if leaks := ScanJSON(body); len(leaks) != 0 {
		t.Fatalf("expected no leaks, got %v", leaks)
	}
}

func TestScanJSONAllForbiddenKeys(t *testing.T) {
	for _, key := range ForbiddenKeys {
		body := []byte(`{"` + key + `": "leak"}`)
		if leaks := ScanJSON(body); len(leaks) != 1 {
			t.Errorf("key %q not detected", key)
		}
	}
}

func TestMustBeCleanErrors(t *testing.T) {
	if err := MustBeClean([]byte(`{"secrets": []}`)); err == nil {
		t.Fatal("expected error for forbidden key")
	}
	if err := MustBeClean([]byte(`{"name": "ok"}`)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
