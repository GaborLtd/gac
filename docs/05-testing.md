# 測試策略與驗收

每個功能合併前都必須同時更新規格文件與測試；測試需能離線執行，外部 AI 只做明確標記的整合測試。

## 核心測試

- Git repository：無 path 時納入 repository 中 tracked staged／unstaged 並排除 untracked，指定單一檔案時只納入 staged，directory path 與多 path 會被拒絕；partial staging 保護、空變更、非 repository；確認不執行獨立 add／`-A`。
- Diff 限制：line／byte 上限、檔案標頭保留、截斷標記、UTF-8 與 binary 檔案。
- Prompt：stat、diff、補充脈絡、語言與限制正確注入；不得意外混入未選範圍。
- Non-interactive：stdout 只有 message、stderr 才有診斷，且不建立 commit，可安全 pipe。
- 互動流程：接受、取消、編輯、重新分析、EOF／無效輸入。
- `[skip ci]`：三種模式、既有 token、重複 append、大小寫與 message 保留。
- Commit：正確傳遞 message、pathspec／staged scope、editor、hook 失敗與不 push。

## Provider 測試

使用 fake executable、fake HTTP server 與固定回應測試：detect、model list 解析、低成本排序與標記、登入提示、文件 URL fallback、timeout、非零退出、空回應、格式清理、credential 遮罩與可選 fallback。

## 文件與品質閘門

- CI 透過 `scripts/check-docs.sh` 檢查每份規格文件不超過 3000 字元、所有文件都被 index 路由，且 index 連結存在。
- 超過上限時，文件整理流程先保留結論與 policy，將細節拆至新文件，更新 index 與 changelog；不得只刪文字造成資訊遺失。
- 文件整理器需測試 dry-run、拆分、索引更新、重跑後穩定性，以及摘要失敗時不覆寫原檔。
- release 流程需測試版本 tag、各平台 artifact、checksum、installer 的 OS／ARCH 判斷與安裝目錄覆寫；離線 installer 測試由 `scripts/test-install.sh` 執行。
- `scripts/test-install.sh` 使用 fake release source 離線驗證 installer；不得依賴網路或真實 GitHub release。
- 執行 `go test ./...`、`go vet ./...`；CLI 整合測試使用暫存 repository，不碰使用者真實 repository。

## 驗收標準

使用者能在沒有網路的情況測試 Git 與互動流程；能選擇已偵測 provider/model；未明確確認時不產生 commit；成功 commit 時 message、scope、`[skip ci]` 行為均符合畫面上的最後選擇。

最後更新：2026-08-12
