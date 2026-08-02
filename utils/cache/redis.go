package cache

import (
	"context"
	"strconv"
	"time"

	"github.com/go-redis/redis"
	log "github.com/sirupsen/logrus"

	"github.com/grapery/grapery/config"
)

var redisCache *RedisClient

type RedisClient struct {
	*redis.Client
	DB int
}

func NewRedisClient(cfg *config.Config) *RedisClient {
	log.Infof("address %s , passwrod %s ", cfg.Redis.Address, cfg.Redis.Password)
	dbid, _ := strconv.Atoi(cfg.Redis.Database)
	client := &RedisClient{
		redis.NewClient(
			&redis.Options{
				Addr:        cfg.Redis.Address,
				Password:    cfg.Redis.Password,
				DB:          dbid,
				MaxRetries:  5,
				DialTimeout: time.Second * 120,
				PoolSize:    20,
			}),
		dbid,
	}
	redisCache = client
	err := client.Ping().Err()
	if err != nil {
		panic(err)
	}
	return client
}

func GetCacheClient() *RedisClient {
	return redisCache
}

func SetCookie(ctx context.Context, key string, val string, ttl int64) error {
	return SetString(ctx, key, val, ttl)
}

func GetCookie(ctx context.Context, key string) (val string, err error) {
	return GetString(ctx, key)
}

func GetInt(ctx context.Context, key string) (val int, err error) {
	v := redisCache.Get(key)
	return v.Int()
}

func GetString(ctx context.Context, key string) (val string, err error) {
	v := redisCache.Get(key)
	return v.String(), nil
}

func GetBytes(ctx context.Context, key string) (val []byte, err error) {
	v := redisCache.Get(key)
	return v.Bytes()
}

func SetInt(ctx context.Context, key string, val int, ttl int64) error {
	cmd := redisCache.Set(key, val, time.Second*time.Duration(ttl))
	err := cmd.Err()
	return err
}

func SetString(ctx context.Context, key string, val string, ttl int64) error {
	cmd := redisCache.Set(key, val, time.Second*time.Duration(ttl))
	err := cmd.Err()
	return err
}

func SetBytes(ctx context.Context, key string, val []byte, ttl int64) error {
	cmd := redisCache.Set(key, val, time.Second*time.Duration(ttl))
	err := cmd.Err()
	return err
}

func SetObject(ctx context.Context, key string, val interface{}, ttl int64) error {
	cmd := redisCache.Set(key, val, time.Second*time.Duration(ttl))
	err := cmd.Err()
	return err
}

func GetObject(ctx context.Context, key string) (val interface{}, err error) {
	v := redisCache.Get(key)
	return v.Bytes()
}

func DelCache(ctx context.Context, key string) error {
	return redisCache.Del(key).Err()
}
