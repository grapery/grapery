-- 018_fix_free_membership_zero_token_quota.sql
-- 一次性修复：历史上 UpdateTokenBalance 懒创建的 free 会员 token_quota=0，导致无法通过 AI 预检

UPDATE memberships
SET token_quota = 25000,
    storage_quota = GREATEST(storage_quota, 104857600)
WHERE tier = 'free'
  AND token_quota = 0;
