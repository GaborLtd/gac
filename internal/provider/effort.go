package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// EffortProvider 是可選的 reasoning effort 擴充；不支援的 provider 會忽略 effort。
type EffortProvider interface {
	Provider
	GenerateWithEffort(ctx context.Context, model, effort, prompt string) (string, error)
}

// GenerateWithEffort 對 Codex 使用目前 CLI 支援的 --config model_reasoning_effort 設定；agy 與 Claude 維持原本 invocation。
func (p executable) GenerateWithEffort(ctx context.Context, model, effort, prompt string) (string, error) {
	args := p.args(model, prompt)
	if p.name == "codex" {
		args = modelArgsWithEffort([]string{"exec"}, model, effort, prompt)
	}
	out, stderr, err := p.runner.Run(ctx, p.command, args)
	if err != nil {
		if stderr != "" {
			return "", fmt.Errorf("%s failed: %s: %w", p.name, strings.TrimSpace(stderr), err)
		}
		return "", fmt.Errorf("%s failed: %w", p.name, err)
	}
	return out, nil
}

func modelArgsWithEffort(prefix []string, model, effort, prompt string) []string {
	args := append([]string{}, prefix...)
	if model != "" {
		args = append(args, "--model", model)
	}
	if effort != "" {
		args = append(args, "--config", "model_reasoning_effort="+strconv.Quote(effort))
	}
	return append(args, prompt)
}
