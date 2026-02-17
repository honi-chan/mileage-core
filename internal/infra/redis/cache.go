package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Cache はRedisキャッシュクライアント
type Cache struct {
	client *redis.Client
	logger *zap.Logger
}

// BalanceCache はキャッシュされた残高情報
type BalanceCache struct {
	Balance int64 `json:"balance"`
	Version int64 `json:"version"`
}

// NewCache は新しいCacheを作成する
func NewCache(addr string, logger *zap.Logger) (*Cache, error) {
	client := redis.NewClient(&redis.Options{
		Addr: addr,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	logger.Info("connected to Redis")
	return &Cache{client: client, logger: logger}, nil
}

// Close はRedis接続を閉じる
func (c *Cache) Close() error {
	return c.client.Close()
}

func balanceKey(userID string) string {
	return "mileage:balance:" + userID
}

// GetBalance はキャッシュから残高を取得する
func (c *Cache) GetBalance(ctx context.Context, userID string) (*BalanceCache, error) {
	data, err := c.client.Get(ctx, balanceKey(userID)).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // キャッシュミス
		}
		return nil, fmt.Errorf("redis get: %w", err)
	}

	var cache BalanceCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, fmt.Errorf("unmarshal cache: %w", err)
	}
	return &cache, nil
}

// SetBalance は残高をキャッシュに保存する
func (c *Cache) SetBalance(ctx context.Context, userID string, balance int64, version int64) error {
	data, err := json.Marshal(BalanceCache{Balance: balance, Version: version})
	if err != nil {
		return fmt.Errorf("marshal cache: %w", err)
	}

	if err := c.client.Set(ctx, balanceKey(userID), data, 5*time.Minute).Err(); err != nil {
		return fmt.Errorf("redis set: %w", err)
	}
	return nil
}

// InvalidateBalance は残高キャッシュを無効化する
func (c *Cache) InvalidateBalance(ctx context.Context, userID string) error {
	if err := c.client.Del(ctx, balanceKey(userID)).Err(); err != nil {
		return fmt.Errorf("redis del: %w", err)
	}
	return nil
}

// Client は生のRedisクライアントを返す（メトリクス用）
func (c *Cache) Client() *redis.Client {
	return c.client
}
