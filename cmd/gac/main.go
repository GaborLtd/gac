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
	fs.BoolVar(&nonInteractive, "n", false, "只輸出 message，不互動也不 commit")
	fs.BoolVar(&nonInteractive, "non-interactive", false, "只輸出 message，不互動也不 commit")
	providerName := fs.String("provider", "", "provider 名稱")
	modelName := fs.String("model", "", "model 名稱")
	language := fs.String("language", "", "輸出語言")
	skipCI := fs.Bool("skip-ci", false, "加入 [skip ci]")
	configPath := fs.String("config", "", "YAML config 路徑")
	if err := fs.Parse(args); err != nil {
		return err
	}

	path := *configPath
	if path == "" {
		var err error
		path, err = config.Path()
		if err != nil {
			return fmt.Errorf("找不到 config 路徑：%w", err)
		}
	}
	cfg, err := config.Load(path)
	if err != nil {
		return fmt.Errorf("讀取 config 失敗：%w", err)
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
	stat, err := repo.Stat(ctx, scope)
	if err != nil {
		return err
	}
	diffText, err := repo.Diff(ctx, scope)
	if err != nil {
		return err
	}
	if strings.TrimSpace(diffText) == "" {
		return errors.New("沒有可 commit 的 staged 變更")
	}
	stagedNamesText, err := repo.StagedNames(ctx, scope)
	if err != nil {
		return err
	}
	stagedNames := nonEmptyLines(stagedNamesText)
	if len(stagedNames) == 0 {
		return errors.New("沒有可 commit 的 staged 檔案")
	}
	unstaged, err := repo.UnstagedNames(ctx, stagedNames)
	if err != nil {
		return err
	}
	if strings.TrimSpace(unstaged) != "" {
		return fmt.Errorf("目標檔案有未 staged 變更，為避免誤 commit 請先整理：%s", strings.TrimSpace(unstaged))
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
		fmt.Fprint(app.out, "Model（可留空使用 provider 預設）：")
		cfg.Model, err = app.readLine()
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
			return fmt.Errorf("儲存 config 失敗：%w", err)
		}
	}

	generate := func(extra string) (string, error) {
		promptText, err := prompt.Build(prompt.Input{Language: cfg.Language, Stat: stat, Diff: limited.Text, Context: extra}, cfg.PromptTemplate)
		if err != nil {
			return "", fmt.Errorf("建立 prompt 失敗：%w", err)
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
	return app.interactiveCommit(msg, generate, repo, stagedNames, commitCtx)
}

func (app *application) selectProvider(ctx context.Context, requested string, nonInteractive bool) (provider.Provider, error) {
	available := provider.DetectAll(provider.NewOSRunner())
	if requested != "" {
		for _, p := range available {
			if p.Name() == requested {
				return p, nil
			}
		}
		return nil, fmt.Errorf("provider 不可用或未安裝：%s", requested)
	}
	if len(available) == 0 {
		return nil, errors.New("找不到可用的 agy、codex 或 claude CLI")
	}
	if nonInteractive {
		return nil, errors.New("non-interactive 模式需要先在 config 設定 provider")
	}
	if len(available) == 1 {
		fmt.Fprintf(app.out, "使用 provider：%s\n", available[0].Name())
		return available[0], nil
	}
	fmt.Fprintln(app.out, "偵測到可用 provider：")
	for i, p := range available {
		fmt.Fprintf(app.out, "%d) %s\n", i+1, p.Name())
	}
	fmt.Fprint(app.out, "請選擇 provider [1]：")
	choice, err := app.readLine()
	if err != nil {
		return nil, err
	}
	if choice == "" {
		choice = "1"
	}
	var index int
	if _, err := fmt.Sscanf(choice, "%d", &index); err != nil || index < 1 || index > len(available) {
		return nil, errors.New("無效的 provider 選擇")
	}
	return available[index-1], nil
}

func (app *application) interactiveCommit(msg string, generate func(string) (string, error), repo git.Repo, stagedNames []string, ctx context.Context) error {
	extra := ""
	for {
		fmt.Fprintf(app.out, "\n📝 建議訊息：\n%s\n", msg)
		fmt.Fprintln(app.out, "[y] commit  [e] 編輯  [a] 補充脈絡  [s] 加入 [skip ci]  [q] 取消")
		fmt.Fprint(app.out, "選擇：")
		choice, err := app.readLine()
		if err != nil {
			return err
		}
		switch strings.ToLower(choice) {
		case "y", "yes":
			if err := repo.Commit(ctx, msg, stagedNames); err != nil {
				return err
			}
			fmt.Fprintln(app.out, "已建立 commit")
			return nil
		case "e", "edit":
			msg, err = editMessage(msg, app.in, app.out, app.err)
			if err != nil {
				return err
			}
		case "a", "add":
			fmt.Fprint(app.out, "請輸入補充脈絡：")
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
			fmt.Fprintln(app.out, "已取消")
			return nil
		default:
			fmt.Fprintln(app.out, "無效選擇")
		}
	}
}

func (app *application) providers() error {
	available := provider.DetectAll(provider.NewOSRunner())
	if len(available) == 0 {
		return errors.New("找不到可用的 agy、codex 或 claude CLI")
	}
	for _, p := range available {
		fmt.Fprintln(app.out, p.Name())
	}
	return nil
}

func (app *application) configure(args []string) error {
	fs := flag.NewFlagSet("gac config", flag.ContinueOnError)
	fs.SetOutput(app.err)
	configPath := fs.String("config", "", "YAML config 路徑")
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
		return errors.New("找不到可用的 agy、codex 或 claude CLI")
	}
	fmt.Fprintln(app.out, "可用 provider：")
	for i, p := range available {
		fmt.Fprintf(app.out, "%d) %s\n", i+1, p.Name())
	}
	fmt.Fprint(app.out, "Provider [1]：")
	choice, err := app.readLine()
	if err != nil {
		return err
	}
	if choice == "" {
		choice = "1"
	}
	var index int
	if _, err := fmt.Sscanf(choice, "%d", &index); err != nil || index < 1 || index > len(available) {
		return errors.New("無效的 provider 選擇")
	}
	cfg.Provider = available[index-1].Name()
	fmt.Fprintf(app.out, "Model [%s]：", cfg.Model)
	model, err := app.readLine()
	if err != nil {
		return err
	}
	if model != "" {
		cfg.Model = model
	}
	fmt.Fprintf(app.out, "Language [%s]：", cfg.Language)
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
	fmt.Fprintln(app.out, "已儲存：", path)
	return nil
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
		return "", fmt.Errorf("editor 失敗：%w", err)
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
