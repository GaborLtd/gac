package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestScopeRejectsOutsideRepository(t *testing.T) {
	root := t.TempDir()
	repo := Repo{Root: root, CWD: filepath.Join(root, "sub")}
	if _, err := repo.Scope([]string{"../../outside"}); err == nil {
		t.Fatal("expected outside path error")
	}
}

func TestPathCommitDoesNotIncludeUnstagedFile(t *testing.T) {
	root := t.TempDir()
	runGit(t, root, "init", "-q")
	runGit(t, root, "config", "user.email", "test@example.com")
	runGit(t, root, "config", "user.name", "Test")
	write(t, filepath.Join(root, "a.txt"), "a0")
	write(t, filepath.Join(root, "b.txt"), "b0")
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-q", "-m", "init")
	write(t, filepath.Join(root, "a.txt"), "a1")
	write(t, filepath.Join(root, "b.txt"), "b1")
	runGit(t, root, "add", "a.txt")

	repo, err := Discover(context.Background(), NewOSRunner(), root)
	if err != nil {
		t.Fatal(err)
	}
	scope, err := repo.Scope([]string{"a.txt"})
	if err != nil {
		t.Fatal(err)
	}
	names, err := repo.StagedNames(context.Background(), scope)
	if err != nil {
		t.Fatal(err)
	}
	staged := nonEmpty(strings.TrimSpace(names))
	if len(staged) != 1 || staged[0] != "a.txt" {
		t.Fatalf("staged names: %v", staged)
	}
	if got, err := repo.UnstagedNames(context.Background(), staged); err != nil || got != "" {
		t.Fatalf("unexpected unstaged target: %q, %v", got, err)
	}
	if err := repo.Commit(context.Background(), "fix: update a", staged); err != nil {
		t.Fatal(err)
	}
	changed := runGit(t, root, "show", "--format=", "--name-only", "HEAD")
	if strings.TrimSpace(changed) != "a.txt" {
		t.Fatalf("commit changed: %q", changed)
	}
}

func TestCommitAllIncludesTrackedChangesAndExcludesUntracked(t *testing.T) {
	root := t.TempDir()
	runGit(t, root, "init", "-q")
	runGit(t, root, "config", "user.email", "test@example.com")
	runGit(t, root, "config", "user.name", "Test")
	write(t, filepath.Join(root, "a.txt"), "a0")
	write(t, filepath.Join(root, "b.txt"), "b0")
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-q", "-m", "init")
	write(t, filepath.Join(root, "a.txt"), "a1")
	write(t, filepath.Join(root, "b.txt"), "b1")
	write(t, filepath.Join(root, "untracked.txt"), "u1")
	runGit(t, root, "add", "a.txt")

	repo, err := Discover(context.Background(), NewOSRunner(), root)
	if err != nil {
		t.Fatal(err)
	}
	scope, err := repo.Scope(nil)
	if err != nil {
		t.Fatal(err)
	}
	diff, err := repo.WorktreeDiff(context.Background(), scope)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(diff, "a1") || !strings.Contains(diff, "b1") {
		t.Fatalf("worktree diff did not include tracked changes: %q", diff)
	}
	names, err := repo.ChangedNames(context.Background(), scope)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(names, "untracked.txt") {
		t.Fatalf("untracked file appeared in changed names: %q", names)
	}
	if err := repo.CommitAll(context.Background(), "feat: update tracked files"); err != nil {
		t.Fatal(err)
	}
	changed := runGit(t, root, "show", "--format=", "--name-only", "HEAD")
	if strings.Contains(changed, "untracked.txt") || !strings.Contains(changed, "a.txt") || !strings.Contains(changed, "b.txt") {
		t.Fatalf("unexpected commit files: %q", changed)
	}
}

func TestWorktreeMethodsHandleRepositoryWithoutHEAD(t *testing.T) {
	root := t.TempDir()
	runGit(t, root, "init", "-q")
	write(t, filepath.Join(root, "a.txt"), "a0")
	runGit(t, root, "add", "a.txt")

	repo, err := Discover(context.Background(), NewOSRunner(), root)
	if err != nil {
		t.Fatal(err)
	}
	scope, err := repo.Scope(nil)
	if err != nil {
		t.Fatal(err)
	}
	diff, err := repo.WorktreeDiff(context.Background(), scope)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(diff, "a0") {
		t.Fatalf("initial staged diff missing: %q", diff)
	}
}

func TestCurrentDirectoryScopeFromSubdirectory(t *testing.T) {
	root := t.TempDir()
	runGit(t, root, "init", "-q")
	os.MkdirAll(filepath.Join(root, "sub"), 0o700)
	write(t, filepath.Join(root, "sub", "a.txt"), "a0")
	write(t, filepath.Join(root, "root.txt"), "r0")
	runGit(t, root, "add", ".")

	sub := filepath.Join(root, "sub")
	repo, err := Discover(context.Background(), NewOSRunner(), sub)
	if err != nil {
		t.Fatal(err)
	}
	scope, err := repo.Scope(nil)
	if err != nil {
		t.Fatal(err)
	}
	names, err := repo.StagedNames(context.Background(), scope)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(names) != "sub/a.txt" {
		t.Fatalf("current directory scope selected: %q", names)
	}
}

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return string(out)
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func nonEmpty(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}
