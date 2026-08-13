package release

import (
	"fmt"
	"regexp"
	"strings"
)

// Commit 是 release notes 使用的已提交變更摘要。
type Commit struct {
	Hash    string
	Subject string
}

// Plan 是由上一個 release tag 到目前 HEAD 的發布計畫。
type Plan struct {
	Version     string
	PreviousTag string
	Commits     []Commit
}

var versionPattern = regexp.MustCompile(`^v([0-9]+)\.([0-9]+)\.([0-9]+)$`)

// ValidateVersion 驗證第一版 release 僅接受 vMAJOR.MINOR.PATCH。
func ValidateVersion(value string) error {
	if !versionPattern.MatchString(strings.TrimSpace(value)) {
		return fmt.Errorf("version must use vMAJOR.MINOR.PATCH format: %s", value)
	}
	return nil
}

// CompareVersion 比較兩個已驗證的版本；左側較新時回傳正數。
func CompareVersion(left, right string) (int, error) {
	leftParts, err := versionParts(left)
	if err != nil {
		return 0, err
	}
	rightParts, err := versionParts(right)
	if err != nil {
		return 0, err
	}
	for i := range leftParts {
		if leftParts[i] > rightParts[i] {
			return 1, nil
		}
		if leftParts[i] < rightParts[i] {
			return -1, nil
		}
	}
	return 0, nil
}

func versionParts(value string) ([3]int, error) {
	var parts [3]int
	matches := versionPattern.FindStringSubmatch(strings.TrimSpace(value))
	if matches == nil {
		return parts, fmt.Errorf("version must use vMAJOR.MINOR.PATCH format: %s", value)
	}
	for i := 0; i < 3; i++ {
		var n int
		if _, err := fmt.Sscanf(matches[i+1], "%d", &n); err != nil {
			return parts, fmt.Errorf("invalid version: %s", value)
		}
		parts[i] = n
	}
	return parts, nil
}

// BuildPrompt 建立只使用 commit 摘要的 release message prompt。
func BuildPrompt(plan Plan, language string) string {
	if strings.TrimSpace(language) == "" {
		language = "en"
	}
	var body strings.Builder
	fmt.Fprintf(&body, "Generate concise Markdown release notes for version %s.\n", plan.Version)
	fmt.Fprintf(&body, "Write in %s. Use only the commits supplied below; do not invent changes.\n", language)
	body.WriteString("Include a short summary and grouped bullet points when useful. Do not include a code fence or commentary about this prompt.\n\n")
	fmt.Fprintf(&body, "Previous release: %s\n\n", plan.PreviousTag)
	body.WriteString("=== Commits ===\n")
	if len(plan.Commits) == 0 {
		body.WriteString("No commits found.\n")
	}
	for _, commit := range plan.Commits {
		if commit.Hash == "" {
			fmt.Fprintf(&body, "- %s\n", commit.Subject)
			continue
		}
		fmt.Fprintf(&body, "- %s %s\n", commit.Hash, commit.Subject)
	}
	body.WriteString("\nFixed output contract: return release notes only, suitable as an annotated Git tag message.")
	return body.String()
}

// CleanMessage 移除常見包裝，但保留 Markdown release notes。
func CleanMessage(raw string) (string, error) {
	message := strings.TrimSpace(raw)
	if strings.HasPrefix(message, "```") && strings.HasSuffix(message, "```") {
		lines := strings.Split(message, "\n")
		if len(lines) >= 2 {
			message = strings.TrimSpace(strings.Join(lines[1:len(lines)-1], "\n"))
		}
	}
	for _, prefix := range []string{"release notes:", "release notes："} {
		if len(message) >= len(prefix) && strings.EqualFold(message[:len(prefix)], prefix) {
			message = strings.TrimSpace(message[len(prefix):])
		}
	}
	if message == "" {
		return "", fmt.Errorf("AI returned an empty release message")
	}
	return message, nil
}
