package git

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Runner 是 Git command abstraction。
type Runner interface {
	Run(ctx context.Context, dir, name string, args []string, stdin string) (stdout, stderr string, err error)
}

type osRunner struct{}

// NewOSRunner 建立實際執行 git 的 runner。
func NewOSRunner() Runner { return osRunner{} }

func (osRunner) Run(ctx context.Context, dir, name string, args []string, stdin string) (string, string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader(stdin)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// Repo 代表目前 Git repository。
type Repo struct {
	Root string
	CWD  string
	Run  Runner
}

// Discover 找出目前 repository root。
func Discover(ctx context.Context, r Runner, cwd string) (Repo, error) {
	canonicalCWD, err := filepath.EvalSymlinks(cwd)
	if err == nil {
		cwd = canonicalCWD
	}
	out, stderr, err := r.Run(ctx, cwd, "git", []string{"rev-parse", "--show-toplevel"}, "")
	if err != nil {
		return Repo{}, fmt.Errorf("不是 Git repository：%s", strings.TrimSpace(stderr))
	}
	root := strings.TrimSpace(out)
	if root == "" {
		return Repo{}, fmt.Errorf("無法取得 Git repository root")
	}
	if canonicalRoot, evalErr := filepath.EvalSymlinks(root); evalErr == nil {
		root = canonicalRoot
	}
	return Repo{Root: root, CWD: cwd, Run: r}, nil
}

// Scope 將使用者輸入轉成 repository-relative pathspec；不允許離開 repository。
func (repo Repo) Scope(paths []string) ([]string, error) {
	if len(paths) == 0 {
		return []string{repo.relative(repo.CWD)}, nil
	}
	result := make([]string, 0, len(paths))
	for _, raw := range paths {
		if raw == "" {
			return nil, fmt.Errorf("path 不可為空")
		}
		path := raw
		if !filepath.IsAbs(path) {
			path = filepath.Join(repo.CWD, path)
		}
		path = filepath.Clean(path)
		rel, err := filepath.Rel(repo.Root, path)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return nil, fmt.Errorf("path 超出 repository：%s", raw)
		}
		if rel == "." {
			rel = "."
		}
		result = append(result, filepath.ToSlash(rel))
	}
	return result, nil
}

func (repo Repo) relative(path string) string {
	rel, err := filepath.Rel(repo.Root, path)
	if err != nil || rel == "" {
		return "."
	}
	return filepath.ToSlash(rel)
}

func (repo Repo) git(ctx context.Context, args ...string) (string, error) {
	out, stderr, err := repo.Run.Run(ctx, repo.Root, "git", args, "")
	if err != nil {
		return "", fmt.Errorf("git %s 失敗：%s", strings.Join(args, " "), strings.TrimSpace(stderr))
	}
	return out, nil
}

// Stat 取得 staged stat。
func (repo Repo) Stat(ctx context.Context, scope []string) (string, error) {
	args := []string{"diff", "--cached", "--stat", "--"}
	args = append(args, scope...)
	return repo.git(ctx, args...)
}

// Diff 取得 staged diff。
func (repo Repo) Diff(ctx context.Context, scope []string) (string, error) {
	args := []string{"diff", "--cached", "--"}
	args = append(args, scope...)
	return repo.git(ctx, args...)
}

// StagedNames 取得 scope 內已 staged 的檔案清單。
func (repo Repo) StagedNames(ctx context.Context, scope []string) (string, error) {
	args := []string{"diff", "--cached", "--name-only", "--"}
	args = append(args, scope...)
	return repo.git(ctx, args...)
}

// UnstagedNames 找出 scope 中尚未 staged 的變更。
func (repo Repo) UnstagedNames(ctx context.Context, scope []string) (string, error) {
	args := []string{"diff", "--name-only", "--"}
	args = append(args, scope...)
	return repo.git(ctx, args...)
}

// Commit 只使用給定的 staged file 清單，不使用 -a。
func (repo Repo) Commit(ctx context.Context, message string, stagedNames []string) error {
	args := []string{"commit", "--only", "-m", message, "--"}
	args = append(args, stagedNames...)
	_, stderr, err := repo.Run.Run(ctx, repo.Root, "git", args, "")
	if err != nil {
		return fmt.Errorf("git commit 失敗：%s", strings.TrimSpace(stderr))
	}
	return nil
}

// IsWorktreePath 回傳 path 是否在目前工作目錄範圍，供呼叫端顯示訊息。
func (repo Repo) IsWorktreePath(path string) bool {
	rel, err := filepath.Rel(repo.CWD, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// EnsureExists 只在需要時提供清楚的本機 path 錯誤；deleted file 仍交給 Git pathspec 處理。
func EnsureExists(path string) error {
	if _, err := os.Stat(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
