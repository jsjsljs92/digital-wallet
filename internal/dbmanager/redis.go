package dbmanager

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/digital-wallet/internal/config"
	"github.com/redis/go-redis/v9"
)

type RedisManager struct {
	client *redis.Client
}

func NewRedisManager(cfg *config.Config) (*RedisManager, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr(),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), config.GetContextTimeout())
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	log.Println("Redis connected successfully")

	return &RedisManager{client: client}, nil
}

func (r *RedisManager) GetClient() *redis.Client {
	return r.client
}

func (r *RedisManager) Close() error {
	return r.client.Close()
}

func (r *RedisManager) Health(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

func (r *RedisManager) Set(ctx context.Context, key string, value interface{}, ttl int64) error {
	return r.client.Set(ctx, key, value, time.Duration(ttl)*time.Second).Err()
}

func (r *RedisManager) Get(ctx context.Context, key string) (string, error) {
	return r.client.Get(ctx, key).Result()
}

func (r *RedisManager) Incr(ctx context.Context, key string) (int64, error) {
	return r.client.Incr(ctx, key).Result()
}

func (r *RedisManager) ExpireAt(ctx context.Context, key string, timestamp time.Time) error {
	return r.client.ExpireAt(ctx, key, timestamp).Err()
}

func (r *RedisManager) Del(ctx context.Context, keys ...string) error {
	return r.client.Del(ctx, keys...).Err()
}
