package main

import (
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
	"github.com/gaborltd/gac/internal/git"
	releasepkg "github.com/gaborltd/gac/internal/release"
)

func (app *application) release(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: gac release preview|tag vMAJOR.MINOR.PATCH")
	}
	mode := args[0]
	if mode != "preview" && mode != "tag" {
		return fmt.Errorf("unknown release action %q; use preview or tag", mode)
	}

	fs := flag.NewFlagSet("gac release "+mode, flag.ContinueOnError)
	fs.SetOutput(app.err)
	configPath := fs.String("config", "", "YAML config path")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if len(fs.Args()) != 1 {
		return fmt.Errorf("usage: gac release %s vMAJOR.MINOR.PATCH", mode)
	}
	version := fs.Arg(0)
	if err := releasepkg.ValidateVersion(version); err != nil {
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

	previous, err := repo.LatestReleaseTag(ctx)
	if err != nil {
		return err
	}
	if previous != "" {
		comparison, compareErr := releasepkg.CompareVersion(version, previous)
		if compareErr != nil {
			return compareErr
		}
		if comparison <= 0 {
			return fmt.Errorf("version %s must be newer than previous release %s", version, previous)
		}
	}
	exists, err := repo.TagExists(ctx, version)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("tag already exists: %s", version)
	}
	commitLines, err := repo.ReleaseCommits(ctx, previous)
	if err != nil {
		return err
	}
	plan := releasepkg.Plan{Version: version, PreviousTag: previous, Commits: parseReleaseCommits(commitLines)}
	if len(plan.Commits) == 0 {
		return errors.New("no commits found since the previous release")
	}

	p, err := app.selectProvider(ctx, cfg.Provider, false)
	if err != nil {
		return err
	}
	if cfg.Provider == "" {
		cfg.Provider = p.Name()
	}
	if cfg.Model == "" {
		cfg.Model, cfg.ReasoningEffort, err = app.chooseModelFromCatalog(ctx, p, cfg.Model, cfg.ReasoningEffort)
		if err != nil {
			return err
		}
	}
	if err := config.Save(path, cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	promptText := releasepkg.BuildPrompt(plan, cfg.Language)
	callCtx, callCancel := context.WithTimeout(context.Background(), time.Duration(cfg.TimeoutSeconds)*time.Second)
	defer callCancel()
	raw, err := generateWithEffort(callCtx, p, cfg.Model, cfg.ReasoningEffort, promptText)
	if err != nil {
		return err
	}
	notes, err := releasepkg.CleanMessage(raw)
	if err != nil {
		return err
	}
	if mode == "preview" {
		fmt.Fprintf(app.out, "Release preview: %s\nPrevious release: %s\nCommits: %d\n\n%s\n", version, displayRelease(previous), len(plan.Commits), notes)
		return nil
	}
	return app.confirmReleaseTag(ctx, repo, version, notes)
}

func parseReleaseCommits(lines []string) []releasepkg.Commit {
	commits := make([]releasepkg.Commit, 0, len(lines))
	for _, line := range lines {
		fields := strings.SplitN(line, "\t", 2)
		if len(fields) == 2 {
			commits = append(commits, releasepkg.Commit{Hash: fields[0], Subject: fields[1]})
		} else if strings.TrimSpace(line) != "" {
			commits = append(commits, releasepkg.Commit{Subject: strings.TrimSpace(line)})
		}
	}
	return commits
}

func displayRelease(tag string) string {
	if tag == "" {
		return "none"
	}
	return tag
}

func (app *application) confirmReleaseTag(ctx context.Context, repo git.Repo, version, notes string) error {
	for {
		fmt.Fprintf(app.out, "\nRelease message for %s:\n%s\n", version, notes)
		fmt.Fprintln(app.out, "[y] create local tag  [e] edit  [q] cancel")
		fmt.Fprint(app.out, "Choice: ")
		choice, err := app.readLine()
		if err != nil {
			return err
		}
		switch strings.ToLower(choice) {
		case "y", "yes":
			status, err := repo.WorktreeStatus(ctx)
			if err != nil {
				return err
			}
			if strings.TrimSpace(status) != "" {
				return errors.New("worktree is not clean; commit or stash changes before creating a release tag")
			}
			if err := repo.CreateAnnotatedTag(ctx, version, notes); err != nil {
				return err
			}
			fmt.Fprintf(app.out, "Created local tag %s. Push it when ready: git push origin %s\n", version, version)
			return nil
		case "e", "edit":
			notes, err = editText(notes, app.in, app.out, app.err)
			if err != nil {
				return err
			}
		case "q", "quit", "cancel", "":
			fmt.Fprintln(app.out, "Cancelled")
			return nil
		default:
			fmt.Fprintln(app.out, "Invalid choice")
		}
	}
}

func editText(text string, in io.Reader, out, errOut io.Writer) (string, error) {
	file, err := os.CreateTemp("", "gac-release-*.md")
	if err != nil {
		return "", err
	}
	path := file.Name()
	defer os.Remove(path)
	if _, err := file.WriteString(text + "\n"); err != nil {
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
	return releasepkg.CleanMessage(string(b))
}
