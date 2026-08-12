package message

import "testing"

func TestCleanRemovesCodeFenceAndPrefix(t *testing.T) {
	got, err := Clean("```text\ncommit message: feat(cli): add generator\n```")
	if err != nil || got != "feat(cli): add generator" {
		t.Fatalf("got %q, err %v", got, err)
	}
}

func TestCleanRejectsNonConventionalMessage(t *testing.T) {
	if _, err := Clean("please commit these changes"); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestCITokens(t *testing.T) {
	if !HasCIToken("fix: issue [CI SKIP]") {
		t.Fatal("expected CI token")
	}
	if got := AppendCIToken("fix: issue"); got != "fix: issue [skip ci]" {
		t.Fatalf("got %q", got)
	}
	if got := AppendCIToken("fix: issue [ci skip]"); got != "fix: issue [ci skip]" {
		t.Fatalf("duplicate token: %q", got)
	}
}
