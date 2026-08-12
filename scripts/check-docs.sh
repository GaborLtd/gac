#!/bin/sh
set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
docs_dir="$root/docs"
index="$docs_dir/index.md"
limit=3000
failed=0

for file in "$docs_dir"/*.md; do
  chars=$(wc -m < "$file" | tr -d '[:space:]')
  if [ "$chars" -gt "$limit" ]; then
    echo "文件超過 ${limit} 字元：${file} (${chars})" >&2
    failed=1
  fi
  base=$(basename "$file")
  if [ "$base" != "index.md" ] && ! grep -Fq "](${base})" "$index"; then
    echo "文件未被 index 路由：${base}" >&2
    failed=1
  fi
done

for link in $(sed -n 's/.*](\([^)]*\.md\)).*/\1/p' "$index"); do
  if [ ! -f "$docs_dir/$link" ]; then
    echo "index link 不存在：${link}" >&2
    failed=1
  fi
done

if [ "$failed" -ne 0 ]; then
  exit 1
fi
echo "文件檢查通過"
