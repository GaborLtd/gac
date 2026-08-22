#!/bin/sh
set -eu

repo="${GAC_REPOSITORY:-gaborltd/gac}"
version="${GAC_VERSION:-latest}"
install_dir="${GAC_INSTALL_DIR:-${HOME:-}/.local/bin}"

if [ -z "${HOME:-}" ] && [ -z "${GAC_INSTALL_DIR:-}" ]; then
  echo "gac: HOME 未設定，請設定 GAC_INSTALL_DIR" >&2
  exit 1
fi

case "$(uname -s)" in
  Darwin) os="darwin" ;;
  Linux) os="linux" ;;
  *) echo "gac: 不支援的作業系統：$(uname -s)" >&2; exit 1 ;;
esac

case "$(uname -m)" in
  x86_64|amd64) arch="amd64" ;;
  arm64|aarch64) arch="arm64" ;;
  *) echo "gac: 不支援的 CPU 架構：$(uname -m)" >&2; exit 1 ;;
esac

asset="gac_${os}_${arch}.tar.gz"
if [ "$version" = "latest" ]; then
  base="https://github.com/${repo}/releases/latest/download"
else
  version="${version#v}"
  base="https://github.com/${repo}/releases/download/v${version}"
fi

tmp="$(mktemp -d 2>/dev/null || mktemp -d -t gac)"
trap 'rm -rf "$tmp"' EXIT HUP INT TERM

echo "下載 gac (${os}/${arch})..." >&2
curl -fsSL "${base}/${asset}" -o "${tmp}/${asset}"
curl -fsSL "${base}/checksums.txt" -o "${tmp}/checksums.txt"

expected="$(awk -v file="$asset" '$2 == file { print $1; exit }' "${tmp}/checksums.txt")"
if [ -z "$expected" ]; then
  echo "gac: checksum 中找不到 ${asset}" >&2
  exit 1
fi

if command -v sha256sum >/dev/null 2>&1; then
  actual="$(sha256sum "${tmp}/${asset}" | awk '{print $1}')"
elif command -v shasum >/dev/null 2>&1; then
  actual="$(shasum -a 256 "${tmp}/${asset}" | awk '{print $1}')"
else
  echo "gac: 找不到 sha256sum 或 shasum" >&2
  exit 1
fi
if [ "$actual" != "$expected" ]; then
  echo "gac: checksum 驗證失敗" >&2
  exit 1
fi

mkdir -p "$install_dir"
tar -xzf "${tmp}/${asset}" -C "$tmp"
cp "${tmp}/gac" "${install_dir}/gac"
chmod 0755 "${install_dir}/gac"
echo "已安裝 ${install_dir}/gac" >&2
case ":${PATH}:" in
  *":${install_dir}:"*) ;;
  *) echo "請將 ${install_dir} 加入 PATH" >&2 ;;
esac

if ! command -v agy >/dev/null 2>&1 && ! command -v codex >/dev/null 2>&1; then
  cat >&2 <<'EOF'

尚未偵測到 agy 或 codex CLI。gac 需要其中一個 AI CLI 才能產生 commit message，請先安裝：

agy（Antigravity CLI）
  macOS / Linux: curl -fsSL https://antigravity.google/cli/install.sh | bash
  Windows (PowerShell): irm https://antigravity.google/cli/install.ps1 | iex
  Windows (CMD): curl -fsSL https://antigravity.google/cli/install.cmd -o install.cmd && install.cmd && del install.cmd

codex（OpenAI Codex CLI）
  macOS / Linux: curl -fsSL https://chatgpt.com/codex/install.sh | sh
  Windows (PowerShell): powershell -ExecutionPolicy ByPass -c "irm https://chatgpt.com/codex/install.ps1 | iex"
  npm: npm install -g @openai/codex
  Homebrew: brew install --cask codex

安裝後請重新開啟 terminal，或重新載入 PATH，再執行 gac。
官方文件：
  agy: https://antigravity.google/docs/cli/getting-started/
  codex: https://github.com/openai/codex#quickstart
EOF
fi
