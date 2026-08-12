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
		cfg.Provider = *providerName
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
	scope, err := repo.Scope(fs.Args())
	if err != nil {
		return err
	}
	allTracked := len(fs.Args()) == 0
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
		cfg.Model, err = app.chooseModel(ctx, p, cfg.Model)
		if err != nil {
			return err
		}
	}
	if onboarding {
		fmt.Fprint(app.out, "Language [en]：")
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
		raw, err := p.Generate(callCtx, cfg.Model, promptText)
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
	return app.interactiveCommit(msg, generate, repo, commitNames, scope, allTracked, commitCtx)
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

func (app *application) interactiveCommit(msg string, generate func(string) (string, error), repo git.Repo, commitNames, scope []string, allTracked bool, ctx context.Context) error {
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
				if err := repo.CommitAll(ctx, msg, scope); err != nil {
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
	cfg.Provider = selectedProvider.Name()
	modelCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	cfg.Model, err = app.chooseModel(modelCtx, selectedProvider, cfg.Model)
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
	models, listErr := p.ListModels(ctx)
	if listErr == nil && len(models) > 0 {
		sort.SliceStable(models, func(i, j int) bool {
			return modelCostRank(models[i]) < modelCostRank(models[j])
		})
		fmt.Fprintln(app.out, "Recommendation: use the cheapest model that is sufficient for a commit message.")
		fmt.Fprintln(app.out, "Available models:")
		for i, model := range models {
			fmt.Fprintf(app.out, "%d) %s%s\n", i+1, model, modelCostHint(model))
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
	defaultLabel := current
	if defaultLabel == "" {
		defaultLabel = "provider default"
	}
	if listErr == nil && len(models) > 0 {
		fmt.Fprintf(app.out, "Model [%s] (press Enter for the provider default, enter a number or model name): ", defaultLabel)
	} else {
		fmt.Fprintf(app.out, "Model [%s] (press Enter for the provider default, or enter a model name): ", defaultLabel)
	}
	choice, err := app.readLine()
	if err != nil {
		return "", err
	}
	if choice == "" {
		return current, nil
	}
	if listErr == nil && len(models) > 0 {
		var index int
		if _, scanErr := fmt.Sscanf(choice, "%d", &index); scanErr == nil {
			if index == 0 {
				fmt.Fprint(app.out, "Model name: ")
				return app.readLine()
			}
			if index >= 1 && index <= len(models) {
				return models[index-1], nil
			}
			return "", errors.New("invalid model number")
		}
	}
	return choice, nil
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
