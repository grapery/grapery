-- =============================================================================
-- App Review demo account — main Grapery DB (users, settings, membership, content)
-- =============================================================================
--
-- Demo login (in-app email, NOT Apple Sandbox):
--   Email:    jingjing@grapery.xyz
--   Password: Gr@p3ryIap2026!
--
-- Fixed user UUID (VipPay FNV must match; max 36 chars for users.id):
--   revw-jingjing-0000-8000-000000000001
--
-- Usage (same DB as server + vippay; does NOT wipe other data):
--   mysql -u root -p -h localhost grapery < scripts/app_review_demo_jingjing.sql
--   mysql ... grapery < scripts/app_review_demo_jingjing_vippay.sql
-- Or: make review-demo-load
--
-- Regenerate password_hash / FNV ids after password change:
--   go run ./cmd/gen-review-demo
--
-- Prerequisites:
--   - Run membership_iap_seed.sql once if subscription_plans empty (optional here)
--   - Start vippay once so iap_products exist, OR load vippay SQL after products seeded
--
-- Verification (after both SQL files):
--   1. POST /api/auth/login {"email":"jingjing@grapery.xyz","password":"Gr@p3ryIap2026!"}
--      → code=1, requiresPhoneVerification=false
--   2. GET /api/v1/auth/me (Bearer token)
--   3. GET /api/v1/membership/current → tier=basic
--   4. GET /api/vippay/vip/info → is_vip=true, credit_limit≈103
--   5. iOS: Login → Email → credentials → no phone gate → Feed/Profile/Membership
--
-- Common failures if login or review breaks:
--   - invalid email or password → password_hash mismatch; re-run gen-review-demo
--   - phone verification gate → pending_oauth_phone_sms=1 or phone_verified_at=0
--   - is_vip=false → vippay SQL not loaded or iap_products missing
--   - AI token errors → memberships.token_quota too low (need 1025000 for basic)
--   - reviewer used Apple/WeChat instead of Email → different user, this SQL unused
--   - injected staging DB but app hits production → account not found
--
-- WARNING: Do NOT run scripts/mock_data.sql on production (it DELETEs all users).
-- =============================================================================

SET NAMES utf8mb4 COLLATE utf8mb4_unicode_ci;

SET @review_user_id   := 'revw-jingjing-0000-8000-000000000001';
SET @review_email     := 'jingjing@grapery.xyz';
SET @review_username  := 'jingjing_review';
SET @review_membership_id := 'membership-review-jingjing-0001';
SET @review_story_id  := 'story-review-jingjing-0001';
SET @review_frag_id   := 'frag-review-jingjing-0001';
-- bcrypt(Gr@p3ryIap2026!) from: go run ./cmd/gen-review-demo
SET @review_password_hash := '$2a$10$CVVwXXdwmiZOlGTOBzapouMbhh9KLpqFS.8xdEub7TaC6Iy2lkXQi';

SET FOREIGN_KEY_CHECKS = 0;

-- Idempotent: remove prior review demo rows only (by email or fixed ids)
DELETE FROM fragments WHERE id = @review_frag_id OR creator_id = @review_user_id;
DELETE FROM stories WHERE id = @review_story_id OR author_id = @review_user_id;
DELETE FROM memberships WHERE user_id = @review_user_id OR id = @review_membership_id;
DELETE FROM user_settings WHERE user_id = @review_user_id;
DELETE FROM users WHERE email = @review_email OR id = @review_user_id;

SET FOREIGN_KEY_CHECKS = 1;

-- ---------------------------------------------------------------------------
-- User (email login; phone gate disabled)
-- ---------------------------------------------------------------------------
INSERT INTO users (
    id, username, email, password_hash, display_name, avatar, background, bio,
    location, website, followers, following, storyboard_count, fragments_count,
    status, email_verified, phone, phone_verified_at, pending_oauth_phone_sms,
    last_login_at, points, referral_code, created_at, updated_at
) VALUES (
    @review_user_id,
    @review_username,
    @review_email,
    @review_password_hash,
    'Jingjing (Review)',
    'https://picsum.photos/seed/jingjing-review/200/200',
    'https://picsum.photos/seed/jingjing-review-bg/800/400',
    'App Store Review demo account. Use Email login in the app.',
    '',
    '',
    0, 0, 0, 1,
    'active',
    1,
    NULL,
    UNIX_TIMESTAMP(),
    0,
    UNIX_TIMESTAMP() * 1000,
    0,
    'JING2026',
    FROM_UNIXTIME(UNIX_TIMESTAMP()),
    FROM_UNIXTIME(UNIX_TIMESTAMP())
);

-- ---------------------------------------------------------------------------
-- User settings (mirrors register flow defaults)
-- ---------------------------------------------------------------------------
INSERT INTO user_settings (
    id, user_id, language, theme, font_size, data_saver,
    profile_visibility, default_story_visibility, default_fragment_visibility,
    allow_follow_from, allow_comments_from, allow_messages_from,
    show_online_status, show_read_receipts, ai_enabled, a_idata_sharing,
    notification_settings, updated_at
) VALUES (
    UUID(),
    @review_user_id,
    'en-US',
    'system',
    'medium',
    0,
    'public',
    'public',
    'public',
    'everyone',
    'everyone',
    'everyone',
    1,
    1,
    1,
    1,
    '{"push":true,"email":true,"likes":true,"comments":true,"follows":true}',
    UNIX_TIMESTAMP()
);

-- ---------------------------------------------------------------------------
-- Membership — basic tier, AI token quota for review (25k free + 100 credits)
-- token_quota = 25000 + (100 * 10000) = 1025000
-- ---------------------------------------------------------------------------
INSERT INTO memberships (
    id, user_id, tier, status, start_date, end_date, auto_renew,
    token_quota, token_used, storage_quota, storage_used, created_at, updated_at
) VALUES (
    @review_membership_id,
    @review_user_id,
    'basic',
    'active',
    NOW(),
    DATE_ADD(NOW(), INTERVAL 1 YEAR),
    1,
    1025000,
    0,
    1073741824,
    0,
    NOW(),
    NOW()
);

-- ---------------------------------------------------------------------------
-- Optional public content (Guideline 2.1 — reviewer sees owned + feed content)
-- ---------------------------------------------------------------------------
INSERT INTO stories (
    id, title, description, cover_image, author_id,
    likes, followers, saves, panels, storyboard_count, default_scene_count,
    genre, style, status, is_collaboration_open, visibility, use_ai, ai_enabled,
    created_at, updated_at
) VALUES (
    @review_story_id,
    'Review Demo Story',
    'A sample public story for App Store Review. Browse Fragments and Stories tabs for more community content.',
    'https://picsum.photos/seed/review-story/400/600',
    @review_user_id,
    12, 3, 2, 0, 0, 3,
    'fantasy',
    '{"artStyle":"anime","colorPalette":"vibrant","lighting":"soft"}',
    'published',
    0,
    'public',
    1,
    1,
    FROM_UNIXTIME(UNIX_TIMESTAMP() - 86400),
    FROM_UNIXTIME(UNIX_TIMESTAMP())
);

INSERT INTO fragments (
    id, creator_id, content, image_urls, visibility, source_type,
    is_converted, likes, comments, shares, views, created_at, updated_at
) VALUES (
    @review_frag_id,
    @review_user_id,
    'Welcome to the review demo account. This fragment is public so reviewers can verify creation and feed features after email login.',
    '["https://picsum.photos/seed/review-frag/400/300"]',
    'public',
    'original',
    0,
    5, 1, 0, 42,
    UNIX_TIMESTAMP() - 3600,
    UNIX_TIMESTAMP() - 3600
);

UPDATE users SET fragments_count = 1, storyboard_count = 0 WHERE id = @review_user_id;

SELECT 'App Review demo user ready' AS status, @review_email AS email, @review_user_id AS user_id;
