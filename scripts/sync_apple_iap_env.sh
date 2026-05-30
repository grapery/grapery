#!/usr/bin/env bash
# 从 GitHub Repository Variables 同步 Apple IAP 配置到 grapery/.env（需 gh auth login）
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
ENV_FILE="${ROOT}/.env"

if ! command -v gh >/dev/null 2>&1; then
  echo "❌ 需要 GitHub CLI: brew install gh && gh auth login"
  exit 1
fi
if ! gh auth status >/dev/null 2>&1; then
  echo "❌ 请先登录: gh auth login"
  exit 1
fi

cd "$ROOT/.."
REPO="$(gh repo view --json nameWithOwner -q .nameWithOwner 2>/dev/null || true)"
if [ -z "$REPO" ]; then
  echo "❌ 请在 fgrapery 仓库根目录运行（或设置 GH_REPO=owner/repo）"
  exit 1
fi

get_var() { gh variable get "$1" 2>/dev/null || true; }

BUNDLE_ID="$(get_var APPLE_BUNDLE_ID)"
ISSUER_ID="$(get_var APPLE_ISSUER_ID)"
KEY_ID="$(get_var APPLE_KEY_ID)"
PRIVATE_KEY="$(get_var APPLE_PRIVATE_KEY)"

if [ -z "$BUNDLE_ID" ] || [ -z "$ISSUER_ID" ] || [ -z "$KEY_ID" ] || [ -z "$PRIVATE_KEY" ]; then
  echo "❌ GitHub Variables 不完整，请在 Settings → Actions → Variables 配置："
  echo "   APPLE_BUNDLE_ID, APPLE_ISSUER_ID, APPLE_KEY_ID, APPLE_PRIVATE_KEY"
  exit 1
fi

P8_PATH="${ROOT}/certs/AuthKey_${KEY_ID}.p8"
mkdir -p "${ROOT}/certs"
printf '%s\n' "$PRIVATE_KEY" > "$P8_PATH"
chmod 600 "$P8_PATH"

touch "$ENV_FILE"
python3 - "$ENV_FILE" "$BUNDLE_ID" "$ISSUER_ID" "$KEY_ID" "$P8_PATH" <<'PY'
import re, sys
path, bundle, issuer, key_id, p8 = sys.argv[1:6]
block = f"""# ========== VIPPay — Apple IAP (StoreKit 2 / App Store Server API) ==========
# 由 scripts/sync_apple_iap_env.sh 从 GitHub Variables 同步；勿提交 .env / .p8
APPLE_BUNDLE_ID={bundle}
APPLE_ISSUER_ID={issuer}
APPLE_KEY_ID={key_id}
APPLE_PRIVATE_KEY_PATH={p8}
"""
text = open(path, encoding="utf-8").read() if __import__("os").path.exists(path) else ""
pat = r"# ========== VIPPay — Apple IAP.*?(?=\n# |\Z)"
if re.search(pat, text, re.S):
    text = re.sub(pat, block.rstrip() + "\n\n", text, count=1)
else:
    text = text.rstrip() + "\n\n" + block
open(path, "w", encoding="utf-8").write(text)
PY

echo "✅ 已写入 ${ENV_FILE}"
echo "   APPLE_BUNDLE_ID=${BUNDLE_ID}"
echo "   APPLE_ISSUER_ID=${ISSUER_ID}"
echo "   APPLE_KEY_ID=${KEY_ID}"
echo "   APPLE_PRIVATE_KEY_PATH=${P8_PATH}"
echo "   （私钥已保存到 certs/，gitignore 应已忽略）"
