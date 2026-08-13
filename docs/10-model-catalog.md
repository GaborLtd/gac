# Model catalog

## 目的

`models.json` 是 gac 的低成本 model 建議清單，讓 onboarding 不必要求使用者在空白欄位手動猜 model 名稱。它不是 provider 的官方完整清單，也不是價格或帳號可用性的保證。

## 來源與 fallback

公開來源：

```text
https://raw.githubusercontent.com/GaborLtd/gac/main/models.json
```

執行 `gac config` 時，gac 會優先讀取這個 URL；下載失敗、格式錯誤或版本不支援時，使用 binary 內嵌的 catalog。可用 `GAC_MODELS_URL` 指向 mirror 或固定版本的 raw URL。

catalog 使用 `schema_version`，每個 provider 保留少量、可互相替代的 `recommended: true` 低成本選項，排序代表 onboarding 的建議順序；catalog 不會在生成失敗後自動切換，以避免未明示的資料路由或成本變化。每個 model 分開記錄：

- `id`：清單識別用。
- `value`：實際傳給 provider CLI 的值。
- `display_name`：給使用者看的名稱。
- `cost_tier`：成本偏好標籤，不是即時價格。
- `availability`：提醒使用者自行確認帳號與 CLI 版本。
- `reasoning_effort`：可選的 reasoning effort；目前 Codex 低成本候選使用 `low`。

## 維護規則

- 新增或移除 model 前先查 provider 官方文件。
- 不把 deprecated model 放入推薦清單。
- 不在 catalog 放 API key、token 或使用者資料。
- 更新後同時更新 `updated_at`、測試與 README 使用說明。
- 若 provider CLI 能列出帳號模型，仍以 CLI 實際錯誤作最後判斷；catalog 只負責減少 onboarding 的猜測成本。

目前清單包含 `agy`、`codex` 與 `claude`。agy 提供 Flash Lite、Flash 的多個 Low 候選；Codex 提供 GPT-5.4 mini、GPT-5.4、GPT-5.5 與 GPT-5.6 Luna／Terra／Sol 的 Low effort 候選；Claude 目前只放入已確認的 Haiku 類低成本選項。這些值可能隨 provider 更新，請在 release 前重新確認。

最後更新：2026-08-13
