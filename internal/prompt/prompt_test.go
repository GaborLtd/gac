package prompt

import "testing"

func TestBuildDefaultIncludesInputsAndContract(t *testing.T) {
	got, err := Build(Input{Language: "en", Stat: "1 file changed", Diff: "diff --git", Context: "release note"}, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"1 file changed", "diff --git", "release note", "Fixed output contract", "Write the message in en"} {
		if !contains(got, want) {
			t.Fatalf("prompt missing %q: %s", want, got)
		}
	}
}

func TestBuildCustomTemplate(t *testing.T) {
	got, err := Build(Input{Language: "zh-TW", Diff: "D"}, "lang={{.Language}} diff={{.Diff}}")
	if err != nil || !contains(got, "lang=zh-TW diff=D") || !contains(got, "Write the message in zh-TW") {
		t.Fatalf("got %q, err %v", got, err)
	}
}

func TestBuildDefaultsLanguageToEnglish(t *testing.T) {
	got, err := Build(Input{Diff: "D"}, "custom")
	if err != nil || !contains(got, "Write the message in en") {
		t.Fatalf("got %q, err %v", got, err)
	}
}

func TestBuildRejectsUnknownVariable(t *testing.T) {
	if _, err := Build(Input{}, "{{.Unknown}}"); err == nil {
		t.Fatal("expected template error")
	}
}

func contains(s, want string) bool {
	for i := 0; i+len(want) <= len(s); i++ {
		if s[i:i+len(want)] == want {
			return true
		}
	}
	return false
}
