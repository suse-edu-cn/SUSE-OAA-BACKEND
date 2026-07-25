package database

import (
	"context"
	"fmt"
	"growthos/internal/config"
	"time"

	"github.com/redis/go-redis/v9"
)

func RedisInit(cfg config.Redis) *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.Database,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	fmt.Println(rdb.Ping(ctx).Err())
	if _, err := rdb.Ping(ctx).Result(); err != nil {
		panic("redis连接失败")
	}
	return rdb
}
