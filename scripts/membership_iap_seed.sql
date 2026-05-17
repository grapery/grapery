-- =============================================================================
-- subscription_plans 初始化 / upsert（与 App Store SKU、vippay `/iap/products` 对齐）
-- =============================================================================
-- `iap_product_id` 必须与 App Store Connect 商品 ID、`productId` 完全一致。
-- `features` 为 JSON 数组，元素为 iOS `Localizable.strings` 中的 key（PaymentView / 后端列表共用）。
--
-- 档位权益摘要：
--   · 对应月点数（基础 100 / 高级 200；Ultra 下架行仍为「不限」占位）
--   · 导出配图到本地（相册）
--   · 作品与配图可无限制分享
--   · 云端配图 / 创作长期保存（不设会员到期删除）
--   · 可使用最新图像生成模型
--   · 基础：故事碎片 / 分镜串行生成（不可并行）；高级：可并行 + AI 任务队列更高优先级
--
-- VipPay：`iap_products` 中行由 SeedGraperyIAPAppleProductsIfMissing 在缺失时插入（已有行不会自动更新）。
--
-- 毛利提示（Seedream ≈¥0.22/张，1 点≈1 张，满额用尽粗算）：基础 ¥29.9·100 点 ≈26%；高级 ¥49.9·200 点 ≈12%
-- 年付：约为 12×月价 × 0.8（约省 20%）

INSERT INTO subscription_plans (
    id,
    name,
    membership_tier,
    billing_period,
    iap_product_id,
    price,
    currency,
    token_quota,
    storage_quota,
    max_stories,
    max_characters,
    features,
    is_active,
    sort_order,
    created_at,
    updated_at
) VALUES
    -- ---------- 基础会员 pro_* ----------
    ('plan_pro_monthly', 'pro_monthly', 'basic', 'monthly', 'com.grapery.pro.monthly',
        29.90, 'CNY', 100, 1073741824, 200, 40,
        JSON_ARRAY(
            'membership_feature_basic_credits',
            'membership_feature_export_local',
            'membership_feature_share_unlimited',
            'membership_feature_keep_forever',
            'membership_feature_latest_models',
            'membership_feature_basic_serial_creation'
        ),
        1, 10, NOW(), NOW()),
    ('plan_pro_quarterly', 'pro_quarterly', 'basic', 'quarterly', 'com.grapery.pro.quarterly',
        89.90, 'CNY', 100, 1073741824, 200, 40,
        JSON_ARRAY(
            'membership_feature_basic_credits',
            'membership_feature_export_local',
            'membership_feature_share_unlimited',
            'membership_feature_keep_forever',
            'membership_feature_latest_models',
            'membership_feature_basic_serial_creation'
        ),
        0, 11, NOW(), NOW()),
    ('plan_pro_yearly', 'pro_yearly', 'basic', 'yearly', 'com.grapery.pro.yearly',
        287.00, 'CNY', 100, 1073741824, 200, 40,
        JSON_ARRAY(
            'membership_feature_basic_credits',
            'membership_feature_export_local',
            'membership_feature_share_unlimited',
            'membership_feature_keep_forever',
            'membership_feature_latest_models',
            'membership_feature_basic_serial_creation'
        ),
        1, 12, NOW(), NOW()),

    -- ---------- 高级会员 prime_* ----------
    ('plan_prime_monthly', 'prime_monthly', 'premium', 'monthly', 'com.grapery.prime.monthly',
        49.90, 'CNY', 200, 4294967296, 400, 80,
        JSON_ARRAY(
            'membership_feature_premium_credits',
            'membership_feature_export_local',
            'membership_feature_share_unlimited',
            'membership_feature_keep_forever',
            'membership_feature_latest_models',
            'membership_feature_premium_parallel_creation',
            'membership_feature_premium_higher_priority'
        ),
        1, 20, NOW(), NOW()),
    ('plan_prime_quarterly', 'prime_quarterly', 'premium', 'quarterly', 'com.grapery.prime.quarterly',
        149.90, 'CNY', 200, 4294967296, 400, 80,
        JSON_ARRAY(
            'membership_feature_premium_credits',
            'membership_feature_export_local',
            'membership_feature_share_unlimited',
            'membership_feature_keep_forever',
            'membership_feature_latest_models',
            'membership_feature_premium_parallel_creation',
            'membership_feature_premium_higher_priority'
        ),
        0, 21, NOW(), NOW()),
    ('plan_prime_yearly', 'prime_yearly', 'premium', 'yearly', 'com.grapery.prime.yearly',
        479.00, 'CNY', 200, 4294967296, 400, 80,
        JSON_ARRAY(
            'membership_feature_premium_credits',
            'membership_feature_export_local',
            'membership_feature_share_unlimited',
            'membership_feature_keep_forever',
            'membership_feature_latest_models',
            'membership_feature_premium_parallel_creation',
            'membership_feature_premium_higher_priority'
        ),
        1, 22, NOW(), NOW()),

    -- ---------- 已下架 Ultra（占位；勿与在售 SKU 混淆）----------
    ('plan_ultra_monthly', 'ultra_monthly', 'premium', 'monthly', 'com.grapery.ultra.monthly',
        128.00, 'CNY', -1, 21474836480, 2000, 200,
        JSON_ARRAY(
            'membership_feature_ultra_quota_unlimited',
            'membership_feature_export_local',
            'membership_feature_share_unlimited',
            'membership_feature_keep_forever',
            'membership_feature_latest_models',
            'membership_feature_premium_parallel_creation',
            'membership_feature_premium_higher_priority'
        ),
        0, 30, NOW(), NOW()),
    ('plan_ultra_quarterly', 'ultra_quarterly', 'premium', 'quarterly', 'com.grapery.ultra.quarterly',
        345.00, 'CNY', -1, 21474836480, 2000, 200,
        JSON_ARRAY(
            'membership_feature_ultra_quota_unlimited',
            'membership_feature_export_local',
            'membership_feature_share_unlimited',
            'membership_feature_keep_forever',
            'membership_feature_latest_models',
            'membership_feature_premium_parallel_creation',
            'membership_feature_premium_higher_priority'
        ),
        0, 31, NOW(), NOW()),
    ('plan_ultra_yearly', 'ultra_yearly', 'premium', 'yearly', 'com.grapery.ultra.yearly',
        1228.00, 'CNY', -1, 21474836480, 2000, 200,
        JSON_ARRAY(
            'membership_feature_ultra_quota_unlimited',
            'membership_feature_export_local',
            'membership_feature_share_unlimited',
            'membership_feature_keep_forever',
            'membership_feature_latest_models',
            'membership_feature_premium_parallel_creation',
            'membership_feature_premium_higher_priority'
        ),
        0, 32, NOW(), NOW())

ON DUPLICATE KEY UPDATE
    membership_tier = VALUES(membership_tier),
    billing_period = VALUES(billing_period),
    iap_product_id = VALUES(iap_product_id),
    price = VALUES(price),
    currency = VALUES(currency),
    token_quota = VALUES(token_quota),
    storage_quota = VALUES(storage_quota),
    max_stories = VALUES(max_stories),
    max_characters = VALUES(max_characters),
    features = VALUES(features),
    is_active = VALUES(is_active),
    sort_order = VALUES(sort_order),
    updated_at = NOW();
