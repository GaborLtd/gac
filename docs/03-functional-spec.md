# 功能規格

## 主要流程

`gac [path...]`：

1. 確認目前位於 Git repository；取得 branch、status 與變更 stat。
2. 解析 pathspec：有 path 就使用指定檔案／目錄；無 path 就使用目前工作目錄 `.`。不執行任何 add。
3. 無 path 時讀取範圍內 tracked 的 `HEAD..worktree` diff；有 path 時讀取 staged diff。沒有內容就結束並回報沒有可 commit 的變更。
4. 依 bytes 與 lines 的雙重限制擷取 diff，先達到的限制生效；保留檔案標頭與 stat，並在 prompt 標示截斷。
5. 取得 provider/model 設定；首次使用進入 onboarding，之後可用 `config` 指令修改。
6. 送出 prompt，預設要求英文的 Conventional Commits message；可選語言、補充內容與自訂 prompt。`language` 會自動追加到固定輸出契約。清理 code fence、前後空白與多餘說明。
7. 顯示建議，讓使用者補充脈絡並重新分析，或直接編輯 message。
8. 提供 CI skip 選項；辨識 `[skip ci]`、`[ci skip]` 等常見 token，顯示最終文字與 commit 範圍。
9. 指定 path 時，commit 前再次確認目標範圍沒有 staged／worktree 差異；互動模式中，`y` 執行限定範圍的 commit。無 path 時，`y` 執行 `git commit -a` 語意；`e` 交給 `$GIT_EDITOR`；其他輸入取消。

## Commit scope 語意

無 path 時，目標是目前工作目錄 `.` 下 tracked 的 staged 與 unstaged 內容，commit 行為等同 `git commit -a`；untracked 檔案不會被納入。這個流程不執行獨立的 `git add`。有 path 時，目標是指定檔案／目錄下已 staged 的內容，並保護 partial staging。`--non-interactive` 則只把產生的 message 寫到 stdout，診斷訊息寫到 stderr，不建立 commit，方便使用者自行 pipe 到 `pbcopy`、檔案或其他指令。

## 暫定指令

- `gac [path...]`：分析並互動式 commit。
- `gac config`：設定 provider、model、語言、diff 上限與 `[skip ci]` 預設。
- `gac providers`：列出偵測結果、健康狀態與可用 model。
- `gac -n, --non-interactive [path...]`：只輸出 message 到 stdout，不 commit；供 script pipe 或重用，缺必要設定時失敗。
- `gac --skip-ci [path...]`：在既有 message 沒有等效 token 時加入 `[skip ci]`。

## 可設定項目

provider、model、語言、diff 最大 bytes／lines、自訂 prompt template、是否啟用 fallback provider、`[skip ci]` 模式與 timeout。設定檔優先採 YAML，不相容時改用 TOML。

## 錯誤原則

錯誤需指出階段（Git／偵測／AI／解析／commit）、可採取的下一步，以及是否已改變 index 或工作區；除使用者要求外，不自動復原既有 staged 變更。

最後更新：2026-08-12
