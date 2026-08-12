# AI Commit Assistant

`gac` 代表 **Generate AI Commit message**。這是一個以 Go 撰寫、可單一 binary 部署的 CLI，協助使用者從 Git 變更產生 Conventional Commits commit message。

非互動輸出可直接 pipe：`gac -n | pbcopy`。第一版只分析已 staged 的內容，不會自動 add。

文件入口是 [docs/index.md](docs/index.md)。CI、binary build 與 GitHub Release 流程已納入 repository。

設計重點：

- `gac [path...]` 只處理已經 staged 的檔案；不會替使用者執行 `git add` 或 `git add -A`。
- 自動偵測本機可用的 AI CLI／服務，並提供 onboarding 選擇 provider 與 model。
- 預設偏好低成本、足以處理 commit message 的 model；設定可被使用者覆寫。
- 支援 staged diff、指定檔案或目錄，並嚴格限制 commit 範圍。
- 分析完成後允許使用者補充脈絡、編輯訊息、選擇是否加入 `[skip ci]`，最後才明確確認 commit。
- 不自動 push、不繞過 Git hook、不在未確認前建立 commit。
- 每項功能都必須有文件與自動化測試。

## 安裝

Release 後可使用：

```sh
curl -fsSL https://raw.githubusercontent.com/gaborltd/gac/main/install.sh | sh
```

installer 預設安裝到 `$HOME/.local/bin`，並驗證 SHA256 checksum；也可先下載 `install.sh` 檢查內容後再執行。

## 開發與 Release

```sh
make check
make build
git tag -a v0.1.0 -m "release v0.1.0"
git push origin v0.1.0
```

tag push 後由 GitHub Actions 建置並發布 binary。詳細流程見 [CI/CD](docs/07-cicd.md) 與 [Release](docs/08-release.md)。
