package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaborltd/gac/internal/git"
)

func TestParseReleaseCommits(t *testing.T) {
	got := parseReleaseCommits([]string{"abc123\tfeat: add release notes", "def456\tfix: avoid push", ""})
	if len(got) != 2 || got[0].Hash != "abc123" || got[1].Subject != "fix: avoid push" {
		t.Fatalf("commits = %#v", got)
	}
}

func TestConfirmReleaseTagCreatesLocalTagOnly(t *testing.T) {
	root := t.TempDir()
	runReleaseGit(t, root, "init", "-q")
	runReleaseGit(t, root, "config", "user.email", "test@example.com")
	runReleaseGit(t, root, "config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("a"), 0o600); err != nil {
		t.Fatal(err)
	}
	runReleaseGit(t, root, "add", "a.txt")
	runReleaseGit(t, root, "commit", "-q", "-m", "feat: initial")
	repo, err := git.Discover(context.Background(), git.NewOSRunner(), root)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	app := application{in: strings.NewReader("y\n"), out: &out, err: &bytes.Buffer{}}
	if err := app.confirmReleaseTag(context.Background(), repo, "v0.1.3", "# Release v0.1.3\n\n- Initial"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Created local tag v0.1.3") || !strings.Contains(out.String(), "git push origin v0.1.3") {
		t.Fatalf("output = %q", out.String())
	}
	if got := strings.TrimSpace(runReleaseGit(t, root, "tag", "--list", "v0.1.3")); got != "v0.1.3" {
		t.Fatalf("tag = %q", got)
	}
}

func runReleaseGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return string(out)
}
