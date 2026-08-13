package git

import (
	"context"
	"fmt"
	"strings"
)

// LatestReleaseTag 回傳目前可達的最新 vMAJOR.MINOR.PATCH tag；沒有 tag 時回傳空字串。
func (repo Repo) LatestReleaseTag(ctx context.Context) (string, error) {
	out, stderr, err := repo.Run.Run(ctx, repo.Root, "git", []string{"describe", "--tags", "--match", "v[0-9]*", "--abbrev=0"}, "")
	if err != nil {
		if strings.Contains(strings.ToLower(stderr), "no names found") || strings.Contains(strings.ToLower(stderr), "cannot describe") {
			return "", nil
		}
		return "", fmt.Errorf("git describe release tag failed: %s", strings.TrimSpace(stderr))
	}
	return strings.TrimSpace(out), nil
}

// ReleaseCommits 回傳上一個 release tag（不含）到 HEAD 的 commit subject。
func (repo Repo) ReleaseCommits(ctx context.Context, previousTag string) ([]string, error) {
	rangeSpec := "HEAD"
	if strings.TrimSpace(previousTag) != "" {
		rangeSpec = previousTag + "..HEAD"
	}
	out, err := repo.git(ctx, "log", "--no-merges", "--format=%h%x09%s", rangeSpec)
	if err != nil {
		return nil, err
	}
	var commits []string
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if strings.TrimSpace(line) != "" {
			commits = append(commits, strings.TrimSpace(line))
		}
	}
	return commits, nil
}

// WorktreeStatus 回傳完整 status；release tag 前可用來阻止 dirty worktree。
func (repo Repo) WorktreeStatus(ctx context.Context) (string, error) {
	return repo.git(ctx, "status", "--porcelain")
}

// TagExists 判斷指定 tag 是否已存在。
func (repo Repo) TagExists(ctx context.Context, tag string) (bool, error) {
	_, _, err := repo.Run.Run(ctx, repo.Root, "git", []string{"show-ref", "--verify", "--quiet", "refs/tags/" + tag}, "")
	if err == nil {
		return true, nil
	}
	// show-ref --verify 以非零狀態表示不存在；其他錯誤會在建立 tag 時再次顯示具體原因。
	return false, nil
}

// CreateAnnotatedTag 建立本機 annotated tag，不執行 push。
func (repo Repo) CreateAnnotatedTag(ctx context.Context, tag, message string) error {
	_, stderr, err := repo.Run.Run(ctx, repo.Root, "git", []string{"tag", "-a", tag, "-F", "-"}, message+"\n")
	if err != nil {
		return fmt.Errorf("git tag failed: %s", strings.TrimSpace(stderr))
	}
	return nil
}
