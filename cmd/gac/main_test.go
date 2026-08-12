package main

import (
	"bytes"
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
