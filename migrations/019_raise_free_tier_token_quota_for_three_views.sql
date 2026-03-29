-- 019_raise_free_tier_token_quota_for_three_views.sql
-- 单张图预检/扣费约 4096 tokens；三视图需约 12288+，旧默认 10000 会在第 3 张失败。
-- 将免费档 token_quota 至少提升到 25000（已有更高配额的不变）。

UPDATE memberships
SET token_quota = GREATEST(token_quota, 25000)
WHERE tier = 'free';
