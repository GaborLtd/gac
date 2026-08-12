# 專案目的與範圍

## 目的

提供一個可部署、可替換 AI backend 的 Go CLI，將 Git 變更整理成一條可直接使用的 Conventional Commits message，降低手動撰寫成本，同時保留使用者對內容與 commit 動作的控制。

binary 名稱為 `gac`，代表 **Generate AI Commit message**。

## 目標使用者

- 已在 Git repository 中工作的開發者。
- 本機可能同時安裝 `agy`、`codex`、`claude` 或其他 AI CLI 的使用者。
- 希望使用低成本 model 產生簡短 commit message，並能在必要時補充脈絡的人。

## 第一版要做

1. 支援檔案與目錄 pathspec 篩選；指定 path 時只讀取 staged 內容，不替使用者 `git add`。
2. 未指定 path 時，處理目前工作目錄下所有 tracked 的 staged 與 unstaged 變更，commit 語意等同 `git commit -a`，但不包含 untracked 檔案。
3. 產生統一 prompt，預設使用英文，並可選擇語言或自訂 prompt；語言要求會固定追加到 prompt contract。
4. 偵測 `agy`、`codex`、`claude`，讓 onboarding 與設定選擇 provider/model。
5. 同時支援互動式流程與 `--non-interactive` message 輸出。
6. 顯示建議、接受補充內容、編輯 message、選擇 `[skip ci]`，再由使用者確認 commit。
7. 以 adapter 隔離各 AI CLI，並提供 mock 測試。

## 明確不做

- 不自動 push、開 PR、修改遠端設定或代替完整 code review。
- 不保證 message 能理解二進位檔、巨大 generated file 或所有語言的語意。
- 不把完整 diff 預設寫入 log、遙測或設定檔。
- 不因 AI 回應而跳過 Git hook、簽章、使用者確認或 repository 權限。

最後更新：2026-08-12
