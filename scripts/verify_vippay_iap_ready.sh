#!/usr/bin/env bash
# 上线前 / Sandbox 真机测试前：检查 VipPay IAP 生产可达性。
set -euo pipefail

BASE="${VIPPAY_BASE_URL:-https://rankquantity.xyz/api/vippay}"

echo "== VipPay IAP readiness =="
echo "Base: $BASE"
echo

health_code="$(curl -sS -o /tmp/vippay_health.json -w '%{http_code}' "$BASE/health")"
echo "GET /health -> HTTP $health_code"
cat /tmp/vippay_health.json
echo

notif_code="$(curl -sS -o /tmp/vippay_notif.json -w '%{http_code}' -X POST "$BASE/iap/apple/notification" \
  -H 'Content-Type: application/json' \
  -d '{"signedPayload":"probe"}')"
echo "POST /iap/apple/notification (probe) -> HTTP $notif_code"
echo "  expect after deploy: 400 + msg failed to parse notification (not invalid request parameters)"
cat /tmp/vippay_notif.json
echo

echo "== ASC Server Notifications V2 =="
echo "Production URL: ${BASE}/iap/apple/notification"
echo "Sandbox URL:    ${BASE}/iap/apple/notification"
echo "After saving in App Store Connect, use「发送测试通知」and check VipPay logs."
echo
echo "== iPhone Sandbox purchase checklist =="
echo "1. Xcode Scheme: voyager (NOT voyager-storekit)"
echo "2. iPhone: Settings -> App Store -> Sandbox Account"
echo "3. App: Membership -> purchase Basic Monthly"
echo "4. Verify GET /vip/info: is_vip=true, credit_limit ~103"
echo "5. Kill app and reopen; membership + credits persist"
