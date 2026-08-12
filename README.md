# gac — Generate AI Commit message

gac is a Go CLI that turns staged Git changes into a Conventional Commits message with the help of an AI CLI.

[繁體中文](README.zh-TW.md)

## Features

- Uses only changes already staged in Git's index.
- Accepts a file or directory path to limit the scope.
- Detects agy, codex, and claude CLIs.
- Supports provider, model, language, and additional context selection.
- Supports interactive review and non-interactive output.
- Supports [skip ci] and [ci skip] detection.
- Never runs git add, git add -A, git commit -a, or git push.

## Installation

After a GitHub Release:

~~~sh
curl -fsSL https://raw.githubusercontent.com/GaborLtd/gac/main/install.sh | sh
~~~

The installer detects macOS or Linux and amd64 or arm64, verifies SHA256, and installs to $HOME/.local/bin by default. Install a specific version with:

~~~sh
GAC_VERSION=0.1.0 curl -fsSL https://raw.githubusercontent.com/GaborLtd/gac/main/install.sh | sh
~~~

GAC_INSTALL_DIR and GAC_REPOSITORY can also be set. The installer does not use sudo.

## Quick start

Stage files yourself, then run gac:

~~~sh
git add path/to/file.go path/to/another-file.go
gac
~~~

Review the generated message and enter y to create the commit.

Without a path, gac uses staged changes under the current working directory. With a path, it uses only staged changes matching that file or directory:

~~~sh
gac
gac src/
gac src/main.go
~~~

Unstaged changes are never sent to the AI. If a target file has both staged and unstaged changes, gac stops to prevent an accidental partial commit.

## Interactive controls

- y / yes: create the commit.
- e / edit: open the message in GIT_EDITOR, VISUAL, or EDITOR.
- a / add: enter context and generate the message again.
- s / skip: append [skip ci] unless [skip ci] or [ci skip] already exists.
- q / cancel: cancel without creating a commit.

The default output language is English. Change it during onboarding or in the configuration.

## Non-interactive mode

Use -n as the short form of --non-interactive. It writes only the final message to stdout, does not create a commit, and is suitable for pipes:

~~~sh
gac -n | pbcopy
gac -n > commit-message.txt
gac -n --skip-ci | git commit -F -
~~~

Diagnostics go to stderr. Configure a provider first or pass one explicitly:

~~~sh
gac config
gac -n --provider agy --model low-cost-model
~~~

## Providers and models

The first release supports:

| Provider | Detection | Invocation |
| --- | --- | --- |
| agy | agy in PATH | agy --model MODEL --print PROMPT |
| codex | codex in PATH | codex exec --model MODEL PROMPT |
| claude | claude in PATH | claude -p PROMPT --model MODEL |

List detected providers:

~~~sh
gac providers
~~~

Choose a low-cost model suitable for short Git diffs. The model name is passed to the selected provider and is not hard-coded by gac.

## Configuration

Run onboarding at any time:

~~~sh
gac config
~~~

The default YAML file is located through the OS user config directory, normally ~/.config/gac/config.yaml on Linux. Use --config to select another file.

Example:

~~~yaml
provider: agy
model: low-cost-model
language: en
diff_max_bytes: 65536
diff_max_lines: 1000
timeout_seconds: 120
skip_ci_mode: ask
prompt_template: |
  Generate one Conventional Commits message in {{.Language}}.
  Changed files:
  {{.Stat}}
  Diff:
  {{.Diff}}
  Additional context:
  {{.Context}}
~~~

prompt_template supports .Language, .Stat, .Diff, and .Context. gac still appends its output contract and validates the generated message.

## Safety

gac is a commit-message assistant, not an autonomous Git agent. It does not push, change remotes, bypass hooks, or create a commit before explicit confirmation. The AI receives the selected staged diff, its stat, and any context you provide.

## Development

~~~sh
make check
make build
~~~

Detailed design and policy documents are in [docs/index.md](docs/index.md).

## Release

Create and push a semantic version tag:

~~~sh
git tag -a v0.1.0 -m "release v0.1.0"
git push origin v0.1.0
~~~

The GitHub Actions release workflow builds the supported binaries, generates SHA256 checksums, and publishes a GitHub Release. See [CI/CD](docs/07-cicd.md) and [Release](docs/08-release.md) for maintainer details.
