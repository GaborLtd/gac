package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gaborltd/gac/internal/config"
	"github.com/gaborltd/gac/internal/diff"
	"github.com/gaborltd/gac/internal/git"
	"github.com/gaborltd/gac/internal/message"
	"github.com/gaborltd/gac/internal/prompt"
	"github.com/gaborltd/gac/internal/provider"
)

type application struct {
	in     io.Reader
	out    io.Writer
	err    io.Writer
	reader *bufio.Reader
}

var version = "dev"

func main() {
	app := application{in: os.Stdin, out: os.Stdout, err: os.Stderr}
	if err := app.run(os.Args[1:]); err != nil {
		fmt.Fprintln(app.err, "gac:", err)
		os.Exit(1)
	}
}

func (app *application) run(args []string) error {
	if len(args) > 0 {
		switch args[0] {
		case "version":
			fmt.Fprintln(app.out, "gac "+version)
			return nil
		case "providers":
			return app.providers()
		case "config":
			return app.configure(args[1:])
		case "release":
			return app.release(args[1:])
		}
	}

	fs := flag.NewFlagSet("gac", flag.ContinueOnError)
	fs.SetOutput(app.err)
	nonInteractive := false
	fs.BoolVar(&nonInteractive, "n", false, "print the message only; do not interact or commit")
	fs.BoolVar(&nonInteractive, "non-interactive", false, "print the message only; do not interact or commit")
	providerName := fs.String("provider", "", "provider name")
	modelName := fs.String("model", "", "model name")
	language := fs.String("language", "", "output language")
	skipCI := fs.Bool("skip-ci", false, "append [skip ci]")
	configPath := fs.String("config", "", "YAML config path")
	if err := fs.Parse(args); err != nil {
		return err
	}

	path := *configPath
	if path == "" {
		var err error
		path, err = config.Path()
		if err != nil {
			return fmt.Errorf("unable to determine config path: %w", err)
		}
	}
	cfg, err := config.Load(path)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	if *providerName != "" {
		switchProviderConfig(&cfg, *providerName)
	}
	if *modelName != "" {
		cfg.Model = *modelName
	}
	if *language != "" {
		cfg.Language = *language
	}
	onboarding := cfg.Provider == "" && !nonInteractive

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.TimeoutSeconds)*time.Second)
	defer cancel()
	repo, err := git.Discover(ctx, git.NewOSRunner(), cwd)
	if err != nil {
		return err
	}
	paths := fs.Args()
	if len(paths) > 1 {
		return errors.New("only one file path may be provided")
	}
	scope, err := repo.Scope(paths)
	if err != nil {
		return err
	}
	allTracked := len(paths) == 0
	if allTracked {
		scope = []string{"."}
	} else {
		filePath := paths[0]
		if !filepath.IsAbs(filePath) {
			filePath = filepath.Join(cwd, filePath)
		}
		info, statErr := os.Stat(filePath)
		if statErr == nil && info.IsDir() {
			return errors.New("directory paths are not supported; provide a file path")
		}
		if statErr != nil && !os.IsNotExist(statErr) {
			return fmt.Errorf("unable to inspect file path: %w", statErr)
		}
	}
	var stat, diffText, namesText string
	if allTracked {
		stat, err = repo.WorktreeStat(ctx, scope)
		if err == nil {
			diffText, err = repo.WorktreeDiff(ctx, scope)
		}
		if err == nil {
			namesText, err = repo.ChangedNames(ctx, scope)
		}
	} else {
		stat, err = repo.Stat(ctx, scope)
		if err == nil {
			diffText, err = repo.Diff(ctx, scope)
		}
		if err == nil {
			namesText, err = repo.StagedNames(ctx, scope)
		}
	}
	if err != nil {
		return err
	}
	if strings.TrimSpace(diffText) == "" {
		return errors.New("no tracked changes to commit")
	}
	commitNames := nonEmptyLines(namesText)
	if len(commitNames) == 0 {
		return errors.New("no tracked files to commit")
	}
	if !allTracked {
		unstaged, unstagedErr := repo.UnstagedNames(ctx, commitNames)
		if unstagedErr != nil {
			return unstagedErr
		}
		if strings.TrimSpace(unstaged) != "" {
			return fmt.Errorf("target files have unstaged changes; review them before continuing: %s", strings.TrimSpace(unstaged))
		}
	}

	limited := diff.Limit(diffText, cfg.DiffMaxBytes, cfg.DiffMaxLines)
	if limited.Truncated {
		limited.Text += "\n[diff truncated by gac limits]"
	}
	p, err := app.selectProvider(ctx, cfg.Provider, nonInteractive)
	if err != nil {
		return err
	}
	if cfg.Provider == "" && !nonInteractive {
		cfg.Provider = p.Name()
	}
	if cfg.Model == "" && !nonInteractive {
		cfg.Model, cfg.ReasoningEffort, err = app.chooseModelFromCatalog(ctx, p, cfg.Model, cfg.ReasoningEffort)
		if err != nil {
			return err
		}
	}
	if onboarding {
		fmt.Fprint(app.out, "Language [en]: ")
		selectedLanguage, readErr := app.readLine()
		if readErr != nil {
			return readErr
		}
		if selectedLanguage != "" {
			cfg.Language = selectedLanguage
		}
	}
	if cfg.Provider != "" && !nonInteractive {
		if err := config.Save(path, cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
	}

	generate := func(extra string) (string, error) {
		promptText, err := prompt.Build(prompt.Input{Language: cfg.Language, Stat: stat, Diff: limited.Text, Context: extra}, cfg.PromptTemplate)
		if err != nil {
			return "", fmt.Errorf("failed to build prompt: %w", err)
		}
		callCtx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.TimeoutSeconds)*time.Second)
		defer cancel()
		raw, err := generateWithEffort(callCtx, p, cfg.Model, cfg.ReasoningEffort, promptText)
		if err != nil {
			return "", err
		}
		return message.Clean(raw)
	}

	msg, err := generate("")
	if err != nil {
		return err
	}
	if *skipCI {
		msg = message.AppendCIToken(msg)
	}
	if nonInteractive {
		_, err = fmt.Fprintln(app.out, msg)
		return err
	}
	commitCtx, commitCancel := context.WithTimeout(context.Background(), time.Duration(cfg.TimeoutSeconds)*time.Second)
	defer commitCancel()
	return app.interactiveCommit(msg, generate, repo, commitNames, allTracked, commitCtx)
}

func switchProviderConfig(cfg *config.Config, providerName string) {
	if cfg.Provider != providerName {
		cfg.Model = ""
		cfg.ReasoningEffort = ""
	}
	cfg.Provider = providerName
}

func generateWithEffort(ctx context.Context, p provider.Provider, model, effort, promptText string) (string, error) {
	if ep, ok := p.(provider.EffortProvider); ok {
		return ep.GenerateWithEffort(ctx, model, effort, promptText)
	}
	return p.Generate(ctx, model, promptText)
}

func (app *application) selectProvider(ctx context.Context, requested string, nonInteractive bool) (provider.Provider, error) {
	available := provider.DetectAll(provider.NewOSRunner())
	if requested != "" {
		for _, p := range available {
			if p.Name() == requested {
				return p, nil
			}
		}
		return nil, fmt.Errorf("provider is unavailable or not installed: %s", requested)
	}
	if len(available) == 0 {
		return nil, errors.New("no usable agy, codex, or claude CLI found")
	}
	if nonInteractive {
		return nil, errors.New("non-interactive mode requires a configured provider")
	}
	if len(available) == 1 {
		fmt.Fprintf(app.out, "Using provider: %s\n", available[0].Name())
		return available[0], nil
	}
	fmt.Fprintln(app.out, "Available providers:")
	for i, p := range available {
		fmt.Fprintf(app.out, "%d) %s\n", i+1, p.Name())
	}
	fmt.Fprint(app.out, "Select provider [1]: ")
	choice, err := app.readLine()
	if err != nil {
		return nil, err
	}
	if choice == "" {
		choice = "1"
	}
	var index int
	if _, err := fmt.Sscanf(choice, "%d", &index); err != nil || index < 1 || index > len(available) {
		return nil, errors.New("invalid provider selection")
	}
	return available[index-1], nil
}

func (app *application) interactiveCommit(msg string, generate func(string) (string, error), repo git.Repo, commitNames []string, allTracked bool, ctx context.Context) error {
	extra := ""
	for {
		fmt.Fprintf(app.out, "\n📝 Suggested message:\n%s\n", msg)
		fmt.Fprintln(app.out, "[y] commit  [e] edit  [a] add context  [s] add [skip ci]  [q] cancel")
		fmt.Fprint(app.out, "Choice: ")
		choice, err := app.readLine()
		if err != nil {
			return err
		}
		switch strings.ToLower(choice) {
		case "y", "yes":
			if allTracked {
				if err := repo.CommitAll(ctx, msg); err != nil {
					return err
				}
			} else if err := repo.Commit(ctx, msg, commitNames); err != nil {
				return err
			}
			fmt.Fprintln(app.out, "Commit created")
			return nil
		case "e", "edit":
			msg, err = editMessage(msg, app.in, app.out, app.err)
			if err != nil {
				return err
			}
		case "a", "add":
			fmt.Fprint(app.out, "Additional context: ")
			addition, err := app.readLine()
			if err != nil {
				return err
			}
			extra = strings.TrimSpace(strings.TrimSpace(extra) + "\n" + addition)
			msg, err = generate(extra)
			if err != nil {
				return err
			}
		case "s", "skip":
			msg = message.AppendCIToken(msg)
		case "q", "quit", "cancel", "":
			fmt.Fprintln(app.out, "Cancelled")
			return nil
		default:
			fmt.Fprintln(app.out, "Invalid choice")
		}
	}
}

func (app *application) providers() error {
	available := provider.DetectAll(provider.NewOSRunner())
	if len(available) == 0 {
		return errors.New("no usable agy, codex, or claude CLI found")
	}
	for _, p := range available {
		fmt.Fprintln(app.out, p.Name())
	}
	return nil
}

func (app *application) configure(args []string) error {
	fs := flag.NewFlagSet("gac config", flag.ContinueOnError)
	fs.SetOutput(app.err)
	configPath := fs.String("config", "", "YAML config path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	path := *configPath
	if path == "" {
		var err error
		path, err = config.Path()
		if err != nil {
			return err
		}
	}
	cfg, err := config.Load(path)
	if err != nil {
		return err
	}
	available := provider.DetectAll(provider.NewOSRunner())
	if len(available) == 0 {
		return errors.New("no usable agy, codex, or claude CLI found")
	}
	fmt.Fprintln(app.out, "Available providers:")
	for i, p := range available {
		fmt.Fprintf(app.out, "%d) %s\n", i+1, p.Name())
	}
	fmt.Fprint(app.out, "Provider [1]: ")
	choice, err := app.readLine()
	if err != nil {
		return err
	}
	if choice == "" {
		choice = "1"
	}
	var index int
	if _, err := fmt.Sscanf(choice, "%d", &index); err != nil || index < 1 || index > len(available) {
		return errors.New("invalid provider selection")
	}
	selectedProvider := available[index-1]
	switchProviderConfig(&cfg, selectedProvider.Name())
	modelCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	cfg.Model, cfg.ReasoningEffort, err = app.chooseModelFromCatalog(modelCtx, selectedProvider, cfg.Model, cfg.ReasoningEffort)
	cancel()
	if err != nil {
		return err
	}
	fmt.Fprintf(app.out, "Language [%s]: ", cfg.Language)
	language, err := app.readLine()
	if err != nil {
		return err
	}
	if language != "" {
		cfg.Language = language
	}
	if err := config.Save(path, cfg); err != nil {
		return err
	}
	fmt.Fprintln(app.out, "Saved:", path)
	return nil
}

func (app *application) chooseModel(ctx context.Context, p provider.Provider, current string) (string, error) {
	model, _, err := app.chooseModelSelection(ctx, p, current, "")
	return model, err
}

func (app *application) chooseModelSelection(ctx context.Context, p provider.Provider, current, currentEffort string) (string, string, error) {
	models, listErr := p.ListModels(ctx)
	currentValue, selectedEffort := normalizeCurrentSelection(current, currentEffort, models)
	if listErr == nil && len(models) > 0 {
		sort.SliceStable(models, func(i, j int) bool {
			return modelCostRank(models[i].ID+" "+models[i].DisplayName) < modelCostRank(models[j].ID+" "+models[j].DisplayName)
		})
		fmt.Fprintln(app.out, "Recommendation: use the cheapest model that is sufficient for a commit message.")
		fmt.Fprintln(app.out, "Available models:")
		for i, model := range models {
			fmt.Fprintf(app.out, "%d) %s%s\n", i+1, displayModelLabel(model), modelCostHint(model.ID+" "+model.DisplayName))
		}
		fmt.Fprintln(app.out, "0) Enter a model name manually")
	}
	if listErr != nil {
		fmt.Fprintln(app.out, "This provider CLI cannot list account-specific models.")
		if p.LoginHint() != "" {
			fmt.Fprintf(app.out, "If you have not signed in yet, run: %s\n", p.LoginHint())
		}
		if p.DocsURL() != "" {
			fmt.Fprintf(app.out, "Model documentation: %s\n", p.DocsURL())
		}
	}
	defaultLabel := currentValue
	if defaultLabel == "" {
		defaultLabel = "provider default"
	}
	if selectedEffort != "" {
		defaultLabel += " (effort: " + selectedEffort + ")"
	}
	if listErr == nil && len(models) > 0 {
		fmt.Fprintf(app.out, "Model [%s] (press Enter for the provider default, enter a number or model name): ", defaultLabel)
	} else {
		fmt.Fprintf(app.out, "Model [%s] (press Enter for the provider default, or enter a model name): ", defaultLabel)
	}
	choice, err := app.readLine()
	if err != nil {
		return "", "", err
	}
	if choice == "" {
		return currentValue, selectedEffort, nil
	}
	if listErr == nil && len(models) > 0 {
		var index int
		if _, scanErr := fmt.Sscanf(choice, "%d", &index); scanErr == nil {
			if index == 0 {
				fmt.Fprint(app.out, "Model name: ")
				model, readErr := app.readLine()
				return model, selectedEffort, readErr
			}
			if index >= 1 && index <= len(models) {
				selected := models[index-1]
				return selected.Value(), selected.ReasoningEffort, nil
			}
			return "", "", errors.New("invalid model number")
		}
	}
	return choice, selectedEffort, nil
}

func displayModelLabel(model provider.Model) string {
	label := model.Label()
	if model.ReasoningEffort != "" {
		label += " (effort: " + model.ReasoningEffort + ")"
	}
	return label
}

func normalizeCurrentSelection(current, effort string, models []provider.Model) (string, string) {
	value := normalizeCurrentModel(current, models)
	if value == "" && len(models) == 0 {
		value = strings.TrimSpace(current)
	}
	if strings.TrimSpace(effort) == "" {
		for _, model := range models {
			if model.Value() == value {
				effort = model.ReasoningEffort
				break
			}
		}
	}
	return value, effort
}

func normalizeCurrentModel(current string, models []provider.Model) string {
	current = strings.TrimSpace(current)
	if current == "" {
		return ""
	}
	for _, model := range models {
		if current == model.ID || current == model.DisplayName {
			return model.Value()
		}
		fields := strings.Fields(current)
		if len(fields) > 0 && fields[0] == model.ID {
			return model.Value()
		}
	}
	return ""
}

func modelCostRank(model string) int {
	name := strings.ToLower(model)
	switch {
	case strings.Contains(name, "low"), strings.Contains(name, "cheap"), strings.Contains(name, "mini"), strings.Contains(name, "nano"), strings.Contains(name, "haiku"):
		return 0
	case strings.Contains(name, "medium"), strings.Contains(name, "sonnet"), strings.Contains(name, "flash"):
		return 1
	case strings.Contains(name, "high"), strings.Contains(name, "opus"), strings.Contains(name, "pro"):
		return 2
	default:
		return 1
	}
}

func modelCostHint(model string) string {
	if strings.Contains(strings.ToLower(model), "low-cost") {
		return ""
	}
	if modelCostRank(model) == 0 {
		return " (low-cost recommendation)"
	}
	return ""
}

func editMessage(msg string, in io.Reader, out, errOut io.Writer) (string, error) {
	file, err := os.CreateTemp("", "gac-message-*.txt")
	if err != nil {
		return "", err
	}
	path := file.Name()
	defer os.Remove(path)
	if _, err := file.WriteString(msg + "\n"); err != nil {
		file.Close()
		return "", err
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	editor := os.Getenv("GIT_EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}
	if editor == "" {
		editor = "vi"
	}
	cmd := exec.Command(editor, path)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = in, out, errOut
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("editor failed: %w", err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return message.Clean(string(b))
}

func (app *application) readLine() (string, error) {
	if app.reader == nil {
		app.reader = bufio.NewReader(app.in)
	}
	line, err := app.reader.ReadString('\n')
	line = strings.TrimSpace(line)
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	if errors.Is(err, io.EOF) && line == "" {
		return "", io.EOF
	}
	return line, nil
}

func nonEmptyLines(s string) []string {
	var lines []string
	for _, line := range strings.Split(s, "\n") {
		if strings.TrimSpace(line) != "" {
			lines = append(lines, strings.TrimSpace(line))
		}
	}
	return lines
}
