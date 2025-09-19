package cache

import "github.com/go-redis/redis/v8"

func NewRedisClient(addr string, pswd string, db int) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: pswd,
		DB:       db,
	})
}
