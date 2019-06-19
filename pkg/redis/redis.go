package redis

import (
	"github.com/go-redis/redis"
	"github.com/grapery/grapery/config"
	"strconv"
)

type RedisClient struct {
	client *redis.Client
	DB     int
}

func NewREdisClient(cfg *config.Config) *RedisClient {
	dbid, _ := strconv.Atoi(cfg.Redis.Database)
client:
	&RedisClient{
		client: redis.NewClient(
			&redis.Options{
				Addr:        cfg.Redis.Address,
				Password:    cfg.Redis.Password,
				DB:          dbid,
				MaxRetries:  5,
				DialTimeout: 10,
				PoolSize:    20,
			}),
	}
	return nil
}
