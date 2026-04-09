-- Seed / upsert subscription plans for iOS IAP alignment.
-- Keep `iap_product_id` exactly equal to the App Store Connect product ID
-- and vippay `/api/vippay/iap/products` `productId`.

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
    -- PRO
    ('plan_pro_monthly', 'pro_monthly', 'pro', 'monthly', 'com.grapery.pro.monthly', 38.00, 'CNY', 200, 1073741824, 200, 20, JSON_ARRAY('membership_feature_pro_credits', 'membership_feature_pro_image', 'membership_feature_pro_collection', 'membership_feature_pro_collab', 'membership_feature_pro_badge'), 1, 10, NOW(), NOW()),
    ('plan_pro_quarterly', 'pro_quarterly', 'pro', 'quarterly', 'com.grapery.pro.quarterly', 102.00, 'CNY', 200, 1073741824, 200, 20, JSON_ARRAY('membership_feature_pro_credits', 'membership_feature_pro_image', 'membership_feature_pro_collection', 'membership_feature_pro_collab', 'membership_feature_pro_badge'), 1, 11, NOW(), NOW()),
    ('plan_pro_yearly', 'pro_yearly', 'pro', 'yearly', 'com.grapery.pro.yearly', 348.00, 'CNY', 200, 1073741824, 200, 20, JSON_ARRAY('membership_feature_pro_credits', 'membership_feature_pro_image', 'membership_feature_pro_collection', 'membership_feature_pro_collab', 'membership_feature_pro_badge'), 1, 12, NOW(), NOW()),
    -- PRIME
    ('plan_prime_monthly', 'prime_monthly', 'prime', 'monthly', 'com.grapery.prime.monthly', 68.00, 'CNY', 600, 4294967296, 600, 60, JSON_ARRAY('membership_feature_prime_credits', 'membership_feature_prime_image', 'membership_feature_prime_video', 'membership_feature_pro_collection', 'membership_feature_pro_collab', 'membership_feature_prime_badge', 'membership_feature_prime_support'), 1, 20, NOW(), NOW()),
    ('plan_prime_quarterly', 'prime_quarterly', 'prime', 'quarterly', 'com.grapery.prime.quarterly', 183.00, 'CNY', 600, 4294967296, 600, 60, JSON_ARRAY('membership_feature_prime_credits', 'membership_feature_prime_image', 'membership_feature_prime_video', 'membership_feature_pro_collection', 'membership_feature_pro_collab', 'membership_feature_prime_badge', 'membership_feature_prime_support'), 1, 21, NOW(), NOW()),
    ('plan_prime_yearly', 'prime_yearly', 'prime', 'yearly', 'com.grapery.prime.yearly', 648.00, 'CNY', 600, 4294967296, 600, 60, JSON_ARRAY('membership_feature_prime_credits', 'membership_feature_prime_image', 'membership_feature_prime_video', 'membership_feature_pro_collection', 'membership_feature_pro_collab', 'membership_feature_prime_badge', 'membership_feature_prime_support'), 1, 22, NOW(), NOW()),
    -- ULTRA
    ('plan_ultra_monthly', 'ultra_monthly', 'ultra', 'monthly', 'com.grapery.ultra.monthly', 128.00, 'CNY', -1, 21474836480, 2000, 200, JSON_ARRAY('membership_feature_ultra_credits', 'membership_feature_ultra_image', 'membership_feature_ultra_video', 'membership_feature_ultra_style', 'membership_feature_ultra_badge', 'membership_feature_ultra_support', 'membership_feature_ultra_gift'), 1, 30, NOW(), NOW()),
    ('plan_ultra_quarterly', 'ultra_quarterly', 'ultra', 'quarterly', 'com.grapery.ultra.quarterly', 345.00, 'CNY', -1, 21474836480, 2000, 200, JSON_ARRAY('membership_feature_ultra_credits', 'membership_feature_ultra_image', 'membership_feature_ultra_video', 'membership_feature_ultra_style', 'membership_feature_ultra_badge', 'membership_feature_ultra_support', 'membership_feature_ultra_gift'), 1, 31, NOW(), NOW()),
    ('plan_ultra_yearly', 'ultra_yearly', 'ultra', 'yearly', 'com.grapery.ultra.yearly', 1228.00, 'CNY', -1, 21474836480, 2000, 200, JSON_ARRAY('membership_feature_ultra_credits', 'membership_feature_ultra_image', 'membership_feature_ultra_video', 'membership_feature_ultra_style', 'membership_feature_ultra_badge', 'membership_feature_ultra_support', 'membership_feature_ultra_gift'), 1, 32, NOW(), NOW())
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
