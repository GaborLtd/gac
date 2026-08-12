package provider

import (
	"context"
	"testing"
)

type captureRunner struct {
	name string
	args []string
}

func (r *captureRunner) Run(_ context.Context, name string, args []string) (string, string, error) {
	r.name, r.args = name, args
	return "feat: generated\n", "", nil
}

func TestAgyInvocation(t *testing.T) {
	r := &captureRunner{}
	p := NewAgy(r)
	got, err := p.Generate(context.Background(), "flash-low", "hello")
	if err != nil || got != "feat: generated\n" {
		t.Fatalf("got %q, err %v", got, err)
	}
	want := []string{"--model", "flash-low", "--print", "hello"}
	if !same(r.args, want) {
		t.Fatalf("args %v, want %v", r.args, want)
	}
}

func TestCodexAndClaudeInvocation(t *testing.T) {
	for _, tc := range []struct {
		name string
		p    Provider
		want []string
	}{
		{"codex", NewCodex(&captureRunner{}), []string{"exec", "--model", "m", "p"}},
		{"claude", NewClaude(&captureRunner{}), []string{"-p", "p", "--model", "m"}},
	} {
		r := &captureRunner{}
		switch tc.name {
		case "codex":
			tc.p = NewCodex(r)
		case "claude":
			tc.p = NewClaude(r)
		}
		if _, err := tc.p.Generate(context.Background(), "m", "p"); err != nil {
			t.Fatal(err)
		}
		if !same(r.args, tc.want) {
			t.Fatalf("%s args %v, want %v", tc.name, r.args, tc.want)
		}
	}
}

func same(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
