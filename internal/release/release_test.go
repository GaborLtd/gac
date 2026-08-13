package release

import (
	"strings"
	"testing"
)

func TestValidateVersion(t *testing.T) {
	for _, value := range []string{"v0.1.3", "v10.20.30"} {
		if err := ValidateVersion(value); err != nil {
			t.Fatalf("ValidateVersion(%q): %v", value, err)
		}
	}
	for _, value := range []string{"0.1.3", "v1.2", "v1.2.3-beta", "release"} {
		if err := ValidateVersion(value); err == nil {
			t.Fatalf("ValidateVersion(%q) unexpectedly succeeded", value)
		}
	}
}

func TestCompareVersion(t *testing.T) {
	got, err := CompareVersion("v0.10.0", "v0.9.9")
	if err != nil || got != 1 {
		t.Fatalf("compare = %d, err = %v", got, err)
	}
	got, err = CompareVersion("v1.0.0", "v1.0.0")
	if err != nil || got != 0 {
		t.Fatalf("equal compare = %d, err = %v", got, err)
	}
}

func TestBuildPromptContainsOnlyReleaseInputs(t *testing.T) {
	prompt := BuildPrompt(Plan{
		Version:     "v0.1.3",
		PreviousTag: "v0.1.2",
		Commits:     []Commit{{Hash: "abc123", Subject: "fix: repair model selection"}},
	}, "zh-TW")
	for _, expected := range []string{"v0.1.3", "v0.1.2", "zh-TW", "abc123 fix: repair model selection"} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("prompt missing %q: %s", expected, prompt)
		}
	}
}

func TestCleanMessageAllowsMarkdownAndRemovesFence(t *testing.T) {
	got, err := CleanMessage("```markdown\n# Release v0.1.3\n\n- Fix model selection\n```")
	if err != nil {
		t.Fatal(err)
	}
	if got != "# Release v0.1.3\n\n- Fix model selection" {
		t.Fatalf("message = %q", got)
	}
}
