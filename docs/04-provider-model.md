# Provider 與 model 設計

## Provider 抽象

Go 內部以 `Provider` adapter 隔離外部工具，至少包含：`Name`、`Detect`、`ListModels`（可選）、`Generate`、`Health`。adapter 不應知道 Git 細節，只接收 prompt 與 model。

## 偵測順序

第一版掃描 PATH，候選為 `agy`、`codex`、`claude`；其他 provider 保留 adapter 擴充點，日後再加入。實際可用性需以版本／health check 判定，不因檔案存在就視為可用。CLI provider 需檢查可執行檔與非互動輸出能力。

同時可用時，不應默默使用昂貴或不透明的 provider：首次 onboarding 顯示候選、能力、預估成本標籤與預設 model，讓使用者選擇。非互動模式則使用設定值，沒有設定就失敗並提示執行 onboarding。

## Model 選擇

- 優先低成本、低延遲、能處理短 diff 的 model；成本只作為偏好，不宣稱精確價格。
+ provider 若能列 model，onboarding 顯示編號清單；每個項目分為 model ID 與人類可讀的 display name，設定檔保存 provider 可接受的 model value（agy 使用 display name），使用者也可選擇手動輸入。
- 第一版的 agy adapter 執行 agy models 並解析清單。
- Codex 與 Claude CLI 沒有可靠的 account-specific model list；不能列出時顯示 adapter 提供的官方文件 URL。
- model list 查詢失敗時提示登入：agy 建議啟動 agy，Codex 建議 codex login，Claude 建議啟動 claude 完成認證；gac 不自動登入、不接觸 token。
- 不能列出時，空白輸入代表使用 provider default，不再讓使用者面對沒有說明的空白欄位。
- 清單名稱若含 Low、Mini、Nano、Haiku 等低成本線索，onboarding 會優先排序並標記建議；這是名稱線索，不是價格保證。
- 使用者可指定 model；設定值優先於自動建議。
- onboarding 預設推薦低成本 model；使用者可選 model，也可在設定中永久覆寫。
- 不把 `Gemini 3.5 Flash (Low)`、`ornith:9b` 等名稱硬編碼成唯一選項；它們只是相容範例。

## Fallback

第一版預設關閉自動 fallback，避免意外把程式碼送到另一個 backend。開啟後只在 timeout／不可用時切換，必須先顯示實際使用的 provider/model；若 prompt 含敏感資料，仍應遵守同一份資料 policy。

## 相容性

CLI invocation 的 flags、model listing、stdin、輸出解析與文件 URL 都放在各自 adapter；核心流程只依賴介面。所有 adapter 都要有 fake command 測試，不以真實帳號或網路作為單元測試條件。

## Prompt 設定

CLI user-facing text 固定使用英文。預設 prompt 與 `language` 設定控制 AI 回覆語言；核心會把語言要求追加到固定輸出契約，即使自訂 template 沒有自行寫入語言，也不能繞過這項要求。

自訂 prompt 放在 YAML 設定的 `prompt_template`。template 可使用變數插入 stat、diff、語言與使用者補充內容；核心仍會追加輸出格式要求、資料限制與解析驗證，避免自訂內容讓 CLI 失去安全契約。

最後更新：2026-08-12
