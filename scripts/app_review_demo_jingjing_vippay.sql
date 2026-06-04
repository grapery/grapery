-- =============================================================================
-- App Review demo account — VipPay tables (same MySQL database as Grapery)
-- =============================================================================
--
-- Pairs with: scripts/app_review_demo_jingjing.sql
--
-- Demo login:
--   Email:    jingjing@grapery.xyz
--   Password: Gr@p3ryIap2026!
--
-- User UUID: revw-jingjing-0000-8000-000000000001
-- VipPay user_id (FNV-1a int64): -1545509627481098784
--   Regenerate: go run ./cmd/gen-review-demo
--
-- Prerequisites:
--   1. app_review_demo_jingjing.sql already applied
--   2. iap_products row for com.grapery.pro.monthly exists
--      (start vippay once, or ensure SeedGraperyIAPAppleProductsIfMissing ran)
--
-- Usage:
--   mysql -u root -p -h localhost grapery < scripts/app_review_demo_jingjing_vippay.sql
-- Or: make review-demo-load
--
-- Verification:
--   GET /api/vippay/vip/info (Bearer token from email login)
--     → is_vip: true, credit_limit ≈ 103 (100 subscription + ~3 free credits)
--   GET /api/v1/membership/current → tier: basic
--
-- Common failures:
--   - is_vip=false → this file not run, or iap_products empty (package_plan_id NULL)
--   - FNV user_id mismatch → wrong UUID in main SQL vs gen-review-demo output
--   - AI still fails → main memberships.token_quota not set (run main SQL first)
--   - Sandbox IAP + manual grant → both may coexist; prefer email login for review
--
-- NOTE: Apple Sandbox tester email is unrelated to jingjing@grapery.xyz.
-- =============================================================================

SET NAMES utf8mb4 COLLATE utf8mb4_unicode_ci;

SET @review_user_uuid := 'revw-jingjing-0000-8000-000000000001';
-- int64(fnv1a64(@review_user_uuid)) — must match cmd/vippay stringToInt64
SET @vippay_user_id := -1545509627481098784;
-- uint64 same bits — for apple_subscriptions.user_id column
SET @vippay_user_id_u := 16901234446228452832;
SET @provider_sub_id := 'app-review-manual-grant';
SET @apple_otx_id := 'app-review-manual-grant-jingjing';

-- Require IAP catalog
SET @package_plan_id := (
    SELECT id FROM iap_products
    WHERE product_id = 'com.grapery.pro.monthly'
    LIMIT 1
);

SELECT IF(
    @package_plan_id IS NULL,
    'ERROR: iap_products missing com.grapery.pro.monthly — start vippay once or seed IAP catalog',
    CONCAT('Using iap_products.id=', @package_plan_id)
) AS iap_check;

-- Idempotent cleanup for manual review grant only
DELETE FROM user_subscriptions WHERE provider_sub_id = @provider_sub_id COLLATE utf8mb4_unicode_ci;
DELETE FROM apple_subscriptions WHERE original_transaction_id = @apple_otx_id COLLATE utf8mb4_unicode_ci;

-- Skip inserts if catalog missing (avoid partial broken row)
INSERT INTO user_subscriptions (
    user_id,
    package_plan_id,
    order_id,
    status,
    start_time,
    end_time,
    auto_renew,
    payment_method,
    payment_provider,
    provider_sub_id,
    amount,
    currency,
    quota_limit,
    quota_used,
    max_roles,
    max_contexts,
    available_models,
    features,
    metadata,
    created_at,
    updated_at
)
SELECT
    @vippay_user_id,
    @package_plan_id,
    0,
    1,
    NOW(),
    DATE_ADD(NOW(), INTERVAL 1 YEAR),
    1,
    1,
    'manual',
    @provider_sub_id,
    2990,
    'CNY',
    1000000,
    0,
    50,
    50,
    '[]',
    '{}',
    '{}',
    NOW(),
    NOW()
FROM DUAL
WHERE @package_plan_id IS NOT NULL;

INSERT INTO apple_subscriptions (
    user_id,
    original_transaction_id,
    product_id,
    purchase_date,
    expires_date,
    status,
    auto_renew_status,
    is_in_intro_offer_period,
    is_in_grace_period,
    app_user_id,
    created_at,
    updated_at
)
SELECT
    @vippay_user_id_u,
    @apple_otx_id,
    'com.grapery.pro.monthly',
    NOW(),
    DATE_ADD(NOW(), INTERVAL 1 YEAR),
    'Active',
    'On',
    0,
    0,
    @review_user_uuid,
    NOW(),
    NOW()
FROM DUAL
WHERE @package_plan_id IS NOT NULL;

SELECT
    CASE
        WHEN @package_plan_id IS NULL THEN 'SKIPPED — seed iap_products first'
        ELSE 'App Review VipPay grant ready'
    END AS status,
    @review_user_uuid AS user_uuid,
    @vippay_user_id AS vippay_user_id;
