package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// DistributedLock 分布式锁服务
// 生产环境应使用 Redis 或其他分布式锁实现
// 这里提供内存版本作为参考
type DistributedLock struct {
	logger   *zap.Logger
	locks    map[string]*lockHolder
	mu       sync.Mutex
}

type lockHolder struct {
	holderID  string
	lockedAt  time.Time
	expiresAt time.Time
}

// NewDistributedLock 创建分布式锁服务
func NewDistributedLock(logger *zap.Logger) *DistributedLock {
	return &DistributedLock{
		logger: logger,
		locks:  make(map[string]*lockHolder),
	}
}

// AcquireLock 获取锁
// key: 锁的键名（如 "quota_lock:user:123"）
// holderID: 持有者标识（如请求ID）
// ttl: 锁的过期时间
func (dl *DistributedLock) AcquireLock(ctx context.Context, key string, holderID string, ttl time.Duration) (bool, error) {
	dl.mu.Lock()
	defer dl.mu.Unlock()

	now := time.Now()
	expiresAt := now.Add(ttl)

	// 检查锁是否已存在
	if lock, exists := dl.locks[key]; exists {
		// 检查是否已过期
		if now.After(lock.expiresAt) {
			// 锁已过期，删除并获取新锁
			delete(dl.locks, key)
		} else {
			// 锁仍被持有
			if lock.holderID == holderID {
				// 同一持有者，延长锁时间
				lock.expiresAt = expiresAt
				return true, nil
			}
			return false, fmt.Errorf("lock already held by: %s", lock.holderID)
		}
	}

	// 获取新锁
	dl.locks[key] = &lockHolder{
		holderID:  holderID,
		lockedAt:  now,
		expiresAt: expiresAt,
	}

	dl.logger.Debug("lock acquired",
		zap.String("key", key),
		zap.String("holderID", holderID),
		zap.Duration("ttl", ttl))

	return true, nil
}

// ReleaseLock 释放锁
func (dl *DistributedLock) ReleaseLock(ctx context.Context, key string, holderID string) error {
	dl.mu.Lock()
	defer dl.mu.Unlock()

	lock, exists := dl.locks[key]
	if !exists {
		return fmt.Errorf("lock not found: %s", key)
	}

	if lock.holderID != holderID {
		return fmt.Errorf("lock held by different holder: %s", lock.holderID)
	}

	delete(dl.locks, key)

	dl.logger.Debug("lock released",
		zap.String("key", key),
		zap.String("holderID", holderID))

	return nil
}

// TryAcquireLockWithRetry 尝试获取锁，支持重试
func (dl *DistributedLock) TryAcquireLockWithRetry(ctx context.Context, key string, holderID string, ttl time.Duration, maxAttempts int, retryInterval time.Duration) (bool, error) {
	for i := 0; i < maxAttempts; i++ {
		acquired, err := dl.AcquireLock(ctx, key, holderID, ttl)
		if err != nil {
			return false, err
		}
		if acquired {
			return true, nil
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

// IsLocked 检查锁是否被持有
func (dl *DistributedLock) IsLocked(key string) bool {
	dl.mu.Lock()
	defer dl.mu.Unlock()

	lock, exists := dl.locks[key]
	if !exists {
		return false
	}

	// 检查是否已过期
	return time.Now().Before(lock.expiresAt)
}

// GetLockInfo 获取锁信息
func (dl *DistributedLock) GetLockInfo(key string) (holderID string, expiresAt time.Time, exists bool) {
	dl.mu.Lock()
	defer dl.mu.Unlock()

	lock, exists := dl.locks[key]
	if !exists {
		return "", time.Time{}, false
	}

	return lock.holderID, lock.expiresAt, true
}

// CleanupExpiredLocks 清理过期的锁
func (dl *DistributedLock) CleanupExpiredLocks() {
	dl.mu.Lock()
	defer dl.mu.Unlock()

	now := time.Now()
	for key, lock := range dl.locks {
		if now.After(lock.expiresAt) {
			delete(dl.locks, key)
			dl.logger.Debug("expired lock cleaned up",
				zap.String("key", key),
				zap.String("holderID", lock.holderID))
		}
	}
}

// WithLock 使用锁执行函数
func (dl *DistributedLock) WithLock(ctx context.Context, key string, holderID string, ttl time.Duration, fn func() error) error {
	// 获取锁
	acquired, err := dl.AcquireLock(ctx, key, holderID, ttl)
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	if !acquired {
		return fmt.Errorf("lock not acquired")
	}

	// 确保释放锁
	defer func() {
		if err := dl.ReleaseLock(ctx, key, holderID); err != nil {
			dl.logger.Error("failed to release lock",
				zap.String("key", key),
				zap.String("holderID", holderID),
				zap.Error(err))
		}
	}()

	// 执行函数
	return fn()
}
