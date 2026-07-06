package runtime

import "testing"

func TestLanguageNormalization(t *testing.T) {
	cases := map[string]string{
		"fa": "fa", "fa-IR": "fa", "Persian": "fa", "farsi": "fa",
		"en": "en", "en-US": "en", "": "en", "de": "en",
	}
	for in, want := range cases {
		if got := Language(in); got != want {
			t.Errorf("Language(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestLanguageDirectiveIsLanguageSpecific: the Persian directive must name
// Persian + rtl and the English directive must name English + ltr, so a
// Persian UI reliably gets Persian AI output and English gets English.
func TestLanguageDirectiveIsLanguageSpecific(t *testing.T) {
	fa := LanguageDirective("fa-IR")
	if !contains(fa, "Persian") || !contains(fa, "rtl") || !contains(fa, `"fa"`) {
		t.Errorf("Persian directive missing language/direction markers:\n%s", fa)
	}
	en := LanguageDirective("en")
	if !contains(en, "English") || !contains(en, "ltr") || !contains(en, `"en"`) {
		t.Errorf("English directive missing language/direction markers:\n%s", en)
	}
	if contains(en, "Persian") {
		t.Errorf("English directive must not mention Persian:\n%s", en)
	}
	// Both must forbid switching languages mid-response.
	for _, d := range []string{fa, en} {
		if !contains(d, "Never switch languages") {
			t.Errorf("directive missing no-switch rule:\n%s", d)
		}
	}
}

func contains(haystack, needle string) bool {
	return containsStr(haystack, needle)
}
