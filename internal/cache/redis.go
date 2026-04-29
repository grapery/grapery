package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

// Cache Redis 缓存接口
type Cache interface {
	Get(ctx context.Context, key string, dest interface{}) error
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Delete(ctx context.Context, keys ...string) error
	Exists(ctx context.Context, key string) (bool, error)
	Expire(ctx context.Context, key string, expiration time.Duration) error

	// 列表操作
	LPush(ctx context.Context, key string, values ...interface{}) error
	LRange(ctx context.Context, key string, start, stop int64) ([]string, error)
	LLen(ctx context.Context, key string) (int64, error)

	// 集合操作
	SAdd(ctx context.Context, key string, members ...interface{}) error
	SMembers(ctx context.Context, key string) ([]string, error)
	SIsMember(ctx context.Context, key string, member interface{}) (bool, error)
	SRem(ctx context.Context, key string, members ...interface{}) error

	// 有序集合操作
	ZAdd(ctx context.Context, key string, members ...*redis.Z) error
	ZRange(ctx context.Context, key string, start, stop int64) ([]string, error)
	ZRevRange(ctx context.Context, key string, start, stop int64) ([]string, error)
	ZCard(ctx context.Context, key string) (int64, error)
	ZRemRangeByRank(ctx context.Context, key string, start, stop int64) error
	ZScore(ctx context.Context, key string, member string) (float64, error)
	ZIncrBy(ctx context.Context, key string, increment float64, member string) (float64, error)

	// 哈希操作
	HSet(ctx context.Context, key string, values ...interface{}) error
	HGet(ctx context.Context, key, field string) (string, error)
	HGetAll(ctx context.Context, key string) (map[string]string, error)
	HIncrBy(ctx context.Context, key, field string, incr int64) (int64, error)

	// 原子操作
	Incr(ctx context.Context, key string) (int64, error)
	Decr(ctx context.Context, key string) (int64, error)
	IncrBy(ctx context.Context, key string, value int64) (int64, error)

	// Eval evaluates a Redis Lua script (Redis-backed cache only; other implementations return an error).
	Eval(ctx context.Context, script string, keys []string, args ...interface{}) (interface{}, error)

	Close() error
}

type redisCache struct {
	client *redis.Client
	logger *zap.Logger
}

// NewRedisCache 创建 Redis 缓存实例
func NewRedisCache(addr, password string, db int, logger *zap.Logger) (Cache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           db,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
		MinIdleConns: 5,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	logger.Info("redis connected successfully", zap.String("addr", addr))

	return &redisCache{
		client: client,
		logger: logger,
	}, nil
}

// Get 获取缓存
func (r *redisCache) Get(ctx context.Context, key string, dest interface{}) error {
	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return fmt.Errorf("key not found: %s", key)
		}
		return err
	}

	return json.Unmarshal([]byte(val), dest)
}

// Set 设置缓存
func (r *redisCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return r.client.Set(ctx, key, data, expiration).Err()
}

// Delete 删除缓存
func (r *redisCache) Delete(ctx context.Context, keys ...string) error {
	return r.client.Del(ctx, keys...).Err()
}

// Exists 检查键是否存在
func (r *redisCache) Exists(ctx context.Context, key string) (bool, error) {
	n, err := r.client.Exists(ctx, key).Result()
	return n > 0, err
}

// Expire 设置过期时间
func (r *redisCache) Expire(ctx context.Context, key string, expiration time.Duration) error {
	if r == nil || r.client == nil {
		return errors.New("redis cache not initialized")
	}
	return r.client.Expire(ctx, key, expiration).Err()
}

// LPush 列表左侧推入
func (r *redisCache) LPush(ctx context.Context, key string, values ...interface{}) error {
	return r.client.LPush(ctx, key, values...).Err()
}

// LRange 获取列表范围
func (r *redisCache) LRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return r.client.LRange(ctx, key, start, stop).Result()
}

// LLen 获取列表长度
func (r *redisCache) LLen(ctx context.Context, key string) (int64, error) {
	return r.client.LLen(ctx, key).Result()
}

// SAdd 集合添加成员
func (r *redisCache) SAdd(ctx context.Context, key string, members ...interface{}) error {
	return r.client.SAdd(ctx, key, members...).Err()
}

// SMembers 获取集合所有成员
func (r *redisCache) SMembers(ctx context.Context, key string) ([]string, error) {
	return r.client.SMembers(ctx, key).Result()
}

// SIsMember 检查是否是集合成员
func (r *redisCache) SIsMember(ctx context.Context, key string, member interface{}) (bool, error) {
	return r.client.SIsMember(ctx, key, member).Result()
}

// SRem 集合删除成员
func (r *redisCache) SRem(ctx context.Context, key string, members ...interface{}) error {
	return r.client.SRem(ctx, key, members...).Err()
}

// ZAdd 有序集合添加成员
func (r *redisCache) ZAdd(ctx context.Context, key string, members ...*redis.Z) error {
	return r.client.ZAdd(ctx, key, members...).Err()
}

// ZRange 有序集合范围查询（升序）
func (r *redisCache) ZRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return r.client.ZRange(ctx, key, start, stop).Result()
}

// ZRevRange 有序集合范围查询（降序）
func (r *redisCache) ZRevRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return r.client.ZRevRange(ctx, key, start, stop).Result()
}

// ZCard returns the number of members in a sorted set.
func (r *redisCache) ZCard(ctx context.Context, key string) (int64, error) {
	return r.client.ZCard(ctx, key).Result()
}

// ZRemRangeByRank removes members in the rank range [start, stop] (inclusive).
func (r *redisCache) ZRemRangeByRank(ctx context.Context, key string, start, stop int64) error {
	return r.client.ZRemRangeByRank(ctx, key, start, stop).Err()
}

// ZScore 获取有序集合成员分数
func (r *redisCache) ZScore(ctx context.Context, key string, member string) (float64, error) {
	return r.client.ZScore(ctx, key, member).Result()
}

// ZIncrBy 有序集合成员分数增加
func (r *redisCache) ZIncrBy(ctx context.Context, key string, increment float64, member string) (float64, error) {
	return r.client.ZIncrBy(ctx, key, increment, member).Result()
}

// HSet 哈希设置字段
func (r *redisCache) HSet(ctx context.Context, key string, values ...interface{}) error {
	return r.client.HSet(ctx, key, values...).Err()
}

// HGet 哈希获取字段
func (r *redisCache) HGet(ctx context.Context, key, field string) (string, error) {
	return r.client.HGet(ctx, key, field).Result()
}

// HGetAll 哈希获取所有字段
func (r *redisCache) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return r.client.HGetAll(ctx, key).Result()
}

// HIncrBy 哈希字段值增加
func (r *redisCache) HIncrBy(ctx context.Context, key, field string, incr int64) (int64, error) {
	return r.client.HIncrBy(ctx, key, field, incr).Result()
}

// Incr 原子递增
func (r *redisCache) Incr(ctx context.Context, key string) (int64, error) {
	if r == nil || r.client == nil {
		return 0, errors.New("redis cache not initialized")
	}
	return r.client.Incr(ctx, key).Result()
}

// Decr 原子递减
func (r *redisCache) Decr(ctx context.Context, key string) (int64, error) {
	return r.client.Decr(ctx, key).Result()
}

// IncrBy 原子增加指定值
func (r *redisCache) IncrBy(ctx context.Context, key string, value int64) (int64, error) {
	return r.client.IncrBy(ctx, key, value).Result()
}

// Eval runs a Lua script (see go-redis Eval).
func (r *redisCache) Eval(ctx context.Context, script string, keys []string, args ...interface{}) (interface{}, error) {
	if r == nil || r.client == nil {
		return nil, errors.New("redis cache not initialized")
	}
	return r.client.Eval(ctx, script, keys, args...).Result()
}

// Close 关闭连接
func (r *redisCache) Close() error {
	return r.client.Close()
}
