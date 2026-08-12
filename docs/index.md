# 文件索引

本索引只放「目前必須知道的入口與路由」，避免把所有細節塞進單一文件。每份文件上限 3000 字元；超過時依文件政策拆分、摘要並更新本索引。

## 先讀

1. [專案目的與範圍](01-purpose.md)：為何做、做什麼、不做什麼。
2. [Policy 與安全邊界](02-policy.md)：資料、Git、外部 CLI、互動與錯誤行為。
3. [功能規格](03-functional-spec.md)：從 diff 到 commit 的完整流程。
4. [Provider 與 model](04-provider-model.md)：偵測、onboarding、成本偏好與 adapter。
5. [測試策略](05-testing.md)：每項功能的測試與驗收要求。
6. [文件政策](06-document-policy.md)：3000 字元限制、整理流程與索引規則。
7. [CI/CD](07-cicd.md)：文件、Go code、installer 的自動驗證。
8. [Release 與安裝](08-release.md)：binary、tag、checksum 與 installer 流程。

## 非必要查詢

- [決策紀錄](changelog.md)：只在需要追溯設計變更時查閱，不是日常入口。
- [待確認問題](open-questions.md)：開始實作前需確認的產品選項。

## 文件維護

- 文件使用繁體中文；技術名詞保留英文並可附中文說明。
- 每份文件必須保持在 3000 字元內，並在檔尾保留 `最後更新`。
- 新增文件必須在本索引分類；歷史資料放入 changelog，不擴大索引正文。
