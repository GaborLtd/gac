# Policy 與安全邊界

## Git policy

- `gac [path...]` 永遠不執行 `git add`、`git add -A` 或 `git commit -a`；檔案與目錄都只作為 Git pathspec 篩選條件。`git commit -a` 會自動加入 tracked 的未 staged 變更，因此不符合本工具的 staged-only policy。
- 有指定 path 時，只分析並 commit 該 pathspec 對應的 staged diff；未指定時使用目前工作目錄 `.` 作為範圍，不擴大到 repository 其他目錄。
- 只分析 index（staged snapshot）。staged 為空時直接回報沒有可 commit 的變更，不 fallback 到未 staged 工作區。
- 若目標 path 同時有未 staged 變更，必須清楚警告未 staged 內容不會送給 AI，也不得被 commit；使用者需自行處理後再執行。
- 為避免 Git pathspec commit 把工作區內容重新納入，真正 commit 前若目標範圍存在 staged／worktree 差異，應停止並要求使用者先整理；第一版不承諾 partial staging 的 commit。
- commit 前顯示最終 message 與 commit 範圍。只有明確 `y/yes` 才執行 commit；`e/edit` 進入編輯；其他輸入取消。
- 不執行 push。Git hook 正常執行；hook 失敗時回報原始錯誤。

## AI 與資料 policy

- prompt 只送出必要的 stat、受限 diff 與使用者補充內容。
- 明確告知目前使用的 provider、model、endpoint；敏感資訊不進 prompt，除非使用者自行輸入。
- API key、token、完整 prompt、完整 diff 不寫入一般 log；錯誤訊息需遮罩 credential。
- 外部 CLI 以 argv 傳參，不透過 shell 拼接執行，避免 command injection。
- provider timeout、空回應、非零退出碼與格式錯誤都必須可理解地失敗，不自動改用另一個 provider 除非使用者開啟 fallback。
- 預設 prompt 語言為英文；語言、使用者補充內容與自訂 prompt 可由設定或參數提供，但仍需保留輸出格式驗證與資料 policy。

## `[skip ci]` policy

- 預設不自動加入。
- 可在確認流程選擇「不加／自動 append／編輯後決定」。辨識至少包含 `[skip ci]` 與 `[ci skip]`；append 前檢查等效 token，避免重複。
- token 的大小寫與常見格式可正規化，但不能改動使用者其他文字。

## 邊界

這個工具是 commit message assistant，不是 autonomous agent。所有會改變 Git 歷史的動作都必須由使用者在當次流程明確授權。

最後更新：2026-08-12
