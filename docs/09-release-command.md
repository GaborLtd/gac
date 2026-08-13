# Release message 與 tag 指令

## 目的

`gac release` 將「確認哪些 commit 可以發布」與「建立正確版本 tag」放進同一個可檢查流程。AI 只讀取上一個 release tag 到目前 `HEAD` 的 commit subject，不讀取未提交 diff，也不自行決定版本號。

## 指令

```sh
gac release preview v0.1.3
gac release tag v0.1.3
```

兩者都會使用設定檔中的 provider、model 與 language。第一個只顯示 AI 產生的 Markdown release notes；第二個會顯示相同內容，提供：

- `y`：建立本機 annotated tag。
- `e`：使用 `$GIT_EDITOR`、`$VISUAL` 或 `$EDITOR` 編輯 message。
- `q`：取消。

第一版只接受 `vMAJOR.MINOR.PATCH`，且版本必須大於目前可達的最新 release tag。沒有前一個 tag 時，會分析整段 commit history。沒有新 commit 時停止。

## 安全邊界

- 不會自動 `git push`；成功後畫面只提供建議的 `git push origin vX.Y.Z`。
- 建立 tag 前要求 worktree clean，避免 tag 指向使用者尚未提交的狀態。
- 已存在的 tag 不會覆寫。
- release message 可自由使用 Markdown，不套用 Conventional Commits 驗證。
- GitHub Actions 仍由使用者 push `v*` tag 後觸發；CI 通過才建立 GitHub Release。

## 建議流程

```sh
make check
gac release preview v0.1.3
gac release tag v0.1.3
git show v0.1.3
git push origin v0.1.3
```

若 preview 顯示的內容不正確，先修正 commit 或取消 tag；不要用錯誤的 tag message 掩蓋未完成的變更。

最後更新：2026-08-12
