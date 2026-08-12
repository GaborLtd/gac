package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestNonInteractiveOutputsOnlyMessageAndDoesNotCommit(t *testing.T) {
	root := t.TempDir()
	run(t, root, "init", "-q")
	run(t, root, "config", "user.email", "test@example.com")
	run(t, root, "config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("a"), 0o600); err != nil {
		t.Fatal(err)
	}
	run(t, root, "add", "a.txt")

	bin := t.TempDir()
	agy := filepath.Join(bin, "agy")
	if err := os.WriteFile(agy, []byte("#!/bin/sh\nprintf 'feat: generated\\n'\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	oldPath := os.Getenv("PATH")
	if err := os.Setenv("PATH", bin+string(os.PathListSeparator)+oldPath); err != nil {
		t.Fatal(err)
	}
	defer os.Setenv("PATH", oldPath)

	cfg := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(cfg, []byte("provider: agy\nmodel: cheap\nlanguage: en\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWD)

	var out, stderr bytes.Buffer
	app := application{in: strings.NewReader(""), out: &out, err: &stderr}
	if err := app.run([]string{"-n", "--config", cfg}); err != nil {
		t.Fatal(err)
	}
	if out.String() != "feat: generated\n" {
		t.Fatalf("stdout: %q", out.String())
	}
	if stderr.String() != "" {
		t.Fatalf("stderr: %q", stderr.String())
	}
	cmd := exec.Command("git", "rev-parse", "--verify", "HEAD")
	cmd.Dir = root
	if err := cmd.Run(); err == nil {
		t.Fatal("non-interactive mode unexpectedly created a commit")
	}
}

func TestChooseModelListsAndSelectsByNumber(t *testing.T) {
	var out bytes.Buffer
	app := application{in: strings.NewReader("1\n"), out: &out, err: &bytes.Buffer{}}
	got, err := app.chooseModel(context.Background(), fakeModelProvider{models: []string{"strong", "cheap"}}, "")
	if err != nil {
		t.Fatal(err)
	}
	if got != "cheap" {
		t.Fatalf("model = %q, want cheap", got)
	}
	if !strings.Contains(out.String(), "1) cheap") || !strings.Contains(out.String(), "2) strong") {
		t.Fatalf("model list missing: %q", out.String())
	}
	if !strings.Contains(out.String(), "low-cost recommendation") {
		t.Fatalf("low-cost recommendation missing: %q", out.String())
	}
}

func TestChooseModelShowsDocsWhenProviderCannotList(t *testing.T) {
	var out bytes.Buffer
	app := application{in: strings.NewReader("\n"), out: &out, err: &bytes.Buffer{}}
	got, err := app.chooseModel(context.Background(), fakeModelProvider{docs: "https://example.com/models", login: "fake login", listErr: true}, "")
	if err != nil {
		t.Fatal(err)
	}
	if got != "" || !strings.Contains(out.String(), "https://example.com/models") || !strings.Contains(out.String(), "fake login") || !strings.Contains(out.String(), "provider default") {
		t.Fatalf("unexpected fallback: model=%q output=%q", got, out.String())
	}
}

func TestRunRejectsDirectoryPath(t *testing.T) {
	root := t.TempDir()
	run(t, root, "init", "-q")
	if err := os.Mkdir(filepath.Join(root, "docs"), 0o700); err != nil {
		t.Fatal(err)
	}
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWD)

	app := application{in: strings.NewReader(""), out: &bytes.Buffer{}, err: &bytes.Buffer{}}
	err = app.run([]string{"--config", filepath.Join(t.TempDir(), "config.yaml"), "docs"})
	if err == nil || !strings.Contains(err.Error(), "directory paths are not supported") {
		t.Fatalf("expected directory path error, got %v", err)
	}
}

func TestRunRejectsMultiplePaths(t *testing.T) {
	root := t.TempDir()
	run(t, root, "init", "-q")
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWD)

	app := application{in: strings.NewReader(""), out: &bytes.Buffer{}, err: &bytes.Buffer{}}
	err = app.run([]string{"--config", filepath.Join(t.TempDir(), "config.yaml"), "a.txt", "b.txt"})
	if err == nil || !strings.Contains(err.Error(), "only one file path may be provided") {
		t.Fatalf("expected multiple path error, got %v", err)
	}
}

type fakeModelProvider struct {
	models  []string
	docs    string
	login   string
	listErr bool
}

func (p fakeModelProvider) Name() string      { return "fake" }
func (p fakeModelProvider) Detect() error     { return nil }
func (p fakeModelProvider) DocsURL() string   { return p.docs }
func (p fakeModelProvider) LoginHint() string { return p.login }
func (p fakeModelProvider) ListModels(context.Context) ([]string, error) {
	if p.listErr {
		return nil, errors.New("not supported")
	}
	return p.models, nil
}
func (p fakeModelProvider) Generate(context.Context, string, string) (string, error) {
	return "feat: fake", nil
}

func run(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return string(out)
}
