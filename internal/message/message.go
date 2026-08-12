package message

import (
	"fmt"
	"regexp"
	"strings"
)

var conventional = regexp.MustCompile(`^[a-z][a-z0-9-]*(\([^)]+\))?!?:\s+\S+`)
var ciToken = regexp.MustCompile(`(?i)\[(skip ci|ci skip)\]`)

// Clean 清理常見的 AI 包裝並驗證 Conventional Commits subject。
func Clean(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	if strings.HasPrefix(s, "```") && strings.HasSuffix(s, "```") {
		lines := strings.Split(s, "\n")
		if len(lines) >= 2 {
			s = strings.TrimSpace(strings.Join(lines[1:len(lines)-1], "\n"))
		}
	}
	for _, prefix := range []string{"commit message:", "commit message："} {
		if len(s) >= len(prefix) && strings.EqualFold(s[:len(prefix)], prefix) {
			s = strings.TrimSpace(s[len(prefix):])
		}
	}
	if s == "" {
		return "", fmt.Errorf("AI returned an empty message")
	}
	if !conventional.MatchString(strings.SplitN(s, "\n", 2)[0]) {
		return "", fmt.Errorf("invalid Conventional Commits message: %q", strings.SplitN(s, "\n", 2)[0])
	}
	return s, nil
}

// HasCIToken 判斷 message 是否已有常見 CI skip token。
func HasCIToken(s string) bool { return ciToken.MatchString(s) }

// AppendCIToken 加入 canonical token，若已有等效 token 則保持原文。
func AppendCIToken(s string) string {
	if HasCIToken(s) {
		return s
	}
	return strings.TrimSpace(s) + " [skip ci]"
}
