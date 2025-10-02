package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// ==================== Pipeline Operations ====================

// Pipeline 创建 Pipeline
func (c *Client) Pipeline() redis.Pipeliner {
	return c.master.Pipeline()
}

// TxPipeline 创建事务 Pipeline
func (c *Client) TxPipeline() redis.Pipeliner {
	return c.master.TxPipeline()
}

// ==================== Transaction Operations ====================

// Watch 监视键
func (c *Client) Watch(ctx context.Context, fn func(*redis.Tx) error, keys ...string) error {
	err := c.master.Watch(ctx, fn, keys...)
	if err != nil {
		c.logger.Error("redis watch failed",
			zap.Strings("keys", keys),
			zap.Error(err),
		)
	}
	return err
}

// ==================== Pub/Sub Operations ====================

// Publish 发布消息
func (c *Client) Publish(ctx context.Context, channel string, message interface{}) (int64, error) {
	n, err := c.master.Publish(ctx, channel, message).Result()
	if err != nil {
		c.logger.Error("redis publish failed",
			zap.String("channel", channel),
			zap.Error(err),
		)
	} else {
		c.logger.Debug("redis message published",
			zap.String("channel", channel),
			zap.Int64("receivers", n),
		)
	}
	return n, err
}

// Subscribe 订阅频道
func (c *Client) Subscribe(ctx context.Context, channels ...string) *redis.PubSub {
	pubsub := c.master.Subscribe(ctx, channels...)
	c.logger.Info("redis subscribed to channels", zap.Strings("channels", channels))
	return pubsub
}

// PSubscribe 订阅模式匹配的频道
func (c *Client) PSubscribe(ctx context.Context, patterns ...string) *redis.PubSub {
	pubsub := c.master.PSubscribe(ctx, patterns...)
	c.logger.Info("redis psubscribed to patterns", zap.Strings("patterns", patterns))
	return pubsub
}

// ==================== Lua Script Operations ====================

// Eval 执行 Lua 脚本
func (c *Client) Eval(ctx context.Context, script string, keys []string, args ...interface{}) (interface{}, error) {
	result, err := c.master.Eval(ctx, script, keys, args...).Result()
	if err != nil {
		c.logger.Error("redis eval failed",
			zap.String("script", script),
			zap.Strings("keys", keys),
			zap.Error(err),
		)
	}
	return result, err
}

// EvalSha 通过 SHA1 执行 Lua 脚本
func (c *Client) EvalSha(ctx context.Context, sha1 string, keys []string, args ...interface{}) (interface{}, error) {
	result, err := c.master.EvalSha(ctx, sha1, keys, args...).Result()
	if err != nil {
		c.logger.Error("redis evalsha failed",
			zap.String("sha1", sha1),
			zap.Strings("keys", keys),
			zap.Error(err),
		)
	}
	return result, err
}

// ScriptLoad 加载 Lua 脚本
func (c *Client) ScriptLoad(ctx context.Context, script string) (string, error) {
	sha1, err := c.master.ScriptLoad(ctx, script).Result()
	if err != nil {
		c.logger.Error("redis script load failed",
			zap.String("script", script),
			zap.Error(err),
		)
	} else {
		c.logger.Debug("redis script loaded",
			zap.String("sha1", sha1),
		)
	}
	return sha1, err
}

// ==================== Scan Operations ====================

// Scan 扫描所有键
func (c *Client) Scan(ctx context.Context, cursor uint64, match string, count int64) ([]string, uint64, error) {
	client := c.getReadClient()
	keys, newCursor, err := client.Scan(ctx, cursor, match, count).Result()
	if err != nil {
		c.logger.Error("redis scan failed",
			zap.Uint64("cursor", cursor),
			zap.String("match", match),
			zap.Int64("count", count),
			zap.Error(err),
		)
	}
	return keys, newCursor, err
}

// HScan 扫描哈希字段
func (c *Client) HScan(ctx context.Context, key string, cursor uint64, match string, count int64) ([]string, uint64, error) {
	client := c.getReadClient()
	keys, newCursor, err := client.HScan(ctx, key, cursor, match, count).Result()
	if err != nil {
		c.logger.Error("redis hscan failed",
			zap.String("key", key),
			zap.Uint64("cursor", cursor),
			zap.String("match", match),
			zap.Int64("count", count),
			zap.Error(err),
		)
	}
	return keys, newCursor, err
}

// SScan 扫描集合成员
func (c *Client) SScan(ctx context.Context, key string, cursor uint64, match string, count int64) ([]string, uint64, error) {
	client := c.getReadClient()
	keys, newCursor, err := client.SScan(ctx, key, cursor, match, count).Result()
	if err != nil {
		c.logger.Error("redis sscan failed",
			zap.String("key", key),
			zap.Uint64("cursor", cursor),
			zap.String("match", match),
			zap.Int64("count", count),
			zap.Error(err),
		)
	}
	return keys, newCursor, err
}

// ZScan 扫描有序集合成员
func (c *Client) ZScan(ctx context.Context, key string, cursor uint64, match string, count int64) ([]string, uint64, error) {
	client := c.getReadClient()
	keys, newCursor, err := client.ZScan(ctx, key, cursor, match, count).Result()
	if err != nil {
		c.logger.Error("redis zscan failed",
			zap.String("key", key),
			zap.Uint64("cursor", cursor),
			zap.String("match", match),
			zap.Int64("count", count),
			zap.Error(err),
		)
	}
	return keys, newCursor, err
}

// ==================== Distributed Lock ====================

// Lock 获取分布式锁
func (c *Client) Lock(ctx context.Context, key string, expiration time.Duration) (string, error) {
	// 生成唯一标识
	token := uuid.New().String()

	// 使用 SetNX 获取锁
	ok, err := c.master.SetNX(ctx, key, token, expiration).Result()
	if err != nil {
		c.logger.Error("redis lock failed",
			zap.String("key", key),
			zap.Error(err),
		)
		return "", err
	}

	if !ok {
		return "", fmt.Errorf("failed to acquire lock: %s", key)
	}

	c.logger.Debug("redis lock acquired",
		zap.String("key", key),
		zap.String("token", token),
		zap.Duration("expiration", expiration),
	)

	return token, nil
}

// Unlock 释放分布式锁（使用 Lua 脚本保证原子性）
func (c *Client) Unlock(ctx context.Context, key, token string) error {
	// Lua 脚本：只有当锁的值等于 token 时才删除
	script := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("del", KEYS[1])
		else
			return 0
		end
	`

	result, err := c.master.Eval(ctx, script, []string{key}, token).Result()
	if err != nil {
		c.logger.Error("redis unlock failed",
			zap.String("key", key),
			zap.String("token", token),
			zap.Error(err),
		)
		return err
	}

	if result.(int64) == 0 {
		return fmt.Errorf("failed to release lock: token mismatch or lock expired")
	}

	c.logger.Debug("redis lock released",
		zap.String("key", key),
		zap.String("token", token),
	)

	return nil
}

// TryLock 尝试获取分布式锁（带重试）
func (c *Client) TryLock(ctx context.Context, key string, expiration time.Duration, maxRetries int, retryDelay time.Duration) (string, error) {
	var token string
	var err error

	for i := 0; i <= maxRetries; i++ {
		token, err = c.Lock(ctx, key, expiration)
		if err == nil {
			return token, nil
		}

		if i < maxRetries {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(retryDelay):
				continue
			}
		}
	}

	c.logger.Warn("redis trylock failed after retries",
		zap.String("key", key),
		zap.Int("retries", maxRetries),
		zap.Error(err),
	)

	return "", fmt.Errorf("failed to acquire lock after %d retries: %w", maxRetries, err)
}

// WithLock 在锁保护下执行函数
func (c *Client) WithLock(ctx context.Context, key string, expiration time.Duration, fn func() error) error {
	token, err := c.Lock(ctx, key, expiration)
	if err != nil {
		return err
	}

	defer func() {
		if err := c.Unlock(ctx, key, token); err != nil {
			c.logger.Error("failed to unlock",
				zap.String("key", key),
				zap.Error(err),
			)
		}
	}()

	return fn()
}

// ==================== GEO Operations ====================

// GeoAdd 添加地理位置
func (c *Client) GeoAdd(ctx context.Context, key string, geoLocation ...*redis.GeoLocation) (int64, error) {
	n, err := c.master.GeoAdd(ctx, key, geoLocation...).Result()
	if err != nil {
		c.logger.Error("redis geoadd failed",
			zap.String("key", key),
			zap.Error(err),
		)
	}
	return n, err
}

// GeoRadius 根据坐标查询半径内的成员
func (c *Client) GeoRadius(ctx context.Context, key string, longitude, latitude float64, query *redis.GeoRadiusQuery) ([]redis.GeoLocation, error) {
	client := c.getReadClient()
	locations, err := client.GeoRadius(ctx, key, longitude, latitude, query).Result()
	if err != nil {
		c.logger.Error("redis georadius failed",
			zap.String("key", key),
			zap.Float64("longitude", longitude),
			zap.Float64("latitude", latitude),
			zap.Error(err),
		)
	}
	return locations, err
}

// GeoRadiusByMember 根据成员查询半径内的成员
func (c *Client) GeoRadiusByMember(ctx context.Context, key, member string, query *redis.GeoRadiusQuery) ([]redis.GeoLocation, error) {
	client := c.getReadClient()
	locations, err := client.GeoRadiusByMember(ctx, key, member, query).Result()
	if err != nil {
		c.logger.Error("redis georadiusbymember failed",
			zap.String("key", key),
			zap.String("member", member),
			zap.Error(err),
		)
	}
	return locations, err
}

// GeoDist 计算两个成员之间的距离
func (c *Client) GeoDist(ctx context.Context, key string, member1, member2, unit string) (float64, error) {
	client := c.getReadClient()
	dist, err := client.GeoDist(ctx, key, member1, member2, unit).Result()
	if err != nil && !IsNil(err) {
		c.logger.Error("redis geodist failed",
			zap.String("key", key),
			zap.String("member1", member1),
			zap.String("member2", member2),
			zap.String("unit", unit),
			zap.Error(err),
		)
	}
	return dist, err
}

// ==================== HyperLogLog Operations ====================

// PFAdd 添加元素到 HyperLogLog
func (c *Client) PFAdd(ctx context.Context, key string, els ...interface{}) (int64, error) {
	n, err := c.master.PFAdd(ctx, key, els...).Result()
	if err != nil {
		c.logger.Error("redis pfadd failed",
			zap.String("key", key),
			zap.Error(err),
		)
	}
	return n, err
}

// PFCount 获取 HyperLogLog 基数估计值
func (c *Client) PFCount(ctx context.Context, keys ...string) (int64, error) {
	client := c.getReadClient()
	n, err := client.PFCount(ctx, keys...).Result()
	if err != nil {
		c.logger.Error("redis pfcount failed",
			zap.Strings("keys", keys),
			zap.Error(err),
		)
	}
	return n, err
}

// PFMerge 合并多个 HyperLogLog
func (c *Client) PFMerge(ctx context.Context, dest string, keys ...string) error {
	err := c.master.PFMerge(ctx, dest, keys...).Err()
	if err != nil {
		c.logger.Error("redis pfmerge failed",
			zap.String("dest", dest),
			zap.Strings("keys", keys),
			zap.Error(err),
		)
	}
	return err
}
