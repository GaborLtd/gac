# 設計決策紀錄

## 2026-08-12

- 確認 `gac` 代表 **Generate AI Commit message**；不是 Go AI Commit。
- 確認 `gac` 無 path 時採用無 path 的 `git commit -a` 語意；指定 path 時只接受單一檔案，不支援 directory path 或多 path。
- 第一版若目標範圍有 staged／worktree 差異，先停止以避免 partial staging 被誤 commit。
- 第一版 provider 鎖定 `agy`、`codex`、`claude`；預設 prompt 語言為英文，可選語言與自訂 prompt。
- 確認支援 `[skip ci]`、`[ci skip]` 等常見 CI skip token；`--non-interactive` 只輸出 message，適合 pipe，不建立 commit。
- 自訂 prompt 放在 YAML `prompt_template`，並保留核心輸出格式驗證。
- 確認 `-n` 作為 `--non-interactive` 短旗標；第一版不做 signed checksum。
- 設定格式優先採 YAML；diff 同時受 bytes 與 lines 限制；規劃 GitHub public release 與 checksum 驗證 installer。
- 完成第一版 Go CLI 核心、provider adapter、YAML config、installer 與離線測試骨架。
- config onboarding 會由 agy models 列出 model；Codex／Claude 無法可靠列出帳號 model 時顯示官方文件 URL，空白可使用 provider default。
- model list 失敗時提供 provider-specific 登入提示；可辨識低成本名稱線索時優先排序並標記建議。
- CLI user-facing text 統一使用英文；`language` 預設為 `en`，並由核心追加到固定 prompt contract，包括自訂 `prompt_template`。
- 無 path 時分析 repository 中 tracked 的 staged／unstaged 變更，確認後採用 `git commit -a` 語意；指定單一檔案時維持 staged-only 與 partial staging 保護。
- 完成 CI workflow、GoReleaser release workflow、Makefile、文件檢查器與 binary 發布文件。
- 建立文件索引，將目的、policy、功能、provider/model、測試策略分離。
- 文件單檔上限設定為 3000 字元；超過時拆分、摘要並更新索引。
- 第一版定位為互動式 commit message assistant，不自動 push、不跳過 hook、不在未確認前 commit。
- provider 以 adapter 隔離，首次使用需讓使用者選 provider 與 model；自動 fallback 預設關閉。
- `[skip ci]` 預設不加入，但可在分析後選擇 append 或編輯。

最後更新：2026-08-12
