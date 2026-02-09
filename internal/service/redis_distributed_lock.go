package service

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// RedisDistributedLock Redis 分布式锁服务
// 使用 Redis SET NX EX 命令实现分布式锁（Redlock 算法简化版）
type RedisDistributedLock struct {
	client *redis.Client
	logger *zap.Logger
}

// NewRedisDistributedLock 创建 Redis 分布式锁服务
func NewRedisDistributedLock(client *redis.Client, logger *zap.Logger) *RedisDistributedLock {
	return &RedisDistributedLock{
		client: client,
		logger: logger,
	}
}

// lockValue 锁的值（用于标识锁的持有者）
type lockValue struct {
	HolderID  string `json:"holder_id"`
	LockedAt  int64  `json:"locked_at"`
	ExpiresAt int64  `json:"expires_at"`
}

// AcquireLock 获取分布式锁
// key: 锁的键名（如 "quota_lock:user:123"）
// holderID: 持有者标识（如请求ID）
// ttl: 锁的过期时间
func (dl *RedisDistributedLock) AcquireLock(ctx context.Context, key string, holderID string, ttl time.Duration) (bool, error) {
	now := time.Now()
	expiresAt := now.Add(ttl)

	lockVal := lockValue{
		HolderID:  holderID,
		LockedAt:  now.Unix(),
		ExpiresAt: expiresAt.Unix(),
	}

	// 序列化锁值
	lockData := fmt.Sprintf("%s|%d|%d", lockVal.HolderID, lockVal.LockedAt, lockVal.ExpiresAt)

	// 使用 SET NX EX 命令实现分布式锁
	// NX: 只在键不存在时设置
	// EX: 设置过期时间
	result, err := dl.client.SetNX(ctx, key, lockData, ttl).Result()
	if err != nil {
		dl.logger.Error("failed to acquire lock",
			zap.String("key", key),
			zap.String("holderID", holderID),
			zap.Error(err))
		return false, fmt.Errorf("failed to acquire lock: %w", err)
	}

	if result {
		dl.logger.Debug("lock acquired",
			zap.String("key", key),
			zap.String("holderID", holderID),
			zap.Duration("ttl", ttl))
	} else {
		// 获取失败，检查当前锁的持有者信息
		existingVal, _ := dl.client.Get(ctx, key).Result()
		if existingVal != "" {
			dl.logger.Debug("lock already held",
				zap.String("key", key),
				zap.String("currentHolder", existingVal))
		}
	}

	return result, nil
}

// ReleaseLock 释放分布式锁
// 使用 Lua 脚本确保只释放自己持有的锁
func (dl *RedisDistributedLock) ReleaseLock(ctx context.Context, key string, holderID string) error {
	// Lua 脚本：确保只删除自己持有的锁
	// 如果锁的持有者匹配，才删除锁
	// 这样可以防止删除其他人的锁
	script := `
		local lockValue = redis.call("GET", KEYS[1])
		if not lockValue then
			return 0
	end

	-- 解析锁值，检查持有者
	local holderID = string.match(lockValue, "^([^|]+)")
	if holderID == ARGV[1] then
		return redis.call("DEL", KEYS[1])
	else
		return 0
	end
	`

	result, err := dl.client.Eval(ctx, script, []string{key}, holderID).Result()
	if err != nil {
		dl.logger.Error("failed to release lock",
			zap.String("key", key),
			zap.String("holderID", holderID),
			zap.Error(err))
		return fmt.Errorf("failed to release lock: %w", err)
	}

	if result.(int64) == 1 {
		dl.logger.Debug("lock released",
			zap.String("key", key),
			zap.String("holderID", holderID))
	} else {
		dl.logger.Warn("lock not released - holder mismatch or lock expired",
			zap.String("key", key),
			zap.String("holderID", holderID))
	}

	return nil
}

// TryAcquireLockWithRetry 尝试获取锁，支持重试
func (dl *RedisDistributedLock) TryAcquireLockWithRetry(ctx context.Context, key string, holderID string, ttl time.Duration, maxAttempts int, retryInterval time.Duration) (bool, error) {
	for i := 0; i < maxAttempts; i++ {
		acquired, err := dl.AcquireLock(ctx, key, holderID, ttl)
		if err != nil {
			return false, err
		}
		if acquired {
			return true, nil
		}

		// 最后一次尝试，不再等待
		if i == maxAttempts-1 {
			break
		}

		// 等待重试
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case <-time.After(retryInterval):
			continue
		}
	}

	return false, fmt.Errorf("failed to acquire lock after %d attempts", maxAttempts)
}

// IsLocked 检查锁是否存在
func (dl *RedisDistributedLock) IsLocked(ctx context.Context, key string) bool {
	exists, err := dl.client.Exists(ctx, key).Result()
	if err != nil {
		return false
	}

	// 即使 key 存在，也需要检查是否已过期
	if exists > 0 {
		// 获取锁值并检查过期时间
		val, err := dl.client.Get(ctx, key).Result()
		if err == nil && val != "" {
			// 简单解析：检查是否过期
			// 格式: holderID|lockedAt|expiresAt
			// 这里我们只检查 key 是否存在，Redis 会自动过期
			return true
		}
	}

	return false
}

// GetLockInfo 获取锁信息
func (dl *RedisDistributedLock) GetLockInfo(ctx context.Context, key string) (holderID string, expiresAt time.Time, exists bool) {
	val, err := dl.client.Get(ctx, key).Result()
	if err != nil || val == "" {
		return "", time.Time{}, false
	}

	// 解析锁值: holderID|lockedAt|expiresAt
	var holderIDStr, lockedAtStr, expiresAtStr string
	_, err = fmt.Sscanf(val, "%s|%s|%s", &holderIDStr, &lockedAtStr, &expiresAtStr)
	if err != nil {
		dl.logger.Warn("failed to parse lock value",
			zap.String("key", key),
			zap.String("value", val),
			zap.Error(err))
		return "", time.Time{}, true
	}

	var expiresAtTime time.Time
	if exp, err := time.ParseDuration(fmt.Sprintf("%ds", expiresAtStr)); err == nil {
		expiresAtTime = time.Unix(0, 0).Add(exp)
	}

	return holderIDStr, expiresAtTime, true
}

// ExtendLock 延长锁的过期时间
// 只有锁的持有者才能延长锁
func (dl *RedisDistributedLock) ExtendLock(ctx context.Context, key string, holderID string, additionalTTL time.Duration) error {
	// Lua 脚本：确保只延长自己持有的锁
	script := `
		local lockValue = redis.call("GET", KEYS[1])
		if not lockValue then
			return -1  -- 锁不存在
		end

		-- 解析锁值，检查持有者
		local currentHolder = string.match(lockValue, "^([^|]+)")
		if currentHolder ~= ARGV[1] then
			return -2  -- 不是锁的持有者
		end

		-- 延长锁的过期时间
		return redis.call("EXPIRE", KEYS[1], ARGV[2])
	`

	expiresInSeconds := int64(additionalTTL.Seconds())
	result, err := dl.client.Eval(ctx, script, []string{key}, holderID, expiresInSeconds).Result()
	if err != nil {
		return fmt.Errorf("failed to extend lock: %w", err)
	}

	resultCode := result.(int64)
	if resultCode == -1 {
		return fmt.Errorf("lock not found: %s", key)
	} else if resultCode == -2 {
		return fmt.Errorf("not the lock holder: %s", holderID)
	}

	dl.logger.Debug("lock extended",
		zap.String("key", key),
		zap.String("holderID", holderID),
		zap.Duration("additionalTTL", additionalTTL))

	return nil
}

// WithLock 使用锁执行函数
func (dl *RedisDistributedLock) WithLock(ctx context.Context, key string, holderID string, ttl time.Duration, fn func() error) error {
	// 生成唯一的持有者ID
	if holderID == "" {
		holderID = uuid.New().String()
	}

	// 获取锁
	acquired, err := dl.AcquireLock(ctx, key, holderID, ttl)
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	if !acquired {
		return fmt.Errorf("lock not acquired for key: %s", key)
	}

	// 确保释放锁
	defer func() {
		if err := dl.ReleaseLock(ctx, key, holderID); err != nil {
			dl.logger.Error("failed to release lock in defer",
				zap.String("key", key),
				zap.String("holderID", holderID),
				zap.Error(err))
		}
	}()

	// 执行函数
	return fn()
}

// CleanupExpiredLocks 清理过期的锁
// Redis 会自动清理过期的 key，所以这个方法主要用于日志记录
func (dl *RedisDistributedLock) CleanupExpiredLocks(ctx context.Context, pattern string) {
	// 使用 SCAN 命令查找所有锁相关的 key
	var cursor uint64
	patternToScan := pattern
	if patternToScan == "" {
		patternToScan = "lock:*"
	}

	keys, nextCursor, err := dl.client.Scan(ctx, cursor, patternToScan, 100).Result()
	if err != nil {
		dl.logger.Error("failed to scan for locks",
			zap.String("pattern", patternToScan),
			zap.Error(err))
		return
	}

	// 检查每个锁的状态
	for _, key := range keys {
		val, err := dl.client.Get(ctx, key).Result()
		if err != nil || val == "" {
			continue
		}

		// 解析锁值并检查是否过期
		// 格式: holderID|lockedAt|expiresAt
		var holderIDStr, lockedAtStr, expiresAtStr string
		_, err = fmt.Sscanf(val, "%s|%s|%s", &holderIDStr, &lockedAtStr, &expiresAtStr)
		if err != nil {
			dl.logger.Debug("found malformed lock value",
				zap.String("key", key),
				zap.String("value", val))
			continue
		}

		dl.logger.Debug("found active lock",
			zap.String("key", key),
			zap.String("holderID", holderIDStr))
	}

	// 如果还有更多 key，继续扫描
	if nextCursor != 0 {
		dl.logger.Debug("more keys to scan",
			zap.Uint64("nextCursor", nextCursor))
	}
}
