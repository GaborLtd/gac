#!/bin/sh
set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
tmp=$(mktemp -d 2>/dev/null || mktemp -d -t gac-install-test)
trap 'rm -rf "$tmp"' EXIT HUP INT TERM

fixture="$tmp/fixture"
fakebin="$tmp/bin"
install_dir="$tmp/install"
mkdir -p "$fixture/payload" "$fakebin"
printf '#!/bin/sh\necho test\n' > "$fixture/payload/gac"
tar -czf "$fixture/gac_darwin_amd64.tar.gz" -C "$fixture/payload" gac
if command -v sha256sum >/dev/null 2>&1; then
  checksum=$(sha256sum "$fixture/gac_darwin_amd64.tar.gz" | awk '{print $1}')
else
  checksum=$(shasum -a 256 "$fixture/gac_darwin_amd64.tar.gz" | awk '{print $1}')
fi
printf '%s  %s\n' "$checksum" gac_darwin_amd64.tar.gz > "$fixture/checksums.txt"

cat > "$fakebin/uname" <<'EOF'
#!/bin/sh
case "${1:-}" in
  -s) echo Darwin ;;
  -m) echo x86_64 ;;
  *) exit 1 ;;
esac
EOF

cat > "$fakebin/curl" <<EOF
#!/bin/sh
set -eu
out=
previous=
for arg in "\$@"; do
  if [ "\$previous" = "-o" ]; then out="\$arg"; fi
  previous="\$arg"
done
case "\${previous:-}" in
  *checksums.txt) cp "$fixture/checksums.txt" "\$out" ;;
  *.tar.gz) cp "$fixture/gac_darwin_amd64.tar.gz" "\$out" ;;
  *) echo "unexpected URL: \${previous:-}" >&2; exit 1 ;;
esac
EOF
chmod 0755 "$fakebin/uname" "$fakebin/curl"

sh -n "$root/install.sh"
PATH="$fakebin:$PATH" GAC_VERSION=1.0.0 GAC_INSTALL_DIR="$install_dir" sh "$root/install.sh"
test -x "$install_dir/gac"
echo "installer test passed"
