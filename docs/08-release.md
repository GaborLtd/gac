# Release 與安裝

## 建置

`make build` 建置目前平台的 `dist/gac`；`make build VERSION=v0.1.0` 會把版本寫入 `gac version`。正式 release 由 GoReleaser 產生 darwin、linux、windows 的 amd64／arm64 archives 與 `checksums.txt`。

## 發布流程

repository 預定為 `gaborltd/gac`，使用 SemVer tag 觸發 `.github/workflows/release.yml`：

```sh
make check
git tag -a v0.1.0 -m "release v0.1.0"
git push origin v0.1.0
```

GitHub Actions 會執行驗證、GoReleaser build、建立 GitHub Release、上傳 binary archives 與 SHA256 checksum。第一版不做 signed checksum；不得在本機手動上傳未經 CI 驗證的 artifact。

## Installer

```sh
curl -fsSL https://raw.githubusercontent.com/gaborltd/gac/main/install.sh | sh
```

installer 依 OS／ARCH 下載 release asset，驗證 SHA256 後安裝到 `$GAC_INSTALL_DIR`；預設是 `$HOME/.local/bin`，不使用 sudo。`GAC_VERSION=0.1.0` 可固定版本。也可先下載 `install.sh`、檢查內容後再執行。

支援的 asset 命名為 `gac_<os>_<arch>.tar.gz`。installer 第一版支援 macOS／Linux 的 amd64／arm64；Windows 使用 GitHub Release 手動下載。

## Release 檔案

- `.goreleaser.yaml`：跨平台 binary、版本注入、archive 與 checksum。
- `.github/workflows/release.yml`：`v*` tag 觸發的發布 job。
- `install.sh`：下載、checksum 驗證與本機安裝。
- `scripts/test-install.sh`：離線 fake release integration test。

最後更新：2026-08-12
