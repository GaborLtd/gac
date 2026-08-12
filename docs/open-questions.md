# 待確認問題

以下項目在開始 Go 實作前確認；其餘設計可先依現有規格推進。

1. CI skip 是否需要獨立的 `--skip-ci` 旗標？token 辨識目前決定支援 `[skip ci]`、`[ci skip]` 等常見格式。
2. diff 預設數值採多少？建議同時限制 bytes 與 lines，先達到者截斷：bytes 控制 payload 成本，lines 保持可讀性。
3. GitHub repository 是否確定使用 `gaborltd/gac`？
4. 第一版不做 signed checksum；只提供 SHA256 checksum。

最後更新：2026-08-12
