package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/keweenaw-endurance/backend/internal/config"
	"github.com/redis/go-redis/v9"
)

// LeaderboardCache caches serialized leaderboard payloads.
type LeaderboardCache interface {
	Get(ctx context.Context, key string) ([]byte, bool)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}

type noopCache struct{}

func (noopCache) Get(context.Context, string) ([]byte, bool) { return nil, false }
func (noopCache) Set(context.Context, string, []byte, time.Duration) error { return nil }
func (noopCache) Delete(context.Context, string) error                     { return nil }

// NewLeaderboardCache connects to Redis when available; otherwise returns a no-op cache.
func NewLeaderboardCache(cfg config.RedisConfig) LeaderboardCache {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return noopCache{}
	}
	return &redisLeaderboardCache{client: client}
}

type redisLeaderboardCache struct {
	client *redis.Client
}

func (c *redisLeaderboardCache) Get(ctx context.Context, key string) ([]byte, bool) {
	val, err := c.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, false
	}
	if err != nil {
		return nil, false
	}
	return val, true
}

func (c *redisLeaderboardCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return c.client.Set(ctx, key, value, ttl).Err()
}

func (c *redisLeaderboardCache) Delete(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}

// MarshalJSON is a helper for encoding cached leaderboard slices.
func MarshalJSON(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// UnmarshalJSON decodes cached leaderboard slices.
func UnmarshalJSON(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
