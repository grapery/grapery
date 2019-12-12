package redis

import (
	"github.com/go-redis/redis"
	"github.com/grapery/grapery/config"
	"strconv"
)

var RedisCache *RedisClient

type RedisClient struct {
	*redis.Client
	DB int
}

func NewRedisClient(cfg *config.Config) *RedisClient {
	dbid, _ := strconv.Atoi(cfg.Redis.Database)
	client := &RedisClient{
		redis.NewClient(
			&redis.Options{
				Addr:        cfg.Redis.Address,
				Password:    cfg.Redis.Password,
				DB:          dbid,
				MaxRetries:  5,
				DialTimeout: 10,
				PoolSize:    20,
			}),
		dbid,
	}
	return client
}
