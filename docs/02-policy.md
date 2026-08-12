# Policy 與安全邊界

## Git policy

- `gac` 永遠不執行獨立的 `git add` 或 `git add -A`。無 path 時，分析目前工作目錄下 tracked 的 staged 與 unstaged 變更，確認後使用 `git commit -a` 語意；untracked 檔案不會被納入。
- 有指定 path 時，只分析並 commit 該 pathspec 對應的 staged diff；使用 `git commit --only`，不納入同一檔案的 unstaged 內容。
- 無 path 時使用目前工作目錄 `.` 作為範圍，不擴大到 repository 其他目錄；有 path 時檔案與目錄都只作為 Git pathspec。
- 指定 path 若同時有 staged 與 unstaged 變更，必須停止並要求使用者先整理，避免產生 partial commit。
- 無 path 時 staged 為空仍可分析 tracked 的 unstaged 變更；完全沒有 tracked 變更才回報沒有可 commit 的變更。
- commit 前顯示最終 message 與 commit 範圍。只有明確 `y/yes` 才執行 commit；`e/edit` 進入編輯；其他輸入取消。
- 不執行 push。Git hook 正常執行；hook 失敗時回報原始錯誤。

## AI 與資料 policy

- prompt 只送出必要的 stat、受限 diff 與使用者補充內容。
- 明確告知目前使用的 provider、model、endpoint；敏感資訊不進 prompt，除非使用者自行輸入。
- API key、token、完整 prompt、完整 diff 不寫入一般 log；錯誤訊息需遮罩 credential。
- 外部 CLI 以 argv 傳參，不透過 shell 拼接執行，避免 command injection。
- provider timeout、空回應、非零退出碼與格式錯誤都必須可理解地失敗，不自動改用另一個 provider 除非使用者開啟 fallback。
- CLI 使用者介面固定使用英文。預設 prompt 語言為英文；語言、使用者補充內容與自訂 prompt 可由設定或參數提供。`language` 會自動追加到固定 prompt contract，即使使用自訂 template，也仍需保留輸出格式驗證與資料 policy。

## `[skip ci]` policy

- 預設不自動加入。
- 可在確認流程選擇「不加／自動 append／編輯後決定」。辨識至少包含 `[skip ci]` 與 `[ci skip]`；append 前檢查等效 token，避免重複。
- token 的大小寫與常見格式可正規化，但不能改動使用者其他文字。

## 邊界

這個工具是 commit message assistant，不是 autonomous agent。所有會改變 Git 歷史的動作都必須由使用者在當次流程明確授權。

最後更新：2026-08-12
