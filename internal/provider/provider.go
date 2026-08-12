package provider

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
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
type Provider interface {
	Name() string
	Detect() error
	Generate(ctx context.Context, model, prompt string) (string, error)
}

type executable struct {
	name    string
	command string
	args    func(model, prompt string) []string
	runner  Runner
}

func (p executable) Name() string { return p.name }

func (p executable) Detect() error {
	if _, err := exec.LookPath(p.command); err != nil {
		return fmt.Errorf("%s 不可用：%w", p.name, err)
	}
	return nil
}

func (p executable) Generate(ctx context.Context, model, prompt string) (string, error) {
	out, stderr, err := p.runner.Run(ctx, p.command, p.args(model, prompt))
	if err != nil {
		if stderr != "" {
			return "", fmt.Errorf("%s 失敗：%s: %w", p.name, stderr, err)
		}
		return "", fmt.Errorf("%s 失敗：%w", p.name, err)
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
	return executable{name: "agy", command: "agy", runner: r, args: func(model, prompt string) []string {
		args := []string{}
		if model != "" {
			args = append(args, "--model", model)
		}
		return append(args, "--print", prompt)
	}}
}

// NewCodex 建立 codex exec adapter。
func NewCodex(r Runner) Provider {
	return executable{name: "codex", command: "codex", runner: r, args: func(model, prompt string) []string {
		return modelArgs([]string{"exec"}, model, prompt)
	}}
}

// NewClaude 建立 Claude Code print adapter。
func NewClaude(r Runner) Provider {
	return executable{name: "claude", command: "claude", runner: r, args: func(model, prompt string) []string {
		args := []string{"-p", prompt}
		if model != "" {
			args = append(args, "--model", model)
		}
		return args
	}}
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
