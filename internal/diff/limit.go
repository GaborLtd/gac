package diff

import "strings"

// Result 是受限制後的 diff。
type Result struct {
	Text      string
	Truncated bool
}

// Limit 同時套用 byte 與 line 上限，先達到的限制生效。
// 只在完整 UTF-8 rune 邊界截斷，避免送出無效文字。
func Limit(input string, maxBytes, maxLines int) Result {
	if input == "" {
		return Result{}
	}
	if maxBytes <= 0 {
		maxBytes = len(input)
	}
	if maxLines <= 0 {
		maxLines = strings.Count(input, "\n") + 1
	}

	var b strings.Builder
	lines := 0
	for _, line := range strings.SplitAfter(input, "\n") {
		if line == "" {
			continue
		}
		if lines >= maxLines {
			return Result{Text: b.String(), Truncated: true}
		}
		if b.Len()+len(line) > maxBytes {
			return Result{Text: b.String(), Truncated: true}
		}
		b.WriteString(line)
		lines++
	}
	return Result{Text: b.String(), Truncated: len(b.String()) < len(input)}
}
