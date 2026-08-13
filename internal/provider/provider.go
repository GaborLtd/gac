package provider

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// Runner 讓 provider 可以用 fake command 做離線測試。
type Runner interface {
	Run(ctx context.Context, name string, args []string) (stdout, stderr string, err error)
}

type osRunner struct{}

// NewOSRunner 建立實際執行 provider CLI 的 runner。
func NewOSRunner() Runner { return osRunner{} }

func (osRunner) Run(ctx context.Context, name string, args []string) (string, string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// Provider 是 AI CLI adapter 的最小介面。
type Model struct {
	ID              string
	DisplayName     string
	ProviderValue   string
	ReasoningEffort string
}

func (m Model) Label() string {
	if m.DisplayName == "" || m.DisplayName == m.ID {
		return m.ID
	}
	return m.ID + " — " + m.DisplayName
}

func (m Model) Value() string {
	if m.ProviderValue != "" {
		return m.ProviderValue
	}
	if m.DisplayName != "" {
		return m.DisplayName
	}
	return m.ID
}

type Provider interface {
	Name() string
	Detect() error
	ListModels(ctx context.Context) ([]Model, error)
	DocsURL() string
	LoginHint() string
	Generate(ctx context.Context, model, prompt string) (string, error)
}

type executable struct {
	name      string
	command   string
	args      func(model, prompt string) []string
	listArgs  func() []string
	docsURL   string
	loginHint string
	runner    Runner
}

func (p executable) Name() string      { return p.name }
func (p executable) DocsURL() string   { return p.docsURL }
func (p executable) LoginHint() string { return p.loginHint }

func (p executable) ListModels(ctx context.Context) ([]Model, error) {
	if p.listArgs == nil {
		return nil, fmt.Errorf("%s CLI has no reliable command for listing account-specific models", p.name)
	}
	out, stderr, err := p.runner.Run(ctx, p.command, p.listArgs())
	if err != nil {
		if stderr != "" {
			return nil, fmt.Errorf("%s model listing failed: %s: %w", p.name, stderr, err)
		}
		return nil, fmt.Errorf("%s model listing failed: %w", p.name, err)
	}
	models := parseModelList(out)
	if len(models) == 0 {
		return nil, fmt.Errorf("%s returned no recognizable models", p.name)
	}
	return models, nil
}

func (p executable) Detect() error {
	if _, err := exec.LookPath(p.command); err != nil {
		return fmt.Errorf("%s is unavailable: %w", p.name, err)
	}
	return nil
}

func (p executable) Generate(ctx context.Context, model, prompt string) (string, error) {
	out, stderr, err := p.runner.Run(ctx, p.command, p.args(model, prompt))
	if err != nil {
		if stderr != "" {
			return "", fmt.Errorf("%s failed: %s: %w", p.name, stderr, err)
		}
		return "", fmt.Errorf("%s failed: %w", p.name, err)
	}
	return out, nil
}

func modelArgs(prefix []string, model, prompt string) []string {
	args := append([]string{}, prefix...)
	if model != "" {
		args = append(args, "--model", model)
	}
	return append(args, prompt)
}

// NewAgy 建立符合既有 shell 用法的 agy adapter。
func NewAgy(r Runner) Provider {
	return executable{name: "agy", command: "agy", listArgs: func() []string {
		return []string{"models"}
	}, docsURL: "https://www.antigravity.google/docs/cli-using", loginHint: "run agy, complete browser sign-in, then retry", runner: r, args: func(model, prompt string) []string {
		args := []string{}
		if model != "" {
			model = strings.TrimSpace(model)
			if strings.Contains(model, "\t") || strings.Contains(model, "  ") {
				model = parseModelLine(model).Value()
			}
			args = append(args, "--model", model)
		}
		return append(args, "--print", prompt)
	}}
}

// NewCodex 建立 codex exec adapter。
func NewCodex(r Runner) Provider {
	return executable{name: "codex", command: "codex", docsURL: "https://github.com/openai/codex", loginHint: "codex login", runner: r, args: func(model, prompt string) []string {
		return modelArgs([]string{"exec"}, model, prompt)
	}}
}

// NewClaude 建立 Claude Code print adapter。
func NewClaude(r Runner) Provider {
	return executable{name: "claude", command: "claude", docsURL: "https://docs.anthropic.com/en/docs/claude-code/cli-usage", loginHint: "run claude and complete sign-in", runner: r, args: func(model, prompt string) []string {
		args := []string{"-p", prompt}
		if model != "" {
			args = append(args, "--model", model)
		}
		return args
	}}
}

func parseModelList(raw string) []Model {
	var models []Model
	seen := make(map[string]bool)
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		line = strings.TrimLeft(line, "*-• ")
		line = strings.TrimSpace(line)
		if line == "" || strings.EqualFold(line, "available models:") || strings.EqualFold(line, "models:") {
			continue
		}
		model := parseModelLine(line)
		if model.ID != "" && !seen[model.ID] {
			models = append(models, model)
			seen[model.ID] = true
		}
	}
	return models
}

func parseModelLine(line string) Model {
	if fields := strings.Fields(line); len(fields) > 1 && looksLikeModelID(fields[0]) {
		return Model{ID: fields[0], DisplayName: strings.Join(fields[1:], " ")}
	}
	return Model{ID: line}
}

func looksLikeModelID(value string) bool {
	return strings.ContainsAny(value, "-_: /") || strings.ContainsAny(value, "0123456789")
}

// DetectAll 回傳 PATH 中可用的第一批 provider。
func DetectAll(r Runner) []Provider {
	all := []Provider{NewAgy(r), NewCodex(r), NewClaude(r)}
	available := make([]Provider, 0, len(all))
	for _, p := range all {
		if p.Detect() == nil {
			available = append(available, p)
		}
	}
	return available
}
