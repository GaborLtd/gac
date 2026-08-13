package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Config 是 gac 的持久化設定。
type Config struct {
	Provider        string `yaml:"provider"`
	Model           string `yaml:"model"`
	ReasoningEffort string `yaml:"reasoning_effort"`
	Language        string `yaml:"language"`
	DiffMaxBytes    int    `yaml:"diff_max_bytes"`
	DiffMaxLines    int    `yaml:"diff_max_lines"`
	PromptTemplate  string `yaml:"prompt_template"`
	TimeoutSeconds  int    `yaml:"timeout_seconds"`
	SkipCIMode      string `yaml:"skip_ci_mode"`
}

// Default 回傳安全且適合 commit message 的預設值。
func Default() Config {
	return Config{
		Language:       "en",
		DiffMaxBytes:   64 * 1024,
		DiffMaxLines:   1000,
		TimeoutSeconds: 120,
		SkipCIMode:     "ask",
	}
}

// Path 回傳 XDG config 位置；測試可直接使用 Load／Save 傳入自己的路徑。
func Path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "gac", "config.yaml"), nil
}

// Load 載入 YAML；檔案不存在時回傳預設設定。
func Load(path string) (Config, error) {
	cfg := Default()
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	if err := parseYAML(string(b), &cfg); err != nil {
		return cfg, err
	}
	if cfg.Language == "" {
		cfg.Language = "en"
	}
	if cfg.DiffMaxBytes <= 0 {
		cfg.DiffMaxBytes = Default().DiffMaxBytes
	}
	if cfg.DiffMaxLines <= 0 {
		cfg.DiffMaxLines = Default().DiffMaxLines
	}
	if cfg.TimeoutSeconds <= 0 {
		cfg.TimeoutSeconds = Default().TimeoutSeconds
	}
	if cfg.SkipCIMode == "" {
		cfg.SkipCIMode = "ask"
	}
	return cfg, nil
}

// Save 以 YAML 寫入設定檔。
func Save(path string, cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(formatYAML(cfg)), 0o600)
}

// parseYAML 支援本專案需要的 YAML subset：頂層 scalar 與 literal block。
// 設定檔刻意保持簡單，避免 CLI 為了少量設定引入大型 runtime dependency。
func parseYAML(input string, cfg *Config) error {
	lines := strings.Split(strings.ReplaceAll(input, "\r\n", "\n"), "\n")
	var blockKey string
	var block []string
	flushBlock := func() {
		if blockKey == "prompt_template" {
			if len(block) > 0 && block[len(block)-1] == "" {
				block = block[:len(block)-1]
			}
			cfg.PromptTemplate = strings.Join(block, "\n")
		}
		blockKey = ""
		block = nil
	}
	for _, line := range lines {
		if blockKey != "" {
			if strings.TrimSpace(line) == "" {
				block = append(block, "")
				continue
			}
			if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
				block = append(block, strings.TrimLeft(line, " \t"))
				continue
			}
			flushBlock()
		}
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			return fmt.Errorf("invalid YAML line: %q", line)
		}
		key, value = strings.TrimSpace(key), strings.TrimSpace(value)
		if value == "|" {
			blockKey = key
			continue
		}
		if err := setValue(cfg, key, unquote(value)); err != nil {
			return err
		}
	}
	flushBlock()
	return nil
}

func unquote(value string) string {
	if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
		if s, err := strconv.Unquote(value); err == nil {
			return s
		}
	}
	if len(value) >= 2 && value[0] == '\'' && value[len(value)-1] == '\'' {
		return strings.ReplaceAll(value[1:len(value)-1], "''", "'")
	}
	return value
}

func setValue(cfg *Config, key, value string) error {
	parseInt := func() (int, error) {
		var n int
		_, err := fmt.Sscanf(value, "%d", &n)
		return n, err
	}
	switch key {
	case "provider":
		cfg.Provider = value
	case "model":
		cfg.Model = value
	case "reasoning_effort":
		cfg.ReasoningEffort = value
	case "language":
		cfg.Language = value
	case "diff_max_bytes":
		n, err := parseInt()
		if err != nil {
			return fmt.Errorf("diff_max_bytes must be an integer")
		}
		cfg.DiffMaxBytes = n
	case "diff_max_lines":
		n, err := parseInt()
		if err != nil {
			return fmt.Errorf("diff_max_lines must be an integer")
		}
		cfg.DiffMaxLines = n
	case "timeout_seconds":
		n, err := parseInt()
		if err != nil {
			return fmt.Errorf("timeout_seconds must be an integer")
		}
		cfg.TimeoutSeconds = n
	case "skip_ci_mode":
		cfg.SkipCIMode = value
	case "prompt_template":
		cfg.PromptTemplate = value
	default:
		return fmt.Errorf("unsupported config field: %s", key)
	}
	return nil
}

func formatYAML(cfg Config) string {
	var b strings.Builder
	fmt.Fprintf(&b, "provider: %s\nmodel: %s\nreasoning_effort: %s\nlanguage: %s\ndiff_max_bytes: %d\ndiff_max_lines: %d\ntimeout_seconds: %d\nskip_ci_mode: %s\n", quote(cfg.Provider), quote(cfg.Model), quote(cfg.ReasoningEffort), quote(cfg.Language), cfg.DiffMaxBytes, cfg.DiffMaxLines, cfg.TimeoutSeconds, quote(cfg.SkipCIMode))
	if cfg.PromptTemplate != "" {
		b.WriteString("prompt_template: |\n")
		for _, line := range strings.Split(cfg.PromptTemplate, "\n") {
			b.WriteString("  ")
			b.WriteString(line)
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func quote(value string) string { return strconv.Quote(value) }
