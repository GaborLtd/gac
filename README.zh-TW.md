# gac — Generate AI Commit message

gac 是使用 Go 撰寫的 CLI，利用 AI CLI 將 Git 變更轉成 Conventional Commits 格式的 commit message。

[English](README.md)

## 功能

- 指定檔案時，只使用 Git index 中已 staged 的變更。
- 不指定 path 時，納入 repository 中所有 tracked 的 staged 與 unstaged 變更，確認後採用等同 `git commit -a` 的行為。
- 自動偵測 agy、codex、claude。
- 可選擇 provider、model、語言與補充脈絡。
- 支援互動式確認與 non-interactive 輸出。
- 支援加入 [skip ci]，也能辨識 [ci skip]。
- 不執行獨立的 git add 或 git add -A，也不會納入 untracked 檔案或 push。

## 安裝

GitHub Release 建立後，可執行：

~~~sh
curl -fsSL https://raw.githubusercontent.com/GaborLtd/gac/main/install.sh | sh
~~~

installer 會偵測 macOS／Linux 以及 amd64／arm64，驗證 SHA256 checksum，預設安裝到 $HOME/.local/bin。指定版本：

~~~sh
GAC_VERSION=0.1.0 curl -fsSL https://raw.githubusercontent.com/GaborLtd/gac/main/install.sh | sh
~~~

也可以設定 GAC_INSTALL_DIR 或 GAC_REPOSITORY。installer 不使用 sudo。

## 快速開始

先由使用者自行 stage，再執行 gac：

~~~sh
git add path/to/file.go path/to/another-file.go
gac
~~~

檢查產生的訊息後輸入 y，建立 commit。

沒有指定 path 時，分析 repository 中所有 tracked 的 staged 與 unstaged 變更，確認後採用等同 `git commit -a` 的行為；指定 path 時，只處理單一檔案的 staged 變更：

~~~sh
gac
gac src/main.go
gac src/main.go
~~~

指定檔案時，未 staged 的變更不會送給 AI。如果檔案同時有 staged 與未 staged 變更，gac 會停止，避免誤產生 partial commit。Directory path 與多個 path 不支援，untracked 檔案永遠不會納入。

## 互動操作

- y / yes：建立 commit。
- e / edit：使用 GIT_EDITOR、VISUAL 或 EDITOR 編輯訊息。
- a / add：輸入補充脈絡並重新分析。
- s / skip：加入 [skip ci]；若已有 [skip ci] 或 [ci skip] 則不重複加入。
- q / cancel：取消，不建立 commit。

CLI 介面固定使用英文；預設 AI 輸出語言是英文，可在 onboarding、`--language` 或 YAML 設定中修改。

## Non-interactive 模式

-n 是 --non-interactive 的短寫法。它只把最後的 message 輸出到 stdout，不建立 commit，適合 pipe：

~~~sh
gac -n | pbcopy
gac -n > commit-message.txt
gac -n --skip-ci | git commit -F -
~~~

診斷訊息會輸出到 stderr。Non-interactive 模式需要先設定 provider，或直接指定：

~~~sh
gac config
gac -n --provider agy --model low-cost-model
~~~

## Provider 與 model

第一版支援：

| Provider | 偵測方式 | 執行形式 |
| --- | --- | --- |
| agy | PATH 中有 agy | agy --model MODEL --print PROMPT |
| codex | PATH 中有 codex | codex exec --model MODEL PROMPT |
| claude | PATH 中有 claude | claude -p PROMPT --model MODEL |

查看目前可用 provider：

~~~sh
gac providers
~~~

執行 gac config 時，gac 會嘗試透過選定的 provider 列出 model。agy 會執行 agy models，將 model ID 與 display name 分開顯示，並排除帳號目前不存在的 catalog model，設定檔保存 provider 可接受的值（agy 使用 model ID）；Codex 與 Claude 目前沒有可靠的 CLI 帳號可用 model 清單，因此 gac 會顯示 provider 文件 URL，使用者可以直接按 Enter 使用 provider 預設值，或自行輸入 model 名稱。

如果 provider 需要登入，gac 會提示對應的登入指令。catalog 刻意保留多個 fallback，因為 provider 帳號權限或 CLI 更新可能讓某個 model 暫時不可用。產生短 commit message 時，建議選擇足夠使用的最便宜 model；gac 不宣稱知道 provider 的精確價格。gac 不會在失敗後默默改送另一個 provider 或 model；這樣 routing 與成本保持可見，使用者可重新執行 `gac config` 選擇其他候選。

repository 也公開一份小型的低成本 model 建議清單：

~~~sh
curl -fsSL https://raw.githubusercontent.com/GaborLtd/gac/main/models.json
~~~

`gac config` 會優先抓取這份清單，失敗時使用 binary 內建版本。清單每個 provider 有多個低成本建議，但不是完整 model list；不保證帳號權限、model 名稱、價格或 CLI 版本一定相同，請依 provider 的登入與 model list 指令確認。若要使用 mirror 或固定版本，可設定 `GAC_MODELS_URL`。

## 設定

隨時可以重新執行 onboarding：

~~~sh
gac config
~~~

預設使用作業系統的 user config directory；Linux 通常是 ~/.config/gac/config.yaml。也可以使用 --config 指定其他檔案。

設定範例：

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

prompt_template 支援 .Language、.Stat、.Diff 與 .Context。gac 仍會追加輸出格式要求，並驗證 AI 回傳的 commit message。

`language` 預設為 `en`。若要讓 commit message 使用其他語言，可設定例如 `zh-TW`。即使使用自訂 `prompt_template`，gac 仍會把語言要求追加到固定 prompt contract。

## 安全邊界

gac 是 commit message assistant，不是 autonomous Git agent。它不會 push、不會修改 remote、不會跳過 Git hook，也不會在使用者明確確認前建立 commit。送給 AI 的內容是選定範圍的 diff、stat，以及使用者自行輸入的補充脈絡。

## 開發與測試

~~~sh
make check
make build
~~~

完整的設計與 policy 文件請從 [docs/index.md](docs/index.md) 開始閱讀。

## Release

產生並檢查 release notes，再在本機建立 tag：

~~~sh
gac release preview v0.1.3
gac release tag v0.1.3
git show v0.1.3
git push origin v0.1.3
~~~

`gac release preview` 會使用設定的 provider，整理上一個 release tag 到目前 `HEAD` 的 commit。`gac release tag` 會讓你編輯並確認 Markdown annotated tag message，只建立本機 tag，不會自動 push。版本必須使用 `vMAJOR.MINOR.PATCH`，且大於前一個 release。

本專案採用 [MIT License](LICENSE)。
