package git

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseHistoryAndAnnotatedTag(t *testing.T) {
	root := t.TempDir()
	runGit(t, root, "init", "-q")
	runGit(t, root, "config", "user.email", "test@example.com")
	runGit(t, root, "config", "user.name", "Test")
	write(t, filepath.Join(root, "a.txt"), "a0")
	runGit(t, root, "add", "a.txt")
	runGit(t, root, "commit", "-q", "-m", "feat: initial")
	runGit(t, root, "tag", "-a", "v0.1.0", "-m", "release v0.1.0")
	write(t, filepath.Join(root, "a.txt"), "a1")
	runGit(t, root, "add", "a.txt")
	runGit(t, root, "commit", "-q", "-m", "fix: update file")

	repo, err := Discover(context.Background(), NewOSRunner(), root)
	if err != nil {
		t.Fatal(err)
	}
	latest, err := repo.LatestReleaseTag(context.Background())
	if err != nil || latest != "v0.1.0" {
		t.Fatalf("latest = %q, err = %v", latest, err)
	}
	commits, err := repo.ReleaseCommits(context.Background(), latest)
	if err != nil || len(commits) != 1 || !strings.Contains(commits[0], "fix: update file") {
		t.Fatalf("commits = %v, err = %v", commits, err)
	}
	if err := repo.CreateAnnotatedTag(context.Background(), "v0.1.1", "# Release v0.1.1\n\n- Update file"); err != nil {
		t.Fatal(err)
	}
	exists, err := repo.TagExists(context.Background(), "v0.1.1")
	if err != nil || !exists {
		t.Fatalf("tag exists = %v, err = %v", exists, err)
	}
	message := runGit(t, root, "for-each-ref", "--format=%(contents)", "refs/tags/v0.1.1")
	if !strings.Contains(message, "Update file") {
		t.Fatalf("tag message = %q", message)
	}
}

func TestReleaseCommitsWithoutPreviousTag(t *testing.T) {
	root := t.TempDir()
	runGit(t, root, "init", "-q")
	runGit(t, root, "config", "user.email", "test@example.com")
	runGit(t, root, "config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("a"), 0o600); err != nil {
		t.Fatal(err)
	}
	runGit(t, root, "add", "a.txt")
	runGit(t, root, "commit", "-q", "-m", "feat: first")
	repo, err := Discover(context.Background(), NewOSRunner(), root)
	if err != nil {
		t.Fatal(err)
	}
	latest, err := repo.LatestReleaseTag(context.Background())
	if err != nil || latest != "" {
		t.Fatalf("latest = %q, err = %v", latest, err)
	}
	commits, err := repo.ReleaseCommits(context.Background(), "")
	if err != nil || len(commits) != 1 {
		t.Fatalf("commits = %v, err = %v", commits, err)
	}
}
