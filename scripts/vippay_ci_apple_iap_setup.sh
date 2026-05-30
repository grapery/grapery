#!/usr/bin/env bash
# CI：将 GitHub Variable APPLE_PRIVATE_KEY 写入 .env 单行（JSON 转义），供 docker compose 解析。
# 变量名保持 APPLE_PRIVATE_KEY，不使用 heredoc。
set -euo pipefail

if [ -z "${APPLE_PRIVATE_KEY:-}" ] || [ ! -f .env ]; then
  exit 0
fi

python3 <<'PY'
import json
import os
from pathlib import Path

pem = os.environ.get("APPLE_PRIVATE_KEY", "").strip()
if not pem:
    raise SystemExit(0)

path = Path(".env")
lines = [
    line
    for line in path.read_text(encoding="utf-8").splitlines()
    if not line.startswith("APPLE_PRIVATE_KEY=")
]
lines.append("APPLE_PRIVATE_KEY=" + json.dumps(pem))
path.write_text("\n".join(lines) + "\n", encoding="utf-8")
PY
