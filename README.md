# gac — Generate AI Commit message

gac is a Go CLI that turns Git changes into a Conventional Commits message with the help of an AI CLI.

[繁體中文](README.zh-TW.md)

## Features

- With a file path, uses only changes already staged in Git's index.
- Without a path, includes all tracked staged and unstaged changes in the repository, with `git commit -a` semantics.
- Detects agy, codex, and claude CLIs.
- Supports provider, model, language, and additional context selection.
- Supports interactive review and non-interactive output.
- Supports [skip ci] and [ci skip] detection.
- Never runs a standalone git add or git add -A, and never includes untracked files.

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

Without a path, gac analyzes all tracked staged and unstaged changes in the repository. After confirmation, it uses `git commit -a` semantics. With a file path, it uses only staged changes for that file:

~~~sh
gac
gac src/main.go
gac src/main.go
~~~

For a file-specific run, unstaged changes are never sent to the AI. If the file has both staged and unstaged changes, gac stops to prevent an accidental partial commit. Directory paths and multiple paths are rejected. Untracked files are never included.

## Interactive controls

- y / yes: create the commit.
- e / edit: open the message in GIT_EDITOR, VISUAL, or EDITOR.
- a / add: enter context and generate the message again.
- s / skip: append [skip ci] unless [skip ci] or [ci skip] already exists.
- q / cancel: cancel without creating a commit.

The CLI interface is always in English. The default AI output language is English. Change it during onboarding, with `--language`, or in the YAML configuration.

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

During gac config, gac loads a small catalog of multiple low-cost candidates for the selected provider. agy queries `agy models` and filters catalog entries against the live account list when available; Codex and Claude do not currently expose a reliable account-specific model list, so gac also shows the provider documentation URL and login hint. Each catalog entry keeps a provider-compatible value separate from its display label, and Codex candidates include `reasoning_effort: low` so short commit messages use the lowest-effort setting by default; the adapter passes this through the current CLI-compatible `--config model_reasoning_effort="low"` override.

If the provider needs authentication, gac suggests the provider login command. The catalog intentionally keeps several fallback candidates because model availability can change with provider accounts or CLI updates. For a short commit message, choose the cheapest model that is sufficient; gac does not claim to know exact provider pricing. gac does not silently retry another provider or model after a generation failure; choose another listed candidate explicitly to keep routing and cost visible.

The repository also publishes a small recommended model catalog:

~~~sh
curl -fsSL https://raw.githubusercontent.com/GaborLtd/gac/main/models.json
~~~

`gac config` downloads this catalog when possible and falls back to the copy embedded in the binary. It contains several low-cost recommendations per provider, not a complete model list; model access, names, pricing, and availability can change by provider account or CLI version. Use the provider.s login command and model-list command to verify before saving a choice. Set `GAC_MODELS_URL` to use a mirror or a pinned catalog.

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
reasoning_effort: low
language: en
diff_max_bytes: 65536
diff_max_lines: 1000
timeout_seconds: 120
skip_ci_mode: ask
prompt_template: |
  Based on the following Git diff, write one commit message that follows the Conventional Commits format.
  Output only the message itself, without explanation or code fences. Write the message in {{.Language}}.
  Changed files:
  {{.Stat}}
  Diff content:
  {{.Diff}}
  Additional context:
  {{.Context}}
~~~

prompt_template supports .Language, .Stat, .Diff, and .Context. gac still appends its output contract and validates the generated message.

`language` defaults to `en`. Set it to a locale such as `zh-TW` when the commit message should use another language. gac always appends the language requirement to the fixed prompt contract, including when `prompt_template` is customized.

## Safety

gac is a commit-message assistant, not an autonomous Git agent. It does not push, change remotes, bypass hooks, or create a commit before explicit confirmation. The AI receives the selected diff, its stat, and any context you provide.

## Development

~~~sh
make check
make build
~~~

Detailed design and policy documents are in [docs/index.md](docs/index.md).

## Release

Generate and review release notes, then create the tag locally:

~~~sh
gac release preview v0.1.3
gac release tag v0.1.3
git show v0.1.3
git push origin v0.1.3
~~~

`gac release preview` asks the configured provider to summarize commits since the previous release tag. `gac release tag` lets you edit and confirm the Markdown annotated tag message, creates only a local tag, and never pushes automatically. The tag must use `vMAJOR.MINOR.PATCH` and be newer than the previous release.

This project is released under the [MIT License](LICENSE).
