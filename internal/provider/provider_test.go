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

func TestAgyInvocationNormalizesLegacyDisplayLabel(t *testing.T) {
	r := &captureRunner{}
	p := NewAgy(r)
	if _, err := p.Generate(context.Background(), "gemini-3.6-flash-low\tGemini 3.6 Flash (Low)", "hello"); err != nil {
		t.Fatal(err)
	}
	want := []string{"--model", "Gemini 3.6 Flash (Low)", "--print", "hello"}
	if !same(r.args, want) {
		t.Fatalf("args %v, want %v", r.args, want)
	}
}
func TestAgyListsModels(t *testing.T) {
	r := &modelsRunner{}
	models, err := NewAgy(r).ListModels(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != 2 || models[0].ID != "gemini-3.5-flash-low" || models[0].DisplayName != "Gemini 3.5 Flash (Low)" || models[1].ID != "gemini-3.5-flash-medium" || models[1].DisplayName != "Gemini 3.5 Flash (Medium)" {
		t.Fatalf("unexpected models: %v", models)
	}
	if !same(r.args, []string{"models"}) {
		t.Fatalf("args %v", r.args)
	}
}

func TestCodexAndClaudeExposeDocumentationWhenModelListingIsUnavailable(t *testing.T) {
	for _, p := range []Provider{NewCodex(&captureRunner{}), NewClaude(&captureRunner{})} {
		if _, err := p.ListModels(context.Background()); err == nil {
			t.Fatalf("%s unexpectedly listed models", p.Name())
		}
		if p.DocsURL() == "" {
			t.Fatalf("%s has no documentation URL", p.Name())
		}
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

func TestCodexInvocationWithLowEffort(t *testing.T) {
	r := &captureRunner{}
	ep, ok := NewCodex(r).(EffortProvider)
	if !ok {
		t.Fatal("codex does not support effort")
	}
	if _, err := ep.GenerateWithEffort(context.Background(), "gpt-5.4-mini", "low", "p"); err != nil {
		t.Fatal(err)
	}
	want := []string{"exec", "--model", "gpt-5.4-mini", "--effort", "low", "p"}
	if !same(r.args, want) {
		t.Fatalf("args %v, want %v", r.args, want)
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

type modelsRunner struct {
	args []string
}

func (r *modelsRunner) Run(_ context.Context, _ string, args []string) (string, string, error) {
	r.args = args
	if len(args) == 1 && args[0] == "models" {
		return "Available models:\ngemini-3.5-flash-low\tGemini 3.5 Flash (Low)\ngemini-3.5-flash-medium  Gemini 3.5 Flash (Medium)\n", "", nil
	}
	return "", "", nil
}
