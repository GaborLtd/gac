# CI/CD

## CI 觸發

`.github/workflows/ci.yml` 在 pull request、`main` push 與手動觸發時執行。它不需要 AI provider、網路 API key 或真實 Git repository。

## CI 檢查

- `gofmt`：所有 Go 檔案必須已格式化。
- `go test ./...`：單元與 Git temporary repository integration tests。
- `go vet ./...`：基本 Go 靜態檢查。
- `go build ./cmd/gac`：確認 binary 可建置。
- `sh -n install.sh` 與 `scripts/test-install.sh`：驗證 installer 語法與 checksum／解壓／安裝流程。
- `scripts/check-docs.sh`：檢查文件字數、index 路由與連結目標。

CI 使用 `go.mod` 指定的 Go 版本，權限只有 `contents: read`。任何檢查失敗都阻止合併。

## 本機對等指令

```sh
make check
```

若沒有 make，也可依序執行：

```sh
go test ./...
go vet ./...
go build ./cmd/gac
sh scripts/check-docs.sh
sh scripts/test-install.sh
```

最後更新：2026-08-12
